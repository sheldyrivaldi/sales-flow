package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
)

type fakeReportRepo struct{ rows []domain.Report }

func (r *fakeReportRepo) Create(_ context.Context, rep *domain.Report) error {
	if rep.ID == "" {
		rep.ID = fmt.Sprintf("report-%d", len(r.rows)+1)
	}
	r.rows = append(r.rows, *rep)
	return nil
}

func (r *fakeReportRepo) GetByID(_ context.Context, id string) (*domain.Report, error) {
	for _, rep := range r.rows {
		if rep.ID == id {
			return &rep, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeReportRepo) List(_ context.Context, f domain.ReportFilter, page, pageSize int) ([]domain.Report, int64, error) {
	var filtered []domain.Report
	for _, rep := range r.rows {
		if f.ReportType != nil && rep.ReportType != *f.ReportType {
			continue
		}
		filtered = append(filtered, rep)
	}
	total := int64(len(filtered))
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (r *fakeReportRepo) Delete(_ context.Context, id string) error {
	for i, rep := range r.rows {
		if rep.ID == id {
			r.rows = append(r.rows[:i], r.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func newTestReportService(stub *stubHermesClient) (*ReportService, *fakeReportRepo) {
	tenderRepo := &fakeScoreTenderRepo{items: map[string]domain.Tender{}}
	prospectRepo := &fakeScoreProspectRepo{items: map[string]domain.Prospect{}}
	reportRepo := &fakeReportRepo{}

	dashboardSvc := NewDashboardService(prospectRepo, tenderRepo)
	gen := ai.NewReportGenerator(stub, "sk-test")
	svc := NewReportService(gen, reportRepo, dashboardSvc)
	return svc, reportRepo
}

func TestReportService_Create_Success(t *testing.T) {
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			return hermes.ChatResponse{Content: "# Weekly Pipeline Report\n\n## Ringkasan\nBaik."}, nil
		},
	}
	svc, reportRepo := newTestReportService(stub)

	report, err := svc.Create(context.Background(), &dto.ReportCreateRequest{
		Type:        "weekly_pipeline",
		PeriodStart: "2026-07-01T00:00:00Z",
		PeriodEnd:   "2026-07-07T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if report.Content == "" {
		t.Error("Content is empty")
	}
	if report.ReportType != domain.ReportWeeklyPipeline {
		t.Errorf("ReportType = %q, want %q", report.ReportType, domain.ReportWeeklyPipeline)
	}
	if len(reportRepo.rows) != 1 {
		t.Fatalf("report rows = %d, want 1", len(reportRepo.rows))
	}
}

func TestReportService_Create_EmitsTelemetry(t *testing.T) {
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			return hermes.ChatResponse{Content: "# Weekly Pipeline Report\n\n## Ringkasan\nBaik."}, nil
		},
	}
	svc, _ := newTestReportService(stub)

	emitter, repo := newTestEmitter()
	svc.SetEmitter(emitter)

	if _, err := svc.Create(context.Background(), &dto.ReportCreateRequest{
		Type:        "weekly_pipeline",
		PeriodStart: "2026-07-01T00:00:00Z",
		PeriodEnd:   "2026-07-07T00:00:00Z",
	}); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	waitForEvents(t, repo, "report_generated", 1)
}

func TestReportService_Create_InvalidType(t *testing.T) {
	svc, reportRepo := newTestReportService(&stubHermesClient{})

	_, err := svc.Create(context.Background(), &dto.ReportCreateRequest{
		Type:        "not_a_real_type",
		PeriodStart: "2026-07-01T00:00:00Z",
		PeriodEnd:   "2026-07-07T00:00:00Z",
	})
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
	if len(reportRepo.rows) != 0 {
		t.Errorf("report rows = %d, want 0", len(reportRepo.rows))
	}
}

func TestReportService_Create_InvalidPeriod_StartAfterEnd(t *testing.T) {
	svc, reportRepo := newTestReportService(&stubHermesClient{})

	_, err := svc.Create(context.Background(), &dto.ReportCreateRequest{
		Type:        "daily_digest",
		PeriodStart: "2026-07-10T00:00:00Z",
		PeriodEnd:   "2026-07-01T00:00:00Z",
	})
	if err == nil {
		t.Fatal("expected error when period_start > period_end, got nil")
	}
	if len(reportRepo.rows) != 0 {
		t.Errorf("report rows = %d, want 0", len(reportRepo.rows))
	}
}

func TestReportService_Create_AIFailure_NoRow(t *testing.T) {
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			return hermes.ChatResponse{}, errors.New("hermes down")
		},
	}
	svc, reportRepo := newTestReportService(stub)

	_, err := svc.Create(context.Background(), &dto.ReportCreateRequest{
		Type:        "daily_digest",
		PeriodStart: "2026-07-01T00:00:00Z",
		PeriodEnd:   "2026-07-01T23:59:59Z",
	})
	if err == nil {
		t.Fatal("expected error when Hermes fails, got nil")
	}
	if len(reportRepo.rows) != 0 {
		t.Errorf("report rows = %d, want 0 (no partial write on AI failure)", len(reportRepo.rows))
	}
}

func TestReportService_List_FiltersByType(t *testing.T) {
	svc, reportRepo := newTestReportService(&stubHermesClient{})
	reportRepo.rows = []domain.Report{
		{ID: "r1", ReportType: domain.ReportDailyDigest},
		{ID: "r2", ReportType: domain.ReportWeeklyPipeline},
		{ID: "r3", ReportType: domain.ReportDailyDigest},
	}

	dailyType := domain.ReportDailyDigest
	items, total, err := svc.List(context.Background(), domain.ReportFilter{ReportType: &dailyType}, 1, 20)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("items/total = %d/%d, want 2/2", len(items), total)
	}
}

func TestReportService_Delete_NotFound(t *testing.T) {
	svc, _ := newTestReportService(&stubHermesClient{})

	err := svc.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}
