package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Summary handles GET /api/dashboard/summary
func (h *DashboardHandler) Summary(c echo.Context) error {
	s, err := h.svc.Summary(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}

	pipeline := make([]dto.PipelineStageResponse, len(s.Pipeline))
	for i, p := range s.Pipeline {
		pipeline[i] = dto.PipelineStageResponse{
			Stage:      string(p.Stage),
			Count:      p.Count,
			TotalValue: p.TotalValue,
		}
	}

	priority := make([]dto.TenderResponse, len(s.PriorityTenders))
	for i, t := range s.PriorityTenders {
		priority[i] = dto.ToTenderResponse(t)
	}

	return c.JSON(http.StatusOK, dto.DashboardSummaryResponse{
		Pipeline:            pipeline,
		TotalPipelineCount:  s.TotalPipelineCount,
		TotalPipelineValue:  s.TotalPipelineValue,
		PriorityTenders:     priority,
		DiscoveryTodayCount: s.DiscoveryTodayCount,
	})
}
