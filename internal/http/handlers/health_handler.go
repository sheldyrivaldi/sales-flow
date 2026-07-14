package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/hermes"
)

// hermesHealthTimeout bounds how long we wait for Hermes before reporting
// disconnected — mirrors healthzHandler's DB ping timeout (internal/http/
// router.go).
const hermesHealthTimeout = 3 * time.Second

// hermesStatus probes Hermes and returns a status payload — the single
// source of truth shared by GET /api/health/hermes (EP-17 TK-17.3.2) and
// GET /api/settings/hermes (EP-18 TK-18.1.1), so "connected" means the same
// thing in both places. A Hermes failure never surfaces as a 5xx — the
// probe result itself communicates degradation.
func hermesStatus(ctx context.Context, hc hermes.Client) map[string]any {
	cctx, cancel := context.WithTimeout(ctx, hermesHealthTimeout)
	defer cancel()

	caps, err := hc.Health(cctx)
	if err != nil {
		return map[string]any{"status": "disconnected"}
	}
	return map[string]any{
		"status":  "connected",
		"version": caps.Version,
		"models":  caps.Models,
	}
}

// HealthHandler exposes Hermes connectivity status, distinct from the DB
// liveness check at /healthz (router.go's healthzHandler).
type HealthHandler struct {
	hc hermes.Client
}

func NewHealthHandler(hc hermes.Client) *HealthHandler {
	return &HealthHandler{hc: hc}
}

// HermesHealth handles GET /api/health/hermes.
func (h *HealthHandler) HermesHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, hermesStatus(c.Request().Context(), h.hc))
}
