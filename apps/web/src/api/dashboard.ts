import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'
import type { Tender } from './tenders'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface PipelineStage {
  stage: string
  count: number
  total_value: number
}

export interface DashboardSummary {
  pipeline: PipelineStage[]
  total_pipeline_count: number
  total_pipeline_value: number
  priority_tenders: Tender[]
  discovery_today_count: number
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useDashboardSummary() {
  return useQuery({
    queryKey: ['dashboard-summary'],
    queryFn: () => apiFetch<DashboardSummary>('/api/dashboard/summary'),
  })
}
