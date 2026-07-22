package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/httperr"
)

// FeedbackFormService mengelola form feedback dinamis (Feedback Client):
// menyusun pertanyaan, terbitkan link publik, terima jawaban client, dan
// agregasi analitiknya. Menggantikan FeedbackService (form kaku 0023) sebagai
// jalur utama; yang lama dibiarkan dorman untuk kompatibilitas link lama.
//
// Saran AI (StartAISuggest dkk di bawah) memakai model TITIP-TUGAS yang sama
// dengan playbook/analisa event: runner+callbackBase+callbackSecret dipakai
// fire-and-forget ke Hermes lewat /v1/agent-task, hasil dilaporkan balik lewat
// callback (lihat CompleteAISuggest + internal/http/handlers/internal_handler.go).
// nil-safe: runner nil → job langsung ditandai gagal, bukan diam-diam macet.
type FeedbackFormService struct {
	repo           domain.FeedbackFormRepository
	runner         hermes.AgentTaskRunner
	callbackBase   string
	callbackSecret string
}

func NewFeedbackFormService(repo domain.FeedbackFormRepository, runner hermes.AgentTaskRunner, callbackBase, callbackSecret string) *FeedbackFormService {
	return &FeedbackFormService{repo: repo, runner: runner, callbackBase: callbackBase, callbackSecret: callbackSecret}
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
	// KECUALI sedang ada job saran AI berjalan/menunggu klarifikasi — simpan
	// draft biasa (mis. user mengedit pertanyaan lain sambil AI masih
	// memproses di background) TIDAK BOLEH menjatuhkan status itu, karena
	// CompleteAISuggest mengecek status ini persis untuk menerapkan hasilnya.
	if f.Status != domain.FormProcessingAI && f.Status != domain.FormNeedClarification {
		f.Status = domain.FormDraft
	}
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

// --- Saran AI (async, model titip-tugas) ---

// deriveFormTitle mengambil judul ringkas dari prompt AI — dipakai saat user
// langsung minta saran AI dari form yang belum punya judul sama sekali
// (form baru dibuat otomatis di sini demi persistensi; judulnya akan ditimpa
// begitu user mengisi kolom judul dan menyimpan draft seperti biasa).
func deriveFormTitle(prompt string) string {
	t := strings.TrimSpace(prompt)
	if i := strings.IndexAny(t, "\n."); i > 0 {
		t = t[:i]
	}
	t = strings.TrimSpace(t)
	if len(t) > 80 {
		t = t[:80] + "…"
	}
	if t == "" {
		return "Form Feedback (draft)"
	}
	return t
}

// StartAISuggest memulai permintaan saran AI. formID kosong berarti form
// BELUM tersimpan — dibuatkan draft baru di sini SEBELUM apa pun lain
// terjadi, supaya progres tidak hilang bila user pindah halaman (generate
// bisa makan waktu lama). Prompt yang jelas-jelas terlalu tipis ditangani
// deterministik (langsung need_clarification) tanpa memanggil AI sama
// sekali; selebihnya instruksi dititipkan ke Hermes fire-and-forget dan
// hasilnya diterapkan lewat CompleteAISuggest saat callback tiba.
func (s *FeedbackFormService) StartAISuggest(
	ctx context.Context,
	formID, prompt string,
	typeSpec map[domain.QuestionType]TypeCountSpec,
	files []hermes.AgentDocument,
	lang domain.FormLanguage,
	createdBy, createdByName *string,
) (*domain.FeedbackForm, error) {
	if !lang.Valid() {
		lang = domain.LangID
	}

	isNew := formID == ""
	var f *domain.FeedbackForm
	if !isNew {
		existing, err := s.repo.GetByID(ctx, formID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, httperr.NewNotFound("form feedback tidak ditemukan")
		}
		if existing.Status == domain.FormProcessingAI {
			return nil, httperr.NewBadRequest("AI_BUSY", "AI sedang memproses form ini, tunggu sampai selesai")
		}
		f = existing
	} else {
		f = &domain.FeedbackForm{
			Title:         deriveFormTitle(prompt),
			CollectEmail:  true,
			Language:      lang,
			Status:        domain.FormDraft,
			Questions:     []domain.FeedbackQuestion{},
			CreatedBy:     createdBy,
			CreatedByName: createdByName,
		}
	}

	job := &domain.FeedbackAIJob{
		Prompt:     prompt,
		Language:   lang,
		Round:      0,
		TypeCounts: typeCountsToStrings(typeSpec),
		UpdatedAt:  time.Now(),
	}

	// Prompt terlalu tipis (tanpa lampiran) — LLM cenderung terlalu percaya
	// diri saat menilai confidence sendiri, jadi kasus ini ditangani pasti di
	// sini, bukan diserahkan ke penilaian model (lihat looksTooVagueForSuggest).
	if looksTooVagueForSuggest(prompt, len(files) > 0) {
		job.Confidence = 40
		job.ClarifyingQuestions = defaultClarifyingQuestions(lang)
		f.AIJob = job
		f.Status = domain.FormNeedClarification
		if err := s.persistAIJobForm(ctx, isNew, f); err != nil {
			return nil, err
		}
		return f, nil
	}

	f.AIJob = job
	f.Status = domain.FormProcessingAI
	if err := s.persistAIJobForm(ctx, isNew, f); err != nil {
		return nil, err
	}

	instruction := buildSuggestPrompt(prompt, lang, typeSpec)
	go s.dispatchAISuggest(f.ID, instruction, files)
	return f, nil
}

// persistAIJobForm creates a brand-new form (isNew) or updates an existing
// one — every AI-job transition goes through here so callers don't branch.
func (s *FeedbackFormService) persistAIJobForm(ctx context.Context, isNew bool, f *domain.FeedbackForm) error {
	if isNew {
		slug, err := s.uniqueSlug(ctx, "", f.Title, "")
		if err != nil {
			return err
		}
		f.Slug = slug
		return s.repo.Create(ctx, f)
	}
	return s.repo.Update(ctx, f)
}

// dispatchAISuggest menitipkan instruksi + lampiran ke Hermes (fire-and-forget,
// TIDAK memakai context request — request sudah selesai & sudah balas ke FE).
// Hasil dilaporkan balik lewat callback (CompleteAISuggest).
func (s *FeedbackFormService) dispatchAISuggest(formID, instruction string, docs []hermes.AgentDocument) {
	if s.runner == nil {
		s.markAISuggestFailed(formID, "AI agent tidak dikonfigurasi.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	task := hermes.AgentTask{
		Instruction:    instruction,
		JobID:          formID,
		CallbackURL:    fmt.Sprintf("%s/internal/feedback-forms/%s/ai-suggest-complete", strings.TrimRight(s.callbackBase, "/"), formID),
		CallbackSecret: s.callbackSecret,
		Documents:      docs,
	}
	if err := s.runner.RunAgentTask(ctx, task); err != nil {
		log.Printf("feedback ai suggest %s: gagal menitipkan tugas ke AI: %v", formID, err)
		s.markAISuggestFailed(formID, "Gagal mengirim tugas ke AI. Coba lagi.")
	}
}

// markAISuggestFailed menandai job gagal HANYA bila form masih processing_ai
// (jangan menimpa hasil yang mungkin sudah dilaporkan lebih dulu, atau job
// yang sudah dibatalkan user — race yang sangat jarang, sama seperti playbook).
func (s *FeedbackFormService) markAISuggestFailed(formID, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	f, err := s.repo.GetByID(ctx, formID)
	if err != nil || f == nil || f.Status != domain.FormProcessingAI {
		return
	}
	f.Status = domain.FormDraft
	if f.AIJob == nil {
		f.AIJob = &domain.FeedbackAIJob{}
	}
	f.AIJob.Error = reason
	f.AIJob.ClarifyingQuestions = nil
	f.AIJob.PendingQuestions = nil
	f.AIJob.UpdatedAt = time.Now()
	_ = s.repo.Update(ctx, f)
}

// CompleteAISuggest dipanggil callback bridge saat generate selesai (sukses
// atau gagal) — content kosong berarti gagal (errMsg berisi alasan). Bila
// form sudah tidak lagi processing_ai (dibatalkan user, atau callback
// duplikat), hasil diabaikan begitu saja alih-alih menimpa state yang lebih
// baru.
func (s *FeedbackFormService) CompleteAISuggest(ctx context.Context, formID string, content []byte, errMsg string) error {
	f, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return err
	}
	if f == nil {
		return httperr.NewNotFound("form feedback tidak ditemukan")
	}
	if f.Status != domain.FormProcessingAI {
		return nil
	}
	if f.AIJob == nil {
		f.AIJob = &domain.FeedbackAIJob{}
	}
	f.AIJob.UpdatedAt = time.Now()

	if len(content) == 0 {
		if errMsg == "" {
			errMsg = "Generate saran AI gagal."
		}
		f.Status = domain.FormDraft
		f.AIJob.Error = errMsg
		f.AIJob.ClarifyingQuestions = nil
		f.AIJob.PendingQuestions = nil
		return s.repo.Update(ctx, f)
	}

	confidence, clarify, questions, perr := evaluateSuggestJSON(content)
	if perr != nil {
		f.Status = domain.FormDraft
		f.AIJob.Error = "Output AI tidak valid."
		f.AIJob.ClarifyingQuestions = nil
		f.AIJob.PendingQuestions = nil
		return s.repo.Update(ctx, f)
	}

	f.AIJob.Confidence = confidence
	f.AIJob.Error = ""
	if len(clarify) > 0 {
		f.Status = domain.FormNeedClarification
		f.AIJob.ClarifyingQuestions = clarify
		f.AIJob.PendingQuestions = nil
	} else {
		f.Status = domain.FormDraft
		f.AIJob.ClarifyingQuestions = nil
		f.AIJob.PendingQuestions = questions
	}
	return s.repo.Update(ctx, f)
}

// SubmitClarifyAnswers menerima jawaban user atas AIJob.ClarifyingQuestions,
// menambahkannya ke riwayat, lalu menitipkan ULANG instruksi ke Hermes dengan
// konteks yang sudah diperjelas. Dibatasi maxClarifyRounds putaran — pada
// putaran terakhir AI dipaksa menyusun kuesioner terbaik dari info yang ada
// alih-alih terus bertanya.
func (s *FeedbackFormService) SubmitClarifyAnswers(ctx context.Context, formID string, answers []string) (*domain.FeedbackForm, error) {
	f, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, httperr.NewNotFound("form feedback tidak ditemukan")
	}
	if f.AIJob == nil || len(f.AIJob.ClarifyingQuestions) == 0 || f.Status != domain.FormNeedClarification {
		return nil, httperr.NewBadRequest("NO_CLARIFICATION", "tidak ada pertanyaan klarifikasi yang menunggu jawaban")
	}

	for i, q := range f.AIJob.ClarifyingQuestions {
		a := "(tidak dijawab)"
		if i < len(answers) {
			if t := strings.TrimSpace(answers[i]); t != "" {
				a = t
			}
		}
		f.AIJob.QAHistory = append(f.AIJob.QAHistory, domain.FeedbackClarifyQA{Question: q, Answer: a})
	}
	nextRound := f.AIJob.Round + 1
	forceFinal := nextRound >= maxClarifyRounds

	f.AIJob.Round = nextRound
	f.AIJob.ClarifyingQuestions = nil
	f.AIJob.Confidence = 0
	f.AIJob.Error = ""
	f.AIJob.UpdatedAt = time.Now()
	f.Status = domain.FormProcessingAI
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}

	effectivePrompt := composeClarifyPrompt(f.AIJob.Prompt, f.AIJob.QAHistory, forceFinal)
	typeSpec := typeCountsFromStrings(f.AIJob.TypeCounts)
	instruction := buildSuggestPrompt(effectivePrompt, f.AIJob.Language, typeSpec)
	// Lampiran TIDAK diulang di putaran klarifikasi: berkas mentah sengaja
	// tidak disimpan di job (JSONB, bukan tempat untuk blob biner) — riwayat
	// tanya-jawab dianggap cukup memperjelas konteks pada putaran lanjutan.
	go s.dispatchAISuggest(f.ID, instruction, nil)
	return f, nil
}

