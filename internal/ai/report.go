// Package ai — report.go implements the Report Generator (EP-15): aggregate
// pipeline/activity data (same shape as service.DashboardSummary, EP-11) and
// ask Hermes to produce a markdown report for one of 3 types (Daily
// Opportunity Digest / Weekly Pipeline / Per-peluang).
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// ReportPeriod is the [Start,End] window a report covers.
type ReportPeriod struct {
	Start time.Time
	End   time.Time
}

// ReportData is the live aggregation snapshot fed into the report prompt —
// same shape as service.DashboardSummary (EP-11 dashboard). The caller
// (ReportService) is responsible for querying it; this package stays free
// of repository dependencies, matching how ScoreInput is built externally
// and passed into Scorer/PlaybookGenerator.
type ReportData struct {
	Pipeline            []domain.ProspectStageSummary
	TotalPipelineCount  int64
	TotalPipelineValue  float64
	PriorityTenders     []domain.Tender
	DiscoveryTodayCount int64
}

func reportTitle(t domain.ReportType) string {
	switch t {
	case domain.ReportDailyDigest:
		return "Daily Opportunity Digest"
	case domain.ReportWeeklyPipeline:
		return "Weekly Pipeline Report"
	case domain.ReportPerOpportunity:
		return "Laporan Per-Peluang"
	default:
		return "Laporan"
	}
}

// buildReportPrompt assembles the report prompt (P-8 style, but asking for
// markdown prose instead of a JSON schema): Bahasa Indonesia, period +
// aggregated data, ending with an instruction to use a fixed heading
// structure so the FE viewer can render consistently regardless of type.
func buildReportPrompt(reportType domain.ReportType, period ReportPeriod, data ReportData) string {
	var b strings.Builder

	title := reportTitle(reportType)
	fmt.Fprintf(&b, "Kamu adalah asisten sales B2B. Buat laporan \"%s\" dalam format MARKDOWN untuk periode %s s/d %s.\n\n",
		title, period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"))

	b.WriteString("## Data Pipeline (per stage)\n")
	if len(data.Pipeline) == 0 {
		b.WriteString("Belum ada data pipeline.\n")
	} else {
		for _, p := range data.Pipeline {
			fmt.Fprintf(&b, "- %s: %d prospek, total nilai %.0f\n", p.Stage, p.Count, p.TotalValue)
		}
	}
	fmt.Fprintf(&b, "- Total keseluruhan: %d prospek, nilai %.0f\n", data.TotalPipelineCount, data.TotalPipelineValue)

	b.WriteString("\n## Tender Prioritas (skor tertinggi)\n")
	if len(data.PriorityTenders) == 0 {
		b.WriteString("Belum ada tender berskor.\n")
	} else {
		for _, t := range data.PriorityTenders {
			score := "-"
			if t.FitScore != nil {
				score = fmt.Sprintf("%d", *t.FitScore)
			}
			fmt.Fprintf(&b, "- %s (skor %s)\n", t.Title, score)
		}
	}

	fmt.Fprintf(&b, "\n## Penemuan AI Hari Ini\n- %d tender baru ditemukan AI hari ini\n", data.DiscoveryTodayCount)

	b.WriteString("\nHasilkan laporan markdown dengan struktur heading PERSIS: `# <Judul Laporan>`, ")
	b.WriteString("lalu `## Ringkasan`, `## Tabel Pipeline`, `## Prospek Prioritas`, `## Insight AI`. ")
	b.WriteString("Isi tiap bagian berdasarkan data di atas, Bahasa Indonesia, ringkas dan actionable untuk tim sales. ")
	b.WriteString("Balas HANYA markdown, tanpa code fence, tanpa penjelasan tambahan di luar laporan itu sendiri.")

	return b.String()
}

// ReportGenerator runs the report prompt through Hermes Chat (non-stream).
// Report output is markdown prose, not a fixed JSON schema, so this uses
// Chat rather than GenerateJSON — the counterpart used by Scorer/
// PlaybookGenerator, both of which need structured JSON.
type ReportGenerator struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewReportGenerator(hc hermes.Client, sk hermes.SessionKey) *ReportGenerator {
	return &ReportGenerator{hc: hc, sk: sk}
}

// Generate builds the prompt for reportType/period/data and calls Chat. On
// any AI failure (Hermes down, timeout, empty response) it returns the
// error unchanged — callers must not persist a partial report (PRD §8:
// "gagal AI → pesan ramah, data utuh").
func (g *ReportGenerator) Generate(ctx context.Context, reportType domain.ReportType, period ReportPeriod, data ReportData) (string, error) {
	prompt := buildReportPrompt(reportType, period, data)

	resp, err := g.hc.Chat(ctx, hermes.ChatRequest{
		Messages:   []hermes.Message{{Role: "user", Content: prompt}},
		Stream:     false,
		SessionKey: g.sk,
	})
	if err != nil {
		return "", fmt.Errorf("ai.ReportGenerator.Generate: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return "", fmt.Errorf("ai.ReportGenerator.Generate: respons kosong dari Hermes")
	}

	return content, nil
}
