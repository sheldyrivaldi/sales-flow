package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/telemetry"
)

// ProspectService handles business logic for the prospect entity.
type ProspectService struct {
	repo     domain.ProspectRepository
	outcomes domain.OutcomeRepository
	learn    LearningHook
	emit     *telemetry.Emitter
}

func NewProspectService(repo domain.ProspectRepository, outcomes domain.OutcomeRepository, learn LearningHook) *ProspectService {
	return &ProspectService{repo: repo, outcomes: outcomes, learn: learn}
}

// SetEmitter wires telemetry (EP-17 ST-17.1) after construction — optional,
// nil-safe, kept out of the constructor so existing call sites/tests are
// unaffected by observability being wired in or not.
func (s *ProspectService) SetEmitter(e *telemetry.Emitter) { s.emit = e }

// Create creates a new prospect from a create request.
func (s *ProspectService) Create(ctx context.Context, req *dto.ProspectCreateRequest, defaultOwnerUserID string) (*domain.Prospect, error) {
	p := &domain.Prospect{
		Name:       req.Name,
		SourceType: domain.ProspectSourceManual,
		Stage:      domain.ProspectStageNew,
	}

	p.Company = req.Company
	p.ContactInfo = req.ContactInfo
	p.SourceID = req.SourceID
	p.EstValue = req.EstValue

	if req.SourceType != nil {
		st := domain.ProspectSource(*req.SourceType)
		if !st.Valid() {
			return nil, httperr.NewBadRequest("INVALID_SOURCE_TYPE", "source_type tidak valid")
		}
		p.SourceType = st
	}
	if req.Stage != nil {
		stage := domain.ProspectStage(*req.Stage)
		if !stage.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STAGE", "stage tidak valid")
		}
		p.Stage = stage
	}

	if req.OwnerUserID != nil {
		p.OwnerUserID = req.OwnerUserID
	} else if defaultOwnerUserID != "" {
		p.OwnerUserID = &defaultOwnerUserID
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("prospect.Create: %w", err)
	}
	return p, nil
}

// Get returns a prospect by ID.
func (s *ProspectService) Get(ctx context.Context, id string) (*domain.Prospect, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("prospek tidak ditemukan")
		}
		return nil, fmt.Errorf("prospect.Get: %w", err)
	}
	return p, nil
}

// List returns paginated prospects matching the filter.
func (s *ProspectService) List(ctx context.Context, f domain.ProspectFilter, page, pageSize int) ([]domain.Prospect, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	return s.repo.List(ctx, f, page, pageSize)
}

// Update applies a partial update to a prospect.
func (s *ProspectService) Update(ctx context.Context, id string, req *dto.ProspectUpdateRequest) (*domain.Prospect, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Company != nil {
		p.Company = req.Company
	}
	if req.ContactInfo != nil {
		p.ContactInfo = req.ContactInfo
	}
	if req.SourceType != nil {
		st := domain.ProspectSource(*req.SourceType)
		if !st.Valid() {
			return nil, httperr.NewBadRequest("INVALID_SOURCE_TYPE", "source_type tidak valid")
		}
		p.SourceType = st
	}
	if req.SourceID != nil {
		p.SourceID = req.SourceID
	}
	if req.Stage != nil {
		stage := domain.ProspectStage(*req.Stage)
		if !stage.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STAGE", "stage tidak valid")
		}
		if err := s.applyStage(ctx, p, stage, ""); err != nil {
			return nil, err
		}
	}
	if req.EstValue != nil {
		p.EstValue = req.EstValue
	}
	if req.OwnerUserID != nil {
		p.OwnerUserID = req.OwnerUserID
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("prospect.Update: %w", err)
	}
	return p, nil
}

// Delete removes a prospect by ID.
func (s *ProspectService) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("prospect.Delete: %w", err)
	}
	return nil
}

// applyStage moves p to the target stage. If the target differs from p's
// current stage (a no-op guard, so repeat/duplicate calls don't emit
// duplicate outcome events) and is WON/LOST, it records an outcome_event
// (EP-16 learning) via the shared recordOutcome helper. Used by both
// UpdateStage (PATCH .../stage) and Update (PUT .../:id) so a stage change
// always fires outcome recording regardless of entry point.
func (s *ProspectService) applyStage(ctx context.Context, p *domain.Prospect, to domain.ProspectStage, notes string) error {
	if p.Stage == to {
		return nil
	}
	p.Stage = to

	if to == domain.ProspectStageWon || to == domain.ProspectStageLost {
		result := domain.OutcomeLost
		if to == domain.ProspectStageWon {
			result = domain.OutcomeWon
		}
		if _, err := recordOutcome(ctx, s.outcomes, s.learn, s.emit, domain.OutcomeTargetProspect, p.ID, result, notes); err != nil {
			return fmt.Errorf("prospect.applyStage: %w", err)
		}
	}
	return nil
}

// UpdateStage moves a prospect to a new stage (any→any). WON/LOST additionally
// records an outcome_event (for EP-16 learning) and notifies the learning hook.
func (s *ProspectService) UpdateStage(ctx context.Context, id string, to domain.ProspectStage, notes string) (*domain.Prospect, error) {
	if !to.Valid() {
		return nil, httperr.NewBadRequest("INVALID_STAGE", "stage tidak valid")
	}

	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.applyStage(ctx, p, to, notes); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("prospect.UpdateStage update: %w", err)
	}
	return p, nil
}
