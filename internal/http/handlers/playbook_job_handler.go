package handlers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// maxPlaybookJobDocBytes caps an uploaded context document at 10 MB (same as
// the other document-driven AI features).
const maxPlaybookJobDocBytes = 10 << 20

// PlaybookJobHandler melayani menu Playbooks (generate async + riwayat).
type PlaybookJobHandler struct {
	svc       *service.PlaybookJobService
	uploadDir string
}

func NewPlaybookJobHandler(svc *service.PlaybookJobService, uploadDir string) *PlaybookJobHandler {
	return &PlaybookJobHandler{svc: svc, uploadDir: uploadDir}
}

// readPlaybookFile reads an optional multipart "file", returning its bytes,
// original name, and a saved public URL (empty when no file was sent).
func (h *PlaybookJobHandler) readPlaybookFile(c echo.Context) (raw []byte, filename, url string, err error) {
	fh, ferr := c.FormFile("file")
	if ferr != nil {
		return nil, "", "", nil // no file — not an error
	}
	if fh.Size > maxPlaybookJobDocBytes {
		return nil, "", "", httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB")
	}
	f, oerr := fh.Open()
	if oerr != nil {
		return nil, "", "", httperr.NewBadRequest("FILE_UNREADABLE", "berkas tidak bisa dibaca")
	}
	defer func() { _ = f.Close() }()
	b, rerr := io.ReadAll(io.LimitReader(f, maxPlaybookJobDocBytes+1))
	if rerr != nil || int64(len(b)) > maxPlaybookJobDocBytes {
		return nil, "", "", httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB")
	}
	savedURL, _, serr := saveUploadBytes(h.uploadDir, "playbook", fh.Filename, b)
	if serr != nil {
		// Non-fatal: still generate, just without a stored openable copy.
		savedURL = ""
	}
	return b, fh.Filename, savedURL, nil
}

// List handles GET /api/playbook-jobs
func (h *PlaybookJobHandler) List(c echo.Context) error {
	items, err := h.svc.List(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Get handles GET /api/playbook-jobs/:id
func (h *PlaybookJobHandler) Get(c echo.Context) error {
	job, err := h.svc.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, job)
}

// Create handles POST /api/playbook-jobs (multipart: field "title" opsional +
// "prompt" + optional "file"). Judul yang diisi user dipakai apa adanya dan
// tidak ditimpa AI. Returns the job immediately with status in_progress.
func (h *PlaybookJobHandler) Create(c echo.Context) error {
	title := c.FormValue("title")
	prompt := c.FormValue("prompt")

	pdfBytes, filename, url, ferr := h.readPlaybookFile(c)
	if ferr != nil {
		return httperr.Write(c, ferr)
	}

	job, err := h.svc.CreateCustom(c.Request().Context(), title, prompt, pdfBytes, filename, url)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, job)
}

// CreateFromEvent handles POST /api/events/:id/playbook-job — generate a
// standardized playbook using the full event context + web research.
func (h *PlaybookJobHandler) CreateFromEvent(c echo.Context) error {
	job, err := h.svc.CreateForEvent(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, job)
}

// Refine handles POST /api/playbook-jobs/:id/refine (multipart: field
// "instruction" + optional "file").
func (h *PlaybookJobHandler) Refine(c echo.Context) error {
	instruction := c.FormValue("instruction")
	pdfBytes, filename, url, ferr := h.readPlaybookFile(c)
	if ferr != nil {
		return httperr.Write(c, ferr)
	}
	job, err := h.svc.Refine(c.Request().Context(), c.Param("id"), instruction, pdfBytes, filename, url)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, job)
}

// Retry handles POST /api/playbook-jobs/:id/retry — jalankan ulang job gagal.
func (h *PlaybookJobHandler) Retry(c echo.Context) error {
	job, err := h.svc.Retry(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, job)
}

// Delete handles DELETE /api/playbook-jobs/:id
func (h *PlaybookJobHandler) Delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
