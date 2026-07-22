package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.FeedbackFormRepository = (*FeedbackFormRepo)(nil)

type FeedbackFormRepo struct {
	db *gorm.DB
}

func NewFeedbackFormRepo(db *gorm.DB) *FeedbackFormRepo {
	return &FeedbackFormRepo{db: db}
}

func (r *FeedbackFormRepo) Create(ctx context.Context, f *domain.FeedbackForm) error {
	if err := r.db.WithContext(ctx).Create(f).Error; err != nil {
		return fmt.Errorf("feedbackForm.Create: %w", err)
	}
	return nil
}

func (r *FeedbackFormRepo) Update(ctx context.Context, f *domain.FeedbackForm) error {
	if err := r.db.WithContext(ctx).Save(f).Error; err != nil {
		return fmt.Errorf("feedbackForm.Update: %w", err)
	}
	return nil
}

func (r *FeedbackFormRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.FeedbackForm{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("feedbackForm.Delete: %w", err)
	}
	return nil
}

func (r *FeedbackFormRepo) GetByID(ctx context.Context, id string) (*domain.FeedbackForm, error) {
	var f domain.FeedbackForm
	if err := r.db.WithContext(ctx).First(&f, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("feedbackForm.GetByID: %w", err)
	}
	return &f, nil
}

func (r *FeedbackFormRepo) GetBySlug(ctx context.Context, slug string) (*domain.FeedbackForm, error) {
	var f domain.FeedbackForm
	err := r.db.WithContext(ctx).First(&f, "slug = ?", slug).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("feedbackForm.GetBySlug: %w", err)
	}
	return &f, nil
}

func (r *FeedbackFormRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&domain.FeedbackForm{}).
		Where("slug = ?", slug).Count(&n).Error; err != nil {
		return false, fmt.Errorf("feedbackForm.SlugExists: %w", err)
	}
	return n > 0, nil
}

func (r *FeedbackFormRepo) List(ctx context.Context) ([]domain.FeedbackForm, error) {
	var items []domain.FeedbackForm
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("feedbackForm.List: %w", err)
	}
	if len(items) == 0 {
		return items, nil
	}

	// Hitung jumlah submission per form dalam satu query agregat, lalu petakan.
	type countRow struct {
		FormID string
		N      int64
	}
	var rows []countRow
	if err := r.db.WithContext(ctx).Model(&domain.FeedbackFormSubmission{}).
		Select("form_id, COUNT(*) AS n").Group("form_id").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("feedbackForm.List: count submissions: %w", err)
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.FormID] = row.N
	}
	for i := range items {
		items[i].SubmissionCount = counts[items[i].ID]
	}
	return items, nil
}

func (r *FeedbackFormRepo) CreateSubmission(ctx context.Context, s *domain.FeedbackFormSubmission) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("feedbackForm.CreateSubmission: %w", err)
	}
	return nil
}

func (r *FeedbackFormRepo) ListSubmissions(ctx context.Context, formID string) ([]domain.FeedbackFormSubmission, error) {
	var items []domain.FeedbackFormSubmission
	if err := r.db.WithContext(ctx).Where("form_id = ?", formID).
		Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("feedbackForm.ListSubmissions: %w", err)
	}
	return items, nil
}

func (r *FeedbackFormRepo) ListAllSubmissions(ctx context.Context) ([]domain.FeedbackFormSubmission, error) {
	var items []domain.FeedbackFormSubmission
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("feedbackForm.ListAllSubmissions: %w", err)
	}
	return items, nil
}
