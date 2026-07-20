package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

var _ domain.FeedbackRepository = (*FeedbackRepo)(nil)

type FeedbackRepo struct {
	db *gorm.DB
}

func NewFeedbackRepo(db *gorm.DB) *FeedbackRepo {
	return &FeedbackRepo{db: db}
}

func (r *FeedbackRepo) CreateRequest(ctx context.Context, req *domain.FeedbackRequest) error {
	if err := r.db.WithContext(ctx).Create(req).Error; err != nil {
		return fmt.Errorf("feedback.CreateRequest: %w", err)
	}
	return nil
}

func (r *FeedbackRepo) DeleteRequest(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.FeedbackRequest{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("feedback.DeleteRequest: %w", err)
	}
	return nil
}

func (r *FeedbackRepo) GetRequestByToken(ctx context.Context, token string) (*domain.FeedbackRequest, error) {
	var req domain.FeedbackRequest
	err := r.db.WithContext(ctx).Preload("Response").First(&req, "token = ?", token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("feedback.GetRequestByToken: %w", err)
	}
	return &req, nil
}

func (r *FeedbackRepo) ListRequests(ctx context.Context) ([]domain.FeedbackRequest, error) {
	var items []domain.FeedbackRequest
	if err := r.db.WithContext(ctx).Preload("Response").
		Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("feedback.ListRequests: %w", err)
	}
	return items, nil
}

func (r *FeedbackRepo) CreateResponse(ctx context.Context, resp *domain.FeedbackResponse) error {
	if err := r.db.WithContext(ctx).Create(resp).Error; err != nil {
		return fmt.Errorf("feedback.CreateResponse: %w", err)
	}
	return nil
}

func (r *FeedbackRepo) ListResponses(ctx context.Context) ([]domain.FeedbackResponse, error) {
	var items []domain.FeedbackResponse
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("feedback.ListResponses: %w", err)
	}
	return items, nil
}
