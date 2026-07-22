package domain

import (
	"context"
	"encoding/json"
	"time"
)

// PlaybookJobStatus adalah siklus hidup satu generate playbook async.
type PlaybookJobStatus string

const (
	PlaybookJobInProgress PlaybookJobStatus = "in_progress"
	PlaybookJobUpdating   PlaybookJobStatus = "updating"
	PlaybookJobSuccess    PlaybookJobStatus = "success"
	PlaybookJobFailed     PlaybookJobStatus = "failed"
)

// PlaybookJob adalah satu entri riwayat generate playbook (menu Playbooks).
// Dibuat langsung saat user menekan generate (status in_progress), lalu
// di-update worker background. content berisi hasil PlaybookContent saat
// success. error_message menjelaskan kegagalan saat failed.
// PlaybookRevision adalah satu entri riwayat revisi (prompt + lampiran).
type PlaybookRevision struct {
	Instruction    string    `json:"instruction"`
	AttachmentName *string   `json:"attachment_name,omitempty"`
	AttachmentURL  *string   `json:"attachment_url,omitempty"`
	At             time.Time `json:"at"`
}

type PlaybookJob struct {
	ID    string `json:"id"           gorm:"primaryKey;default:gen_random_uuid()"`
	Title string `json:"title"        gorm:"not null"`
	// UserTitled menandai judul diisi user di form create — judul seperti itu
	// TIDAK boleh ditimpa judul karangan AI saat callback selesai.
	UserTitled bool   `json:"user_titled" gorm:"column:user_titled;not null;default:false"`
	Prompt     string `json:"prompt"       gorm:"not null"`
	Status         PlaybookJobStatus `json:"status"       gorm:"not null;default:'in_progress'"`
	Content        json.RawMessage   `json:"content,omitempty"       gorm:"type:jsonb"`
	ErrorMessage   *string           `json:"error_message,omitempty" gorm:"column:error_message"`
	AttachmentName *string           `json:"attachment_name,omitempty" gorm:"column:attachment_name"`
	AttachmentURL  *string           `json:"attachment_url,omitempty"  gorm:"column:attachment_url"`
	// Revisions menyimpan riwayat prompt revisi + lampirannya (bisa dibuka).
	Revisions []PlaybookRevision `json:"revisions" gorm:"column:revisions;serializer:json;type:jsonb"`
	Source    string             `json:"source"    gorm:"not null;default:'custom'"`
	// EventID menautkan playbook ke event asalnya (Source=="event"). SATU event
	// hanya boleh punya SATU playbook tertaut: generate ulang melepas yang lama
	// (EventID di-nil-kan) dan menautkan yang baru. nil untuk playbook custom
	// atau playbook event yang tautannya sudah dilepas.
	EventID   *string   `json:"event_id,omitempty" gorm:"column:event_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PlaybookJob) TableName() string { return "playbook_job" }

type PlaybookJobRepository interface {
	Create(ctx context.Context, j *PlaybookJob) error
	Update(ctx context.Context, j *PlaybookJob) error
	GetByID(ctx context.Context, id string) (*PlaybookJob, error)
	// GetByEventID mengembalikan playbook yang saat ini tertaut ke sebuah event,
	// atau (nil, nil) bila belum ada — dipakai untuk melepas tautan lama sebelum
	// generate ulang.
	GetByEventID(ctx context.Context, eventID string) (*PlaybookJob, error)
	List(ctx context.Context) ([]PlaybookJob, error)
	Delete(ctx context.Context, id string) error
}
