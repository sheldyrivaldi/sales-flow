package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.SourceRepository = (*SourceRepo)(nil)

type SourceRepo struct {
	db *gorm.DB
}

func NewSourceRepo(db *gorm.DB) *SourceRepo {
	return &SourceRepo{db: db}
}

func (r *SourceRepo) Create(ctx context.Context, s *domain.Source) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("source.Create: %w", err)
	}
	return nil
}

func (r *SourceRepo) GetByID(ctx context.Context, id string) (*domain.Source, error) {
	var s domain.Source
	if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("source.GetByID: %w", err)
	}
	return &s, nil
}

func (r *SourceRepo) GetByPresetKey(ctx context.Context, key string) (*domain.Source, error) {
	var s domain.Source
	if err := r.db.WithContext(ctx).First(&s, "preset_key = ?", key).Error; err != nil {
		return nil, fmt.Errorf("source.GetByPresetKey: %w", err)
	}
	return &s, nil
}

// ListPresetKeys returns the preset_key of every source that originated from
// the preset catalog, in a single query (used to annotate the preset list
// without an N+1 lookup per catalog entry).
func (r *SourceRepo) ListPresetKeys(ctx context.Context) ([]string, error) {
	var keys []string
	if err := r.db.WithContext(ctx).Model(&domain.Source{}).
		Where("preset_key IS NOT NULL").
		Pluck("preset_key", &keys).Error; err != nil {
		return nil, fmt.Errorf("source.ListPresetKeys: %w", err)
	}
	return keys, nil
}

func (r *SourceRepo) List(ctx context.Context, f domain.SourceFilter, page, pageSize int) ([]domain.Source, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Source{})

	if f.Enabled != nil {
		q = q.Where("enabled = ?", *f.Enabled)
	}
	if f.Access != nil {
		q = q.Where("access = ?", *f.Access)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("name ILIKE ? OR url ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("source.List count: %w", err)
	}

	offset := (page - 1) * pageSize
	var sources []domain.Source
	if err := q.Order("priority DESC, created_at DESC").Limit(pageSize).Offset(offset).Find(&sources).Error; err != nil {
		return nil, 0, fmt.Errorf("source.List: %w", err)
	}
	return sources, total, nil
}

func (r *SourceRepo) Update(ctx context.Context, s *domain.Source) error {
	if err := r.db.WithContext(ctx).Save(s).Error; err != nil {
		return fmt.Errorf("source.Update: %w", err)
	}
	return nil
}

func (r *SourceRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Source{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("source.Delete: %w", err)
	}
	return nil
}
