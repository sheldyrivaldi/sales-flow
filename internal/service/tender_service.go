package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
)

// tenderTransitions defines the set of valid target statuses for each current status.
var tenderTransitions = map[domain.TenderStatus][]domain.TenderStatus{
	domain.TenderStatusIdentified: {domain.TenderStatusQualifying, domain.TenderStatusLost},
	domain.TenderStatusQualifying: {domain.TenderStatusBidding, domain.TenderStatusLost},
	domain.TenderStatusBidding:    {domain.TenderStatusSubmitted, domain.TenderStatusWon, domain.TenderStatusLost},
	domain.TenderStatusSubmitted:  {domain.TenderStatusWon, domain.TenderStatusLost},
	domain.TenderStatusWon:        {},
	domain.TenderStatusLost:       {},
}

func canTransition(from, to domain.TenderStatus) bool {
	targets, ok := tenderTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// LearningHook is called after an outcome is recorded so EP-16 can push to Hermes memory.
type LearningHook interface {
	RecordOutcome(ctx context.Context, e domain.OutcomeEvent)
}

type noopLearningHook struct{}

func (noopLearningHook) RecordOutcome(_ context.Context, _ domain.OutcomeEvent) {}

// NoopLearningHook returns a LearningHook that does nothing (placeholder until EP-16).
func NoopLearningHook() LearningHook { return noopLearningHook{} }

// TenderService handles business logic for the tender entity.
type TenderService struct {
	repo     domain.TenderRepository
	outcomes domain.OutcomeRepository
	learn    LearningHook
}

func NewTenderService(repo domain.TenderRepository, outcomes domain.OutcomeRepository, learn LearningHook) *TenderService {
	return &TenderService{repo: repo, outcomes: outcomes, learn: learn}
}

// Create creates a new tender from a create request.
func (s *TenderService) Create(ctx context.Context, req *dto.TenderCreateRequest) (*domain.Tender, error) {
	t := &domain.Tender{
		Title:    req.Title,
		Status:   domain.TenderStatusIdentified,
		Currency: "IDR",
		Origin:   domain.OriginManual,
	}

	// Apply optional fields.
	t.BuyerName = req.BuyerName
	t.BuyerCountry = req.BuyerCountry
	t.BuyerIndustry = req.BuyerIndustry
	t.ValueEstimate = req.ValueEstimate
	t.SourceName = req.SourceName
	t.SourceURL = req.SourceURL
	t.ServiceCategory = req.ServiceCategory
	t.ScopeSummary = req.ScopeSummary
	t.EligibilityRequirements = req.EligibilityRequirements
	t.TechnicalRequirements = req.TechnicalRequirements
	t.DedupKey = req.DedupKey

	if req.Currency != nil {
		t.Currency = *req.Currency
	}
	if req.Status != nil {
		s := domain.TenderStatus(*req.Status)
		if !s.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STATUS", "status tidak valid")
		}
		t.Status = s
	}
	if req.PublishedDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PublishedDate)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format published_date tidak valid, gunakan RFC3339")
		}
		t.PublishedDate = &parsed
	}
	if req.SubmissionDeadline != nil {
		parsed, err := time.Parse(time.RFC3339, *req.SubmissionDeadline)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format submission_deadline tidak valid, gunakan RFC3339")
		}
		t.SubmissionDeadline = &parsed
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.Create: %w", err)
	}
	return t, nil
}

// Get returns a tender by ID.
func (s *TenderService) Get(ctx context.Context, id string) (*domain.Tender, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("tender tidak ditemukan")
		}
		return nil, fmt.Errorf("tender.Get: %w", err)
	}
	return t, nil
}

// List returns paginated tenders matching the filter.
func (s *TenderService) List(ctx context.Context, f domain.TenderFilter, page, pageSize int) ([]domain.Tender, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	return s.repo.List(ctx, f, page, pageSize)
}

