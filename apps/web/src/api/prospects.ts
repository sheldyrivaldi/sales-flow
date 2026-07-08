import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'
import { toast } from '../lib/toast'

// ── Types ─────────────────────────────────────────────────────────────────────

export type ProspectStage = 'NEW' | 'QUALIFIED' | 'ENGAGED' | 'PROPOSAL' | 'WON' | 'LOST'
export type ProspectSource = 'manual' | 'event' | 'tender'

export interface Prospect {
  id: string
  name: string
  company: string | null
  contact_info: string | null
  source_type: ProspectSource
  source_id: string | null
  stage: ProspectStage
  est_value: number | null
  owner_user_id: string | null
  created_at: string
  updated_at: string
}

export interface ProspectListResponse {
  items: Prospect[]
  total: number
  page: number
  page_size: number
}

export interface ProspectFilters {
  stage?: ProspectStage
  owner_user_id?: string
  source_type?: ProspectSource
  search?: string
  page?: number
  page_size?: number
}

export interface ProspectCreateBody {
  name: string
  company?: string
  contact_info?: string
  source_type?: ProspectSource
  source_id?: string
  stage?: ProspectStage
  est_value?: number
  owner_user_id?: string
}

export type ProspectUpdateBody = Partial<ProspectCreateBody>

// ── Constants ─────────────────────────────────────────────────────────────────

export const PROSPECT_STAGES: ProspectStage[] = [
  'NEW',
  'QUALIFIED',
  'ENGAGED',
  'PROPOSAL',
  'WON',
  'LOST',
]

export const STAGE_LABELS: Record<ProspectStage, string> = {
  NEW: 'Baru',
  QUALIFIED: 'Qualified',
  ENGAGED: 'Engaged',
  PROPOSAL: 'Proposal',
  WON: 'Won',
  LOST: 'Lost',
}

export const SOURCE_LABELS: Record<ProspectSource, string> = {
  manual: 'Manual',
  event: 'Event',
  tender: 'Tender',
}

/** Stage terminal (WON/LOST) — dipakai untuk gating modal catatan opsional
 * sebelum mencatat outcome. Sumber tunggal, dipakai baik oleh Board maupun
 * Drawer agar aturan gating tidak bisa drift antar-file. */
export function isTerminalStage(stage: ProspectStage): stage is 'WON' | 'LOST' {
  return stage === 'WON' || stage === 'LOST'
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useProspects(filters: ProspectFilters = {}) {
  return useQuery({
    queryKey: ['prospects', filters],
    queryFn: () => apiFetch<ProspectListResponse>(`/api/prospects${buildQueryString({ ...filters })}`),
  })
}

export function useProspect(id?: string) {
  return useQuery({
    queryKey: ['prospect', id],
    queryFn: () => apiFetch<Prospect>(`/api/prospects/${id}`),
    enabled: !!id,
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

export function useCreateProspect() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: ProspectCreateBody) =>
      apiFetch<Prospect>('/api/prospects', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['prospects'] })
    },
  })
}

export function useUpdateProspect() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: ProspectUpdateBody }) =>
      apiFetch<Prospect>(`/api/prospects/${id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['prospects'] })
      queryClient.invalidateQueries({ queryKey: ['prospect', id] })
    },
  })
}

export function useDeleteProspect() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch(`/api/prospects/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['prospects'] })
    },
  })
}

/**
 * Ubah stage prospect (drag-drop kanban / aksi cepat drawer). Optimistic update
 * pada semua cache `['prospects', …]` yang sedang aktif, dengan rollback via
 * `onError` bila request gagal (Design §4.8: drag-drop optimistic + rollback).
 */
export function useUpdateProspectStage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, stage, notes }: { id: string; stage: ProspectStage; notes?: string }) =>
      apiFetch<Prospect>(`/api/prospects/${id}/stage`, {
        method: 'PATCH',
        body: JSON.stringify({ stage, notes }),
      }),
    onMutate: async ({ id, stage }) => {
      await queryClient.cancelQueries({ queryKey: ['prospects'] })

      const previous = queryClient.getQueriesData<ProspectListResponse>({ queryKey: ['prospects'] })

      previous.forEach(([key, data]) => {
        if (!data) return
        queryClient.setQueryData<ProspectListResponse>(key, {
          ...data,
          items: data.items.map((p) => (p.id === id ? { ...p, stage } : p)),
        })
      })

      const previousDetail = queryClient.getQueryData<Prospect>(['prospect', id])
      if (previousDetail) {
        queryClient.setQueryData<Prospect>(['prospect', id], { ...previousDetail, stage })
      }

      return { previous, previousDetail, id }
    },
    onError: (_err, _vars, context) => {
      context?.previous.forEach(([key, data]: [readonly unknown[], ProspectListResponse | undefined]) => {
        queryClient.setQueryData(key, data)
      })
      if (context?.previousDetail) {
        queryClient.setQueryData(['prospect', context.id], context.previousDetail)
      }
      toast.error('Gagal memindahkan stage prospek.')
      // Rollback restores the pre-mutation snapshot; refetch once more to
      // guard against drift from any other change that landed in between.
      queryClient.invalidateQueries({ queryKey: ['prospects'] })
      if (context?.id) queryClient.invalidateQueries({ queryKey: ['prospect', context.id] })
    },
    onSuccess: (data, { id }) => {
      // Server response is authoritative — patch caches directly instead of
      // an extra refetch (the optimistic onMutate write already got us here).
      queryClient.setQueryData<Prospect>(['prospect', id], data)
      queryClient.setQueriesData<ProspectListResponse>({ queryKey: ['prospects'] }, (old) => {
        if (!old) return old
        return { ...old, items: old.items.map((p) => (p.id === id ? data : p)) }
      })
    },
  })
}
