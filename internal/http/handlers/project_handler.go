package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
)

// ProjectHandler melayani menu Proyek Berjalan (ongoing projects).
type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func parseDateOpt(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

func applyProjectRequest(p *domain.Project, req dto.ProjectUpsertRequest) {
	p.Name = req.Name
	p.ClientName = req.ClientName
	p.ContractValue = req.ContractValue
	if req.Currency != nil && *req.Currency != "" {
		p.Currency = *req.Currency
	}
	p.StartDate = parseDateOpt(req.StartDate)
	p.EndDate = parseDateOpt(req.EndDate)
	if req.Status != nil {
		p.Status = domain.ProjectStatus(*req.Status)
	}
	if req.Progress != nil {
		p.Progress = *req.Progress
	}
	p.Description = req.Description
	if req.Milestones != nil {
		p.Milestones = req.Milestones
	}
}

// List handles GET /api/projects
func (h *ProjectHandler) List(c echo.Context) error {
	f := domain.ProjectFilter{Search: c.QueryParam("search")}
	if v := c.QueryParam("status"); v != "" {
		st := domain.ProjectStatus(v)
		f.Status = &st
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	items, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ProjectListResponse{Items: items, Total: total, Page: page, PageSize: pageSize})
}

// Summary handles GET /api/projects/summary
func (h *ProjectHandler) Summary(c echo.Context) error {
	sum, err := h.svc.Summary(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, sum)
}

// Get handles GET /api/projects/:id
func (h *ProjectHandler) Get(c echo.Context) error {
	p, err := h.svc.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, p)
}

// Create handles POST /api/projects
func (h *ProjectHandler) Create(c echo.Context) error {
	var req dto.ProjectUpsertRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	p := &domain.Project{Currency: "IDR", Status: domain.ProjectOnTrack}
	applyProjectRequest(p, req)

	created, err := h.svc.Create(c.Request().Context(), p)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, created)
}

// Update handles PUT /api/projects/:id
func (h *ProjectHandler) Update(c echo.Context) error {
	var req dto.ProjectUpsertRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	p, err := h.svc.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	applyProjectRequest(p, req)

	updated, err := h.svc.Update(c.Request().Context(), p)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, updated)
}

// Delete handles DELETE /api/projects/:id
func (h *ProjectHandler) Delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// AddActivity handles POST /api/projects/:id/activities
func (h *ProjectHandler) AddActivity(c echo.Context) error {
	var req dto.ProjectActivityRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}
	p, err := h.svc.AddActivity(c.Request().Context(), c.Param("id"), req.Note)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, p)
}
