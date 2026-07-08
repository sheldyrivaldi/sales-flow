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
