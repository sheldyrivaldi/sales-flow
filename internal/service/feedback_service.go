package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
)

// FeedbackService mengelola siklus feedback pasca-proyek: buat link publik,
// terima jawaban client (sekali per link), dan agregasi analitiknya.
type FeedbackService struct {
	repo domain.FeedbackRepository
}

func NewFeedbackService(repo domain.FeedbackRepository) *FeedbackService {
	return &FeedbackService{repo: repo}
}

func (s *FeedbackService) CreateRequest(ctx context.Context, projectName string, clientName, projectID *string) (*domain.FeedbackRequest, error) {
	req := &domain.FeedbackRequest{
		// Token pendek tapi tetap acak-kuat (uuid tanpa strip) — enak dibaca
		// di URL yang dibagikan ke client.
		Token:       strings.ReplaceAll(uuid.NewString(), "-", ""),
		ProjectName: projectName,
		ClientName:  clientName,
		ProjectID:   projectID,
	}
	if err := s.repo.CreateRequest(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *FeedbackService) DeleteRequest(ctx context.Context, id string) error {
	return s.repo.DeleteRequest(ctx, id)
}

func (s *FeedbackService) ListRequests(ctx context.Context) ([]domain.FeedbackRequest, error) {
	return s.repo.ListRequests(ctx)
}

// PublicInfo mengembalikan info minimum untuk halaman publik /f/:token.
func (s *FeedbackService) PublicInfo(ctx context.Context, token string) (*domain.FeedbackRequest, error) {
	req, err := s.repo.GetRequestByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, httperr.NewNotFound("link feedback tidak ditemukan")
	}
	return req, nil
}

// Submit menyimpan jawaban client — hanya sekali per link.
func (s *FeedbackService) Submit(ctx context.Context, token string, resp *domain.FeedbackResponse) error {
	req, err := s.repo.GetRequestByToken(ctx, token)
	if err != nil {
		return err
	}
	if req == nil {
		return httperr.NewNotFound("link feedback tidak ditemukan")
	}
	if req.Response != nil {
		return httperr.NewBadRequest("ALREADY_SUBMITTED", "feedback untuk proyek ini sudah pernah diisi, terima kasih")
	}
	if resp.OverallRating < 1 || resp.OverallRating > 5 {
		return httperr.NewBadRequest("INVALID_RATING", "rating keseluruhan harus 1 sampai 5")
	}
	resp.RequestID = req.ID
	return s.repo.CreateResponse(ctx, resp)
}

// FeedbackAnalytics adalah agregat untuk halaman Analisa Pasca-Proyek.
type FeedbackAnalytics struct {
	TotalRequests      int        `json:"total_requests"`
	TotalResponses     int        `json:"total_responses"`
	AvgOverall         float64    `json:"avg_overall"`
	AvgQuality         float64    `json:"avg_quality"`
	AvgCommunication   float64    `json:"avg_communication"`
	AvgTimeliness      float64    `json:"avg_timeliness"`
	NPS                int        `json:"nps"` // -100..100
	RatingDistribution [5]int     `json:"rating_distribution"`
	Comments           []struct {
		ProjectName string `json:"project_name"`
		ClientName  string `json:"client_name"`
		Rating      int    `json:"rating"`
		Comment     string `json:"comment"`
		CreatedAt   string `json:"created_at"`
	} `json:"comments"`
}

func (s *FeedbackService) Analytics(ctx context.Context) (*FeedbackAnalytics, error) {
	requests, err := s.repo.ListRequests(ctx)
	if err != nil {
		return nil, err
	}

	a := &FeedbackAnalytics{TotalRequests: len(requests)}
	var sumOverall, sumQ, sumC, sumT, nQ, nC, nT int
	var promoters, detractors, npsCount int

	for _, req := range requests {
		r := req.Response
		if r == nil {
			continue
		}
		a.TotalResponses++
		sumOverall += r.OverallRating
		if r.OverallRating >= 1 && r.OverallRating <= 5 {
			a.RatingDistribution[r.OverallRating-1]++
		}
		if r.QualityRating != nil {
			sumQ += *r.QualityRating
			nQ++
		}
		if r.CommunicationRating != nil {
			sumC += *r.CommunicationRating
			nC++
		}
		if r.TimelinessRating != nil {
			sumT += *r.TimelinessRating
			nT++
		}
		if r.NPS != nil {
			npsCount++
			if *r.NPS >= 9 {
				promoters++
			} else if *r.NPS <= 6 {
				detractors++
			}
		}
		if r.Comment != nil && *r.Comment != "" {
			client := ""
			if req.ClientName != nil {
				client = *req.ClientName
			}
			a.Comments = append(a.Comments, struct {
				ProjectName string `json:"project_name"`
				ClientName  string `json:"client_name"`
				Rating      int    `json:"rating"`
				Comment     string `json:"comment"`
				CreatedAt   string `json:"created_at"`
			}{req.ProjectName, client, r.OverallRating, *r.Comment, r.CreatedAt.Format("2006-01-02")})
		}
	}

	if a.TotalResponses > 0 {
		a.AvgOverall = round1(float64(sumOverall) / float64(a.TotalResponses))
	}
	if nQ > 0 {
		a.AvgQuality = round1(float64(sumQ) / float64(nQ))
	}
	if nC > 0 {
		a.AvgCommunication = round1(float64(sumC) / float64(nC))
	}
	if nT > 0 {
		a.AvgTimeliness = round1(float64(sumT) / float64(nT))
	}
	if npsCount > 0 {
		a.NPS = (promoters*100 - detractors*100) / npsCount
	}
	return a, nil
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
