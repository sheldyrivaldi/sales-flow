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
	svc := NewFeedbackFormService(repo, nil, "", "")
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
	svc := NewFeedbackFormService(repo, nil, "", "")
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
	svc := NewFeedbackFormService(repo, nil, "", "")
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
	svc := NewFeedbackFormService(repo, nil, "", "")
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
	svc := NewFeedbackFormService(repo, nil, "", "")
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
	svc := NewFeedbackFormService(repo, nil, "", "")
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

// --- Saran AI async ---

func TestFeedbackForm_StartAISuggest_VaguePromptNeedsClarificationWithoutDispatch(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	// runner nil sengaja: prompt tipis TIDAK BOLEH memicu dispatch sama
	// sekali, jadi runner yang nil pun tidak boleh menyebabkan masalah.
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, err := svc.StartAISuggest(ctx, "", "buatkan feedback", nil, nil, domain.LangID, nil, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if f.Status != domain.FormNeedClarification {
		t.Fatalf("status = %q, want need_clarification", f.Status)
	}
	if f.AIJob == nil || len(f.AIJob.ClarifyingQuestions) == 0 {
		t.Fatalf("ai_job harus berisi clarifying_questions: %+v", f.AIJob)
	}
	if f.ID == "" || f.Slug == "" {
		t.Fatal("form baru harus langsung tersimpan (ID + slug terisi)")
	}
}

func TestFeedbackForm_StartAISuggest_DescriptivePromptGoesProcessing(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, err := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if f.Status != domain.FormProcessingAI {
		t.Fatalf("status = %q, want processing_ai", f.Status)
	}
	if f.AIJob == nil || f.AIJob.Prompt != "ukur kepuasan client proyek migrasi ERP" {
		t.Fatalf("ai_job.prompt tidak tersimpan: %+v", f.AIJob)
	}
}

func TestFeedbackForm_StartAISuggest_RejectsWhenAlreadyProcessing(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, err := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	if err != nil {
		t.Fatalf("start pertama: %v", err)
	}
	if _, err := svc.StartAISuggest(ctx, f.ID, "prompt lain yang juga deskriptif", nil, nil, domain.LangID, nil, nil); err == nil {
		t.Fatal("harus ditolak selagi form masih processing_ai")
	}
}

func TestFeedbackForm_CompleteAISuggest_AppliesQuestionsOnHighConfidence(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, err := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	content := []byte(`{"confidence":92,"questions":[{"type":"text","label":"Saran perbaikan"}]}`)
	if err := svc.CompleteAISuggest(ctx, f.ID, content, ""); err != nil {
		t.Fatalf("complete: %v", err)
	}

	got, err := repo.GetByID(ctx, f.ID)
	if err != nil || got == nil {
		t.Fatalf("get after complete: %v", err)
	}
	if got.Status != domain.FormDraft {
		t.Fatalf("status = %q, want draft", got.Status)
	}
	if got.AIJob == nil || len(got.AIJob.PendingQuestions) != 1 {
		t.Fatalf("pending_questions harus terisi 1: %+v", got.AIJob)
	}
}

func TestFeedbackForm_CompleteAISuggest_LowConfidenceNeedsClarification(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, _ := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	content := []byte(`{"confidence":30,"clarifying_questions":["Untuk divisi apa?"],"questions":[]}`)
	if err := svc.CompleteAISuggest(ctx, f.ID, content, ""); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, _ := repo.GetByID(ctx, f.ID)
	if got.Status != domain.FormNeedClarification {
		t.Fatalf("status = %q, want need_clarification", got.Status)
	}
	if len(got.AIJob.ClarifyingQuestions) != 1 {
		t.Fatalf("clarifying_questions = %+v", got.AIJob.ClarifyingQuestions)
	}
}

func TestFeedbackForm_CompleteAISuggest_FailureRevertsToDraftWithError(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, _ := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	if err := svc.CompleteAISuggest(ctx, f.ID, nil, "Provider gagal"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, _ := repo.GetByID(ctx, f.ID)
	if got.Status != domain.FormDraft {
		t.Fatalf("status = %q, want draft", got.Status)
	}
	if got.AIJob == nil || got.AIJob.Error != "Provider gagal" {
		t.Fatalf("ai_job.error tidak tersimpan: %+v", got.AIJob)
	}
}

func TestFeedbackForm_CompleteAISuggest_IgnoresStaleCallback(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, _ := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	// User membatalkan sebelum callback tiba — status sudah bukan processing_ai lagi.
	if _, err := svc.ClearAIJob(ctx, f.ID); err != nil {
		t.Fatalf("clear: %v", err)
	}
	content := []byte(`{"confidence":92,"questions":[{"type":"text","label":"Terlambat"}]}`)
	if err := svc.CompleteAISuggest(ctx, f.ID, content, ""); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, _ := repo.GetByID(ctx, f.ID)
	if got.AIJob != nil {
		t.Fatalf("callback basi tidak boleh mengisi ulang ai_job: %+v", got.AIJob)
	}
}

func TestFeedbackForm_SubmitClarifyAnswers_TracksRoundAndHistory(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, _ := svc.StartAISuggest(ctx, "", "buatkan feedback", nil, nil, domain.LangID, nil, nil)
	if f.Status != domain.FormNeedClarification {
		t.Fatalf("prasyarat: status = %q, want need_clarification", f.Status)
	}
	nQuestions := len(f.AIJob.ClarifyingQuestions)
	answers := make([]string, nQuestions)
	for i := range answers {
		answers[i] = "Jawaban " + string(rune('A'+i))
	}
	answered, err := svc.SubmitClarifyAnswers(ctx, f.ID, answers)
	if err != nil {
		t.Fatalf("submit clarify: %v", err)
	}
	if answered.Status != domain.FormProcessingAI {
		t.Fatalf("status = %q, want processing_ai setelah menjawab", answered.Status)
	}
	if answered.AIJob.Round != 1 {
		t.Fatalf("round = %d, want 1", answered.AIJob.Round)
	}
	if len(answered.AIJob.QAHistory) != nQuestions {
		t.Fatalf("qa_history = %d, want %d: %+v", len(answered.AIJob.QAHistory), nQuestions, answered.AIJob.QAHistory)
	}
}

func TestFeedbackForm_SubmitClarifyAnswers_RejectsWithoutPendingClarification(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()
	f, _ := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	if _, err := svc.SubmitClarifyAnswers(ctx, f.ID, []string{"x"}); err == nil {
		t.Fatal("harus ditolak karena tidak ada klarifikasi yang menunggu")
	}
}

func TestFeedbackForm_ClearAIJob_ResetsToDraft(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()
	f, _ := svc.StartAISuggest(ctx, "", "buatkan feedback", nil, nil, domain.LangID, nil, nil)

	cleared, err := svc.ClearAIJob(ctx, f.ID)
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if cleared.Status != domain.FormDraft {
		t.Fatalf("status = %q, want draft", cleared.Status)
	}
	if cleared.AIJob != nil {
		t.Fatalf("ai_job harus nil setelah dibersihkan: %+v", cleared.AIJob)
	}
}

func TestFeedbackForm_ReapStaleAISuggest_FailsOldProcessingForms(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, _ := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)
	// olderThan=0 → cutoff di masa depan sedikit pun sudah melewati UpdatedAt
	// baru saja di-set repo palsu (yang tidak mengelola timestamp otomatis),
	// jadi ini mensimulasikan "job sudah lama mandek".
	if err := svc.ReapStaleAISuggest(ctx, 0); err != nil {
		t.Fatalf("reap: %v", err)
	}
	got, _ := repo.GetByID(ctx, f.ID)
	if got.Status != domain.FormDraft {
		t.Fatalf("status = %q, want draft setelah di-reap", got.Status)
	}
	if got.AIJob == nil || got.AIJob.Error == "" {
		t.Fatalf("ai_job.error harus terisi setelah di-reap: %+v", got.AIJob)
	}
}

func TestFeedbackForm_Update_PreservesInFlightAIStatus(t *testing.T) {
	repo := newFakeFeedbackFormRepo()
	svc := NewFeedbackFormService(repo, nil, "", "")
	ctx := context.Background()

	f, _ := svc.StartAISuggest(ctx, "", "ukur kepuasan client proyek migrasi ERP", nil, nil, domain.LangID, nil, nil)

	// Simpan draft biasa (mis. user mengetik judul) SAAT job AI masih
	// processing_ai — status TIDAK BOLEH jatuh ke draft, atau callback yang
	// datang belakangan akan diabaikan (lihat guard di Complete/CompleteAISuggest).
	updated, err := svc.Update(ctx, f.ID, func(form *domain.FeedbackForm) {
		form.Title = "Judul Baru"
	}, "")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != domain.FormProcessingAI {
		t.Fatalf("status = %q, harus tetap processing_ai", updated.Status)
	}
	if updated.AIJob == nil {
		t.Fatal("ai_job tidak boleh hilang akibat simpan draft biasa")
	}
	if updated.Title != "Judul Baru" {
		t.Fatalf("judul tidak tersimpan: %q", updated.Title)
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
