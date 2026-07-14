import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'
import type { RecommendedAction } from '../lib/score'

// ── Types ─────────────────────────────────────────────────────────────────────

export type TenderStatus = 'IDENTIFIED' | 'QUALIFYING' | 'BIDDING' | 'SUBMITTED' | 'WON' | 'LOST'
export type TenderApiAction = 'PURSUE' | 'REVIEW' | 'WATCHLIST' | 'REJECT' | 'NEED_PARTNER'
export type TenderOrigin = 'manual' | 'discovery'

export interface Tender {
  id: string
  title: string
  buyer_name: string | null
  buyer_country: string | null
  buyer_industry: string | null
  value_estimate: number | null
  currency: string
  published_date: string | null
  submission_deadline: string | null
  source_name: string | null
  source_url: string | null
  service_category: string | null
  scope_summary: string | null
  eligibility_requirements: string | null
  technical_requirements: string | null
  status: TenderStatus
  fit_score: number | null
  recommended_action: TenderApiAction | null
  risk_flags: unknown
  reasoning_summary: string | null
  dedup_key: string | null
  origin: TenderOrigin
  created_at: string
  updated_at: string
}

export interface TenderListResponse {
  items: Tender[]
  total: number
  page: number
  page_size: number
}

export interface TenderFilters {
  status?: TenderStatus
  buyer?: string
  recommended_action?: TenderApiAction
  origin?: TenderOrigin
  deadline_from?: string
  deadline_to?: string
  search?: string
  page?: number
  page_size?: number
}

export interface TenderCreateBody {
  title: string
  buyer_name?: string
  buyer_country?: string
  buyer_industry?: string
  value_estimate?: number
  currency?: string
  published_date?: string
  submission_deadline?: string
  source_name?: string
  source_url?: string
  service_category?: string
  scope_summary?: string
  eligibility_requirements?: string
  technical_requirements?: string
  status?: TenderStatus
}

export type TenderUpdateBody = Partial<TenderCreateBody>

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Peta UPPERCASE API → Title-case FE (Design §2.1 / score.ts). */
export function actionToLabel(a: TenderApiAction): RecommendedAction {
  const map: Record<TenderApiAction, RecommendedAction> = {
    PURSUE: 'Pursue',
    REVIEW: 'Review',
    WATCHLIST: 'Watchlist',
    REJECT: 'Reject',
    NEED_PARTNER: 'Need Partner',
  }
  return map[a]
}

/** Kebalikan: Title-case FE → UPPERCASE API. */
export function labelToAction(l: RecommendedAction): TenderApiAction {
  const map: Record<RecommendedAction, TenderApiAction> = {
    Pursue: 'PURSUE',
    Review: 'REVIEW',
    Watchlist: 'WATCHLIST',
    Reject: 'REJECT',
    'Need Partner': 'NEED_PARTNER',
  }
  return map[l]
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useTenders(filters: TenderFilters = {}) {
  return useQuery({
    queryKey: ['tenders', filters],
    queryFn: () => apiFetch<TenderListResponse>(`/api/tenders${buildQueryString({ ...filters })}`),
  })
}

export function useTender(id?: string) {
  return useQuery({
    queryKey: ['tender', id],
    queryFn: () => apiFetch<Tender>(`/api/tenders/${id}`),
    enabled: !!id,
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

export function useCreateTender() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: TenderCreateBody) =>
      apiFetch<Tender>('/api/tenders', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenders'] })
    },
  })
}

export function useUpdateTender() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: TenderUpdateBody }) =>
      apiFetch<Tender>(`/api/tenders/${id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['tenders'] })
      queryClient.invalidateQueries({ queryKey: ['tender', id] })
    },
  })
}

export function useDeleteTender() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/tenders/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenders'] })
    },
  })
}

export function useUpdateTenderStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: TenderStatus }) =>
      apiFetch<Tender>(`/api/tenders/${id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      }),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['tenders'] })
      queryClient.invalidateQueries({ queryKey: ['tender', id] })
    },
  })
}

export function useRecordOutcome() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, result, notes }: { id: string; result: 'WON' | 'LOST'; notes?: string }) =>
      apiFetch<Tender>(`/api/tenders/${id}/outcome`, {
        method: 'POST',
        body: JSON.stringify({ result, notes }),
      }),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['tenders'] })
      queryClient.invalidateQueries({ queryKey: ['tender', id] })
    },
  })
}

export function usePromoteTender() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<Tender>(`/api/tenders/${id}/promote`, { method: 'POST' }),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['tenders'] })
      queryClient.invalidateQueries({ queryKey: ['tender', id] })
      queryClient.invalidateQueries({ queryKey: ['discovery-inbox'] })
    },
  })
}

/** Discovery Inbox "Watchlist" (reason omitted) / "Tolak" (reason filled) —
 * marks a discovery-origin tender as reviewed without promoting it. */
export function useReviewTender() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      apiFetch<Tender>(`/api/tenders/${id}/review`, {
        method: 'POST',
        body: JSON.stringify({ reason }),
      }),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['tenders'] })
      queryClient.invalidateQueries({ queryKey: ['tender', id] })
      queryClient.invalidateQueries({ queryKey: ['discovery-inbox'] })
    },
  })
}
