package dto

import (
	"time"

	"salespilot/internal/domain"
)

// ReportCreateRequest is POST /api/reports — dates are RFC3339 strings
// (parsed + period-order validated in service.ReportService.Create).
type ReportCreateRequest struct {
	Type        string `json:"type"         validate:"required,oneof=daily_digest weekly_pipeline per_opportunity"`
	PeriodStart string `json:"period_start" validate:"required"`
	PeriodEnd   string `json:"period_end"   validate:"required"`
}

// ReportResponse is the JSON shape for one report (create/get/list item).
type ReportResponse struct {
	ID          string    `json:"id"`
	ReportType  string    `json:"report_type"`
	Title       string    `json:"title"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Content     string    `json:"content"`
	Model       *string   `json:"model"`
	CreatedAt   time.Time `json:"created_at"`
}

// ToReportResponse maps domain.Report → ReportResponse.
func ToReportResponse(r domain.Report) ReportResponse {
	return ReportResponse{
		ID:          r.ID,
		ReportType:  string(r.ReportType),
		Title:       r.Title,
		PeriodStart: r.PeriodStart,
		PeriodEnd:   r.PeriodEnd,
		Content:     r.Content,
		Model:       r.Model,
		CreatedAt:   r.CreatedAt,
	}
}

// ReportListResponse is the P-4 paginated list shape.
type ReportListResponse struct {
	Items    []ReportResponse `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}
