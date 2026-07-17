package hermes

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// --- Wire types (privat, tidak diekspor) ---

// wireChatReq adalah body JSON yang dikirim ke /v1/chat/completions.
type wireChatReq struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	// Lampiran dokumen opsional untuk pesan user terakhir (lihat
	// ChatRequest.DocumentBase64) — bridge merender PDF ke gambar per
	// halaman dan mengirimnya sebagai konten multimodal.
	DocumentBase64   string `json:"document_base64,omitempty"`
	DocumentFilename string `json:"document_filename,omitempty"`
}

// wireFunction adalah bagian nested "function" pada tool call dari OpenAI wire format.
type wireFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// wireToolCall adalah representasi tool call dalam wire format OpenAI (nested).
type wireToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function wireFunction `json:"function"`
}

// wireChatResp adalah respons non-stream dari /v1/chat/completions.
type wireChatResp struct {
	Choices []struct {
		Message struct {
			Content   string         `json:"content"`
			ToolCalls []wireToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// wireStreamChunk adalah satu chunk SSE dari response stream.
type wireStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string         `json:"content"`
			ToolCalls []wireToolCall `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

// toToolCall mengkonversi wireToolCall (nested) ke ToolCall domain (flat).
func toToolCall(w wireToolCall) ToolCall {
	return ToolCall{
		ID:        w.ID,
		Type:      w.Type,
		Name:      w.Function.Name,
		Arguments: w.Function.Arguments,
	}
}

// --- Error helper ---

// readErr membaca body response error (dibatasi 4KB) dan membentuk error informatif.
// Caller harus sudah memanggil defer resp.Body.Close() sebelum memanggil readErr.
func readErr(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	return fmt.Errorf("hermes: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
}

// --- Chat (non-stream) ---

// Chat mengirim request ke Hermes dan mengembalikan respons lengkap (non-streaming).
// Timeout dikontrol oleh ctx yang diberikan caller.
func (c *httpClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body, err := json.Marshal(wireChatReq{
		Model:            defaultModel,
		Messages:         req.Messages,
		Stream:           false,
		DocumentBase64:   req.DocumentBase64,
		DocumentFilename: req.DocumentFilename,
	})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("hermes chat: marshal: %w", err)
	}

	httpReq, err := c.newReq(ctx, http.MethodPost, "/v1/chat/completions", bytes.NewReader(body), req.SessionKey, req.SessionID)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("hermes chat: build request: %w", err)
	}

	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("hermes chat: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, readErr(resp)
	}

	var wire wireChatResp
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return ChatResponse{}, fmt.Errorf("hermes chat: decode response: %w", err)
	}
	if len(wire.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("hermes chat: respon tanpa choices")
	}

	msg := wire.Choices[0].Message
	tcs := make([]ToolCall, 0, len(msg.ToolCalls))
	for _, w := range msg.ToolCalls {
		tcs = append(tcs, toToolCall(w))
	}

	return ChatResponse{Content: msg.Content, ToolCalls: tcs}, nil
}

// --- ChatStream (SSE) ---

// ChatStream mengirim request stream ke Hermes dan mengembalikan channel Chunk.
// Channel ditutup otomatis saat stream selesai ([DONE]), ctx dibatalkan, atau terjadi error.
// Caller bertanggung jawab mengonsumsi seluruh channel agar goroutine tidak leak.
func (c *httpClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan Chunk, error) {
	body, err := json.Marshal(wireChatReq{
		Model:            defaultModel,
		Messages:         req.Messages,
		Stream:           true,
		DocumentBase64:   req.DocumentBase64,
		DocumentFilename: req.DocumentFilename,
	})
	if err != nil {
		return nil, fmt.Errorf("hermes stream: marshal: %w", err)
	}

	httpReq, err := c.newReq(ctx, http.MethodPost, "/v1/chat/completions", bytes.NewReader(body), req.SessionKey, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("hermes stream: build request: %w", err)
	}

	resp, err := c.hc.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("hermes stream: do request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		return nil, readErr(resp)
	}

	ch := make(chan Chunk)

	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()

		send := func(chunk Chunk) bool {
			select {
			case ch <- chunk:
				return true
			case <-ctx.Done():
				ch <- Chunk{Err: ctx.Err()}
				return false
			}
		}

		// Gunakan buffer besar (1MB per baris) untuk menangani tool call arguments yang panjang.
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

		for scanner.Scan() {
			line := scanner.Text()

			// Abaikan baris kosong dan komentar SSE.
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// Hanya proses baris "data: ..."
			payload, ok := strings.CutPrefix(line, "data: ")
			if !ok {
				continue
			}

			// Akhir stream.
			if payload == "[DONE]" {
				send(Chunk{Done: true})
				return
			}

			// Parse chunk JSON.
			var chunk wireStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				// Baris tidak valid — lewati, jangan hentikan stream.
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta

			if delta.Content != "" {
				if !send(Chunk{Delta: delta.Content}) {
					return
				}
			}

			for _, w := range delta.ToolCalls {
				tc := toToolCall(w)
				if !send(Chunk{ToolCall: &tc}) {
					return
				}
			}
		}

		// Loop selesai tanpa [DONE] — cek error scanner.
		if err := scanner.Err(); err != nil {
			// Bedakan error akibat ctx canceled dari error I/O murni.
			if ctx.Err() != nil {
				ch <- Chunk{Err: ctx.Err()}
			} else {
				ch <- Chunk{Err: fmt.Errorf("hermes stream: read: %w", err)}
			}
		}
	}()

	return ch, nil
}
