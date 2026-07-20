package domain

import (
	"context"
	"time"
)

// FeedbackRequest adalah permintaan feedback pasca-proyek — token unik
// menjadi link publik (/f/:token) yang dibagikan ke client tanpa login.
type FeedbackRequest struct {
	ID          string    `json:"id"           gorm:"primaryKey;default:gen_random_uuid()"`
	Token       string    `json:"token"        gorm:"not null;uniqueIndex"`
	ProjectName string    `json:"project_name" gorm:"not null"`
	ClientName  *string   `json:"client_name"`
	ProjectID   *string   `json:"project_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Response di-preload saat listing/detail (nil = belum diisi client).
	Response *FeedbackResponse `json:"response,omitempty" gorm:"foreignKey:RequestID"`
}

func (FeedbackRequest) TableName() string { return "feedback_request" }

// FeedbackResponse adalah jawaban client — satu per request (form singkat:
// rating keseluruhan wajib, 3 aspek + NPS + komentar opsional).
type FeedbackResponse struct {
	ID                  string    `json:"id"                   gorm:"primaryKey;default:gen_random_uuid()"`
	RequestID           string    `json:"request_id"           gorm:"not null;uniqueIndex"`
	OverallRating       int       `json:"overall_rating"       gorm:"not null"`
	QualityRating       *int      `json:"quality_rating"`
	CommunicationRating *int      `json:"communication_rating"`
	TimelinessRating    *int      `json:"timeliness_rating"`
	NPS                 *int      `json:"nps"                  gorm:"column:nps"`
	Comment             *string   `json:"comment"`
	RespondentName      *string   `json:"respondent_name"`
	CreatedAt           time.Time `json:"created_at"`
}

func (FeedbackResponse) TableName() string { return "feedback_response" }

type FeedbackRepository interface {
	CreateRequest(ctx context.Context, r *FeedbackRequest) error
	DeleteRequest(ctx context.Context, id string) error
	// GetRequestByToken returns (nil, nil) when the token doesn't exist —
	// link publik yang salah bukan error sistem.
	GetRequestByToken(ctx context.Context, token string) (*FeedbackRequest, error)
	ListRequests(ctx context.Context) ([]FeedbackRequest, error)
	CreateResponse(ctx context.Context, r *FeedbackResponse) error
	ListResponses(ctx context.Context) ([]FeedbackResponse, error)
}
