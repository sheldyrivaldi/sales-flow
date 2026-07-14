package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSavePDF_ValidPDF(t *testing.T) {
	dir := t.TempDir()
	content := []byte("%PDF-1.4\n%% minimal fake pdf content for test\n")

	docRef, size, err := SavePDF(dir, bytes.NewReader(content), 1024)
	if err != nil {
		t.Fatalf("SavePDF: unexpected error: %v", err)
	}
	if !strings.HasSuffix(docRef, ".pdf") {
		t.Fatalf("docRef = %q, want suffix .pdf", docRef)
	}
	if size != int64(len(content)) {
		t.Fatalf("size = %d, want %d", size, len(content))
	}

	got, err := os.ReadFile(filepath.Join(dir, docRef))
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("saved content mismatch")
	}
}

func TestSavePDF_RejectsNonPDF(t *testing.T) {
	dir := t.TempDir()
	content := []byte("this is just plain text, not a pdf")

	_, _, err := SavePDF(dir, bytes.NewReader(content), 1024)
	if err != ErrInvalidPDF {
		t.Fatalf("err = %v, want ErrInvalidPDF", err)
	}
}

func TestSavePDF_RejectsEmpty(t *testing.T) {
	dir := t.TempDir()

	_, _, err := SavePDF(dir, bytes.NewReader(nil), 1024)
	if err != ErrInvalidPDF {
		t.Fatalf("err = %v, want ErrInvalidPDF", err)
	}
}

func TestSavePDF_RejectsOversize(t *testing.T) {
	dir := t.TempDir()
	content := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("a"), 100)...)

	_, _, err := SavePDF(dir, bytes.NewReader(content), 10)
	if err != ErrFileTooLarge {
		t.Fatalf("err = %v, want ErrFileTooLarge", err)
	}
}

func TestFullPath_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()

	if _, err := FullPath(dir, "../../etc/passwd"); err == nil {
		t.Fatal("expected error for path traversal doc_ref, got nil")
	}
	if _, err := FullPath(dir, "sub/dir.pdf"); err == nil {
		t.Fatal("expected error for doc_ref containing separator, got nil")
	}

	got, err := FullPath(dir, "abc123.pdf")
	if err != nil {
		t.Fatalf("FullPath: unexpected error: %v", err)
	}
	want := filepath.Join(dir, "abc123.pdf")
	if got != want {
		t.Fatalf("FullPath = %q, want %q", got, want)
	}
}
