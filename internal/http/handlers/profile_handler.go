package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type ProfileHandler struct {
	svc *service.ProfileService
}

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

// Get handles GET /api/profile — returns the current version, or a default
// (unsaved) template when the workspace has never configured one.
func (h *ProfileHandler) Get(c echo.Context) error {
	agg, err := h.svc.GetCurrent(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToProfileResponse(*agg))
}

// Ingest handles POST /api/profile/ingest — uploads a PDF for AI-assisted
// Company Profile drafting (EP-13). Stores the file, then best-effort
// extracts a field Draft via Hermes (Degraded=true, Draft=nil on any AI
// failure — the upload itself always succeeds independently of AI).
func (h *ProfileHandler) Ingest(c echo.Context) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("MISSING_FILE", "berkas PDF wajib disertakan (field 'file')"))
	}

	result, err := h.svc.IngestUpload(c.Request().Context(), fh)
	if err != nil {
		return httperr.Write(c, err)
	}

	return c.JSON(http.StatusOK, dto.IngestResponse{
		DocRef:   result.DocRef,
		Filename: result.Filename,
		Size:     result.Size,
		Draft:    result.Draft,
		Degraded: result.Degraded,
	})
}

// Update handles PUT /api/profile — always creates a new version.
func (h *ProfileHandler) Update(c echo.Context) error {
	var req dto.ProfileUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	agg, err := h.svc.Save(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToProfileResponse(*agg))
}
