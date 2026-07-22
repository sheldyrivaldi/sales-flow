package domain

import (
	"context"
	"time"
)

// Status siklus hidup satu undangan terjadwal.
const (
	InviteStatusPending   = "pending"
	InviteStatusSent      = "sent"
	InviteStatusFailed    = "failed"
	InviteStatusCancelled = "cancelled"
)

// EventInvite adalah satu undangan event yang dijadwalkan dikirim dari server
// pada waktu tertentu ke seluruh daftar peserta. Subject/Body dibekukan saat
// dijadwalkan (snapshot) agar isi undangan tidak berubah bila event diedit
// sebelum terkirim.
type EventInvite struct {
	ID          string     `json:"id"           gorm:"primaryKey;default:gen_random_uuid()"`
	EventID     string     `json:"event_id"     gorm:"column:event_id;not null"`
	Subject     string     `json:"subject"      gorm:"not null"`
	Body        string     `json:"body"         gorm:"not null"`
	SenderName  *string    `json:"sender_name,omitempty"  gorm:"column:sender_name"`
	SenderEmail *string    `json:"sender_email,omitempty" gorm:"column:sender_email"`
	Recipients  []string   `json:"recipients"   gorm:"column:recipients;serializer:json;type:jsonb"`
	ScheduledAt time.Time  `json:"scheduled_at" gorm:"column:scheduled_at;not null"`
	Status      string     `json:"status"       gorm:"not null;default:'pending'"`
	SentAt      *time.Time `json:"sent_at,omitempty" gorm:"column:sent_at"`
	Error       *string    `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (EventInvite) TableName() string { return "event_invite" }

type EventInviteRepository interface {
	Create(ctx context.Context, inv *EventInvite) error
	Update(ctx context.Context, inv *EventInvite) error
	GetByID(ctx context.Context, id string) (*EventInvite, error)
	ListByEvent(ctx context.Context, eventID string) ([]EventInvite, error)
	// ListDue mengembalikan undangan pending yang waktunya sudah tiba (<= now),
	// dipakai scheduler untuk mengirim.
	ListDue(ctx context.Context, now time.Time, limit int) ([]EventInvite, error)
	Delete(ctx context.Context, id string) error
}
