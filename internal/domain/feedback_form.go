package domain

import (
	"context"
	"time"
)

// QuestionType adalah jenis field pertanyaan pada form feedback dinamis.
type QuestionType string

const (
	QuestionRating QuestionType = "rating" // bintang 1..Scale (default 5)
	QuestionText   QuestionType = "text"   // jawaban teks bebas
	QuestionChoice QuestionType = "choice" // pilihan dari Options (single/multiple)
	QuestionNPS    QuestionType = "nps"    // skor 0..10 (Net Promoter Score)
)

func (t QuestionType) Valid() bool {
	switch t {
	case QuestionRating, QuestionText, QuestionChoice, QuestionNPS:
		return true
	}
	return false
}

// FeedbackQuestion adalah satu pertanyaan dalam form. Disimpan sebagai elemen
// JSONB pada feedback_form.questions — bukan tabel tersendiri, mengikuti pola
// project.milestones. ID pendek stabil (dibuat FE/BE) agar jawaban bisa
// menautkan ke pertanyaan meski urutan berubah.
type FeedbackQuestion struct {
	ID          string       `json:"id"`
	Type        QuestionType `json:"type"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Required    bool         `json:"required"`
	// Scale hanya untuk rating (default 5). Options+Multiple hanya untuk choice.
	Scale    int      `json:"scale,omitempty"`
	Options  []string `json:"options,omitempty"`
	Multiple bool     `json:"multiple,omitempty"`
	// MinLabel/MaxLabel hanya untuk rating: keterangan ujung skala kiri (nilai
	// terendah) & kanan (tertinggi). Kosong → FE memakai default sesuai bahasa.
	MinLabel string `json:"min_label,omitempty"`
	MaxLabel string `json:"max_label,omitempty"`
}

// FeedbackAnswer adalah jawaban satu pertanyaan. Hanya satu field nilai yang
// terisi sesuai tipe pertanyaan (Text untuk text, Rating untuk rating/nps,
// Choice untuk choice).
type FeedbackAnswer struct {
	QuestionID string   `json:"question_id"`
	Text       string   `json:"text,omitempty"`
	Rating     *int     `json:"rating,omitempty"`
	Choice     []string `json:"choice,omitempty"`
}

// FormLanguage adalah bahasa isi form (memengaruhi bahasa saran AI & label
// default rating pada halaman publik).
type FormLanguage string

const (
	LangID FormLanguage = "id" // Bahasa Indonesia (default)
	LangEN FormLanguage = "en" // English
)

func (l FormLanguage) Valid() bool { return l == LangID || l == LangEN }

// FeedbackFormStatus adalah siklus hidup form. processing_ai/need_clarification
// menumpang di enum yang sama (tampil di badge Status yang sama pada daftar)
// selama saran AI berjalan async — lihat FeedbackAIJob.
type FeedbackFormStatus string

const (
	FormDraft     FeedbackFormStatus = "draft"
	FormPublished FeedbackFormStatus = "published"
	FormClosed    FeedbackFormStatus = "closed"
	// FormProcessingAI: job saran AI sedang dititipkan ke Hermes (fire-and-forget),
	// menunggu callback. Form tetap bisa diedit; Update() TIDAK menimpa status ini
	// (lihat feedback_form_service.go) supaya simpan draft biasa tidak menjatuhkan
	// hasil AI yang sedang diproses di background.
	FormProcessingAI FeedbackFormStatus = "processing_ai"
	// FormNeedClarification: AI menilai konteks belum cukup jelas dan menunggu
	// user menjawab AIJob.ClarifyingQuestions sebelum melanjutkan.
	FormNeedClarification FeedbackFormStatus = "need_clarification"
)

// SuggestedQuestion adalah satu pertanyaan usulan AI — kandidat yang BELUM
// masuk ke Questions sampai user memilih & menambahkannya (tanpa ID/Required,
// beda dari FeedbackQuestion yang sudah jadi bagian form).
type SuggestedQuestion struct {
	Type        QuestionType `json:"type"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Scale       int          `json:"scale,omitempty"`
	Options     []string     `json:"options,omitempty"`
	Multiple    bool         `json:"multiple,omitempty"`
	MinLabel    string       `json:"min_label,omitempty"`
	MaxLabel    string       `json:"max_label,omitempty"`
}

