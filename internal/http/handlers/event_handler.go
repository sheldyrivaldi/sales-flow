package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type EventHandler struct {
	svc *service.EventService
}

func NewEventHandler(svc *service.EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

// List handles GET /api/events
func (h *EventHandler) List(c echo.Context) error {
	f := domain.EventFilter{}

	if v := c.QueryParam("type"); v != "" {
		t := domain.EventType(v)
		f.Type = &t
	}
	if v := c.QueryParam("status"); v != "" {
		s := domain.EventStatus(v)
		f.Status = &s
	}
	f.Search = c.QueryParam("search")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))

	events, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	items := make([]dto.EventResponse, len(events))
	for i, e := range events {
		items[i] = dto.ToEventResponse(e)
	}

	return c.JSON(http.StatusOK, dto.EventListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Get handles GET /api/events/:id
func (h *EventHandler) Get(c echo.Context) error {
	id := c.Param("id")
	e, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToEventResponse(*e))
}

// Create handles POST /api/events
func (h *EventHandler) Create(c echo.Context) error {
	var req dto.EventCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	e, err := h.svc.Create(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToEventResponse(*e))
}

// Update handles PUT /api/events/:id
func (h *EventHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.EventUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	e, err := h.svc.Update(c.Request().Context(), id, &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToEventResponse(*e))
}

// Delete handles DELETE /api/events/:id
func (h *EventHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Convert handles POST /api/events/:id/convert
func (h *EventHandler) Convert(c echo.Context) error {
	id := c.Param("id")

	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	p, err := h.svc.Convert(c.Request().Context(), id, user.ID)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToProspectResponse(*p))
}
