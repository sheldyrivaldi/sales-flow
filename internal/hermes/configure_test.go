package hermes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"salespilot/internal/config"
)

func TestConfigure_Success(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := New(&config.Config{
		HermesBaseURL:       srv.URL,
		APIServerKey:        "test-key",
		WorkspaceSessionKey: "sk",
		Port:                "8080",
		DatabaseURL:         "dummy",
		JWTSecret:           "dummy",
		SalesMCPToken:       "dummy",
		SeedAdminEmail:      "dummy",
		SeedAdminPassword:   "dummy",
	})

	baseURL := "https://openrouter.ai/api/v1"
	err := c.Configure(context.Background(), ProviderConfig{
		Provider: "openrouter",
		Model:    "anthropic/claude-sonnet-4-6",
		BaseURL:  &baseURL,
		APIKey:   "or-key",
	})
	if err != nil {
		t.Fatalf("Configure error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q; want POST", gotMethod)
	}
	if gotPath != "/admin/config" {
		t.Errorf("path = %q; want /admin/config", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q; want Bearer test-key", gotAuth)
	}
	if gotBody["provider"] != "openrouter" {
		t.Errorf("body.provider = %v; want openrouter", gotBody["provider"])
	}
	if gotBody["model"] != "anthropic/claude-sonnet-4-6" {
		t.Errorf("body.model = %v", gotBody["model"])
	}
	if gotBody["api_key"] != "or-key" {
		t.Errorf("body.api_key = %v; want or-key", gotBody["api_key"])
	}
}

func TestConfigure_NilBaseURL(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := New(&config.Config{
		HermesBaseURL:       srv.URL,
		APIServerKey:        "key",
		WorkspaceSessionKey: "sk",
		Port:                "8080",
		DatabaseURL:         "dummy",
		JWTSecret:           "dummy",
		SalesMCPToken:       "dummy",
		SeedAdminEmail:      "dummy",
		SeedAdminPassword:   "dummy",
	})

	err := c.Configure(context.Background(), ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		BaseURL:  nil,
		APIKey:   "sk-key",
	})
	if err != nil {
		t.Fatalf("Configure error: %v", err)
	}
	// base_url harus null (bukan string kosong) supaya bridge pakai default.
	if v, ok := gotBody["base_url"]; ok && v != nil {
		t.Errorf("base_url = %v; want null/nil", v)
	}
}

func TestConfigure_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := New(&config.Config{
		HermesBaseURL:       srv.URL,
		APIServerKey:        "wrong",
		WorkspaceSessionKey: "sk",
		Port:                "8080",
		DatabaseURL:         "dummy",
		JWTSecret:           "dummy",
		SalesMCPToken:       "dummy",
		SeedAdminEmail:      "dummy",
		SeedAdminPassword:   "dummy",
	})

	err := c.Configure(context.Background(), ProviderConfig{Provider: "openai", Model: "gpt-4o", APIKey: "k"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error harus mengandung '401', got: %v", err)
	}
}
