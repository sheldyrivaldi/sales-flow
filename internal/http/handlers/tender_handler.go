package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type TenderHandler struct {
	svc *service.TenderService
}

func NewTenderHandler(svc *service.TenderService) *TenderHandler {
	return &TenderHandler{svc: svc}
}

// List handles GET /api/tenders
func (h *TenderHandler) List(c echo.Context) error {
	f := domain.TenderFilter{}

	if v := c.QueryParam("status"); v != "" {
		s := domain.TenderStatus(v)
		f.Status = &s
	}
	if v := c.QueryParam("recommended_action"); v != "" {
		a := domain.RecommendedAction(v)
		f.RecommendedAction = &a
	}
	if v := c.QueryParam("origin"); v != "" {
		o := domain.TenderOrigin(v)
		f.Origin = &o
	}
	if v := c.QueryParam("deadline_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.DeadlineFrom = &t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DeadlineFrom = &t
		}
	}
	if v := c.QueryParam("deadline_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.DeadlineTo = &t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DeadlineTo = &t
		}
	}
	f.Search = c.QueryParam("search")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))

	tenders, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	items := make([]dto.TenderResponse, len(tenders))
	for i, t := range tenders {
		items[i] = dto.ToTenderResponse(t)
	}

	return c.JSON(http.StatusOK, dto.TenderListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Get handles GET /api/tenders/:id
func (h *TenderHandler) Get(c echo.Context) error {
	id := c.Param("id")
	t, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// Create handles POST /api/tenders
func (h *TenderHandler) Create(c echo.Context) error {
	var req dto.TenderCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	t, err := h.svc.Create(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToTenderResponse(*t))
}

// Update handles PUT /api/tenders/:id
func (h *TenderHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.TenderUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	t, err := h.svc.Update(c.Request().Context(), id, &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// Delete handles DELETE /api/tenders/:id
func (h *TenderHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateStatus handles PATCH /api/tenders/:id/status
func (h *TenderHandler) UpdateStatus(c echo.Context) error {
	id := c.Param("id")

	var req dto.TenderStatusRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	t, err := h.svc.UpdateStatus(c.Request().Context(), id, domain.TenderStatus(req.Status))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// Promote handles POST /api/tenders/:id/promote
func (h *TenderHandler) Promote(c echo.Context) error {
	id := c.Param("id")
	t, err := h.svc.Promote(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// RecordOutcome handles POST /api/tenders/:id/outcome
func (h *TenderHandler) RecordOutcome(c echo.Context) error {
	id := c.Param("id")

	var req dto.TenderOutcomeRequest
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

	t, err := h.svc.RecordOutcome(c.Request().Context(), id, domain.OutcomeResult(req.Result), notes)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}
