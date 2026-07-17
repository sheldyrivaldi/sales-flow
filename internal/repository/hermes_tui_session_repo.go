package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.HermesTuiSessionRepository = (*HermesTuiSessionRepo)(nil)

type HermesTuiSessionRepo struct {
	db *gorm.DB
}

func NewHermesTuiSessionRepo(db *gorm.DB) *HermesTuiSessionRepo {
	return &HermesTuiSessionRepo{db: db}
}

func (r *HermesTuiSessionRepo) Create(ctx context.Context, s *domain.HermesTuiSession) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("hermesTuiSession.Create: %w", err)
	}
	return nil
}

func (r *HermesTuiSessionRepo) End(ctx context.Context, id string, endedAt time.Time) error {
	err := r.db.WithContext(ctx).Model(&domain.HermesTuiSession{}).
		Where("id = ?", id).
		Update("ended_at", endedAt).Error
	if err != nil {
		return fmt.Errorf("hermesTuiSession.End: %w", err)
	}
	return nil
}
