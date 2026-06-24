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
