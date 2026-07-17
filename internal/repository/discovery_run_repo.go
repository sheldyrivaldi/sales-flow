package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.DiscoveryRunRepository = (*DiscoveryRunRepo)(nil)

type DiscoveryRunRepo struct {
	db *gorm.DB
}

func NewDiscoveryRunRepo(db *gorm.DB) *DiscoveryRunRepo {
	return &DiscoveryRunRepo{db: db}
}

func (r *DiscoveryRunRepo) Create(ctx context.Context, run *domain.DiscoveryRun) error {
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("discoveryRun.Create: %w", err)
	}
	return nil
}

func (r *DiscoveryRunRepo) Update(ctx context.Context, run *domain.DiscoveryRun) error {
	if err := r.db.WithContext(ctx).Save(run).Error; err != nil {
		return fmt.Errorf("discoveryRun.Update: %w", err)
	}
	return nil
}

func (r *DiscoveryRunRepo) GetByID(ctx context.Context, id string) (*domain.DiscoveryRun, error) {
	var run domain.DiscoveryRun
	if err := r.db.WithContext(ctx).First(&run, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("discoveryRun.GetByID: %w", err)
	}
	return &run, nil
}

func (r *DiscoveryRunRepo) List(ctx context.Context, page, pageSize int) ([]domain.DiscoveryRun, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.DiscoveryRun{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("discoveryRun.List count: %w", err)
	}

	offset := (page - 1) * pageSize
	var runs []domain.DiscoveryRun
	if err := q.Order("started_at DESC").Limit(pageSize).Offset(offset).Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("discoveryRun.List: %w", err)
	}
	return runs, total, nil
}

func (r *DiscoveryRunRepo) GetByCorrelationKey(ctx context.Context, key string) (*domain.DiscoveryRun, error) {
	var run domain.DiscoveryRun
	err := r.db.WithContext(ctx).First(&run, "correlation_key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("discoveryRun.GetByCorrelationKey: %w", err)
	}
	return &run, nil
}

// GetActive returns the newest pending/running run, or (nil, nil) when no
// run is in flight — server-side single-flight guard for discovery triggers.
func (r *DiscoveryRunRepo) GetActive(ctx context.Context) (*domain.DiscoveryRun, error) {
	var run domain.DiscoveryRun
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{string(domain.DiscoveryStatusPending), string(domain.DiscoveryStatusRunning)}).
		Order("started_at DESC").
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("discoveryRun.GetActive: %w", err)
	}
	return &run, nil
}
