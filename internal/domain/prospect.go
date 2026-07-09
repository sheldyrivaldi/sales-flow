package domain

import (
	"context"
	"time"
)

type ProspectStage string

const (
	ProspectStageNew       ProspectStage = "NEW"
	ProspectStageQualified ProspectStage = "QUALIFIED"
	ProspectStageEngaged   ProspectStage = "ENGAGED"
	ProspectStageProposal  ProspectStage = "PROPOSAL"
	ProspectStageWon       ProspectStage = "WON"
	ProspectStageLost      ProspectStage = "LOST"
)

func (s ProspectStage) Valid() bool {
	switch s {
	case ProspectStageNew, ProspectStageQualified, ProspectStageEngaged,
		ProspectStageProposal, ProspectStageWon, ProspectStageLost:
		return true
	}
	return false
}

type ProspectSource string

const (
	ProspectSourceManual ProspectSource = "manual"
	ProspectSourceEvent  ProspectSource = "event"
	ProspectSourceTender ProspectSource = "tender"
)

func (s ProspectSource) Valid() bool {
	switch s {
	case ProspectSourceManual, ProspectSourceEvent, ProspectSourceTender:
		return true
	}
	return false
}

type Prospect struct {
	ID          string         `json:"id"            gorm:"primaryKey;default:gen_random_uuid()"`
	Name        string         `json:"name"          gorm:"not null"`
	Company     *string        `json:"company"`
	ContactInfo *string        `json:"contact_info"  gorm:"column:contact_info"`
	SourceType  ProspectSource `json:"source_type"   gorm:"column:source_type;not null;default:'manual'"`
	SourceID    *string        `json:"source_id"     gorm:"column:source_id"`
	Stage       ProspectStage  `json:"stage"         gorm:"not null;default:'NEW'"`
	EstValue    *float64       `json:"est_value"     gorm:"column:est_value"`
	OwnerUserID *string        `json:"owner_user_id" gorm:"column:owner_user_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (Prospect) TableName() string { return "prospect" }

// ProspectFilter narrows a List query. Zero-value fields are ignored.
type ProspectFilter struct {
	Stage       *ProspectStage
	OwnerUserID *string
	SourceType  *ProspectSource
	Search      string
}

type ProspectRepository interface {
	Create(ctx context.Context, p *Prospect) error
	GetByID(ctx context.Context, id string) (*Prospect, error)
	GetBySource(ctx context.Context, srcType ProspectSource, srcID string) (*Prospect, error)
	List(ctx context.Context, f ProspectFilter, page, pageSize int) ([]Prospect, int64, error)
	Update(ctx context.Context, p *Prospect) error
	Delete(ctx context.Context, id string) error
	SummaryByStage(ctx context.Context) ([]ProspectStageSummary, error)
}

// ProspectStageSummary aggregates prospect count and total est_value for one
// stage. Used by the MCP get_pipeline_summary/get_revenue_summary tools
// (EP-09) — there is no dashboard aggregation elsewhere yet (EP-11).
type ProspectStageSummary struct {
	Stage      ProspectStage `json:"stage"`
	Count      int64         `json:"count"`
	TotalValue float64       `json:"total_value"`
}
