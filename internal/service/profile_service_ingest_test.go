package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"testing"

	"salespilot/internal/ai"
	"salespilot/internal/hermes"
	"salespilot/internal/http/httperr"
)

// buildMultipartPDF wraps a fake PDF's bytes into a *multipart.FileHeader,
// exactly as Echo's c.FormFile("file") would produce from a real HTTP
// upload — so IngestUpload is exercised the same way the handler calls it.
func buildMultipartPDF(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/", &body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}
	fh := req.MultipartForm.File["file"][0]
	return fh
}

func TestProfileService_IngestUpload_RejectsNonPDF(t *testing.T) {
	svc := NewProfileService(nil, t.TempDir(), nil, nil)
	fh := buildMultipartPDF(t, "notes.txt", []byte("just plain text"))

	_, err := svc.IngestUpload(context.Background(), fh)
	var apiErr *httperr.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "INVALID_FILE_TYPE" {
		t.Fatalf("err = %v, want INVALID_FILE_TYPE", err)
	}
}

func TestProfileService_IngestUpload_NoExtractor_Degrades(t *testing.T) {
	dir := t.TempDir()
	svc := NewProfileService(nil, dir, nil, nil) // extractor=nil, crawl=nil

	fh := buildMultipartPDF(t, "profile.pdf", []byte("%PDF-1.4\nfake content"))

	result, err := svc.IngestUpload(context.Background(), fh)
	if err != nil {
		t.Fatalf("IngestUpload: unexpected error: %v", err)
	}
	if !result.Degraded {
		t.Error("Degraded = false, want true (no extractor wired)")
	}
	if result.Draft != nil {
		t.Errorf("Draft = %+v, want nil", result.Draft)
	}
	if result.DocRef == "" {
		t.Error("DocRef is empty — upload should still have succeeded")
	}
}

func TestProfileService_IngestUpload_HermesFails_Degrades(t *testing.T) {
	dir := t.TempDir()
	extractor := ai.NewExtractor(&stubHermesClient{
		generateJSONFromDocument: func(_ context.Context, _ string, _ string, _ []byte, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}, "sk-test")
	svc := NewProfileService(nil, dir, extractor, nil)

	// The AI ingest path (vision-based, sends raw bytes to Hermes) no longer
	// needs a real PDF structure — Go's own upload validation only checks
	// the magic bytes (storage.SavePDF), and any deeper PDF validity check
	// now happens bridge-side against the actual rendering pipeline.
	fh := buildMultipartPDF(t, "profile.pdf", []byte("%PDF-1.4\nfake content"))

	result, err := svc.IngestUpload(context.Background(), fh)
	if err != nil {
		t.Fatalf("IngestUpload: unexpected error: %v", err)
	}
	if !result.Degraded {
		t.Error("Degraded = false, want true (Hermes GenerateJSONFromDocument failed)")
	}
	if result.Draft != nil {
		t.Errorf("Draft = %+v, want nil", result.Draft)
	}
	if result.DocRef == "" {
		t.Error("DocRef is empty — upload should still have succeeded despite AI failure")
	}
}

func TestProfileService_IngestUpload_Success(t *testing.T) {
	dir := t.TempDir()
	extractor := ai.NewExtractor(&stubHermesClient{
		generateJSONFromDocument: func(_ context.Context, _ string, filename string, fileBytes []byte, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			if filename != "profile.pdf" {
				t.Errorf("filename = %q, want profile.pdf", filename)
			}
			if len(fileBytes) == 0 {
				t.Error("fileBytes empty, want the raw uploaded PDF bytes")
			}
			raw := []byte(`{"company_name":"PT Contoh","one_liner":"Kami membangun aplikasi","service_categories":["Web App"],"tech_stack":["Go"]}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema: %v", err)
			}
			return raw, nil
		},
	}, "sk-test")
	svc := NewProfileService(nil, dir, extractor, nil)

	fh := buildMultipartPDF(t, "profile.pdf", []byte("%PDF-1.4\nPT Contoh company profile"))

	result, err := svc.IngestUpload(context.Background(), fh)
	if err != nil {
		t.Fatalf("IngestUpload: unexpected error: %v", err)
	}
	if result.Degraded {
		t.Error("Degraded = true, want false")
	}
	if result.Draft == nil {
		t.Fatal("Draft is nil, want populated")
	}
	if result.Draft.CompanyName != "PT Contoh" {
		t.Errorf("Draft.CompanyName = %q, want %q", result.Draft.CompanyName, "PT Contoh")
	}
}
