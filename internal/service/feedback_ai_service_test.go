package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// --- evaluateSuggestJSON (shared decision logic used by the async callback) ---

func TestEvaluateSuggestJSON_SanitizesQuestions(t *testing.T) {
	raw := json.RawMessage(`{"confidence":95,"questions":[
		{"type":"rating","label":"Kepuasan","scale":5},
		{"type":"text","label":"Saran"},
		{"type":"choice","label":"Rekomendasi","options":["Ya","Tidak"]},
		{"type":"choice","label":"Buang aku","options":["cuma satu"]},
		{"type":"bogus","label":"Tak valid"}
	]}`)
	confidence, clarify, questions, err := evaluateSuggestJSON(raw)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if confidence != 95 {
		t.Fatalf("confidence = %d, want 95", confidence)
	}
	if len(clarify) != 0 {
		t.Fatalf("clarify harus kosong, got %+v", clarify)
	}
	// bogus (tipe tak valid) & choice satu-opsi harus dibuang → sisa 3.
	if len(questions) != 3 {
		t.Fatalf("questions = %d, want 3: %+v", len(questions), questions)
	}
	if questions[0].Type != domain.QuestionRating || questions[0].Scale != 5 {
		t.Fatalf("q0 = %+v", questions[0])
	}
}

func TestEvaluateSuggestJSON_LowConfidenceReturnsClarify(t *testing.T) {
	raw := json.RawMessage(`{"confidence":40,"clarifying_questions":["Ini kuesioner untuk proyek jenis apa?","Siapa target respondennya?"],"questions":[]}`)
	confidence, clarify, questions, err := evaluateSuggestJSON(raw)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if confidence != 40 {
		t.Fatalf("confidence = %d, want 40", confidence)
	}
	if len(questions) != 0 {
		t.Fatalf("questions harus kosong saat confidence rendah, got %d", len(questions))
	}
	if len(clarify) != 2 {
		t.Fatalf("clarify = %d, want 2: %+v", len(clarify), clarify)
	}
}

func TestEvaluateSuggestJSON_MissingConfidenceStillReturnsQuestions(t *testing.T) {
	// Model tidak mengisi confidence/clarifying_questions sama sekali — tanpa
	// clarifying_questions, hasil TETAP dipakai (jangan nyangkut di mode
	// klarifikasi hanya karena confidence default 0).
	raw := json.RawMessage(`{"questions":[{"type":"text","label":"Saran perbaikan"}]}`)
	_, clarify, questions, err := evaluateSuggestJSON(raw)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(clarify) != 0 {
		t.Fatalf("clarify harus kosong, got %+v", clarify)
	}
	if len(questions) != 1 {
		t.Fatalf("questions = %d, want 1: %+v", len(questions), questions)
	}
}

func TestEvaluateSuggestJSON_ClampsConfidenceAndTruncatesClarify(t *testing.T) {
	// confidence rendah (di bawah ambang) supaya benar-benar masuk cabang
	// klarifikasi — confidence tinggi akan mengabaikan clarifying_questions
	// sama sekali (lihat TestEvaluateSuggestJSON_SanitizesQuestions).
	raw := json.RawMessage(`{"confidence":-10,"clarifying_questions":["a","b","c","d","e","f"],"questions":[]}`)
	confidence, clarify, questions, err := evaluateSuggestJSON(raw)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if confidence != 0 {
		t.Fatalf("confidence = %d, want clamped 0", confidence)
	}
	if len(clarify) != maxClarifyingQuestions {
		t.Fatalf("clarify = %d, want truncated to %d", len(clarify), maxClarifyingQuestions)
	}
	if len(questions) != 0 {
		t.Fatalf("questions harus kosong saat klarifikasi diminta, got %+v", questions)
	}
}

func TestEvaluateSuggestJSON_InvalidJSON(t *testing.T) {
	if _, _, _, err := evaluateSuggestJSON(json.RawMessage(`not json`)); err == nil {
		t.Fatal("harus error untuk JSON tidak valid")
	}
}

// --- looksTooVagueForSuggest / defaultClarifyingQuestions ---

func TestLooksTooVagueForSuggest(t *testing.T) {
	if !looksTooVagueForSuggest("buatkan feedback", false) {
		t.Fatal("prompt 2 kata tanpa lampiran harus dianggap terlalu tipis")
	}
	if looksTooVagueForSuggest("tolong", true) {
		t.Fatal("prompt singkat DENGAN lampiran tidak boleh dianggap terlalu tipis")
	}
	if looksTooVagueForSuggest("ukur kepuasan client proyek A", false) {
		t.Fatal("prompt deskriptif tidak boleh dianggap terlalu tipis")
	}
	if len(defaultClarifyingQuestions(domain.LangID)) == 0 {
		t.Fatal("harus ada pertanyaan klarifikasi baku (id)")
	}
	if len(defaultClarifyingQuestions(domain.LangEN)) == 0 {
		t.Fatal("harus ada pertanyaan klarifikasi baku (en)")
	}
}

