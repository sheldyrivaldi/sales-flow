package service

import (
	"context"
	"testing"

	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
)

// fakeFeedbackFormRepo is an in-memory domain.FeedbackFormRepository for unit
// tests (no DB). Slug uniqueness is enforced so slugify tests are meaningful.
type fakeFeedbackFormRepo struct {
	forms map[string]*domain.FeedbackForm
	subs  []domain.FeedbackFormSubmission
	seq   int
}

var _ domain.FeedbackFormRepository = (*fakeFeedbackFormRepo)(nil)

func newFakeFeedbackFormRepo() *fakeFeedbackFormRepo {
	return &fakeFeedbackFormRepo{forms: map[string]*domain.FeedbackForm{}}
}

func (r *fakeFeedbackFormRepo) Create(_ context.Context, f *domain.FeedbackForm) error {
	r.seq++
	if f.ID == "" {
		f.ID = "form-" + string(rune('a'+r.seq))
	}
	cp := *f
	r.forms[f.ID] = &cp
	return nil
}

func (r *fakeFeedbackFormRepo) Update(_ context.Context, f *domain.FeedbackForm) error {
	cp := *f
	r.forms[f.ID] = &cp
	return nil
}

func (r *fakeFeedbackFormRepo) Delete(_ context.Context, id string) error {
	delete(r.forms, id)
	return nil
}

func (r *fakeFeedbackFormRepo) GetByID(_ context.Context, id string) (*domain.FeedbackForm, error) {
	f, ok := r.forms[id]
	if !ok {
		return nil, nil
	}
	cp := *f
	return &cp, nil
}

