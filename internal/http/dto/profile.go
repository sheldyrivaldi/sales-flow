package dto

import (
	"time"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
)

// --- Update (PUT /api/profile — always creates a new version) ---

type TargetCriteriaRequest struct {
	Countries          []string `json:"countries"            validate:"omitempty"`
	Industries         []string `json:"industries"           validate:"omitempty"`
	ValueMin           *float64 `json:"value_min"            validate:"omitempty,gte=0"`
	ValueIdeal         *float64 `json:"value_ideal"          validate:"omitempty,gte=0"`
	ValueMax           *float64 `json:"value_max"            validate:"omitempty,gte=0"`
	Currency           *string  `json:"currency"             validate:"omitempty"`
	DeadlineMinDays    *int     `json:"deadline_min_days"    validate:"omitempty,gte=0"`
	ProcurementTypes   []string `json:"procurement_types"    validate:"omitempty"`
	BuyerSizeNote      *string  `json:"buyer_size_note"      validate:"omitempty"`
	DocumentLanguages  []string `json:"document_languages"   validate:"omitempty"`
	WorkModel          *string  `json:"work_model"           validate:"omitempty"`
	OnsiteLimitNote    *string  `json:"onsite_limit_note"    validate:"omitempty"`
	DecisionMakerRoles []string `json:"decision_maker_roles" validate:"omitempty"`
}

type NoGoRuleRequest struct {
	PresetFlags []string `json:"preset_flags" validate:"omitempty"`
	Custom      []string `json:"custom"       validate:"omitempty"`
}

type KeywordSetRequest struct {
	Category         *string  `json:"category"`
	Keywords         []string `json:"keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
	Language         *string  `json:"language" validate:"omitempty"`
}

// ScoringConfigRequest carries the configurable rubric weights + thresholds
// (RFI §8). All fields optional/pointer so a partial PUT keeps the prior
// version's values (same coalesce convention as TargetCriteriaRequest) —
// weights are not validated to sum to 100 here since Ops is expected to
// iterate on them (RFI: "template awal, harus dikonfirmasi").
type ScoringConfigRequest struct {
	WeightCapabilityFit             *int `json:"weight_capability_fit"              validate:"omitempty,gte=0,lte=100"`
	WeightPortfolioMatch            *int `json:"weight_portfolio_match"             validate:"omitempty,gte=0,lte=100"`
	WeightCommercialAttractiveness  *int `json:"weight_commercial_attractiveness"   validate:"omitempty,gte=0,lte=100"`
	WeightEligibilityFit            *int `json:"weight_eligibility_fit"             validate:"omitempty,gte=0,lte=100"`
	WeightDeadlineFeasibility       *int `json:"weight_deadline_feasibility"        validate:"omitempty,gte=0,lte=100"`
	WeightStrategicAccountValue     *int `json:"weight_strategic_account_value"     validate:"omitempty,gte=0,lte=100"`
	WeightDeliveryRisk              *int `json:"weight_delivery_risk"               validate:"omitempty,gte=0,lte=100"`
	WeightCompetitionWinProbability *int `json:"weight_competition_win_probability" validate:"omitempty,gte=0,lte=100"`
	ThresholdPursue                 *int `json:"threshold_pursue"                   validate:"omitempty,gte=0,lte=100"`
	ThresholdReview                 *int `json:"threshold_review"                   validate:"omitempty,gte=0,lte=100"`
	ThresholdWatchlist              *int `json:"threshold_watchlist"                validate:"omitempty,gte=0,lte=100"`
}

type ProfileUpdateRequest struct {
	CompanyName       string                 `json:"company_name"        validate:"required"`
	OneLiner          *string                `json:"one_liner"`
	ServiceCategories []string               `json:"service_categories"`
	TechStack         []string               `json:"tech_stack"`
	Products          []string               `json:"products"`
	Vision            *string                `json:"vision"`
	Mission           *string                `json:"mission"`
	SourceDocRefs     []string               `json:"source_doc_refs"`
	PortfolioRefs     []string               `json:"portfolio_refs"`
	SupportDocuments  []string               `json:"support_documents"`
	CrawlFrequency    *string                `json:"crawl_frequency" validate:"omitempty,oneof=harian 2-3x mingguan"`
	CrawlEnabled      *bool                  `json:"crawl_enabled"`
	Target            *TargetCriteriaRequest `json:"target"   validate:"omitempty"`
	NoGo              *NoGoRuleRequest       `json:"nogo"     validate:"omitempty"`
	Keywords          []KeywordSetRequest    `json:"keywords" validate:"omitempty,dive"`
	Scoring           *ScoringConfigRequest  `json:"scoring"  validate:"omitempty"`
}

// --- Ingest (POST /api/profile/ingest — EP-13 PDF Ingest) ---

// IngestResponse is returned by POST /api/profile/ingest. Draft is nil and
// Degraded is false when the PDF has no text layer or Hermes is unavailable
// — the upload itself still succeeds (AI extraction is non-blocking), the
// caller just falls back to manual entry.
type IngestResponse struct {
	DocRef   string           `json:"doc_ref"`
	Filename string           `json:"filename"`
	Size     int64            `json:"size"`
	Draft    *ai.ProfileDraft `json:"draft"`
	Degraded bool             `json:"degraded"`
}

// --- Response ---

type TargetCriteriaResponse struct {
	Countries          []string `json:"countries"`
	Industries         []string `json:"industries"`
	ValueMin           *float64 `json:"value_min"`
	ValueIdeal         *float64 `json:"value_ideal"`
	ValueMax           *float64 `json:"value_max"`
	Currency           string   `json:"currency"`
	DeadlineMinDays    *int     `json:"deadline_min_days"`
	ProcurementTypes   []string `json:"procurement_types"`
	BuyerSizeNote      *string  `json:"buyer_size_note"`
	DocumentLanguages  []string `json:"document_languages"`
	WorkModel          *string  `json:"work_model"`
	OnsiteLimitNote    *string  `json:"onsite_limit_note"`
	DecisionMakerRoles []string `json:"decision_maker_roles"`
}

type NoGoRuleResponse struct {
	PresetFlags []string `json:"preset_flags"`
	Custom      []string `json:"custom"`
}

type KeywordSetResponse struct {
	ID               string   `json:"id"`
	Category         *string  `json:"category"`
	Keywords         []string `json:"keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
	Language         string   `json:"language"`
}

