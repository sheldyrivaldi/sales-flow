// Package mcp exposes SalesFlow's data as MCP (Model Context Protocol)
// tools over HTTP at /mcp, so Hermes can read sales data and (for a small
// whitelisted set) propose actions. See task.plan.md Pola P-9.
package mcp

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// Deps holds the services/repositories the MCP tools query and mutate. It is
// constructed once in internal/http/router.go, reusing the same service
// instances wired for the REST API (P-9: read tools query the service layer,
// not repos directly, so validation/defaults/hooks stay consistent).
type Deps struct {
	Tender   *service.TenderService
	Event    *service.EventService
	Prospect *service.ProspectService
	Profile  *service.ProfileService

	// ProspectRepo backs get_pipeline_summary/get_revenue_summary, which
	// need an aggregation (SummaryByStage) not exposed on ProspectService.
	ProspectRepo domain.ProspectRepository

	Audit    domain.AuditRepository
	Playbook domain.PlaybookDraftRepository

	// PlaybookJob backs the callback-via-MCP flow: Hermes writes a finished
	// (or failed) playbook back into a job row it was handed, so the app never
	// holds a long connection waiting for generation (lihat save_playbook_job).
	PlaybookJob domain.PlaybookJobRepository
}

// NewServer builds the MCP server and registers all read/write tools.
// Tool names and schemas are the stable contract in
// deploy/hermes/config.yaml.example (mcp_servers.sales.tools.include) —
// changes here must stay additive (P-9: "aditif, jangan rename").
func NewServer(deps Deps) *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer(
		"salespilot",
		"1.0.0",
		mcpserver.WithToolCapabilities(false),
		mcpserver.WithRecovery(),
	)

	registerReadTools(s, deps)
	registerWriteTools(s, deps)

	return s
}

// Handler wraps srv as a StreamableHTTP http.Handler mountable at /mcp.
func Handler(srv *mcpserver.MCPServer) http.Handler {
	return mcpserver.NewStreamableHTTPServer(srv)
}

// BearerAuth requires "Authorization: Bearer <token>" using a constant-time
// comparison. Required per docs/architecture.md: "/mcp: Authenticated with
// bearer token (SALES_MCP_TOKEN); constant-time comparison prevents timing
// attacks."
func BearerAuth(token string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				return httperr.Write(c, httperr.NewUnauthorized("token tidak ditemukan atau format salah"))
			}
			supplied := strings.TrimPrefix(header, prefix)
			if subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				return httperr.Write(c, httperr.NewUnauthorized("token tidak valid"))
			}
			return next(c)
		}
	}
}
