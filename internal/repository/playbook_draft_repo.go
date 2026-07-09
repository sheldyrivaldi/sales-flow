package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.PlaybookDraftRepository = (*PlaybookDraftRepo)(nil)

type PlaybookDraftRepo struct {
	db *gorm.DB
}

func NewPlaybookDraftRepo(db *gorm.DB) *PlaybookDraftRepo {
	return &PlaybookDraftRepo{db: db}
}

func (r *PlaybookDraftRepo) Create(ctx context.Context, d *domain.PlaybookDraft) error {
	if err := r.db.WithContext(ctx).Create(d).Error; err != nil {
		return fmt.Errorf("playbookDraft.Create: %w", err)
	}
	return nil
}

func (r *PlaybookDraftRepo) List(ctx context.Context, targetType, targetID string) ([]domain.PlaybookDraft, error) {
	var drafts []domain.PlaybookDraft
	err := r.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("created_at DESC").
		Find(&drafts).Error
	if err != nil {
		return nil, fmt.Errorf("playbookDraft.List: %w", err)
	}
	return drafts, nil
}
