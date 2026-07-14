// Package ai — profile_extract.go implements PDF text extraction for Company
// Profile ingest (EP-13 ST-13.2).
//
// Library choice (investigated per EP-13 ST-13.2.1): github.com/ledongthuc/pdf
// — pure Go, zero transitive dependencies (verified via `go mod tidy`), so it
// needs no system binary (e.g. poppler's pdftotext) and builds identically on
// native Windows and in the Docker image. It only reads text-layer content;
// scanned/image-only PDFs correctly yield no extractable text (handled below
// as ErrNoExtractableText, not silently returning an empty string to Hermes).
package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"

	"salespilot/internal/hermes"
)

// ErrNoExtractableText is returned when a PDF opens successfully but
// contains no text layer (typically a scanned/image-only document) — the
// caller should surface a friendly "coba isi manual" message, not treat it
// as a hard failure.
var ErrNoExtractableText = errors.New("PDF tidak mengandung teks (kemungkinan hasil scan)")

// ExtractText opens the PDF at path and returns its concatenated plain text
// across all pages, trimmed of leading/trailing whitespace. Returns
// ErrNoExtractableText when the PDF has no text layer, or a wrapped error
// for a corrupt/unreadable/encrypted file.
func ExtractText(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("ai.ExtractText: buka PDF: %w", err)
	}
	defer func() { _ = f.Close() }()

	textReader, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("ai.ExtractText: ekstrak teks: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(textReader); err != nil {
		return "", fmt.Errorf("ai.ExtractText: baca teks: %w", err)
	}

	text := strings.TrimSpace(buf.String())
	if text == "" {
		return "", ErrNoExtractableText
	}

	return text, nil
}

// maxExtractPromptChars caps how much PDF text is fed into the extraction
// prompt — a company profile / capability deck rarely needs more than this
// to capture the fields we draft, and truncating keeps the prompt cheap and
// within context regardless of how long the source PDF is.
const maxExtractPromptChars = 12000

// ProfileDraft is the AI-extracted subset of Company Profile fields offered
// for review before the user confirms and saves them via PUT /api/profile
// (EP-13 ST-13.2/13.3) — field names deliberately mirror
// dto.ProfileUpdateRequest so the FE can merge this directly into the edit
// form. It is never persisted directly; only PUT /api/profile persists.
type ProfileDraft struct {
	CompanyName       string   `json:"company_name"`
	OneLiner          string   `json:"one_liner"`
	ServiceCategories []string `json:"service_categories"`
	TechStack         []string `json:"tech_stack"`
}

// truncateRunes returns the prefix of s containing at most maxRunes runes,
// cut on a rune boundary. Unlike string([]rune(s)[:maxRunes]) it does not
// materialize the whole string as a []rune — it advances rune-by-rune and
// slices the original string once, so a long input isn't fully copied when
// only its head is kept.
func truncateRunes(s string, maxRunes int) string {
	count := 0
	// range over a string iterates once per rune, with i the rune's starting
	// byte offset — so when we've seen maxRunes runes, s[:i] is exactly that
	// many runes, cut on a valid boundary.
	for i := range s {
		if count == maxRunes {
			return s[:i]
		}
		count++
	}
	return s
}

// buildExtractPrompt assembles the Company Profile field-extraction prompt
// (P-8): Bahasa Indonesia, ends with a strict "balas HANYA JSON" instruction
// matching ProfileDraft's schema exactly.
func buildExtractPrompt(text string) string {
	// Truncate by rune, not byte: a byte-offset cut can split a multi-byte
	// UTF-8 character in half (common in Indonesian text with curly quotes/
	// accented names), producing invalid UTF-8 in the prompt sent to Hermes.
	// Walk runes to find the cut point instead of allocating a full []rune of
	// the (possibly multi-MB) extracted text just to keep the first N runes.
	text = truncateRunes(text, maxExtractPromptChars)

	var b strings.Builder
	b.WriteString("Kamu membantu mengisi profil perusahaan (Company Profile) dari dokumen PDF berikut ")
	b.WriteString("(company profile / capability deck / RFI). Baca teks dokumen dan ekstrak field yang relevan. ")
	b.WriteString("Bila sebuah field tidak ditemukan di dokumen, kembalikan string kosong atau array kosong — JANGAN mengarang.\n\n")
	b.WriteString("## Teks Dokumen\n")
	b.WriteString(text)
	b.WriteString("\n\n")
	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"company_name": "...", "one_liner": "...", "service_categories": ["..."], "tech_stack": ["..."]}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	return b.String()
}

// Extractor turns raw PDF text into a ProfileDraft via Hermes GenerateJSON.
type Extractor struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewExtractor(hc hermes.Client, sk hermes.SessionKey) *Extractor {
	return &Extractor{hc: hc, sk: sk}
}

// Extract builds the extraction prompt for text and calls GenerateJSON. On
// any AI failure (Hermes down, timeout, invalid JSON after GenerateJSON's
// own retry) it returns the error unchanged — the caller (ProfileService)
// treats this as a non-fatal "degrade to manual entry" signal, not a failed
// upload (PRD §8: "gagal AI → pesan ramah, data utuh").
func (e *Extractor) Extract(ctx context.Context, text string) (*ProfileDraft, error) {
	prompt := buildExtractPrompt(text)

	var draft ProfileDraft
	if _, err := e.hc.GenerateJSON(ctx, prompt, &draft, e.sk); err != nil {
		return nil, fmt.Errorf("ai.Extract: %w", err)
	}

	return &draft, nil
}
