// Package ai — profile_extract.go implements Company Profile PDF ingest
// (EP-13 ST-13.2): the raw PDF is sent to Hermes, which renders it to
// images and reads it via vision — not a local Go-side text extraction pass.
//
// This replaced an earlier text-extraction approach (github.com/ledongthuc/pdf
// + embedding the extracted text in the prompt), which was lossy for
// table-heavy documents (a Company Profile / RFI is mostly tables) and
// couldn't read scanned/image-only PDFs at all. Sending the actual file lets
// the model read the document the way a person would, tables included, and
// removes the "no extractable text layer" failure mode entirely.
package ai

import (
	"context"
	"fmt"
	"strings"

	"salespilot/internal/hermes"
)

// ProfileDraftTarget is the target-criteria slice of a ProfileDraft — grouped
// separately (rather than flattened) because it mirrors
// dto.TargetCriteriaRequest's own nested shape.
type ProfileDraftTarget struct {
	Countries          []string `json:"countries"`
	Industries         []string `json:"industries"`
	ValueMin           *float64 `json:"value_min"`
	ValueIdeal         *float64 `json:"value_ideal"`
	ValueMax           *float64 `json:"value_max"`
	DeadlineMinDays    *int     `json:"deadline_min_days"`
	ProcurementTypes   []string `json:"procurement_types"`
	BuyerSizeNote      string   `json:"buyer_size_note"`
	DocumentLanguages  []string `json:"document_languages"`
	WorkModel          string   `json:"work_model"`
	OnsiteLimitNote    string   `json:"onsite_limit_note"`
	DecisionMakerRoles []string `json:"decision_maker_roles"`
}

// ProfileDraft is the AI-extracted subset of Company Profile fields offered
// for review before the user confirms and saves them via PUT /api/profile
// (EP-13 ST-13.2/13.3) — field names deliberately mirror
// dto.ProfileUpdateRequest so the FE can merge this directly into the edit
// form. It is never persisted directly; only PUT /api/profile persists.
// NoGo is deliberately extracted as free-text "nogo_custom" only — the AI
// never maps findings onto the fixed NOGO_PRESET_FLAGS, to avoid a
// plausible-sounding mismatch silently ticking a preset the document didn't
// actually support.
type ProfileDraft struct {
	CompanyName       string             `json:"company_name"`
	OneLiner          string             `json:"one_liner"`
	ServiceCategories []string           `json:"service_categories"`
	TechStack         []string           `json:"tech_stack"`
	Products          []string           `json:"products"`
	Vision            string             `json:"vision"`
	Mission           string             `json:"mission"`
	PortfolioRefs     []string           `json:"portfolio_refs"`
	Keywords          []string           `json:"keywords"`
	NegativeKeywords  []string           `json:"negative_keywords"`
	NogoCustom        []string           `json:"nogo_custom"`
	Target            ProfileDraftTarget `json:"target"`
}

// buildExtractPrompt assembles the Company Profile field-extraction prompt
// (P-8): Bahasa Indonesia, ends with a strict "balas HANYA JSON" instruction
// matching ProfileDraft's schema exactly. The document itself is attached
// separately (as images) by GenerateJSONFromDocument — this prompt is pure
// instruction, no document text embedded in it.
func buildExtractPrompt() string {
	var b strings.Builder
	b.WriteString("Kamu membantu mengisi profil perusahaan (Company Profile) dari dokumen PDF terlampir ")
	b.WriteString("(company profile / capability deck / RFI kebutuhan AI agent procurement). ")
	b.WriteString("Baca SELURUH halaman dokumen — termasuk tabel — dan ekstrak field yang relevan. ")
	b.WriteString("Bila sebuah field tidak ditemukan di dokumen, kembalikan string kosong atau array kosong — JANGAN mengarang. ")
	b.WriteString("Untuk nogo_custom: HANYA salin kondisi larangan/no-go yang eksplisit ada di dokumen sebagai teks bebas; JANGAN mencoba mencocokkan ke daftar preset apa pun.\n\n")
	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"company_name": "...", "one_liner": "...", "service_categories": ["..."], "tech_stack": ["..."], ` +
		`"products": ["..."], "vision": "...", "mission": "...", "portfolio_refs": ["..."], ` +
		`"keywords": ["..."], "negative_keywords": ["..."], "nogo_custom": ["..."], ` +
		`"target": {"countries": ["..."], "industries": ["..."], "value_min": 0, "value_ideal": 0, "value_max": 0, ` +
		`"deadline_min_days": 0, "procurement_types": ["..."], "buyer_size_note": "...", ` +
		`"document_languages": ["..."], "work_model": "...", "onsite_limit_note": "...", "decision_maker_roles": ["..."]}}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")
	return b.String()
}

// Extractor turns a PDF into a ProfileDraft via Hermes's vision-based
// document reading (hermes.DocumentExtractor).
type Extractor struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewExtractor(hc hermes.Client, sk hermes.SessionKey) *Extractor {
	return &Extractor{hc: hc, sk: sk}
}

// Extract sends pdfBytes to Hermes for vision-based field extraction. On any
// AI failure (Hermes down, timeout, invalid JSON after GenerateJSON's own
// retry) it returns the error unchanged — the caller (ProfileService) treats
// this as a non-fatal "degrade to manual entry" signal, not a failed upload
// (PRD §8: "gagal AI → pesan ramah, data utuh").
func (e *Extractor) Extract(ctx context.Context, pdfBytes []byte, filename string) (*ProfileDraft, error) {
	de, ok := e.hc.(hermes.DocumentExtractor)
	if !ok {
		return nil, fmt.Errorf("ai.Extract: hermes client tidak mendukung ekstraksi dokumen (DocumentExtractor)")
	}

	prompt := buildExtractPrompt()

	var draft ProfileDraft
	if _, err := de.GenerateJSONFromDocument(ctx, prompt, filename, pdfBytes, &draft, e.sk); err != nil {
		return nil, fmt.Errorf("ai.Extract: %w", err)
	}

	return &draft, nil
}