// Update applies a partial update to a tender.
func (s *TenderService) Update(ctx context.Context, id string, req *dto.TenderUpdateRequest) (*domain.Tender, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.BuyerName != nil {
		t.BuyerName = req.BuyerName
	}
	if req.BuyerCountry != nil {
		t.BuyerCountry = req.BuyerCountry
	}
	if req.BuyerIndustry != nil {
		t.BuyerIndustry = req.BuyerIndustry
	}
	if req.ValueEstimate != nil {
		t.ValueEstimate = req.ValueEstimate
	}
	if req.Currency != nil {
		t.Currency = *req.Currency
	}
	if req.SourceName != nil {
		t.SourceName = req.SourceName
	}
	if req.SourceURL != nil {
		t.SourceURL = req.SourceURL
	}
	if req.ServiceCategory != nil {
		t.ServiceCategory = req.ServiceCategory
	}
	if req.ScopeSummary != nil {
		t.ScopeSummary = req.ScopeSummary
	}
	if req.EligibilityRequirements != nil {
		t.EligibilityRequirements = req.EligibilityRequirements
	}
	if req.TechnicalRequirements != nil {
		t.TechnicalRequirements = req.TechnicalRequirements
	}
	if req.Status != nil {
		newStatus := domain.TenderStatus(*req.Status)
		if !newStatus.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STATUS", "status tidak valid")
		}
		t.Status = newStatus
	}
	if req.PublishedDate != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PublishedDate)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format published_date tidak valid, gunakan RFC3339")
		}
		t.PublishedDate = &parsed
	}
	if req.SubmissionDeadline != nil {
		parsed, err := time.Parse(time.RFC3339, *req.SubmissionDeadline)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format submission_deadline tidak valid, gunakan RFC3339")
		}
		t.SubmissionDeadline = &parsed
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.Update: %w", err)
	}
	return t, nil
}

// Delete removes a tender by ID.
func (s *TenderService) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("tender.Delete: %w", err)
	}
	return nil
}

// UpdateStatus validates and applies a status transition.
func (s *TenderService) UpdateStatus(ctx context.Context, id string, to domain.TenderStatus) (*domain.Tender, error) {
	if !to.Valid() {
		return nil, httperr.NewBadRequest("INVALID_STATUS", "status tidak valid")
	}

	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if !canTransition(t.Status, to) {
		return nil, httperr.NewBadRequest(
			"INVALID_TRANSITION",
			fmt.Sprintf("transisi dari %s ke %s tidak diizinkan", t.Status, to),
		)
	}

	t.Status = to
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.UpdateStatus: %w", err)
	}
	return t, nil
}

// Promote moves a discovery-origin tender from inbox to the active pipeline
// by transitioning its status from IDENTIFIED to QUALIFYING.
func (s *TenderService) Promote(ctx context.Context, id string) (*domain.Tender, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if t.Origin != domain.OriginDiscovery {
		return nil, httperr.NewBadRequest("NOT_DISCOVERY", "hanya tender temuan AI yang bisa dipromosikan")
	}
	if t.Status != domain.TenderStatusIdentified {
		return nil, httperr.NewBadRequest("ALREADY_PROMOTED", "tender sudah keluar dari inbox discovery")
	}

	t.Status = domain.TenderStatusQualifying
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.Promote: %w", err)
	}
	return t, nil
}

// RecordOutcome records a WON/LOST outcome, creates an outcome_event row,
// sets tender status to the terminal state, and notifies the learning hook.
func (s *TenderService) RecordOutcome(ctx context.Context, id string, result domain.OutcomeResult, notes string) (*domain.Tender, error) {
	if !result.Valid() {
		return nil, httperr.NewBadRequest("INVALID_RESULT", "result harus WON atau LOST")
	}

	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	targetStatus := domain.TenderStatusLost
	if result == domain.OutcomeWon {
		targetStatus = domain.TenderStatusWon
	}
	if !canTransition(t.Status, targetStatus) {
		return nil, httperr.NewBadRequest(
			"INVALID_TRANSITION",
			fmt.Sprintf("tidak bisa merekam outcome %s dari status %s", result, t.Status),
		)
	}

	if _, err := recordOutcome(ctx, s.outcomes, s.learn, domain.OutcomeTargetTender, t.ID, result, notes); err != nil {
		return nil, fmt.Errorf("tender.RecordOutcome: %w", err)
	}

	t.Status = targetStatus
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("tender.RecordOutcome update status: %w", err)
	}

	return t, nil
}
