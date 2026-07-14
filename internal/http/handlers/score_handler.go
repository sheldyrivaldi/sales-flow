package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/ai"
	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type ScoreHandler struct {
	svc   *service.ScoreService
	audit domain.AuditRepository
}

func NewScoreHandler(svc *service.ScoreService, audit domain.AuditRepository) *ScoreHandler {
	return &ScoreHandler{svc: svc, audit: audit}
}

// ScoreTender handles POST /api/tenders/:id/score
func (h *ScoreHandler) ScoreTender(c echo.Context) error {
	return h.runScore(c, ai.ScoreTargetTender)
}

// ScoreProspect handles POST /api/prospects/:id/score
func (h *ScoreHandler) ScoreProspect(c echo.Context) error {
	return h.runScore(c, ai.ScoreTargetProspect)
}

// GetTenderScore handles GET /api/tenders/:id/score
func (h *ScoreHandler) GetTenderScore(c echo.Context) error {
	return h.getScore(c, ai.ScoreTargetTender)
}

// GetProspectScore handles GET /api/prospects/:id/score
func (h *ScoreHandler) GetProspectScore(c echo.Context) error {
	return h.getScore(c, ai.ScoreTargetProspect)
}

func (h *ScoreHandler) runScore(c echo.Context, targetType ai.ScoreTargetType) error {
	id := c.Param("id")
	s, err := h.svc.ScoreTarget(c.Request().Context(), targetType, id)
	if err != nil {
		return httperr.Write(c, err)
	}

	var actor string
	if u, ok := auth.UserFromContext(c); ok {
		actor = u.ID
	}
	var model, reasoning string
	if s.Model != nil {
		model = *s.Model
	}
	if s.Reasoning != nil {
		reasoning = *s.Reasoning
	}
	writeAIAuditEvent(c.Request().Context(), h.audit, actor, "ai.score", s.TargetType, s.TargetID, map[string]any{
		"model":              model,
		"fit_score":          s.FitScore,
		"recommended_action": s.RecommendedAction,
		"reasoning":          reasoning,
	})

	return c.JSON(http.StatusOK, dto.ToScoreResponse(*s))
}

// getScore returns the latest score, or a null body (200) when the target
// has never been scored yet — that is a normal state, not a 404.
func (h *ScoreHandler) getScore(c echo.Context, targetType ai.ScoreTargetType) error {
	id := c.Param("id")
	s, err := h.svc.GetLatestScore(c.Request().Context(), targetType, id)
	if err != nil {
		return httperr.Write(c, err)
	}
	if s == nil {
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusOK, dto.ToScoreResponse(*s))
}
