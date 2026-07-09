//go:build contract

package mcp_test

import (
	"context"
	"os"
	"testing"
	"time"

	"salespilot/internal/config"
	"salespilot/internal/hermes"
)

// TestContract_ChatTriggersListTenders verifies the full round-trip: a real
// Hermes chat request phrased as "tender prioritas?" should decide to call
// the SalesPilot MCP list_tenders tool. Requires the full stack running
// (API serving /mcp, hermes-bridge with ENABLED_TOOLSETS including mcp,
// config.yaml registering mcp_servers.sales). Skipped cleanly when
// HERMES_BASE_URL is unset, matching internal/hermes/contract_test.go.
func TestContract_ChatTriggersListTenders(t *testing.T) {
	base := os.Getenv("HERMES_BASE_URL")
	if base == "" {
		t.Skip("HERMES_BASE_URL tidak diset; lewati contract test MCP")
	}
	apiKey := os.Getenv("API_SERVER_KEY")
	sk := hermes.SessionKey(os.Getenv("WORKSPACE_SESSION_KEY"))

	c := hermes.New(&config.Config{
		HermesBaseURL:       base,
		APIServerKey:        apiKey,
		WorkspaceSessionKey: string(sk),
		// Field lain diisi dummy agar validate() tidak fatal (pola sama
		// internal/hermes/contract_test.go).
		Port:              "8080",
		DatabaseURL:       "dummy",
		JWTSecret:         "dummy",
		SalesMCPToken:     "dummy",
		SeedAdminEmail:    "dummy",
		SeedAdminPassword: "dummy",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := c.Chat(ctx, hermes.ChatRequest{
		Messages: []hermes.Message{
			{Role: "user", Content: "Tender prioritas apa yang harus saya kejar sekarang?"},
		},
		SessionKey: sk,
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	var calledListTenders bool
	for _, tc := range resp.ToolCalls {
		if tc.Name == "list_tenders" {
			calledListTenders = true
			break
		}
	}
	// No fallback on response content: the AC this test exists to prove is
	// specifically that the chat round-trip *calls* list_tenders via MCP —
	// a content mention of "tender" (e.g. the model answering from its own
	// knowledge, or apologizing that it has no data) is not evidence of
	// that and must not make this test pass.
	if !calledListTenders {
		t.Errorf("chat tidak memicu list_tenders: tool_calls=%v content=%q", resp.ToolCalls, resp.Content)
	}
	t.Logf("Chat ok: tool_calls=%v content=%q", resp.ToolCalls, resp.Content)
}
