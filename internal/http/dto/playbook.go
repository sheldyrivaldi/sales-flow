package dto

import (
	"encoding/json"
	"time"

	"salespilot/internal/domain"
)

// PlaybookResponse is the JSON shape for one immutable playbook version
// (generate, get-by-id, and each item in a list response).
type PlaybookResponse struct {
	ID         string          `json:"id"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Version    int             `json:"version"`
	Content    json.RawMessage `json:"content"`
	Model      *string         `json:"model"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ToPlaybookResponse maps domain.Playbook → PlaybookResponse.
func ToPlaybookResponse(p domain.Playbook) PlaybookResponse {
	return PlaybookResponse{
		ID:         p.ID,
		TargetType: p.TargetType,
		TargetID:   p.TargetID,
		Version:    p.Version,
		Content:    p.Content,
		Model:      p.Model,
		CreatedAt:  p.CreatedAt,
	}
}

// PlaybookListResponse is the full version history for one target — not
// paginated (a target realistically accumulates a handful of versions, not
// pages of them), unlike the P-4 list-response convention for entity lists.
type PlaybookListResponse struct {
	Items []PlaybookResponse `json:"items"`
}
