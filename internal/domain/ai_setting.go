package domain

import (
	"context"
	"encoding/json"
	"time"
)

type AIProvider string

const (
	AIProviderOpenAI     AIProvider = "openai"
	AIProviderOpenRouter AIProvider = "openrouter"
)

func (p AIProvider) Valid() bool {
	switch p {
	case AIProviderOpenAI, AIProviderOpenRouter:
		return true
	}
	return false
}

// AISetting is one AI provider configuration (EP-18 ST-18.4) — provider,
// model, optional base_url, and an API key that is ALWAYS stored encrypted
// (AES-256-GCM, internal/auth/crypto.go) and never round-tripped to the
// client as plaintext (see dto.AISettingResponse's masked field). At most
// one row may have is_active=true, enforced by a partial unique index
// (0019_ai_provider_setting.up.sql) rather than application logic.
type AISetting struct {
	ID              string          `json:"id"                gorm:"primaryKey;default:gen_random_uuid()"`
	Provider        AIProvider      `json:"provider"           gorm:"not null"`
	Model           string          `json:"model"              gorm:"not null"`
	BaseURL         *string         `json:"base_url"`
	APIKeyEncrypted string          `json:"-"                  gorm:"column:api_key_encrypted;not null"`
	EnabledToolsets json.RawMessage `json:"enabled_toolsets"   gorm:"type:jsonb"`
	IsActive        bool            `json:"is_active"          gorm:"not null;default:false"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (AISetting) TableName() string { return "ai_provider_setting" }

type AISettingRepository interface {
	// GetActive returns the current active config, or (nil, nil) if none
	// has ever been set — a normal state, not an error.
	GetActive(ctx context.Context) (*AISetting, error)
	// Upsert persists s as the new active config, deactivating any
	// previously active row in the same transaction (single-active
	// invariant is enforced by the DB, but Upsert must not violate it
	// mid-write).
	Upsert(ctx context.Context, s *AISetting) error
}
