package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// FeedbackHandler melayani modul Pasca-Proyek: manajemen link feedback
// (authd) + endpoint publik untuk client mengisi form (tanpa login).
type FeedbackHandler struct {
	svc *service.FeedbackService
}

func NewFeedbackHandler(svc *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{svc: svc}
}

// Create handles POST /api/feedback — buat link feedback baru.
func (h *FeedbackHandler) Create(c echo.Context) error {
	var req dto.FeedbackCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}
	created, err := h.svc.CreateRequest(c.Request().Context(), req.ProjectName, req.ClientName, req.ProjectID)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, created)
}

// List handles GET /api/feedback — semua permintaan + respon (bila ada).
func (h *FeedbackHandler) List(c echo.Context) error {
	items, err := h.svc.ListRequests(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Delete handles DELETE /api/feedback/:id
func (h *FeedbackHandler) Delete(c echo.Context) error {
	if err := h.svc.DeleteRequest(c.Request().Context(), c.Param("id")); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Analytics handles GET /api/feedback/analytics
func (h *FeedbackHandler) Analytics(c echo.Context) error {
	a, err := h.svc.Analytics(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, a)
}

// PublicInfo handles GET /api/public/feedback/:token — dipanggil halaman
// publik /f/:token sebelum menampilkan form.
func (h *FeedbackHandler) PublicInfo(c echo.Context) error {
	req, err := h.svc.PublicInfo(c.Request().Context(), c.Param("token"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.FeedbackPublicInfo{
		ProjectName: req.ProjectName,
		ClientName:  req.ClientName,
		Submitted:   req.Response != nil,
	})
}

// PublicSubmit handles POST /api/public/feedback/:token — client submit
// jawaban (sekali per link).
func (h *FeedbackHandler) PublicSubmit(c echo.Context) error {
	var req dto.FeedbackSubmitRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	resp := &domain.FeedbackResponse{
		OverallRating:       req.OverallRating,
		QualityRating:       req.QualityRating,
		CommunicationRating: req.CommunicationRating,
		TimelinessRating:    req.TimelinessRating,
		NPS:                 req.NPS,
		Comment:             req.Comment,
		RespondentName:      req.RespondentName,
	}
	if err := h.svc.Submit(c.Request().Context(), c.Param("token"), resp); err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
}
