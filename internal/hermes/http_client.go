package hermes

import (
	"context"
	"io"
	"net/http"
	"strings"

	"salespilot/internal/config"
)

// defaultModel dikirim ke API Hermes sebagai field "model" wajib OpenAI-compatible.
// Hermes gateway memilih model aktual sendiri; nilai ini hanya placeholder.
// Bila contract test menolak, ganti konstanta ini (terisolasi di sini).
const defaultModel = "default"

// httpClient adalah implementasi konkret Client berbasis net/http.
type httpClient struct {
	baseURL string
	apiKey  string
	hc      *http.Client
}

// compile-time assertion: httpClient harus memenuhi seluruh interface Client.
var _ Client = (*httpClient)(nil)

// New membuat Client baru dari konfigurasi aplikasi.
// Tidak menyimpan WorkspaceSessionKey di sini — session key dilewatkan per-call
// oleh caller (biasanya cfg.WorkspaceSessionKey).
func New(cfg *config.Config) Client {
	return &httpClient{
		baseURL: strings.TrimRight(cfg.HermesBaseURL, "/"),
		apiKey:  cfg.APIServerKey,
		hc:      &http.Client{}, // tanpa Timeout global; timeout dikontrol via ctx per-call
	}
}

// newReq membangun *http.Request dengan semua header wajib Hermes terpasang.
// Ini satu-satunya tempat header auth & sesi dipasang — tidak boleh diulang di tempat lain.
func (c *httpClient) newReq(ctx context.Context, method, path string, body io.Reader, sk SessionKey, sessionID string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if sk != "" {
		req.Header.Set("X-Hermes-Session-Key", string(sk))
	}
	if sessionID != "" {
		req.Header.Set("X-Hermes-Session-Id", sessionID)
	}
	return req, nil
}

