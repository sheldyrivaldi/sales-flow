package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
)

type ReportHandler struct {
	svc   *service.ReportService
	audit domain.AuditRepository
}

func NewReportHandler(svc *service.ReportService, audit domain.AuditRepository) *ReportHandler {
	return &ReportHandler{svc: svc, audit: audit}
}

// Create handles POST /api/reports
func (h *ReportHandler) Create(c echo.Context) error {
	var req dto.ReportCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	r, err := h.svc.Create(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}

	var actor string
	if u, ok := auth.UserFromContext(c); ok {
		actor = u.ID
	}
	var model string
	if r.Model != nil {
		model = *r.Model
	}
	writeAIAuditEvent(c.Request().Context(), h.audit, actor, "ai.report", "report", r.ID, map[string]any{
		"model":        model,
		"report_type":  string(r.ReportType),
		"period_start": r.PeriodStart,
		"period_end":   r.PeriodEnd,
	})

	return c.JSON(http.StatusCreated, dto.ToReportResponse(*r))
}

// List handles GET /api/reports
func (h *ReportHandler) List(c echo.Context) error {
	f := domain.ReportFilter{}
	if v := c.QueryParam("type"); v != "" {
		rt := domain.ReportType(v)
		f.ReportType = &rt
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	reports, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.ReportResponse, len(reports))
	for i, r := range reports {
		items[i] = dto.ToReportResponse(r)
	}

	return c.JSON(http.StatusOK, dto.ReportListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Get handles GET /api/reports/:id
func (h *ReportHandler) Get(c echo.Context) error {
	id := c.Param("id")
	r, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToReportResponse(*r))
}

// Delete handles DELETE /api/reports/:id
func (h *ReportHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
