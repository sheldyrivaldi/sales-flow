package domain

import (
	"context"
	"encoding/json"
	"time"
)

// TelemetryEvent records one metric-worthy action (EP-17 ST-17.1) — e.g.
// chat_opened, review_pursue, scoring_generated, report_generated,
// outcome_recorded. Modeled on AuditEvent: append-only, no update/delete.
type TelemetryEvent struct {
	ID        string          `json:"id"         gorm:"primaryKey;default:gen_random_uuid()"`
	Event     string          `json:"event"      gorm:"not null"`
	Props     json.RawMessage `json:"props"      gorm:"type:jsonb"`
	Actor     *string         `json:"actor"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (TelemetryEvent) TableName() string { return "telemetry_event" }

type TelemetryRepository interface {
	Create(ctx context.Context, e *TelemetryEvent) error
	CountByEvent(ctx context.Context, event string, since time.Time) (int64, error)
}
