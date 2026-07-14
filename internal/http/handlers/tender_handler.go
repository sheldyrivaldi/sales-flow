package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
	"salespilot/internal/telemetry"
)

type TenderHandler struct {
	svc   *service.TenderService
	audit domain.AuditRepository
	emit  *telemetry.Emitter
}

func NewTenderHandler(svc *service.TenderService, audit domain.AuditRepository) *TenderHandler {
	return &TenderHandler{svc: svc, audit: audit}
}

// SetEmitter wires telemetry (EP-17 ST-17.1) after construction — optional,
// nil-safe.
func (h *TenderHandler) SetEmitter(e *telemetry.Emitter) { h.emit = e }

// List handles GET /api/tenders
func (h *TenderHandler) List(c echo.Context) error {
	f := domain.TenderFilter{}

	if v := c.QueryParam("status"); v != "" {
		s := domain.TenderStatus(v)
		f.Status = &s
	}
	if v := c.QueryParam("recommended_action"); v != "" {
		a := domain.RecommendedAction(v)
		f.RecommendedAction = &a
	}
	if v := c.QueryParam("origin"); v != "" {
		o := domain.TenderOrigin(v)
		f.Origin = &o
	}
	f.BuyerName = c.QueryParam("buyer")
	if v := c.QueryParam("deadline_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.DeadlineFrom = &t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DeadlineFrom = &t
		}
	}
	if v := c.QueryParam("deadline_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.DeadlineTo = &t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DeadlineTo = &t
		}
	}
	f.Search = c.QueryParam("search")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	tenders, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.TenderResponse, len(tenders))
	for i, t := range tenders {
		items[i] = dto.ToTenderResponse(t)
	}

	return c.JSON(http.StatusOK, dto.TenderListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Get handles GET /api/tenders/:id
func (h *TenderHandler) Get(c echo.Context) error {
	id := c.Param("id")
	t, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// Create handles POST /api/tenders
func (h *TenderHandler) Create(c echo.Context) error {
	var req dto.TenderCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	t, err := h.svc.Create(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToTenderResponse(*t))
}

// Update handles PUT /api/tenders/:id
func (h *TenderHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.TenderUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	t, err := h.svc.Update(c.Request().Context(), id, &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// Delete handles DELETE /api/tenders/:id
func (h *TenderHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateStatus handles PATCH /api/tenders/:id/status
func (h *TenderHandler) UpdateStatus(c echo.Context) error {
	id := c.Param("id")

	var req dto.TenderStatusRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	t, err := h.svc.UpdateStatus(c.Request().Context(), id, domain.TenderStatus(req.Status))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// Promote handles POST /api/tenders/:id/promote — this is the Discovery
// Inbox "Pursue" action (EP-12), tracked as the review_pursue metric
// (EP-17 ST-17.1).
func (h *TenderHandler) Promote(c echo.Context) error {
	id := c.Param("id")
	t, err := h.svc.Promote(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}

	if h.emit != nil {
		var actor string
		if u, ok := auth.UserFromContext(c); ok {
			actor = u.ID
		}
		h.emit.Emit(c.Request().Context(), "review_pursue", map[string]any{"tender_id": t.ID, "actor": actor})
	}

	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// Review handles POST /api/tenders/:id/review — Discovery Inbox "Watchlist"
// (reason omitted) / "Tolak" (reason filled) actions (EP-12 ST-12.7): marks a
// discovery-origin tender as reviewed (removing it from the inbox) without
// promoting it. The reason, if any, is written to audit_log AND notifies the
// learning hook (EP-16 TK-16.2.1, inside TenderService.Review) — best-effort,
// mirrors the writeAudit/writeSourceAudit pattern already used in
// internal/mcp and internal/ai/discovery.go.
func (h *TenderHandler) Review(c echo.Context) error {
	id := c.Param("id")

	var req dto.TenderReviewRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}

	reason := ""
	if req.Reason != nil {
		reason = *req.Reason
	}

	t, err := h.svc.Review(c.Request().Context(), id, reason)
	if err != nil {
		return httperr.Write(c, err)
	}

	writeDiscoveryReviewAudit(c.Request().Context(), h.audit, t.ID, req.Reason)

	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}

// writeDiscoveryReviewAudit best-effort records a discovery_review audit_log
// row. A failure here must not fail the request — the tender's reviewed_at
// state (the part that actually matters for the inbox) is already committed
// by the time this runs.
func writeDiscoveryReviewAudit(ctx context.Context, audit domain.AuditRepository, tenderID string, reason *string) {
	if audit == nil {
		return
	}
	reasonText := ""
	if reason != nil {
		reasonText = *reason
	}
	payloadJSON, err := json.Marshal(map[string]any{"reason": reasonText})
	if err != nil {
		log.Printf("tender: AUDIT FAILURE: marshal payload untuk discovery_review (tender=%s): %v", tenderID, err)
		return
	}
	targetType := "tender"
	e := &domain.AuditEvent{
		Actor:      "user",
		Action:     "discovery_review",
		TargetType: &targetType,
		TargetID:   &tenderID,
		Payload:    payloadJSON,
	}
	if err := audit.Create(ctx, e); err != nil {
		log.Printf("tender: AUDIT FAILURE: gagal menulis audit_log discovery_review (tender=%s): %v", tenderID, err)
	}
}

// RecordOutcome handles POST /api/tenders/:id/outcome
func (h *TenderHandler) RecordOutcome(c echo.Context) error {
	id := c.Param("id")

	var req dto.TenderOutcomeRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	t, err := h.svc.RecordOutcome(c.Request().Context(), id, domain.OutcomeResult(req.Result), notes)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToTenderResponse(*t))
}
