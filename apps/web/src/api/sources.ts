import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────
// Mirror internal/http/dto/source.go — keep in sync.

export type SourceAccess = 'publik' | 'login' | 'manual'

export interface Source {
  id: string
  name: string
  url: string
  country: string | null
  access: SourceAccess
  legal_note: string | null
  enabled: boolean
  priority: number
  preset_key: string | null
  created_at: string
  updated_at: string
}

export interface SourceListResponse {
  items: Source[]
  total: number
  page: number
  page_size: number
}

export interface SourcePreset {
  key: string
  name: string
  url: string
  country: string
  access: SourceAccess
  legal_note: string
  activated: boolean
}

export interface SourceFilters {
  enabled?: boolean
  access?: SourceAccess
  search?: string
  page?: number
  page_size?: number
}

export interface SourceCreateBody {
  name: string
  url: string
  country?: string
  access?: SourceAccess
  legal_note?: string
  enabled?: boolean
  priority?: number
}

export type SourceUpdateBody = Partial<SourceCreateBody>

export const ACCESS_LABELS: Record<SourceAccess, string> = {
  publik: 'Publik',
  login: 'Login',
  manual: 'Manual',
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useSources(filters: SourceFilters = {}) {
  return useQuery({
    queryKey: ['sources', filters],
    queryFn: () => apiFetch<SourceListResponse>(`/api/sources${buildQueryString({ ...filters })}`),
  })
}

export function useSourcePresets() {
  return useQuery({
    queryKey: ['sourcePresets'],
    queryFn: () => apiFetch<SourcePreset[]>('/api/sources/presets'),
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

export function useCreateSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: SourceCreateBody) =>
      apiFetch<Source>('/api/sources', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sources'] })
    },
  })
}

export function useUpdateSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: SourceUpdateBody }) =>
      apiFetch<Source>(`/api/sources/${id}`, {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sources'] })
    },
  })
}

export function useDeleteSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch(`/api/sources/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sources'] })
    },
  })
}

export function useActivatePreset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (key: string) =>
      apiFetch<Source>('/api/sources/presets', {
        method: 'POST',
        body: JSON.stringify({ key }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sources'] })
      queryClient.invalidateQueries({ queryKey: ['sourcePresets'] })
    },
  })
}
