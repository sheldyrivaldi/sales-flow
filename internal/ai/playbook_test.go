package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

func TestPlaybookGenerator_Generate_Success(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, prompt string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			for _, want := range []string{"PLAYBOOK", "PT Contoh Teknologi", "Tender Pengadaan Aplikasi"} {
				if !strings.Contains(prompt, want) {
					t.Errorf("prompt missing expected content %q", want)
				}
			}
			raw := []byte(`{
				"summary": "Peluang bagus untuk PT Contoh",
				"value_prop": "Solusi terintegrasi hemat biaya",
				"stakeholders": ["Kepala Dinas IT", "Tim Procurement"],
				"strategy_checklist": ["Siapkan proposal teknis", "Jadwalkan demo"],
				"timeline": ["Minggu 1: kickoff", "Minggu 2: submit proposal"],
				"risks": ["Kompetitor lokal kuat"],
				"next_actions": ["Hubungi PIC buyer"]
			}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}

	gen := NewPlaybookGenerator(stub, "sk-test")
	in := ScoreInputFromTender(domain.Tender{
		Title:     "Tender Pengadaan Aplikasi",
		BuyerName: strPtr("Kementerian Contoh"),
	}, sampleProfile())

	content, err := gen.Generate(context.Background(), in)
	if err != nil {
		t.Fatalf("Generate: unexpected error: %v", err)
	}
	if content.Summary == "" {
		t.Error("Summary is empty")
	}
	if content.ValueProp == "" {
		t.Error("ValueProp is empty")
	}
	if len(content.Stakeholders) != 2 {
		t.Errorf("Stakeholders = %v, want 2 items", content.Stakeholders)
	}
	if len(content.StrategyChecklist) != 2 {
		t.Errorf("StrategyChecklist = %v, want 2 items", content.StrategyChecklist)
	}
	if len(content.Timeline) != 2 {
		t.Errorf("Timeline = %v, want 2 items", content.Timeline)
	}
	if len(content.Risks) != 1 {
		t.Errorf("Risks = %v, want 1 item", content.Risks)
	}
	if len(content.NextActions) != 1 {
		t.Errorf("NextActions = %v, want 1 item", content.NextActions)
	}
}

func TestPlaybookGenerator_Generate_HermesError(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes unavailable")
		},
	}

	gen := NewPlaybookGenerator(stub, "sk-test")
	in := ScoreInputFromProspect(domain.Prospect{
		Name:    "Prospek A",
		Company: strPtr("PT Prospek"),
		Stage:   domain.ProspectStageQualified,
	}, sampleProfile())

	_, err := gen.Generate(context.Background(), in)
	if err == nil {
		t.Fatal("expected error when Hermes fails, got nil")
	}
}
