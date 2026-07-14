import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'
import type { TenderApiAction, TenderListResponse } from './tenders'

// ── Types ─────────────────────────────────────────────────────────────────────

export type DiscoveryRunStatus = 'pending' | 'running' | 'success' | 'failed'

export interface DiscoveryRun {
  id: string
  started_at: string
  finished_at: string | null
  source_ids: string[]
  status: DiscoveryRunStatus
  found_count: number
  summary: string | null
  correlation_key: string | null
  created_at: string
  updated_at: string
}

export interface DiscoveryRunListResponse {
  items: DiscoveryRun[]
  total: number
  page: number
  page_size: number
}

export interface DiscoveryInboxFilters {
  recommended_action?: TenderApiAction
  min_score?: number
  page?: number
  page_size?: number
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useDiscoveryInbox(filters: DiscoveryInboxFilters = {}) {
  return useQuery({
    queryKey: ['discovery-inbox', filters],
    queryFn: () => apiFetch<TenderListResponse>(`/api/discovery/inbox${buildQueryString({ ...filters })}`),
  })
}

/** Recent runs, newest first — used for header status + crawl-in-progress detection. */
export function useDiscoveryRuns(options: { refetchInterval?: number | false } = {}) {
  return useQuery({
    queryKey: ['discovery-runs'],
    queryFn: () => apiFetch<DiscoveryRunListResponse>('/api/discovery/runs?page_size=5'),
    refetchInterval: options.refetchInterval,
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

export function useRunDiscovery() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiFetch<DiscoveryRun>('/api/discovery/run', { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['discovery-runs'] })
      queryClient.invalidateQueries({ queryKey: ['discovery-inbox'] })
    },
  })
}