// FeedbackClarifyQA adalah satu pasang tanya-jawab klarifikasi (AI bertanya ke
// USER, bukan ke responden) — riwayatnya dikirim ulang ke AI di putaran
// berikutnya agar konteks yang sudah diperjelas tidak hilang.
type FeedbackClarifyQA struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// FeedbackAIJob adalah state saran AI yang SEMENTARA secara bisnis (dibuang
// begitu user memilih pertanyaan yang mau ditambahkan, atau membatalkan) tapi
// harus PERSISTEN secara teknis: generate bisa makan waktu lama, dan user
// boleh meninggalkan halaman builder tanpa kehilangan progres. Disimpan
// sebagai JSONB pada baris form yang sama (bukan tabel job terpisah) —
// mengikuti pola playbook_job/event_analysis tapi tanpa tabel job berdiri
// sendiri karena job ini SELALU tertaut ke satu form.
type FeedbackAIJob struct {
	// Prompt adalah kebutuhan ASLI dari user (tidak berubah antar-putaran);
	// QAHistory ditambahkan terpisah supaya prompt asli tetap bisa dibaca ulang.
	Prompt   string       `json:"prompt"`
	Language FormLanguage `json:"language"`
	// Round: 0 = permintaan pertama; bertambah tiap kali user menjawab
	// klarifikasi. Dipakai membatasi jumlah putaran klarifikasi.
	Round     int                 `json:"round"`
	QAHistory []FeedbackClarifyQA `json:"qa_history,omitempty"`
	// TypeCounts: konfigurasi tipe & jumlah pertanyaan dari user (opsional),
	// disimpan agar putaran klarifikasi berikutnya memakai konfigurasi yang
	// sama tanpa user mengulang input. Nilai per key: "random" atau angka
	// (mis. "3"). Key yang tidak ada berarti tipe itu bebas ditentukan AI.
	TypeCounts map[QuestionType]string `json:"type_counts,omitempty"`

	// Confidence + ClarifyingQuestions terisi ketika AI menilai konteks belum
	// cukup jelas (lihat internal/service/feedback_ai_service.go). Kosong
	// berarti tidak sedang menunggu klarifikasi.
	Confidence          int      `json:"confidence"`
	ClarifyingQuestions []string `json:"clarifying_questions,omitempty"`
	// PendingQuestions: opsi hasil generate AI menunggu dipilih user — bagian
	// PALING sementara dari job ini, dihapus (seluruh AIJob di-nil-kan) begitu
	// user menambahkan pilihannya ke form.
	PendingQuestions []SuggestedQuestion `json:"pending_questions,omitempty"`
	// Error terisi bila putaran generate terakhir gagal (network/AI down);
	// form tetap kembali ke draft supaya user tidak macet, tapi pesannya
	// ditampilkan agar user tahu perlu mencoba lagi.
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FeedbackForm adalah kuesioner dinamis (Feedback Client). Slug menjadi link
// publik /form/:slug yang dibagikan ke client tanpa login.
type FeedbackForm struct {
	ID           string             `json:"id"           gorm:"primaryKey;default:gen_random_uuid()"`
	Title        string             `json:"title"        gorm:"not null"`
	Description  *string            `json:"description"`
	Slug         string             `json:"slug"         gorm:"not null;uniqueIndex"`
	Status       FeedbackFormStatus `json:"status"       gorm:"not null;default:'draft'"`
	Language     FormLanguage       `json:"language"     gorm:"not null;default:'id'"`
	CollectEmail bool               `json:"collect_email" gorm:"not null;default:true"`
	Questions    []FeedbackQuestion `json:"questions"    gorm:"serializer:json;type:jsonb"`
	ProjectID    *string            `json:"project_id"`
	// AIJob: job saran AI async yang sedang berjalan/menunggu klarifikasi/
	// menunggu dipilih, nil bila tidak ada. Lihat FeedbackAIJob.
	AIJob *FeedbackAIJob `json:"ai_job,omitempty" gorm:"serializer:json;type:jsonb"`
	// CreatedBy menyimpan pembuat form (untuk kolom "Dibuat oleh" di daftar).
	// CreatedByName di-denormalisasi saat create agar daftar tak perlu join.
	CreatedBy     *string   `json:"created_by"`
	CreatedByName *string   `json:"created_by_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// SubmissionCount di-hitung saat List (bukan kolom) — nol pada Get tunggal.
	SubmissionCount int64 `json:"submission_count" gorm:"-"`
}

func (FeedbackForm) TableName() string { return "feedback_form" }

// FeedbackFormSubmission adalah satu pengisian form oleh client.
type FeedbackFormSubmission struct {
	ID                 string           `json:"id"               gorm:"primaryKey;default:gen_random_uuid()"`
	FormID             string           `json:"form_id"          gorm:"not null;index"`
	RespondentEmail    *string          `json:"respondent_email"`
	RespondentName     *string          `json:"respondent_name"`
	RespondentDivision *string          `json:"respondent_division"`
	Answers            []FeedbackAnswer `json:"answers"          gorm:"serializer:json;type:jsonb"`
	CreatedAt          time.Time        `json:"created_at"`
}

func (FeedbackFormSubmission) TableName() string { return "feedback_form_submission" }

// FeedbackFormRepository — persistensi form dinamis + submission.
type FeedbackFormRepository interface {
	Create(ctx context.Context, f *FeedbackForm) error
	Update(ctx context.Context, f *FeedbackForm) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*FeedbackForm, error)
	// GetBySlug mengembalikan (nil, nil) bila slug tidak ada — link publik
	// salah bukan error sistem.
	GetBySlug(ctx context.Context, slug string) (*FeedbackForm, error)
	// SlugExists dipakai saat men-generate slug unik.
	SlugExists(ctx context.Context, slug string) (bool, error)
	// List mengembalikan semua form terurut terbaru, dengan SubmissionCount terisi.
	List(ctx context.Context) ([]FeedbackForm, error)

	CreateSubmission(ctx context.Context, s *FeedbackFormSubmission) error
	ListSubmissions(ctx context.Context, formID string) ([]FeedbackFormSubmission, error)
	// ListAllSubmissions dipakai analitik lintas-form.
	ListAllSubmissions(ctx context.Context) ([]FeedbackFormSubmission, error)
}
