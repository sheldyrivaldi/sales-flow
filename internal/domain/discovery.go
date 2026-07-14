package domain

import (
	"context"
	"time"
)

type DiscoveryStatus string

const (
	DiscoveryStatusPending DiscoveryStatus = "pending"
	DiscoveryStatusRunning DiscoveryStatus = "running"
	DiscoveryStatusSuccess DiscoveryStatus = "success"
	DiscoveryStatusFailed  DiscoveryStatus = "failed"
)

func (s DiscoveryStatus) Valid() bool {
	switch s {
	case DiscoveryStatusPending, DiscoveryStatusRunning, DiscoveryStatusSuccess, DiscoveryStatusFailed:
		return true
	}
	return false
}

// DiscoveryRun records one discovery orchestrator execution (EP-12): which
// sources were targeted, its lifecycle status, how many tenders it found,
// and an idempotency key so a retried/scheduled trigger doesn't start a
// second concurrent run for the same logical request.
type DiscoveryRun struct {
	ID             string          `json:"id"               gorm:"primaryKey;default:gen_random_uuid()"`
	StartedAt      time.Time       `json:"started_at"       gorm:"column:started_at;not null"`
	FinishedAt     *time.Time      `json:"finished_at"      gorm:"column:finished_at"`
	SourceIDs      []string        `json:"source_ids"       gorm:"column:source_ids;serializer:json;type:jsonb"`
	Status         DiscoveryStatus `json:"status"           gorm:"not null;default:'pending'"`
	FoundCount     int             `json:"found_count"      gorm:"column:found_count;not null;default:0"`
	Summary        *string         `json:"summary"`
	CorrelationKey *string         `json:"correlation_key"  gorm:"column:correlation_key"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (DiscoveryRun) TableName() string { return "discovery_run" }

type DiscoveryRunRepository interface {
	Create(ctx context.Context, r *DiscoveryRun) error
	Update(ctx context.Context, r *DiscoveryRun) error
	GetByID(ctx context.Context, id string) (*DiscoveryRun, error)
	List(ctx context.Context, page, pageSize int) ([]DiscoveryRun, int64, error)
	// GetByCorrelationKey returns (nil, nil) when no run exists for key —
	// "no existing run" is a normal state, not an error (mirrors
	// ProspectScoreRepository.GetLatest).
	GetByCorrelationKey(ctx context.Context, key string) (*DiscoveryRun, error)
}
