// Package storage provides plain filesystem persistence for uploaded
// documents (EP-13 PDF Ingest). It has no DB/domain dependency so it can be
// unit tested without Postgres.
package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// ErrInvalidPDF is returned when the uploaded content is empty or does not
// start with the PDF magic bytes ("%PDF") — a cheap defense against
// MIME-spoofed uploads, not a full PDF validator.
var ErrInvalidPDF = errors.New("berkas bukan PDF yang valid")

// ErrFileTooLarge is returned when the uploaded content exceeds maxBytes.
var ErrFileTooLarge = errors.New("ukuran berkas melebihi batas")

const pdfMagic = "%PDF"

// SavePDF validates r as a PDF (magic bytes + size limit) and persists it
// under dir with a generated UUID filename — the original filename is never
// used for the path, which rules out path traversal and collisions. Returns
// the generated doc ref (just the filename, storage-relative) and byte size.
func SavePDF(dir string, r io.Reader, maxBytes int64) (docRef string, size int64, err error) {
	limited := io.LimitReader(r, maxBytes+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return "", 0, fmt.Errorf("storage.SavePDF: read: %w", err)
	}
	if int64(len(buf)) > maxBytes {
		return "", 0, ErrFileTooLarge
	}
	if len(buf) == 0 || !bytes.HasPrefix(buf, []byte(pdfMagic)) {
		return "", 0, ErrInvalidPDF
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("storage.SavePDF: mkdir: %w", err)
	}

	docRef = uuid.NewString() + ".pdf"
	path := filepath.Join(dir, docRef)
	if err := os.WriteFile(path, buf, 0o644); err != nil { //nolint:gosec // upload dir intentionally not secret
		return "", 0, fmt.Errorf("storage.SavePDF: write: %w", err)
	}

	return docRef, int64(len(buf)), nil
}

// FullPath resolves a doc ref to an absolute path under dir, for later
// reading (e.g. text extraction). Rejects anything that isn't a plain
// filename we generated ourselves — defense in depth against path traversal
// if a doc ref ever came from an untrusted source.
func FullPath(dir, docRef string) (string, error) {
	clean := filepath.Base(docRef)
	if clean != docRef || clean == "." || clean == string(filepath.Separator) {
		return "", fmt.Errorf("storage.FullPath: doc_ref tidak valid: %q", docRef)
	}
	return filepath.Join(dir, clean), nil
}
