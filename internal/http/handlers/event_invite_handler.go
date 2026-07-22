package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// EventInviteHandler menjadwalkan/mengelola undangan event terjadwal.
type EventInviteHandler struct {
	svc   *service.EventInviteService
	users domain.UserRepository
}

func NewEventInviteHandler(svc *service.EventInviteService, users domain.UserRepository) *EventInviteHandler {
	return &EventInviteHandler{svc: svc, users: users}
}

// scheduleRequest adalah body POST /api/events/:id/invites.
type scheduleRequest struct {
	// When: same_day | h1 | h3 | h7 | custom.
	When string `json:"when" validate:"required"`
	// CustomAt (RFC3339) wajib bila when=custom — tanggal+jam kirim.
	CustomAt string `json:"custom_at"`
}

// Schedule handles POST /api/events/:id/invites — jadwalkan undangan. Pengirim
// diambil dari user yang login (nama + email), bukan diminta di body.
func (h *EventInviteHandler) Schedule(c echo.Context) error {
	var req scheduleRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	in := service.ScheduleInput{When: service.InviteWhen(req.When)}
	if in.When == service.InviteCustom {
		t, err := time.Parse(time.RFC3339, req.CustomAt)
		if err != nil {
			return httperr.Write(c, httperr.NewBadRequest("BAD_CUSTOM_TIME", "tanggal/jam kirim tidak valid"))
		}
		in.CustomAt = &t
	}

	// Pengirim = user yang memicu.
	if u, ok := auth.UserFromContext(c); ok {
		if usr, err := h.users.GetByID(c.Request().Context(), u.ID); err == nil && usr != nil {
			in.SenderName = usr.Name
			in.SenderEmail = usr.Email
		}
	}

	inv, err := h.svc.Schedule(c.Request().Context(), c.Param("id"), in)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, inv)
}

// List handles GET /api/events/:id/invites — jadwal undangan event.
func (h *EventInviteHandler) List(c echo.Context) error {
	items, err := h.svc.List(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Cancel handles DELETE /api/events/:id/invites/:inviteId — batalkan jadwal.
func (h *EventInviteHandler) Cancel(c echo.Context) error {
	if err := h.svc.Cancel(c.Request().Context(), c.Param("inviteId")); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
