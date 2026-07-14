import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────
// Mirror internal/http/dto/playbook.go — keep in sync.

export type PlaybookTargetType = 'tender' | 'prospect'

export interface PlaybookContent {
  summary: string
  value_prop: string
  stakeholders: string[]
  strategy_checklist: string[]
  timeline: string[]
  risks: string[]
  next_actions: string[]
}

export interface Playbook {
  id: string
  target_type: PlaybookTargetType
  target_id: string
  version: number
  content: PlaybookContent
  model: string | null
  created_at: string
}

interface PlaybookListResponseDTO {
  items: Playbook[]
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function generateUrl(targetType: PlaybookTargetType, id: string) {
  return `/api/${targetType}s/${id}/playbook`
}

function listUrl(targetType: PlaybookTargetType, id: string) {
  return `/api/${targetType}s/${id}/playbooks`
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

/** All versions for a target, newest first (backend ORDER BY version DESC). */
export function usePlaybooks(targetType: PlaybookTargetType, id?: string) {
  return useQuery({
    queryKey: ['playbooks', targetType, id],
    queryFn: () =>
      apiFetch<PlaybookListResponseDTO>(listUrl(targetType, id as string)).then((r) => r.items),
    enabled: !!id,
  })
}

/** One playbook version by id — used for the "Bandingkan" (compare) view. */
export function usePlaybook(id?: string) {
  return useQuery({
    queryKey: ['playbook', id],
    queryFn: () => apiFetch<Playbook>(`/api/playbooks/${id}`),
    enabled: !!id,
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

/** Generates a new playbook version (latest+1) — prior versions are never touched. */
export function useGeneratePlaybook(targetType: PlaybookTargetType) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch<Playbook>(generateUrl(targetType, id), { method: 'POST' }),
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['playbooks', targetType, id] })
    },
  })
}