type ScoringConfigResponse struct {
	WeightCapabilityFit             int `json:"weight_capability_fit"`
	WeightPortfolioMatch            int `json:"weight_portfolio_match"`
	WeightCommercialAttractiveness  int `json:"weight_commercial_attractiveness"`
	WeightEligibilityFit            int `json:"weight_eligibility_fit"`
	WeightDeadlineFeasibility       int `json:"weight_deadline_feasibility"`
	WeightStrategicAccountValue     int `json:"weight_strategic_account_value"`
	WeightDeliveryRisk              int `json:"weight_delivery_risk"`
	WeightCompetitionWinProbability int `json:"weight_competition_win_probability"`
	ThresholdPursue                 int `json:"threshold_pursue"`
	ThresholdReview                 int `json:"threshold_review"`
	ThresholdWatchlist              int `json:"threshold_watchlist"`
}

// ProfileResponse is the full "Otak Agent" snapshot for the current version.
type ProfileResponse struct {
	ID                string                  `json:"id"`
	CompanyName       string                  `json:"company_name"`
	OneLiner          *string                 `json:"one_liner"`
	ServiceCategories []string                `json:"service_categories"`
	TechStack         []string                `json:"tech_stack"`
	Products          []string                `json:"products"`
	Vision            *string                 `json:"vision"`
	Mission           *string                 `json:"mission"`
	SourceDocRefs     []string                `json:"source_doc_refs"`
	PortfolioRefs     []string                `json:"portfolio_refs"`
	SupportDocuments  []string                `json:"support_documents"`
	CrawlFrequency    string                  `json:"crawl_frequency"`
	CrawlEnabled      bool                    `json:"crawl_enabled"`
	Version           int                     `json:"version"`
	IsCurrent         bool                    `json:"is_current"`
	Target            *TargetCriteriaResponse `json:"target"`
	NoGo              *NoGoRuleResponse       `json:"nogo"`
	Keywords          []KeywordSetResponse    `json:"keywords"`
	Scoring           *ScoringConfigResponse  `json:"scoring"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

// ToProfileResponse maps a domain.ProfileAggregate → ProfileResponse.
func ToProfileResponse(agg domain.ProfileAggregate) ProfileResponse {
	resp := ProfileResponse{
		ID:                agg.Profile.ID,
		CompanyName:       agg.Profile.CompanyName,
		OneLiner:          agg.Profile.OneLiner,
		ServiceCategories: orEmpty(agg.Profile.ServiceCategories),
		TechStack:         orEmpty(agg.Profile.TechStack),
		Products:          orEmpty(agg.Profile.Products),
		Vision:            agg.Profile.Vision,
		Mission:           agg.Profile.Mission,
		SourceDocRefs:     orEmpty(agg.Profile.SourceDocRefs),
		PortfolioRefs:     orEmpty(agg.Profile.PortfolioRefs),
		SupportDocuments:  orEmpty(agg.Profile.SupportDocuments),
		CrawlFrequency:    agg.Profile.CrawlFrequency,
		CrawlEnabled:      agg.Profile.CrawlEnabled,
		Version:           agg.Profile.Version,
		IsCurrent:         agg.Profile.IsCurrent,
		CreatedAt:         agg.Profile.CreatedAt,
		UpdatedAt:         agg.Profile.UpdatedAt,
	}

	if agg.Target != nil {
		resp.Target = &TargetCriteriaResponse{
			Countries:          orEmpty(agg.Target.Countries),
			Industries:         orEmpty(agg.Target.Industries),
			ValueMin:           agg.Target.ValueMin,
			ValueIdeal:         agg.Target.ValueIdeal,
			ValueMax:           agg.Target.ValueMax,
			Currency:           agg.Target.Currency,
			DeadlineMinDays:    agg.Target.DeadlineMinDays,
			ProcurementTypes:   orEmpty(agg.Target.ProcurementTypes),
			BuyerSizeNote:      agg.Target.BuyerSizeNote,
			DocumentLanguages:  orEmpty(agg.Target.DocumentLanguages),
			WorkModel:          agg.Target.WorkModel,
			OnsiteLimitNote:    agg.Target.OnsiteLimitNote,
			DecisionMakerRoles: orEmpty(agg.Target.DecisionMakerRoles),
		}
	}

	if agg.NoGo != nil {
		resp.NoGo = &NoGoRuleResponse{
			PresetFlags: orEmpty(agg.NoGo.PresetFlags),
			Custom:      orEmpty(agg.NoGo.Custom),
		}
	}

	resp.Keywords = make([]KeywordSetResponse, len(agg.Keywords))
	for i, k := range agg.Keywords {
		resp.Keywords[i] = KeywordSetResponse{
			ID:               k.ID,
			Category:         k.Category,
			Keywords:         orEmpty(k.Keywords),
			NegativeKeywords: orEmpty(k.NegativeKeywords),
			Language:         k.Language,
		}
	}

	if agg.ScoringConfig != nil {
		sc := agg.ScoringConfig
		resp.Scoring = &ScoringConfigResponse{
			WeightCapabilityFit:             sc.WeightCapabilityFit,
			WeightPortfolioMatch:            sc.WeightPortfolioMatch,
			WeightCommercialAttractiveness:  sc.WeightCommercialAttractiveness,
			WeightEligibilityFit:            sc.WeightEligibilityFit,
			WeightDeadlineFeasibility:       sc.WeightDeadlineFeasibility,
			WeightStrategicAccountValue:     sc.WeightStrategicAccountValue,
			WeightDeliveryRisk:              sc.WeightDeliveryRisk,
			WeightCompetitionWinProbability: sc.WeightCompetitionWinProbability,
			ThresholdPursue:                 sc.ThresholdPursue,
			ThresholdReview:                 sc.ThresholdReview,
			ThresholdWatchlist:              sc.ThresholdWatchlist,
		}
	}

	return resp
}

// orEmpty ensures a nil slice serializes as `[]` instead of `null`.
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
