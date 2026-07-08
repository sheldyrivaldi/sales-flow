package dto

import (
	"time"

	"salespilot/internal/domain"
)

// --- Update (PUT /api/profile — always creates a new version) ---

type TargetCriteriaRequest struct {
	Countries        []string `json:"countries"          validate:"omitempty"`
	Industries       []string `json:"industries"         validate:"omitempty"`
	ValueMin         *float64 `json:"value_min"          validate:"omitempty,gte=0"`
	ValueIdeal       *float64 `json:"value_ideal"        validate:"omitempty,gte=0"`
	ValueMax         *float64 `json:"value_max"          validate:"omitempty,gte=0"`
	Currency         *string  `json:"currency"           validate:"omitempty"`
	DeadlineMinDays  *int     `json:"deadline_min_days"  validate:"omitempty,gte=0"`
	ProcurementTypes []string `json:"procurement_types"  validate:"omitempty"`
}

type NoGoRuleRequest struct {
	PresetFlags []string `json:"preset_flags" validate:"omitempty"`
	Custom      []string `json:"custom"       validate:"omitempty"`
}

type KeywordSetRequest struct {
	Category         *string  `json:"category"`
	Keywords         []string `json:"keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
	Language         *string  `json:"language" validate:"omitempty"`
}

type ProfileUpdateRequest struct {
	CompanyName       string                 `json:"company_name"        validate:"required"`
	OneLiner          *string                `json:"one_liner"`
	ServiceCategories []string               `json:"service_categories"`
	TechStack         []string               `json:"tech_stack"`
	SourceDocRefs     []string               `json:"source_doc_refs"`
	Target            *TargetCriteriaRequest `json:"target"   validate:"omitempty"`
	NoGo              *NoGoRuleRequest       `json:"nogo"     validate:"omitempty"`
	Keywords          []KeywordSetRequest    `json:"keywords" validate:"omitempty,dive"`
}

// --- Response ---

type TargetCriteriaResponse struct {
	Countries        []string `json:"countries"`
	Industries       []string `json:"industries"`
	ValueMin         *float64 `json:"value_min"`
	ValueIdeal       *float64 `json:"value_ideal"`
	ValueMax         *float64 `json:"value_max"`
	Currency         string   `json:"currency"`
	DeadlineMinDays  *int     `json:"deadline_min_days"`
	ProcurementTypes []string `json:"procurement_types"`
}

type NoGoRuleResponse struct {
	PresetFlags []string `json:"preset_flags"`
	Custom      []string `json:"custom"`
}

type KeywordSetResponse struct {
	ID               string   `json:"id"`
	Category         *string  `json:"category"`
	Keywords         []string `json:"keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
	Language         string   `json:"language"`
}

// ProfileResponse is the full "Otak Agent" snapshot for the current version.
type ProfileResponse struct {
	ID                string                  `json:"id"`
	CompanyName       string                  `json:"company_name"`
	OneLiner          *string                 `json:"one_liner"`
	ServiceCategories []string                `json:"service_categories"`
	TechStack         []string                `json:"tech_stack"`
	SourceDocRefs     []string                `json:"source_doc_refs"`
	Version           int                     `json:"version"`
	IsCurrent         bool                    `json:"is_current"`
	Target            *TargetCriteriaResponse `json:"target"`
	NoGo              *NoGoRuleResponse       `json:"nogo"`
	Keywords          []KeywordSetResponse    `json:"keywords"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

// ToProfileResponse maps a domain.ProfileAggregate → ProfileResponse.
func ToProfileResponse(agg domain.ProfileAggregate) ProfileResponse {
	resp := ProfileResponse{
		ID:                agg.Profile.ID,
		CompanyName:       agg.Profile.CompanyName,
		OneLiner:          agg.Profile.OneLiner,
		ServiceCategories: orEmpty(agg.Profile.ServiceCategories),
		TechStack:         orEmpty(agg.Profile.TechStack),
		SourceDocRefs:     orEmpty(agg.Profile.SourceDocRefs),
		Version:           agg.Profile.Version,
		IsCurrent:         agg.Profile.IsCurrent,
		CreatedAt:         agg.Profile.CreatedAt,
		UpdatedAt:         agg.Profile.UpdatedAt,
	}

	if agg.Target != nil {
		resp.Target = &TargetCriteriaResponse{
			Countries:        orEmpty(agg.Target.Countries),
			Industries:       orEmpty(agg.Target.Industries),
			ValueMin:         agg.Target.ValueMin,
			ValueIdeal:       agg.Target.ValueIdeal,
			ValueMax:         agg.Target.ValueMax,
			Currency:         agg.Target.Currency,
			DeadlineMinDays:  agg.Target.DeadlineMinDays,
			ProcurementTypes: orEmpty(agg.Target.ProcurementTypes),
		}
	}

	if agg.NoGo != nil {
		resp.NoGo = &NoGoRuleResponse{
			PresetFlags: orEmpty(agg.NoGo.PresetFlags),
			Custom:      orEmpty(agg.NoGo.Custom),
		}
	}

	resp.Keywords = make([]KeywordSetResponse, len(agg.Keywords))
	for i, k := range agg.Keywords {
		resp.Keywords[i] = KeywordSetResponse{
			ID:               k.ID,
			Category:         k.Category,
			Keywords:         orEmpty(k.Keywords),
			NegativeKeywords: orEmpty(k.NegativeKeywords),
			Language:         k.Language,
		}
	}

	return resp
}

// orEmpty ensures a nil slice serializes as `[]` instead of `null`.
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
