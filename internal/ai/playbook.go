// Package ai — playbook.go implements the Playbook Generator (EP-14):
// build a prompt from a tender/prospect + Company Profile (reusing the same
// ScoreInput normalization as scoring.go) and call Hermes GenerateJSON for a
// structured 7-section playbook.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// PlaybookContent is the schema hint + unmarshal target passed to
// hermes.GenerateJSON — the 7 structured sections a playbook always has
// (Design §4.10: Ringkasan/Value Prop/Stakeholders/Strategi/Timeline/Risiko/
// Next Actions).
// PlaybookTimelineItem adalah satu aktivitas pada rencana kerja bergaya
// Gantt: mulai hari ke-N (relatif dari kickoff), berlangsung M hari.
type PlaybookTimelineItem struct {
	Activity     string `json:"activity"`
	StartDay     int    `json:"start_day"`
	DurationDays int    `json:"duration_days"`
}

type PlaybookContent struct {
	Summary           string   `json:"summary"`
	ValueProp         string   `json:"value_prop"`
	Stakeholders      []string `json:"stakeholders"`
	StrategyChecklist []string `json:"strategy_checklist"`
	Timeline          []string `json:"timeline"`
	Risks             []string `json:"risks"`
	NextActions       []string `json:"next_actions"`
	// TimelinePlan melengkapi Timeline (list bebas) dengan rencana kerja
	// terstruktur untuk render Gantt — playbook lama tanpa field ini tetap
	// valid (FE fallback ke list).
	TimelinePlan []PlaybookTimelineItem `json:"timeline_plan"`
}

// buildPlaybookPrompt assembles the playbook prompt (P-8): Bahasa Indonesia,
// Company Profile + target data (via the same ScoreInput shape scoring.go
// builds), ending with a strict "balas HANYA JSON" instruction. Reuses
// ScoreInput/buildScoringPrompt's section style deliberately so both prompts
// read consistently to whoever reviews Hermes output.
func buildPlaybookPrompt(in ScoreInput) string {
	var b strings.Builder

	b.WriteString("Kamu adalah asisten strategi sales B2B. Buat PLAYBOOK terstruktur untuk memenangkan peluang berikut, ")
	b.WriteString("berdasarkan profil perusahaan dan data peluang di bawah.\n\n")

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

	b.WriteString("\nHasilkan playbook dengan 7 bagian: ringkasan peluang, value proposition yang relevan, ")
	b.WriteString("daftar stakeholder kunci yang perlu didekati, checklist strategi memenangkan peluang ini, ")
	b.WriteString("timeline langkah-langkah, risiko yang perlu diwaspadai, dan next actions konkret.\n\n")

	b.WriteString("Untuk timeline_plan: susun rencana kerja bergaya Gantt — tiap aktivitas punya start_day (hari ke-N dari kickoff, mulai 0) dan duration_days; aktivitas boleh paralel.\n")
	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(playbookSchemaJSON)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	return b.String()
}

// PlaybookGenerator runs the playbook prompt through Hermes GenerateJSON.
type PlaybookGenerator struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewPlaybookGenerator(hc hermes.Client, sk hermes.SessionKey) *PlaybookGenerator {
	return &PlaybookGenerator{hc: hc, sk: sk}
}

// Generate builds the prompt for in and calls GenerateJSON. On any AI
// failure (Hermes down, timeout, invalid JSON after GenerateJSON's own
// retry) it returns the error unchanged — callers must not persist a
// partial playbook (PRD §8: "gagal AI → pesan ramah, data utuh").
func (g *PlaybookGenerator) Generate(ctx context.Context, in ScoreInput) (*PlaybookContent, error) {
	prompt := buildPlaybookPrompt(in)

	var content PlaybookContent
	if _, err := g.hc.GenerateJSON(ctx, prompt, &content, g.sk); err != nil {
		return nil, fmt.Errorf("ai.PlaybookGenerator.Generate: %w", err)
	}

	return &content, nil
}

// playbookSchemaJSON is PlaybookContent's shape as a prompt instruction —
// shared by all three generation paths so output selalu terstandarisasi.
const playbookSchemaJSON = `{"summary": "...", "value_prop": "...", "stakeholders": ["..."], ` +
	`"strategy_checklist": ["..."], "timeline": ["..."], "risks": ["..."], "next_actions": ["..."], ` +
	`"timeline_plan": [{"activity": "...", "start_day": 0, "duration_days": 1}]}`

