package dto

import (
	"time"

	"salespilot/internal/domain"
)

// --- Create ---

type EventCreateRequest struct {
	Name      string  `json:"name"      validate:"required"`
	Type      string  `json:"type"      validate:"required,oneof=EXPO CONFERENCE SEMINAR WORKSHOP NETWORKING OTHER"`
	Date      *string `json:"date"      validate:"omitempty"`
	Location  *string `json:"location"  validate:"omitempty"`
	Organizer *string `json:"organizer" validate:"omitempty"`
	Notes     *string `json:"notes"     validate:"omitempty"`
	Status    *string `json:"status"    validate:"omitempty,oneof=PLANNED ATTENDED CANCELLED"`
}

// --- Update ---

type EventUpdateRequest struct {
	Name      *string `json:"name"      validate:"omitempty"`
	Type      *string `json:"type"      validate:"omitempty,oneof=EXPO CONFERENCE SEMINAR WORKSHOP NETWORKING OTHER"`
	Date      *string `json:"date"      validate:"omitempty"`
	Location  *string `json:"location"  validate:"omitempty"`
	Organizer *string `json:"organizer" validate:"omitempty"`
	Notes     *string `json:"notes"     validate:"omitempty"`
	Status    *string `json:"status"    validate:"omitempty,oneof=PLANNED ATTENDED CANCELLED"`
}

// --- Response ---

type EventResponse struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Date      *time.Time  `json:"date"`
	Location  *string     `json:"location"`
	Organizer *string     `json:"organizer"`
	Notes     *string     `json:"notes"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// --- List response ---

type EventListResponse struct {
	Items    []EventResponse `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// ToEventResponse maps domain.Event → EventResponse.
func ToEventResponse(e domain.Event) EventResponse {
	return EventResponse{
		ID:        e.ID,
		Name:      e.Name,
		Type:      string(e.Type),
		Date:      e.Date,
		Location:  e.Location,
		Organizer: e.Organizer,
		Notes:     e.Notes,
		Status:    string(e.Status),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
