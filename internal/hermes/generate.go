package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// --- Wire types (privat) ---

type wireResponsesReq struct {
	Prompt         string          `json:"prompt"`
	ResponseFormat wireRespFormat  `json:"response_format"`
}

type wireRespFormat struct {
	Type       string          `json:"type"`
	JSONSchema wireJSONSchema  `json:"json_schema"`
}

type wireJSONSchema struct {
	Schema any `json:"schema"`
}

type wireResponsesResp struct {
	OutputText string `json:"output_text"`
}

// stripJSONFence menghapus pembungkus code-fence ```json / ``` dari output model.
// Model sering membungkus JSON dalam code-fence; hapus agar parsing deterministik.
func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Buang baris fence pembuka (``` atau ```json dsb.)
	idx := strings.Index(s, "\n")
	if idx == -1 {
		return s
	}
	s = s[idx+1:]
	// Buang fence penutup ``` di akhir.
	if end := strings.LastIndex(s, "```"); end != -1 {
		s = s[:end]
	}
	return strings.TrimSpace(s)
}

// doResponses melakukan satu round-trip ke /v1/responses dan mengembalikan
// raw text output (sudah di-strip fence). Dipakai oleh GenerateJSON + retry.
func (c *httpClient) doResponses(ctx context.Context, prompt string, schema any, sk SessionKey) (json.RawMessage, error) {
	payload, err := json.Marshal(wireResponsesReq{
		Prompt: prompt,
		ResponseFormat: wireRespFormat{
			Type:       "json_schema",
			JSONSchema: wireJSONSchema{Schema: schema},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("hermes generate: marshal: %w", err)
	}

	req, err := c.newReq(ctx, "POST", "/v1/responses", bytes.NewReader(payload), sk, "")
	if err != nil {
		return nil, fmt.Errorf("hermes generate: build request: %w", err)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hermes generate: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readErr(resp)
	}

	var wire wireResponsesResp
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, fmt.Errorf("hermes generate: decode response: %w", err)
	}

	out := stripJSONFence(wire.OutputText)
	if out == "" {
		return nil, fmt.Errorf("hermes generate: respon kosong")
	}

	return json.RawMessage(out), nil
}

// GenerateJSON memanggil /v1/responses untuk mendapatkan output JSON terstruktur.
// schema harus berupa pointer ke struct target (mis. &ScoreResult{}) — dipakai
// sebagai hint shape di request DAN target unmarshal untuk validasi.
// Bila schema == nil, kembalikan raw tanpa validasi.
// Bila output pertama bukan JSON valid, retry 1x dengan instruksi eksplisit.
func (c *httpClient) GenerateJSON(ctx context.Context, prompt string, schema any, sk SessionKey) (json.RawMessage, error) {
	raw, err := c.doResponses(ctx, prompt, schema, sk)
	if err != nil {
		return nil, err
	}

	// Bila schema nil, skip validasi — caller hanya mau raw JSON.
	if schema == nil {
		return raw, nil
	}

	if err := json.Unmarshal(raw, schema); err == nil {
		return raw, nil
	}

	// Retry 1x dengan instruksi eksplisit JSON-only.
	retryPrompt := prompt + "\n\nPENTING: Balas HANYA JSON valid yang cocok dengan schema. Tanpa penjelasan, tanpa markdown, tanpa code fence."
	raw2, err2 := c.doResponses(ctx, retryPrompt, schema, sk)
	if err2 != nil {
		return nil, fmt.Errorf("hermes generate: retry gagal: %w", err2)
	}

	if unmarshalErr := json.Unmarshal(raw2, schema); unmarshalErr != nil {
		return nil, fmt.Errorf("hermes generate: output bukan JSON valid setelah retry: %w (output: %.256s)", unmarshalErr, raw2)
	}

	return raw2, nil
}
