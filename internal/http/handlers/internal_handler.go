package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/ai"
	"salespilot/internal/service"
)

// InternalHandler serves endpoints meant for machine callers inside the
// trust boundary (not end users) — gated by a shared secret instead of a
// user JWT, and mounted outside the /api JWT-protected group.
type InternalHandler struct {
	scheduler     *ai.Scheduler
	secret        string
	playbookJobs  *service.PlaybookJobService
	eventAnalysis *service.EventAnalysisService
	feedbackForms *service.FeedbackFormService
}

func NewInternalHandler(scheduler *ai.Scheduler, secret string, playbookJobs *service.PlaybookJobService, eventAnalysis *service.EventAnalysisService, feedbackForms *service.FeedbackFormService) *InternalHandler {
	return &InternalHandler{scheduler: scheduler, secret: secret, playbookJobs: playbookJobs, eventAnalysis: eventAnalysis, feedbackForms: feedbackForms}
}

// checkSecret validates the shared secret from query param or X-Cron-Secret
// header (mirrors TriggerDiscovery's dual accept).
func (h *InternalHandler) checkSecret(c echo.Context) bool {
	provided := c.QueryParam("secret")
	if provided == "" {
		provided = c.Request().Header.Get("X-Cron-Secret")
	}
	return h.secret != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(h.secret)) == 1
}

// CompletePlaybookJob handles POST /internal/playbook-jobs/:id/complete — the
// bridge posts back a finished playbook here (fire-and-forget callback). Body:
// {"content": {...}} on success, or {"error": "..."} on failure.
func (h *InternalHandler) CompletePlaybookJob(c echo.Context) error {
	if !h.checkSecret(c) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}
	var body struct {
		Content json.RawMessage `json:"content"`
		Error   string          `json:"error"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "body tidak valid"})
	}
	// json.RawMessage of the literal "null" is non-nil but not real content.
	content := body.Content
	if string(content) == "null" {
		content = nil
	}
	if err := h.playbookJobs.Complete(c.Request().Context(), c.Param("id"), content, body.Error); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
}

// CompleteEventAnalysis handles POST /internal/events/:id/analysis-complete —
// Hermes melapor balik hasil Analisa AI di sini, pola callback yang SAMA
// dengan playbook. Body: {"content": {...}} bila sukses, {"error": "..."}
// bila gagal.
func (h *InternalHandler) CompleteEventAnalysis(c echo.Context) error {
	if !h.checkSecret(c) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}
	var body struct {
		Content json.RawMessage `json:"content"`
		Error   string          `json:"error"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "body tidak valid"})
	}
	content := body.Content
	if string(content) == "null" {
		content = nil
	}
	if h.eventAnalysis == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "analisa event tidak dikonfigurasi"})
	}
	if err := h.eventAnalysis.Complete(c.Request().Context(), c.Param("id"), content, body.Error); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
}

// CompleteFeedbackAISuggest handles POST
// /internal/feedback-forms/:id/ai-suggest-complete — bridge melapor balik
// hasil generate saran pertanyaan (pola callback yang SAMA dengan playbook /
// analisa event). Body: {"content": {...}} bila sukses, {"error": "..."} bila
// gagal.
func (h *InternalHandler) CompleteFeedbackAISuggest(c echo.Context) error {
	if !h.checkSecret(c) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
	}
	var body struct {
		Content json.RawMessage `json:"content"`
		Error   string          `json:"error"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "body tidak valid"})
	}
	content := body.Content
	if string(content) == "null" {
		content = nil
	}
	if h.feedbackForms == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "feedback forms tidak dikonfigurasi"})
	}
	if err := h.feedbackForms.CompleteAISuggest(c.Request().Context(), c.Param("id"), content, body.Error); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
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
