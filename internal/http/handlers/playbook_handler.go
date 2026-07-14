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

type PlaybookHandler struct {
	svc   *service.PlaybookService
	audit domain.AuditRepository
}

func NewPlaybookHandler(svc *service.PlaybookService, audit domain.AuditRepository) *PlaybookHandler {
	return &PlaybookHandler{svc: svc, audit: audit}
}

// GenerateTender handles POST /api/tenders/:id/playbook
func (h *PlaybookHandler) GenerateTender(c echo.Context) error {
	return h.generate(c, ai.ScoreTargetTender)
}

// GenerateProspect handles POST /api/prospects/:id/playbook
func (h *PlaybookHandler) GenerateProspect(c echo.Context) error {
	return h.generate(c, ai.ScoreTargetProspect)
}

// ListTenderPlaybooks handles GET /api/tenders/:id/playbooks
func (h *PlaybookHandler) ListTenderPlaybooks(c echo.Context) error {
	return h.list(c, ai.ScoreTargetTender)
}

// ListProspectPlaybooks handles GET /api/prospects/:id/playbooks
func (h *PlaybookHandler) ListProspectPlaybooks(c echo.Context) error {
	return h.list(c, ai.ScoreTargetProspect)
}

func (h *PlaybookHandler) generate(c echo.Context, targetType ai.ScoreTargetType) error {
	id := c.Param("id")
	pb, err := h.svc.Generate(c.Request().Context(), targetType, id)
	if err != nil {
		return httperr.Write(c, err)
	}

	var actor string
	if u, ok := auth.UserFromContext(c); ok {
		actor = u.ID
	}
	var model string
	if pb.Model != nil {
		model = *pb.Model
	}
	writeAIAuditEvent(c.Request().Context(), h.audit, actor, "ai.playbook", pb.TargetType, pb.TargetID, map[string]any{
		"model":   model,
		"version": pb.Version,
	})

	return c.JSON(http.StatusOK, dto.ToPlaybookResponse(*pb))
}

func (h *PlaybookHandler) list(c echo.Context, targetType ai.ScoreTargetType) error {
	id := c.Param("id")
	items, err := h.svc.ListByTarget(c.Request().Context(), targetType, id)
	if err != nil {
		return httperr.Write(c, err)
	}
	resp := make([]dto.PlaybookResponse, len(items))
	for i, p := range items {
		resp[i] = dto.ToPlaybookResponse(p)
	}
	return c.JSON(http.StatusOK, dto.PlaybookListResponse{Items: resp})
}

// Get handles GET /api/playbooks/:id
func (h *PlaybookHandler) Get(c echo.Context) error {
	id := c.Param("id")
	pb, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToPlaybookResponse(*pb))
}
