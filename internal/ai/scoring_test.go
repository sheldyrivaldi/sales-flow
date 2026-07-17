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

// stubHermesClient is a minimal hermes.Client stub for unit testing without a
// real Hermes bridge. Mirrors internal/service/keyword_service_test.go's
// stubHermesClient. generateJSON is meaningful for scoring tests; chat is
// additionally set by discovery tests that exercise hermesCrawler.Discover's
// full Chat->GenerateJSON round trip.
type stubHermesClient struct {
	generateJSON func(ctx context.Context, prompt string, schema any, sk hermes.SessionKey) (json.RawMessage, error)
	chat         func(ctx context.Context, req hermes.ChatRequest) (hermes.ChatResponse, error)
}

var _ hermes.Client = (*stubHermesClient)(nil)

func (s *stubHermesClient) Chat(ctx context.Context, req hermes.ChatRequest) (hermes.ChatResponse, error) {
	if s.chat != nil {
		return s.chat(ctx, req)
	}
	return hermes.ChatResponse{}, errors.New("not implemented")
}

func (s *stubHermesClient) ChatStream(_ context.Context, _ hermes.ChatRequest) (<-chan hermes.Chunk, error) {
	return nil, errors.New("not implemented")
}

func (s *stubHermesClient) GenerateJSON(ctx context.Context, prompt string, schema any, sk hermes.SessionKey) (json.RawMessage, error) {
	return s.generateJSON(ctx, prompt, schema, sk)
}

func (s *stubHermesClient) Health(_ context.Context) (hermes.Capabilities, error) {
	return hermes.Capabilities{}, errors.New("not implemented")
}

func (s *stubHermesClient) Configure(_ context.Context, _ hermes.ProviderConfig) error {
	return errors.New("not implemented")
}

func (s *stubHermesClient) ResetMemory(_ context.Context, _ hermes.SessionKey) error {
	return errors.New("not implemented")
}

func sampleProfile() *domain.ProfileAggregate {
	oneLiner := "Kami membangun aplikasi web & integrasi sistem."
	return &domain.ProfileAggregate{
		Profile: domain.CompanyProfile{
			CompanyName:       "PT Contoh Teknologi",
			OneLiner:          &oneLiner,
			ServiceCategories: []string{"Web App", "Integrasi Sistem"},
			TechStack:         []string{"Go", "React"},
		},
		Target: &domain.TargetCriteria{
			Countries:  []string{"Indonesia"},
			Industries: []string{"Pemerintahan", "BUMN"},
			Currency:   "IDR",
		},
		NoGo: &domain.NoGoRule{
			PresetFlags: []string{"Membutuhkan TKDN 100%"},
			Custom:      []string{"Nilai proyek di bawah 50 juta"},
		},
	}
}

func TestBuildScoringPrompt_ContainsAllRubricDimensions(t *testing.T) {
	in := ScoreInputFromTender(domain.Tender{
		Title:     "Pengadaan Sistem Informasi",
		BuyerName: strPtr("Kementerian Contoh"),
	}, sampleProfile())

	prompt := buildScoringPrompt(in)

	for _, r := range scoringRubric {
		if !strings.Contains(prompt, r.Dimension) {
			t.Errorf("prompt missing rubric dimension %q", r.Dimension)
		}
	}
}

func TestBuildScoringPrompt_ContainsTargetAndNoGoAndWeights(t *testing.T) {
	in := ScoreInputFromTender(domain.Tender{
		Title:     "Pengadaan Sistem Informasi",
		BuyerName: strPtr("Kementerian Contoh"),
	}, sampleProfile())

	prompt := buildScoringPrompt(in)

	if !strings.Contains(prompt, "Pengadaan Sistem Informasi") {
		t.Error("prompt missing tender title")
	}
	if !strings.Contains(prompt, "Kementerian Contoh") {
		t.Error("prompt missing buyer name")
	}
	if !strings.Contains(prompt, "Membutuhkan TKDN 100%") {
		t.Error("prompt missing no-go preset flag")
	}
	if !strings.Contains(prompt, "Nilai proyek di bawah 50 juta") {
		t.Error("prompt missing no-go custom rule")
	}
	if !strings.Contains(prompt, "20%") {
		t.Error("prompt missing Capability fit weight (20%)")
	}
}

func TestBuildScoringPrompt_UsesConfiguredWeights(t *testing.T) {
	profile := sampleProfile()
	profile.ScoringConfig = &domain.ScoringConfig{
		WeightCapabilityFit:             42,
		WeightPortfolioMatch:            15,
		WeightCommercialAttractiveness:  15,
		WeightEligibilityFit:            15,
		WeightDeadlineFeasibility:       10,
		WeightStrategicAccountValue:     1,
		WeightDeliveryRisk:              1,
		WeightCompetitionWinProbability: 1,
	}

	in := ScoreInputFromTender(domain.Tender{Title: "T", BuyerName: strPtr("B")}, profile)
	prompt := buildScoringPrompt(in)

	if !strings.Contains(prompt, "Capability fit (cocok kapabilitas utama) — bobot 42%") {
		t.Errorf("prompt did not use configured Capability fit weight (42%%):\n%s", prompt)
	}
	if strings.Contains(prompt, "bobot 20%") {
		t.Error("prompt still contains the default 20% weight instead of the configured one")
	}
}

func TestScoreInputFromProspect(t *testing.T) {
	in := ScoreInputFromProspect(domain.Prospect{
		Name:    "Prospek A",
		Company: strPtr("PT Prospek"),
		Stage:   domain.ProspectStageQualified,
	}, sampleProfile())

	if in.TargetType != ScoreTargetProspect {
		t.Errorf("TargetType = %q, want %q", in.TargetType, ScoreTargetProspect)
	}
	if in.Title != "Prospek A" {
		t.Errorf("Title = %q, want %q", in.Title, "Prospek A")
	}
	if in.StatusOrStage != string(domain.ProspectStageQualified) {
		t.Errorf("StatusOrStage = %q, want %q", in.StatusOrStage, domain.ProspectStageQualified)
	}
}

func TestScorer_Score_Success(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"fit_score":130,"recommended_action":"PURSUE","confidence":1.5,` +
				`"reasoning":"cocok","evidence":[{"dimension":"Capability fit","verdict":"pass","note":"ok"}],` +
				`"risk_flags":["deadline ketat"],"no_go_triggered":false,"need_partner":false}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema failed: %v", err)
			}
			return raw, nil
		},
	}

	scorer := NewScorer(stub, "sk-test")
	res, err := scorer.Score(context.Background(), ScoreInputFromTender(domain.Tender{Title: "T"}, nil))
	if err != nil {
		t.Fatalf("Score error: %v", err)
	}

	if res.FitScore != 100 {
		t.Errorf("FitScore = %d, want clamped to 100", res.FitScore)
	}
	if res.Confidence != 1 {
		t.Errorf("Confidence = %v, want clamped to 1", res.Confidence)
	}
	if len(res.Evidence) != 1 {
		t.Errorf("Evidence = %v, want 1 item", res.Evidence)
	}
	if len(res.RiskFlags) != 1 {
		t.Errorf("RiskFlags = %v, want 1 item", res.RiskFlags)
	}
}

func TestScorer_Score_HermesError(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}

	scorer := NewScorer(stub, "sk-test")
	_, err := scorer.Score(context.Background(), ScoreInputFromTender(domain.Tender{Title: "T"}, nil))
	if err == nil {
		t.Fatal("Score should return an error when Hermes fails, got nil")
	}
}

func strPtr(s string) *string { return &s }
