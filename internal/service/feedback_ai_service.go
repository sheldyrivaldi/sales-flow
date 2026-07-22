package service

import (
	"context"
	"fmt"
	"strings"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// FeedbackAIService membantu menyusun & menganalisa form feedback dengan AI.
// Semua metode degrade-graceful (pola KeywordService): bila Hermes gagal,
// kembalikan hasil kosong + Degraded=true, bukan error — UI tetap bisa jalan
// manual.
type FeedbackAIService struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewFeedbackAIService(hc hermes.Client, sk hermes.SessionKey) *FeedbackAIService {
	return &FeedbackAIService{hc: hc, sk: sk}
}

// SuggestedQuestion adalah pertanyaan usulan AI (tanpa ID — ID diberi saat
// user memilih & menambahkannya ke form).
type SuggestedQuestion struct {
	Type        domain.QuestionType `json:"type"`
	Label       string              `json:"label"`
	Description string              `json:"description,omitempty"`
	Scale       int                 `json:"scale,omitempty"`
	Options     []string            `json:"options,omitempty"`
	Multiple    bool                `json:"multiple,omitempty"`
	MinLabel    string              `json:"min_label,omitempty"`
	MaxLabel    string              `json:"max_label,omitempty"`
}

// SuggestQuestionsResult membungkus usulan + flag degrade.
type SuggestQuestionsResult struct {
	Questions []SuggestedQuestion `json:"questions"`
	Degraded  bool                `json:"degraded"`
}

// SuggestQuestions meminta AI mengusulkan pertanyaan kuesioner dari deskripsi
// kebutuhan (prompt) + lampiran opsional (satu atau BANYAK PDF/gambar konteks
// proyek). lang menentukan bahasa pertanyaan yang dihasilkan.
func (s *FeedbackAIService) SuggestQuestions(ctx context.Context, prompt string, files []hermes.AgentDocument, lang domain.FormLanguage) (*SuggestQuestionsResult, error) {
	instruction := buildSuggestPrompt(prompt, lang)

	// PENTING: GenerateJSON memakai nilai `out` sekaligus sebagai contoh schema
	// yang dikirim ke model. Slice/objek NIL akan ter-serialisasi jadi
	// `{"questions":null}` — schema kosong yang membuat provider malah
	// mengembalikan `{"questions":null}` (bukan menghasilkan isi). Pra-isi satu
	// elemen contoh agar schema informatif; hasil unmarshal menimpanya penuh.
	out := struct {
		Questions []SuggestedQuestion `json:"questions"`
	}{Questions: []SuggestedQuestion{exampleSuggested()}}

	err := s.generate(ctx, instruction, files, &out)
	if err != nil && len(files) > 0 {
		// Lampiran gagal diproses (mis. berkas tak terbaca): jangan sampai
		// melampirkan berkas malah menggagalkan segalanya — coba ulang tanpa
		// lampiran memakai prompt saja.
		out.Questions = []SuggestedQuestion{exampleSuggested()}
		err = s.generate(ctx, instruction, nil, &out)
	}
	if err != nil {
		return &SuggestQuestionsResult{Questions: []SuggestedQuestion{}, Degraded: true}, nil
	}
	return &SuggestQuestionsResult{Questions: sanitizeSuggestions(out.Questions), Degraded: false}, nil
}

// generate memilih jalur Hermes yang tepat: banyak dokumen → MultiDocumentExtractor,
// satu dokumen → DocumentExtractor, tanpa dokumen → GenerateJSON biasa. Bila
// kapabilitas dokumen tak tersedia, jatuh ke teks-saja agar tetap jalan.
func (s *FeedbackAIService) generate(ctx context.Context, instruction string, files []hermes.AgentDocument, out any) error {
	nonEmpty := make([]hermes.AgentDocument, 0, len(files))
	for _, f := range files {
		if len(f.Bytes) > 0 {
			nonEmpty = append(nonEmpty, f)
		}
	}
	switch {
	case len(nonEmpty) > 1:
		if de, ok := s.hc.(hermes.MultiDocumentExtractor); ok {
			_, err := de.GenerateJSONFromDocuments(ctx, instruction, nonEmpty, out, s.sk)
			return err
		}
		// Fallback: kirim dokumen pertama saja lewat DocumentExtractor.
		if de, ok := s.hc.(hermes.DocumentExtractor); ok {
			_, err := de.GenerateJSONFromDocument(ctx, instruction, nonEmpty[0].Filename, nonEmpty[0].Bytes, out, s.sk)
			return err
		}
	case len(nonEmpty) == 1:
		if de, ok := s.hc.(hermes.DocumentExtractor); ok {
			_, err := de.GenerateJSONFromDocument(ctx, instruction, nonEmpty[0].Filename, nonEmpty[0].Bytes, out, s.sk)
			return err
		}
	}
	_, err := s.hc.GenerateJSON(ctx, instruction, out, s.sk)
	return err
}

// RefineQuestionResult membungkus satu pertanyaan hasil revisi + flag degrade.
type RefineQuestionResult struct {
	Question SuggestedQuestion `json:"question"`
	Degraded bool              `json:"degraded"`
}

// RefineQuestion merevisi satu pertanyaan mengikuti instruksi user (fitur
// "edit dengan AI" per pertanyaan). Bila AI gagal, kembalikan pertanyaan asal.
func (s *FeedbackAIService) RefineQuestion(ctx context.Context, q SuggestedQuestion, instruction string, lang domain.FormLanguage) (*RefineQuestionResult, error) {
	prompt := buildRefinePrompt(q, instruction, lang)
	// Pra-isi contoh (lihat catatan di SuggestQuestions) agar schema informatif.
	out := struct {
		Question SuggestedQuestion `json:"question"`
	}{Question: exampleSuggested()}
	if _, err := s.hc.GenerateJSON(ctx, prompt, &out, s.sk); err != nil {
		return &RefineQuestionResult{Question: q, Degraded: true}, nil
	}
	cleaned := sanitizeSuggestions([]SuggestedQuestion{out.Question})
	if len(cleaned) == 0 {
		return &RefineQuestionResult{Question: q, Degraded: true}, nil
	}
	return &RefineQuestionResult{Question: cleaned[0], Degraded: false}, nil
}

// FeedbackInsight adalah hasil analisa AI atas kumpulan feedback.
type FeedbackInsight struct {
	Summary      string   `json:"summary"`
	Strengths    []string `json:"strengths"`
	Weaknesses   []string `json:"weaknesses"`
	Improvements []string `json:"improvements"`
	Themes       []string `json:"themes"`
	Degraded     bool     `json:"degraded"`
}

// AnalyzeFeedback meminta AI meringkas kekuatan, kekurangan, saran improvement,
// dan tema komentar dari agregat feedback (dipanggil menu Analisa Feedback).
func (s *FeedbackAIService) AnalyzeFeedback(ctx context.Context, a *FormAnalytics, lang domain.FormLanguage) (*FeedbackInsight, error) {
	if a == nil || a.TotalSubmissions == 0 {
		msg := "Belum ada cukup data feedback untuk dianalisa."
		if lang == domain.LangEN {
			msg = "Not enough feedback data to analyze yet."
		}
		return &FeedbackInsight{
			Summary:   msg,
			Strengths: []string{}, Weaknesses: []string{}, Improvements: []string{}, Themes: []string{},
			Degraded: false,
		}, nil
	}
	prompt := buildAnalyzePrompt(a, lang)
	// Pra-isi contoh (lihat catatan di SuggestQuestions) agar slice tidak
	// ter-serialisasi jadi null di schema — jika tidak, model mengembalikan
	// semua array null.
	out := struct {
		Summary      string   `json:"summary"`
		Strengths    []string `json:"strengths"`
		Weaknesses   []string `json:"weaknesses"`
		Improvements []string `json:"improvements"`
		Themes       []string `json:"themes"`
	}{
		Summary:      "ringkasan",
		Strengths:    []string{"contoh kekuatan"},
		Weaknesses:   []string{"contoh kekurangan"},
		Improvements: []string{"contoh saran"},
		Themes:       []string{"contoh tema"},
	}
	if _, err := s.hc.GenerateJSON(ctx, prompt, &out, s.sk); err != nil {
		return &FeedbackInsight{
			Summary:   "",
			Strengths: []string{}, Weaknesses: []string{}, Improvements: []string{}, Themes: []string{},
			Degraded: true,
		}, nil
	}
	return &FeedbackInsight{
		Summary:      strings.TrimSpace(out.Summary),
		Strengths:    nonNil(out.Strengths),
		Weaknesses:   nonNil(out.Weaknesses),
		Improvements: nonNil(out.Improvements),
		Themes:       nonNil(out.Themes),
		Degraded:     false,
	}, nil
}

// --- Prompt builders ---

// langName mengembalikan nama bahasa target untuk instruksi AI.
func langName(lang domain.FormLanguage) string {
	if lang == domain.LangEN {
		return "English"
	}
	return "Bahasa Indonesia"
}

func buildSuggestPrompt(userPrompt string, lang domain.FormLanguage) string {
	var b strings.Builder
	b.WriteString("Kamu konsultan riset pengalaman client yang menyusun kuesioner feedback pasca-proyek untuk sebuah perusahaan jasa/teknologi. ")
	b.WriteString("Tujuan kuesioner: mengukur kepuasan client SEKALIGUS menghasilkan data yang bernilai untuk analisa kekurangan dan rekomendasi perbaikan.\n\n")

	b.WriteString("KONTEKS DARI USER: ")
	if strings.TrimSpace(userPrompt) == "" {
		b.WriteString("(tidak diisi) — susun kuesioner kepuasan client umum untuk proyek jasa/teknologi.")
	} else {
		b.WriteString(strings.TrimSpace(userPrompt))
	}
	b.WriteString("\n\n")

	b.WriteString("LAMPIRAN: Bila ada dokumen/gambar terlampir (mis. SRS, proposal, kontrak, laporan proyek), BACA dengan teliti dan JADIKAN DASAR utama. ")
	b.WriteString("Kaitkan pertanyaan dengan hal konkret dari lampiran: nama/lingkup proyek, deliverable, milestone, teknologi, pihak yang terlibat, dan janji layanan. ")
	b.WriteString("Prioritaskan isi lampiran + konteks user di atas pertanyaan generik.\n\n")

	b.WriteString("PRINSIP PENYUSUNAN (agar hasilnya bernilai untuk analisa & improvement):\n")
	b.WriteString("- Setiap pertanyaan mengukur SATU dimensi yang jelas dan dapat ditindaklanjuti. Cakup dimensi kunci yang relevan: kualitas hasil, kesesuaian dengan kebutuhan/scope, ketepatan waktu, komunikasi & responsivitas, kompetensi tim, dukungan/after-sales, dan nilai bisnis.\n")
	b.WriteString("- Utamakan tipe terukur (rating/nps) untuk dimensi yang perlu dipantau & dibandingkan antar-waktu, sehingga skor rendah langsung menandai area perbaikan. Sertakan MINIMAL SATU pertanyaan teks terbuka untuk menggali alasan & usulan perbaikan dari client.\n")
	b.WriteString("- Wajib sertakan satu pertanyaan NPS (kemungkinan merekomendasikan).\n")
	b.WriteString("- Pada field description, tuliskan singkat DIMENSI yang diukur pertanyaan itu (mis. \"komunikasi\", \"ketepatan waktu\") agar mudah dianalisa.\n")
	b.WriteString("- Usulkan 6–9 pertanyaan; hindari pertanyaan ganda/berlebihan.\n\n")

	b.WriteString("Tipe yang boleh dipakai: ")
	b.WriteString(`"rating" (angka 1..scale, default scale 5; WAJIB isi min_label & max_label sebagai keterangan ujung skala kiri/kanan), "text" (jawaban bebas), `)
	b.WriteString(`"choice" (pilihan dari options minimal 2; set multiple=true bila boleh pilih banyak), "nps" (skor 0..10). `)
	b.WriteString(fmt.Sprintf("Tulis SEMUA label, description, options, dan keterangan dalam %s yang sopan dan ringkas.\n\n", langName(lang)))

	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"questions": [{"type": "rating|text|choice|nps", "label": "...", "description": "dimensi yang diukur", "scale": 5, "options": ["..."], "multiple": false, "min_label": "", "max_label": ""}]}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")
	return b.String()
}

