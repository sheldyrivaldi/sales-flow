package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
)

type SourceHandler struct {
	svc *service.SourceService
}

func NewSourceHandler(svc *service.SourceService) *SourceHandler {
	return &SourceHandler{svc: svc}
}

// List handles GET /api/sources
func (h *SourceHandler) List(c echo.Context) error {
	f := domain.SourceFilter{}

	if v := c.QueryParam("enabled"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err == nil {
			f.Enabled = &enabled
		}
	}
	if v := c.QueryParam("access"); v != "" {
		access := domain.SourceAccess(v)
		f.Access = &access
	}
	f.Search = c.QueryParam("search")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	sources, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.SourceResponse, len(sources))
	for i, s := range sources {
		items[i] = dto.ToSourceResponse(s)
	}

	return c.JSON(http.StatusOK, dto.SourceListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Presets handles GET /api/sources/presets
func (h *SourceHandler) Presets(c echo.Context) error {
	presets, err := h.svc.Presets(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, presets)
}

// ActivatePreset handles POST /api/sources/presets
func (h *SourceHandler) ActivatePreset(c echo.Context) error {
	var req dto.SourcePresetActivateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	src, err := h.svc.ActivatePreset(c.Request().Context(), req.Key)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToSourceResponse(*src))
}

// Get handles GET /api/sources/:id
func (h *SourceHandler) Get(c echo.Context) error {
	id := c.Param("id")
	src, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToSourceResponse(*src))
}

// Create handles POST /api/sources
func (h *SourceHandler) Create(c echo.Context) error {
	var req dto.SourceCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	src, err := h.svc.Create(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToSourceResponse(*src))
}

// Update handles PUT /api/sources/:id
func (h *SourceHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.SourceUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	src, err := h.svc.Update(c.Request().Context(), id, &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToSourceResponse(*src))
}

// Delete handles DELETE /api/sources/:id
func (h *SourceHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
