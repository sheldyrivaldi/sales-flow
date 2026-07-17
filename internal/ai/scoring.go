// Package ai implements the AI Scoring & Recommendation feature (EP-10):
// build a scoring prompt from a tender/prospect + Company Profile + rubrik
// §8, call Hermes GenerateJSON for a structured result, then map the result
// to a deterministic recommended_action (see recommend.go).
package ai

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// scoringRubric is rubrik §8 (PRD.md) — 8 dimensions + default weights,
// verbatim. Used as-is whenever a profile has no ScoringConfig (never
// configured by Ops); resolveRubric substitutes the configured weights when
// one exists. Order and wording feed directly into the prompt; do not
// reorder/reword without updating PRD.md in lockstep.
var scoringRubric = []struct {
	Dimension string
	Weight    int
}{
	{"Capability fit (cocok kapabilitas utama)", 20},
	{"Portfolio match (ada pengalaman sejenis)", 15},
	{"Commercial attractiveness (nilai/margin)", 15},
	{"Eligibility fit (syarat legal/sertifikasi/pengalaman)", 15},
	{"Deadline feasibility (cukup waktu proposal)", 10},
	{"Strategic account value (buyer strategis)", 10},
	{"Delivery risk (scope/onsite/dependency)", 10},
	{"Competition / win probability", 5},
}

// resolveRubric returns cfg's configured weights (RFI §8: Ops-adjustable) in
// the same fixed dimension order as scoringRubric, or scoringRubric itself
// unchanged when cfg is nil — so a profile that has never touched the
// Scoring card scores identically to before ScoringConfig existed.
func resolveRubric(cfg *domain.ScoringConfig) []struct {
	Dimension string
	Weight    int
} {
	if cfg == nil {
		return scoringRubric
	}
	weights := []int{
		cfg.WeightCapabilityFit,
		cfg.WeightPortfolioMatch,
		cfg.WeightCommercialAttractiveness,
		cfg.WeightEligibilityFit,
		cfg.WeightDeadlineFeasibility,
		cfg.WeightStrategicAccountValue,
		cfg.WeightDeliveryRisk,
		cfg.WeightCompetitionWinProbability,
	}
	out := make([]struct {
		Dimension string
		Weight    int
	}, len(scoringRubric))
	for i, r := range scoringRubric {
		out[i] = struct {
			Dimension string
			Weight    int
		}{Dimension: r.Dimension, Weight: weights[i]}
	}
	return out
}

// ScoreTargetType identifies whether a ScoreInput was built from a tender or
// a prospect. Both are scored through the same prompt/rubric shape.
type ScoreTargetType string

const (
	ScoreTargetTender   ScoreTargetType = "tender"
	ScoreTargetProspect ScoreTargetType = "prospect"
	// ScoreTargetEvent/ScoreTargetCustom dipakai playbook (bukan scoring):
	// playbook bisa dibuat untuk event tertentu atau topik custom mandiri.
	ScoreTargetEvent  ScoreTargetType = "event"
	ScoreTargetCustom ScoreTargetType = "custom"
)

// ScoreInput is the normalized shape both domain.Tender and domain.Prospect
// are reduced to before prompt building, so buildScoringPrompt only needs
// one code path regardless of target type.
type ScoreInput struct {
	TargetType      ScoreTargetType
	Title           string
	Buyer           string
	Country         string
	Industry        string
	Value           *float64
	Currency        string
	Deadline        *time.Time
	ServiceCategory string
	ScopeSummary    string
	Eligibility     string
	Technical       string
	StatusOrStage   string
	Profile         *domain.ProfileAggregate
}

// ScoreInputFromTender normalizes a domain.Tender into a ScoreInput.
func ScoreInputFromTender(t domain.Tender, profile *domain.ProfileAggregate) ScoreInput {
	return ScoreInput{
		TargetType:      ScoreTargetTender,
		Title:           t.Title,
		Buyer:           derefStr(t.BuyerName),
		Country:         derefStr(t.BuyerCountry),
		Industry:        derefStr(t.BuyerIndustry),
		Value:           t.ValueEstimate,
		Currency:        t.Currency,
		Deadline:        t.SubmissionDeadline,
		ServiceCategory: derefStr(t.ServiceCategory),
		ScopeSummary:    derefStr(t.ScopeSummary),
		Eligibility:     derefStr(t.EligibilityRequirements),
		Technical:       derefStr(t.TechnicalRequirements),
		StatusOrStage:   string(t.Status),
		Profile:         profile,
	}
}

// ScoreInputFromProspect normalizes a domain.Prospect into a ScoreInput.
// Prospect has no currency/deadline/scope columns — those stay empty and
// the prompt naturally omits them for prospect targets.
func ScoreInputFromProspect(p domain.Prospect, profile *domain.ProfileAggregate) ScoreInput {
	return ScoreInput{
		TargetType:    ScoreTargetProspect,
		Title:         p.Name,
		Buyer:         derefStr(p.Company),
		Value:         p.EstValue,
		StatusOrStage: string(p.Stage),
		Profile:       profile,
	}
}

