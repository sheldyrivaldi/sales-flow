package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

func samplePeriod() ReportPeriod {
	return ReportPeriod{
		Start: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC),
	}
}

func sampleReportData() ReportData {
	score := 85
	return ReportData{
		Pipeline: []domain.ProspectStageSummary{
			{Stage: domain.ProspectStageNew, Count: 3, TotalValue: 300_000_000},
			{Stage: domain.ProspectStageQualified, Count: 2, TotalValue: 500_000_000},
		},
		TotalPipelineCount: 5,
		TotalPipelineValue: 800_000_000,
		PriorityTenders: []domain.Tender{
			{Title: "Tender Prioritas A", FitScore: &score},
		},
		DiscoveryTodayCount: 4,
	}
}

func TestReportGenerator_Generate_AllThreeTypes(t *testing.T) {
	types := []domain.ReportType{
		domain.ReportDailyDigest,
		domain.ReportWeeklyPipeline,
		domain.ReportPerOpportunity,
	}

	for _, rt := range types {
		t.Run(string(rt), func(t *testing.T) {
			stub := &stubHermesClient{
				chat: func(_ context.Context, req hermes.ChatRequest) (hermes.ChatResponse, error) {
					if len(req.Messages) != 1 {
						t.Fatalf("expected 1 message, got %d", len(req.Messages))
					}
					prompt := req.Messages[0].Content
					if !strings.Contains(prompt, "Tender Prioritas A") {
						t.Error("prompt missing priority tender title")
					}
					return hermes.ChatResponse{Content: "# " + reportTitle(rt) + "\n\n## Ringkasan\nBaik.\n\n## Tabel Pipeline\n...\n\n## Prospek Prioritas\n...\n\n## Insight AI\n..."}, nil
				},
			}
			gen := NewReportGenerator(stub, "sk-test")

			md, err := gen.Generate(context.Background(), rt, samplePeriod(), sampleReportData())
			if err != nil {
				t.Fatalf("Generate error: %v", err)
			}
			for _, heading := range []string{"## Ringkasan", "## Tabel Pipeline", "## Prospek Prioritas", "## Insight AI"} {
				if !strings.Contains(md, heading) {
					t.Errorf("markdown missing heading %q", heading)
				}
			}
		})
	}
}

func TestReportGenerator_Generate_HermesError(t *testing.T) {
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			return hermes.ChatResponse{}, errors.New("hermes down")
		},
	}
	gen := NewReportGenerator(stub, "sk-test")

	_, err := gen.Generate(context.Background(), domain.ReportDailyDigest, samplePeriod(), sampleReportData())
	if err == nil {
		t.Fatal("expected error when Hermes fails, got nil")
	}
}

func TestReportGenerator_Generate_EmptyResponse(t *testing.T) {
	stub := &stubHermesClient{
		chat: func(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
			return hermes.ChatResponse{Content: "   "}, nil
		},
	}
	gen := NewReportGenerator(stub, "sk-test")

	_, err := gen.Generate(context.Background(), domain.ReportDailyDigest, samplePeriod(), sampleReportData())
	if err == nil {
		t.Fatal("expected error for empty/whitespace-only response, got nil")
	}
}

func TestReportGenerator_Generate_EmptyData_StillProducesMarkdown(t *testing.T) {
	stub := &stubHermesClient{
		chat: func(_ context.Context, req hermes.ChatRequest) (hermes.ChatResponse, error) {
			prompt := req.Messages[0].Content
			if !strings.Contains(prompt, "Belum ada data pipeline") {
				t.Error("prompt should mention empty pipeline")
			}
			if !strings.Contains(prompt, "Belum ada tender berskor") {
				t.Error("prompt should mention no scored tenders")
			}
			return hermes.ChatResponse{Content: "# Weekly Pipeline Report\n\n## Ringkasan\nBelum ada data.\n\n## Tabel Pipeline\n-\n\n## Prospek Prioritas\n-\n\n## Insight AI\n-"}, nil
		},
	}
	gen := NewReportGenerator(stub, "sk-test")

	md, err := gen.Generate(context.Background(), domain.ReportWeeklyPipeline, samplePeriod(), ReportData{})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if md == "" {
		t.Error("expected non-empty markdown even for empty data")
	}
}
