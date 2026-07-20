package service

import (
	"context"
	"encoding/json"
	"testing"

	"salespilot/internal/domain"
)

// fakePlaybookJobRepo adalah repo in-memory minimal untuk menguji aturan judul.
type fakePlaybookJobRepo struct {
	job *domain.PlaybookJob
}

func (f *fakePlaybookJobRepo) Create(ctx context.Context, j *domain.PlaybookJob) error {
	j.ID = "job-1"
	f.job = j
	return nil
}
func (f *fakePlaybookJobRepo) Update(ctx context.Context, j *domain.PlaybookJob) error {
	f.job = j
	return nil
}
func (f *fakePlaybookJobRepo) GetByID(ctx context.Context, id string) (*domain.PlaybookJob, error) {
	return f.job, nil
}
func (f *fakePlaybookJobRepo) List(ctx context.Context) ([]domain.PlaybookJob, error) {
	return []domain.PlaybookJob{*f.job}, nil
}
func (f *fakePlaybookJobRepo) Delete(ctx context.Context, id string) error { return nil }

// aiContent meniru payload yang dikirim balik agent (judul karangan AI).
func aiContent(title string) []byte {
	b, _ := json.Marshal(map[string]any{"title": title, "deck": []any{}})
	return b
}

// Judul yang diketik user adalah final — inilah yang berkali-kali salah:
// judul TIDAK boleh diganti judul karangan AI saat generate selesai.
func TestCompleteKeepsUserTitle(t *testing.T) {
	repo := &fakePlaybookJobRepo{job: &domain.PlaybookJob{
		ID:         "job-1",
		Title:      "Judul Dari User",
		UserTitled: true,
		Status:     domain.PlaybookJobInProgress,
	}}
	svc := &PlaybookJobService{repo: repo}

	if err := svc.Complete(context.Background(), "job-1", aiContent("Judul Karangan AI"), ""); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := repo.job.Title; got != "Judul Dari User" {
		t.Errorf("judul user ditimpa AI: got %q, want %q", got, "Judul Dari User")
	}
	if repo.job.Status != domain.PlaybookJobSuccess {
		t.Errorf("status = %q, want success", repo.job.Status)
	}
}

// Bila user mengosongkan judul, AI yang menyusunnya.
func TestCompleteUsesAITitleWhenUserLeftItBlank(t *testing.T) {
	repo := &fakePlaybookJobRepo{job: &domain.PlaybookJob{
		ID:         "job-1",
		Title:      "strategi masuk sektor kesehatan", // turunan prompt
		UserTitled: false,
		Status:     domain.PlaybookJobInProgress,
	}}
	svc := &PlaybookJobService{repo: repo}

	if err := svc.Complete(context.Background(), "job-1", aiContent("Menang di Sektor Kesehatan 2026"), ""); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got := repo.job.Title; got != "Menang di Sektor Kesehatan 2026" {
		t.Errorf("judul AI tidak dipakai: got %q", got)
	}
}

// Judul user dikirim ke AI sebagai judul WAJIB, judul turunan prompt tidak.
func TestUserTitleOnlyForwardedWhenUserTyped(t *testing.T) {
	typed := &domain.PlaybookJob{Title: "Judul Saya", UserTitled: true}
	if got := userTitle(typed); got != "Judul Saya" {
		t.Errorf("userTitle(typed) = %q, want %q", got, "Judul Saya")
	}
	derived := &domain.PlaybookJob{Title: "turunan prompt", UserTitled: false}
	if got := userTitle(derived); got != "" {
		t.Errorf("userTitle(derived) = %q, want empty", got)
	}
	if !contains(customInstruction("Judul Saya", "buat playbook", nil, false), "Judul Saya") {
		t.Error("instruksi tidak memuat judul user")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
