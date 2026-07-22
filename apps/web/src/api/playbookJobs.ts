import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'
import type { PlaybookContent } from './playbooks'

export type PlaybookJobStatus = 'in_progress' | 'updating' | 'success' | 'failed'

export interface PlaybookRevision {
  instruction: string
  attachment_name?: string
  attachment_url?: string
  at: string
}

export interface PlaybookJob {
  id: string
  title: string
  /** true bila judul diketik user — hasil generate tidak menimpanya. */
  user_titled?: boolean
  prompt: string
  status: PlaybookJobStatus
  content?: PlaybookContent
  error_message?: string
  attachment_name?: string
  attachment_url?: string
  revisions?: PlaybookRevision[]
  source: 'custom' | 'event'
  /** Terisi bila playbook tertaut ke sebuah event (Source==='event'). SATU
   * event hanya punya SATU playbook tertaut; generate ulang memindah tautan. */
  event_id?: string
  created_at: string
  updated_at: string
}

export const PLAYBOOK_STATUS_LABEL: Record<PlaybookJobStatus, string> = {
  in_progress: 'Diproses',
  updating: 'Merevisi',
  success: 'Selesai',
  failed: 'Gagal',
}

/** true selama job masih berjalan — dipakai untuk memutuskan polling. */
export function isJobActive(s: PlaybookJobStatus): boolean {
  return s === 'in_progress' || s === 'updating'
}

export function usePlaybookJobs() {
  return useQuery({
    queryKey: ['playbook-jobs'],
    queryFn: () => apiFetch<{ items: PlaybookJob[] }>('/api/playbook-jobs').then((r) => r.items),
    // Poll selama ada job aktif supaya status berubah otomatis tanpa refresh.
    refetchInterval: (query) => {
      const items = query.state.data as PlaybookJob[] | undefined
      return items?.some((j) => isJobActive(j.status)) ? 3000 : false
    },
  })
}

export function useCreatePlaybookJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ title, prompt, file }: { title?: string; prompt: string; file?: File | null }) => {
      const fd = new FormData()
      if (title?.trim()) fd.append('title', title.trim())
      fd.append('prompt', prompt)
      if (file) fd.append('file', file)
      return apiFetch<PlaybookJob>('/api/playbook-jobs', { method: 'POST', body: fd })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['playbook-jobs'] }),
  })
}

export function useRefinePlaybookJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, instruction, file }: { id: string; instruction: string; file?: File | null }) => {
      const fd = new FormData()
      fd.append('instruction', instruction)
      if (file) fd.append('file', file)
      return apiFetch<PlaybookJob>(`/api/playbook-jobs/${id}/refine`, { method: 'POST', body: fd })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['playbook-jobs'] }),
  })
}

export function useRetryPlaybookJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch<PlaybookJob>(`/api/playbook-jobs/${id}/retry`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['playbook-jobs'] }),
  })
}

export function useDeletePlaybookJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/api/playbook-jobs/${id}`, { method: 'DELETE' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['playbook-jobs'] }),
  })
}

/** Generate (atau generate ulang) playbook tertaut ke sebuah event. Modalnya
 * identik dengan menu Playbooks: title & prompt opsional + lampiran tambahan
 * opsional. SELURUH lampiran event otomatis ikut sebagai konteks (server-side),
 * dan generate ulang melepas playbook lama lalu menautkan yang baru. */
export function useCreateEventPlaybookJob() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      eventId,
      title,
      prompt,
      file,
    }: {
      eventId: string
      title?: string
      prompt?: string
      file?: File | null
    }) => {
      const fd = new FormData()
      if (title?.trim()) fd.append('title', title.trim())
      if (prompt?.trim()) fd.append('prompt', prompt.trim())
      if (file) fd.append('file', file)
      return apiFetch<PlaybookJob>(`/api/events/${eventId}/playbook-job`, { method: 'POST', body: fd })
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['playbook-jobs'] }),
  })
}
