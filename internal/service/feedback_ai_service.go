package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// FeedbackAIService membantu menyusun & menganalisa form feedback dengan AI.
// RefineQuestion/AnalyzeFeedback tetap SINKRON (satu pertanyaan/satu ringkasan
// — cepat) dan degrade-graceful (pola KeywordService): bila Hermes gagal,
// kembalikan hasil kosong/asal + Degraded=true, bukan error.
//
// Susun kuesioner (SuggestQuestions dulu) kini ASINKRON — lihat
// FeedbackFormService.StartAISuggest — karena generate 6-9 pertanyaan +
// lampiran bisa makan waktu lama; helper prompt-building & evaluasi hasil di
// bawah ini dipakai bersama oleh jalur async tersebut.
type FeedbackAIService struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewFeedbackAIService(hc hermes.Client, sk hermes.SessionKey) *FeedbackAIService {
	return &FeedbackAIService{hc: hc, sk: sk}
}

// suggestConfidenceThreshold adalah ambang confidence (0-100) di bawah mana
// AI diminta bertanya klarifikasi ke user dulu, alih-alih mengarang kuesioner
// dari konteks yang masih tipis/ambigu.
const suggestConfidenceThreshold = 85

// maxClarifyingQuestions membatasi jumlah pertanyaan klarifikasi per putaran
// agar tidak membebani user.
const maxClarifyingQuestions = 4

// maxClarifyRounds membatasi berapa kali user boleh diminta klarifikasi
// sebelum AI dipaksa menyusun kuesioner terbaik dari info yang tersedia
// (mencegah percakapan klarifikasi tak berkesudahan).
const maxClarifyRounds = 3

// minSuggestPromptWords adalah ambang jumlah kata prompt di bawah mana
// konteks dianggap terlalu tipis untuk disimpulkan sendiri oleh AI. LLM
// cenderung terlalu percaya diri saat diminta menilai confidence sendiri
// (selalu merasa "cukup jelas" agar terlihat membantu) — jadi kasus paling
// umum (prompt kosong/sangat singkat seperti "buatkan feedback") ditangani
// deterministik di sini, tanpa bergantung pada penilaian model.
const minSuggestPromptWords = 3

// looksTooVagueForSuggest menandai prompt yang terlalu singkat untuk disusun
// jadi kuesioner relevan (tanpa lampiran yang bisa jadi konteks pengganti).
func looksTooVagueForSuggest(prompt string, hasFiles bool) bool {
	if hasFiles {
		return false
	}
	return len(strings.Fields(strings.TrimSpace(prompt))) < minSuggestPromptWords
}

// defaultClarifyingQuestions adalah pertanyaan klarifikasi baku untuk prompt
// yang terlalu tipis (dipakai looksTooVagueForSuggest), dalam bahasa form.
func defaultClarifyingQuestions(lang domain.FormLanguage) []string {
	if lang == domain.LangEN {
		return []string{
			"What kind of project or service is this feedback form for?",
			"Who will be filling out this form (the respondent's role)?",
			"What aspect do you most want to measure or improve from this feedback?",
		}
	}
	return []string{
		"Form ini untuk jenis proyek atau layanan apa?",
		"Siapa yang akan mengisi form ini (peran/posisi client)?",
		"Aspek apa yang paling ingin diukur atau diperbaiki dari feedback ini?",
	}
}

// TypeCountSpec adalah konfigurasi jumlah pertanyaan per tipe dari user:
// Random=true berarti AI bebas menentukan jumlahnya (minimal 1); selain itu
// Count adalah jumlah PERSIS yang wajib dihasilkan (0 = jangan sertakan sama
// sekali tipe itu).
type TypeCountSpec struct {
	Random bool
	Count  int
}

// parseTypeCounts mem-parse field multipart "type_counts" (JSON object,
// mis. {"rating":"random","text":2,"choice":0,"nps":"random"}) menjadi
// spesifikasi per tipe. String kosong/"{}" berarti user tidak mengatur
// apa pun → map kosong (AI bebas menentukan tipe & jumlah sepenuhnya).
func ParseSuggestTypeCounts(raw string) (map[domain.QuestionType]TypeCountSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return map[domain.QuestionType]TypeCountSpec{}, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, fmt.Errorf("type_counts: JSON tidak valid: %w", err)
	}
	out := make(map[domain.QuestionType]TypeCountSpec, len(fields))
	for k, v := range fields {
		t := domain.QuestionType(k)
		if !t.Valid() {
			continue
		}
		var n int
		if err := json.Unmarshal(v, &n); err == nil {
			if n < 0 {
				n = 0
			}
			out[t] = TypeCountSpec{Count: n}
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err == nil && strings.EqualFold(strings.TrimSpace(s), "random") {
			out[t] = TypeCountSpec{Random: true}
			continue
		}
		return nil, fmt.Errorf("type_counts: nilai untuk %q harus angka atau \"random\"", k)
	}
	return out, nil
}

