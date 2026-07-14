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

// Playbook is one immutable, versioned playbook (EP-14) generated for a
// tender or prospect. Generating a new playbook always creates
// version = latest+1 — existing rows are never updated or deleted, so the
// full history stays browsable/comparable (PRD AC: "versi baru, lama tetap").
type Playbook struct {
	ID         string          `json:"id"          gorm:"primaryKey;default:gen_random_uuid()"`
	TargetType string          `json:"target_type" gorm:"column:target_type;not null"`
	TargetID   string          `json:"target_id"   gorm:"column:target_id;not null"`
	Version    int             `json:"version"     gorm:"not null"`
	Content    json.RawMessage `json:"content"      gorm:"type:jsonb;not null"`
	Model      *string         `json:"model"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (Playbook) TableName() string { return "playbook" }

// PlaybookRepository persists immutable versioned playbooks — Create-only,
// no Update/Delete method exists because none should ever be needed.
type PlaybookRepository interface {
	Create(ctx context.Context, p *Playbook) error
	GetByID(ctx context.Context, id string) (*Playbook, error)
	ListByTarget(ctx context.Context, targetType, targetID string) ([]Playbook, error)
	// GetLatestVersion returns the highest existing version for a target, or
	// 0 if none exists yet (so the caller's next version is 0+1 = 1).
	GetLatestVersion(ctx context.Context, targetType, targetID string) (int, error)
}
