package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

// compile-time check: *ChatRepo implements domain.ChatRepository.
var _ domain.ChatRepository = (*ChatRepo)(nil)

type ChatRepo struct {
	db *gorm.DB
}

func NewChatRepo(db *gorm.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

func (r *ChatRepo) CreateConversation(ctx context.Context, c *domain.Conversation) error {
	if err := r.db.WithContext(ctx).Create(c).Error; err != nil {
		return fmt.Errorf("chat.CreateConversation: %w", err)
	}
	return nil
}

// GetConversationByID returns the conversation only if it belongs to ownerID.
// Returns gorm.ErrRecordNotFound if not found or ownership mismatch.
func (r *ChatRepo) GetConversationByID(ctx context.Context, id, ownerID string) (*domain.Conversation, error) {
	var c domain.Conversation
	if err := r.db.WithContext(ctx).
		First(&c, "id = ? AND owner_user_id = ?", id, ownerID).Error; err != nil {
		return nil, fmt.Errorf("chat.GetConversationByID: %w", err)
	}
	return &c, nil
}

func (r *ChatRepo) ListConversations(ctx context.Context, f domain.ConversationFilter, page, pageSize int) ([]domain.Conversation, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Conversation{})

	if f.OwnerUserID != "" {
		q = q.Where("owner_user_id = ?", f.OwnerUserID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("chat.ListConversations count: %w", err)
	}

	offset := (page - 1) * pageSize
	var convs []domain.Conversation
	if err := q.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&convs).Error; err != nil {
		return nil, 0, fmt.Errorf("chat.ListConversations: %w", err)
	}
	return convs, total, nil
}

func (r *ChatRepo) UpdateConversation(ctx context.Context, c *domain.Conversation) error {
	if err := r.db.WithContext(ctx).Save(c).Error; err != nil {
		return fmt.Errorf("chat.UpdateConversation: %w", err)
	}
	return nil
}

// DeleteConversation deletes the row only if it belongs to ownerID. Returns
// gorm.ErrRecordNotFound if no matching row was deleted (not found, or
// ownership mismatch) so the service can 404 rather than silently no-op.
func (r *ChatRepo) DeleteConversation(ctx context.Context, id, ownerID string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND owner_user_id = ?", id, ownerID).
		Delete(&domain.Conversation{})
	if res.Error != nil {
		return fmt.Errorf("chat.DeleteConversation: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("chat.DeleteConversation: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

func (r *ChatRepo) CreateMessage(ctx context.Context, m *domain.Message) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("chat.CreateMessage: %w", err)
	}
	return nil
}

func (r *ChatRepo) ListMessages(ctx context.Context, conversationID string) ([]domain.Message, error) {
	var msgs []domain.Message
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&msgs).Error; err != nil {
		return nil, fmt.Errorf("chat.ListMessages: %w", err)
	}
	return msgs, nil
}