func buildRefinePrompt(q SuggestedQuestion, instruction string, lang domain.FormLanguage) string {
	var b strings.Builder
	b.WriteString("Revisi satu pertanyaan kuesioner feedback sesuai instruksi. Pertahankan maksud aslinya kecuali diminta berbeda.\n\n")
	b.WriteString(fmt.Sprintf("Pertanyaan saat ini: tipe=%s, label=%q", q.Type, q.Label))
	if len(q.Options) > 0 {
		b.WriteString(fmt.Sprintf(", options=%v", q.Options))
	}
	b.WriteString(".\n")
	b.WriteString("Instruksi: ")
	b.WriteString(instruction)
	b.WriteString(fmt.Sprintf("\n\nTulis hasilnya dalam %s.", langName(lang)))
	b.WriteString("\n\nBalas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"question": {"type": "rating|text|choice|nps", "label": "...", "description": "", "scale": 5, "options": ["..."], "multiple": false, "min_label": "", "max_label": ""}}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")
	return b.String()
}

func buildAnalyzePrompt(a *FormAnalytics, lang domain.FormLanguage) string {
	var b strings.Builder
	b.WriteString("Kamu analis pengalaman client. Analisa ringkasan feedback berikut dan beri wawasan yang jujur & actionable untuk perusahaan jasa/teknologi.\n\n")
	b.WriteString(fmt.Sprintf("Total respon: %d. Rata-rata rating (skala 5): %.1f. NPS: %d.\n", a.TotalSubmissions, a.AvgRating, a.NPS))
	b.WriteString("Ringkasan per pertanyaan:\n")
	for _, q := range a.Questions {
		switch q.Type {
		case string(domain.QuestionRating):
			b.WriteString(fmt.Sprintf("- [rating] %s: rata-rata %.1f (%d respon)\n", q.Label, q.Average, q.Responses))
		case string(domain.QuestionNPS):
			b.WriteString(fmt.Sprintf("- [nps] %s: rata-rata %.1f (%d respon)\n", q.Label, q.Average, q.Responses))
		case string(domain.QuestionChoice):
			b.WriteString(fmt.Sprintf("- [pilihan] %s: %v\n", q.Label, q.Choices))
		case string(domain.QuestionText):
			b.WriteString(fmt.Sprintf("- [teks] %s (%d jawaban):\n", q.Label, q.Responses))
			for i, t := range q.Texts {
				if i >= 30 {
					b.WriteString("  … (dipangkas)\n")
					break
				}
				b.WriteString("  • " + strings.ReplaceAll(t, "\n", " ") + "\n")
			}
		}
	}
	b.WriteString("\nInstruksi analisa (beri BOBOT & prioritas, bukan sekadar daftar):\n")
	b.WriteString("- Tafsirkan skor: rating/NPS RENDAH = area lemah berprioritas tinggi; kaitkan tiap kekurangan dengan dimensi & angka pendukungnya (mis. \"komunikasi rata-rata 2.8/5\").\n")
	b.WriteString("- weaknesses: urutkan dari dampak paling besar; sebut dimensi + bukti angka/komentar.\n")
	b.WriteString("- improvements: saran konkret & actionable untuk tiap kelemahan utama, URUT dari prioritas tertinggi (dampak terbesar × paling mendesak); bila memungkinkan sebut langkah nyata, bukan normatif.\n")
	b.WriteString("- strengths: dimensi berskor tinggi yang harus dipertahankan. themes: pola berulang dari komentar teks.\n")
	b.WriteString("- summary: 1–2 kalimat menyorot kondisi umum + 1 prioritas perbaikan teratas.\n")
	b.WriteString(fmt.Sprintf("Tulis SEMUA dalam %s. Bila data tipis, katakan apa adanya dan jangan mengarang.\n\n", langName(lang)))
	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"summary": "...", "strengths": ["..."], "weaknesses": ["..."], "improvements": ["..."], "themes": ["..."]}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")
	return b.String()
}

