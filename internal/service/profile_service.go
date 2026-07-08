package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
)

// Defaults applied to a brand-new profile (PRD §10 / ST-08.2 AC): value_min
// Rp 1.000.000.000 (Rp 1 miliar), deadline_min_days 7, countries=[Indonesia],
// plus a standard Indonesian procurement-type preset.
var (
	defaultValueMin        = 1_000_000_000.0
	defaultDeadlineMinDays = 7
)

func defaultProcurementTypes() []string {
	return []string{"Barang", "Jasa Konsultansi", "Jasa Lainnya", "Pekerjaan Konstruksi"}
}

// ProfileService handles read/write of the versioned CompanyProfile aggregate.
type ProfileService struct {
	repo domain.ProfileRepository
}

func NewProfileService(repo domain.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

// defaultAggregate builds the template used both for a never-configured
// workspace (GetCurrent when no profile exists) and as the base merged
// against on the very first Save.
func defaultAggregate(companyName string) *domain.ProfileAggregate {
	return &domain.ProfileAggregate{
		Profile: domain.CompanyProfile{
			CompanyName:       companyName,
			ServiceCategories: []string{},
			TechStack:         []string{},
			SourceDocRefs:     []string{},
			Version:           0,
			IsCurrent:         false,
		},
		Target: &domain.TargetCriteria{
			Countries:        []string{"Indonesia"},
			Industries:       []string{},
			ValueMin:         &defaultValueMin,
			Currency:         "IDR",
			DeadlineMinDays:  &defaultDeadlineMinDays,
			ProcurementTypes: defaultProcurementTypes(),
		},
		NoGo: &domain.NoGoRule{
			PresetFlags: []string{},
			Custom:      []string{},
		},
		Keywords: []domain.KeywordSet{},
	}
}

// GetCurrent returns the current profile version, or a default (unsaved)
// template when the workspace has never configured one — this is a 200
// onboarding template, not a 404.
func (s *ProfileService) GetCurrent(ctx context.Context) (*domain.ProfileAggregate, error) {
	agg, err := s.repo.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultAggregate(""), nil
		}
		return nil, fmt.Errorf("profile.GetCurrent: %w", err)
	}
	return agg, nil
}

// Save merges req over the current version (falling back to defaults when no
// version exists yet) and persists the result as a brand-new version.
func (s *ProfileService) Save(ctx context.Context, req *dto.ProfileUpdateRequest) (*domain.ProfileAggregate, error) {
	base, err := s.repo.GetCurrent(ctx)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("profile.Save: %w", err)
		}
		base = defaultAggregate("")
	}

	next := &domain.ProfileAggregate{
		Profile: domain.CompanyProfile{
			CompanyName:       req.CompanyName,
			OneLiner:          coalesceStr(req.OneLiner, base.Profile.OneLiner),
			ServiceCategories: coalesce(req.ServiceCategories, base.Profile.ServiceCategories),
			TechStack:         coalesce(req.TechStack, base.Profile.TechStack),
			SourceDocRefs:     coalesce(req.SourceDocRefs, base.Profile.SourceDocRefs),
		},
	}

	next.Target = mergeTarget(req.Target, base.Target)
	next.NoGo = mergeNoGo(req.NoGo, base.NoGo)
	next.Keywords = mergeKeywords(req.Keywords, base.Keywords)

	created, err := s.repo.CreateVersion(ctx, next)
	if err != nil {
		return nil, fmt.Errorf("profile.Save: %w", err)
	}
	return created, nil
}

func mergeTarget(req *dto.TargetCriteriaRequest, base *domain.TargetCriteria) *domain.TargetCriteria {
	if base == nil {
		base = defaultAggregate("").Target
	}
	if req == nil {
		cp := *base
		return &cp
	}
	t := &domain.TargetCriteria{
		Countries:        coalesce(req.Countries, base.Countries),
		Industries:       coalesce(req.Industries, base.Industries),
		ValueMin:         coalesceFloat(req.ValueMin, base.ValueMin),
		ValueIdeal:       coalesceFloat(req.ValueIdeal, base.ValueIdeal),
		ValueMax:         coalesceFloat(req.ValueMax, base.ValueMax),
		Currency:         base.Currency,
		DeadlineMinDays:  coalesceInt(req.DeadlineMinDays, base.DeadlineMinDays),
		ProcurementTypes: coalesce(req.ProcurementTypes, base.ProcurementTypes),
	}
	if req.Currency != nil {
		t.Currency = *req.Currency
	}
	return t
}

func mergeNoGo(req *dto.NoGoRuleRequest, base *domain.NoGoRule) *domain.NoGoRule {
	if base == nil {
		base = defaultAggregate("").NoGo
	}
	if req == nil {
		cp := *base
		return &cp
	}
	return &domain.NoGoRule{
		PresetFlags: coalesce(req.PresetFlags, base.PresetFlags),
		Custom:      coalesce(req.Custom, base.Custom),
	}
}

func mergeKeywords(req []dto.KeywordSetRequest, base []domain.KeywordSet) []domain.KeywordSet {
	if req == nil {
		return base
	}
	out := make([]domain.KeywordSet, len(req))
	for i, k := range req {
		language := "id"
		if k.Language != nil {
			language = *k.Language
		}
		out[i] = domain.KeywordSet{
			Category:         k.Category,
			Keywords:         orEmptySlice(k.Keywords),
			NegativeKeywords: orEmptySlice(k.NegativeKeywords),
			Language:         language,
		}
	}
	return out
}

// coalesce returns primary when non-nil, else fallback. An explicit empty
// slice (non-nil, len 0) from the request is treated as "clear this field".
func coalesce(primary, fallback []string) []string {
	if primary != nil {
		return primary
	}
	return fallback
}

func orEmptySlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func coalesceStr(primary, fallback *string) *string {
	if primary != nil {
		return primary
	}
	return fallback
}

func coalesceFloat(primary, fallback *float64) *float64 {
	if primary != nil {
		return primary
	}
	return fallback
}

func coalesceInt(primary, fallback *int) *int {
	if primary != nil {
		return primary
	}
	return fallback
}
