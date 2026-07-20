package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/config"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
	"salespilot/internal/telemetry"
)

// mimeByExt maps a lowercase file extension to a content type — enough for the
// chat attachment types (PDF + common images) so the browser opens rather than
// downloads them.
func mimeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

// saveChatAttachment decodes a base64 payload and writes it under
// <uploadDir>/chat/<uuid><ext>, returning the public URL ("/uploads/chat/...")
// and detected MIME. The UUID filename keeps stored files unguessable.
func saveChatAttachment(uploadDir, filename, b64 string) (url, mime string, err error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", "", fmt.Errorf("decode base64: %w", err)
	}
	ext := filepath.Ext(filename)
	name := uuid.NewString() + ext
	dir := filepath.Join(uploadDir, "chat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		return "", "", fmt.Errorf("write file: %w", err)
	}
	return "/uploads/chat/" + name, mimeByExt(ext), nil
}

type ChatHandler struct {
	svc    *service.ChatService
	hermes hermes.Client
	cfg    *config.Config
	emit   *telemetry.Emitter
}

func NewChatHandler(svc *service.ChatService, hc hermes.Client, cfg *config.Config) *ChatHandler {
	return &ChatHandler{svc: svc, hermes: hc, cfg: cfg}
}

// SetEmitter wires telemetry (EP-17 ST-17.1) after construction — optional,
// nil-safe.
func (h *ChatHandler) SetEmitter(e *telemetry.Emitter) { h.emit = e }

// Create handles POST /api/conversations
func (h *ChatHandler) Create(c echo.Context) error {
	var req dto.ConversationCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	title := ""
	if req.Title != nil {
		title = *req.Title
	}
	firstMsg := ""
	if req.FirstMessage != nil {
		firstMsg = *req.FirstMessage
	}

	conv, err := h.svc.CreateConversation(c.Request().Context(), user.ID, title, firstMsg)
	if err != nil {
		return httperr.Write(c, err)
	}

	if h.emit != nil {
		h.emit.Emit(c.Request().Context(), "chat_opened", map[string]any{"actor": user.ID})
	}

	return c.JSON(http.StatusCreated, dto.ToConversationResponse(*conv))
}

