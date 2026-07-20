package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.ProjectRepository = (*ProjectRepo)(nil)

type ProjectRepo struct {
	db *gorm.DB
}

func NewProjectRepo(db *gorm.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) Create(ctx context.Context, p *domain.Project) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("project.Create: %w", err)
	}
	return nil
}

func (r *ProjectRepo) Update(ctx context.Context, p *domain.Project) error {
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return fmt.Errorf("project.Update: %w", err)
	}
	return nil
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Project{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("project.Delete: %w", err)
	}
	return nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	var p domain.Project
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("project.GetByID: %w", err)
	}
	return &p, nil
}

func (r *ProjectRepo) List(ctx context.Context, f domain.ProjectFilter, page, pageSize int) ([]domain.Project, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Project{})

	if f.Status != nil {
		q = q.Where("status = ?", *f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("name ILIKE ? OR client_name ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("project.List: count: %w", err)
	}

	var items []domain.Project
	if err := q.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("project.List: %w", err)
	}
	return items, total, nil
}
