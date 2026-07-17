package handlers

import (
	"crypto/subtle"
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/ai"
)

// InternalHandler serves endpoints meant for machine callers inside the
// trust boundary (not end users) — gated by a shared secret instead of a
// user JWT, and mounted outside the /api JWT-protected group.
type InternalHandler struct {
	scheduler *ai.Scheduler
	secret    string
}

func NewInternalHandler(scheduler *ai.Scheduler, secret string) *InternalHandler {
	return &InternalHandler{scheduler: scheduler, secret: secret}
}

// TriggerDiscovery handles POST/GET /internal/discovery/trigger — the
// callback the Hermes cron job upserted on profile save (EP-12) hits to ask
// "run discovery now if due". Accepts both methods since it's invoked by an
// LLM-driven tool call whose exact HTTP verb support isn't guaranteed the
// way a browser fetch's would be.
//
// Auth is a single shared secret (CronTriggerSecret), accepted as either the
// "secret" query param or an "X-Cron-Secret" header — the query-param path
// exists because a generic web-fetch tool can usually be given a full URL
// but not always a custom header.
func (h *InternalHandler) TriggerDiscovery(c echo.Context) error {
	if h.secret == "" {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "CRON_TRIGGER_SECRET belum diset"})
	}

	provided := c.QueryParam("secret")
	if provided == "" {
		provided = c.Request().Header.Get("X-Cron-Secret")
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.secret)) != 1 {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}

	h.scheduler.TriggerIfDue(c.Request().Context())
	return c.JSON(http.StatusAccepted, echo.Map{"status": "triggered"})
}