// GenerateFromDocument menyusun playbook dari DOKUMEN yang diunggah user
// (proposal lama, playbook existing, notulen strategi, dsb) — dokumen dibaca
// via vision (PDF dirender per halaman oleh bridge), lalu isinya dipetakan
// ke schema playbook baku yang sama dengan Generate.
func (g *PlaybookGenerator) GenerateFromDocument(ctx context.Context, in ScoreInput, pdfBytes []byte, filename string) (*PlaybookContent, error) {
	de, ok := g.hc.(hermes.DocumentExtractor)
	if !ok {
		return nil, fmt.Errorf("ai.PlaybookGenerator.GenerateFromDocument: hermes client tidak mendukung ekstraksi dokumen")
	}

	var b strings.Builder
	b.WriteString("Kamu adalah asisten strategi sales B2B. Baca SELURUH dokumen terlampir ")
	b.WriteString("(bisa berupa proposal lama, playbook existing, atau materi strategi) dan susun ulang isinya menjadi PLAYBOOK terstruktur ")
	b.WriteString("untuk peluang di bawah. Prioritaskan informasi dari dokumen; lengkapi seperlunya dari profil perusahaan. JANGAN mengarang.\n\n")

	if in.Profile != nil {
		p := in.Profile.Profile
		fmt.Fprintf(&b, "## Profil Perusahaan\n- Nama: %s\n", p.CompanyName)
		if len(p.ServiceCategories) > 0 {
			fmt.Fprintf(&b, "- Layanan: %s\n", strings.Join(p.ServiceCategories, ", "))
		}
	}
	fmt.Fprintf(&b, "\n## Peluang Target\n- Tipe: %s\n- Judul/Nama: %s\n", in.TargetType, in.Title)
	if in.Buyer != "" {
		fmt.Fprintf(&b, "- Buyer: %s\n", in.Buyer)
	}

	b.WriteString("\nBalas HANYA JSON dengan schema persis: ")
	b.WriteString(playbookSchemaJSON)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	var content PlaybookContent
	if _, err := de.GenerateJSONFromDocument(ctx, b.String(), filename, pdfBytes, &content, g.sk); err != nil {
		return nil, fmt.Errorf("ai.PlaybookGenerator.GenerateFromDocument: %w", err)
	}
	return &content, nil
}

// Refine merevisi playbook yang sudah ada berdasarkan instruksi bebas dari
// user ("tambahkan mitigasi risiko X", "persingkat timeline", dst) — bagian
// yang tidak disinggung instruksi dipertahankan apa adanya, dan output tetap
// pada schema baku yang sama.
func (g *PlaybookGenerator) Refine(ctx context.Context, current *PlaybookContent, instruction string) (*PlaybookContent, error) {
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("ai.PlaybookGenerator.Refine: marshal current: %w", err)
	}

	var b strings.Builder
	b.WriteString("Kamu adalah asisten strategi sales B2B. Berikut playbook SAAT INI dalam JSON:\n\n")
	b.Write(currentJSON)
	b.WriteString("\n\nRevisi playbook tersebut sesuai instruksi user berikut. ")
	b.WriteString("PERTAHANKAN bagian yang tidak disinggung instruksi apa adanya — hanya ubah yang diminta.\n\n")
	fmt.Fprintf(&b, "## Instruksi User\n%s\n", instruction)
	b.WriteString("\nBalas HANYA JSON lengkap (semua field) dengan schema persis: ")
	b.WriteString(playbookSchemaJSON)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	var content PlaybookContent
	if _, err := g.hc.GenerateJSON(ctx, b.String(), &content, g.sk); err != nil {
		return nil, fmt.Errorf("ai.PlaybookGenerator.Refine: %w", err)
	}
	return &content, nil
}

// GenerateCustomTopic menyusun playbook mandiri dari TOPIK bebas user
// (bukan tender/prospect/event tertentu) — mis. "strategi masuk sektor
// kesehatan" — tetap pada schema baku yang sama.
func (g *PlaybookGenerator) GenerateCustomTopic(ctx context.Context, topic string, profile *domain.ProfileAggregate) (*PlaybookContent, error) {
	var b strings.Builder
	b.WriteString("Kamu adalah asisten strategi sales B2B. Buat PLAYBOOK terstruktur untuk topik/inisiatif berikut, ")
	b.WriteString("berdasarkan profil perusahaan di bawah.\n\n")

	if profile != nil {
		p := profile.Profile
		fmt.Fprintf(&b, "## Profil Perusahaan\n- Nama: %s\n", p.CompanyName)
		if len(p.ServiceCategories) > 0 {
			fmt.Fprintf(&b, "- Layanan: %s\n", strings.Join(p.ServiceCategories, ", "))
		}
		if len(p.Products) > 0 {
			fmt.Fprintf(&b, "- Produk: %s\n", strings.Join(p.Products, ", "))
		}
	}

	fmt.Fprintf(&b, "\n## Topik Playbook\n%s\n", topic)
	b.WriteString("\nUntuk timeline_plan: susun rencana kerja bergaya Gantt — tiap aktivitas punya start_day (hari ke-N dari kickoff, mulai 0) dan duration_days; aktivitas boleh paralel.\n")
	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(playbookSchemaJSON)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	var content PlaybookContent
	if _, err := g.hc.GenerateJSON(ctx, b.String(), &content, g.sk); err != nil {
		return nil, fmt.Errorf("ai.PlaybookGenerator.GenerateCustomTopic: %w", err)
	}
	return &content, nil
}
