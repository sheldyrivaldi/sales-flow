package ai

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildMinimalPDF assembles a byte-accurate minimal single-page PDF (no
// external tool/library needed to generate it — just careful offset
// bookkeeping) whose content stream shows contentText via the Tj operator.
// Pass contentText="" to produce a PDF with no visible text (empty content
// stream) for the "no extractable text" test case.
func buildMinimalPDF(contentText string) []byte {
	var buf bytes.Buffer
	var offsets [6]int // index 1..5 used (object numbers)

	buf.WriteString("%PDF-1.4\n")

	writeObj := func(num int, body string) {
		offsets[num] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", num, body)
	}

	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 200 200] /Contents 5 0 R >>")
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	var content string
	if contentText != "" {
		content = fmt.Sprintf("BT /F1 24 Tf 10 100 Td (%s) Tj ET", contentText)
	} else {
		content = "BT ET"
	}
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

func writeTempPDF(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.pdf")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestExtractText_ExtractsKnownText(t *testing.T) {
	const marker = "Rahasia Unik 42"
	path := writeTempPDF(t, buildMinimalPDF(marker))

	text, err := ExtractText(path)
	if err != nil {
		t.Fatalf("ExtractText: unexpected error: %v", err)
	}
	if !strings.Contains(text, marker) {
		t.Fatalf("extracted text %q does not contain marker %q", text, marker)
	}
}

func TestExtractText_NoTextLayer(t *testing.T) {
	path := writeTempPDF(t, buildMinimalPDF(""))

	_, err := ExtractText(path)
	if err != ErrNoExtractableText {
		t.Fatalf("err = %v, want ErrNoExtractableText", err)
	}
}

func TestExtractText_CorruptFile(t *testing.T) {
	path := writeTempPDF(t, []byte("this is not a pdf at all, just plain garbage bytes"))

	_, err := ExtractText(path)
	if err == nil {
		t.Fatal("expected error for corrupt file, got nil")
	}
}
