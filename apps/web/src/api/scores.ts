import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'
import type { TenderApiAction } from './tenders'

// ── Types ─────────────────────────────────────────────────────────────────────

export type ScoreTargetType = 'tender' | 'prospect'

export interface EvidenceItem {
  dimension: string
  verdict: 'pass' | 'warn' | 'fail'
  note: string
}

export interface ScoreResponse {
  id: string
  target_type: ScoreTargetType
  target_id: string
  fit_score: number
  recommended_action: TenderApiAction
  confidence: number | null
  reasoning: string | null
  evidence: EvidenceItem[] | null
  risk_flags: string[] | null
  model: string | null
  created_at: string
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function scoreUrl(targetType: ScoreTargetType, id: string) {
  return `/api/${targetType}s/${id}/score`
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

/** Latest score for a target, or null if it has never been analyzed yet. */
export function useScore(targetType: ScoreTargetType, id?: string) {
  return useQuery({
    queryKey: ['score', targetType, id],
    queryFn: () => apiFetch<ScoreResponse | null>(scoreUrl(targetType, id as string)),
    enabled: !!id,
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

/** Runs (or re-runs) AI analysis for a target and persists a new score row. */
export function useRunScore(targetType: ScoreTargetType) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<ScoreResponse>(scoreUrl(targetType, id), { method: 'POST' }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['score', targetType, id] })
      if (targetType === 'tender') {
        queryClient.invalidateQueries({ queryKey: ['tender', id] })
        queryClient.invalidateQueries({ queryKey: ['tenders'] })
      }
    },
  })
}
