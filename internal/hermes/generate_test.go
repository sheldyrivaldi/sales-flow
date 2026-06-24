package hermes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// sampleSchema dipakai sebagai target unmarshal di test GenerateJSON.
type sampleSchema struct {
	OK bool `json:"ok"`
}

// TestGenerateJSON_Success verifikasi: JSON valid dikembalikan tanpa error,
// header auth & session key terkirim ke bridge.
func TestGenerateJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testkey" {
			t.Errorf("Authorization header salah: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Hermes-Session-Key") != "sk-test" {
			t.Errorf("X-Hermes-Session-Key salah: %q", r.Header.Get("X-Hermes-Session-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireResponsesResp{OutputText: `{"ok":true}`})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	target := &sampleSchema{}
	raw, err := c.GenerateJSON(context.Background(), "Balas JSON {\"ok\":true}", target, "sk-test")
	if err != nil {
		t.Fatalf("GenerateJSON error: %v", err)
	}
	if !target.OK {
		t.Errorf("target.OK = false, want true")
	}
	if string(raw) != `{"ok":true}` {
		t.Errorf("raw = %s, want {\"ok\":true}", raw)
	}
}

// TestGenerateJSON_StripFence verifikasi fence ```json ... ``` dibuang otomatis.
func TestGenerateJSON_StripFence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fenced := "```json\n{\"ok\":true}\n```"
		_ = json.NewEncoder(w).Encode(wireResponsesResp{OutputText: fenced})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	target := &sampleSchema{}
	raw, err := c.GenerateJSON(context.Background(), "prompt", target, "")
	if err != nil {
		t.Fatalf("GenerateJSON (fence) error: %v", err)
	}
	if !target.OK {
		t.Errorf("target.OK = false setelah strip fence")
	}
	_ = raw
}

// TestGenerateJSON_NonOKStatus verifikasi non-2xx → error berisi status code.
func TestGenerateJSON_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "internal server error")
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	_, err := c.GenerateJSON(context.Background(), "prompt", &sampleSchema{}, "")
	if err == nil {
		t.Fatal("expected error untuk status 500, got nil")
	}
	if !containsStr(err.Error(), "500") {
		t.Errorf("error tidak mengandung status code: %v", err)
	}
}

// TestGenerateJSON_SchemaNil verifikasi schema nil → kembalikan raw tanpa unmarshal.
func TestGenerateJSON_SchemaNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireResponsesResp{OutputText: `{"x":42}`})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	raw, err := c.GenerateJSON(context.Background(), "prompt", nil, "")
	if err != nil {
		t.Fatalf("GenerateJSON (nil schema) error: %v", err)
	}
	if string(raw) != `{"x":42}` {
		t.Errorf("raw = %s, want {\"x\":42}", raw)
	}
}

// TestGenerateJSON_RetryRecovers verifikasi: jika percobaan pertama mengembalikan
// non-JSON, percobaan kedua mengembalikan JSON valid → sukses, dipanggil 2x.
func TestGenerateJSON_RetryRecovers(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			// Percobaan pertama: teks biasa (bukan JSON valid untuk unmarshal ke sampleSchema).
			_ = json.NewEncoder(w).Encode(wireResponsesResp{OutputText: "Berikut jawabannya: bukan JSON"})
		} else {
			_ = json.NewEncoder(w).Encode(wireResponsesResp{OutputText: `{"ok":true}`})
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	target := &sampleSchema{}
	raw, err := c.GenerateJSON(context.Background(), "prompt", target, "")
	if err != nil {
		t.Fatalf("GenerateJSON (retry) error: %v", err)
	}
	if !target.OK {
		t.Errorf("target.OK = false setelah retry")
	}
	_ = raw
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("callCount = %d, want 2", atomic.LoadInt32(&callCount))
	}
}

// TestGenerateJSON_RetryFails verifikasi: jika kedua percobaan gagal → error jelas
// berisi cuplikan output, tidak panic.
func TestGenerateJSON_RetryFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireResponsesResp{OutputText: "ini bukan JSON sama sekali"})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "testkey")
	_, err := c.GenerateJSON(context.Background(), "prompt", &sampleSchema{}, "")
	if err == nil {
		t.Fatal("expected error setelah retry gagal, got nil")
	}
	if !containsStr(err.Error(), "setelah retry") {
		t.Errorf("error tidak mengandung kata 'setelah retry': %v", err)
	}
}

// --- Unit test stripJSONFence ---

func TestStripJSONFence(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{`{"x":1}`, `{"x":1}`},
		{"```json\n{\"x\":1}\n```", `{"x":1}`},
		{"```\n{\"x\":1}\n```", `{"x":1}`},
		{"  ```json\n{\"x\":1}\n```  ", `{"x":1}`},
	}
	for _, tc := range cases {
		got := stripJSONFence(tc.input)
		if got != tc.want {
			t.Errorf("stripJSONFence(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
