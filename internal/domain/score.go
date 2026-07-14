package domain

import (
	"context"
	"encoding/json"
	"time"
)

// ProspectScore records one AI scoring run (EP-10) against a tender or
// prospect. Rows are append-only history: "Analisa ulang" writes a new row
// rather than overwriting the previous one, so ProspectScoreRepository.
// GetLatest (ORDER BY created_at DESC LIMIT 1) is the "current" score, while
// the full table doubles as the scoring audit trail (model + evidence +
// timestamp per run).
type ProspectScore struct {
	ID                string          `json:"id"                 gorm:"primaryKey;default:gen_random_uuid()"`
	TargetType        string          `json:"target_type"        gorm:"column:target_type;not null"`
	TargetID          string          `json:"target_id"          gorm:"column:target_id;not null"`
	FitScore          int             `json:"fit_score"          gorm:"column:fit_score;not null"`
	RecommendedAction string          `json:"recommended_action" gorm:"column:recommended_action;not null"`
	Confidence        *float64        `json:"confidence"`
	Reasoning         *string         `json:"reasoning"`
	Evidence          json.RawMessage `json:"evidence"           gorm:"type:jsonb"`
	RiskFlags         json.RawMessage `json:"risk_flags"         gorm:"column:risk_flags;type:jsonb"`
	Model             *string         `json:"model"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (ProspectScore) TableName() string { return "prospect_score" }

// ProspectScoreRepository persists append-only scoring runs. GetLatest
// returns (nil, nil) when no score exists yet for the target — callers treat
// "no score" as a normal, non-error state (a target may simply never have
// been analyzed).
type ProspectScoreRepository interface {
	Create(ctx context.Context, s *ProspectScore) error
	GetLatest(ctx context.Context, targetType, targetID string) (*ProspectScore, error)
}
