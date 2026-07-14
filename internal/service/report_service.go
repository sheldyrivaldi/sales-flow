package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/telemetry"
)

// reportTitles gives each report_type a human-readable Bahasa Indonesia
// title for storage/display — independent of ai.reportTitle (used only
// inside the prompt), so the two layers stay decoupled.
var reportTitles = map[domain.ReportType]string{
	domain.ReportDailyDigest:    "Daily Opportunity Digest",
	domain.ReportWeeklyPipeline: "Weekly Pipeline Report",
	domain.ReportPerOpportunity: "Laporan Per-Peluang",
}

// ReportService orchestrates one report generation run end-to-end (EP-15):
// validate type + period, aggregate live pipeline data (via DashboardService,
// reused from EP-11 rather than re-querying repos directly), call the AI
// generator, and persist the resulting markdown.
type ReportService struct {
	gen       *ai.ReportGenerator
	repo      domain.ReportRepository
	dashboard *DashboardService
	emit      *telemetry.Emitter
}

func NewReportService(gen *ai.ReportGenerator, repo domain.ReportRepository, dashboard *DashboardService) *ReportService {
	return &ReportService{gen: gen, repo: repo, dashboard: dashboard}
}

// SetEmitter wires telemetry (EP-17 ST-17.1) after construction — optional,
// nil-safe.
func (s *ReportService) SetEmitter(e *telemetry.Emitter) { s.emit = e }

// Create validates type/period, aggregates current pipeline data, generates
// the report via Hermes, and persists it. On any AI failure it returns a
// friendly error and persists nothing (PRD §8: "gagal AI → pesan ramah, data
// utuh").
func (s *ReportService) Create(ctx context.Context, req *dto.ReportCreateRequest) (*domain.Report, error) {
	reportType := domain.ReportType(req.Type)
	if !reportType.Valid() {
		return nil, httperr.NewBadRequest("INVALID_TYPE", "type harus daily_digest, weekly_pipeline, atau per_opportunity")
	}

	start, err := time.Parse(time.RFC3339, req.PeriodStart)
	if err != nil {
		return nil, httperr.NewBadRequest("INVALID_DATE", "format period_start tidak valid, gunakan RFC3339")
	}
	end, err := time.Parse(time.RFC3339, req.PeriodEnd)
	if err != nil {
		return nil, httperr.NewBadRequest("INVALID_DATE", "format period_end tidak valid, gunakan RFC3339")
	}
	if start.After(end) {
		return nil, httperr.NewBadRequest("INVALID_PERIOD", "period_start harus sebelum atau sama dengan period_end")
	}

	summary, err := s.dashboard.Summary(ctx)
	if err != nil {
		return nil, fmt.Errorf("report.Create: aggregate: %w", err)
	}

	data := ai.ReportData{
		Pipeline:            summary.Pipeline,
		TotalPipelineCount:  summary.TotalPipelineCount,
		TotalPipelineValue:  summary.TotalPipelineValue,
		PriorityTenders:     summary.PriorityTenders,
		DiscoveryTodayCount: summary.DiscoveryTodayCount,
	}

	genStart := time.Now()
	content, err := s.gen.Generate(ctx, reportType, ai.ReportPeriod{Start: start, End: end}, data)
	if err != nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "Generate laporan AI sedang tidak tersedia, coba lagi nanti")
	}
	genDuration := time.Since(genStart)

	model := ai.ModelLabel
	report := &domain.Report{
		ReportType:  reportType,
		Title:       reportTitles[reportType],
		PeriodStart: start,
		PeriodEnd:   end,
		Content:     content,
		Model:       &model,
	}
	if err := s.repo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("report.Create: %w", err)
	}

	if s.emit != nil {
		s.emit.Emit(ctx, "report_generated", map[string]any{
			"report_type": string(reportType),
			"duration_ms": genDuration.Milliseconds(),
		})
	}

	return report, nil
}

// List returns paginated reports, optionally filtered by type.
func (s *ReportService) List(ctx context.Context, f domain.ReportFilter, page, pageSize int) ([]domain.Report, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	items, total, err := s.repo.List(ctx, f, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("report.List: %w", err)
	}
	return items, total, nil
}

// Get returns a report by ID.
func (s *ReportService) Get(ctx context.Context, id string) (*domain.Report, error) {
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("laporan tidak ditemukan")
		}
		return nil, fmt.Errorf("report.Get: %w", err)
	}
	return r, nil
}

// Delete removes a report by ID.
func (s *ReportService) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("report.Delete: %w", err)
	}
	return nil
}
