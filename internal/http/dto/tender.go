package dto

import (
	"encoding/json"
	"time"

	"salespilot/internal/domain"
)

// --- Create ---

type TenderCreateRequest struct {
	Title                   string   `json:"title"                     validate:"required"`
	BuyerName               *string  `json:"buyer_name"                validate:"omitempty"`
	BuyerCountry            *string  `json:"buyer_country"             validate:"omitempty"`
	BuyerIndustry           *string  `json:"buyer_industry"            validate:"omitempty"`
	ValueEstimate           *float64 `json:"value_estimate"            validate:"omitempty,gte=0"`
	Currency                *string  `json:"currency"                  validate:"omitempty"`
	PublishedDate           *string  `json:"published_date"            validate:"omitempty"`
	SubmissionDeadline      *string  `json:"submission_deadline"       validate:"omitempty"`
	SourceName              *string  `json:"source_name"               validate:"omitempty"`
	SourceURL               *string  `json:"source_url"                validate:"omitempty"`
	ServiceCategory         *string  `json:"service_category"          validate:"omitempty"`
	ScopeSummary            *string  `json:"scope_summary"             validate:"omitempty"`
	EligibilityRequirements *string  `json:"eligibility_requirements"  validate:"omitempty"`
	TechnicalRequirements   *string  `json:"technical_requirements"    validate:"omitempty"`
	Status                  *string  `json:"status"                    validate:"omitempty,oneof=IDENTIFIED QUALIFYING BIDDING SUBMITTED WON LOST"`
	DedupKey                *string  `json:"dedup_key"                 validate:"omitempty"`
}

// --- Update ---

type TenderUpdateRequest struct {
	Title                   *string  `json:"title"                     validate:"omitempty"`
	BuyerName               *string  `json:"buyer_name"                validate:"omitempty"`
	BuyerCountry            *string  `json:"buyer_country"             validate:"omitempty"`
	BuyerIndustry           *string  `json:"buyer_industry"            validate:"omitempty"`
	ValueEstimate           *float64 `json:"value_estimate"            validate:"omitempty,gte=0"`
	Currency                *string  `json:"currency"                  validate:"omitempty"`
	PublishedDate           *string  `json:"published_date"            validate:"omitempty"`
	SubmissionDeadline      *string  `json:"submission_deadline"       validate:"omitempty"`
	SourceName              *string  `json:"source_name"               validate:"omitempty"`
	SourceURL               *string  `json:"source_url"                validate:"omitempty"`
	ServiceCategory         *string  `json:"service_category"          validate:"omitempty"`
	ScopeSummary            *string  `json:"scope_summary"             validate:"omitempty"`
	EligibilityRequirements *string  `json:"eligibility_requirements"  validate:"omitempty"`
	TechnicalRequirements   *string  `json:"technical_requirements"    validate:"omitempty"`
	Status                  *string  `json:"status"                    validate:"omitempty,oneof=IDENTIFIED QUALIFYING BIDDING SUBMITTED WON LOST"`
}

// --- Status transition ---

type TenderStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=IDENTIFIED QUALIFYING BIDDING SUBMITTED WON LOST"`
}

// --- Outcome ---

type TenderOutcomeRequest struct {
	Result string  `json:"result" validate:"required,oneof=WON LOST"`
	Notes  *string `json:"notes"  validate:"omitempty"`
}

// --- Review (Discovery Inbox Watchlist/Tolak) ---

type TenderReviewRequest struct {
	// Reason is optional free text (e.g. why a discovery-origin tender was
	// rejected) — recorded to audit_log for future learning (EP-16), not
	// stored on the tender row.
	Reason *string `json:"reason" validate:"omitempty"`
}

// --- Response ---

type TenderResponse struct {
	ID                      string          `json:"id"`
	Title                   string          `json:"title"`
	BuyerName               *string         `json:"buyer_name"`
	BuyerCountry            *string         `json:"buyer_country"`
	BuyerIndustry           *string         `json:"buyer_industry"`
	ValueEstimate           *float64        `json:"value_estimate"`
	Currency                string          `json:"currency"`
	PublishedDate           *time.Time      `json:"published_date"`
	SubmissionDeadline      *time.Time      `json:"submission_deadline"`
	SourceName              *string         `json:"source_name"`
	SourceURL               *string         `json:"source_url"`
	ServiceCategory         *string         `json:"service_category"`
	ScopeSummary            *string         `json:"scope_summary"`
	EligibilityRequirements *string         `json:"eligibility_requirements"`
	TechnicalRequirements   *string         `json:"technical_requirements"`
	Status                  string          `json:"status"`
	FitScore                *int            `json:"fit_score"`
	RecommendedAction       *string         `json:"recommended_action"`
	RiskFlags               json.RawMessage `json:"risk_flags"`
	ReasoningSummary        *string         `json:"reasoning_summary"`
	DedupKey                *string         `json:"dedup_key"`
	Origin                  string          `json:"origin"`
	ReviewedAt              *time.Time      `json:"reviewed_at"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

// --- List response ---

type TenderListResponse struct {
	Items    []TenderResponse `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// ToTenderResponse maps domain.Tender → TenderResponse.
func ToTenderResponse(t domain.Tender) TenderResponse {
	var ra *string
	if t.RecommendedAction != nil {
		s := string(*t.RecommendedAction)
		ra = &s
	}
	return TenderResponse{
		ID:                      t.ID,
		Title:                   t.Title,
		BuyerName:               t.BuyerName,
		BuyerCountry:            t.BuyerCountry,
		BuyerIndustry:           t.BuyerIndustry,
		ValueEstimate:           t.ValueEstimate,
		Currency:                t.Currency,
		PublishedDate:           t.PublishedDate,
		SubmissionDeadline:      t.SubmissionDeadline,
		SourceName:              t.SourceName,
		SourceURL:               t.SourceURL,
		ServiceCategory:         t.ServiceCategory,
		ScopeSummary:            t.ScopeSummary,
		EligibilityRequirements: t.EligibilityRequirements,
		TechnicalRequirements:   t.TechnicalRequirements,
		Status:                  string(t.Status),
		FitScore:                t.FitScore,
		RecommendedAction:       ra,
		RiskFlags:               t.RiskFlags,
		ReasoningSummary:        t.ReasoningSummary,
		DedupKey:                t.DedupKey,
		Origin:                  string(t.Origin),
		ReviewedAt:              t.ReviewedAt,
		CreatedAt:               t.CreatedAt,
		UpdatedAt:               t.UpdatedAt,
	}
}
