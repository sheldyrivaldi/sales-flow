import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'
import { AI_MUTATION_KEYS } from '../lib/aiMutation'
import type { AIMutationMeta } from '../lib/aiMutation'
import type { Prospect } from './prospects'

export type { Prospect }

// ── Types ─────────────────────────────────────────────────────────────────────

export type EventType = 'EXPO' | 'CONFERENCE' | 'SEMINAR' | 'WORKSHOP' | 'NETWORKING' | 'OTHER'
export type EventStatus = 'PLANNED' | 'ATTENDED' | 'CANCELLED'

export interface Event {
  id: string
  name: string
  type: EventType
  date: string | null
  location: string | null
  organizer: string | null
  notes: string | null
  status: EventStatus
  created_at: string
  updated_at: string
}

export interface EventListResponse {
  items: Event[]
  total: number
  page: number
  page_size: number
}

export interface EventFilters {
  type?: EventType
  status?: EventStatus
  search?: string
  page?: number
  page_size?: number
}

export interface EventCreateBody {
  name: string
  type: EventType
  date?: string
  location?: string
  organizer?: string
  notes?: string
  status?: EventStatus
}

export type EventUpdateBody = Partial<EventCreateBody>

// ── Helpers ───────────────────────────────────────────────────────────────────

export const EVENT_TYPE_LABELS: Record<EventType, string> = {
  EXPO: 'Expo',
  CONFERENCE: 'Conference',
  SEMINAR: 'Seminar',
  WORKSHOP: 'Workshop',
  NETWORKING: 'Networking',
  OTHER: 'Lainnya',
}

export const EVENT_STATUS_LABELS: Record<EventStatus, string> = {
  PLANNED: 'Direncanakan',
  ATTENDED: 'Dihadiri',
  CANCELLED: 'Dibatalkan',
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useEvents(filters: EventFilters = {}) {
  return useQuery({
    queryKey: ['events', filters],
    queryFn: () => apiFetch<EventListResponse>(`/api/events${buildQueryString({ ...filters })}`),
  })
}

export function useEvent(id?: string) {
  return useQuery({
    queryKey: ['event', id],
    queryFn: () => apiFetch<Event>(`/api/events/${id}`),
    enabled: !!id,
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

export function useCreateEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: EventCreateBody) =>
      apiFetch<Event>('/api/events', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
  })
}

export function useUpdateEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: EventUpdateBody }) =>
      apiFetch<Event>(`/api/events/${id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
      queryClient.invalidateQueries({ queryKey: ['event', id] })
    },
  })
}

export function useDeleteEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/events/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
  })
}

export function useConvertEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<Prospect>(`/api/events/${id}/convert`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['events'] })
      queryClient.invalidateQueries({ queryKey: ['prospects'] })
    },
  })
}

// ── Analisa Peserta Event (AI, on-demand) ─────────────────────────────────────

export type EventQuadrant = 'prioritas_utama' | 'perlu_digarap' | 'quick_win' | 'dipantau'

export interface EventCompanyInsight {
  name: string
  industry: string
  potential: 'tinggi' | 'rendah'
  interest: 'tinggi' | 'rendah'
  quadrant: EventQuadrant
  note: string
}

export interface EventAnalysis {
  companies: EventCompanyInsight[]
  summary: string
  timeline_suggestions: string[]
}

/** Analisa dokumen peserta event: PDF dikirim apa adanya (dibaca AI via
 * vision), Excel dikonversi ke CSV di browser dan dikirim sebagai teks. */
export function useAnalyzeEvent() {
  return useMutation({
    mutationKey: [...AI_MUTATION_KEYS.eventAnalysis],
    meta: {
      successToast: 'Analisa peserta event selesai.',
      errorToast: 'Analisa peserta gagal, coba lagi nanti.',
    } satisfies AIMutationMeta,
    mutationFn: ({ id, file, tableText }: { id: string; file?: File; tableText?: string }) => {
      const formData = new FormData()
      if (file) formData.append('file', file)
      if (tableText) formData.append('table_text', tableText)
      return apiFetch<EventAnalysis>(`/api/events/${id}/analyze`, {
        method: 'POST',
        body: formData,
      })
    },
  })
}