// List handles GET /api/conversations
func (h *ChatHandler) List(c echo.Context) error {
	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	convs, total, err := h.svc.ListConversations(c.Request().Context(), user.ID, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.ConversationResponse, len(convs))
	for i, cv := range convs {
		items[i] = dto.ToConversationResponse(cv)
	}

	return c.JSON(http.StatusOK, dto.ConversationListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Get handles GET /api/conversations/:id
func (h *ChatHandler) Get(c echo.Context) error {
	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	id := c.Param("id")
	conv, msgs, err := h.svc.GetConversationDetail(c.Request().Context(), id, user.ID)
	if err != nil {
		return httperr.Write(c, err)
	}

	msgResp := make([]dto.MessageResponse, len(msgs))
	for i, m := range msgs {
		msgResp[i] = dto.ToMessageResponse(m)
	}

	return c.JSON(http.StatusOK, dto.ConversationDetailResponse{
		ID:        conv.ID,
		Title:     conv.Title,
		Messages:  msgResp,
		CreatedAt: conv.CreatedAt,
		UpdatedAt: conv.UpdatedAt,
	})
}

// Delete handles DELETE /api/conversations/:id
func (h *ChatHandler) Delete(c echo.Context) error {
	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	id := c.Param("id")
	if err := h.svc.DeleteConversation(c.Request().Context(), id, user.ID); err != nil {
		return httperr.Write(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// chatGuardrailPrompt membatasi kanal chat/Tanya AI ke percakapan analitis:
// membaca & menganalisa data ya, aksi tulis tidak. Dikirim sebagai system
// message di SETIAP giliran (bukan hanya giliran pertama) supaya batasan
// tetap berlaku sepanjang percakapan.
const chatGuardrailPrompt = "Kamu adalah asisten AI SalesFlow untuk tim sales B2B. " +
	"Batasan KERAS untuk kanal chat ini: kamu HANYA boleh menjawab pertanyaan, menganalisa, merangkum, dan MEMBACA data " +
	"(tender, prospek, event, profil perusahaan) atau mencari informasi publik. " +
	"Kamu DILARANG melakukan aksi tulis dalam bentuk apa pun dari chat ini: dilarang membuat/mengubah/menghapus data aplikasi, " +
	"dilarang membuat atau menulis file, dilarang menjalankan kode/perintah/skrip, dan dilarang memanggil tool yang bersifat menulis atau mengeksekusi. " +
	"Bila user meminta aksi seperti itu, jelaskan dengan sopan bahwa aksi dilakukan lewat menu aplikasi terkait (mis. tombol di halaman Tender/Pipeline/Playbook), " +
	"lalu bantu dengan analisa atau langkah panduannya. Jangan pernah menyebut nama teknologi/engine internal di balik layar — sebut dirimu sebagai AI SalesFlow."

// Chat handles POST /api/conversations/:id/chat (SSE relay)
func (h *ChatHandler) Chat(c echo.Context) error {
	var req dto.ChatMessageRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	id := c.Param("id")
	conv, err := h.svc.GetConversation(c.Request().Context(), id, user.ID)
	if err != nil {
		return httperr.Write(c, err)
	}

	// Optional document attachment (PDF/image) — validated here, forwarded to
	// the bridge which renders PDFs to page images for native vision reading.
	// ~10 MB raw ≈ 14 MB base64; same cap as Company Profile PDF ingest.
	var attachB64, attachName string
	if req.AttachmentBase64 != nil && *req.AttachmentBase64 != "" {
		attachB64 = *req.AttachmentBase64
		if len(attachB64) > 14*1024*1024 {
			return httperr.Write(c, httperr.NewBadRequest("FILE_TOO_LARGE", "lampiran melebihi batas ukuran 10 MB"))
		}
		attachName = "document.pdf"
		if req.AttachmentName != nil && *req.AttachmentName != "" {
			attachName = *req.AttachmentName
		}
	}

	// Persist the attachment bytes to disk so the file stays openable from
	// the conversation history (image preview / PDF open). Failure to save is
	// non-fatal — the chat turn still proceeds, just without a stored file.
	var attURL, attNamePtr, attMimePtr *string
	if attachB64 != "" {
		if url, mime, err := saveChatAttachment(h.cfg.UploadDir, attachName, attachB64); err != nil {
			log.Printf("chat: gagal menyimpan lampiran: %v", err)
		} else {
			n := attachName
			attURL, attNamePtr, attMimePtr = &url, &n, &mime
		}
	}

	// Persist user message (and auto-set title if blank). Attachment metadata
	// (URL/name/mime) is stored so the history can render a clickable preview.
	if _, err := h.svc.AppendUserMessageWithAttachment(c.Request().Context(), conv, req.Content, attURL, attNamePtr, attMimePtr); err != nil {
		return httperr.Write(c, err)
	}

	// Build hermes message history.
	history, err := h.svc.ListMessages(c.Request().Context(), conv.ID)
	if err != nil {
		return httperr.Write(c, err)
	}
	// Pesan system pembatas SELALU di depan: chat/Tanya AI adalah kanal
	// percakapan — analisa dan membaca data saja. Aksi tulis (buat/ubah/
	// hapus data, buat file, jalankan kode/perintah) dilakukan lewat fitur
	// aplikasi masing-masing, bukan dari chat.
	hermesMessages := append([]hermes.Message{{Role: "system", Content: chatGuardrailPrompt}}, toHermesMessages(history)...)

	hermesReq := hermes.ChatRequest{
		Messages:         hermesMessages,
		Stream:           true,
		SessionKey:       hermes.SessionKey(conv.SessionKey),
		SessionID:        conv.HermesSessionID,
		DocumentBase64:   attachB64,
		DocumentFilename: attachName,
	}

	ch, streamErr := h.hermes.ChatStream(c.Request().Context(), hermesReq)
	if streamErr != nil {
		// Header not sent yet — return JSON error.
		return httperr.Write(c, httperr.NewBadRequest("AI_UNAVAILABLE", "Agent AI sedang tidak tersedia. Coba lagi sebentar lagi."))
	}

	// Set SSE headers.
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")
	c.Response().WriteHeader(http.StatusOK)

	var assistantContent strings.Builder
	var toolCallsAccum []hermes.ToolCall

	for chunk := range ch {
		if chunk.Err != nil {
			if errors.Is(chunk.Err, context.Canceled) {
				// Client disconnected — stop silently.
				break
			}
			// AI or I/O error — emit friendly SSE event.
			writeSSEEvent(c, map[string]string{
				"type":    "error",
				"message": "Agent AI sedang tidak tersedia. Coba lagi sebentar lagi.",
			})
			break
		}

		if chunk.Delta != "" {
			assistantContent.WriteString(chunk.Delta)
			writeSSEEvent(c, map[string]interface{}{
				"type":    "delta",
				"content": chunk.Delta,
			})
			c.Response().Flush()
		}

		if chunk.ToolCall != nil {
			toolCallsAccum = append(toolCallsAccum, *chunk.ToolCall)
			writeSSEEvent(c, map[string]interface{}{
				"type":      "tool_call",
				"id":        chunk.ToolCall.ID,
				"name":      chunk.ToolCall.Name,
				"arguments": chunk.ToolCall.Arguments,
			})
			c.Response().Flush()
		}

		if chunk.Done {
			writeSSEEvent(c, map[string]string{"type": "done"})
			fmt.Fprint(c.Response(), "data: [DONE]\n\n")
			c.Response().Flush()
			break
		}
	}

	// Best-effort: persist assistant message even if stream was interrupted.
	// Uses a detached context — the request context may already be canceled
	// (client disconnect) by the time we get here.
	var toolCallsJSON []byte
	if len(toolCallsAccum) > 0 {
		toolCallsJSON, _ = json.Marshal(toolCallsAccum)
	}
	persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = h.svc.SaveAssistantMessage(persistCtx, conv.ID, assistantContent.String(), toolCallsJSON)

	return nil
}

// writeSSEEvent writes a single SSE data frame.
func writeSSEEvent(c echo.Context, payload interface{}) {
	b, _ := json.Marshal(payload)
	fmt.Fprintf(c.Response(), "data: %s\n\n", b)
}

// toHermesMessages maps domain messages to hermes wire messages.
func toHermesMessages(msgs []domain.Message) []hermes.Message {
	out := make([]hermes.Message, 0, len(msgs))
	for _, m := range msgs {
		hm := hermes.Message{
			Role:    string(m.Role),
			Content: m.Content,
		}
		if len(m.ToolCalls) > 0 {
			var tcs []hermes.ToolCall
			if err := json.Unmarshal(m.ToolCalls, &tcs); err == nil {
				hm.ToolCalls = tcs
			}
		}
		out = append(out, hm)
	}
	return out
}
