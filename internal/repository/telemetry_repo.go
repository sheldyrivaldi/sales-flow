package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.TelemetryRepository = (*TelemetryRepo)(nil)

type TelemetryRepo struct {
	db *gorm.DB
}

func NewTelemetryRepo(db *gorm.DB) *TelemetryRepo {
	return &TelemetryRepo{db: db}
}

func (r *TelemetryRepo) Create(ctx context.Context, e *domain.TelemetryEvent) error {
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("telemetry.Create: %w", err)
	}
	return nil
}

func (r *TelemetryRepo) CountByEvent(ctx context.Context, event string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.TelemetryEvent{}).
		Where("event = ? AND created_at >= ?", event, since).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("telemetry.CountByEvent: %w", err)
	}
	return count, nil
}