func (r *fakeFeedbackFormRepo) GetBySlug(_ context.Context, slug string) (*domain.FeedbackForm, error) {
	for _, f := range r.forms {
		if f.Slug == slug {
			cp := *f
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeFeedbackFormRepo) SlugExists(_ context.Context, slug string) (bool, error) {
	for _, f := range r.forms {
		if f.Slug == slug {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeFeedbackFormRepo) List(_ context.Context) ([]domain.FeedbackForm, error) {
	out := make([]domain.FeedbackForm, 0, len(r.forms))
	for _, f := range r.forms {
		cp := *f
		for _, s := range r.subs {
			if s.FormID == f.ID {
				cp.SubmissionCount++
			}
		}
		out = append(out, cp)
	}
	return out, nil
}

func (r *fakeFeedbackFormRepo) CreateSubmission(_ context.Context, s *domain.FeedbackFormSubmission) error {
	s.ID = "sub-" + string(rune('a'+len(r.subs)))
	r.subs = append(r.subs, *s)
	return nil
}

func (r *fakeFeedbackFormRepo) ListSubmissions(_ context.Context, formID string) ([]domain.FeedbackFormSubmission, error) {
	var out []domain.FeedbackFormSubmission
	for _, s := range r.subs {
		if s.FormID == formID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeFeedbackFormRepo) ListAllSubmissions(_ context.Context) ([]domain.FeedbackFormSubmission, error) {
	return append([]domain.FeedbackFormSubmission(nil), r.subs...), nil
}

func ptrInt(i int) *int       { return &i }
func ptrStr(s string) *string { return &s }

func TestFeedbackForm_Create_SlugifyAndUnique(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo)
	ctx := context.Background()

	q := []domain.FeedbackQuestion{{Type: domain.QuestionRating, Label: "Kepuasan"}}

	// Slug diturunkan dari judul.
	f1, err := svc.Create(ctx, &domain.FeedbackForm{Title: "Feedback Proyek A!", Questions: q}, "")
	if err != nil {
		t.Fatalf("create f1: %v", err)
	}
	if f1.Slug != "feedback-proyek-a" {
		t.Fatalf("slug = %q, want feedback-proyek-a", f1.Slug)
	}
	if f1.Questions[0].ID == "" {
		t.Fatalf("question ID tidak di-generate")
	}
	if f1.Questions[0].Scale != 5 {
		t.Fatalf("rating scale default = %d, want 5", f1.Questions[0].Scale)
	}

	// Judul sama → slug harus dibuat unik (-2).
	f2, err := svc.Create(ctx, &domain.FeedbackForm{Title: "Feedback Proyek A!", Questions: q}, "")
	if err != nil {
		t.Fatalf("create f2: %v", err)
	}
	if f2.Slug != "feedback-proyek-a-2" {
		t.Fatalf("slug f2 = %q, want feedback-proyek-a-2", f2.Slug)
	}

	// Slug custom dihormati (setelah sanitasi).
	f3, err := svc.Create(ctx, &domain.FeedbackForm{Title: "X", Questions: q}, "Survey KLIEN 2026")
	if err != nil {
		t.Fatalf("create f3: %v", err)
	}
	if f3.Slug != "survey-klien-2026" {
		t.Fatalf("slug f3 = %q, want survey-klien-2026", f3.Slug)
	}
}

func TestFeedbackForm_Create_RejectsChoiceWithoutOptions(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo)
	_, err := svc.Create(context.Background(), &domain.FeedbackForm{
		Title:     "Bad",
		Questions: []domain.FeedbackQuestion{{Type: domain.QuestionChoice, Label: "Pilih"}},
	}, "")
	if err == nil {
		t.Fatal("expected error untuk choice tanpa options")
	}
}

func TestFeedbackForm_Publish_RequiresQuestions(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo)
	ctx := context.Background()
	f, _ := svc.Create(ctx, &domain.FeedbackForm{Title: "Kosong"}, "")
	if _, err := svc.Publish(ctx, f.ID); err == nil {
		t.Fatal("expected publish gagal saat tanpa pertanyaan")
	}
}

func setupPublishedForm(t *testing.T, svc *FeedbackFormService) *domain.FeedbackForm {
	t.Helper()
	ctx := context.Background()
	f, err := svc.Create(ctx, &domain.FeedbackForm{
		Title:        "Kepuasan Client",
		CollectEmail: true,
		Questions: []domain.FeedbackQuestion{
			{ID: "r1", Type: domain.QuestionRating, Label: "Kualitas", Scale: 5, Required: true},
			{ID: "t1", Type: domain.QuestionText, Label: "Saran", Required: false},
			{ID: "c1", Type: domain.QuestionChoice, Label: "Rekomendasi", Options: []string{"Ya", "Tidak"}, Required: true},
		},
	}, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Publish(ctx, f.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	return f
}

func TestFeedbackForm_Submit_Validations(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo)
	ctx := context.Background()
	f := setupPublishedForm(t, svc)

	// Rating di luar rentang → error.
	err := svc.Submit(ctx, f.Slug, &domain.FeedbackFormSubmission{
		RespondentEmail: ptrStr("a@b.com"),
		Answers: []domain.FeedbackAnswer{
			{QuestionID: "r1", Rating: ptrInt(9)},
			{QuestionID: "c1", Choice: []string{"Ya"}},
		},
	})
	if err == nil {
		t.Fatal("expected error rating di luar rentang")
	}

	// Pertanyaan wajib (c1) kosong → error.
	err = svc.Submit(ctx, f.Slug, &domain.FeedbackFormSubmission{
		RespondentEmail: ptrStr("a@b.com"),
		Answers:         []domain.FeedbackAnswer{{QuestionID: "r1", Rating: ptrInt(4)}},
	})
	if err == nil {
		t.Fatal("expected error required c1 kosong")
	}

	// Email wajib tapi kosong → error.
	err = svc.Submit(ctx, f.Slug, &domain.FeedbackFormSubmission{
		Answers: []domain.FeedbackAnswer{
			{QuestionID: "r1", Rating: ptrInt(4)},
			{QuestionID: "c1", Choice: []string{"Ya"}},
		},
	})
	if err == nil {
		t.Fatal("expected error email wajib")
	}

	// Pilihan tidak valid → error.
	err = svc.Submit(ctx, f.Slug, &domain.FeedbackFormSubmission{
		RespondentEmail: ptrStr("a@b.com"),
		Answers: []domain.FeedbackAnswer{
			{QuestionID: "r1", Rating: ptrInt(4)},
			{QuestionID: "c1", Choice: []string{"Mungkin"}},
		},
	})
	if err == nil {
		t.Fatal("expected error pilihan tidak valid")
	}

	// Nama wajib tapi kosong → error.
	err = svc.Submit(ctx, f.Slug, &domain.FeedbackFormSubmission{
		RespondentEmail: ptrStr("a@b.com"),
		Answers: []domain.FeedbackAnswer{
			{QuestionID: "r1", Rating: ptrInt(4)},
			{QuestionID: "c1", Choice: []string{"Ya"}},
		},
	})
	if err == nil {
		t.Fatal("expected error nama wajib")
	}

	// Valid submit.
	err = svc.Submit(ctx, f.Slug, &domain.FeedbackFormSubmission{
		RespondentEmail: ptrStr("a@b.com"),
		RespondentName:  ptrStr("Budi"),
		Answers: []domain.FeedbackAnswer{
			{QuestionID: "r1", Rating: ptrInt(4)},
			{QuestionID: "t1", Text: "Bagus"},
			{QuestionID: "c1", Choice: []string{"Ya"}},
		},
	})
	if err != nil {
		t.Fatalf("valid submit gagal: %v", err)
	}
}

func TestFeedbackForm_PublicGet_OnlyPublished(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo)
	ctx := context.Background()
	f, _ := svc.Create(ctx, &domain.FeedbackForm{
		Title:     "Draft",
		Questions: []domain.FeedbackQuestion{{Type: domain.QuestionRating, Label: "R"}},
	}, "")
	// Belum published → NotFound.
	if _, err := svc.PublicGet(ctx, f.Slug); err == nil {
		t.Fatal("expected NotFound untuk form draft")
	} else if ae, ok := err.(*httperr.APIError); !ok || ae.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestFeedbackForm_Analytics_Aggregates(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo)
	ctx := context.Background()
	f := setupPublishedForm(t, svc)

	for _, r := range []int{4, 5, 3} {
		if err := svc.Submit(ctx, f.Slug, &domain.FeedbackFormSubmission{
			RespondentEmail: ptrStr("x@y.com"),
			RespondentName:  ptrStr("Client"),
			Answers: []domain.FeedbackAnswer{
				{QuestionID: "r1", Rating: ptrInt(r)},
				{QuestionID: "c1", Choice: []string{"Ya"}},
			},
		}); err != nil {
			t.Fatalf("submit r=%d: %v", r, err)
		}
	}

	a, err := svc.Analytics(ctx, f.ID)
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}
	if a.TotalSubmissions != 3 {
		t.Fatalf("total submissions = %d, want 3", a.TotalSubmissions)
	}
	// Rata-rata rating (4+5+3)/3 = 4.0 pada skala 5.
	if a.AvgRating != 4.0 {
		t.Fatalf("avg rating = %.2f, want 4.0", a.AvgRating)
	}
	// Cari statistik pertanyaan rating r1.
	var found bool
	for _, q := range a.Questions {
		if q.QuestionID == "r1" {
			found = true
			if q.Responses != 3 || q.Average != 4.0 {
				t.Fatalf("r1 stat = %+v", q)
			}
			if len(q.Distribution) != 5 || q.Distribution[3] != 1 || q.Distribution[4] != 1 || q.Distribution[2] != 1 {
				t.Fatalf("r1 distribution = %v", q.Distribution)
			}
		}
		if q.QuestionID == "c1" {
			if q.Choices["Ya"] != 3 {
				t.Fatalf("c1 choices = %v", q.Choices)
			}
		}
	}
	if !found {
		t.Fatal("statistik r1 tidak ditemukan")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Feedback Proyek A!": "feedback-proyek-a",
		"  Hello  World  ":   "hello-world",
		"Survey/Klien-2026":  "survey-klien-2026",
		"***":                "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
