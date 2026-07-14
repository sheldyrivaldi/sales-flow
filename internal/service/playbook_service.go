package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
)

// PlaybookService orchestrates one playbook generation run end-to-end
// (EP-14): resolve the target + Company Profile, call the AI generator, and
// persist a new immutable version (latest+1) — existing versions are never
// updated or deleted. Mirrors ScoreService's shape (EP-10) deliberately.
type PlaybookService struct {
	gen       *ai.PlaybookGenerator
	repo      domain.PlaybookRepository
	tenders   *TenderService
	prospects *ProspectService
	profile   *ProfileService
}

func NewPlaybookService(gen *ai.PlaybookGenerator, repo domain.PlaybookRepository, tenders *TenderService, prospects *ProspectService, profile *ProfileService) *PlaybookService {
	return &PlaybookService{gen: gen, repo: repo, tenders: tenders, prospects: prospects, profile: profile}
}

// Generate runs a fresh playbook generation for a tender or prospect and
// persists it as a new version (latest+1). On any AI failure (Hermes down,
// invalid output) it returns a friendly error and persists nothing — the
// prior version (if any) remains the latest, untouched (PRD §8: "gagal AI →
// pesan ramah, data utuh").
func (s *PlaybookService) Generate(ctx context.Context, targetType ai.ScoreTargetType, id string) (*domain.Playbook, error) {
	profile, err := s.profile.GetCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("playbook.Generate: profile: %w", err)
	}

	var in ai.ScoreInput
	switch targetType {
	case ai.ScoreTargetTender:
		t, err := s.tenders.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		in = ai.ScoreInputFromTender(*t, profile)
	case ai.ScoreTargetProspect:
		p, err := s.prospects.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		in = ai.ScoreInputFromProspect(*p, profile)
	default:
		return nil, httperr.NewBadRequest("INVALID_TARGET_TYPE", "target_type harus tender atau prospect")
	}

	content, err := s.gen.Generate(ctx, in)
	if err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Generate playbook AI sedang tidak tersedia, coba lagi nanti")
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("playbook.Generate: marshal content: %w", err)
	}

	// Assign version = latest+1 and persist. GetLatestVersion+1 is a
	// check-then-create race under concurrent generation for the same target;
	// the UNIQUE (target_type, target_id, version) constraint
	// (0015_playbook.up.sql) is the authority. On a unique-violation, another
	// generation just took our version number — re-read and retry with the
	// bumped one rather than failing the request and discarding the (already
	// paid-for) AI content.
	model := ai.ModelLabel
	var pb *domain.Playbook
	const maxVersionRetries = 3
	for attempt := 0; ; attempt++ {
		latestVersion, verr := s.repo.GetLatestVersion(ctx, string(targetType), id)
		if verr != nil {
			return nil, fmt.Errorf("playbook.Generate: latest version: %w", verr)
		}

		pb = &domain.Playbook{
			TargetType: string(targetType),
			TargetID:   id,
			Version:    latestVersion + 1,
			Content:    contentJSON,
			Model:      &model,
		}
		cerr := s.repo.Create(ctx, pb)
		if cerr == nil {
			break
		}
		if isUniqueViolation(cerr) && attempt < maxVersionRetries {
			continue
		}
		return nil, fmt.Errorf("playbook.Generate: %w", cerr)
	}

	return pb, nil
}

// ListByTarget returns all playbook versions for a target, newest first.
func (s *PlaybookService) ListByTarget(ctx context.Context, targetType ai.ScoreTargetType, id string) ([]domain.Playbook, error) {
	items, err := s.repo.ListByTarget(ctx, string(targetType), id)
	if err != nil {
		return nil, fmt.Errorf("playbook.ListByTarget: %w", err)
	}
	return items, nil
}

// GetByID returns one playbook version by id.
func (s *PlaybookService) GetByID(ctx context.Context, id string) (*domain.Playbook, error) {
	pb, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("playbook tidak ditemukan")
		}
		return nil, fmt.Errorf("playbook.GetByID: %w", err)
	}
	return pb, nil
}
