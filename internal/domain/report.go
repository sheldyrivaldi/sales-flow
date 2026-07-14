package domain

import (
	"context"
	"time"
)

// ReportType enumerates the 3 report kinds EP-15 generates.
type ReportType string

const (
	ReportDailyDigest    ReportType = "daily_digest"
	ReportWeeklyPipeline ReportType = "weekly_pipeline"
	ReportPerOpportunity ReportType = "per_opportunity"
)

func (t ReportType) Valid() bool {
	switch t {
	case ReportDailyDigest, ReportWeeklyPipeline, ReportPerOpportunity:
		return true
	}
	return false
}

// Report is one generated report (EP-15): a markdown document covering a
// period, produced by internal/ai's report generator from a live
// pipeline/activity aggregation snapshot at generation time.
type Report struct {
	ID          string     `json:"id"           gorm:"primaryKey;default:gen_random_uuid()"`
	ReportType  ReportType `json:"report_type"  gorm:"column:report_type;not null"`
	Title       string     `json:"title"        gorm:"not null"`
	PeriodStart time.Time  `json:"period_start" gorm:"column:period_start;not null"`
	PeriodEnd   time.Time  `json:"period_end"   gorm:"column:period_end;not null"`
	Content     string     `json:"content"      gorm:"not null"`
	Model       *string    `json:"model"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Report) TableName() string { return "report" }

// ReportFilter narrows List to one report_type when set.
type ReportFilter struct {
	ReportType *ReportType
}

type ReportRepository interface {
	Create(ctx context.Context, r *Report) error
	GetByID(ctx context.Context, id string) (*Report, error)
	List(ctx context.Context, f ReportFilter, page, pageSize int) ([]Report, int64, error)
	Delete(ctx context.Context, id string) error
}
