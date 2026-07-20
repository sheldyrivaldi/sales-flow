package domain

import (
	"context"
	"time"
)

// ProjectStatus adalah kesehatan proyek berjalan dari kacamata sales.
type ProjectStatus string

const (
	ProjectOnTrack   ProjectStatus = "ON_TRACK"
	ProjectAtRisk    ProjectStatus = "AT_RISK"
	ProjectDelayed   ProjectStatus = "DELAYED"
	ProjectCompleted ProjectStatus = "COMPLETED"
)

func (s ProjectStatus) Valid() bool {
	switch s {
	case ProjectOnTrack, ProjectAtRisk, ProjectDelayed, ProjectCompleted:
		return true
	}
	return false
}

// ProjectMilestone adalah satu tonggak pencapaian proyek (checklist).
type ProjectMilestone struct {
	Title   string  `json:"title"`
	DueDate *string `json:"due_date,omitempty"`
	Done    bool    `json:"done"`
}

// ProjectActivity adalah satu catatan check-in/aktivitas pada proyek.
type ProjectActivity struct {
	Date time.Time `json:"date"`
	Note string    `json:"note"`
}

// Project adalah proyek BERJALAN (menu Proyek Berjalan): tender/prospek yang
// sudah dimenangkan (atau input manual) dan sedang dikerjakan — sales
// memantau progress, status kesehatan, milestone, dan catatan aktivitas.
type Project struct {
	ID            string             `json:"id"             gorm:"primaryKey;default:gen_random_uuid()"`
	Name          string             `json:"name"           gorm:"not null"`
	ClientName    *string            `json:"client_name"`
	ContractValue *float64           `json:"contract_value"`
	Currency      string             `json:"currency"       gorm:"not null;default:'IDR'"`
	StartDate     *time.Time         `json:"start_date"     gorm:"column:start_date"`
	EndDate       *time.Time         `json:"end_date"       gorm:"column:end_date"`
	Status        ProjectStatus      `json:"status"         gorm:"not null;default:'ON_TRACK'"`
	Progress      int                `json:"progress"       gorm:"not null;default:0"`
	Description   *string            `json:"description"`
	Milestones    []ProjectMilestone `json:"milestones"     gorm:"serializer:json;type:jsonb"`
	Activities    []ProjectActivity  `json:"activities"     gorm:"serializer:json;type:jsonb"`
	SourceType    *string            `json:"source_type"`
	SourceID      *string            `json:"source_id"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func (Project) TableName() string { return "project" }

type ProjectFilter struct {
	Status *ProjectStatus
	Search string
}

type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Project, error)
	List(ctx context.Context, f ProjectFilter, page, pageSize int) ([]Project, int64, error)
}
