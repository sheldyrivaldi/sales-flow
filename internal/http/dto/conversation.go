package dto

import (
	"time"

	"salespilot/internal/domain"
)

// --- Create ---

type ConversationCreateRequest struct {
	Title        *string `json:"title"         validate:"omitempty"`
	FirstMessage *string `json:"first_message" validate:"omitempty"`
}

// --- Response types ---

type ConversationResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToConversationResponse(c domain.Conversation) ConversationResponse {
	return ConversationResponse{
		ID:        c.ID,
		Title:     c.Title,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

type ConversationListResponse struct {
	Items    []ConversationResponse `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// --- Detail (with messages) ---

type MessageResponse struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	Role           string          `json:"role"`
	Content        string          `json:"content"`
	ToolCalls      interface{}     `json:"tool_calls,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

func ToMessageResponse(m domain.Message) MessageResponse {
	return MessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           string(m.Role),
		Content:        m.Content,
		ToolCalls:      m.ToolCalls,
		CreatedAt:      m.CreatedAt,
	}
}

type ConversationDetailResponse struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Messages  []MessageResponse `json:"messages"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// --- Chat message request ---

type ChatMessageRequest struct {
	Content string `json:"content" validate:"required"`
}
