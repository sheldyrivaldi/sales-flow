package handlers

import (
	"io"
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

// maxPlaybookDocBytes caps the uploaded source document (same 10 MB cap as
// Company Profile PDF ingest).
const maxPlaybookDocBytes = 10 * 1024 * 1024

// GenerateTenderFromDocument handles POST /api/tenders/:id/playbook/from-document (multipart).
func (h *PlaybookHandler) GenerateTenderFromDocument(c echo.Context) error {
	return h.generateFromDocument(c, ai.ScoreTargetTender)
}

// GenerateProspectFromDocument handles POST /api/prospects/:id/playbook/from-document (multipart).
func (h *PlaybookHandler) GenerateProspectFromDocument(c echo.Context) error {
	return h.generateFromDocument(c, ai.ScoreTargetProspect)
}

func (h *PlaybookHandler) generateFromDocument(c echo.Context, targetType ai.ScoreTargetType) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("FILE_REQUIRED", "unggah satu file PDF pada field 'file'"))
	}
	if fh.Size > maxPlaybookDocBytes {
		return httperr.Write(c, httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB"))
	}
	f, err := fh.Open()
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("FILE_UNREADABLE", "berkas tidak bisa dibaca"))
	}
	defer func() { _ = f.Close() }()
	pdfBytes, err := io.ReadAll(io.LimitReader(f, maxPlaybookDocBytes+1))
	if err != nil || int64(len(pdfBytes)) > maxPlaybookDocBytes {
		return httperr.Write(c, httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB"))
	}

	pb, err := h.svc.GenerateFromDocument(c.Request().Context(), targetType, c.Param("id"), pdfBytes, fh.Filename)
	if err != nil {
		return httperr.Write(c, err)
	}

	var actor string
	if u, ok := auth.UserFromContext(c); ok {
		actor = u.ID
	}
	writeAIAuditEvent(c.Request().Context(), h.audit, actor, "ai.playbook_from_document", pb.TargetType, pb.TargetID, map[string]any{
		"filename": fh.Filename,
		"version":  pb.Version,
	})

	return c.JSON(http.StatusOK, dto.ToPlaybookResponse(*pb))
}

// Refine handles POST /api/playbooks/:id/refine — merevisi playbook dengan
// instruksi bebas user; hasil dipersist sebagai versi baru.
func (h *PlaybookHandler) Refine(c echo.Context) error {
	var req dto.PlaybookRefineRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	pb, err := h.svc.Refine(c.Request().Context(), c.Param("id"), req.Instruction)
	if err != nil {
		return httperr.Write(c, err)
	}

	var actor string
	if u, ok := auth.UserFromContext(c); ok {
		actor = u.ID
	}
	writeAIAuditEvent(c.Request().Context(), h.audit, actor, "ai.playbook_refine", pb.TargetType, pb.TargetID, map[string]any{
		"version": pb.Version,
	})

	return c.JSON(http.StatusOK, dto.ToPlaybookResponse(*pb))
}

// GenerateEvent handles POST /api/events/:id/playbook
func (h *PlaybookHandler) GenerateEvent(c echo.Context) error {
	return h.generate(c, ai.ScoreTargetEvent)
}

// GenerateEventFromDocument handles POST /api/events/:id/playbook/from-document (multipart).
func (h *PlaybookHandler) GenerateEventFromDocument(c echo.Context) error {
	return h.generateFromDocument(c, ai.ScoreTargetEvent)
}

// ListEventPlaybooks handles GET /api/events/:id/playbooks
func (h *PlaybookHandler) ListEventPlaybooks(c echo.Context) error {
	return h.list(c, ai.ScoreTargetEvent)
}

// GenerateCustom handles POST /api/playbooks/custom — playbook mandiri dari
// topik bebas (menu Playbooks).
func (h *PlaybookHandler) GenerateCustom(c echo.Context) error {
	var req dto.PlaybookCustomRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	pb, err := h.svc.GenerateCustom(c.Request().Context(), req.Topic)
	if err != nil {
		return httperr.Write(c, err)
	}

	var actor string
	if u, ok := auth.UserFromContext(c); ok {
		actor = u.ID
	}
	writeAIAuditEvent(c.Request().Context(), h.audit, actor, "ai.playbook_custom", pb.TargetType, pb.TargetID, map[string]any{
		"version": pb.Version,
	})

	return c.JSON(http.StatusOK, dto.ToPlaybookResponse(*pb))
}

// ListCustom handles GET /api/playbooks/custom — versi terbaru tiap playbook
// custom, terbaru dulu.
func (h *PlaybookHandler) ListCustom(c echo.Context) error {
	items, err := h.svc.ListCustom(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	resp := make([]dto.PlaybookResponse, len(items))
	for i, p := range items {
		resp[i] = dto.ToPlaybookResponse(p)
	}
	return c.JSON(http.StatusOK, dto.PlaybookListResponse{Items: resp})
}
