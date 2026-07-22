import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────

export type QuestionType = 'rating' | 'text' | 'choice' | 'nps'

export interface FeedbackQuestion {
  id: string
  type: QuestionType
  label: string
  description?: string
  required: boolean
  scale?: number
  options?: string[]
  multiple?: boolean
  /** Keterangan ujung skala rating (kiri = nilai terendah, kanan = tertinggi). */
  min_label?: string
  max_label?: string
}

// processing_ai/need_clarification: saran AI async sedang berjalan/menunggu
// jawaban klarifikasi — lihat FeedbackAIJob. Form kembali ke draft begitu
// selesai (sukses maupun gagal).
export type FeedbackFormStatus = 'draft' | 'published' | 'closed' | 'processing_ai' | 'need_clarification'

export type FormLanguage = 'id' | 'en'

/** Satu pasang tanya-jawab klarifikasi (AI bertanya ke USER, bukan responden). */
export interface ClarifyQA {
  question: string
  answer: string
}

/**
 * Job saran AI async yang SEMENTARA secara bisnis (dibuang begitu user
 * menambahkan pilihannya ke form, atau membatalkan) tapi PERSISTEN secara
 * teknis — generate bisa makan waktu lama, jadi progres tidak boleh hilang
 * hanya karena user pindah halaman. Dipoll lewat useFeedbackForm selama
 * status form processing_ai.
 */
export interface FeedbackAIJob {
  prompt: string
  language: FormLanguage
  round: number
  qa_history: ClarifyQA[]
  /** Konfigurasi tipe & jumlah pertanyaan dari user (opsional). Nilai per key:
   * "random" atau angka dalam bentuk string. */
  type_counts?: Partial<Record<QuestionType, string>>
  confidence: number
  clarifying_questions: string[]
  pending_questions: SuggestedQuestion[]
  error: string
  updated_at: string
}

export interface FeedbackForm {
  id: string
  title: string
  description: string | null
  slug: string
  status: FeedbackFormStatus
  language: FormLanguage
  collect_email: boolean
  questions: FeedbackQuestion[]
  project_id: string | null
  created_by: string | null
  created_by_name: string | null
  created_at: string
  updated_at: string
  submission_count: number
  ai_job?: FeedbackAIJob | null
}

export interface FeedbackAnswer {
  question_id: string
  text?: string
  rating?: number
  choice?: string[]
}

export interface FeedbackSubmission {
  id: string
  form_id: string
  respondent_email: string | null
  respondent_name: string | null
  respondent_division: string | null
  answers: FeedbackAnswer[]
  created_at: string
}

export interface QuestionStat {
  question_id: string
  label: string
  type: QuestionType
  responses: number
  average?: number
  distribution?: number[]
  choices?: Record<string, number>
  texts?: string[]
}

export interface FormAnalytics {
  form_id?: string
  total_forms: number
  total_submissions: number
  avg_rating: number
  nps: number
  questions: QuestionStat[]
}

export interface SuggestedQuestion {
  type: QuestionType
  label: string
  description?: string
  scale?: number
  options?: string[]
  multiple?: boolean
  min_label?: string
  max_label?: string
}

/** Konfigurasi jumlah pertanyaan per tipe yang dikirim ke ai/suggest — nilai
 * 'random' berarti AI bebas menentukan jumlahnya (minimal 1), 0 berarti tipe
 * itu dikecualikan, N berarti persis N pertanyaan. Key yang tidak disertakan
 * berarti bebas ditentukan AI. */
export type TypeCountsInput = Partial<Record<QuestionType, number | 'random'>>

export interface RefineResult {
  question: SuggestedQuestion
  degraded: boolean
}

export interface FeedbackInsight {
  summary: string
  strengths: string[]
  weaknesses: string[]
  improvements: string[]
  themes: string[]
  degraded: boolean
}

export interface FeedbackFormPublic {
  title: string
  description: string | null
  slug: string
  language: FormLanguage
  collect_email: boolean
  questions: FeedbackQuestion[]
}

export interface FeedbackFormUpsert {
  title: string
  description?: string | null
  slug?: string
  language?: FormLanguage
  collect_email?: boolean
  questions: FeedbackQuestion[]
  project_id?: string | null
}

// Public link ke halaman form (tanpa login).
export function publicFormLink(slug: string): string {
  return `${window.location.origin}/form/${slug}`
}

// ── Admin hooks (authd) ───────────────────────────────────────────────────────

export function useFeedbackForms() {
  return useQuery({
    queryKey: ['feedback-forms'],
    queryFn: () => apiFetch<{ items: FeedbackForm[] }>('/api/feedback-forms').then((r) => r.items),
  })
}

export function useFeedbackForm(id: string | undefined) {
  return useQuery({
    queryKey: ['feedback-form', id],
    queryFn: () => apiFetch<FeedbackForm>(`/api/feedback-forms/${id}`),
    enabled: !!id,
    // Poll selagi job saran AI async berjalan di background, supaya progres
    // (dan hasilnya) tampil tanpa perlu reload manual — lihat FeedbackAIJob.
    refetchInterval: (query) => {
      const data = query.state.data as FeedbackForm | undefined
      return data?.status === 'processing_ai' ? 3000 : false
    },
  })
}

export function useFeedbackSubmissions(id: string | undefined) {
  return useQuery({
    queryKey: ['feedback-form-submissions', id],
    queryFn: () =>
      apiFetch<{ items: FeedbackSubmission[] }>(`/api/feedback-forms/${id}/submissions`).then((r) => r.items),
    enabled: !!id,
  })
}