// --- ParseSuggestTypeCounts / typeCounts round-trip ---

func TestParseSuggestTypeCounts_Empty(t *testing.T) {
	for _, raw := range []string{"", "{}"} {
		spec, err := ParseSuggestTypeCounts(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if len(spec) != 0 {
			t.Fatalf("spec kosong untuk %q, got %+v", raw, spec)
		}
	}
}

func TestParseSuggestTypeCounts_MixedRandomAndFixed(t *testing.T) {
	spec, err := ParseSuggestTypeCounts(`{"rating":"random","text":2,"choice":0}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !spec[domain.QuestionRating].Random {
		t.Fatalf("rating harus random, got %+v", spec[domain.QuestionRating])
	}
	if spec[domain.QuestionText].Random || spec[domain.QuestionText].Count != 2 {
		t.Fatalf("text harus tetap 2, got %+v", spec[domain.QuestionText])
	}
	if spec[domain.QuestionChoice].Random || spec[domain.QuestionChoice].Count != 0 {
		t.Fatalf("choice harus 0, got %+v", spec[domain.QuestionChoice])
	}
	if _, ok := spec[domain.QuestionNPS]; ok {
		t.Fatal("nps tidak disebutkan, tidak boleh ada di map")
	}
}

func TestParseSuggestTypeCounts_InvalidValue(t *testing.T) {
	if _, err := ParseSuggestTypeCounts(`{"rating":"banyak"}`); err == nil {
		t.Fatal("harus error untuk nilai selain angka/\"random\"")
	}
	if _, err := ParseSuggestTypeCounts(`not json`); err == nil {
		t.Fatal("harus error untuk JSON tidak valid")
	}
}

func TestTypeCountsRoundTrip(t *testing.T) {
	spec := map[domain.QuestionType]TypeCountSpec{
		domain.QuestionRating: {Random: true},
		domain.QuestionText:   {Count: 3},
	}
	back := typeCountsFromStrings(typeCountsToStrings(spec))
	if !reflect.DeepEqual(spec, back) {
		t.Fatalf("round-trip mismatch: %+v != %+v", spec, back)
	}
	if typeCountsToStrings(map[domain.QuestionType]TypeCountSpec{}) != nil {
		t.Fatal("spec kosong harus menghasilkan map nil (tidak menyimpan apa-apa)")
	}
}

// --- AnalyzeFeedback (sinkron, tidak berubah) ---

func TestFeedbackAI_AnalyzeFeedback_DegradesOnError(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("down")
		},
	}
	svc := NewFeedbackAIService(stub, "sk")
	insight, err := svc.AnalyzeFeedback(context.Background(), &FormAnalytics{TotalSubmissions: 3, AvgRating: 4.0}, domain.LangID)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !insight.Degraded {
		t.Fatal("harus degraded saat Hermes gagal")
	}
	// Slice tidak boleh nil (agar JSON-nya [] bukan null).
	if insight.Weaknesses == nil || insight.Improvements == nil {
		t.Fatal("slice insight harus non-nil")
	}
}

func TestFeedbackAI_AnalyzeFeedback_NoData(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			t.Fatal("tidak boleh memanggil AI saat tanpa data")
			return nil, nil
		},
	}
	svc := NewFeedbackAIService(stub, "sk")
	insight, err := svc.AnalyzeFeedback(context.Background(), &FormAnalytics{TotalSubmissions: 0}, domain.LangID)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if insight.Degraded {
		t.Fatal("tanpa data bukan kondisi degraded")
	}
}

// --- RefineQuestion (sinkron, tidak berubah) ---

func TestFeedbackAI_RefineQuestion_DegradesOnError(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("down")
		},
	}
	svc := NewFeedbackAIService(stub, "sk")
	original := domain.SuggestedQuestion{Type: domain.QuestionText, Label: "Asli"}
	res, err := svc.RefineQuestion(context.Background(), original, "perjelas", domain.LangID)
	if err != nil {
		t.Fatalf("refine: %v", err)
	}
	if !res.Degraded {
		t.Fatal("harus degraded saat Hermes gagal")
	}
	if res.Question.Label != "Asli" {
		t.Fatalf("harus kembali ke pertanyaan asal saat degrade, got %+v", res.Question)
	}
}
