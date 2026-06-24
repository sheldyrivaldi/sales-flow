package hermes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient membuat httpClient yang mengarah ke server test.
func newTestClient(baseURL, apiKey string) *httpClient {
	return &httpClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		hc:      &http.Client{},
	}
}

// --- TK-01.2.1: Chat non-stream ---

func TestChat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifikasi header
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Errorf("Authorization header salah: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Hermes-Session-Key") != "sessionkey" {
			t.Errorf("X-Hermes-Session-Key header salah: %q", r.Header.Get("X-Hermes-Session-Key"))
		}
		if r.Header.Get("X-Hermes-Session-Id") != "sess-123" {
			t.Errorf("X-Hermes-Session-Id header salah: %q", r.Header.Get("X-Hermes-Session-Id"))
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "Halo dari Hermes",
						"tool_calls": []map[string]any{
							{
								"id":   "tc-1",
								"type": "function",
								"function": map[string]any{
									"name":      "list_tenders",
									"arguments": json.RawMessage(`{"limit":10}`),
								},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	got, err := c.Chat(context.Background(), ChatRequest{
		Messages:   []Message{{Role: "user", Content: "halo"}},
		SessionKey: "sessionkey",
		SessionID:  "sess-123",
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if got.Content != "Halo dari Hermes" {
		t.Errorf("Content = %q, want %q", got.Content, "Halo dari Hermes")
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(got.ToolCalls))
	}
	if got.ToolCalls[0].Name != "list_tenders" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", got.ToolCalls[0].Name, "list_tenders")
	}
	if string(got.ToolCalls[0].Arguments) != `{"limit":10}` {
		t.Errorf("ToolCalls[0].Arguments = %s", got.ToolCalls[0].Arguments)
	}
}

// --- TK-01.2.3: Chat error → status non-2xx ---

func TestChat_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "internal server error from hermes")
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "key")
	_, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !containsStr(err.Error(), "500") {
		t.Errorf("error tidak mengandung status code: %v", err)
	}
}

// TK-01.2.3: Chat saat Hermes "mati" (port tertutup) → tidak panic, return error
func TestChat_ServerDown(t *testing.T) {
	c := newTestClient("http://127.0.0.1:1", "key") // port tidak ada
	_, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err == nil {
		t.Fatal("expected error saat server mati, got nil")
	}
}

// TK-01.2.3: Chat dengan context deadline → error deadline, tidak hang
func TestChat_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// server sengaja lambat
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newTestClient(srv.URL, "key")
	_, err := c.Chat(ctx, ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// --- TK-01.2.2: ChatStream SSE ---

func TestChatStream_Success(t *testing.T) {
	deltas := []string{"Halo", " dari", " stream"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifikasi header
		if r.Header.Get("Authorization") != "Bearer streamkey" {
			t.Errorf("Authorization header salah: %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("ResponseWriter tidak mendukung Flusher")
			return
		}

		for _, d := range deltas {
			payload := fmt.Sprintf(`{"choices":[{"delta":{"content":%q}}]}`, d)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "streamkey")
	ch, err := c.ChatStream(context.Background(), ChatRequest{
		Messages:   []Message{{Role: "user", Content: "halo"}},
		SessionKey: "sk",
		SessionID:  "id",
	})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	var got []string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		if chunk.Done {
			break
		}
		if chunk.Delta != "" {
			got = append(got, chunk.Delta)
		}
	}

	want := deltas
	if len(got) != len(want) {
		t.Fatalf("got %d deltas, want %d: %v", len(got), len(want), got)
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("delta[%d] = %q, want %q", i, g, want[i])
		}
	}
}

// TK-01.2.3: ChatStream kirim Chunk{Err} saat cancel
func TestChatStream_Cancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		// Kirim satu chunk lalu tunggu (simulasikan stream panjang)
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"awal\"}}]}\n\n")
		flusher.Flush()
		// Server tunggu — simulasikan stream belum selesai
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	c := newTestClient(srv.URL, "key")

	ch, err := c.ChatStream(ctx, ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	// Baca satu chunk lalu cancel
	first := <-ch
	if first.Err != nil {
		t.Fatalf("first chunk error: %v", first.Err)
	}
	cancel()

	// Drain channel setelah cancel — harus berakhir, tidak leak
	for range ch {
	}
}

// TK-01.2.3: ChatStream saat server mati → return error, tidak buka channel
func TestChatStream_ServerDown(t *testing.T) {
	c := newTestClient("http://127.0.0.1:1", "key")
	ch, err := c.ChatStream(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err == nil {
		// Bila channel dibuka, drain dulu
		if ch != nil {
			for range ch {
			}
		}
		t.Fatal("expected error saat server mati, got nil")
	}
}

// TK-01.2.3: ChatStream status non-2xx → return error sebelum channel
func TestChatStream_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "unauthorized")
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "key")
	_, err := c.ChatStream(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err == nil {
		t.Fatal("expected error untuk status 401, got nil")
	}
	if !containsStr(err.Error(), "401") {
		t.Errorf("error tidak mengandung status code: %v", err)
	}
}

// --- helper ---

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
