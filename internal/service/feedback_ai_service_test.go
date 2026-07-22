package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

func TestFeedbackAI_SuggestQuestions_Success(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"questions":[
				{"type":"rating","label":"Kepuasan","scale":5},
				{"type":"text","label":"Saran"},
				{"type":"choice","label":"Rekomendasi","options":["Ya","Tidak"]},
				{"type":"choice","label":"Buang aku","options":["cuma satu"]},
				{"type":"bogus","label":"Tak valid"}
			]}`)
			_ = json.Unmarshal(raw, schema)
			return raw, nil
		},
	}
	svc := NewFeedbackAIService(stub, "sk")
	res, err := svc.SuggestQuestions(context.Background(), "ukur kepuasan", nil, domain.LangID)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if res.Degraded {
		t.Fatal("tidak seharusnya degraded")
	}
	// bogus (tipe tak valid) & choice satu-opsi harus dibuang → sisa 3.
	if len(res.Questions) != 3 {
		t.Fatalf("questions = %d, want 3: %+v", len(res.Questions), res.Questions)
	}
	if res.Questions[0].Type != domain.QuestionRating || res.Questions[0].Scale != 5 {
		t.Fatalf("q0 = %+v", res.Questions[0])
	}
}

func TestFeedbackAI_SuggestQuestions_DegradesOnError(t *testing.T) {
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}
	svc := NewFeedbackAIService(stub, "sk")
	res, err := svc.SuggestQuestions(context.Background(), "apa saja", nil, domain.LangID)
	if err != nil {
		t.Fatalf("suggest tidak boleh mengembalikan error (degrade): %v", err)
	}
	if !res.Degraded {
		t.Fatal("harus degraded saat Hermes gagal")
	}
	if len(res.Questions) != 0 {
		t.Fatalf("questions harus kosong saat degrade, got %d", len(res.Questions))
	}
}

func TestFeedbackAI_SuggestQuestions_UsesDocumentExtractor(t *testing.T) {
	var docCalled bool
	stub := &stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("harusnya pakai document extractor")
		},
		generateJSONFromDocument: func(_ context.Context, _, _ string, _ []byte, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			docCalled = true
			raw := []byte(`{"questions":[{"type":"text","label":"Dari dokumen"}]}`)
			_ = json.Unmarshal(raw, schema)
			return raw, nil
		},
	}
	svc := NewFeedbackAIService(stub, "sk")
	res, err := svc.SuggestQuestions(context.Background(), "konteks", []hermes.AgentDocument{{Filename: "brief.pdf", Bytes: []byte("%PDF-1.4")}}, domain.LangID)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if !docCalled {
		t.Fatal("lampiran ada tapi GenerateJSONFromDocument tidak dipanggil")
	}
	if len(res.Questions) != 1 || res.Questions[0].Label != "Dari dokumen" {
		t.Fatalf("questions = %+v", res.Questions)
	}
}

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
