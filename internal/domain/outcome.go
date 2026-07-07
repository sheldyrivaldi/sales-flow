package domain

import (
	"context"
	"time"
)

type OutcomeResult string

const (
	OutcomeWon  OutcomeResult = "WON"
	OutcomeLost OutcomeResult = "LOST"
)

func (r OutcomeResult) Valid() bool {
	switch r {
	case OutcomeWon, OutcomeLost:
		return true
	}
	return false
}

type OutcomeTargetType string

const (
	OutcomeTargetTender  OutcomeTargetType = "tender"
	OutcomeTargetProspect OutcomeTargetType = "prospect"
)

func (t OutcomeTargetType) Valid() bool {
	switch t {
	case OutcomeTargetTender, OutcomeTargetProspect:
		return true
	}
	return false
}

type OutcomeEvent struct {
	ID         string            `json:"id"          gorm:"primaryKey;default:gen_random_uuid()"`
	TargetType OutcomeTargetType `json:"target_type" gorm:"column:target_type;not null"`
	TargetID   string            `json:"target_id"   gorm:"column:target_id;not null"`
	Result     OutcomeResult     `json:"result"      gorm:"not null"`
	Notes      *string           `json:"notes"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

func (OutcomeEvent) TableName() string { return "outcome_event" }

type OutcomeRepository interface {
	Create(ctx context.Context, e *OutcomeEvent) error
}
