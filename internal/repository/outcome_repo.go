package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

// compile-time check: *OutcomeRepo implements domain.OutcomeRepository.
var _ domain.OutcomeRepository = (*OutcomeRepo)(nil)

type OutcomeRepo struct {
	db *gorm.DB
}

func NewOutcomeRepo(db *gorm.DB) *OutcomeRepo {
	return &OutcomeRepo{db: db}
}

func (r *OutcomeRepo) Create(ctx context.Context, e *domain.OutcomeEvent) error {
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("outcome.Create: %w", err)
	}
	return nil
}
