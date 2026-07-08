package dto

// --- Generate (POST /api/profile/keywords/generate — draft only, not persisted) ---

type KeywordGenerateRequest struct {
	ServiceCategories []string `json:"service_categories" validate:"required,min=1"`
	Language          *string  `json:"language"           validate:"omitempty"`
}

// --- Response ---

// KeywordGenerateResponse is a draft keyword set generated from capabilities.
// It is not persisted — the caller reviews/edits it and saves via PUT /api/profile.
// Degraded=true when the AI call failed (Hermes down/invalid output); in that
// case Keywords is empty and NegativeKeywords falls back to the deterministic
// preset only, so the CRUD flow keeps working without AI.
type KeywordGenerateResponse struct {
	Keywords         []string `json:"keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
	Language         string   `json:"language"`
	Degraded         bool     `json:"degraded"`
}
