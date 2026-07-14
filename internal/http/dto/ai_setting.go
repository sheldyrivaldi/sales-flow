package dto

import (
	"encoding/json"
	"time"

	"salespilot/internal/domain"
)

// AISettingUpdateRequest is PUT /api/settings/ai (EP-18 ST-18.4). APIKey is
// write-only and optional: nil/empty means "keep the currently stored key
// unchanged" (service.AISettingService.Update honors this) — the key is
// never round-tripped to the client, so the UI has no other way to "not
// change" it.
type AISettingUpdateRequest struct {
	Provider        string          `json:"provider" validate:"required,oneof=openai openrouter"`
	Model           string          `json:"model"    validate:"required"`
	BaseURL         *string         `json:"base_url"`
	APIKey          *string         `json:"api_key"`
	EnabledToolsets json.RawMessage `json:"enabled_toolsets"`
}

// AISettingResponse never carries a plaintext key — only a masked hint
// (e.g. "sk-...abcd") built by AISettingService from the decrypted value.
type AISettingResponse struct {
	Provider        string          `json:"provider"`
	Model           string          `json:"model"`
	BaseURL         *string         `json:"base_url"`
	APIKeyMasked    string          `json:"api_key_masked"`
	EnabledToolsets json.RawMessage `json:"enabled_toolsets"`
	IsActive        bool            `json:"is_active"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ToAISettingResponse maps domain.AISetting → AISettingResponse. maskedKey
// must already be computed by the caller (service layer — the only place
// that holds the decryption key).
func ToAISettingResponse(s domain.AISetting, maskedKey string) AISettingResponse {
	return AISettingResponse{
		Provider:        string(s.Provider),
		Model:           s.Model,
		BaseURL:         s.BaseURL,
		APIKeyMasked:    maskedKey,
		EnabledToolsets: s.EnabledToolsets,
		IsActive:        s.IsActive,
		UpdatedAt:       s.UpdatedAt,
	}
}
