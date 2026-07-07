package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/config"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
)

type ChatHandler struct {
	svc    *service.ChatService
	hermes hermes.Client
	cfg    *config.Config
}

func NewChatHandler(svc *service.ChatService, hc hermes.Client, cfg *config.Config) *ChatHandler {
	return &ChatHandler{svc: svc, hermes: hc, cfg: cfg}
}

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

	// Persist user message (and auto-set title if blank).
	if _, err := h.svc.AppendUserMessage(c.Request().Context(), conv, req.Content); err != nil {
		return httperr.Write(c, err)
	}

	// Build hermes message history.
	history, err := h.svc.ListMessages(c.Request().Context(), conv.ID)
	if err != nil {
		return httperr.Write(c, err)
	}
	hermesMessages := toHermesMessages(history)

	hermesReq := hermes.ChatRequest{
		Messages:   hermesMessages,
		Stream:     true,
		SessionKey: hermes.SessionKey(conv.SessionKey),
		SessionID:  conv.HermesSessionID,
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
