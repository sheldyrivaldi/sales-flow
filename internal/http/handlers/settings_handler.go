package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/config"
	"salespilot/internal/hermes"
)

// SettingsHandler backs the Settings page's AI/Hermes tab (EP-18 ST-18.1).
type SettingsHandler struct {
	hc  hermes.Client
	cfg *config.Config
}

func NewSettingsHandler(hc hermes.Client, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{hc: hc, cfg: cfg}
}

// HermesStatus handles GET /api/settings/hermes — a superset of
// GET /api/health/hermes (EP-17 TK-17.3.2, reused via hermesStatus so both
// endpoints agree on what "connected" means) plus memory_active: continuous
// learning (EP-16) is wired whenever a workspace session key is configured,
// since that key is what scopes every Hermes memory write
// (ai.LearningHermes/ai.NewExtractor/ai.NewScorer/... all key off it).
func (h *SettingsHandler) HermesStatus(c echo.Context) error {
	status := hermesStatus(c.Request().Context(), h.hc)
	status["memory_active"] = h.cfg.WorkspaceSessionKey != ""
	return c.JSON(http.StatusOK, status)
}

// TestHermes handles POST /api/settings/hermes/test — a fresh connectivity
// probe. Deliberately reuses hermesStatus (Health only, never Chat) so
// testing the connection never writes anything to workspace memory.
func (h *SettingsHandler) TestHermes(c echo.Context) error {
	status := hermesStatus(c.Request().Context(), h.hc)
	if status["status"] == "connected" {
		return c.JSON(http.StatusOK, map[string]any{"status": "ok", "version": status["version"]})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "failed"})
}
