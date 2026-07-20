package service

import (
	"context"
	"fmt"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
)

// ProjectService mengelola proyek BERJALAN (menu Proyek Berjalan) — CRUD
// tipis di atas repo + agregasi ringkasan kesehatan portofolio proyek.
type ProjectService struct {
	repo domain.ProjectRepository
}

func NewProjectService(repo domain.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) Create(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	if !p.Status.Valid() {
		return nil, httperr.NewBadRequest("INVALID_STATUS", "status proyek tidak valid")
	}
	if p.Milestones == nil {
		p.Milestones = []domain.ProjectMilestone{}
	}
	if p.Activities == nil {
		p.Activities = []domain.ProjectActivity{}
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProjectService) Update(ctx context.Context, p *domain.Project) (*domain.Project, error) {
	if !p.Status.Valid() {
		return nil, httperr.NewBadRequest("INVALID_STATUS", "status proyek tidak valid")
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProjectService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProjectService) Get(ctx context.Context, id string) (*domain.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) List(ctx context.Context, f domain.ProjectFilter, page, pageSize int) ([]domain.Project, int64, error) {
	return s.repo.List(ctx, f, page, pageSize)
}

// AddActivity menambah satu catatan check-in ke proyek (append-only).
func (s *ProjectService) AddActivity(ctx context.Context, id, note string) (*domain.Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Activities = append([]domain.ProjectActivity{{Date: time.Now(), Note: note}}, p.Activities...)
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ProjectSummary adalah agregat untuk halaman Ringkasan Proyek Berjalan.
type ProjectSummary struct {
	TotalActive   int     `json:"total_active"`
	TotalValue    float64 `json:"total_value"`
	OnTrack       int     `json:"on_track"`
	AtRisk        int     `json:"at_risk"`
	Delayed       int     `json:"delayed"`
	Completed     int     `json:"completed"`
	AvgProgress   int     `json:"avg_progress"`
	EndingSoon    int     `json:"ending_soon"` // berakhir <= 30 hari lagi
}

func (s *ProjectService) Summary(ctx context.Context) (*ProjectSummary, error) {
	items, _, err := s.repo.List(ctx, domain.ProjectFilter{}, 1, 500)
	if err != nil {
		return nil, fmt.Errorf("project.Summary: %w", err)
	}
	sum := &ProjectSummary{}
	progressTotal, activeCount := 0, 0
	soonCutoff := time.Now().AddDate(0, 0, 30)
	for _, p := range items {
		switch p.Status {
		case domain.ProjectOnTrack:
			sum.OnTrack++
		case domain.ProjectAtRisk:
			sum.AtRisk++
		case domain.ProjectDelayed:
			sum.Delayed++
		case domain.ProjectCompleted:
			sum.Completed++
		}
		if p.Status != domain.ProjectCompleted {
			activeCount++
			progressTotal += p.Progress
			if p.ContractValue != nil {
				sum.TotalValue += *p.ContractValue
			}
			if p.EndDate != nil && p.EndDate.Before(soonCutoff) && p.EndDate.After(time.Now()) {
				sum.EndingSoon++
			}
		}
	}
	sum.TotalActive = activeCount
	if activeCount > 0 {
		sum.AvgProgress = progressTotal / activeCount
	}
	return sum, nil
}
