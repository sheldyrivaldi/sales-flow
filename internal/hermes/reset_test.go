package hermes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"salespilot/internal/config"
)

func newTestConfigForReset(baseURL string) *config.Config {
	return &config.Config{
		HermesBaseURL:       baseURL,
		APIServerKey:        "test-key",
		WorkspaceSessionKey: "sk",
		Port:                "8080",
		DatabaseURL:         "dummy",
		JWTSecret:           "dummy",
		SalesMCPToken:       "dummy",
		SeedAdminEmail:      "dummy",
		SeedAdminPassword:   "dummy",
	}
}

func TestResetMemory_Success(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotSessionKey string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotSessionKey = r.Header.Get("X-Hermes-Session-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := New(newTestConfigForReset(srv.URL))

	if err := c.ResetMemory(context.Background(), "sk-workspace"); err != nil {
		t.Fatalf("ResetMemory error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q; want POST", gotMethod)
	}
	if gotPath != "/admin/reset-memory" {
		t.Errorf("path = %q; want /admin/reset-memory", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q; want Bearer test-key", gotAuth)
	}
	if gotSessionKey != "sk-workspace" {
		t.Errorf("X-Hermes-Session-Key = %q; want sk-workspace", gotSessionKey)
	}
}

func TestResetMemory_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := New(newTestConfigForReset(srv.URL))

	err := c.ResetMemory(context.Background(), "sk-workspace")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error harus mengandung '401', got: %v", err)
	}
}
