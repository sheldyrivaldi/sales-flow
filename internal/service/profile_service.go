package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"os"

	"gorm.io/gorm"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/storage"
)

// maxUploadBytes caps Company Profile PDF ingest uploads (EP-13 ST-13.1) at
// 10 MB — generous for a text-based capability deck / company profile PDF
// without letting a request hold an unbounded amount of memory (SavePDF
// reads the whole body to validate the PDF magic bytes before writing).
const maxUploadBytes = 10 * 1024 * 1024

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

// defaultScoringConfig mirrors the hardcoded rubric weights/thresholds in
// internal/ai/scoring.go (scoringRubric) and internal/ai/recommend.go
// (80/65/50) — a never-configured workspace scores identically whether or
// not this row has been persisted yet (ai.buildScoringPrompt/RecommendAction
// fall back to the same constants when ScoringConfig is nil).
func defaultScoringConfig() *domain.ScoringConfig {
	return &domain.ScoringConfig{
		WeightCapabilityFit:             20,
		WeightPortfolioMatch:            15,
		WeightCommercialAttractiveness:  15,
		WeightEligibilityFit:            15,
		WeightDeadlineFeasibility:       10,
		WeightStrategicAccountValue:     10,
		WeightDeliveryRisk:              10,
		WeightCompetitionWinProbability: 5,
		ThresholdPursue:                 80,
		ThresholdReview:                 65,
		ThresholdWatchlist:              50,
	}
}

// ProfileService handles read/write of the versioned CompanyProfile aggregate.
type ProfileService struct {
	repo      domain.ProfileRepository
	uploadDir string
	extractor *ai.Extractor    // nil-able: PDF ingest still stores the file (degraded=true) without AI extraction
	crawl     *CrawlAutomation // nil-able: crawl automation upsert (EP-12) simply doesn't sync to Hermes without it
}

func NewProfileService(repo domain.ProfileRepository, uploadDir string, extractor *ai.Extractor, crawl *CrawlAutomation) *ProfileService {
	return &ProfileService{repo: repo, uploadDir: uploadDir, extractor: extractor, crawl: crawl}
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
			Products:          []string{},
			SourceDocRefs:     []string{},
			PortfolioRefs:     []string{},
			SupportDocuments:  []string{},
			CrawlFrequency:    "harian",
			CrawlEnabled:      false,
			Version:           0,
			IsCurrent:         false,
		},
		Target: &domain.TargetCriteria{
			Countries:          []string{"Indonesia"},
			Industries:         []string{},
			ValueMin:           &defaultValueMin,
			Currency:           "IDR",
			DeadlineMinDays:    &defaultDeadlineMinDays,
			ProcurementTypes:   defaultProcurementTypes(),
			DocumentLanguages:  []string{},
			DecisionMakerRoles: []string{},
		},
		NoGo: &domain.NoGoRule{
			PresetFlags: []string{},
			Custom:      []string{},
		},
		Keywords:      []domain.KeywordSet{},
		ScoringConfig: defaultScoringConfig(),
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

// IngestResult is returned by IngestUpload. Draft is nil whenever AI
// extraction didn't happen or didn't succeed (Degraded=true in that case) —
// the upload itself has still succeeded (the file is stored, DocRef is
// valid) so the caller can always fall back to manual entry.
type IngestResult struct {
	DocRef   string
	Filename string
	Size     int64
	Draft    *ai.ProfileDraft
	Degraded bool
}

