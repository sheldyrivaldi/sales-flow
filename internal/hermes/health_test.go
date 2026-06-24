package hermes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newHealthServer membuat test server yang melayani /health dan /v1/capabilities.
func newHealthServer(t *testing.T, caps Capabilities) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/v1/capabilities":
			_ = json.NewEncoder(w).Encode(caps)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, "path tidak dikenal: %s", r.URL.Path)
		}
	}))
}

// TestHealth_Success verifikasi: Capabilities ter-parse dengan benar.
func TestHealth_Success(t *testing.T) {
	want := Capabilities{
		Version:  "1.2.3",
		Models:   []string{"gpt-4o", "claude-3"},
		Features: []string{"chat", "memory", "tools"},
	}
	srv := newHealthServer(t, want)
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	caps, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health error: %v", err)
	}
	if caps.Version != "1.2.3" {
		t.Errorf("caps.Version = %q, want %q", caps.Version, "1.2.3")
	}
	if len(caps.Models) == 0 {
		t.Error("caps.Models kosong")
	}
	if caps.Models[0] != "gpt-4o" {
		t.Errorf("caps.Models[0] = %q, want %q", caps.Models[0], "gpt-4o")
	}
}

// TestHealth_LivenessFailure verifikasi: /health status 503 → error.
func TestHealth_LivenessFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprint(w, "service unavailable")
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error untuk /health 503, got nil")
	}
	if !containsStr(err.Error(), "503") {
		t.Errorf("error tidak mengandung status code 503: %v", err)
	}
}

// TestHealth_ServerDown verifikasi: server tidak tersedia → error, tidak panic.
func TestHealth_ServerDown(t *testing.T) {
	c := newTestClient("http://127.0.0.1:1", "testkey")
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error saat server mati, got nil")
	}
}

// TestHealth_BadStatus verifikasi: /health status != "ok" → error.
func TestHealth_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "degraded"})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error untuk status=degraded, got nil")
	}
	if !containsStr(err.Error(), "degraded") {
		t.Errorf("error tidak mengandung 'degraded': %v", err)
	}
}
