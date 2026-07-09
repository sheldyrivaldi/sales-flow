package domain

import (
	"context"
	"encoding/json"
	"time"
)

// AuditEvent records one write action for traceability (EP-09 MCP write
// tools; a minimal precursor to the full audit trail in EP-17). Modeled on
// OutcomeEvent: Create-only, no update/delete.
type AuditEvent struct {
	ID         string          `json:"id"          gorm:"primaryKey;default:gen_random_uuid()"`
	Actor      string          `json:"actor"       gorm:"not null"`
	Action     string          `json:"action"      gorm:"not null"`
	TargetType *string         `json:"target_type" gorm:"column:target_type"`
	TargetID   *string         `json:"target_id"   gorm:"column:target_id"`
	Payload    json.RawMessage `json:"payload"     gorm:"type:jsonb"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (AuditEvent) TableName() string { return "audit_log" }

type AuditRepository interface {
	Create(ctx context.Context, e *AuditEvent) error
}
