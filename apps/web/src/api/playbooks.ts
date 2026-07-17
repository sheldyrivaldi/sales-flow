import { useQuery, useMutation } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'
import { AI_MUTATION_KEYS } from '../lib/aiMutation'
import type { AIMutationMeta } from '../lib/aiMutation'

// ── Types ─────────────────────────────────────────────────────────────────────
// Mirror internal/http/dto/playbook.go — keep in sync.

export type PlaybookTargetType = 'tender' | 'prospect' | 'event'

export interface PlaybookTimelineItem {
  activity: string
  start_day: number
  duration_days: number
}

export interface PlaybookContent {
  summary: string
  value_prop: string
  stakeholders: string[]
  strategy_checklist: string[]
  timeline: string[]
  risks: string[]
  next_actions: string[]
  /** Rencana kerja terstruktur untuk render Gantt — playbook lama mungkin
   * tidak memilikinya (fallback ke `timeline`). */
  timeline_plan?: PlaybookTimelineItem[]
}

export interface Playbook {
  id: string
  target_type: PlaybookTargetType | 'custom'
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

/** Meta invalidasi bersama: refresh daftar versi target terkait + daftar
 * custom — dieksekusi MutationCache global walau komponen sudah unmount. */
function playbookMeta(successToast: string): AIMutationMeta {
  return {
    successToast,
    errorToast: 'Aksi playbook gagal, coba lagi nanti.',
    invalidate: (variables) => {
      const keys: unknown[][] = [['playbooks-custom']]
      const v = variables as Record<string, unknown> | string
      const id = typeof v === 'string' ? v : (v?.targetId ?? v?.id)
      for (const t of ['tender', 'prospect', 'event']) keys.push(['playbooks', t, id])
      return keys
    },
  }
}

/** Generates a new playbook version (latest+1) — prior versions are never touched. */
export function useGeneratePlaybook(targetType: PlaybookTargetType) {
  return useMutation({
    mutationKey: [...AI_MUTATION_KEYS.playbook],
    meta: playbookMeta('Playbook berhasil dibuat.'),
    mutationFn: (id: string) => apiFetch<Playbook>(generateUrl(targetType, id), { method: 'POST' }),
  })
}

/** Generates a new playbook version from an uploaded source document (PDF). */
export function useGeneratePlaybookFromDocument(targetType: PlaybookTargetType) {
  return useMutation({
    mutationKey: [...AI_MUTATION_KEYS.playbook],
    meta: playbookMeta('Playbook berhasil dibuat dari dokumen.'),
    mutationFn: ({ id, file }: { id: string; file: File }) => {
      const formData = new FormData()
      formData.append('file', file)
      return apiFetch<Playbook>(`${generateUrl(targetType, id)}/from-document`, {
        method: 'POST',
        body: formData,
      })
    },
  })
}

/** Refines an existing playbook with a free-form instruction — hasil
 * dipersist sebagai versi baru pada target yang sama. */
export function useRefinePlaybook() {
  return useMutation({
    mutationKey: [...AI_MUTATION_KEYS.playbook],
    meta: playbookMeta('Playbook direvisi sebagai versi baru.'),
    mutationFn: ({ playbookId, instruction }: { playbookId: string; instruction: string; targetId: string }) =>
      apiFetch<Playbook>(`/api/playbooks/${playbookId}/refine`, {
        method: 'POST',
        body: JSON.stringify({ instruction }),
      }),
  })
}

// ── Custom playbooks (menu Playbooks, mandiri tanpa target) ──────────────────

/** Versi terbaru tiap playbook custom, terbaru dulu. */
export function useCustomPlaybooks() {
  return useQuery({
    queryKey: ['playbooks-custom'],
    queryFn: () =>
      apiFetch<PlaybookListResponseDTO>('/api/playbooks/custom').then((r) => r.items),
  })
}

/** Playbook mandiri dari topik bebas user. */
export function useGenerateCustomPlaybook() {
  return useMutation({
    mutationKey: [...AI_MUTATION_KEYS.playbook],
    meta: playbookMeta('Playbook custom berhasil dibuat.'),
    mutationFn: ({ topic }: { topic: string }) =>
      apiFetch<Playbook>('/api/playbooks/custom', {
        method: 'POST',
        body: JSON.stringify({ topic }),
      }),
  })
}
