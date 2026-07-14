package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

// businessTimeZone is the timezone "today" is anchored to for date-bucketed
// queries (e.g. CountDiscoveryToday). Hardcoded to WIB since SalesPilot is a
// single-workspace Indonesian product; evaluating dates in the DB session
// timezone (often UTC in containers) would misbucket rows for up to 7 hours
// around every local midnight.
const businessTimeZone = "Asia/Jakarta"

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

	if f.OnlyInbox {
		q = q.Where("origin = ? AND status = ? AND reviewed_at IS NULL", domain.OriginDiscovery, domain.TenderStatusIdentified)
	}
	if f.Status != nil {
		q = q.Where("status = ?", *f.Status)
	}
	if f.RecommendedAction != nil {
		q = q.Where("recommended_action = ?", *f.RecommendedAction)
	}
	if f.MinFitScore != nil {
		q = q.Where("fit_score >= ?", *f.MinFitScore)
	}
	if f.Origin != nil {
		q = q.Where("origin = ?", *f.Origin)
	}
	if f.BuyerName != "" {
		q = q.Where("buyer_name ILIKE ?", "%"+f.BuyerName+"%")
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

func (r *TenderRepo) TopByFitScore(ctx context.Context, limit int) ([]domain.Tender, error) {
	var tenders []domain.Tender
	// Only surface still-actionable opportunities as dashboard "priority":
	// a WON/LOST tender keeps its fit_score, so without this status filter a
	// closed deal with a high score would outrank genuinely-open ones.
	err := r.db.WithContext(ctx).
		Where("fit_score IS NOT NULL").
		Where("status NOT IN ?", []domain.TenderStatus{domain.TenderStatusWon, domain.TenderStatusLost}).
		Order("fit_score DESC").
		Order("submission_deadline ASC NULLS LAST").
		Limit(limit).
		Find(&tenders).Error
	if err != nil {
		return nil, fmt.Errorf("tender.TopByFitScore: %w", err)
	}
	return tenders, nil
}

func (r *TenderRepo) CountDiscoveryToday(ctx context.Context) (int64, error) {
	var count int64
	// Evaluate both sides of the date comparison in the business timezone so
	// "today" means the local calendar day, not the DB session's day.
	err := r.db.WithContext(ctx).Model(&domain.Tender{}).
		Where(
			"origin = ? AND (created_at AT TIME ZONE ?)::date = (now() AT TIME ZONE ?)::date",
			domain.OriginDiscovery, businessTimeZone, businessTimeZone,
		).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("tender.CountDiscoveryToday: %w", err)
	}
	return count, nil
}

func (r *TenderRepo) GetByDedupKey(ctx context.Context, key string) (*domain.Tender, error) {
	var t domain.Tender
	err := r.db.WithContext(ctx).First(&t, "dedup_key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tender.GetByDedupKey: %w", err)
	}
	return &t, nil
}