// --- helpers ---

// exampleSuggested adalah satu pertanyaan contoh untuk mengisi schema hint
// GenerateJSON (agar tidak ter-serialisasi null). Nilainya tidak pernah
// dipakai sebagai hasil — selalu ditimpa oleh output model.
func exampleSuggested() SuggestedQuestion {
	return SuggestedQuestion{
		Type:        domain.QuestionRating,
		Label:       "contoh pertanyaan",
		Description: "",
		Scale:       5,
		Options:     []string{"opsi"},
		Multiple:    false,
		MinLabel:    "rendah",
		MaxLabel:    "tinggi",
	}
}

// sanitizeSuggestions memangkas usulan tak valid & menerapkan default aman.
func sanitizeSuggestions(in []SuggestedQuestion) []SuggestedQuestion {
	out := make([]SuggestedQuestion, 0, len(in))
	for _, q := range in {
		q.Label = strings.TrimSpace(q.Label)
		if q.Label == "" || !q.Type.Valid() {
			continue
		}
		switch q.Type {
		case domain.QuestionRating:
			if q.Scale < 2 || q.Scale > 10 {
				q.Scale = 5
			}
			q.Options, q.Multiple = nil, false
			q.MinLabel = strings.TrimSpace(q.MinLabel)
			q.MaxLabel = strings.TrimSpace(q.MaxLabel)
		case domain.QuestionChoice:
			cleaned := make([]string, 0, len(q.Options))
			for _, o := range q.Options {
				if o = strings.TrimSpace(o); o != "" {
					cleaned = append(cleaned, o)
				}
			}
			if len(cleaned) < 2 {
				continue // choice tanpa opsi memadai — buang
			}
			q.Options, q.Scale = cleaned, 0
			q.MinLabel, q.MaxLabel = "", ""
		case domain.QuestionNPS, domain.QuestionText:
			q.Scale, q.Options, q.Multiple = 0, nil, false
			q.MinLabel, q.MaxLabel = "", ""
		}
		out = append(out, q)
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
