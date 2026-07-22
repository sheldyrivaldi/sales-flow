package handlers

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// FeedbackFormHandler melayani menu Feedback Client (form builder dinamis) +
// endpoint publik pengisian form (tanpa login) + bantuan AI.
type FeedbackFormHandler struct {
	svc   *service.FeedbackFormService
	ai    *service.FeedbackAIService
	users domain.UserRepository
}

func NewFeedbackFormHandler(svc *service.FeedbackFormService, ai *service.FeedbackAIService, users domain.UserRepository) *FeedbackFormHandler {
	return &FeedbackFormHandler{svc: svc, ai: ai, users: users}
}

// maxSuggestUpload membatasi tiap lampiran konteks AI (dokumen dibaca via vision).
const maxSuggestUpload = 10 << 20 // 10 MB per berkas

// maxSuggestFiles membatasi jumlah lampiran per permintaan saran AI.
const maxSuggestFiles = 5

func parseLanguage(v string) domain.FormLanguage {
	l := domain.FormLanguage(v)
	if l.Valid() {
		return l
	}
	return domain.LangID
}

func applyFormRequest(f *domain.FeedbackForm, req dto.FeedbackFormUpsertRequest) {
	f.Title = req.Title
	f.Description = req.Description
	f.Questions = req.Questions
	f.ProjectID = req.ProjectID
	if req.Language != nil {
		f.Language = parseLanguage(*req.Language)
	}
	if req.CollectEmail != nil {
		f.CollectEmail = *req.CollectEmail
	}
	if f.Questions == nil {
		f.Questions = []domain.FeedbackQuestion{}
	}
}

// List handles GET /api/feedback-forms
func (h *FeedbackFormHandler) List(c echo.Context) error {
	items, err := h.svc.List(c.Request().Context())
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Create handles POST /api/feedback-forms
func (h *FeedbackFormHandler) Create(c echo.Context) error {
	var req dto.FeedbackFormUpsertRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}
	f := &domain.FeedbackForm{CollectEmail: true, Language: domain.LangID}
	applyFormRequest(f, req)
	h.stampCreator(c, f)
	desiredSlug := ""
	if req.Slug != nil {
		desiredSlug = *req.Slug
	}
	created, err := h.svc.Create(c.Request().Context(), f, desiredSlug)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, created)
}

// stampCreator mengisi CreatedBy + CreatedByName dari user terautentikasi
// (dipakai kolom "Dibuat oleh" di daftar). Best-effort: tanpa user, dilewati.
func (h *FeedbackFormHandler) stampCreator(c echo.Context, f *domain.FeedbackForm) {
	f.CreatedBy, f.CreatedByName = h.creatorFields(c)
}

// creatorFields mengembalikan (CreatedBy, CreatedByName) dari user
// terautentikasi — dipakai baik saat Create biasa maupun saat StartAISuggest
// membuat form baru secara implisit. Best-effort: tanpa user dikenali,
// mengembalikan (nil, nil).
func (h *FeedbackFormHandler) creatorFields(c echo.Context) (*string, *string) {
	u, ok := auth.UserFromContext(c)
	if !ok || u.ID == "" {
		return nil, nil
	}
	id := u.ID
	var name *string
	if h.users != nil {
		if usr, err := h.users.GetByID(c.Request().Context(), u.ID); err == nil && usr != nil && usr.Name != "" {
			n := usr.Name
			name = &n
		}
	}
	return &id, name
}

// Get handles GET /api/feedback-forms/:id
func (h *FeedbackFormHandler) Get(c echo.Context) error {
	f, err := h.svc.Get(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, f)
}

// Update handles PUT /api/feedback-forms/:id
func (h *FeedbackFormHandler) Update(c echo.Context) error {
	var req dto.FeedbackFormUpsertRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}
	desiredSlug := ""
	if req.Slug != nil {
		desiredSlug = *req.Slug
	}
	updated, err := h.svc.Update(c.Request().Context(), c.Param("id"), func(f *domain.FeedbackForm) {
		applyFormRequest(f, req)
	}, desiredSlug)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, updated)
}

// Publish handles POST /api/feedback-forms/:id/publish
func (h *FeedbackFormHandler) Publish(c echo.Context) error {
	f, err := h.svc.Publish(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, f)
}

