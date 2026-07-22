package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
)

// FeedbackFormService mengelola form feedback dinamis (Feedback Client):
// menyusun pertanyaan, terbitkan link publik, terima jawaban client, dan
// agregasi analitiknya. Menggantikan FeedbackService (form kaku 0023) sebagai
// jalur utama; yang lama dibiarkan dorman untuk kompatibilitas link lama.
type FeedbackFormService struct {
	repo domain.FeedbackFormRepository
}

func NewFeedbackFormService(repo domain.FeedbackFormRepository) *FeedbackFormService {
	return &FeedbackFormService{repo: repo}
}

var slugSanitize = regexp.MustCompile(`[^a-z0-9]+`)

// slugify menormalkan teks menjadi slug aman-URL: lowercase, hanya [a-z0-9-],
// tanpa tanda hubung menggantung.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugSanitize.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// uniqueSlug memastikan slug unik. Bila kosong, diturunkan dari judul; bila
// tetap kosong setelah sanitasi dipakai fallback acak. Bentrok ditambah
// sufiks -2, -3, … Pengecualian: excludeID dilewati agar Update pada form yang
// sama tidak dianggap bentrok dengan dirinya sendiri.
func (s *FeedbackFormService) uniqueSlug(ctx context.Context, desired, title string, currentSlug string) (string, error) {
	base := slugify(desired)
	if base == "" {
		base = slugify(title)
	}
	if base == "" {
		base = "form-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	}
	// Tidak berubah dari nilai form saat ini → tak perlu cek ulang.
	if base == currentSlug {
		return base, nil
	}
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// normalizeQuestions memberi ID pada pertanyaan yang belum punya, memangkas
// spasi, dan menerapkan default (Scale=5 untuk rating). Mengembalikan error
// validasi bila tipe/label tidak sah.
func normalizeQuestions(qs []domain.FeedbackQuestion) ([]domain.FeedbackQuestion, error) {
	out := make([]domain.FeedbackQuestion, 0, len(qs))
	for _, q := range qs {
		q.Label = strings.TrimSpace(q.Label)
		if q.Label == "" {
			return nil, httperr.NewBadRequest("INVALID_QUESTION", "setiap pertanyaan wajib punya label")
		}
		if !q.Type.Valid() {
			return nil, httperr.NewBadRequest("INVALID_QUESTION", fmt.Sprintf("tipe pertanyaan %q tidak dikenal", q.Type))
		}
		if q.ID == "" {
			q.ID = "q" + strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
		}
		switch q.Type {
		case domain.QuestionRating:
			if q.Scale < 2 || q.Scale > 10 {
				q.Scale = 5
			}
			q.Options, q.Multiple = nil, false
			q.MinLabel = strings.TrimSpace(q.MinLabel)
			q.MaxLabel = strings.TrimSpace(q.MaxLabel)
		case domain.QuestionNPS:
			q.Scale, q.Options, q.Multiple = 0, nil, false
			q.MinLabel, q.MaxLabel = "", ""
		case domain.QuestionChoice:
			cleaned := make([]string, 0, len(q.Options))
			for _, o := range q.Options {
				if o = strings.TrimSpace(o); o != "" {
					cleaned = append(cleaned, o)
				}
			}
			if len(cleaned) == 0 {
				return nil, httperr.NewBadRequest("INVALID_QUESTION", fmt.Sprintf("pertanyaan pilihan %q butuh minimal satu opsi", q.Label))
			}
			q.Options, q.Scale = cleaned, 0
			q.MinLabel, q.MaxLabel = "", ""
		case domain.QuestionText:
			q.Scale, q.Options, q.Multiple = 0, nil, false
			q.MinLabel, q.MaxLabel = "", ""
		}
		out = append(out, q)
	}
	return out, nil
}

// Create membuat form baru (status draft). Slug diturunkan/di-unik-kan.
func (s *FeedbackFormService) Create(ctx context.Context, f *domain.FeedbackForm, desiredSlug string) (*domain.FeedbackForm, error) {
	qs, err := normalizeQuestions(f.Questions)
	if err != nil {
		return nil, err
	}
	f.Questions = qs
	slug, err := s.uniqueSlug(ctx, desiredSlug, f.Title, "")
	if err != nil {
		return nil, err
	}
	f.Slug = slug
	if f.Status == "" {
		f.Status = domain.FormDraft
	}
	if !f.Language.Valid() {
		f.Language = domain.LangID
	}
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// Update menyimpan perubahan form. desiredSlug kosong berarti pertahankan slug
// sekarang.
func (s *FeedbackFormService) Update(ctx context.Context, id string, apply func(*domain.FeedbackForm), desiredSlug string) (*domain.FeedbackForm, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, httperr.NewNotFound("form feedback tidak ditemukan")
	}
	apply(f)
	qs, err := normalizeQuestions(f.Questions)
	if err != nil {
		return nil, err
	}
	f.Questions = qs
	if !f.Language.Valid() {
		f.Language = domain.LangID
	}
	// Menyimpan perubahan mengembalikan form ke draft: konten berubah maka link
	// publik lama tidak boleh menyajikan versi baru sampai diterbitkan ulang
	// (request: edit form terbit → status kembali draft, tombol Terbitkan muncul).
	f.Status = domain.FormDraft
	if strings.TrimSpace(desiredSlug) != "" {
		slug, err := s.uniqueSlug(ctx, desiredSlug, f.Title, f.Slug)
		if err != nil {
			return nil, err
		}
		f.Slug = slug
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// Publish menerbitkan form (draft→published). Wajib punya minimal satu pertanyaan.
func (s *FeedbackFormService) Publish(ctx context.Context, id string) (*domain.FeedbackForm, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, httperr.NewNotFound("form feedback tidak ditemukan")
	}
	if len(f.Questions) == 0 {
		return nil, httperr.NewBadRequest("EMPTY_FORM", "tambahkan minimal satu pertanyaan sebelum menerbitkan form")
	}
	f.Status = domain.FormPublished
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *FeedbackFormService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *FeedbackFormService) Get(ctx context.Context, id string) (*domain.FeedbackForm, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, httperr.NewNotFound("form feedback tidak ditemukan")
	}
	return f, nil
}

func (s *FeedbackFormService) List(ctx context.Context) ([]domain.FeedbackForm, error) {
	return s.repo.List(ctx)
}

// PublicGet mengembalikan form yang boleh diisi client (harus published).
func (s *FeedbackFormService) PublicGet(ctx context.Context, slug string) (*domain.FeedbackForm, error) {
	f, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if f == nil || f.Status != domain.FormPublished {
		return nil, httperr.NewNotFound("form feedback tidak ditemukan atau belum aktif")
	}
	return f, nil
}

// Submit menyimpan jawaban client setelah validasi terhadap definisi pertanyaan.
func (s *FeedbackFormService) Submit(ctx context.Context, slug string, sub *domain.FeedbackFormSubmission) error {
	f, err := s.PublicGet(ctx, slug)
	if err != nil {
		return err
	}
	answers, err := validateAnswers(f, sub.Answers)
	if err != nil {
		return err
	}
	// Email & nama pengisi selalu wajib (seperti Google Form); divisi opsional.
	if sub.RespondentEmail == nil || strings.TrimSpace(*sub.RespondentEmail) == "" {
		return httperr.NewBadRequest("EMAIL_REQUIRED", "email wajib diisi")
	}
	if sub.RespondentName == nil || strings.TrimSpace(*sub.RespondentName) == "" {
		return httperr.NewBadRequest("NAME_REQUIRED", "nama wajib diisi")
	}
	sub.FormID = f.ID
	sub.Answers = answers
	return s.repo.CreateSubmission(ctx, sub)
}

// validateAnswers memeriksa jawaban wajib terisi & nilai berada dalam rentang
// yang sah, lalu mengembalikan jawaban yang sudah dibersihkan (dipetakan by
// question ID sehingga jawaban liar dibuang).
func validateAnswers(f *domain.FeedbackForm, given []domain.FeedbackAnswer) ([]domain.FeedbackAnswer, error) {
	byQ := make(map[string]domain.FeedbackAnswer, len(given))
	for _, a := range given {
		byQ[a.QuestionID] = a
	}
	out := make([]domain.FeedbackAnswer, 0, len(f.Questions))
	for _, q := range f.Questions {
		a, ok := byQ[q.ID]
		answered := false
		switch q.Type {
		case domain.QuestionText:
			a.Text = strings.TrimSpace(a.Text)
			answered = a.Text != ""
			a.Rating, a.Choice = nil, nil
		case domain.QuestionRating:
			if a.Rating != nil {
				if *a.Rating < 1 || *a.Rating > q.Scale {
					return nil, httperr.NewBadRequest("INVALID_ANSWER", fmt.Sprintf("nilai untuk %q harus 1..%d", q.Label, q.Scale))
				}
				answered = true
			}
			a.Text, a.Choice = "", nil
		case domain.QuestionNPS:
			if a.Rating != nil {
				if *a.Rating < 0 || *a.Rating > 10 {
					return nil, httperr.NewBadRequest("INVALID_ANSWER", fmt.Sprintf("skor untuk %q harus 0..10", q.Label))
				}
				answered = true
			}
			a.Text, a.Choice = "", nil
		case domain.QuestionChoice:
			valid := make(map[string]struct{}, len(q.Options))
			for _, o := range q.Options {
				valid[o] = struct{}{}
			}
			cleaned := make([]string, 0, len(a.Choice))
			for _, c := range a.Choice {
				if _, okc := valid[c]; !okc {
					return nil, httperr.NewBadRequest("INVALID_ANSWER", fmt.Sprintf("pilihan untuk %q tidak valid", q.Label))
				}
				cleaned = append(cleaned, c)
			}
			if !q.Multiple && len(cleaned) > 1 {
				return nil, httperr.NewBadRequest("INVALID_ANSWER", fmt.Sprintf("%q hanya boleh satu pilihan", q.Label))
			}
			a.Choice = cleaned
			answered = len(cleaned) > 0
			a.Text, a.Rating = "", nil
		}
		if q.Required && !answered {
			return nil, httperr.NewBadRequest("REQUIRED_ANSWER", fmt.Sprintf("pertanyaan %q wajib diisi", q.Label))
		}
		if ok || answered {
			a.QuestionID = q.ID
			out = append(out, a)
		}
	}
	return out, nil
}

func (s *FeedbackFormService) ListSubmissions(ctx context.Context, formID string) ([]domain.FeedbackFormSubmission, error) {
	return s.repo.ListSubmissions(ctx, formID)
}

// --- Analitik ---

// QuestionStat adalah agregat satu pertanyaan.
type QuestionStat struct {
	QuestionID   string         `json:"question_id"`
	Label        string         `json:"label"`
	Type         string         `json:"type"`
	Responses    int            `json:"responses"`
	Average      float64        `json:"average,omitempty"`      // rating/nps
	Distribution []int          `json:"distribution,omitempty"` // rating: index 0 = bintang 1
	Choices      map[string]int `json:"choices,omitempty"`      // choice: opsi → jumlah
	Texts        []string       `json:"texts,omitempty"`        // text: kumpulan jawaban
}

// FormAnalytics adalah agregat untuk satu form atau lintas-form.
type FormAnalytics struct {
	FormID           string         `json:"form_id,omitempty"`
	TotalForms       int            `json:"total_forms"`
	TotalSubmissions int            `json:"total_submissions"`
	AvgRating        float64        `json:"avg_rating"` // rata-rata semua jawaban rating (skala dinormalkan ke 5)
	NPS              int            `json:"nps"`        // -100..100 dari pertanyaan NPS bila ada
	Questions        []QuestionStat `json:"questions"`
}

// Analytics mengagregasi satu form (formID != "") atau seluruh form (formID == "").
func (s *FeedbackFormService) Analytics(ctx context.Context, formID string) (*FormAnalytics, error) {
	if formID != "" {
		f, err := s.Get(ctx, formID)
		if err != nil {
			return nil, err
		}
		subs, err := s.repo.ListSubmissions(ctx, formID)
		if err != nil {
			return nil, err
		}
		a := aggregate([]domain.FeedbackForm{*f}, subs)
		a.FormID = formID
		return a, nil
	}
	forms, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	subs, err := s.repo.ListAllSubmissions(ctx)
	if err != nil {
		return nil, err
	}
	return aggregate(forms, subs), nil
}

// aggregate menghitung statistik per-pertanyaan lintas kumpulan form + submission.
func aggregate(forms []domain.FeedbackForm, subs []domain.FeedbackFormSubmission) *FormAnalytics {
	// Kumpulkan definisi pertanyaan (unik by ID) & scale-nya.
	type qdef struct {
		q     domain.FeedbackQuestion
		order int
	}
	defs := map[string]*qdef{}
	order := 0
	for _, f := range forms {
		for _, q := range f.Questions {
			if _, ok := defs[q.ID]; !ok {
				defs[q.ID] = &qdef{q: q, order: order}
				order++
			}
		}
	}

	stats := map[string]*QuestionStat{}
	var ratingSum float64
	var ratingN int
	var npsPromoters, npsDetractors, npsN int

	for _, sub := range subs {
		for _, ans := range sub.Answers {
			d, ok := defs[ans.QuestionID]
			if !ok {
				continue
			}
			st := stats[ans.QuestionID]
			if st == nil {
				st = &QuestionStat{QuestionID: ans.QuestionID, Label: d.q.Label, Type: string(d.q.Type)}
				if d.q.Type == domain.QuestionRating {
					st.Distribution = make([]int, d.q.Scale)
				} else if d.q.Type == domain.QuestionChoice {
					st.Choices = map[string]int{}
				}
				stats[ans.QuestionID] = st
			}
			switch d.q.Type {
			case domain.QuestionRating:
				if ans.Rating != nil && *ans.Rating >= 1 && *ans.Rating <= d.q.Scale {
					st.Responses++
					st.Average += float64(*ans.Rating)
					st.Distribution[*ans.Rating-1]++
					// Normalkan ke skala 5 untuk rata-rata global.
					ratingSum += float64(*ans.Rating) / float64(d.q.Scale) * 5
					ratingN++
				}
			case domain.QuestionNPS:
				if ans.Rating != nil && *ans.Rating >= 0 && *ans.Rating <= 10 {
					st.Responses++
					st.Average += float64(*ans.Rating)
					npsN++
					if *ans.Rating >= 9 {
						npsPromoters++
					} else if *ans.Rating <= 6 {
						npsDetractors++
					}
				}
			case domain.QuestionChoice:
				if len(ans.Choice) > 0 {
					st.Responses++
					for _, c := range ans.Choice {
						st.Choices[c]++
					}
				}
			case domain.QuestionText:
				if t := strings.TrimSpace(ans.Text); t != "" {
					st.Responses++
					st.Texts = append(st.Texts, t)
				}
			}
		}
	}

	// Finalisasi rata-rata per-pertanyaan.
	for _, st := range stats {
		if st.Responses > 0 && (st.Type == string(domain.QuestionRating) || st.Type == string(domain.QuestionNPS)) {
			st.Average = round1(st.Average / float64(st.Responses))
		}
	}

	// Urutkan pertanyaan sesuai urutan definisi.
	ordered := make([]QuestionStat, 0, len(defs))
	// pastikan semua pertanyaan muncul (meski 0 respon)
	for id, d := range defs {
		if _, ok := stats[id]; !ok {
			empty := QuestionStat{QuestionID: id, Label: d.q.Label, Type: string(d.q.Type)}
			if d.q.Type == domain.QuestionRating {
				empty.Distribution = make([]int, d.q.Scale)
			} else if d.q.Type == domain.QuestionChoice {
				empty.Choices = map[string]int{}
			}
			stats[id] = &empty
		}
	}
	idsByOrder := make([]string, len(defs))
	for id, d := range defs {
		idsByOrder[d.order] = id
	}
	for _, id := range idsByOrder {
		if st := stats[id]; st != nil {
			ordered = append(ordered, *st)
		}
	}

	a := &FormAnalytics{
		TotalForms:       len(forms),
		TotalSubmissions: len(subs),
		Questions:        ordered,
	}
	if ratingN > 0 {
		a.AvgRating = round1(ratingSum / float64(ratingN))
	}
	if npsN > 0 {
		a.NPS = (npsPromoters*100 - npsDetractors*100) / npsN
	}
	return a
}
