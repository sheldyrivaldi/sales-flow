package dto

import "salespilot/internal/domain"

// --- Proyek Berjalan (menu Ongoing) ---

type ProjectUpsertRequest struct {
	Name          string                    `json:"name" validate:"required"`
	ClientName    *string                   `json:"client_name"`
	ContractValue *float64                  `json:"contract_value"`
	Currency      *string                   `json:"currency"`
	StartDate     *string                   `json:"start_date"` // YYYY-MM-DD
	EndDate       *string                   `json:"end_date"`
	Status        *string                   `json:"status" validate:"omitempty,oneof=ON_TRACK AT_RISK DELAYED COMPLETED"`
	Progress      *int                      `json:"progress" validate:"omitempty,min=0,max=100"`
	Description   *string                   `json:"description"`
	Milestones    []domain.ProjectMilestone `json:"milestones"`
}

type ProjectActivityRequest struct {
	Note string `json:"note" validate:"required"`
}

type ProjectListResponse struct {
	Items    []domain.Project `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// --- Pasca-Proyek (feedback client) ---

type FeedbackCreateRequest struct {
	ProjectName string  `json:"project_name" validate:"required"`
	ClientName  *string `json:"client_name"`
	ProjectID   *string `json:"project_id"`
}

type FeedbackSubmitRequest struct {
	OverallRating       int     `json:"overall_rating" validate:"required,min=1,max=5"`
	QualityRating       *int    `json:"quality_rating" validate:"omitempty,min=1,max=5"`
	CommunicationRating *int    `json:"communication_rating" validate:"omitempty,min=1,max=5"`
	TimelinessRating    *int    `json:"timeliness_rating" validate:"omitempty,min=1,max=5"`
	NPS                 *int    `json:"nps" validate:"omitempty,min=0,max=10"`
	Comment             *string `json:"comment"`
	RespondentName      *string `json:"respondent_name"`
}

// FeedbackPublicInfo adalah data minimum untuk halaman publik /f/:token —
// tidak membocorkan apa pun selain nama proyek/klien & status terisi.
type FeedbackPublicInfo struct {
	ProjectName string  `json:"project_name"`
	ClientName  *string `json:"client_name"`
	Submitted   bool    `json:"submitted"`
}
