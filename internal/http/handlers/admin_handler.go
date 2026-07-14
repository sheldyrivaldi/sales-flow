package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/config"
	"salespilot/internal/hermes"
	"salespilot/internal/http/httperr"
)

// AdminHandler holds admin-only Hermes operations (EP-16 ST-16.3) —
// currently just reset-memory; distinct from the per-entity handlers since
// it doesn't front a domain service, only the Hermes ACL directly.
type AdminHandler struct {
	hc  hermes.Client
	cfg *config.Config
}

func NewAdminHandler(hc hermes.Client, cfg *config.Config) *AdminHandler {
	return &AdminHandler{hc: hc, cfg: cfg}
}

// ResetHermesMemory handles POST /api/admin/hermes/reset-memory (ADMIN
// only, enforced by auth.RequireCapability(CapManageUsers) at the route).
// On any Hermes/bridge failure it returns a friendly error rather than a
// raw 500 — resetting memory is best-effort admin tooling, not part of the
// CRUD critical path.
func (h *AdminHandler) ResetHermesMemory(c echo.Context) error {
	if err := h.hc.ResetMemory(c.Request().Context(), hermes.SessionKey(h.cfg.WorkspaceSessionKey)); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("AI_UNAVAILABLE", "Reset memory Hermes sedang tidak tersedia, coba lagi nanti"))
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
