package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// aiSettingTestTimeout bounds Test's real provider round-trip (an actual
// LLM call, unlike hermesStatus's bare liveness check) — generous enough for
// a real completion, short enough that a hung provider doesn't hang the
// request forever.
const aiSettingTestTimeout = 20 * time.Second

// AISettingHandler backs the Settings page's AI Provider tab (EP-18
// ST-18.4). All three routes are ADMIN-only (wired with CapManageUsers in
// router.go) since provider/model/API-key configuration is sensitive.
type AISettingHandler struct {
	svc *service.AISettingService
	hc  hermes.Client
}

func NewAISettingHandler(svc *service.AISettingService, hc hermes.Client) *AISettingHandler {
	return &AISettingHandler{svc: svc, hc: hc}
}

// Get handles GET /api/settings/ai — key always masked (never plaintext).
func (h *AISettingHandler) Get(c echo.Context) error {
	resp, err := h.svc.Get(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Update handles PUT /api/settings/ai — saves to DB then pushes to the
// bridge via hermes.Configure.
func (h *AISettingHandler) Update(c echo.Context) error {
	var req dto.AISettingUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	resp, err := h.svc.Update(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Test handles POST /api/settings/ai/test — a real probe of the configured
// provider/model/API-key (service.AISettingService.Test makes one actual
// LLM round-trip), not just bridge liveness: hermesStatus/Health alone would
// report "connected" even with an invalid key, since it never talks to the
// provider.
func (h *AISettingHandler) Test(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), aiSettingTestTimeout)
	defer cancel()

	if !h.svc.Test(ctx) {
		return c.JSON(http.StatusOK, map[string]any{"status": "failed"})
	}
	status := hermesStatus(ctx, h.hc)
	return c.JSON(http.StatusOK, map[string]any{"status": "ok", "version": status["version"]})
}
