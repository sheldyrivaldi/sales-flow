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
	ID                string    `json:"id"                  gorm:"primaryKey;default:gen_random_uuid()"`
	CompanyName       string    `json:"company_name"        gorm:"column:company_name;not null"`
	OneLiner          *string   `json:"one_liner"            gorm:"column:one_liner"`
	ServiceCategories []string  `json:"service_categories"   gorm:"column:service_categories;serializer:json;type:jsonb"`
	TechStack         []string  `json:"tech_stack"           gorm:"column:tech_stack;serializer:json;type:jsonb"`
	SourceDocRefs     []string  `json:"source_doc_refs"      gorm:"column:source_doc_refs;serializer:json;type:jsonb"`
	Version           int       `json:"version"              gorm:"not null;default:1"`
	IsCurrent         bool      `json:"is_current"           gorm:"column:is_current;not null;default:true"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (CompanyProfile) TableName() string { return "company_profile" }

// TargetCriteria holds buyer/opportunity scoring criteria for one profile version.
type TargetCriteria struct {
	ID               string    `json:"id"                 gorm:"primaryKey;default:gen_random_uuid()"`
	ProfileID        string    `json:"profile_id"         gorm:"column:profile_id;not null"`
	Countries        []string  `json:"countries"          gorm:"serializer:json;type:jsonb"`
	Industries       []string  `json:"industries"         gorm:"serializer:json;type:jsonb"`
	ValueMin         *float64  `json:"value_min"          gorm:"column:value_min"`
	ValueIdeal       *float64  `json:"value_ideal"        gorm:"column:value_ideal"`
	ValueMax         *float64  `json:"value_max"          gorm:"column:value_max"`
	Currency         string    `json:"currency"           gorm:"not null;default:'IDR'"`
	DeadlineMinDays  *int      `json:"deadline_min_days"  gorm:"column:deadline_min_days"`
	ProcurementTypes []string  `json:"procurement_types"  gorm:"column:procurement_types;serializer:json;type:jsonb"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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

// ProfileAggregate is the full "Otak Agent" snapshot for one profile version:
// the root CompanyProfile plus its versioned children.
type ProfileAggregate struct {
	Profile  CompanyProfile
	Target   *TargetCriteria
	NoGo     *NoGoRule
	Keywords []KeywordSet
}

// ProfileRepository persists versioned CompanyProfile aggregates. GetCurrent
// returns the aggregate with IsCurrent=true; CreateVersion clones agg into a
// brand-new version (incrementing Version, flipping IsCurrent), leaving the
// prior version's rows intact for history.
type ProfileRepository interface {
	GetCurrent(ctx context.Context) (*ProfileAggregate, error)
	CreateVersion(ctx context.Context, agg *ProfileAggregate) (*ProfileAggregate, error)
}
