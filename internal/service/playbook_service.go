package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
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
	// events (nil-able, set via SetEvents) melengkapi target playbook:
	// selain tender/prospect, playbook bisa dibuat untuk event tertentu.
	events *EventService
}

// SetEvents wires the event target support after construction — optional,
// nil-safe (pattern yang sama dengan TenderService.SetEmitter).
func (s *PlaybookService) SetEvents(ev *EventService) { s.events = ev }

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
	case ai.ScoreTargetEvent:
		if s.events == nil {
			return nil, httperr.NewBadRequest("INVALID_TARGET_TYPE", "target event belum dikonfigurasi")
		}
		ev, err := s.events.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		in = ai.ScoreInputFromEvent(*ev, profile)
	default:
		return nil, httperr.NewBadRequest("INVALID_TARGET_TYPE", "target_type harus tender, prospect, atau event")
	}

	content, err := s.gen.Generate(ctx, in)
	if err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Generate playbook AI sedang tidak tersedia, coba lagi nanti")
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("playbook.Generate: marshal content: %w", err)
	}

	return s.persistNewVersion(ctx, string(targetType), id, contentJSON)
}

// persistNewVersion assigns version = latest+1 and persists. GetLatestVersion+1
// is a check-then-create race under concurrent generation for the same target;
// the UNIQUE (target_type, target_id, version) constraint (0015_playbook.up.sql)
// is the authority. On a unique-violation, another generation just took our
// version number — re-read and retry with the bumped one rather than failing
// the request and discarding the (already paid-for) AI content.
func (s *PlaybookService) persistNewVersion(ctx context.Context, targetType, targetID string, contentJSON []byte) (*domain.Playbook, error) {
	model := ai.ModelLabel
	var pb *domain.Playbook
	const maxVersionRetries = 3
	for attempt := 0; ; attempt++ {
		latestVersion, verr := s.repo.GetLatestVersion(ctx, targetType, targetID)
		if verr != nil {
			return nil, fmt.Errorf("playbook.persistNewVersion: latest version: %w", verr)
		}

		pb = &domain.Playbook{
			TargetType: targetType,
			TargetID:   targetID,
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
		return nil, fmt.Errorf("playbook.persistNewVersion: %w", cerr)
	}

	return pb, nil
}

// GenerateFromDocument menyusun playbook dari dokumen yang diunggah (PDF) —
// dibaca via vision oleh bridge — dan mempersistnya sebagai versi baru pada
// target yang sama (immutable versioning, sama seperti Generate).
func (s *PlaybookService) GenerateFromDocument(ctx context.Context, targetType ai.ScoreTargetType, id string, pdfBytes []byte, filename string) (*domain.Playbook, error) {
	profile, err := s.profile.GetCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("playbook.GenerateFromDocument: profile: %w", err)
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
	case ai.ScoreTargetEvent:
		if s.events == nil {
			return nil, httperr.NewBadRequest("INVALID_TARGET_TYPE", "target event belum dikonfigurasi")
		}
		ev, err := s.events.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		in = ai.ScoreInputFromEvent(*ev, profile)
	default:
		return nil, httperr.NewBadRequest("INVALID_TARGET_TYPE", "target_type harus tender, prospect, atau event")
	}

	content, err := s.gen.GenerateFromDocument(ctx, in, pdfBytes, filename)
	if err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Generate playbook dari dokumen sedang tidak tersedia, coba lagi nanti")
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("playbook.GenerateFromDocument: marshal content: %w", err)
	}
	return s.persistNewVersion(ctx, string(targetType), id, contentJSON)
}

// Refine merevisi playbook existing dengan instruksi bebas user dan
// mempersist hasilnya sebagai VERSI BARU pada target yang sama — versi lama
// tidak pernah diubah, sehingga revisi selalu bisa dibandingkan/di-rollback.
func (s *PlaybookService) Refine(ctx context.Context, playbookID, instruction string) (*domain.Playbook, error) {
	existing, err := s.GetByID(ctx, playbookID)
	if err != nil {
		return nil, err
	}

	var current ai.PlaybookContent
	if err := json.Unmarshal(existing.Content, &current); err != nil {
		return nil, fmt.Errorf("playbook.Refine: unmarshal existing: %w", err)
	}

	content, err := s.gen.Refine(ctx, &current, instruction)
	if err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Revisi playbook AI sedang tidak tersedia, coba lagi nanti")
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("playbook.Refine: marshal content: %w", err)
	}
	return s.persistNewVersion(ctx, existing.TargetType, existing.TargetID, contentJSON)
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

// GenerateCustom membuat playbook mandiri dari topik bebas user (menu
// Playbooks) — target_type 'custom' dengan target_id UUID baru, sehingga
// tiap playbook custom punya jalur versinya sendiri (refine tetap bekerja).
func (s *PlaybookService) GenerateCustom(ctx context.Context, topic string) (*domain.Playbook, error) {
	profile, err := s.profile.GetCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("playbook.GenerateCustom: profile: %w", err)
	}

	content, err := s.gen.GenerateCustomTopic(ctx, topic, profile)
	if err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Generate playbook AI sedang tidak tersedia, coba lagi nanti")
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("playbook.GenerateCustom: marshal content: %w", err)
	}
	return s.persistNewVersion(ctx, string(ai.ScoreTargetCustom), uuid.NewString(), contentJSON)
}

// ListCustom returns the LATEST version of every custom playbook, newest
// first — daftar untuk menu Playbooks.
func (s *PlaybookService) ListCustom(ctx context.Context) ([]domain.Playbook, error) {
	rows, err := s.repo.ListByTargetType(ctx, string(ai.ScoreTargetCustom))
	if err != nil {
		return nil, fmt.Errorf("playbook.ListCustom: %w", err)
	}
	// rows terurut terbaru dulu; simpan hanya versi tertinggi per target.
	seen := map[string]bool{}
	var out []domain.Playbook
	for _, r := range rows {
		if seen[r.TargetID] {
			continue
		}
		seen[r.TargetID] = true
		out = append(out, r)
	}
	return out, nil
}
