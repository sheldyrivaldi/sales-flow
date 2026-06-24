package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

// compile-time check: *TenderRepo implements domain.TenderRepository.
var _ domain.TenderRepository = (*TenderRepo)(nil)

type TenderRepo struct {
	db *gorm.DB
}

func NewTenderRepo(db *gorm.DB) *TenderRepo {
	return &TenderRepo{db: db}
}

func (r *TenderRepo) Create(ctx context.Context, t *domain.Tender) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return fmt.Errorf("tender.Create: %w", err)
	}
	return nil
}

func (r *TenderRepo) GetByID(ctx context.Context, id string) (*domain.Tender, error) {
	var t domain.Tender
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("tender.GetByID: %w", err)
	}
	return &t, nil
}

func (r *TenderRepo) List(ctx context.Context, f domain.TenderFilter, page, pageSize int) ([]domain.Tender, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Tender{})

	if f.Status != nil {
		q = q.Where("status = ?", *f.Status)
	}
	if f.RecommendedAction != nil {
		q = q.Where("recommended_action = ?", *f.RecommendedAction)
	}
	if f.Origin != nil {
		q = q.Where("origin = ?", *f.Origin)
	}
	if f.DeadlineFrom != nil {
		q = q.Where("submission_deadline >= ?", *f.DeadlineFrom)
	}
	if f.DeadlineTo != nil {
		q = q.Where("submission_deadline <= ?", *f.DeadlineTo)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("title ILIKE ? OR buyer_name ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("tender.List count: %w", err)
	}

	offset := (page - 1) * pageSize
	var tenders []domain.Tender
	if err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&tenders).Error; err != nil {
		return nil, 0, fmt.Errorf("tender.List: %w", err)
	}
	return tenders, total, nil
}

func (r *TenderRepo) Update(ctx context.Context, t *domain.Tender) error {
	if err := r.db.WithContext(ctx).Save(t).Error; err != nil {
		return fmt.Errorf("tender.Update: %w", err)
	}
	return nil
}

func (r *TenderRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Tender{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("tender.Delete: %w", err)
	}
	return nil
}