export function useFeedbackFormAnalytics(id: string | undefined) {
  return useQuery({
    queryKey: ['feedback-form-analytics', id],
    queryFn: () => apiFetch<FormAnalytics>(`/api/feedback-forms/${id}/analytics`),
    enabled: !!id,
  })
}

export function useFeedbackGlobalAnalytics() {
  return useQuery({
    queryKey: ['feedback-forms-analytics'],
    queryFn: () => apiFetch<FormAnalytics>('/api/feedback-forms/analytics'),
  })
}

export function useCreateFeedbackForm() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: FeedbackFormUpsert) =>
      apiFetch<FeedbackForm>('/api/feedback-forms', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['feedback-forms'] }),
  })
}

export function useUpdateFeedbackForm() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: FeedbackFormUpsert }) =>
      apiFetch<FeedbackForm>(`/api/feedback-forms/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    onSuccess: (_data, { id }) => {
      void qc.invalidateQueries({ queryKey: ['feedback-forms'] })
      void qc.invalidateQueries({ queryKey: ['feedback-form', id] })
    },
  })
}

export function usePublishFeedbackForm() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<FeedbackForm>(`/api/feedback-forms/${id}/publish`, { method: 'POST' }),
    onSuccess: (_data, id) => {
      void qc.invalidateQueries({ queryKey: ['feedback-forms'] })
      void qc.invalidateQueries({ queryKey: ['feedback-form', id] })
    },
  })
}

export function useDeleteFeedbackForm() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/api/feedback-forms/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['feedback-forms'] })
      void qc.invalidateQueries({ queryKey: ['feedback-forms-analytics'] })
    },
  })
}

// AI: minta saran pertanyaan (multipart: prompt + bahasa + tipe/jumlah opsional
// + banyak lampiran opsional). ASINKRON: form disimpan seketika dengan status
// processing_ai/need_clarification dan dikembalikan langsung — hasil
// sebenarnya datang lewat polling useFeedbackForm (lihat ai_job pada form).
// form_id kosong berarti form belum tersimpan — dibuatkan draft baru di
// server, dan id-nya HARUS diadopsi oleh caller (mis. builder mengganti URL)
// supaya penyimpanan berikutnya mengedit baris yang sama, bukan membuat baru.
export function useSuggestQuestions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({
      formId,
      prompt,
      files,
      language,
      typeCounts,
    }: {
      formId?: string
      prompt: string
      files?: File[]
      language: FormLanguage
      typeCounts?: TypeCountsInput
    }) => {
      const fd = new FormData()
      if (formId) fd.append('form_id', formId)
      fd.append('prompt', prompt)
      fd.append('language', language)
      if (typeCounts && Object.keys(typeCounts).length > 0) fd.append('type_counts', JSON.stringify(typeCounts))
      for (const f of files ?? []) fd.append('files', f)
      return apiFetch<FeedbackForm>('/api/feedback-forms/ai/suggest', { method: 'POST', body: fd })
    },
    onSuccess: (form) => {
      qc.setQueryData(['feedback-form', form.id], form)
      void qc.invalidateQueries({ queryKey: ['feedback-forms'] })
    },
  })
}

// AI: jawab pertanyaan klarifikasi — memicu putaran generate berikutnya.
export function useSubmitClarifyAnswers() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ formId, answers }: { formId: string; answers: string[] }) =>
      apiFetch<FeedbackForm>(`/api/feedback-forms/${formId}/ai/suggest/clarify`, {
        method: 'POST',
        body: JSON.stringify({ answers }),
      }),
    onSuccess: (form) => qc.setQueryData(['feedback-form', form.id], form),
  })
}

// AI: bersihkan job saran yang sementara — dipanggil setelah pertanyaan
// terpilih ditambahkan ke form, atau saat membatalkan alur klarifikasi.
export function useClearAIJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (formId: string) =>
      apiFetch<FeedbackForm>(`/api/feedback-forms/${formId}/ai/suggest/clear`, { method: 'POST' }),
    onSuccess: (form) => qc.setQueryData(['feedback-form', form.id], form),
  })
}

// AI: revisi satu pertanyaan.
export function useRefineQuestion() {
  return useMutation({
    mutationFn: (body: { question: SuggestedQuestion; instruction: string; language: FormLanguage }) =>
      apiFetch<RefineResult>('/api/feedback-forms/ai/refine', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
  })
}

// AI: analisa feedback (opsional per form).
export function useAnalyzeFeedback() {
  return useMutation({
    mutationFn: (formId?: string) =>
      apiFetch<FeedbackInsight>(`/api/feedback-forms/ai/analyze${formId ? `?form_id=${formId}` : ''}`, {
        method: 'POST',
      }),
  })
}

// ── Public hooks (halaman /form/:slug, tanpa login) ──────────────────────────

export function usePublicForm(slug: string | undefined) {
  return useQuery({
    queryKey: ['public-form', slug],
    queryFn: () => apiFetch<FeedbackFormPublic>(`/api/public/forms/${slug}`),
    enabled: !!slug,
    retry: false,
  })
}

export function useSubmitPublicForm(slug: string | undefined) {
  return useMutation({
    mutationFn: (body: {
      respondent_email?: string
      respondent_name?: string
      respondent_division?: string
      answers: FeedbackAnswer[]
    }) =>
      apiFetch<{ status: string }>(`/api/public/forms/${slug}`, {
        method: 'POST',
        body: JSON.stringify(body),
      }),
  })
}
