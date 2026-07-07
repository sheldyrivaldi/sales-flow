package domain

import (
	"context"
	"encoding/json"
	"time"
)

type TenderStatus string

const (
	TenderStatusIdentified TenderStatus = "IDENTIFIED"
	TenderStatusQualifying TenderStatus = "QUALIFYING"
	TenderStatusBidding    TenderStatus = "BIDDING"
	TenderStatusSubmitted  TenderStatus = "SUBMITTED"
	TenderStatusWon        TenderStatus = "WON"
	TenderStatusLost       TenderStatus = "LOST"
)

func (s TenderStatus) Valid() bool {
	switch s {
	case TenderStatusIdentified, TenderStatusQualifying, TenderStatusBidding,
		TenderStatusSubmitted, TenderStatusWon, TenderStatusLost:
		return true
	}
	return false
}

type RecommendedAction string

const (
	ActionPursue      RecommendedAction = "PURSUE"
	ActionReview      RecommendedAction = "REVIEW"
	ActionWatchlist   RecommendedAction = "WATCHLIST"
	ActionReject      RecommendedAction = "REJECT"
	ActionNeedPartner RecommendedAction = "NEED_PARTNER"
)

func (a RecommendedAction) Valid() bool {
	switch a {
	case ActionPursue, ActionReview, ActionWatchlist, ActionReject, ActionNeedPartner:
		return true
	}
	return false
}

type TenderOrigin string

const (
	OriginManual    TenderOrigin = "manual"
	OriginDiscovery TenderOrigin = "discovery"
)

func (o TenderOrigin) Valid() bool {
	switch o {
	case OriginManual, OriginDiscovery:
		return true
	}
	return false
}

type Tender struct {
	ID                      string             `json:"id"                        gorm:"primaryKey;default:gen_random_uuid()"`
	Title                   string             `json:"title"                     gorm:"not null"`
	BuyerName               *string            `json:"buyer_name"                gorm:"column:buyer_name"`
	BuyerCountry            *string            `json:"buyer_country"             gorm:"column:buyer_country"`
	BuyerIndustry           *string            `json:"buyer_industry"            gorm:"column:buyer_industry"`
	ValueEstimate           *float64           `json:"value_estimate"            gorm:"column:value_estimate"`
	Currency                string             `json:"currency"                  gorm:"not null;default:'IDR'"`
	PublishedDate           *time.Time         `json:"published_date"            gorm:"column:published_date"`
	SubmissionDeadline      *time.Time         `json:"submission_deadline"       gorm:"column:submission_deadline"`
	SourceName              *string            `json:"source_name"               gorm:"column:source_name"`
	SourceURL               *string            `json:"source_url"                gorm:"column:source_url"`
	ServiceCategory         *string            `json:"service_category"          gorm:"column:service_category"`
	ScopeSummary            *string            `json:"scope_summary"             gorm:"column:scope_summary"`
	EligibilityRequirements *string            `json:"eligibility_requirements"  gorm:"column:eligibility_requirements"`
	TechnicalRequirements   *string            `json:"technical_requirements"    gorm:"column:technical_requirements"`
	Status                  TenderStatus       `json:"status"                    gorm:"not null;default:'IDENTIFIED'"`
	FitScore                *int               `json:"fit_score"                 gorm:"column:fit_score"`
	RecommendedAction       *RecommendedAction `json:"recommended_action"        gorm:"column:recommended_action"`
	RiskFlags               json.RawMessage    `json:"risk_flags"                gorm:"type:jsonb"`
	ReasoningSummary        *string            `json:"reasoning_summary"         gorm:"column:reasoning_summary"`
	DedupKey                *string            `json:"dedup_key"                 gorm:"column:dedup_key"`
	Origin                  TenderOrigin       `json:"origin"                    gorm:"not null;default:'manual'"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
}

func (Tender) TableName() string { return "tender" }

type TenderFilter struct {
	Status            *TenderStatus
	BuyerName         string
	RecommendedAction *RecommendedAction
	Origin            *TenderOrigin
	DeadlineFrom      *time.Time
	DeadlineTo        *time.Time
	Search            string
}

type TenderRepository interface {
	Create(ctx context.Context, t *Tender) error
	GetByID(ctx context.Context, id string) (*Tender, error)
	List(ctx context.Context, f TenderFilter, page, pageSize int) ([]Tender, int64, error)
	Update(ctx context.Context, t *Tender) error
	Delete(ctx context.Context, id string) error
}
