package domain

import (
	"context"
	"time"
)

// CompanyProfile is the versioned root of the "Otak Agent" knowledge base.
// Only one row has IsCurrent=true at any time (enforced by a partial unique
// index in the DB); each PUT creates a new version and clones the child
// entities (TargetCriteria, NoGoRule, KeywordSet) so full history is kept.
type CompanyProfile struct {
	ID                string   `json:"id"                  gorm:"primaryKey;default:gen_random_uuid()"`
	CompanyName       string   `json:"company_name"        gorm:"column:company_name;not null"`
	OneLiner          *string  `json:"one_liner"            gorm:"column:one_liner"`
	ServiceCategories []string `json:"service_categories"   gorm:"column:service_categories;serializer:json;type:jsonb"`
	TechStack         []string `json:"tech_stack"           gorm:"column:tech_stack;serializer:json;type:jsonb"`
	// Products, Vision, Mission feed directly into the discovery prompt
	// (ai.buildDiscoveryPrompt) so Hermes can judge tender relevance against
	// what the company actually sells and aims for — not just keywords.
	Products      []string `json:"products"             gorm:"column:products;serializer:json;type:jsonb"`
	Vision        *string  `json:"vision"                gorm:"column:vision"`
	Mission       *string  `json:"mission"               gorm:"column:mission"`
	SourceDocRefs []string `json:"source_doc_refs"      gorm:"column:source_doc_refs;serializer:json;type:jsonb"`
	// PortfolioRefs are free-text evidence references (client/project names,
	// case study links) backing the capabilities above — RFI §4.1
	// "Bukti/Portfolio", which the source RFI treats as the single most
	// important input for judging tender relevance against real capability.
	PortfolioRefs []string `json:"portfolio_refs"       gorm:"column:portfolio_refs;serializer:json;type:jsonb"`
	// CrawlFrequency drives the discovery scheduler (EP-12 ST-12.5.1): one of
	// "harian" (daily), "2-3x" (2-3x/week), "mingguan" (weekly). CrawlEnabled
	// defaults to false so a freshly onboarded workspace never auto-crawls
	// until the user deliberately turns it on (PRD §10 Kartu 5).
	CrawlFrequency string    `json:"crawl_frequency"      gorm:"column:crawl_frequency;not null;default:'harian'"`
	CrawlEnabled   bool      `json:"crawl_enabled"        gorm:"column:crawl_enabled;not null;default:false"`
	Version        int       `json:"version"              gorm:"not null;default:1"`
	IsCurrent      bool      `json:"is_current"           gorm:"column:is_current;not null;default:true"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (CompanyProfile) TableName() string { return "company_profile" }

// TargetCriteria holds buyer/opportunity scoring criteria for one profile version.
type TargetCriteria struct {
	ID               string   `json:"id"                 gorm:"primaryKey;default:gen_random_uuid()"`
	ProfileID        string   `json:"profile_id"         gorm:"column:profile_id;not null"`
	Countries        []string `json:"countries"          gorm:"serializer:json;type:jsonb"`
	Industries       []string `json:"industries"         gorm:"serializer:json;type:jsonb"`
	ValueMin         *float64 `json:"value_min"          gorm:"column:value_min"`
	ValueIdeal       *float64 `json:"value_ideal"        gorm:"column:value_ideal"`
	ValueMax         *float64 `json:"value_max"          gorm:"column:value_max"`
	Currency         string   `json:"currency"           gorm:"not null;default:'IDR'"`
	DeadlineMinDays  *int     `json:"deadline_min_days"  gorm:"column:deadline_min_days"`
	ProcurementTypes []string `json:"procurement_types"  gorm:"column:procurement_types;serializer:json;type:jsonb"`
	// Buyer-profile fields from RFI §5.1 with no home in the original schema:
	// buyer scale, tender document language, work model/onsite limits, and
	// which decision-maker roles to target for contact enrichment.
	BuyerSizeNote      *string   `json:"buyer_size_note"       gorm:"column:buyer_size_note"`
	DocumentLanguages  []string  `json:"document_languages"    gorm:"column:document_languages;serializer:json;type:jsonb"`
	WorkModel          *string   `json:"work_model"            gorm:"column:work_model"`
	OnsiteLimitNote    *string   `json:"onsite_limit_note"     gorm:"column:onsite_limit_note"`
	DecisionMakerRoles []string  `json:"decision_maker_roles"  gorm:"column:decision_maker_roles;serializer:json;type:jsonb"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (TargetCriteria) TableName() string { return "target_criteria" }

// NoGoRule holds auto-reject conditions for one profile version.
type NoGoRule struct {
	ID          string    `json:"id"           gorm:"primaryKey;default:gen_random_uuid()"`
	ProfileID   string    `json:"profile_id"   gorm:"column:profile_id;not null"`
	PresetFlags []string  `json:"preset_flags" gorm:"column:preset_flags;serializer:json;type:jsonb"`
	Custom      []string  `json:"custom"       gorm:"serializer:json;type:jsonb"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (NoGoRule) TableName() string { return "nogo_rule" }

// KeywordSet holds discovery search keywords for one category, for one
// profile version. A profile can have multiple keyword sets (one per category).
type KeywordSet struct {
	ID               string    `json:"id"                gorm:"primaryKey;default:gen_random_uuid()"`
	ProfileID        string    `json:"profile_id"        gorm:"column:profile_id;not null"`
	Category         *string   `json:"category"`
	Keywords         []string  `json:"keywords"          gorm:"serializer:json;type:jsonb"`
	NegativeKeywords []string  `json:"negative_keywords" gorm:"column:negative_keywords;serializer:json;type:jsonb"`
	Language         string    `json:"language"          gorm:"not null;default:'id'"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (KeywordSet) TableName() string { return "keyword_set" }

// ScoringConfig holds the configurable rubric weights and recommendation
// thresholds for one profile version (RFI §8: "bobot di atas adalah template
// awal dan harus dikonfirmasi oleh Ops"). Weights are read by
// ai.buildScoringPrompt (falling back to the hardcoded defaults below when
// nil, so an unconfigured workspace scores exactly as it did before this
// existed); thresholds are read by ai.RecommendAction the same way.
type ScoringConfig struct {
	ID                              string    `json:"id"                                  gorm:"primaryKey;default:gen_random_uuid()"`
	ProfileID                       string    `json:"profile_id"                          gorm:"column:profile_id;not null"`
	WeightCapabilityFit             int       `json:"weight_capability_fit"               gorm:"column:weight_capability_fit;not null;default:20"`
	WeightPortfolioMatch            int       `json:"weight_portfolio_match"              gorm:"column:weight_portfolio_match;not null;default:15"`
	WeightCommercialAttractiveness  int       `json:"weight_commercial_attractiveness"    gorm:"column:weight_commercial_attractiveness;not null;default:15"`
	WeightEligibilityFit            int       `json:"weight_eligibility_fit"              gorm:"column:weight_eligibility_fit;not null;default:15"`
	WeightDeadlineFeasibility       int       `json:"weight_deadline_feasibility"         gorm:"column:weight_deadline_feasibility;not null;default:10"`
	WeightStrategicAccountValue     int       `json:"weight_strategic_account_value"      gorm:"column:weight_strategic_account_value;not null;default:10"`
	WeightDeliveryRisk              int       `json:"weight_delivery_risk"                gorm:"column:weight_delivery_risk;not null;default:10"`
	WeightCompetitionWinProbability int       `json:"weight_competition_win_probability"  gorm:"column:weight_competition_win_probability;not null;default:5"`
	ThresholdPursue                 int       `json:"threshold_pursue"                    gorm:"column:threshold_pursue;not null;default:80"`
	ThresholdReview                 int       `json:"threshold_review"                    gorm:"column:threshold_review;not null;default:65"`
	ThresholdWatchlist              int       `json:"threshold_watchlist"                 gorm:"column:threshold_watchlist;not null;default:50"`
	CreatedAt                       time.Time `json:"created_at"`
	UpdatedAt                       time.Time `json:"updated_at"`
}

func (ScoringConfig) TableName() string { return "scoring_config" }

// ProfileAggregate is the full "Otak Agent" snapshot for one profile version:
// the root CompanyProfile plus its versioned children.
type ProfileAggregate struct {
	Profile       CompanyProfile
	Target        *TargetCriteria
	NoGo          *NoGoRule
	Keywords      []KeywordSet
	ScoringConfig *ScoringConfig
}

// ProfileRepository persists versioned CompanyProfile aggregates. GetCurrent
// returns the aggregate with IsCurrent=true; CreateVersion clones agg into a
// brand-new version (incrementing Version, flipping IsCurrent), leaving the
// prior version's rows intact for history.
type ProfileRepository interface {
	GetCurrent(ctx context.Context) (*ProfileAggregate, error)
	CreateVersion(ctx context.Context, agg *ProfileAggregate) (*ProfileAggregate, error)
}
