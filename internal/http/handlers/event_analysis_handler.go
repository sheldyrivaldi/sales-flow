package handlers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/ai"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// maxEventDocBytes caps the uploaded participant document (10 MB, same as
// the other document-ingest paths).
const maxEventDocBytes = 10 * 1024 * 1024

// EventAnalysisHandler serves POST /api/events/:id/analyze — analisa peserta
// event pasca-acara (dokumen PDF via vision dan/atau tabel CSV hasil
// konversi Excel di FE) menjadi pemetaan kuadran + ringkasan + saran
// timeline follow-up. On-demand, tidak dipersist.
type EventAnalysisHandler struct {
	analyzer *ai.EventAnalyzer
	events   *service.EventService
	profiles *service.ProfileService
}

func NewEventAnalysisHandler(analyzer *ai.EventAnalyzer, events *service.EventService, profiles *service.ProfileService) *EventAnalysisHandler {
	return &EventAnalysisHandler{analyzer: analyzer, events: events, profiles: profiles}
}

// Analyze handles POST /api/events/:id/analyze (multipart: optional "file"
// PDF + optional "table_text" CSV; minimal salah satu).
func (h *EventAnalysisHandler) Analyze(c echo.Context) error {
	ev, err := h.events.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	profile, err := h.profiles.GetCurrent(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}

	tableText := c.FormValue("table_text")

	var docBytes []byte
	var filename string
	if fh, ferr := c.FormFile("file"); ferr == nil {
		if fh.Size > maxEventDocBytes {
			return httperr.Write(c, httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB"))
		}
		f, oerr := fh.Open()
		if oerr != nil {
			return httperr.Write(c, httperr.NewBadRequest("FILE_UNREADABLE", "berkas tidak bisa dibaca"))
		}
		defer func() { _ = f.Close() }()
		docBytes, err = io.ReadAll(io.LimitReader(f, maxEventDocBytes+1))
		if err != nil || int64(len(docBytes)) > maxEventDocBytes {
			return httperr.Write(c, httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB"))
		}
		filename = fh.Filename
	}

	if len(docBytes) == 0 && tableText == "" {
		return httperr.Write(c, httperr.NewBadRequest("INPUT_REQUIRED", "unggah dokumen peserta (PDF) atau tabel Excel"))
	}

	out, err := h.analyzer.Analyze(c.Request().Context(), *ev, profile, docBytes, filename, tableText)
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("AI_UNAVAILABLE", "Analisa AI sedang tidak tersedia. Coba lagi sebentar lagi."))
	}
	return c.JSON(http.StatusOK, out)
}
