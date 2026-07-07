package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.ProspectRepository = (*ProspectRepo)(nil)

type ProspectRepo struct {
	db *gorm.DB
}

func NewProspectRepo(db *gorm.DB) *ProspectRepo {
	return &ProspectRepo{db: db}
}

func (r *ProspectRepo) Create(ctx context.Context, p *domain.Prospect) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("prospect.Create: %w", err)
	}
	return nil
}

func (r *ProspectRepo) GetByID(ctx context.Context, id string) (*domain.Prospect, error) {
	var p domain.Prospect
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("prospect.GetByID: %w", err)
	}
	return &p, nil
}

func (r *ProspectRepo) GetBySource(ctx context.Context, srcType domain.ProspectSource, srcID string) (*domain.Prospect, error) {
	var p domain.Prospect
	if err := r.db.WithContext(ctx).First(&p, "source_type = ? AND source_id = ?", srcType, srcID).Error; err != nil {
		return nil, fmt.Errorf("prospect.GetBySource: %w", err)
	}
	return &p, nil
}

func (r *ProspectRepo) List(ctx context.Context, f domain.ProspectFilter, page, pageSize int) ([]domain.Prospect, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Prospect{})

	if f.Stage != nil {
		q = q.Where("stage = ?", *f.Stage)
	}
	if f.OwnerUserID != nil {
		q = q.Where("owner_user_id = ?", *f.OwnerUserID)
	}
	if f.SourceType != nil {
		q = q.Where("source_type = ?", *f.SourceType)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("name ILIKE ? OR company ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("prospect.List count: %w", err)
	}

	offset := (page - 1) * pageSize
	var prospects []domain.Prospect
	if err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&prospects).Error; err != nil {
		return nil, 0, fmt.Errorf("prospect.List: %w", err)
	}
	return prospects, total, nil
}

func (r *ProspectRepo) Update(ctx context.Context, p *domain.Prospect) error {
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return fmt.Errorf("prospect.Update: %w", err)
	}
	return nil
}

func (r *ProspectRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Prospect{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("prospect.Delete: %w", err)
	}
	return nil
}