// typeCountsToStrings mengubah spesifikasi menjadi map[type]string untuk
// disimpan pada domain.FeedbackAIJob.TypeCounts (JSONB) — dipakai lagi tanpa
// perubahan pada putaran klarifikasi berikutnya.
func typeCountsToStrings(spec map[domain.QuestionType]TypeCountSpec) map[domain.QuestionType]string {
	if len(spec) == 0 {
		return nil
	}
	out := make(map[domain.QuestionType]string, len(spec))
	for t, s := range spec {
		if s.Random {
			out[t] = "random"
		} else {
			out[t] = strconv.Itoa(s.Count)
		}
	}
	return out
}

// typeCountsFromStrings adalah kebalikan typeCountsToStrings, dipakai saat
// membaca AIJob.TypeCounts kembali dari DB untuk putaran klarifikasi lanjutan.
func typeCountsFromStrings(m map[domain.QuestionType]string) map[domain.QuestionType]TypeCountSpec {
	if len(m) == 0 {
		return map[domain.QuestionType]TypeCountSpec{}
	}
	out := make(map[domain.QuestionType]TypeCountSpec, len(m))
	for t, v := range m {
		if strings.EqualFold(v, "random") {
			out[t] = TypeCountSpec{Random: true}
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			n = 0
		}
		out[t] = TypeCountSpec{Count: n}
	}
	return out
}

// evaluateSuggestJSON mem-parse output mentah model (dari callback
// /v1/agent-task ATAU dari GenerateJSON) dan menerapkan aturan keputusan
// confidence/klarifikasi yang SAMA di satu tempat, supaya jalur async (lihat
// FeedbackFormService.CompleteAISuggest) tidak duplikasi logika ini.
func evaluateSuggestJSON(raw json.RawMessage) (confidence int, clarify []string, questions []domain.SuggestedQuestion, err error) {
	var out struct {
		Confidence          int                        `json:"confidence"`
		ClarifyingQuestions []string                    `json:"clarifying_questions"`
		Questions           []domain.SuggestedQuestion `json:"questions"`
	}
	if e := json.Unmarshal(raw, &out); e != nil {
		return 0, nil, nil, e
	}
	confidence = out.Confidence
	switch {
	case confidence < 0:
		confidence = 0
	case confidence > 100:
		confidence = 100
	}
	clarify = nonNil(out.ClarifyingQuestions)
	if len(clarify) > maxClarifyingQuestions {
		clarify = clarify[:maxClarifyingQuestions]
	}
	if confidence < suggestConfidenceThreshold && len(clarify) > 0 {
		return confidence, clarify, []domain.SuggestedQuestion{}, nil
	}
	return confidence, []string{}, sanitizeSuggestions(out.Questions), nil
}

// RefineQuestionResult membungkus satu pertanyaan hasil revisi + flag degrade.
type RefineQuestionResult struct {
	Question domain.SuggestedQuestion `json:"question"`
	Degraded bool                     `json:"degraded"`
}

