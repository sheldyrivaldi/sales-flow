package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.EventRepository = (*EventRepo)(nil)

type EventRepo struct {
	db *gorm.DB
}

func NewEventRepo(db *gorm.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) Create(ctx context.Context, e *domain.Event) error {
	if err := r.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("event.Create: %w", err)
	}
	return nil
}

func (r *EventRepo) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	var e domain.Event
	if err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("event.GetByID: %w", err)
	}
	return &e, nil
}

func (r *EventRepo) List(ctx context.Context, f domain.EventFilter, page, pageSize int) ([]domain.Event, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Event{})

	// Beberapa nilai dalam satu kolom = OR (IN), antar kolom = AND.
	if len(f.Types) > 0 {
		q = q.Where("type IN ?", f.Types)
	}
	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	}
	if f.Search != "" {
		// Kurung WAJIB: tanpa itu, AND mengikat lebih kuat dari OR sehingga
		// filter lain (mis. type) hanya berlaku pada cabang OR pertama dan
		// baris yang tidak cocok ikut lolos.
		like := "%" + f.Search + "%"
		q = q.Where(
			"(name ILIKE ? OR organizer ILIKE ? OR location ILIKE ? OR notes ILIKE ?)",
			like, like, like, like,
		)
	}
	if f.Location != "" {
		q = q.Where("location ILIKE ?", "%"+f.Location+"%")
	}
	if f.Organizer != "" {
		q = q.Where("organizer ILIKE ?", "%"+f.Organizer+"%")
	}
	if f.DateFrom != nil {
		q = q.Where("event_date >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where("event_date <= ?", *f.DateTo)
	}
	// jsonb_array_length butuh kolom non-null; default '[]' menjamin itu.
	if f.HasAttachment != nil {
		if *f.HasAttachment {
			q = q.Where("jsonb_array_length(attachments) > 0")
		} else {
			q = q.Where("jsonb_array_length(attachments) = 0")
		}
	}
	if f.HasParticipant != nil {
		if *f.HasParticipant {
			q = q.Where("jsonb_array_length(participant_emails) > 0")
		} else {
			q = q.Where("jsonb_array_length(participant_emails) = 0")
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("event.List count: %w", err)
	}

	offset := (page - 1) * pageSize
	var events []domain.Event
	if err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("event.List: %w", err)
	}
	return events, total, nil
}

func (r *EventRepo) Update(ctx context.Context, e *domain.Event) error {
	if err := r.db.WithContext(ctx).Save(e).Error; err != nil {
		return fmt.Errorf("event.Update: %w", err)
	}
	return nil
}

func (r *EventRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Event{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("event.Delete: %w", err)
	}
	return nil
}
