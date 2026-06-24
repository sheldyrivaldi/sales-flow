package domain

import (
	"context"
	"encoding/json"
	"time"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

func (r MessageRole) Valid() bool {
	switch r {
	case RoleUser, RoleAssistant, RoleSystem, RoleTool:
		return true
	}
	return false
}

type Conversation struct {
	ID              string    `json:"id"                gorm:"primaryKey"`
	OwnerUserID     string    `json:"owner_user_id"     gorm:"column:owner_user_id;not null"`
	Title           string    `json:"title"             gorm:"not null;default:''"`
	SessionKey      string    `json:"-"                 gorm:"column:session_key;not null"`
	HermesSessionID string    `json:"-"                 gorm:"column:hermes_session_id;not null"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Conversation) TableName() string { return "conversation" }

type Message struct {
	ID             string          `json:"id"              gorm:"primaryKey"`
	ConversationID string          `json:"conversation_id" gorm:"column:conversation_id;not null"`
	Role           MessageRole     `json:"role"            gorm:"not null"`
	Content        string          `json:"content"         gorm:"not null;default:''"`
	ToolCalls      json.RawMessage `json:"tool_calls,omitempty" gorm:"type:jsonb"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (Message) TableName() string { return "message" }

type ConversationFilter struct {
	OwnerUserID string
}

type ChatRepository interface {
	CreateConversation(ctx context.Context, c *Conversation) error
	GetConversationByID(ctx context.Context, id, ownerID string) (*Conversation, error)
	ListConversations(ctx context.Context, f ConversationFilter, page, pageSize int) ([]Conversation, int64, error)
	UpdateConversation(ctx context.Context, c *Conversation) error
	CreateMessage(ctx context.Context, m *Message) error
	ListMessages(ctx context.Context, conversationID string) ([]Message, error)
}
