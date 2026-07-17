package hermes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// --- Wire types (privat) ---

type wireResponsesReq struct {
	Prompt         string         `json:"prompt"`
	ResponseFormat wireRespFormat `json:"response_format"`
	// DocumentBase64/DocumentFilename are set only by
	// GenerateJSONFromDocument — the bridge renders the PDF's pages to
	// images and feeds them to the model as native vision input instead of
	// relying on prompt text alone.
	DocumentBase64   string `json:"document_base64,omitempty"`
	DocumentFilename string `json:"document_filename,omitempty"`
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
// filename/fileBytes kosong berarti tidak ada dokumen terlampir (prompt teks
// biasa); non-kosong mengirim dokumen sebagai base64 untuk dibaca via vision
// oleh bridge (lihat GenerateJSONFromDocument).
func (c *httpClient) doResponses(ctx context.Context, prompt string, schema any, sk SessionKey, filename string, fileBytes []byte) (json.RawMessage, error) {
	wireReq := wireResponsesReq{
		Prompt: prompt,
		ResponseFormat: wireRespFormat{
			Type:       "json_schema",
			JSONSchema: wireJSONSchema{Schema: schema},
		},
	}
	if len(fileBytes) > 0 {
		wireReq.DocumentBase64 = base64.StdEncoding.EncodeToString(fileBytes)
		wireReq.DocumentFilename = filename
	}

	payload, err := json.Marshal(wireReq)
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

// generateJSON adalah implementasi bersama GenerateJSON/GenerateJSONFromDocument:
// panggil /v1/responses, retry 1x dengan instruksi eksplisit bila output
// pertama bukan JSON valid. filename/fileBytes kosong = tanpa lampiran.
func (c *httpClient) generateJSON(ctx context.Context, prompt string, schema any, sk SessionKey, filename string, fileBytes []byte) (json.RawMessage, error) {
	raw, err := c.doResponses(ctx, prompt, schema, sk, filename, fileBytes)
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
	raw2, err2 := c.doResponses(ctx, retryPrompt, schema, sk, filename, fileBytes)
	if err2 != nil {
		return nil, fmt.Errorf("hermes generate: retry gagal: %w", err2)
	}

	if unmarshalErr := json.Unmarshal(raw2, schema); unmarshalErr != nil {
		return nil, fmt.Errorf("hermes generate: output bukan JSON valid setelah retry: %w (output: %.256s)", unmarshalErr, raw2)
	}

	return raw2, nil
}

// GenerateJSON memanggil /v1/responses untuk mendapatkan output JSON terstruktur.
// schema harus berupa pointer ke struct target (mis. &ScoreResult{}) — dipakai
// sebagai hint shape di request DAN target unmarshal untuk validasi.
// Bila schema == nil, kembalikan raw tanpa validasi.
// Bila output pertama bukan JSON valid, retry 1x dengan instruksi eksplisit.
func (c *httpClient) GenerateJSON(ctx context.Context, prompt string, schema any, sk SessionKey) (json.RawMessage, error) {
	return c.generateJSON(ctx, prompt, schema, sk, "", nil)
}

// GenerateJSONFromDocument is like GenerateJSON but attaches fileBytes (a
// PDF) to the request — the bridge renders its pages to images and feeds
// them to the model as native vision input (EP-13), so extraction quality
// doesn't depend on a lossy local text-extraction pass first.
func (c *httpClient) GenerateJSONFromDocument(ctx context.Context, prompt, filename string, fileBytes []byte, schema any, sk SessionKey) (json.RawMessage, error) {
	return c.generateJSON(ctx, prompt, schema, sk, filename, fileBytes)
}
