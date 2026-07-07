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

type ProspectHandler struct {
	svc *service.ProspectService
}

func NewProspectHandler(svc *service.ProspectService) *ProspectHandler {
	return &ProspectHandler{svc: svc}
}

// List handles GET /api/prospects
func (h *ProspectHandler) List(c echo.Context) error {
	f := domain.ProspectFilter{}

	if v := c.QueryParam("stage"); v != "" {
		s := domain.ProspectStage(v)
		f.Stage = &s
	}
	if v := c.QueryParam("owner_user_id"); v != "" {
		f.OwnerUserID = &v
	}
	if v := c.QueryParam("source_type"); v != "" {
		st := domain.ProspectSource(v)
		f.SourceType = &st
	}
	f.Search = c.QueryParam("search")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	prospects, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.ProspectResponse, len(prospects))
	for i, p := range prospects {
		items[i] = dto.ToProspectResponse(p)
	}

	return c.JSON(http.StatusOK, dto.ProspectListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Get handles GET /api/prospects/:id
func (h *ProspectHandler) Get(c echo.Context) error {
	id := c.Param("id")
	p, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToProspectResponse(*p))
}

// Create handles POST /api/prospects
func (h *ProspectHandler) Create(c echo.Context) error {
	var req dto.ProspectCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	defaultOwnerUserID := ""
	if user, ok := auth.UserFromContext(c); ok {
		defaultOwnerUserID = user.ID
	}

	p, err := h.svc.Create(c.Request().Context(), &req, defaultOwnerUserID)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToProspectResponse(*p))
}

// Update handles PUT /api/prospects/:id
func (h *ProspectHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.ProspectUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	p, err := h.svc.Update(c.Request().Context(), id, &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToProspectResponse(*p))
}

// Delete handles DELETE /api/prospects/:id
func (h *ProspectHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateStage handles PATCH /api/prospects/:id/stage
func (h *ProspectHandler) UpdateStage(c echo.Context) error {
	id := c.Param("id")

	var req dto.ProspectStageRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	p, err := h.svc.UpdateStage(c.Request().Context(), id, domain.ProspectStage(req.Stage), notes)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToProspectResponse(*p))
}
