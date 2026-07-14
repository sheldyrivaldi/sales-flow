package service

import (
	"context"
	"testing"

	"salespilot/internal/domain"
)

type fakeDashboardProspectRepo struct {
	summary []domain.ProspectStageSummary
}

func (r *fakeDashboardProspectRepo) Create(_ context.Context, _ *domain.Prospect) error { return nil }
func (r *fakeDashboardProspectRepo) GetByID(_ context.Context, _ string) (*domain.Prospect, error) {
	return nil, nil
}
func (r *fakeDashboardProspectRepo) GetBySource(_ context.Context, _ domain.ProspectSource, _ string) (*domain.Prospect, error) {
	return nil, nil
}
func (r *fakeDashboardProspectRepo) List(_ context.Context, _ domain.ProspectFilter, _, _ int) ([]domain.Prospect, int64, error) {
	return nil, 0, nil
}
func (r *fakeDashboardProspectRepo) Update(_ context.Context, _ *domain.Prospect) error { return nil }
func (r *fakeDashboardProspectRepo) Delete(_ context.Context, _ string) error           { return nil }
func (r *fakeDashboardProspectRepo) SummaryByStage(_ context.Context) ([]domain.ProspectStageSummary, error) {
	return r.summary, nil
}

type fakeDashboardTenderRepo struct {
	topByFitScore  []domain.Tender
	discoveryToday int64
}

func (r *fakeDashboardTenderRepo) Create(_ context.Context, _ *domain.Tender) error { return nil }
func (r *fakeDashboardTenderRepo) GetByID(_ context.Context, _ string) (*domain.Tender, error) {
	return nil, nil
}
func (r *fakeDashboardTenderRepo) List(_ context.Context, _ domain.TenderFilter, _, _ int) ([]domain.Tender, int64, error) {
	return nil, 0, nil
}
func (r *fakeDashboardTenderRepo) Update(_ context.Context, _ *domain.Tender) error { return nil }
func (r *fakeDashboardTenderRepo) Delete(_ context.Context, _ string) error         { return nil }
func (r *fakeDashboardTenderRepo) TopByFitScore(_ context.Context, limit int) ([]domain.Tender, error) {
	if limit < len(r.topByFitScore) {
		return r.topByFitScore[:limit], nil
	}
	return r.topByFitScore, nil
}
func (r *fakeDashboardTenderRepo) CountDiscoveryToday(_ context.Context) (int64, error) {
	return r.discoveryToday, nil
}
func (r *fakeDashboardTenderRepo) GetByDedupKey(_ context.Context, _ string) (*domain.Tender, error) {
	return nil, nil
}

func TestDashboardService_Summary_Aggregates(t *testing.T) {
	prospects := &fakeDashboardProspectRepo{summary: []domain.ProspectStageSummary{
		{Stage: domain.ProspectStageNew, Count: 3, TotalValue: 100},
		{Stage: domain.ProspectStageWon, Count: 2, TotalValue: 250},
	}}
	fitScore88 := 88
	tenders := &fakeDashboardTenderRepo{
		topByFitScore:  []domain.Tender{{ID: "t1", Title: "Tender A", FitScore: &fitScore88}},
		discoveryToday: 4,
	}

	svc := NewDashboardService(prospects, tenders)
	summary, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary error: %v", err)
	}

	if summary.TotalPipelineCount != 5 {
		t.Errorf("TotalPipelineCount = %d, want 5", summary.TotalPipelineCount)
	}
	if summary.TotalPipelineValue != 350 {
		t.Errorf("TotalPipelineValue = %v, want 350", summary.TotalPipelineValue)
	}
	if len(summary.PriorityTenders) != 1 || summary.PriorityTenders[0].ID != "t1" {
		t.Errorf("PriorityTenders = %+v, want [t1]", summary.PriorityTenders)
	}
	if summary.DiscoveryTodayCount != 4 {
		t.Errorf("DiscoveryTodayCount = %d, want 4", summary.DiscoveryTodayCount)
	}
}

func TestDashboardService_Summary_EmptyDB(t *testing.T) {
	svc := NewDashboardService(&fakeDashboardProspectRepo{}, &fakeDashboardTenderRepo{})
	summary, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary error on empty DB: %v", err)
	}
	if summary.TotalPipelineCount != 0 || summary.TotalPipelineValue != 0 {
		t.Errorf("expected zero pipeline totals on empty DB, got count=%d value=%v",
			summary.TotalPipelineCount, summary.TotalPipelineValue)
	}
	if len(summary.PriorityTenders) != 0 {
		t.Errorf("expected empty PriorityTenders on empty DB, got %+v", summary.PriorityTenders)
	}
	if summary.DiscoveryTodayCount != 0 {
		t.Errorf("expected DiscoveryTodayCount 0, got %d", summary.DiscoveryTodayCount)
	}
}

func TestDashboardService_Summary_PriorityLimit(t *testing.T) {
	tenders := &fakeDashboardTenderRepo{}
	for i := 0; i < 10; i++ {
		score := 100 - i
		tenders.topByFitScore = append(tenders.topByFitScore, domain.Tender{ID: "t", FitScore: &score})
	}

	svc := NewDashboardService(&fakeDashboardProspectRepo{}, tenders)
	summary, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary error: %v", err)
	}
	if len(summary.PriorityTenders) != dashboardPriorityLimit {
		t.Errorf("PriorityTenders length = %d, want %d (dashboardPriorityLimit)",
			len(summary.PriorityTenders), dashboardPriorityLimit)
	}
}
