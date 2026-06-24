package dto

import (
	"time"

	"salespilot/internal/domain"
)

// ProspectResponse is the API response for a prospect (minimal, expanded by EP-07).
type ProspectResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Company     *string    `json:"company"`
	ContactInfo *string    `json:"contact_info"`
	SourceType  string     `json:"source_type"`
	SourceID    *string    `json:"source_id"`
	Stage       string     `json:"stage"`
	EstValue    *float64   `json:"est_value"`
	OwnerUserID *string    `json:"owner_user_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToProspectResponse maps domain.Prospect → ProspectResponse.
func ToProspectResponse(p domain.Prospect) ProspectResponse {
	return ProspectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Company:     p.Company,
		ContactInfo: p.ContactInfo,
		SourceType:  string(p.SourceType),
		SourceID:    p.SourceID,
		Stage:       string(p.Stage),
		EstValue:    p.EstValue,
		OwnerUserID: p.OwnerUserID,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
