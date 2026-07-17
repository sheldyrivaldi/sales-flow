import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface HermesStatus {
  status: 'connected' | 'disconnected'
  version?: string
  models?: string[]
  memory_active?: boolean
}

export interface HermesTestResult {
  status: 'ok' | 'failed'
  version?: string
}

// ── Query/Mutation Hooks ──────────────────────────────────────────────────────

export function useHermesStatus() {
  return useQuery({
    queryKey: ['settings', 'hermes'],
    queryFn: () => apiFetch<HermesStatus>('/api/settings/hermes'),
    staleTime: 30_000,
  })
}

export function useTestHermes() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => apiFetch<HermesTestResult>('/api/settings/hermes/test', { method: 'POST' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings', 'hermes'] }),
  })
}

/** Reset Hermes workspace memory (ADMIN only, BE-enforced — EP-16 TK-16.3.1
 * endpoint, wired from Settings UI here per EP-18 ST-18.3). */
export function useResetHermesMemory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => apiFetch<{ status: string }>('/api/admin/hermes/reset-memory', { method: 'POST' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings', 'hermes'] }),
  })
}

// ── AI Provider Config (EP-18 ST-18.4) ───────────────────────────────────────
// All three endpoints are ADMIN-only (BE-enforced via CapManageUsers).

export type AIProvider = 'openai' | 'openrouter'

/** Mirrors dto.AISettingResponse — api_key_masked is a display hint only
 * (e.g. "sk-...abcd"), never the real key. */
export interface AISetting {
  provider: AIProvider | ''
  model: string
  base_url: string | null
  api_key_masked: string
  enabled_toolsets: string[] | null
  is_active: boolean
  updated_at: string
}

/** Mirrors dto.AISettingUpdateRequest. api_key is write-only and optional:
 * omit it (or send undefined) to keep the currently stored key unchanged. */
export interface AISettingUpdateBody {
  provider: AIProvider
  model: string
  base_url?: string | null
  api_key?: string
  enabled_toolsets?: string[]
}

export interface AISettingTestResult {
  status: 'ok' | 'failed'
  version?: string
}

export function useAISetting() {
  return useQuery({
    queryKey: ['settings', 'ai'],
    queryFn: () => apiFetch<AISetting>('/api/settings/ai'),
    staleTime: 30_000,
  })
}

export function useUpdateAISetting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: AISettingUpdateBody) =>
      apiFetch<AISetting>('/api/settings/ai', { method: 'PUT', body: JSON.stringify(body) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings', 'ai'] }),
  })
}

export function useTestAISetting() {
  return useMutation({
    mutationFn: () => apiFetch<AISettingTestResult>('/api/settings/ai/test', { method: 'POST' }),
  })
}

// ── Hermes TUI (admin-only native Hermes CLI/TUI console) ──────────────────
// A ticket → session-cookie handoff, then a reverse proxy into ttyd — the
// frontend never renders a terminal itself; tui_url points at ttyd's own
// unmodified page, loaded in an iframe/new tab. See backend plan for the
// full connection lifecycle this is the browser side of.

export interface HermesTuiTicket {
  ticket: string
  expires_in: number
  tui_url: string
}

/** Mints a fresh, single-use, ~30s ticket — call this immediately before
 * navigating to tui_url (iframe src or window.open), never reuse the
 * result across two navigations. */
export function useIssueHermesTuiTicket() {
  return useMutation({
    mutationFn: () => apiFetch<HermesTuiTicket>('/api/admin/hermes/tui/ticket', { method: 'POST' }),
  })
}

export function useEndHermesTuiSession() {
  return useMutation({
    mutationFn: () => apiFetch<{ status: string }>('/api/admin/hermes/tui/end', { method: 'POST' }),
  })
}
