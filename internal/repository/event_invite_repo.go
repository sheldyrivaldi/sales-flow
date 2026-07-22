package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.EventInviteRepository = (*EventInviteRepo)(nil)

type EventInviteRepo struct {
	db *gorm.DB
}

func NewEventInviteRepo(db *gorm.DB) *EventInviteRepo {
	return &EventInviteRepo{db: db}
}

func (r *EventInviteRepo) Create(ctx context.Context, inv *domain.EventInvite) error {
	if err := r.db.WithContext(ctx).Create(inv).Error; err != nil {
		return fmt.Errorf("eventInvite.Create: %w", err)
	}
	return nil
}

func (r *EventInviteRepo) Update(ctx context.Context, inv *domain.EventInvite) error {
	if err := r.db.WithContext(ctx).Save(inv).Error; err != nil {
		return fmt.Errorf("eventInvite.Update: %w", err)
	}
	return nil
}

func (r *EventInviteRepo) GetByID(ctx context.Context, id string) (*domain.EventInvite, error) {
	var inv domain.EventInvite
	if err := r.db.WithContext(ctx).First(&inv, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("eventInvite.GetByID: %w", err)
	}
	return &inv, nil
}

func (r *EventInviteRepo) ListByEvent(ctx context.Context, eventID string) ([]domain.EventInvite, error) {
	var items []domain.EventInvite
	if err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Order("scheduled_at ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("eventInvite.ListByEvent: %w", err)
	}
	return items, nil
}

func (r *EventInviteRepo) ListDue(ctx context.Context, now time.Time, limit int) ([]domain.EventInvite, error) {
	var items []domain.EventInvite
	if err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ?", domain.InviteStatusPending, now).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("eventInvite.ListDue: %w", err)
	}
	return items, nil
}

func (r *EventInviteRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.EventInvite{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("eventInvite.Delete: %w", err)
	}
	return nil
}
