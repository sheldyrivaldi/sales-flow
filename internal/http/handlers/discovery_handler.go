package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
)

type DiscoveryHandler struct {
	discovery *service.DiscoveryService
	tenders   *service.TenderService
}

func NewDiscoveryHandler(discovery *service.DiscoveryService, tenders *service.TenderService) *DiscoveryHandler {
	return &DiscoveryHandler{discovery: discovery, tenders: tenders}
}

// Run handles POST /api/discovery/run — starts (or, when correlation_key
// matches an in-flight/finished run, reuses — EP-12 ST-12.3.2 idempotency) a
// discovery run and returns immediately; the crawl/score pipeline itself
// executes in the background (DiscoveryService.RunAsync).
func (h *DiscoveryHandler) Run(c echo.Context) error {
	var req dto.DiscoveryRunRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}

	run, err := h.discovery.RunAsync(c.Request().Context(), req.CorrelationKey)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusAccepted, dto.ToDiscoveryRunResponse(*run))
}

// ListRuns handles GET /api/discovery/runs
func (h *DiscoveryHandler) ListRuns(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	runs, total, err := h.discovery.ListRuns(c.Request().Context(), page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.DiscoveryRunResponse, len(runs))
	for i, r := range runs {
		items[i] = dto.ToDiscoveryRunResponse(r)
	}

	return c.JSON(http.StatusOK, dto.DiscoveryRunListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Inbox handles GET /api/discovery/inbox — discovery-origin tenders awaiting
// human review (reuses TenderService.List with the OnlyInbox filter, so it
// stays in lockstep with the same status-transition rules as the rest of the
// tender list). Supports the two highest-value Discovery Inbox filters
// (Design §4.3): recommended_action and min_score.
func (h *DiscoveryHandler) Inbox(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	f := domain.TenderFilter{OnlyInbox: true}
	if v := c.QueryParam("recommended_action"); v != "" {
		a := domain.RecommendedAction(v)
		f.RecommendedAction = &a
	}
	if v := c.QueryParam("min_score"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinFitScore = &n
		}
	}

	tenders, total, err := h.tenders.List(c.Request().Context(), f, page, pageSize)
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
