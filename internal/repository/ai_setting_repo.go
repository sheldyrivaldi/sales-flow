package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.AISettingRepository = (*AISettingRepo)(nil)

type AISettingRepo struct {
	db *gorm.DB
}

func NewAISettingRepo(db *gorm.DB) *AISettingRepo {
	return &AISettingRepo{db: db}
}

func (r *AISettingRepo) GetActive(ctx context.Context) (*domain.AISetting, error) {
	var s domain.AISetting
	err := r.db.WithContext(ctx).Where("is_active = ?", true).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ai_setting.GetActive: %w", err)
	}
	return &s, nil
}

// Upsert deactivates whatever is currently active (if anything) and creates
// s as the new active row, inside one transaction — never leaves two active
// rows visible even momentarily, and never violates the partial unique
// index (0019_ai_provider_setting.up.sql) mid-write.
func (r *AISettingRepo) Upsert(ctx context.Context, s *domain.AISetting) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.AISetting{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			return fmt.Errorf("ai_setting.Upsert: deactivate: %w", err)
		}
		s.IsActive = true
		if err := tx.Create(s).Error; err != nil {
			return fmt.Errorf("ai_setting.Upsert: create: %w", err)
		}
		return nil
	})
}
