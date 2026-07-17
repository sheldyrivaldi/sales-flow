package dto

import (
	"time"

	"salespilot/internal/domain"
)

// --- Create ---

type SourceCreateRequest struct {
	Name      string   `json:"name"       validate:"required"`
	URL       string   `json:"url"        validate:"required,url"`
	Country   *string  `json:"country"    validate:"omitempty"`
	Access    *string  `json:"access"     validate:"omitempty,oneof=publik login manual"`
	LegalNote *string  `json:"legal_note" validate:"omitempty"`
	Enabled   *bool    `json:"enabled"    validate:"omitempty"`
	Priority  *int     `json:"priority"   validate:"omitempty"`
	Frequency *string  `json:"frequency"  validate:"omitempty,oneof=harian 2-3x mingguan manual"`
	DataTypes []string `json:"data_types" validate:"omitempty"`
}

// --- Update ---

type SourceUpdateRequest struct {
	Name      *string  `json:"name"       validate:"omitempty"`
	URL       *string  `json:"url"        validate:"omitempty,url"`
	Country   *string  `json:"country"    validate:"omitempty"`
	Access    *string  `json:"access"     validate:"omitempty,oneof=publik login manual"`
	LegalNote *string  `json:"legal_note" validate:"omitempty"`
	Enabled   *bool    `json:"enabled"    validate:"omitempty"`
	Priority  *int     `json:"priority"   validate:"omitempty"`
	Frequency *string  `json:"frequency"  validate:"omitempty,oneof=harian 2-3x mingguan manual"`
	DataTypes []string `json:"data_types" validate:"omitempty"`
}

// --- Activate preset ---

type SourcePresetActivateRequest struct {
	Key string `json:"key" validate:"required"`
}

// --- Response ---

type SourceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Country   *string   `json:"country"`
	Access    string    `json:"access"`
	LegalNote *string   `json:"legal_note"`
	Enabled   bool      `json:"enabled"`
	Priority  int       `json:"priority"`
	PresetKey *string   `json:"preset_key"`
	Frequency string    `json:"frequency"`
	DataTypes []string  `json:"data_types"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToSourceResponse(s domain.Source) SourceResponse {
	dataTypes := s.DataTypes
	if dataTypes == nil {
		dataTypes = []string{}
	}
	return SourceResponse{
		ID:        s.ID,
		Name:      s.Name,
		URL:       s.URL,
		Country:   s.Country,
		Access:    string(s.Access),
		LegalNote: s.LegalNote,
		Enabled:   s.Enabled,
		Priority:  s.Priority,
		PresetKey: s.PresetKey,
		Frequency: s.Frequency,
		DataTypes: dataTypes,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

type SourceListResponse struct {
	Items    []SourceResponse `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// SourcePresetResponse describes one catalog entry plus whether it has
// already been activated (a `source` row with this preset_key exists).
type SourcePresetResponse struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Country   string `json:"country"`
	Access    string `json:"access"`
	LegalNote string `json:"legal_note"`
	Activated bool   `json:"activated"`
}
