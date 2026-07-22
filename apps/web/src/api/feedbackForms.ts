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

export type FeedbackFormStatus = 'draft' | 'published' | 'closed'

export type FormLanguage = 'id' | 'en'

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

export interface SuggestResult {
  questions: SuggestedQuestion[]
  degraded: boolean
}

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

// AI: minta saran pertanyaan (multipart: prompt + bahasa + banyak lampiran opsional).
export function useSuggestQuestions() {
  return useMutation({
    mutationFn: async ({
      prompt,
      files,
      language,
    }: {
      prompt: string
      files?: File[]
      language: FormLanguage
    }) => {
      const fd = new FormData()
      fd.append('prompt', prompt)
      fd.append('language', language)
      for (const f of files ?? []) fd.append('files', f)
      return apiFetch<SuggestResult>('/api/feedback-forms/ai/suggest', { method: 'POST', body: fd })
    },
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
