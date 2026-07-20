package handlers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// saveUploadBytes writes raw bytes under <uploadDir>/<subdir>/<uuid><ext> and
// returns the public URL ("/uploads/<subdir>/...") plus detected MIME. UUID
// filenames keep stored files unguessable. Shared by chat + playbook uploads.
func saveUploadBytes(uploadDir, subdir, filename string, raw []byte) (url, mime string, err error) {
	ext := filepath.Ext(filename)
	name := uuid.NewString() + ext
	dir := filepath.Join(uploadDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		return "", "", fmt.Errorf("write file: %w", err)
	}
	return "/uploads/" + subdir + "/" + name, mimeByExt(ext), nil
}
