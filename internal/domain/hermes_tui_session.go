package domain

import (
	"context"
	"time"
)

// HermesTuiSession is an audit record for an admin's access to the native
// Hermes CLI/TUI (see hermes-tui sidecar). Metadata only — who/when/duration
// /IP — never terminal input/output content, by design.
type HermesTuiSession struct {
	ID        string     `json:"id"         gorm:"primaryKey;default:gen_random_uuid()"`
	UserID    string     `json:"user_id"    gorm:"column:user_id;not null"`
	StartedAt time.Time  `json:"started_at" gorm:"column:started_at;not null"`
	EndedAt   *time.Time `json:"ended_at"   gorm:"column:ended_at"`
	RemoteIP  string     `json:"remote_ip"  gorm:"column:remote_ip"`
}

func (HermesTuiSession) TableName() string { return "hermes_tui_session" }

type HermesTuiSessionRepository interface {
	Create(ctx context.Context, s *HermesTuiSession) error
	End(ctx context.Context, id string, endedAt time.Time) error
}
