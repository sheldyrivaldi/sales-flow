package domain

import (
	"context"
	"encoding/json"
	"time"
)

// PlaybookDraft is a lightweight, non-final draft saved by the MCP
// save_playbook_draft write tool (EP-09). It intentionally does not
// implement the full versioned Playbook entity (EP-14) — it exists so the
// tool has somewhere real to persist to, forward-compatible with EP-14
// picking it up as a starting draft.
type PlaybookDraft struct {
	ID         string          `json:"id"          gorm:"primaryKey;default:gen_random_uuid()"`
	TargetType string          `json:"target_type" gorm:"column:target_type;not null"`
	TargetID   string          `json:"target_id"   gorm:"column:target_id;not null"`
	Title      *string         `json:"title"`
	Content    json.RawMessage `json:"content"     gorm:"type:jsonb;not null"`
	Source     string          `json:"source"      gorm:"not null;default:'mcp'"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (PlaybookDraft) TableName() string { return "playbook_draft" }

// PlaybookTargetType enumerates valid PlaybookDraft.TargetType values.
type PlaybookTargetType string

const (
	PlaybookTargetTender   PlaybookTargetType = "tender"
	PlaybookTargetProspect PlaybookTargetType = "prospect"
)

func (t PlaybookTargetType) Valid() bool {
	switch t {
	case PlaybookTargetTender, PlaybookTargetProspect:
		return true
	}
	return false
}

type PlaybookDraftRepository interface {
	Create(ctx context.Context, d *PlaybookDraft) error
	List(ctx context.Context, targetType, targetID string) ([]PlaybookDraft, error)
}
