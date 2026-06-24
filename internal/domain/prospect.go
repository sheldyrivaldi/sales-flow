package domain

import (
	"context"
	"time"
)

type ProspectStage string

const (
	ProspectStageNew      ProspectStage = "NEW"
	ProspectStageQualified ProspectStage = "QUALIFIED"
	ProspectStageEngaged  ProspectStage = "ENGAGED"
	ProspectStageProposal ProspectStage = "PROPOSAL"
	ProspectStageWon      ProspectStage = "WON"
	ProspectStageLost     ProspectStage = "LOST"
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
	ID          string         `json:"id"            gorm:"primaryKey"`
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

type ProspectRepository interface {
	Create(ctx context.Context, p *Prospect) error
	GetByID(ctx context.Context, id string) (*Prospect, error)
	GetBySource(ctx context.Context, srcType ProspectSource, srcID string) (*Prospect, error)
}