// IngestUpload validates and stores an uploaded PDF for Company Profile
// ingest (EP-13 ST-13.1/13.2), then best-effort sends the raw file to Hermes
// for vision-based field extraction (ai.Extractor — the model reads the PDF
// itself, not a Go-side text extraction pass). AI extraction failures
// (Hermes down/invalid JSON) never fail the request — they only set
// Degraded=true with Draft=nil, since the whole point of AI ingest is
// optional convenience on top of an upload that must always succeed on its
// own (PRD §8: "gagal AI → pesan ramah, data utuh").
func (s *ProfileService) IngestUpload(ctx context.Context, fh *multipart.FileHeader) (*IngestResult, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("profile.IngestUpload: open: %w", err)
	}
	defer func() { _ = f.Close() }()

	docRef, size, err := storage.SavePDF(s.uploadDir, f, maxUploadBytes)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrInvalidPDF):
			return nil, httperr.NewBadRequest("INVALID_FILE_TYPE", "berkas harus PDF yang valid")
		case errors.Is(err, storage.ErrFileTooLarge):
			return nil, httperr.NewBadRequest("FILE_TOO_LARGE", fmt.Sprintf("berkas melebihi batas ukuran %d MB", maxUploadBytes/(1024*1024)))
		default:
			return nil, fmt.Errorf("profile.IngestUpload: %w", err)
		}
	}

	result := &IngestResult{DocRef: docRef, Filename: fh.Filename, Size: size}

	if s.extractor == nil {
		result.Degraded = true
		return result, nil
	}

	path, err := storage.FullPath(s.uploadDir, docRef)
	if err != nil {
		log.Printf("profile: IngestUpload: FullPath gagal untuk doc_ref %q: %v", docRef, err)
		result.Degraded = true
		return result, nil
	}

	pdfBytes, err := os.ReadFile(path)
	if err != nil {
		log.Printf("profile: IngestUpload: baca berkas gagal (doc_ref=%s): %v", docRef, err)
		result.Degraded = true
		return result, nil
	}

	draft, err := s.extractor.Extract(ctx, pdfBytes, fh.Filename)
	if err != nil {
		log.Printf("profile: IngestUpload: ekstraksi field AI gagal (doc_ref=%s): %v", docRef, err)
		result.Degraded = true
		return result, nil
	}

	result.Draft = draft
	return result, nil
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
			Products:          coalesce(req.Products, base.Profile.Products),
			Vision:            coalesceStr(req.Vision, base.Profile.Vision),
			Mission:           coalesceStr(req.Mission, base.Profile.Mission),
			SourceDocRefs:     coalesce(req.SourceDocRefs, base.Profile.SourceDocRefs),
			PortfolioRefs:     coalesce(req.PortfolioRefs, base.Profile.PortfolioRefs),
			SupportDocuments:  coalesce(req.SupportDocuments, base.Profile.SupportDocuments),
			CrawlFrequency:    coalesceCrawlFrequency(req.CrawlFrequency, base.Profile.CrawlFrequency),
			CrawlEnabled:      coalesceBool(req.CrawlEnabled, base.Profile.CrawlEnabled),
		},
	}

	next.Target = mergeTarget(req.Target, base.Target)
	next.NoGo = mergeNoGo(req.NoGo, base.NoGo)
	next.Keywords = mergeKeywords(req.Keywords, base.Keywords)
	next.ScoringConfig = mergeScoringConfig(req.Scoring, base.ScoringConfig)

	created, err := s.repo.CreateVersion(ctx, next)
	if err != nil {
		return nil, fmt.Errorf("profile.Save: %w", err)
	}

	s.syncCrawlAutomation(ctx, created.Profile.CompanyName, created.Profile.CrawlEnabled, created.Profile.CrawlFrequency)

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
		Countries:          coalesce(req.Countries, base.Countries),
		Industries:         coalesce(req.Industries, base.Industries),
		ValueMin:           coalesceFloat(req.ValueMin, base.ValueMin),
		ValueIdeal:         coalesceFloat(req.ValueIdeal, base.ValueIdeal),
		ValueMax:           coalesceFloat(req.ValueMax, base.ValueMax),
		Currency:           base.Currency,
		DeadlineMinDays:    coalesceInt(req.DeadlineMinDays, base.DeadlineMinDays),
		ProcurementTypes:   coalesce(req.ProcurementTypes, base.ProcurementTypes),
		BuyerSizeNote:      coalesceStr(req.BuyerSizeNote, base.BuyerSizeNote),
		DocumentLanguages:  coalesce(req.DocumentLanguages, base.DocumentLanguages),
		WorkModel:          coalesceStr(req.WorkModel, base.WorkModel),
		OnsiteLimitNote:    coalesceStr(req.OnsiteLimitNote, base.OnsiteLimitNote),
		DecisionMakerRoles: coalesce(req.DecisionMakerRoles, base.DecisionMakerRoles),
	}
	if req.Currency != nil {
		t.Currency = *req.Currency
	}
	return t
}

// mergeScoringConfig applies the same coalesce-per-field pattern as
// mergeTarget/mergeNoGo: a nil req keeps base's config entirely (falling
// back to defaultScoringConfig() if base itself has never been configured);
// a non-nil req only overrides the fields it sets, so a partial tweak (e.g.
// changing one weight) never clobbers the rest.
func mergeScoringConfig(req *dto.ScoringConfigRequest, base *domain.ScoringConfig) *domain.ScoringConfig {
	if base == nil {
		base = defaultScoringConfig()
	}
	if req == nil {
		cp := *base
		return &cp
	}
	return &domain.ScoringConfig{
		WeightCapabilityFit:             coalesceIntVal(req.WeightCapabilityFit, base.WeightCapabilityFit),
		WeightPortfolioMatch:            coalesceIntVal(req.WeightPortfolioMatch, base.WeightPortfolioMatch),
		WeightCommercialAttractiveness:  coalesceIntVal(req.WeightCommercialAttractiveness, base.WeightCommercialAttractiveness),
		WeightEligibilityFit:            coalesceIntVal(req.WeightEligibilityFit, base.WeightEligibilityFit),
		WeightDeadlineFeasibility:       coalesceIntVal(req.WeightDeadlineFeasibility, base.WeightDeadlineFeasibility),
		WeightStrategicAccountValue:     coalesceIntVal(req.WeightStrategicAccountValue, base.WeightStrategicAccountValue),
		WeightDeliveryRisk:              coalesceIntVal(req.WeightDeliveryRisk, base.WeightDeliveryRisk),
		WeightCompetitionWinProbability: coalesceIntVal(req.WeightCompetitionWinProbability, base.WeightCompetitionWinProbability),
		ThresholdPursue:                 coalesceIntVal(req.ThresholdPursue, base.ThresholdPursue),
		ThresholdReview:                 coalesceIntVal(req.ThresholdReview, base.ThresholdReview),
		ThresholdWatchlist:              coalesceIntVal(req.ThresholdWatchlist, base.ThresholdWatchlist),
	}
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

// coalesceIntVal is coalesceInt's counterpart for plain (non-pointer) int
// fields like ScoringConfig's weights/thresholds: primary is the optional
// request override, fallback is the persisted value to keep when absent.
func coalesceIntVal(primary *int, fallback int) int {
	if primary != nil {
		return *primary
	}
	return fallback
}

func coalesceCrawlFrequency(primary *string, fallback string) string {
	if primary != nil {
		return *primary
	}
	return fallback
}

func coalesceBool(primary *bool, fallback bool) bool {
	if primary != nil {
		return *primary
	}
	return fallback
}
