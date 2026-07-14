import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────
// Mirror internal/http/dto/report.go — keep in sync.

export type ReportType = 'daily_digest' | 'weekly_pipeline' | 'per_opportunity'

export const REPORT_TYPE_LABELS: Record<ReportType, string> = {
  daily_digest: 'Daily Opportunity Digest',
  weekly_pipeline: 'Weekly Pipeline Report',
  per_opportunity: 'Laporan Per-Peluang',
}

export interface Report {
  id: string
  report_type: ReportType
  title: string
  period_start: string
  period_end: string
  content: string
  model: string | null
  created_at: string
}

export interface ReportListResponse {
  items: Report[]
  total: number
  page: number
  page_size: number
}

export interface ReportListFilter {
  type?: ReportType
  page?: number
  page_size?: number
}

export interface ReportCreateBody {
  type: ReportType
  period_start: string
  period_end: string
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useReports(filter: ReportListFilter = {}) {
  return useQuery({
    queryKey: ['reports', filter],
    queryFn: () => apiFetch<ReportListResponse>(`/api/reports${buildQueryString({ ...filter })}`),
  })
}

export function useReport(id?: string) {
  return useQuery({
    queryKey: ['report', id],
    queryFn: () => apiFetch<Report>(`/api/reports/${id}`),
    enabled: !!id,
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

export function useGenerateReport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: ReportCreateBody) =>
      apiFetch<Report>('/api/reports', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] })
    },
  })
}

export function useDeleteReport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/api/reports/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] })
    },
  })
}
