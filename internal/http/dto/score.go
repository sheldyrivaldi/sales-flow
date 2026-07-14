package dto

import (
	"encoding/json"
	"time"

	"salespilot/internal/domain"
)

// ScoreResponse is the JSON shape for both the POST (run analysis) and GET
// (read latest) score endpoints — one prospect_score row.
type ScoreResponse struct {
	ID                string          `json:"id"`
	TargetType        string          `json:"target_type"`
	TargetID          string          `json:"target_id"`
	FitScore          int             `json:"fit_score"`
	RecommendedAction string          `json:"recommended_action"`
	Confidence        *float64        `json:"confidence"`
	Reasoning         *string         `json:"reasoning"`
	Evidence          json.RawMessage `json:"evidence"`
	RiskFlags         json.RawMessage `json:"risk_flags"`
	Model             *string         `json:"model"`
	CreatedAt         time.Time       `json:"created_at"`
}

// ToScoreResponse maps domain.ProspectScore → ScoreResponse.
func ToScoreResponse(s domain.ProspectScore) ScoreResponse {
	return ScoreResponse{
		ID:                s.ID,
		TargetType:        s.TargetType,
		TargetID:          s.TargetID,
		FitScore:          s.FitScore,
		RecommendedAction: s.RecommendedAction,
		Confidence:        s.Confidence,
		Reasoning:         s.Reasoning,
		Evidence:          s.Evidence,
		RiskFlags:         s.RiskFlags,
		Model:             s.Model,
		CreatedAt:         s.CreatedAt,
	}
}
