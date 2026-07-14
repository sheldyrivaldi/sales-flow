package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.ReportRepository = (*ReportRepo)(nil)

type ReportRepo struct {
	db *gorm.DB
}

func NewReportRepo(db *gorm.DB) *ReportRepo {
	return &ReportRepo{db: db}
}

func (r *ReportRepo) Create(ctx context.Context, rep *domain.Report) error {
	if err := r.db.WithContext(ctx).Create(rep).Error; err != nil {
		return fmt.Errorf("report.Create: %w", err)
	}
	return nil
}

func (r *ReportRepo) GetByID(ctx context.Context, id string) (*domain.Report, error) {
	var rep domain.Report
	if err := r.db.WithContext(ctx).First(&rep, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("report.GetByID: %w", err)
	}
	return &rep, nil
}

func (r *ReportRepo) List(ctx context.Context, f domain.ReportFilter, page, pageSize int) ([]domain.Report, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Report{})

	if f.ReportType != nil {
		q = q.Where("report_type = ?", *f.ReportType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("report.List count: %w", err)
	}

	offset := (page - 1) * pageSize
	var reports []domain.Report
	if err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&reports).Error; err != nil {
		return nil, 0, fmt.Errorf("report.List: %w", err)
	}
	return reports, total, nil
}

func (r *ReportRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Report{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("report.Delete: %w", err)
	}
	return nil
}
