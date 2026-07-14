package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	svc := NewProfileService(nil, t.TempDir(), nil)
	fh := buildMultipartPDF(t, "notes.txt", []byte("just plain text"))

	_, err := svc.IngestUpload(context.Background(), fh)
	var apiErr *httperr.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "INVALID_FILE_TYPE" {
		t.Fatalf("err = %v, want INVALID_FILE_TYPE", err)
	}
}

func TestProfileService_IngestUpload_NoExtractor_Degrades(t *testing.T) {
	dir := t.TempDir()
	svc := NewProfileService(nil, dir, nil) // extractor=nil

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

func TestProfileService_IngestUpload_NoTextLayer_Degrades(t *testing.T) {
	dir := t.TempDir()
	extractor := ai.NewExtractor(&stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			t.Fatal("GenerateJSON should not be called when the PDF has no text layer")
			return nil, nil
		},
	}, "sk-test")
	svc := NewProfileService(nil, dir, extractor)

	// A minimal PDF with no readable text layer — ExtractText itself is
	// unlikely to even parse this malformed content, which is fine: any
	// extraction failure (no text OR unparsable) must degrade the same way.
	fh := buildMultipartPDF(t, "scan.pdf", []byte("%PDF-1.4\nnot a real pdf structure"))

	result, err := svc.IngestUpload(context.Background(), fh)
	if err != nil {
		t.Fatalf("IngestUpload: unexpected error: %v", err)
	}
	if !result.Degraded {
		t.Error("Degraded = false, want true (unreadable/no-text PDF)")
	}
	if result.Draft != nil {
		t.Errorf("Draft = %+v, want nil", result.Draft)
	}
}

func TestProfileService_IngestUpload_HermesFails_Degrades(t *testing.T) {
	dir := t.TempDir()
	extractor := ai.NewExtractor(&stubHermesClient{
		generateJSON: func(_ context.Context, _ string, _ any, _ hermes.SessionKey) (json.RawMessage, error) {
			return nil, errors.New("hermes down")
		},
	}, "sk-test")
	svc := NewProfileService(nil, dir, extractor)

	// Valid single-page PDF with real extractable text, built the same way
	// as internal/ai/profile_extract_test.go's buildMinimalPDF, inlined here
	// to avoid a cross-package test dependency.
	content := minimalPDFWithText(t, "Perusahaan Contoh Teknologi")
	fh := buildMultipartPDF(t, "profile.pdf", content)

	result, err := svc.IngestUpload(context.Background(), fh)
	if err != nil {
		t.Fatalf("IngestUpload: unexpected error: %v", err)
	}
	if !result.Degraded {
		t.Error("Degraded = false, want true (Hermes GenerateJSON failed)")
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
		generateJSON: func(_ context.Context, _ string, schema any, _ hermes.SessionKey) (json.RawMessage, error) {
			raw := []byte(`{"company_name":"PT Contoh","one_liner":"Kami membangun aplikasi","service_categories":["Web App"],"tech_stack":["Go"]}`)
			if err := json.Unmarshal(raw, schema); err != nil {
				t.Fatalf("unmarshal into schema: %v", err)
			}
			return raw, nil
		},
	}, "sk-test")
	svc := NewProfileService(nil, dir, extractor)

	content := minimalPDFWithText(t, "PT Contoh company profile")
	fh := buildMultipartPDF(t, "profile.pdf", content)

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

// minimalPDFWithText builds a byte-accurate minimal single-page PDF whose
// content stream shows text via the Tj operator — mirrors
// internal/ai/profile_extract_test.go's buildMinimalPDF (duplicated rather
// than exported across packages purely for a test fixture).
func minimalPDFWithText(t *testing.T, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	var offsets [6]int

	buf.WriteString("%PDF-1.4\n")
	writeObj := func(num int, body string) {
		offsets[num] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", num, body)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 200 200] /Contents 5 0 R >>")
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	content := fmt.Sprintf("BT /F1 24 Tf 10 100 Td (%s) Tj ET", text)
	streamBody := fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)
	writeObj(5, streamBody)

	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF", xrefOffset)

	return buf.Bytes()
}
