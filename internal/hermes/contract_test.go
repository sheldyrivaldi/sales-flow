//go:build contract

package hermes

import (
	"context"
	"os"
	"testing"
	"time"

	"salespilot/internal/config"
)

// contractClient membuat Client yang mengarah ke bridge nyata.
// Test di-skip rapi bila HERMES_BASE_URL tidak diset.
func contractClient(t *testing.T) (Client, SessionKey) {
	t.Helper()
	base := os.Getenv("HERMES_BASE_URL")
	if base == "" {
		t.Skip("HERMES_BASE_URL tidak diset; lewati contract test")
	}
	apiKey := os.Getenv("API_SERVER_KEY")
	sk := SessionKey(os.Getenv("WORKSPACE_SESSION_KEY"))
	c := New(&config.Config{
		HermesBaseURL:       base,
		APIServerKey:        apiKey,
		WorkspaceSessionKey: string(sk),
		// Field lain diisi dummy agar validate() tidak fatal.
		Port:              "8080",
		DatabaseURL:       "dummy",
		JWTSecret:         "dummy",
		SalesMCPToken:     "dummy",
		SeedAdminEmail:    "dummy",
		SeedAdminPassword: "dummy",
	})
	return c, sk
}

// TestContract_Health verifikasi GET /health + /v1/capabilities terhadap bridge nyata.
func TestContract_Health(t *testing.T) {
	c, _ := contractClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	caps, err := c.Health(ctx)
	if err != nil {
		t.Fatalf("Health error: %v", err)
	}
	if caps.Version == "" {
		t.Error("caps.Version kosong")
	}
	if len(caps.Models) == 0 {
		t.Error("caps.Models kosong")
	}
	t.Logf("Health ok: version=%s models=%v features=%v", caps.Version, caps.Models, caps.Features)
}

// TestContract_ChatStream verifikasi POST /v1/chat/completions stream=true terhadap bridge.
func TestContract_ChatStream(t *testing.T) {
	c, sk := contractClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ch, err := c.ChatStream(ctx, ChatRequest{
		Messages:   []Message{{Role: "user", Content: "Halo, balas satu kalimat singkat."}},
		SessionKey: sk,
	})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	var deltaCount int
	var gotDone bool
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		if chunk.Done {
			gotDone = true
			break
		}
		if chunk.Delta != "" {
			deltaCount++
		}
	}

	if deltaCount == 0 {
		t.Error("tidak ada delta diterima dari stream")
	}
	if !gotDone {
		t.Error("stream tidak diakhiri chunk Done")
	}
	t.Logf("ChatStream ok: %d delta(s) diterima", deltaCount)
}

// TestContract_GenerateJSON verifikasi POST /v1/responses terhadap bridge.
func TestContract_GenerateJSON(t *testing.T) {
	c, sk := contractClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	type sample struct {
		OK bool `json:"ok"`
	}
	target := &sample{}
	raw, err := c.GenerateJSON(ctx, `Balas JSON persis ini: {"ok": true}`, target, sk)
	if err != nil {
		t.Fatalf("GenerateJSON error: %v", err)
	}
	if !target.OK {
		t.Errorf("target.OK = false, raw = %s", raw)
	}
	t.Logf("GenerateJSON ok: raw=%s", raw)
}
