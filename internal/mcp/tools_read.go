package mcp

import (
	"context"
	"errors"
	"log"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
)

// registerReadTools registers the 10 read-only tools from
// deploy/hermes/config.yaml.example. Each queries the same service layer
// used by the REST API and returns concise JSON (P-9).
func registerReadTools(s *mcpserver.MCPServer, deps Deps) {
	s.AddTool(mcpgo.NewTool("list_tenders",
		mcpgo.WithDescription("Daftar tender dengan filter opsional (status, buyer_name, recommended_action, origin, search) dan paginasi."),
		mcpgo.WithString("status", mcpgo.Description("Filter status: IDENTIFIED, QUALIFYING, BIDDING, SUBMITTED, WON, LOST")),
		mcpgo.WithString("buyer_name", mcpgo.Description("Filter substring nama buyer")),
		mcpgo.WithString("recommended_action", mcpgo.Description("Filter aksi rekomendasi AI: PURSUE, REVIEW, WATCHLIST, REJECT, NEED_PARTNER")),
		mcpgo.WithString("origin", mcpgo.Description("Filter asal: manual, discovery")),
		mcpgo.WithString("search", mcpgo.Description("Pencarian substring pada title/buyer_name")),
		mcpgo.WithNumber("page", mcpgo.Description("Nomor halaman (default 1)"), mcpgo.DefaultNumber(1)),
		mcpgo.WithNumber("page_size", mcpgo.Description("Ukuran halaman (default 20, maks 500)"), mcpgo.DefaultNumber(20)),
	), listTendersHandler(deps))

	s.AddTool(mcpgo.NewTool("get_tender",
		mcpgo.WithDescription("Ambil satu tender berdasarkan id."),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("ID tender (UUID)")),
	), getTenderHandler(deps))

	s.AddTool(mcpgo.NewTool("search_tenders",
		mcpgo.WithDescription("Cari tender berdasarkan substring pada title/buyer_name (bukan full-text search)."),
		mcpgo.WithString("q", mcpgo.Required(), mcpgo.Description("Kata kunci pencarian")),
		mcpgo.WithNumber("page", mcpgo.Description("Nomor halaman (default 1)"), mcpgo.DefaultNumber(1)),
		mcpgo.WithNumber("page_size", mcpgo.Description("Ukuran halaman (default 20, maks 500)"), mcpgo.DefaultNumber(20)),
	), searchTendersHandler(deps))

	s.AddTool(mcpgo.NewTool("list_events",
		mcpgo.WithDescription("Daftar event dengan filter opsional (type, status, search) dan paginasi."),
		mcpgo.WithString("type", mcpgo.Description("Filter tipe: EXPO, CONFERENCE, SEMINAR, WORKSHOP, NETWORKING, OTHER")),
		mcpgo.WithString("status", mcpgo.Description("Filter status: PLANNED, ATTENDED, CANCELLED")),
		mcpgo.WithString("search", mcpgo.Description("Pencarian substring pada name/organizer")),
		mcpgo.WithNumber("page", mcpgo.Description("Nomor halaman (default 1)"), mcpgo.DefaultNumber(1)),
		mcpgo.WithNumber("page_size", mcpgo.Description("Ukuran halaman (default 20, maks 500)"), mcpgo.DefaultNumber(20)),
	), listEventsHandler(deps))

	s.AddTool(mcpgo.NewTool("get_event",
		mcpgo.WithDescription("Ambil satu event berdasarkan id."),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("ID event (UUID)")),
	), getEventHandler(deps))

	s.AddTool(mcpgo.NewTool("list_prospects",
		mcpgo.WithDescription("Daftar prospek dengan filter opsional (stage, owner_user_id, source_type, search) dan paginasi."),
		mcpgo.WithString("stage", mcpgo.Description("Filter stage: NEW, QUALIFIED, ENGAGED, PROPOSAL, WON, LOST")),
		mcpgo.WithString("owner_user_id", mcpgo.Description("Filter ID pemilik prospek")),
		mcpgo.WithString("source_type", mcpgo.Description("Filter sumber: manual, event, tender")),
		mcpgo.WithString("search", mcpgo.Description("Pencarian substring pada name/company")),
		mcpgo.WithNumber("page", mcpgo.Description("Nomor halaman (default 1)"), mcpgo.DefaultNumber(1)),
		mcpgo.WithNumber("page_size", mcpgo.Description("Ukuran halaman (default 20, maks 500)"), mcpgo.DefaultNumber(20)),
	), listProspectsHandler(deps))

	s.AddTool(mcpgo.NewTool("get_prospect",
		mcpgo.WithDescription("Ambil satu prospek berdasarkan id."),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("ID prospek (UUID)")),
	), getProspectHandler(deps))

	s.AddTool(mcpgo.NewTool("get_pipeline_summary",
		mcpgo.WithDescription("Ringkasan pipeline prospek: jumlah & total nilai (est_value) per stage."),
	), getPipelineSummaryHandler(deps))

	s.AddTool(mcpgo.NewTool("get_revenue_summary",
		mcpgo.WithDescription("Ringkasan revenue prospek: total nilai WON, pipeline terbuka (NEW..PROPOSAL), dan LOST."),
	), getRevenueSummaryHandler(deps))

	s.AddTool(mcpgo.NewTool("get_company_profile",
		mcpgo.WithDescription("Ambil Company Profile (Otak Agent) versi terkini: kapabilitas, target, no-go rules, keywords. Mengembalikan template default bila belum dikonfigurasi."),
	), getCompanyProfileHandler(deps))
}

