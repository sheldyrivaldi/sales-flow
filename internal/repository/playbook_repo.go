package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.PlaybookRepository = (*PlaybookRepo)(nil)

type PlaybookRepo struct {
	db *gorm.DB
}

func NewPlaybookRepo(db *gorm.DB) *PlaybookRepo {
	return &PlaybookRepo{db: db}
}

func (r *PlaybookRepo) Create(ctx context.Context, p *domain.Playbook) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("playbook.Create: %w", err)
	}
	return nil
}

func (r *PlaybookRepo) GetByID(ctx context.Context, id string) (*domain.Playbook, error) {
	var p domain.Playbook
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("playbook.GetByID: %w", err)
	}
	return &p, nil
}

func (r *PlaybookRepo) ListByTarget(ctx context.Context, targetType, targetID string) ([]domain.Playbook, error) {
	var items []domain.Playbook
	err := r.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("version DESC").
		Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("playbook.ListByTarget: %w", err)
	}
	return items, nil
}

func (r *PlaybookRepo) GetLatestVersion(ctx context.Context, targetType, targetID string) (int, error) {
	var maxVersion *int
	err := r.db.WithContext(ctx).
		Model(&domain.Playbook{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Select("MAX(version)").
		Scan(&maxVersion).Error
	if err != nil {
		return 0, fmt.Errorf("playbook.GetLatestVersion: %w", err)
	}
	if maxVersion == nil {
		return 0, nil
	}
	return *maxVersion, nil
}
