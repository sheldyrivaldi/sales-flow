package service

import (
	"context"
	"fmt"

	"salespilot/internal/domain"
)

// dashboardPriorityLimit caps how many top-scored tenders the dashboard
// "prioritas" section surfaces.
const dashboardPriorityLimit = 5

// DashboardSummary aggregates the numbers the Dashboard page needs in a
// single round-trip: pipeline (per-stage prospect count/value), top-scored
// tenders, and today's discovery count.
type DashboardSummary struct {
	Pipeline            []domain.ProspectStageSummary
	TotalPipelineCount  int64
	TotalPipelineValue  float64
	PriorityTenders     []domain.Tender
	DiscoveryTodayCount int64
}

// DashboardService aggregates read-only cross-entity summaries. It holds no
// state of its own — every number is computed live from ProspectRepository/
// TenderRepository on each call (EP-11 AC: "agregasi SQL efisien").
type DashboardService struct {
	prospects domain.ProspectRepository
	tenders   domain.TenderRepository
}

func NewDashboardService(prospects domain.ProspectRepository, tenders domain.TenderRepository) *DashboardService {
	return &DashboardService{prospects: prospects, tenders: tenders}
}

// Summary computes the full dashboard payload. All three sub-queries degrade
// to zero/empty on an empty database rather than erroring.
func (s *DashboardService) Summary(ctx context.Context) (*DashboardSummary, error) {
	pipeline, err := s.prospects.SummaryByStage(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard.Summary: pipeline: %w", err)
	}

	var totalCount int64
	var totalValue float64
	for _, p := range pipeline {
		totalCount += p.Count
		totalValue += p.TotalValue
	}

	priority, err := s.tenders.TopByFitScore(ctx, dashboardPriorityLimit)
	if err != nil {
		return nil, fmt.Errorf("dashboard.Summary: priority tenders: %w", err)
	}

	discoveryToday, err := s.tenders.CountDiscoveryToday(ctx)
	if err != nil {
		return nil, fmt.Errorf("dashboard.Summary: discovery today: %w", err)
	}

	return &DashboardSummary{
		Pipeline:            pipeline,
		TotalPipelineCount:  totalCount,
		TotalPipelineValue:  totalValue,
		PriorityTenders:     priority,
		DiscoveryTodayCount: discoveryToday,
	}, nil
}
