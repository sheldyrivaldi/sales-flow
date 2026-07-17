package domain

import (
	"context"
	"time"
)

// SourceAccess describes how a crawling source can be reached. Sources with
// access other than "publik" must not be crawled automatically (PRD §9
// compliance guard) — they are only tracked/marked in the UI.
type SourceAccess string

const (
	SourceAccessPublik SourceAccess = "publik"
	SourceAccessLogin  SourceAccess = "login"
	SourceAccessManual SourceAccess = "manual"
)

func (a SourceAccess) Valid() bool {
	switch a {
	case SourceAccessPublik, SourceAccessLogin, SourceAccessManual:
		return true
	}
	return false
}

// Source is a crawling source (procurement portal, API, etc.), global to the
// workspace (not versioned with CompanyProfile). PresetKey is set when the
// row originated from the hardcoded Indonesia preset catalog, and is used to
// make preset activation idempotent.
type Source struct {
	ID        string       `json:"id"         gorm:"primaryKey;default:gen_random_uuid()"`
	Name      string       `json:"name"       gorm:"not null"`
	URL       string       `json:"url"        gorm:"not null"`
	Country   *string      `json:"country"`
	Access    SourceAccess `json:"access" gorm:"not null;default:'publik'"`
	LegalNote *string      `json:"legal_note" gorm:"column:legal_note"`
	Enabled   bool         `json:"enabled"    gorm:"not null;default:false"`
	Priority  int          `json:"priority"   gorm:"not null;default:0"`
	PresetKey *string      `json:"preset_key" gorm:"column:preset_key"`
	// Frequency is this source's own monitoring cadence (RFI §6.1), distinct
	// from the profile-wide crawl_frequency: a high-priority government
	// portal might be checked "harian" while an aggregator is only "2-3x" or
	// "manual" (CAPTCHA/paywall — never auto-crawled regardless of this
	// value; see filterCrawlableSources's Access-based compliance guard).
	Frequency string    `json:"frequency"  gorm:"not null;default:'harian'"`
	DataTypes []string  `json:"data_types" gorm:"column:data_types;serializer:json;type:jsonb"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Source) TableName() string { return "source" }

// SourceFilter narrows a List query. Zero-value fields are ignored.
type SourceFilter struct {
	Enabled *bool
	Access  *SourceAccess
	Search  string
}

type SourceRepository interface {
	Create(ctx context.Context, s *Source) error
	GetByID(ctx context.Context, id string) (*Source, error)
	GetByPresetKey(ctx context.Context, key string) (*Source, error)
	ListPresetKeys(ctx context.Context) ([]string, error)
	List(ctx context.Context, f SourceFilter, page, pageSize int) ([]Source, int64, error)
	Update(ctx context.Context, s *Source) error
	Delete(ctx context.Context, id string) error
}
