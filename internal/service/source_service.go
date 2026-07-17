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
)

// SourcePreset is one hardcoded catalog entry that can be activated with a
// single click (ST-08.3 AC). Login-gated sources are marked so discovery
// (EP-12) knows not to crawl them (PRD §9 compliance).
type SourcePreset struct {
	Key       string
	Name      string
	URL       string
	Country   string
	Access    domain.SourceAccess
	LegalNote string
}

// sourcePresets is the Indonesia procurement source catalog (ST-08.3 AC).
var sourcePresets = []SourcePreset{
	{
		Key:       "spse-lkpp",
		Name:      "SPSE / Inaproc (LKPP)",
		URL:       "https://inaproc.id",
		Country:   "Indonesia",
		Access:    domain.SourceAccessPublik,
		LegalNote: "Portal resmi pengadaan pemerintah",
	},
	{
		Key:       "eproc-pln",
		Name:      "eProc PLN",
		URL:       "https://eproc.pln.co.id",
		Country:   "Indonesia",
		Access:    domain.SourceAccessLogin,
		LegalNote: "Butuh akun vendor — tidak di-crawl otomatis",
	},
	{
		Key:       "eproc-pertamina",
		Name:      "eProc Pertamina",
		URL:       "https://eproc.pertamina.com",
		Country:   "Indonesia",
		Access:    domain.SourceAccessLogin,
		LegalNote: "Butuh akun vendor — tidak di-crawl otomatis",
	},
	{
		Key:       "smile-telkom",
		Name:      "Telkom SMILE",
		URL:       "https://smile.telkom.co.id",
		Country:   "Indonesia",
		Access:    domain.SourceAccessLogin,
		LegalNote: "Butuh akun vendor — tidak di-crawl otomatis",
	},
	{
		Key:       "padi-umkm",
		Name:      "PaDi UMKM",
		URL:       "https://padiumkm.id",
		Country:   "Indonesia",
		Access:    domain.SourceAccessPublik,
		LegalNote: "Marketplace BUMN",
	},
}

// SourceService handles CRUD + preset activation for crawling sources.
type SourceService struct {
	repo domain.SourceRepository
}

func NewSourceService(repo domain.SourceRepository) *SourceService {
	return &SourceService{repo: repo}
}

func (s *SourceService) Create(ctx context.Context, req *dto.SourceCreateRequest) (*domain.Source, error) {
	src := &domain.Source{
		Name:      req.Name,
		URL:       req.URL,
		Access:    domain.SourceAccessPublik,
		Frequency: "harian",
		DataTypes: []string{},
	}
	src.Country = req.Country
	src.LegalNote = req.LegalNote

	if req.Access != nil {
		access := domain.SourceAccess(*req.Access)
		if !access.Valid() {
			return nil, httperr.NewBadRequest("INVALID_ACCESS", "access tidak valid")
		}
		src.Access = access
	}
	if req.Enabled != nil {
		src.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		src.Priority = *req.Priority
	}
	if req.Frequency != nil {
		src.Frequency = *req.Frequency
	}
	if req.DataTypes != nil {
		src.DataTypes = req.DataTypes
	}

	if err := s.repo.Create(ctx, src); err != nil {
		return nil, fmt.Errorf("source.Create: %w", err)
	}
	return src, nil
}

func (s *SourceService) Get(ctx context.Context, id string) (*domain.Source, error) {
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("sumber tidak ditemukan")
		}
		return nil, fmt.Errorf("source.Get: %w", err)
	}
	return src, nil
}

func (s *SourceService) List(ctx context.Context, f domain.SourceFilter, page, pageSize int) ([]domain.Source, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	return s.repo.List(ctx, f, page, pageSize)
}

func (s *SourceService) Update(ctx context.Context, id string, req *dto.SourceUpdateRequest) (*domain.Source, error) {
	src, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		src.Name = *req.Name
	}
	if req.URL != nil {
		src.URL = *req.URL
	}
	if req.Country != nil {
		src.Country = req.Country
	}
	if req.Access != nil {
		access := domain.SourceAccess(*req.Access)
		if !access.Valid() {
			return nil, httperr.NewBadRequest("INVALID_ACCESS", "access tidak valid")
		}
		src.Access = access
	}
	if req.LegalNote != nil {
		src.LegalNote = req.LegalNote
	}
	if req.Enabled != nil {
		src.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		src.Priority = *req.Priority
	}
	if req.Frequency != nil {
		src.Frequency = *req.Frequency
	}
	if req.DataTypes != nil {
		src.DataTypes = req.DataTypes
	}

	if err := s.repo.Update(ctx, src); err != nil {
		return nil, fmt.Errorf("source.Update: %w", err)
	}
	return src, nil
}

func (s *SourceService) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("source.Delete: %w", err)
	}
	return nil
}

// Presets returns the hardcoded catalog annotated with whether each entry
// has already been activated in this workspace.
func (s *SourceService) Presets(ctx context.Context) ([]dto.SourcePresetResponse, error) {
	keys, err := s.repo.ListPresetKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("source.Presets: %w", err)
	}
	activated := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		activated[k] = struct{}{}
	}

	out := make([]dto.SourcePresetResponse, len(sourcePresets))
	for i, p := range sourcePresets {
		_, isActivated := activated[p.Key]
		out[i] = dto.SourcePresetResponse{
			Key:       p.Key,
			Name:      p.Name,
			URL:       p.URL,
			Country:   p.Country,
			Access:    string(p.Access),
			LegalNote: p.LegalNote,
			Activated: isActivated,
		}
	}
	return out, nil
}

// ActivatePreset turns on (or creates) the source row for a catalog entry.
// Idempotent: calling it again on an already-activated preset is a no-op
// beyond re-asserting enabled=true.
func (s *SourceService) ActivatePreset(ctx context.Context, key string) (*domain.Source, error) {
	var preset *SourcePreset
	for i := range sourcePresets {
		if sourcePresets[i].Key == key {
			preset = &sourcePresets[i]
			break
		}
	}
	if preset == nil {
		return nil, httperr.NewBadRequest("UNKNOWN_PRESET", "preset sumber tidak dikenal")
	}

	existing, err := s.repo.GetByPresetKey(ctx, key)
	if err == nil {
		existing.Enabled = true
		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("source.ActivatePreset update: %w", err)
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("source.ActivatePreset lookup: %w", err)
	}

	presetKey := preset.Key
	country := preset.Country
	legalNote := preset.LegalNote
	src := &domain.Source{
		Name:      preset.Name,
		URL:       preset.URL,
		Country:   &country,
		Access:    preset.Access,
		LegalNote: &legalNote,
		Enabled:   true,
		PresetKey: &presetKey,
		Frequency: "harian",
		DataTypes: []string{},
	}
	if err := s.repo.Create(ctx, src); err != nil {
		return nil, fmt.Errorf("source.ActivatePreset create: %w", err)
	}
	return src, nil
}