// ScoreInputFromEvent normalizes a domain.Event into a ScoreInput — dipakai
// generator playbook untuk target event (scoring sendiri tidak menerima
// event).
func ScoreInputFromEvent(e domain.Event, profile *domain.ProfileAggregate) ScoreInput {
	in := ScoreInput{
		TargetType:    ScoreTargetEvent,
		Title:         e.Name,
		StatusOrStage: string(e.Status),
		Profile:       profile,
	}
	if e.Organizer != nil {
		in.Buyer = *e.Organizer
	}
	if e.Notes != nil {
		in.ScopeSummary = *e.Notes
	}
	return in
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// buildScoringPrompt assembles the full scoring prompt: role, Company
// Profile context (capability/target/no-go), rubrik §8, target data, and
// the JSON output instruction. Mirrors the style of buildKeywordPrompt
// (internal/service/keyword_service.go): Bahasa Indonesia, ends with a
// strict "balas HANYA JSON" instruction.
func buildScoringPrompt(in ScoreInput) string {
	var b strings.Builder

	b.WriteString("Kamu adalah analis tender/peluang bisnis untuk tim sales B2B. ")
	b.WriteString("Nilai kelayakan peluang berikut berdasarkan profil perusahaan, kriteria target, aturan no-go, dan rubrik scoring di bawah.\n\n")

	if in.Profile != nil {
		p := in.Profile.Profile
		b.WriteString("## Profil Perusahaan\n")
		fmt.Fprintf(&b, "- Nama: %s\n", p.CompanyName)
		if p.OneLiner != nil && *p.OneLiner != "" {
			fmt.Fprintf(&b, "- Deskripsi singkat: %s\n", *p.OneLiner)
		}
		if len(p.ServiceCategories) > 0 {
			fmt.Fprintf(&b, "- Kategori layanan: %s\n", strings.Join(p.ServiceCategories, ", "))
		}
		if len(p.TechStack) > 0 {
			fmt.Fprintf(&b, "- Tech stack: %s\n", strings.Join(p.TechStack, ", "))
		}

		if t := in.Profile.Target; t != nil {
			b.WriteString("\n## Kriteria Target\n")
			if len(t.Countries) > 0 {
				fmt.Fprintf(&b, "- Negara target: %s\n", strings.Join(t.Countries, ", "))
			}
			if len(t.Industries) > 0 {
				fmt.Fprintf(&b, "- Industri target: %s\n", strings.Join(t.Industries, ", "))
			}
			if t.ValueMin != nil || t.ValueIdeal != nil || t.ValueMax != nil {
				fmt.Fprintf(&b, "- Rentang nilai ideal (%s): min %s, ideal %s, maks %s\n",
					t.Currency, formatFloatPtr(t.ValueMin), formatFloatPtr(t.ValueIdeal), formatFloatPtr(t.ValueMax))
			}
			if t.DeadlineMinDays != nil {
				fmt.Fprintf(&b, "- Minimal hari sebelum deadline agar layak dikejar: %d\n", *t.DeadlineMinDays)
			}
			if len(t.ProcurementTypes) > 0 {
				fmt.Fprintf(&b, "- Jenis pengadaan yang diterima: %s\n", strings.Join(t.ProcurementTypes, ", "))
			}
		}

		if ng := in.Profile.NoGo; ng != nil && (len(ng.PresetFlags) > 0 || len(ng.Custom) > 0) {
			b.WriteString("\n## Aturan No-Go (WAJIB diperiksa)\n")
			b.WriteString("Bila peluang melanggar salah satu kondisi berikut, tandai no_go_triggered=true di output. ")
			b.WriteString("Bila pelanggaran berupa gap yang masih bisa ditutup partner (mis. sertifikasi/lokasi/kapasitas), tandai juga need_partner=true.\n")
			for _, f := range ng.PresetFlags {
				fmt.Fprintf(&b, "- %s\n", f)
			}
			for _, c := range ng.Custom {
				fmt.Fprintf(&b, "- %s\n", c)
			}
		}
	}

	var scoringCfg *domain.ScoringConfig
	if in.Profile != nil {
		scoringCfg = in.Profile.ScoringConfig
	}

	b.WriteString("\n## Rubrik Scoring (8 dimensi, bobot total 100%)\n")
	b.WriteString("Nilai TIAP dimensi berikut dan berikan evidence singkat (pass/warn/fail) untuk masing-masing:\n")
	for _, r := range resolveRubric(scoringCfg) {
		fmt.Fprintf(&b, "- %s — bobot %d%%\n", r.Dimension, r.Weight)
	}

	b.WriteString("\n## Data Peluang\n")
	fmt.Fprintf(&b, "- Tipe: %s\n", in.TargetType)
	fmt.Fprintf(&b, "- Judul/Nama: %s\n", in.Title)
	if in.Buyer != "" {
		fmt.Fprintf(&b, "- Buyer/Perusahaan: %s\n", in.Buyer)
	}
	if in.Country != "" {
		fmt.Fprintf(&b, "- Negara: %s\n", in.Country)
	}
	if in.Industry != "" {
		fmt.Fprintf(&b, "- Industri: %s\n", in.Industry)
	}
	if in.Value != nil {
		fmt.Fprintf(&b, "- Nilai estimasi: %s %s\n", in.Currency, formatFloatPtr(in.Value))
	}
	if in.Deadline != nil {
		fmt.Fprintf(&b, "- Deadline submission: %s\n", in.Deadline.Format("2006-01-02"))
	}
	if in.ServiceCategory != "" {
		fmt.Fprintf(&b, "- Kategori layanan: %s\n", in.ServiceCategory)
	}
	if in.ScopeSummary != "" {
		fmt.Fprintf(&b, "- Ringkasan scope: %s\n", in.ScopeSummary)
	}
	if in.Eligibility != "" {
		fmt.Fprintf(&b, "- Syarat eligibilitas: %s\n", in.Eligibility)
	}
	if in.Technical != "" {
		fmt.Fprintf(&b, "- Syarat teknis: %s\n", in.Technical)
	}
	if in.StatusOrStage != "" {
		fmt.Fprintf(&b, "- Status/stage saat ini: %s\n", in.StatusOrStage)
	}

	b.WriteString("\nBalas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"fit_score": 0-100, "recommended_action": "PURSUE|REVIEW|WATCHLIST|REJECT|NEED_PARTNER", ` +
		`"confidence": 0.0-1.0, "reasoning": "...", ` +
		`"evidence": [{"dimension": "...", "verdict": "pass|warn|fail", "note": "..."}], ` +
		`"risk_flags": ["..."], "no_go_triggered": true|false, "need_partner": true|false}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	return b.String()
}

func formatFloatPtr(f *float64) string {
	if f == nil {
		return "-"
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

// EvidenceItem is one rubrik-dimension assessment inside a ScoreResult.
type EvidenceItem struct {
	Dimension string `json:"dimension"`
	Verdict   string `json:"verdict"` // "pass" | "warn" | "fail"
	Note      string `json:"note"`
}

// ScoreResult is the schema hint + unmarshal target passed to
// hermes.GenerateJSON. RecommendedAction here is the LLM's own suggestion —
// the value actually persisted is recomputed deterministically by
// RecommendAction (recommend.go) from FitScore/NoGoTriggered/NeedPartner.
type ScoreResult struct {
	FitScore          int            `json:"fit_score"`
	RecommendedAction string         `json:"recommended_action"`
	Confidence        float64        `json:"confidence"`
	Reasoning         string         `json:"reasoning"`
	Evidence          []EvidenceItem `json:"evidence"`
	RiskFlags         []string       `json:"risk_flags"`
	NoGoTriggered     bool           `json:"no_go_triggered"`
	NeedPartner       bool           `json:"need_partner"`
}

// ModelLabel identifies the AI subsystem that produced a score, recorded on
// each prospect_score row. Hermes-bridge chooses the actual underlying LLM
// itself (see hermes.defaultModel) — Go has no visibility into which model
// literally answered a given request, so this is a fixed subsystem label,
// not a precise model identifier.
const ModelLabel = "hermes"

// Scorer runs the scoring prompt through Hermes GenerateJSON.
type Scorer struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewScorer(hc hermes.Client, sk hermes.SessionKey) *Scorer {
	return &Scorer{hc: hc, sk: sk}
}

// Score builds the prompt for in and calls GenerateJSON. On any AI failure
// (Hermes down, timeout, invalid JSON after GenerateJSON's own retry) it
// returns the error unchanged — callers must not persist a partial score
// (PRD §8: "gagal AI → pesan ramah, data utuh").
func (s *Scorer) Score(ctx context.Context, in ScoreInput) (*ScoreResult, error) {
	prompt := buildScoringPrompt(in)

	var res ScoreResult
	if _, err := s.hc.GenerateJSON(ctx, prompt, &res, s.sk); err != nil {
		return nil, fmt.Errorf("ai.Score: %w", err)
	}

	if res.FitScore < 0 {
		res.FitScore = 0
	}
	if res.FitScore > 100 {
		res.FitScore = 100
	}
	if res.Confidence < 0 {
		res.Confidence = 0
	}
	if res.Confidence > 1 {
		res.Confidence = 1
	}

	return &res, nil
}
