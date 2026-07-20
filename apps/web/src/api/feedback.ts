import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface FeedbackResponse {
  id: string
  request_id: string
  overall_rating: number
  quality_rating: number | null
  communication_rating: number | null
  timeliness_rating: number | null
  nps: number | null
  comment: string | null
  respondent_name: string | null
  created_at: string
}

export interface FeedbackRequest {
  id: string
  token: string
  project_name: string
  client_name: string | null
  project_id: string | null
  created_at: string
  response?: FeedbackResponse | null
}

export interface FeedbackAnalytics {
  total_requests: number
  total_responses: number
  avg_overall: number
  avg_quality: number
  avg_communication: number
  avg_timeliness: number
  nps: number
  rating_distribution: number[]
  comments:
    | {
        project_name: string
        client_name: string
        rating: number
        comment: string
        created_at: string
      }[]
    | null
}

export interface FeedbackPublicInfo {
  project_name: string
  client_name: string | null
  submitted: boolean
}

export interface FeedbackSubmitBody {
  overall_rating: number
  quality_rating?: number
  communication_rating?: number
  timeliness_rating?: number
  nps?: number
  comment?: string
  respondent_name?: string
}

// ── Admin hooks (authd) ───────────────────────────────────────────────────────

export function useFeedbackRequests() {
  return useQuery({
    queryKey: ['feedback-requests'],
    queryFn: () => apiFetch<{ items: FeedbackRequest[] }>('/api/feedback').then((r) => r.items),
  })
}

export function useFeedbackAnalytics() {
  return useQuery({
    queryKey: ['feedback-analytics'],
    queryFn: () => apiFetch<FeedbackAnalytics>('/api/feedback/analytics'),
  })
}

export function useCreateFeedbackRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { project_name: string; client_name?: string; project_id?: string }) =>
      apiFetch<FeedbackRequest>('/api/feedback', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['feedback-requests'] }),
  })
}

export function useDeleteFeedbackRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/api/feedback/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['feedback-requests'] })
      void qc.invalidateQueries({ queryKey: ['feedback-analytics'] })
    },
  })
}

// ── Public hooks (halaman /f/:token, tanpa login) ────────────────────────────

export function usePublicFeedbackInfo(token: string | undefined) {
  return useQuery({
    queryKey: ['public-feedback', token],
    queryFn: () => apiFetch<FeedbackPublicInfo>(`/api/public/feedback/${token}`),
    enabled: !!token,
    retry: false,
  })
}

export function useSubmitPublicFeedback(token: string | undefined) {
  return useMutation({
    mutationFn: (body: FeedbackSubmitBody) =>
      apiFetch<{ status: string }>(`/api/public/feedback/${token}`, {
        method: 'POST',
        body: JSON.stringify(body),
      }),
  })
}