// RefineQuestion merevisi satu pertanyaan mengikuti instruksi user (fitur
// "edit dengan AI" per pertanyaan). Bila AI gagal, kembalikan pertanyaan asal.
func (s *FeedbackAIService) RefineQuestion(ctx context.Context, q domain.SuggestedQuestion, instruction string, lang domain.FormLanguage) (*RefineQuestionResult, error) {
	prompt := buildRefinePrompt(q, instruction, lang)
	// Pra-isi contoh agar schema hint informatif (lihat catatan di
	// exampleSuggested) — hasil unmarshal menimpanya penuh.
	out := struct {
		Question domain.SuggestedQuestion `json:"question"`
	}{Question: exampleSuggested()}
	if _, err := s.hc.GenerateJSON(ctx, prompt, &out, s.sk); err != nil {
		return &RefineQuestionResult{Question: q, Degraded: true}, nil
	}
	cleaned := sanitizeSuggestions([]domain.SuggestedQuestion{out.Question})
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
	// Pra-isi contoh (lihat catatan di exampleSuggested) agar slice tidak
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

// questionTypeOrder adalah urutan tampil tetap untuk instruksi tipe/jumlah.
var questionTypeOrder = []domain.QuestionType{
	domain.QuestionRating, domain.QuestionText, domain.QuestionChoice, domain.QuestionNPS,
}

// buildTypeCountInstruction merakit instruksi WAJIB tentang tipe & jumlah
// pertanyaan dari konfigurasi user. Map kosong berarti user tidak mengatur
// apa pun → AI bebas menentukan tipe DAN jumlah sepenuhnya sendiri.
func buildTypeCountInstruction(spec map[domain.QuestionType]TypeCountSpec) string {
	if len(spec) == 0 {
		return "Tipe & jumlah pertanyaan BEBAS kamu tentukan sendiri — pilih campuran tipe (rating/text/choice/nps) dan total jumlah yang paling pas untuk konteks ini, SELAMA variatif dan bernilai untuk analisa (lihat STANDAR NILAI ANALISA di bawah).\n"
	}
	var b strings.Builder
	b.WriteString("Ikuti KETENTUAN JUMLAH PERTANYAAN PER TIPE berikut (WAJIB dipatuhi persis, ini permintaan eksplisit user):\n")
	specified := make(map[domain.QuestionType]bool, len(spec))
	for _, t := range questionTypeOrder {
		c, ok := spec[t]
		if !ok {
			continue
		}
		specified[t] = true
		switch {
		case c.Random:
			fmt.Fprintf(&b, "- Tipe %q: sertakan MINIMAL 1, kamu bebas menentukan jumlah persisnya (tetap variatif & bernilai analisa).\n", string(t))
		case c.Count <= 0:
			fmt.Fprintf(&b, "- Tipe %q: JANGAN sertakan sama sekali.\n", string(t))
		default:
			fmt.Fprintf(&b, "- Tipe %q: sertakan TEPAT %d pertanyaan.\n", string(t), c.Count)
		}
	}
	if len(specified) < len(questionTypeOrder) {
		b.WriteString("Tipe lain yang tidak disebutkan di atas: bebas kamu tentukan (boleh 0 boleh lebih) sesuai kebutuhan analisa.\n")
	}
	return b.String()
}

func buildSuggestPrompt(userPrompt string, lang domain.FormLanguage, typeSpec map[domain.QuestionType]TypeCountSpec) string {
	var b strings.Builder
	b.WriteString("Kamu konsultan riset pengalaman client yang membantu MENYUSUNKAN kuesioner feedback pasca-proyek untuk sebuah perusahaan jasa/teknologi. ")
	b.WriteString("Kamu berbicara ke DUA pihak berbeda: (1) USER, staf perusahaan yang meminta bantuanmu menyusun form — ke merekalah kamu mengajukan pertanyaan klarifikasi bila konteks belum jelas; (2) RESPONDEN, client yang nanti mengisi kuesioner — untuk merekalah isi kuesioner ditulis. Jangan sampai tertukar.\n\n")
	b.WriteString("Tujuan kuesioner: mengukur kepuasan client SEKALIGUS menghasilkan data yang bernilai untuk analisa kekurangan dan rekomendasi perbaikan.\n\n")

	b.WriteString("KONTEKS DARI USER: ")
	if strings.TrimSpace(userPrompt) == "" {
		b.WriteString("(tidak diisi)")
	} else {
		b.WriteString(strings.TrimSpace(userPrompt))
	}
	b.WriteString("\n\n")

	b.WriteString("LAMPIRAN: Bila ada dokumen/gambar terlampir (mis. SRS, proposal, kontrak, laporan proyek), BACA dengan teliti dan JADIKAN DASAR utama. ")
	b.WriteString("Kaitkan pertanyaan dengan hal konkret dari lampiran: nama/lingkup proyek, deliverable, milestone, teknologi, pihak yang terlibat, dan janji layanan. ")
	b.WriteString("Prioritaskan isi lampiran + konteks user di atas pertanyaan generik.\n\n")

	b.WriteString("LANGKAH 1 — NILAI KEJELASAN KONTEKS lebih dulu. Tentukan skor confidence 0-100 seberapa jelas kamu memahami: jenis/lingkup proyek, siapa target responden, dan aspek yang paling ingin diketahui perusahaan dari feedback ini (dari konteks user + lampiran di atas).\n")
	b.WriteString("- Bila confidence DI BAWAH 85 (konteks belum cukup jelas, ada istilah ambigu, atau tujuan pengukuran belum jelas): JANGAN mengarang isi kuesioner. Tulis 2 sampai 4 PERTANYAAN KLARIFIKASI singkat untuk USER (bukan untuk responden) — misalnya jenis/nama proyek, siapa yang akan mengisi form, atau aspek yang paling ingin diukur. Kosongkan \"questions\" (array kosong).\n")
	b.WriteString("- Bila confidence 85 KE ATAS: kosongkan \"clarifying_questions\" (array kosong) dan lanjut ke LANGKAH 2.\n\n")

	b.WriteString("LANGKAH 2 — SUSUN PERTANYAAN KUESIONER (hanya bila confidence sudah 85 ke atas).\n\n")
	b.WriteString(buildTypeCountInstruction(typeSpec))
	b.WriteString("\n")

	b.WriteString("PRINSIP PENYUSUNAN (agar hasilnya bernilai untuk analisa & improvement):\n")
	b.WriteString("- Setiap pertanyaan mengukur SATU dimensi yang jelas dan dapat ditindaklanjuti. Cakup dimensi kunci yang relevan: kualitas hasil, kesesuaian dengan kebutuhan/scope, ketepatan waktu, komunikasi & responsivitas, kompetensi tim, dukungan/after-sales, dan nilai bisnis.\n")
	b.WriteString("- Utamakan tipe terukur (rating/nps) untuk dimensi yang perlu dipantau & dibandingkan antar-waktu, sehingga skor rendah langsung menandai area perbaikan. Sertakan MINIMAL SATU pertanyaan teks terbuka untuk menggali alasan & usulan perbaikan dari client (kecuali user secara eksplisit meminta 0 pertanyaan tipe text di atas).\n")
	b.WriteString("- Bila tidak dilarang eksplisit di atas, sertakan satu pertanyaan NPS (kemungkinan merekomendasikan).\n")
	b.WriteString("- Pada field description, tuliskan singkat DIMENSI yang diukur pertanyaan itu (mis. \"komunikasi\", \"ketepatan waktu\") agar mudah dianalisa.\n\n")

	b.WriteString("STANDAR NILAI ANALISA (WAJIB — jangan sekadar formalitas):\n")
	b.WriteString("- Sebelum memasukkan sebuah pertanyaan, uji: \"apakah jawaban atas pertanyaan ini bisa DIAGREGASI/DIANALISA lintas responden dan mengarah ke satu tindakan konkret?\" Bila tidak, buang atau perbaiki pertanyaan itu.\n")
	b.WriteString("- DILARANG pertanyaan generik tanpa dimensi jelas (mis. \"Bagaimana pendapat Anda secara umum?\", \"Apakah Anda puas?\" tanpa konteks). Tiap pertanyaan HARUS terikat satu dimensi terukur & actionable.\n")
	b.WriteString("- DILARANG dua pertanyaan yang pada dasarnya menanyakan hal yang sama (duplikat makna), termasuk antar-pertanyaan tipe rating yang sama — tiap rating harus mengukur dimensi BERBEDA.\n")
	b.WriteString("- Pertanyaan choice: opsi jawaban harus saling lepas (mutually exclusive) dan mencakup kemungkinan yang wajar, bukan sekadar Ya/Tidak generik kecuali memang itu yang paling tepat untuk dimensinya.\n\n")

	b.WriteString("Tipe yang boleh dipakai: ")
	b.WriteString(`"rating" (angka 1..scale, default scale 5; WAJIB isi min_label & max_label sebagai keterangan ujung skala kiri/kanan), "text" (jawaban bebas), `)
	b.WriteString(`"choice" (pilihan dari options minimal 2; set multiple=true bila boleh pilih banyak), "nps" (skor 0..10). `)
	b.WriteString(fmt.Sprintf("Tulis SEMUA pertanyaan klarifikasi, label, description, options, dan keterangan dalam %s yang sopan dan ringkas.\n\n", langName(lang)))

	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"confidence": 0, "clarifying_questions": ["..."], "questions": [{"type": "rating|text|choice|nps", "label": "...", "description": "dimensi yang diukur", "scale": 5, "options": ["..."], "multiple": false, "min_label": "", "max_label": ""}]}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")
	return b.String()
}

func buildRefinePrompt(q domain.SuggestedQuestion, instruction string, lang domain.FormLanguage) string {
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
func exampleSuggested() domain.SuggestedQuestion {
	return domain.SuggestedQuestion{
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
func sanitizeSuggestions(in []domain.SuggestedQuestion) []domain.SuggestedQuestion {
	out := make([]domain.SuggestedQuestion, 0, len(in))
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
