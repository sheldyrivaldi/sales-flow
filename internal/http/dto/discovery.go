package dto

import (
	"time"

	"salespilot/internal/domain"
)

// --- Request ---

// DiscoveryRunRequest is the optional body for POST /api/discovery/run.
// CorrelationKey lets a caller (e.g. the scheduler, TK-12.5.1) make a
// trigger idempotent; omitted/empty always starts a fresh run.
type DiscoveryRunRequest struct {
	CorrelationKey *string `json:"correlation_key" validate:"omitempty"`
}

// --- Response ---

type DiscoveryRunResponse struct {
	ID             string     `json:"id"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	SourceIDs      []string   `json:"source_ids"`
	Status         string     `json:"status"`
	FoundCount     int        `json:"found_count"`
	Summary        *string    `json:"summary"`
	CorrelationKey *string    `json:"correlation_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type DiscoveryRunListResponse struct {
	Items    []DiscoveryRunResponse `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// ToDiscoveryRunResponse maps domain.DiscoveryRun → DiscoveryRunResponse.
func ToDiscoveryRunResponse(r domain.DiscoveryRun) DiscoveryRunResponse {
	sourceIDs := r.SourceIDs
	if sourceIDs == nil {
		sourceIDs = []string{}
	}
	return DiscoveryRunResponse{
		ID:             r.ID,
		StartedAt:      r.StartedAt,
		FinishedAt:     r.FinishedAt,
		SourceIDs:      sourceIDs,
		Status:         string(r.Status),
		FoundCount:     r.FoundCount,
		Summary:        r.Summary,
		CorrelationKey: r.CorrelationKey,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}
