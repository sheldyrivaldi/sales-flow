// event_analysis.go — analisa peserta event pasca-acara (item "upload doc
// after event"): user mengunggah dokumen daftar peserta/notulen (PDF via
// vision, atau tabel Excel yang FE konversi ke CSV), AI mengekstrak
// perusahaan yang hadir, memperkaya dengan riset web, lalu memetakan tiap
// perusahaan ke KUADRAN 2x2 (potensi nilai × tingkat minat) plus ringkasan
// dan saran timeline follow-up sales otomatis.
//
// Berbeda dari generator lain (yang memakai /v1/responses tanpa toolset),
// analisa ini sengaja lewat Chat — mode chat bridge mengaktifkan toolset web
// sehingga hermes benar-benar bisa mencari data perusahaan di internet.
package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// EventCompanyInsight adalah satu perusahaan peserta hasil analisa.
type EventCompanyInsight struct {
	Name     string `json:"name"`
	Industry string `json:"industry"`
	// Potential/Interest: "tinggi" | "rendah" — dua sumbu kuadran.
	Potential string `json:"potential"`
	Interest  string `json:"interest"`
	// Quadrant: "prioritas_utama" (tinggi/tinggi), "perlu_digarap"
	// (potensi tinggi, minat rendah), "quick_win" (potensi rendah, minat
	// tinggi), "dipantau" (rendah/rendah).
	Quadrant string `json:"quadrant"`
	Note     string `json:"note"`
}

// EventAnalysis adalah hasil lengkap analisa peserta satu event.
type EventAnalysis struct {
	Companies []EventCompanyInsight `json:"companies"`
	Summary   string                `json:"summary"`
	// TimelineSuggestions: rencana follow-up berurutan waktu (mis. "H+2: kirim
	// email terima kasih ke semua peserta", "Minggu 1: meeting 1-on-1 dengan
	// kuadran prioritas utama", ...).
	TimelineSuggestions []string `json:"timeline_suggestions"`
}

// EventAnalyzer menjalankan analisa via hermes Chat (toolset web aktif).
type EventAnalyzer struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewEventAnalyzer(hc hermes.Client, sk hermes.SessionKey) *EventAnalyzer {
	return &EventAnalyzer{hc: hc, sk: sk}
}

// stripJSONFences membuang pagar markdown ```json ... ``` bila model
// membungkus jawabannya — Chat (beda dengan /v1/responses) tidak punya
// penanganan fence bawaan.
func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	// Bila masih ada teks pengantar, ambil dari '{' pertama s.d. '}' terakhir.
	if !strings.HasPrefix(strings.TrimSpace(s), "{") {
		start := strings.Index(s, "{")
		end := strings.LastIndex(s, "}")
		if start != -1 && end > start {
			s = s[start : end+1]
		}
	}
	return strings.TrimSpace(s)
}

// Analyze menganalisa dokumen peserta event. docBytes (PDF, dibaca via
// vision) dan tableText (CSV hasil konversi Excel di FE) boleh salah satu
// atau keduanya; minimal satu harus terisi (divalidasi handler).
func (a *EventAnalyzer) Analyze(ctx context.Context, event domain.Event, profile *domain.ProfileAggregate, docBytes []byte, filename, tableText string) (*EventAnalysis, error) {
	var b strings.Builder
	b.WriteString("Kamu adalah analis market intelligence untuk tim sales B2B. Analisa peserta event berikut.\n\n")

	fmt.Fprintf(&b, "## Event\n- Nama: %s\n- Tipe: %s\n", event.Name, event.Type)
	if event.Date != nil {
		fmt.Fprintf(&b, "- Tanggal: %s\n", event.Date.Format("2006-01-02"))
	}
	if event.Location != nil && *event.Location != "" {
		fmt.Fprintf(&b, "- Lokasi: %s\n", *event.Location)
	}
	if event.Notes != nil && *event.Notes != "" {
		fmt.Fprintf(&b, "- Catatan: %s\n", *event.Notes)
	}

	writeProfileContext(&b, profile)

	if tableText != "" {
		b.WriteString("\n## Data Peserta (tabel)\n")
		b.WriteString(tableText)
		b.WriteString("\n")
	}
	if len(docBytes) > 0 {
		b.WriteString("\nDokumen peserta juga terlampir — baca seluruh halamannya.\n")
	}

	b.WriteString("\n## Tugas\n")
	b.WriteString("1. Ekstrak SEMUA perusahaan peserta dari data/dokumen.\n")
	b.WriteString("2. Untuk tiap perusahaan, cari informasi tambahan di internet (industri, skala, kebutuhan digitalisasi) bila memungkinkan.\n")
	b.WriteString("3. Petakan tiap perusahaan ke kuadran 2x2 relatif terhadap profil perusahaan kami:\n")
	b.WriteString("   - sumbu POTENSI nilai bisnis (tinggi/rendah) dan sumbu MINAT/kesiapan membeli (tinggi/rendah)\n")
	b.WriteString("   - quadrant: prioritas_utama (tinggi/tinggi), perlu_digarap (potensi tinggi, minat rendah), quick_win (potensi rendah, minat tinggi), dipantau (rendah/rendah)\n")
	b.WriteString("4. Tulis ringkasan temuan + saran TIMELINE follow-up sales berurutan waktu (H+1, minggu 1, bulan 1, dst).\n")
	b.WriteString("JANGAN mengarang perusahaan yang tidak ada di data. Bila data internet tidak ditemukan, nilai dari konteks event saja dan catat di note.\n")

	b.WriteString("\nBalas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"companies": [{"name": "...", "industry": "...", "potential": "tinggi|rendah", "interest": "tinggi|rendah", ` +
		`"quadrant": "prioritas_utama|perlu_digarap|quick_win|dipantau", "note": "..."}], ` +
		`"summary": "...", "timeline_suggestions": ["..."]}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	req := hermes.ChatRequest{
		Messages:   []hermes.Message{{Role: "user", Content: b.String()}},
		SessionKey: a.sk,
	}
	if len(docBytes) > 0 {
		req.DocumentBase64 = base64.StdEncoding.EncodeToString(docBytes)
		req.DocumentFilename = filename
	}

	resp, err := a.hc.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ai.EventAnalyzer.Analyze: %w", err)
	}

	var out EventAnalysis
	if err := json.Unmarshal([]byte(stripJSONFences(resp.Content)), &out); err != nil {
		return nil, fmt.Errorf("ai.EventAnalyzer.Analyze: output bukan JSON valid: %w", err)
	}
	return &out, nil
}
