package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.PlaybookJobRepository = (*PlaybookJobRepo)(nil)

type PlaybookJobRepo struct {
	db *gorm.DB
}

func NewPlaybookJobRepo(db *gorm.DB) *PlaybookJobRepo {
	return &PlaybookJobRepo{db: db}
}

func (r *PlaybookJobRepo) Create(ctx context.Context, j *domain.PlaybookJob) error {
	if err := r.db.WithContext(ctx).Create(j).Error; err != nil {
		return fmt.Errorf("playbookJob.Create: %w", err)
	}
	return nil
}

func (r *PlaybookJobRepo) Update(ctx context.Context, j *domain.PlaybookJob) error {
	if err := r.db.WithContext(ctx).Save(j).Error; err != nil {
		return fmt.Errorf("playbookJob.Update: %w", err)
	}
	return nil
}

func (r *PlaybookJobRepo) GetByID(ctx context.Context, id string) (*domain.PlaybookJob, error) {
	var j domain.PlaybookJob
	if err := r.db.WithContext(ctx).First(&j, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("playbookJob.GetByID: %w", err)
	}
	return &j, nil
}

func (r *PlaybookJobRepo) List(ctx context.Context) ([]domain.PlaybookJob, error) {
	var items []domain.PlaybookJob
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("playbookJob.List: %w", err)
	}
	return items, nil
}

func (r *PlaybookJobRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.PlaybookJob{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("playbookJob.Delete: %w", err)
	}
	return nil
}

// FailInterrupted menandai semua job yang masih in_progress/updating menjadi
// failed — dipanggil saat boot karena worker in-process tidak bertahan
// melewati restart, jadi job "berjalan" apa pun setelah restart pasti mati.
func (r *PlaybookJobRepo) FailInterrupted(ctx context.Context) error {
	msg := "Proses terputus karena server dimulai ulang. Silakan generate ulang."
	if err := r.db.WithContext(ctx).
		Model(&domain.PlaybookJob{}).
		Where("status IN ?", []string{string(domain.PlaybookJobInProgress), string(domain.PlaybookJobUpdating)}).
		Updates(map[string]any{"status": string(domain.PlaybookJobFailed), "error_message": msg}).Error; err != nil {
		return fmt.Errorf("playbookJob.FailInterrupted: %w", err)
	}
	return nil
}
