package domain

import (
	"context"
	"encoding/json"
	"time"
)

type EventType string

const (
	EventTypeExpo       EventType = "EXPO"
	EventTypeConference EventType = "CONFERENCE"
	EventTypeSeminar    EventType = "SEMINAR"
	EventTypeWorkshop   EventType = "WORKSHOP"
	EventTypeNetworking EventType = "NETWORKING"
	EventTypeOther      EventType = "OTHER"
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

// EventAttachment adalah satu berkas pendukung event (rundown, undangan,
// denah booth). Disimpan sebagai metadata; berkasnya sendiri ada di UploadDir
// dan diakses lewat URL publik.
type EventAttachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Mime string `json:"mime,omitempty"`
	Size int64  `json:"size,omitempty"`
}

// Status Analisa AI.
const (
	AnalysisIdle    = "idle"
	AnalysisRunning = "running"
	AnalysisSuccess = "success"
	AnalysisFailed  = "failed"
)

type Event struct {
	ID        string      `json:"id"         gorm:"primaryKey;default:gen_random_uuid()"`
	Name      string      `json:"name"       gorm:"not null"`
	Type      EventType   `json:"type"       gorm:"not null"`
	Date      *time.Time  `json:"date"       gorm:"column:event_date"`
	Location  *string     `json:"location"`
	Organizer *string     `json:"organizer"`
	Notes     *string     `json:"notes"`
	Status    EventStatus `json:"status"     gorm:"not null;default:'PLANNED'"`
	// ParticipantEmails adalah undangan lepas — peserta TIDAK harus punya akun
	// di aplikasi ini, jadi sengaja disimpan sebagai alamat email biasa dan
	// bukan relasi ke tabel user.
	ParticipantEmails []string          `json:"participant_emails" gorm:"column:participant_emails;serializer:json;type:jsonb"`
	Attachments       []EventAttachment `json:"attachments"        gorm:"column:attachments;serializer:json;type:jsonb"`
	// Analysis menyimpan hasil Analisa AI apa adanya (json.RawMessage) supaya
	// skema analisa bisa berkembang tanpa migrasi ulang. Nilai utama menu ini
	// ada di sini, jadi ia HARUS bertahan antar-sesi.
	Analysis   json.RawMessage `json:"analysis,omitempty"    gorm:"column:analysis;type:jsonb"`
	AnalyzedAt *time.Time      `json:"analyzed_at,omitempty" gorm:"column:analyzed_at"`
	// AnalysisStatus mengunci event selama analisa berjalan: idle | running |
	// success | failed. UI memakainya untuk melarang edit, tambah lampiran,
	// dan analisa ulang agar hasil tidak dihitung dari data yang berubah di
	// tengah jalan.
	AnalysisStatus string    `json:"analysis_status" gorm:"column:analysis_status;not null;default:'idle'"`
	AnalysisError  *string   `json:"analysis_error,omitempty" gorm:"column:analysis_error"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Event) TableName() string { return "event" }

// EventFilter mendukung penyaringan multi-kolom bergaya Jira: satu kolom bisa
// dipilih beberapa nilai sekaligus (OR di dalam kolom), dan antar kolom
// digabung dengan AND.
type EventFilter struct {
	Types    []EventType
	Statuses []EventStatus
	// Search menyapu beberapa kolom teks sekaligus (nama, penyelenggara,
	// lokasi, catatan).
	Search    string
	DateFrom  *time.Time
	DateTo    *time.Time
	Location  string
	Organizer string
	// HasAttachment/HasParticipant: nil = tidak disaring.
	HasAttachment  *bool
	HasParticipant *bool
}

type EventRepository interface {
	Create(ctx context.Context, e *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	List(ctx context.Context, f EventFilter, page, pageSize int) ([]Event, int64, error)
	Update(ctx context.Context, e *Event) error
	Delete(ctx context.Context, id string) error
}