// composeClarifyPrompt menyusun ulang prompt dengan riwayat tanya-jawab
// terlampir, supaya AI tidak kehilangan konteks dari putaran sebelumnya.
func composeClarifyPrompt(prompt string, history []domain.FeedbackClarifyQA, forceFinal bool) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(prompt))
	if len(history) > 0 {
		b.WriteString("\n\nJawaban klarifikasi dari user:\n")
		for _, h := range history {
			fmt.Fprintf(&b, "- %s\n  Jawaban: %s\n", h.Question, h.Answer)
		}
	}
	if forceFinal {
		b.WriteString("\n(Putaran klarifikasi sudah maksimal. Tetap susun pertanyaan terbaik dari info yang tersedia, JANGAN minta klarifikasi lagi.)")
	}
	return b.String()
}

// ClearAIJob membuang job AI yang sementara secara bisnis — dipanggil SETELAH
// user memilih pertanyaan yang mau ditambahkan ke form (FE menambahkannya ke
// state lokal lalu memanggil ini untuk bersih-bersih), ATAU saat user
// membatalkan alur klarifikasi. Form kembali ke draft bila statusnya masih
// processing_ai/need_clarification.
func (s *FeedbackFormService) ClearAIJob(ctx context.Context, formID string) (*domain.FeedbackForm, error) {
	f, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, httperr.NewNotFound("form feedback tidak ditemukan")
	}
	f.AIJob = nil
	if f.Status == domain.FormProcessingAI || f.Status == domain.FormNeedClarification {
		f.Status = domain.FormDraft
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// ReapStaleAISuggest menandai job saran AI yang mandek (processing_ai lebih
// lama dari olderThan) sebagai gagal — jaring pengaman bila Hermes tak pernah
// melapor balik. Dipanggil berkala via ticker, dan sekali di boot dengan
// olderThan=0 untuk menyapu job yang terputus saat server restart (goroutine
// dispatch tidak selamat dari restart, beda dari state di DB).
func (s *FeedbackFormService) ReapStaleAISuggest(ctx context.Context, olderThan time.Duration) error {
	forms, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-olderThan)
	for i := range forms {
		f := forms[i]
		if f.Status != domain.FormProcessingAI || f.UpdatedAt.After(cutoff) {
			continue
		}
		f.Status = domain.FormDraft
		if f.AIJob == nil {
			f.AIJob = &domain.FeedbackAIJob{}
		}
		f.AIJob.Error = "Waktu habis menunggu AI menyelesaikan saran. Coba lagi."
		f.AIJob.ClarifyingQuestions = nil
		f.AIJob.PendingQuestions = nil
		f.AIJob.UpdatedAt = time.Now()
		if err := s.repo.Update(ctx, &f); err != nil {
			log.Printf("feedback ai suggest reaper: gagal menandai form %s: %v", f.ID, err)
		}
	}
	return nil
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