// Delete handles DELETE /api/feedback-forms/:id
func (h *FeedbackFormHandler) Delete(c echo.Context) error {
	if err := h.svc.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Submissions handles GET /api/feedback-forms/:id/submissions
func (h *FeedbackFormHandler) Submissions(c echo.Context) error {
	items, err := h.svc.ListSubmissions(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Analytics handles GET /api/feedback-forms/:id/analytics
func (h *FeedbackFormHandler) Analytics(c echo.Context) error {
	a, err := h.svc.Analytics(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, a)
}

// AnalyticsGlobal handles GET /api/feedback-forms/analytics (lintas-form).
func (h *FeedbackFormHandler) AnalyticsGlobal(c echo.Context) error {
	a, err := h.svc.Analytics(c.Request().Context(), "")
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, a)
}

// readSuggestFiles membaca lampiran multipart untuk saran AI: bentuk jamak
// "files" (utama) plus "file" tunggal demi kompatibilitas. Tiap berkas dibatasi
// maxSuggestUpload, total dibatasi maxSuggestFiles.
func readSuggestFiles(c echo.Context) ([]hermes.AgentDocument, error) {
	var headers []*multipart.FileHeader
	if form, err := c.MultipartForm(); err == nil && form != nil {
		headers = append(headers, form.File["files"]...)
		headers = append(headers, form.File["file"]...)
	}
	if len(headers) > maxSuggestFiles {
		return nil, httperr.NewBadRequest("TOO_MANY_FILES", "maksimal 5 lampiran per permintaan")
	}
	docs := make([]hermes.AgentDocument, 0, len(headers))
	for _, fh := range headers {
		if fh.Size > maxSuggestUpload {
			return nil, httperr.NewBadRequest("FILE_TOO_LARGE", "tiap lampiran maksimal 10 MB")
		}
		src, err := fh.Open()
		if err != nil {
			return nil, httperr.NewBadRequest("FILE_ERROR", "gagal membaca lampiran")
		}
		b, err := io.ReadAll(io.LimitReader(src, maxSuggestUpload))
		_ = src.Close()
		if err != nil {
			return nil, httperr.NewBadRequest("FILE_ERROR", "gagal membaca lampiran")
		}
		docs = append(docs, hermes.AgentDocument{Filename: fh.Filename, Bytes: b})
	}
	return docs, nil
}

// AISuggest handles POST /api/feedback-forms/ai/suggest (multipart: prompt +
// language + type_counts opsional [JSON] + form_id opsional + lampiran
// opsional "files" [banyak]). ASINKRON: form langsung disimpan dengan status
// processing_ai/need_clarification dan dikembalikan seketika (202) — generate
// yang sebenarnya berjalan di background Hermes dan melapor balik lewat
// callback (lihat InternalHandler.CompleteFeedbackAISuggest). form_id kosong
// berarti form belum tersimpan sama sekali — dibuatkan draft baru di sini
// supaya progres tidak hilang bila user pindah halaman.
func (h *FeedbackFormHandler) AISuggest(c echo.Context) error {
	formID := c.FormValue("form_id")
	prompt := c.FormValue("prompt")
	lang := parseLanguage(c.FormValue("language"))
	typeSpec, err := service.ParseSuggestTypeCounts(c.FormValue("type_counts"))
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("INVALID_TYPE_COUNTS", err.Error()))
	}
	docs, err := readSuggestFiles(c)
	if err != nil {
		return httperr.Write(c, err)
	}
	var createdBy, createdByName *string
	if formID == "" {
		createdBy, createdByName = h.creatorFields(c)
	}
	f, err := h.svc.StartAISuggest(c.Request().Context(), formID, prompt, typeSpec, docs, lang, createdBy, createdByName)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusAccepted, f)
}

// AISuggestClarify handles POST /api/feedback-forms/:id/ai/suggest/clarify —
// user menjawab AIJob.ClarifyingQuestions, memicu putaran generate berikutnya.
func (h *FeedbackFormHandler) AISuggestClarify(c echo.Context) error {
	var req dto.FeedbackClarifyAnswersRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	f, err := h.svc.SubmitClarifyAnswers(c.Request().Context(), c.Param("id"), req.Answers)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusAccepted, f)
}

// AISuggestClear handles POST /api/feedback-forms/:id/ai/suggest/clear —
// membuang job saran AI yang sementara (dipanggil FE setelah user menambahkan
// pertanyaan terpilih ke form, atau saat membatalkan alur klarifikasi).
func (h *FeedbackFormHandler) AISuggestClear(c echo.Context) error {
	f, err := h.svc.ClearAIJob(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, f)
}

// AIRefine handles POST /api/feedback-forms/ai/refine — revisi 1 pertanyaan.
func (h *FeedbackFormHandler) AIRefine(c echo.Context) error {
	var req dto.FeedbackRefineRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}
	q := domain.SuggestedQuestion{
		Type:        domain.QuestionType(req.Question.Type),
		Label:       req.Question.Label,
		Description: req.Question.Description,
		Scale:       req.Question.Scale,
		Options:     req.Question.Options,
		Multiple:    req.Question.Multiple,
		MinLabel:    req.Question.MinLabel,
		MaxLabel:    req.Question.MaxLabel,
	}
	lang := domain.LangID
	if req.Language != nil {
		lang = parseLanguage(*req.Language)
	}
	res, err := h.ai.RefineQuestion(c.Request().Context(), q, req.Instruction, lang)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, res)
}

// AIAnalyze handles POST /api/feedback-forms/ai/analyze (opsional ?form_id=).
func (h *FeedbackFormHandler) AIAnalyze(c echo.Context) error {
	formID := c.QueryParam("form_id")
	a, err := h.svc.Analytics(c.Request().Context(), formID)
	if err != nil {
		return httperr.Write(c, err)
	}
	// Bahasa analisa mengikuti bahasa form bila diketahui (else default id).
	lang := domain.LangID
	if formID != "" {
		if f, ferr := h.svc.Get(c.Request().Context(), formID); ferr == nil && f != nil && f.Language.Valid() {
			lang = f.Language
		}
	}
	insight, err := h.ai.AnalyzeFeedback(c.Request().Context(), a, lang)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, insight)
}

// PublicInfo handles GET /api/public/forms/:slug — halaman publik /form/:slug.
func (h *FeedbackFormHandler) PublicInfo(c echo.Context) error {
	f, err := h.svc.PublicGet(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.FeedbackFormPublicResponse{
		Title:        f.Title,
		Description:  f.Description,
		Slug:         f.Slug,
		Language:     f.Language,
		CollectEmail: f.CollectEmail,
		Questions:    f.Questions,
	})
}

// PublicSubmit handles POST /api/public/forms/:slug — client mengisi form.
func (h *FeedbackFormHandler) PublicSubmit(c echo.Context) error {
	var req dto.FeedbackFormSubmitRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	sub := &domain.FeedbackFormSubmission{
		RespondentEmail:    req.RespondentEmail,
		RespondentName:     req.RespondentName,
		RespondentDivision: req.RespondentDivision,
		Answers:            req.Answers,
	}
	if err := h.svc.Submit(c.Request().Context(), c.Param("slug"), sub); err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
}
