package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.ProspectScoreRepository = (*ProspectScoreRepo)(nil)

type ProspectScoreRepo struct {
	db *gorm.DB
}

func NewProspectScoreRepo(db *gorm.DB) *ProspectScoreRepo {
	return &ProspectScoreRepo{db: db}
}

func (r *ProspectScoreRepo) Create(ctx context.Context, s *domain.ProspectScore) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("prospectScore.Create: %w", err)
	}
	return nil
}

func (r *ProspectScoreRepo) GetLatest(ctx context.Context, targetType, targetID string) (*domain.ProspectScore, error) {
	var s domain.ProspectScore
	err := r.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Order("created_at DESC").
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("prospectScore.GetLatest: %w", err)
	}
	return &s, nil
}
