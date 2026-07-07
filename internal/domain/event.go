package domain

import (
	"context"
	"time"
)

type EventType string

const (
	EventTypeExpo        EventType = "EXPO"
	EventTypeConference  EventType = "CONFERENCE"
	EventTypeSeminar     EventType = "SEMINAR"
	EventTypeWorkshop    EventType = "WORKSHOP"
	EventTypeNetworking  EventType = "NETWORKING"
	EventTypeOther       EventType = "OTHER"
)

func (t EventType) Valid() bool {
	switch t {
	case EventTypeExpo, EventTypeConference, EventTypeSeminar,
		EventTypeWorkshop, EventTypeNetworking, EventTypeOther:
		return true
	}
	return false
}

type EventStatus string

const (
	EventStatusPlanned   EventStatus = "PLANNED"
	EventStatusAttended  EventStatus = "ATTENDED"
	EventStatusCancelled EventStatus = "CANCELLED"
)

func (s EventStatus) Valid() bool {
	switch s {
	case EventStatusPlanned, EventStatusAttended, EventStatusCancelled:
		return true
	}
	return false
}

type Event struct {
	ID        string      `json:"id"         gorm:"primaryKey;default:gen_random_uuid()"`
	Name      string      `json:"name"       gorm:"not null"`
	Type      EventType   `json:"type"       gorm:"not null"`
	Date      *time.Time  `json:"date"       gorm:"column:event_date"`
	Location  *string     `json:"location"`
	Organizer *string     `json:"organizer"`
	Notes     *string     `json:"notes"`
	Status    EventStatus `json:"status"     gorm:"not null;default:'PLANNED'"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (Event) TableName() string { return "event" }

type EventFilter struct {
	Type   *EventType
	Status *EventStatus
	Search string
}

type EventRepository interface {
	Create(ctx context.Context, e *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	List(ctx context.Context, f EventFilter, page, pageSize int) ([]Event, int64, error)
	Update(ctx context.Context, e *Event) error
	Delete(ctx context.Context, id string) error
}
