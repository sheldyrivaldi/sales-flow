package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/ai"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// TenderAssistHandler serves the on-demand per-tender AI assists: document
// completeness checklist and standardized proposal draft. Results are not
// persisted — each call is a fresh generation against the current tender +
// company profile.
type TenderAssistHandler struct {
	assist   *ai.TenderAssist
	tenders  *service.TenderService
	profiles *service.ProfileService
}

func NewTenderAssistHandler(assist *ai.TenderAssist, tenders *service.TenderService, profiles *service.ProfileService) *TenderAssistHandler {
	return &TenderAssistHandler{assist: assist, tenders: tenders, profiles: profiles}
}

// DocChecklist handles POST /api/tenders/:id/doc-checklist.
func (h *TenderAssistHandler) DocChecklist(c echo.Context) error {
	t, err := h.tenders.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	profile, err := h.profiles.GetCurrent(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}

	out, err := h.assist.GenerateDocChecklist(c.Request().Context(), *t, profile)
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("AI_UNAVAILABLE", "Agent AI sedang tidak tersedia. Coba lagi sebentar lagi."))
	}
	return c.JSON(http.StatusOK, out)
}

// ProposalDraft handles POST /api/tenders/:id/proposal-draft.
func (h *TenderAssistHandler) ProposalDraft(c echo.Context) error {
	t, err := h.tenders.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	profile, err := h.profiles.GetCurrent(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}

	out, err := h.assist.GenerateProposalDraft(c.Request().Context(), *t, profile)
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("AI_UNAVAILABLE", "Agent AI sedang tidak tersedia. Coba lagi sebentar lagi."))
	}
	return c.JSON(http.StatusOK, out)
}
