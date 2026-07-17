package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"salespilot/internal/hermes"
)

// documentStub implements hermes.Client (minimally, all "not implemented"
// except what a given test needs) plus hermes.DocumentExtractor — the
// optional capability Extractor.Extract type-asserts for.
type documentStub struct {
	generateJSONFromDocument func(ctx context.Context, prompt, filename string, fileBytes []byte, schema any, sk hermes.SessionKey) (json.RawMessage, error)
}

var _ hermes.Client = (*documentStub)(nil)
var _ hermes.DocumentExtractor = (*documentStub)(nil)

func (s *documentStub) Chat(_ context.Context, _ hermes.ChatRequest) (hermes.ChatResponse, error) {
	return hermes.ChatResponse{}, errors.New("not implemented")
}

func (s *documentStub) ChatStream(_ context.Context, _ hermes.ChatRequest) (<-chan hermes.Chunk, error) {
	return nil, errors.New("not implemented")
}

func (s *documentStub) GenerateJSON(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
	return nil, errors.New("not implemented")
}

func (s *documentStub) Health(_ context.Context) (hermes.Capabilities, error) {
	return hermes.Capabilities{}, errors.New("not implemented")
}

func (s *documentStub) Configure(_ context.Context, _ hermes.ProviderConfig) error {
	return errors.New("not implemented")
}

func (s *documentStub) ResetMemory(_ context.Context, _ hermes.SessionKey) error {
	return errors.New("not implemented")
}

func (s *documentStub) GenerateJSONFromDocument(ctx context.Context, prompt, filename string, fileBytes []byte, schema any, sk hermes.SessionKey) (json.RawMessage, error) {
	return s.generateJSONFromDocument(ctx, prompt, filename, fileBytes, schema, sk)
}

func TestBuildExtractPrompt_ContainsSchemaAndInstructions(t *testing.T) {
	prompt := buildExtractPrompt()
	for _, want := range []string{
		"company_name", "portfolio_refs", "nogo_custom", "decision_maker_roles",
		"JANGAN mengarang", "JANGAN mencoba mencocokkan ke daftar preset",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestExtractor_Extract_SendsPDFBytesAndFilename(t *testing.T) {
	const wantFilename = "profile.pdf"
	wantBytes := []byte("%PDF-1.4 fake content")

	stub := &documentStub{
		generateJSONFromDocument: func(_ context.Context, _ string, filename string, fileBytes []byte, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			if filename != wantFilename {
				t.Errorf("filename = %q, want %q", filename, wantFilename)
			}
			if string(fileBytes) != string(wantBytes) {
				t.Errorf("fileBytes = %q, want %q (the raw PDF must be sent, not extracted text)", fileBytes, wantBytes)
			}
			raw := []byte(`{"company_name":"PT Contoh","one_liner":"Kami membangun aplikasi","service_categories":["Web App"],"tech_stack":["Go"],"products":[],"vision":"","mission":"","portfolio_refs":[],"keywords":[],"negative_keywords":[],"nogo_custom":[],"target":{"countries":[],"industries":[],"procurement_types":[],"buyer_size_note":"","document_languages":[],"work_model":"","onsite_limit_note":"","decision_maker_roles":[]}}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema: %v", err)
			}
			return raw, nil
		},
	}

	extractor := NewExtractor(stub, "sk-test")
	draft, err := extractor.Extract(context.Background(), wantBytes, wantFilename)
	if err != nil {
		t.Fatalf("Extract: unexpected error: %v", err)
	}
	if draft.CompanyName != "PT Contoh" {
		t.Errorf("CompanyName = %q, want %q", draft.CompanyName, "PT Contoh")
	}
}

func TestExtractor_Extract_HermesError(t *testing.T) {
	stub := &documentStub{
		generateJSONFromDocument: func(_ context.Context, _ string, _ string, _ []byte, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}

	extractor := NewExtractor(stub, "sk-test")
	_, err := extractor.Extract(context.Background(), []byte("%PDF-1.4 fake"), "profile.pdf")
	if err == nil {
		t.Fatal("Extract should return an error when Hermes fails, got nil")
	}
}

func TestExtractor_Extract_ClientWithoutDocumentSupport(t *testing.T) {
	// stubHermesClient (scoring_test.go, same package) only implements the
	// base hermes.Client — Extract must degrade with a clear error instead
	// of panicking on the failed type assertion.
	extractor := NewExtractor(&stubHermesClient{}, "sk-test")
	_, err := extractor.Extract(context.Background(), []byte("%PDF-1.4 fake"), "profile.pdf")
	if err == nil {
		t.Fatal("Extract should return an error when the client doesn't support DocumentExtractor")
	}
}
