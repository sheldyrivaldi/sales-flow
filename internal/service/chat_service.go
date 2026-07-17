package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"salespilot/internal/config"
	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
)

type ChatService struct {
	repo domain.ChatRepository
	cfg  *config.Config
}

func NewChatService(repo domain.ChatRepository, cfg *config.Config) *ChatService {
	return &ChatService{repo: repo, cfg: cfg}
}

// newSessionID generates a random session identifier using crypto/rand.
func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// deriveTitle extracts up to 6 words from msg as a conversation title.
func deriveTitle(msg string) string {
	words := strings.Fields(strings.TrimSpace(msg))
	if len(words) == 0 {
		return "Percakapan baru"
	}
	if len(words) > 6 {
		words = words[:6]
	}
	title := strings.Join(words, " ")
	if len(title) > 80 {
		title = title[:80]
	}
	return title
}

// CreateConversation creates a new conversation for ownerID.
// If firstMsg is non-empty the title is derived from it; otherwise title is used as-is.
func (s *ChatService) CreateConversation(ctx context.Context, ownerID, title, firstMsg string) (*domain.Conversation, error) {
	if title == "" && firstMsg != "" {
		title = deriveTitle(firstMsg)
	}

	c := &domain.Conversation{
		OwnerUserID:     ownerID,
		Title:           title,
		SessionKey:      s.cfg.WorkspaceSessionKey,
		HermesSessionID: newSessionID(),
	}
	if err := s.repo.CreateConversation(ctx, c); err != nil {
		return nil, fmt.Errorf("chat.CreateConversation: %w", err)
	}
	return c, nil
}

// SetTitleIfEmpty updates the conversation title from the first user message if title is blank.
func (s *ChatService) SetTitleIfEmpty(ctx context.Context, conv *domain.Conversation, firstUserMsg string) error {
	if conv.Title != "" {
		return nil
	}
	conv.Title = deriveTitle(firstUserMsg)
	if err := s.repo.UpdateConversation(ctx, conv); err != nil {
		return fmt.Errorf("chat.SetTitleIfEmpty: %w", err)
	}
	return nil
}

// GetConversation returns a conversation if it belongs to ownerID.
func (s *ChatService) GetConversation(ctx context.Context, id, ownerID string) (*domain.Conversation, error) {
	c, err := s.repo.GetConversationByID(ctx, id, ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("percakapan tidak ditemukan")
		}
		return nil, fmt.Errorf("chat.GetConversation: %w", err)
	}
	return c, nil
}

// DeleteConversation removes a conversation (and its messages, via DB
// cascade) if it belongs to ownerID.
func (s *ChatService) DeleteConversation(ctx context.Context, id, ownerID string) error {
	if err := s.repo.DeleteConversation(ctx, id, ownerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httperr.NewNotFound("percakapan tidak ditemukan")
		}
		return fmt.Errorf("chat.DeleteConversation: %w", err)
	}
	return nil
}

// ListConversations returns paginated conversations for ownerID.
func (s *ChatService) ListConversations(ctx context.Context, ownerID string, page, pageSize int) ([]domain.Conversation, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	return s.repo.ListConversations(ctx, domain.ConversationFilter{OwnerUserID: ownerID}, page, pageSize)
}

// GetConversationDetail returns conversation + its messages.
func (s *ChatService) GetConversationDetail(ctx context.Context, id, ownerID string) (*domain.Conversation, []domain.Message, error) {
	conv, err := s.GetConversation(ctx, id, ownerID)
	if err != nil {
		return nil, nil, err
	}
	msgs, err := s.repo.ListMessages(ctx, conv.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("chat.GetConversationDetail messages: %w", err)
	}
	return conv, msgs, nil
}

// AppendUserMessage persists a user message. It also sets the title if blank.
func (s *ChatService) AppendUserMessage(ctx context.Context, conv *domain.Conversation, content string) (*domain.Message, error) {
	if err := s.SetTitleIfEmpty(ctx, conv, content); err != nil {
		// non-blocking: log-worthy but don't fail the request
		_ = err
	}

	m := &domain.Message{
		ConversationID: conv.ID,
		Role:           domain.RoleUser,
		Content:        content,
	}
	if err := s.repo.CreateMessage(ctx, m); err != nil {
		return nil, fmt.Errorf("chat.AppendUserMessage: %w", err)
	}
	return m, nil
}

// SaveAssistantMessage persists the assistant's response.
func (s *ChatService) SaveAssistantMessage(ctx context.Context, convID, content string, toolCalls []byte) error {
	m := &domain.Message{
		ConversationID: convID,
		Role:           domain.RoleAssistant,
		Content:        content,
		ToolCalls:      toolCalls,
	}
	if err := s.repo.CreateMessage(ctx, m); err != nil {
		return fmt.Errorf("chat.SaveAssistantMessage: %w", err)
	}
	return nil
}

// BuildHermesMessages converts domain messages to hermes wire format.
// Returned as []map so we avoid a hard import of the hermes package from service.
func (s *ChatService) ListMessages(ctx context.Context, conversationID string) ([]domain.Message, error) {
	msgs, err := s.repo.ListMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("chat.ListMessages: %w", err)
	}
	return msgs, nil
}
