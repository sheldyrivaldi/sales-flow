package hermes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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
	// Documents: BANYAK lampiran sekaligus (bentuk jamak). Bridge merender tiap
	// dokumen ke gambar per halaman dan menggabung semuanya jadi satu pesan.
	Documents []wireDocument `json:"documents,omitempty"`
}

type wireRespFormat struct {
	Type       string         `json:"type"`
	JSONSchema wireJSONSchema `json:"json_schema"`
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

// looksLikeProviderError mendeteksi teks yang jelas-jelas pesan kegagalan
// provider (bukan JSON), sehingga tidak salah di-parse sebagai output.
func looksLikeProviderError(s string) bool {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[") {
		return false
	}
	lower := strings.ToLower(t)
	return strings.Contains(lower, "api call failed") ||
		strings.Contains(lower, "connection error") ||
		strings.Contains(lower, "broken pipe")
}

// doResponses melakukan satu round-trip ke /v1/responses dan mengembalikan
// raw text output (sudah di-strip fence). Dipakai oleh GenerateJSON + retry.
// filename/fileBytes kosong berarti tidak ada dokumen terlampir (prompt teks
// biasa); non-kosong mengirim dokumen sebagai base64 untuk dibaca via vision
// oleh bridge (lihat GenerateJSONFromDocument).
func (c *httpClient) doResponses(ctx context.Context, prompt string, schema any, sk SessionKey, docs []AgentDocument) (json.RawMessage, error) {
	wireReq := wireResponsesReq{
		Prompt: prompt,
		ResponseFormat: wireRespFormat{
			Type:       "json_schema",
			JSONSchema: wireJSONSchema{Schema: schema},
		},
	}
	for _, d := range docs {
		if len(d.Bytes) == 0 {
			continue
		}
		wireReq.Documents = append(wireReq.Documents, wireDocument{
			Base64:   base64.StdEncoding.EncodeToString(d.Bytes),
			Filename: d.Filename,
		})
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

	// Safety net: bila bridge tetap mengembalikan penanda kegagalan provider
	// sebagai teks (mis. "API call failed after N retries: ..."), jangan coba
	// parse sebagai JSON — surfacing sebagai error yang jelas.
	if looksLikeProviderError(out) {
		return nil, fmt.Errorf("hermes generate: provider gagal: %.200s", out)
	}

	return json.RawMessage(out), nil
}

// isTransientProviderErr menandai error yang layak dicoba ulang: kegagalan
// koneksi/jaringan ke provider yang sifatnya sementara (bukan error prompt/
// schema). Endpoint provider kadang melempar "Connection error" secara
// intermiten — retry di sini memberi kesempatan koneksi yang sehat.
func isTransientProviderErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, marker := range []string{
		"provider gagal", "connection error", "broken pipe", "api call failed",
		"connection reset", "eof", "do request", "timeout", "502",
	} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// generateJSON adalah implementasi bersama GenerateJSON/GenerateJSONFromDocument.
// Membungkus generateJSONAttempt dengan retry berjenjang khusus kegagalan
// koneksi provider yang sementara — tiap attempt punya backoff dan sadar
// pembatalan context (deadline job).
func (c *httpClient) generateJSON(ctx context.Context, prompt string, schema any, sk SessionKey, docs []AgentDocument) (json.RawMessage, error) {
	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Backoff singkat sebelum mencoba lagi; hormati deadline context.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 3 * time.Second):
			}
		}
		raw, err := c.generateJSONAttempt(ctx, prompt, schema, sk, docs)
		if err == nil {
			return raw, nil
		}
		lastErr = err
		if !isTransientProviderErr(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("hermes generate: gagal setelah %d percobaan: %w", maxAttempts, lastErr)
}

// generateJSONAttempt melakukan satu kali generate: panggil /v1/responses,
// retry 1x dengan instruksi eksplisit bila output pertama bukan JSON valid.
func (c *httpClient) generateJSONAttempt(ctx context.Context, prompt string, schema any, sk SessionKey, docs []AgentDocument) (json.RawMessage, error) {
	raw, err := c.doResponses(ctx, prompt, schema, sk, docs)
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
	raw2, err2 := c.doResponses(ctx, retryPrompt, schema, sk, docs)
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
	return c.generateJSON(ctx, prompt, schema, sk, nil)
}

// GenerateJSONFromDocument is like GenerateJSON but attaches fileBytes (a
// PDF) to the request — the bridge renders its pages to images and feeds
// them to the model as native vision input (EP-13), so extraction quality
// doesn't depend on a lossy local text-extraction pass first.
func (c *httpClient) GenerateJSONFromDocument(ctx context.Context, prompt, filename string, fileBytes []byte, schema any, sk SessionKey) (json.RawMessage, error) {
	return c.generateJSON(ctx, prompt, schema, sk, []AgentDocument{{Filename: filename, Bytes: fileBytes}})
}

// GenerateJSONFromDocuments is like GenerateJSONFromDocument but attaches
// MANY documents at once (each rendered to page images by the bridge and
// concatenated) — used when the caller gathers several context files (mis.
// menyusun kuesioner feedback dari beberapa lampiran).
func (c *httpClient) GenerateJSONFromDocuments(ctx context.Context, prompt string, docs []AgentDocument, schema any, sk SessionKey) (json.RawMessage, error) {
	return c.generateJSON(ctx, prompt, schema, sk, docs)
}
