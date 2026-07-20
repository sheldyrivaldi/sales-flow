package dto

import (
	"encoding/json"
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
	// Peserta diundang lewat email lepas — tidak perlu punya akun.
	ParticipantEmails []string             `json:"participant_emails" validate:"omitempty,dive,email"`
	Attachments       []EventAttachmentDTO `json:"attachments"        validate:"omitempty,dive"`
}

// EventAttachmentDTO adalah metadata berkas hasil unggah (POST
// /api/events/attachments), dikirim balik saat menyimpan event.
type EventAttachmentDTO struct {
	Name string `json:"name" validate:"required"`
	URL  string `json:"url"  validate:"required"`
	Mime string `json:"mime" validate:"omitempty"`
	Size int64  `json:"size" validate:"omitempty"`
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
	// Pointer ke slice: nil = tidak diubah, slice kosong = dikosongkan.
	ParticipantEmails *[]string             `json:"participant_emails" validate:"omitempty,dive,email"`
	Attachments       *[]EventAttachmentDTO `json:"attachments"        validate:"omitempty,dive"`
}

// --- Response ---

type EventResponse struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	Type              string                   `json:"type"`
	Date              *time.Time               `json:"date"`
	Location          *string                  `json:"location"`
	Organizer         *string                  `json:"organizer"`
	Notes             *string                  `json:"notes"`
	Status            string                   `json:"status"`
	ParticipantEmails []string                 `json:"participant_emails"`
	Attachments       []domain.EventAttachment `json:"attachments"`
	Analysis          json.RawMessage          `json:"analysis,omitempty"`
	AnalyzedAt        *time.Time               `json:"analyzed_at,omitempty"`
	AnalysisStatus    string                   `json:"analysis_status"`
	AnalysisError     *string                  `json:"analysis_error,omitempty"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
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
		ID:                e.ID,
		Name:              e.Name,
		Type:              string(e.Type),
		Date:              e.Date,
		Location:          e.Location,
		Organizer:         e.Organizer,
		Notes:             e.Notes,
		Status:            string(e.Status),
		ParticipantEmails: nonNilStrings(e.ParticipantEmails),
		Attachments:       nonNilAttachments(e.Attachments),
		Analysis:          e.Analysis,
		AnalyzedAt:        e.AnalyzedAt,
		AnalysisStatus:    e.AnalysisStatus,
		AnalysisError:     e.AnalysisError,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

// nonNilStrings memastikan JSON mengirim [] bukan null, supaya frontend tidak
// perlu menjaga dua bentuk kosong yang berbeda.
func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nonNilAttachments(v []domain.EventAttachment) []domain.EventAttachment {
	if v == nil {
		return []domain.EventAttachment{}
	}
	return v
}