func pageArgs(req mcpgo.CallToolRequest) (int, int) {
	page := req.GetInt("page", 1)
	pageSize := req.GetInt("page_size", 20)
	return page, pageSize
}

func listTendersHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		var f domain.TenderFilter
		if v := req.GetString("status", ""); v != "" {
			status := domain.TenderStatus(v)
			if !status.Valid() {
				return mcpgo.NewToolResultError("status tidak valid"), nil
			}
			f.Status = &status
		}
		f.BuyerName = req.GetString("buyer_name", "")
		if v := req.GetString("recommended_action", ""); v != "" {
			action := domain.RecommendedAction(v)
			if !action.Valid() {
				return mcpgo.NewToolResultError("recommended_action tidak valid"), nil
			}
			f.RecommendedAction = &action
		}
		if v := req.GetString("origin", ""); v != "" {
			origin := domain.TenderOrigin(v)
			if !origin.Valid() {
				return mcpgo.NewToolResultError("origin tidak valid"), nil
			}
			f.Origin = &origin
		}
		f.Search = req.GetString("search", "")

		page, pageSize := pageArgs(req)
		items, total, err := deps.Tender.List(ctx, f, page, pageSize)
		if err != nil {
			return toolResultFromErr(err, "gagal mengambil daftar tender")
		}
		return mcpgo.NewToolResultJSON(listResult(items, total, page, pageSize))
	}
}

func getTenderHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcpgo.NewToolResultError("id wajib diisi"), nil
		}
		t, err := deps.Tender.Get(ctx, id)
		if err != nil {
			return toolResultFromErr(err, "tender tidak ditemukan")
		}
		return mcpgo.NewToolResultJSON(t)
	}
}

func searchTendersHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		q, err := req.RequireString("q")
		if err != nil {
			return mcpgo.NewToolResultError("q wajib diisi"), nil
		}
		page, pageSize := pageArgs(req)
		items, total, err := deps.Tender.List(ctx, domain.TenderFilter{Search: q}, page, pageSize)
		if err != nil {
			return toolResultFromErr(err, "gagal mencari tender")
		}
		return mcpgo.NewToolResultJSON(listResult(items, total, page, pageSize))
	}
}

func listEventsHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		var f domain.EventFilter
		if v := req.GetString("type", ""); v != "" {
			t := domain.EventType(v)
			if !t.Valid() {
				return mcpgo.NewToolResultError("type tidak valid"), nil
			}
			f.Type = &t
		}
		if v := req.GetString("status", ""); v != "" {
			st := domain.EventStatus(v)
			if !st.Valid() {
				return mcpgo.NewToolResultError("status tidak valid"), nil
			}
			f.Status = &st
		}
		f.Search = req.GetString("search", "")

		page, pageSize := pageArgs(req)
		items, total, err := deps.Event.List(ctx, f, page, pageSize)
		if err != nil {
			return toolResultFromErr(err, "gagal mengambil daftar event")
		}
		return mcpgo.NewToolResultJSON(listResult(items, total, page, pageSize))
	}
}

func getEventHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcpgo.NewToolResultError("id wajib diisi"), nil
		}
		e, err := deps.Event.Get(ctx, id)
		if err != nil {
			return toolResultFromErr(err, "event tidak ditemukan")
		}
		return mcpgo.NewToolResultJSON(e)
	}
}

func listProspectsHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		var f domain.ProspectFilter
		if v := req.GetString("stage", ""); v != "" {
			stage := domain.ProspectStage(v)
			if !stage.Valid() {
				return mcpgo.NewToolResultError("stage tidak valid"), nil
			}
			f.Stage = &stage
		}
		if v := req.GetString("owner_user_id", ""); v != "" {
			f.OwnerUserID = &v
		}
		if v := req.GetString("source_type", ""); v != "" {
			st := domain.ProspectSource(v)
			if !st.Valid() {
				return mcpgo.NewToolResultError("source_type tidak valid"), nil
			}
			f.SourceType = &st
		}
		f.Search = req.GetString("search", "")

		page, pageSize := pageArgs(req)
		items, total, err := deps.Prospect.List(ctx, f, page, pageSize)
		if err != nil {
			return toolResultFromErr(err, "gagal mengambil daftar prospek")
		}
		return mcpgo.NewToolResultJSON(listResult(items, total, page, pageSize))
	}
}

func getProspectHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcpgo.NewToolResultError("id wajib diisi"), nil
		}
		p, err := deps.Prospect.Get(ctx, id)
		if err != nil {
			return toolResultFromErr(err, "prospek tidak ditemukan")
		}
		return mcpgo.NewToolResultJSON(p)
	}
}

// pipelineSummary is the get_pipeline_summary output shape.
type pipelineSummary struct {
	ByStage    []domain.ProspectStageSummary `json:"by_stage"`
	TotalCount int64                         `json:"total_count"`
	TotalValue float64                       `json:"total_value"`
}

func getPipelineSummaryHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		rows, err := deps.ProspectRepo.SummaryByStage(ctx)
		if err != nil {
			return toolResultFromErr(err, "gagal mengambil ringkasan pipeline")
		}
		summary := pipelineSummary{ByStage: rows}
		for _, r := range rows {
			summary.TotalCount += r.Count
			summary.TotalValue += r.TotalValue
		}
		return mcpgo.NewToolResultJSON(summary)
	}
}

// revenueSummary is the get_revenue_summary output shape. Derived from
// ProspectStageSummary — WON/LOST are terminal, everything else is open
// pipeline. Prospect-based only (tender revenue is a future, additive
// extension — not part of this MVP).
type revenueSummary struct {
	WonValue          float64 `json:"won_value"`
	OpenPipelineValue float64 `json:"open_pipeline_value"`
	LostValue         float64 `json:"lost_value"`
	Currency          string  `json:"currency"`
}

func getRevenueSummaryHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		rows, err := deps.ProspectRepo.SummaryByStage(ctx)
		if err != nil {
			return toolResultFromErr(err, "gagal mengambil ringkasan revenue")
		}
		summary := revenueSummary{Currency: "IDR"}
		for _, r := range rows {
			switch r.Stage {
			case domain.ProspectStageWon:
				summary.WonValue += r.TotalValue
			case domain.ProspectStageLost:
				summary.LostValue += r.TotalValue
			default:
				summary.OpenPipelineValue += r.TotalValue
			}
		}
		return mcpgo.NewToolResultJSON(summary)
	}
}

func getCompanyProfileHandler(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		agg, err := deps.Profile.GetCurrent(ctx)
		if err != nil {
			return toolResultFromErr(err, "gagal mengambil company profile")
		}
		// Reuse the same DTO mapper as GET /api/profile so the MCP tool
		// returns the identical snake_case shape the REST API already
		// exposes, instead of the domain struct's untagged Go field names.
		return mcpgo.NewToolResultJSON(dto.ToProfileResponse(*agg))
	}
}

// listPage is the shared shape for all list_* tools.
type listPage[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func listResult[T any](items []T, total int64, page, pageSize int) listPage[T] {
	// service.List returns a nil slice (not empty) when nothing matches —
	// force it to a non-nil empty slice so the JSON result is "items":[]
	// rather than "items":null, keeping the shape consistent regardless of
	// whether the query matched anything.
	if items == nil {
		items = []T{}
	}
	return listPage[T]{Items: items, Total: total, Page: page, PageSize: pageSize}
}

// toolResultFromErr maps a service error to a tool result: NOT_FOUND-shaped
// httperr.APIError becomes its user-facing message (already safe to show);
// anything else is logged server-side and reduced to a generic friendly
// message — raw internal errors (DB text, stack-adjacent detail) must never
// reach the MCP client/agent, mirroring httperr.Write's behavior on the REST
// side. Neither path aborts the MCP call itself (errors are reported inside
// the result per mcp-go convention).
func toolResultFromErr(err error, friendlyMsg string) (*mcpgo.CallToolResult, error) {
	var apiErr *httperr.APIError
	if errors.As(err, &apiErr) {
		return mcpgo.NewToolResultError(apiErr.Message), nil
	}
	log.Printf("mcp: %s: %v", friendlyMsg, err)
	return mcpgo.NewToolResultError(friendlyMsg + ", coba lagi nanti"), nil
}
