package dto

import "salespilot/internal/domain"

// --- Feedback Client (form builder dinamis) ---

// FeedbackFormUpsertRequest membuat/mengubah form. Slug opsional (diturunkan
// dari judul & di-unik-kan bila kosong).
type FeedbackFormUpsertRequest struct {
	Title        string                    `json:"title" validate:"required"`
	Description  *string                   `json:"description"`
	Slug         *string                   `json:"slug"`
	Language     *string                   `json:"language"`
	CollectEmail *bool                     `json:"collect_email"`
	Questions    []domain.FeedbackQuestion `json:"questions"`
	ProjectID    *string                   `json:"project_id"`
}

// FeedbackFormPublicResponse adalah bentuk form untuk halaman publik /form/:slug.
type FeedbackFormPublicResponse struct {
	Title        string                    `json:"title"`
	Description  *string                   `json:"description"`
	Slug         string                    `json:"slug"`
	Language     domain.FormLanguage       `json:"language"`
	CollectEmail bool                      `json:"collect_email"`
	Questions    []domain.FeedbackQuestion `json:"questions"`
}

// FeedbackFormSubmitRequest adalah pengisian form oleh client. Email & nama
// wajib (divalidasi service); divisi opsional.
type FeedbackFormSubmitRequest struct {
	RespondentEmail    *string                 `json:"respondent_email"`
	RespondentName     *string                 `json:"respondent_name"`
	RespondentDivision *string                 `json:"respondent_division"`
	Answers            []domain.FeedbackAnswer `json:"answers"`
}

// FeedbackRefineRequest — "edit dengan AI" satu pertanyaan.
type FeedbackRefineRequest struct {
	Question    SuggestedQuestionDTO `json:"question"`
	Instruction string               `json:"instruction" validate:"required"`
	Language    *string              `json:"language"`
}

// SuggestedQuestionDTO mencerminkan service.SuggestedQuestion agar handler
// tidak mengimpor tipe service ke body request.
type SuggestedQuestionDTO struct {
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Scale       int      `json:"scale"`
	Options     []string `json:"options"`
	Multiple    bool     `json:"multiple"`
	MinLabel    string   `json:"min_label"`
	MaxLabel    string   `json:"max_label"`
}
