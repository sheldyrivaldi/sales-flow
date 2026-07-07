package dto

import (
	"time"

	"salespilot/internal/domain"
)

// --- Create ---

type ProspectCreateRequest struct {
	Name        string   `json:"name"          validate:"required"`
	Company     *string  `json:"company"       validate:"omitempty"`
	ContactInfo *string  `json:"contact_info"  validate:"omitempty"`
	SourceType  *string  `json:"source_type"   validate:"omitempty,oneof=manual event tender"`
	SourceID    *string  `json:"source_id"     validate:"omitempty"`
	Stage       *string  `json:"stage"         validate:"omitempty,oneof=NEW QUALIFIED ENGAGED PROPOSAL WON LOST"`
	EstValue    *float64 `json:"est_value"     validate:"omitempty,gte=0"`
	OwnerUserID *string  `json:"owner_user_id" validate:"omitempty"`
}

// --- Update ---

type ProspectUpdateRequest struct {
	Name        *string  `json:"name"          validate:"omitempty"`
	Company     *string  `json:"company"       validate:"omitempty"`
	ContactInfo *string  `json:"contact_info"  validate:"omitempty"`
	SourceType  *string  `json:"source_type"   validate:"omitempty,oneof=manual event tender"`
	SourceID    *string  `json:"source_id"     validate:"omitempty"`
	Stage       *string  `json:"stage"         validate:"omitempty,oneof=NEW QUALIFIED ENGAGED PROPOSAL WON LOST"`
	EstValue    *float64 `json:"est_value"     validate:"omitempty,gte=0"`
	OwnerUserID *string  `json:"owner_user_id" validate:"omitempty"`
}

// --- Stage transition ---

type ProspectStageRequest struct {
	Stage string  `json:"stage" validate:"required,oneof=NEW QUALIFIED ENGAGED PROPOSAL WON LOST"`
	Notes *string `json:"notes" validate:"omitempty"`
}

// ProspectResponse is the API response for a prospect (minimal, expanded by EP-07).
type ProspectResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Company     *string   `json:"company"`
	ContactInfo *string   `json:"contact_info"`
	SourceType  string    `json:"source_type"`
	SourceID    *string   `json:"source_id"`
	Stage       string    `json:"stage"`
	EstValue    *float64  `json:"est_value"`
	OwnerUserID *string   `json:"owner_user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

// ProspectListResponse is the paginated list response for prospects.
type ProspectListResponse struct {
	Items    []ProspectResponse `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}
