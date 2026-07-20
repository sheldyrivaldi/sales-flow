import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'
import { AI_MUTATION_KEYS } from '../lib/aiMutation'
import type { AIMutationMeta } from '../lib/aiMutation'


// ── Types ─────────────────────────────────────────────────────────────────────

export type EventType = 'EXPO' | 'CONFERENCE' | 'SEMINAR' | 'WORKSHOP' | 'NETWORKING' | 'OTHER'
export type EventStatus = 'PLANNED' | 'ATTENDED' | 'CANCELLED'

/** Berkas pendukung event (rundown, undangan, denah booth). */
export interface EventAttachment {
  name: string
  url: string
  mime?: string
  size?: number
}

export interface Event {
  id: string
  name: string
  type: EventType
  date: string | null
  location: string | null
  organizer: string | null
  notes: string | null
  status: EventStatus
  /** Undangan lepas — peserta tidak perlu punya akun di aplikasi ini. */
  participant_emails: string[]
  attachments: EventAttachment[]
  /** Hasil Analisa AI tersimpan — bertahan antar sesi. */
  analysis?: EventAnalysis
  analyzed_at?: string
  analysis_status: AnalysisStatus
  analysis_error?: string
  created_at: string
  updated_at: string
}

export interface EventListResponse {
  items: Event[]
  total: number
  page: number
  page_size: number
}

/** Filter multi-kolom: satu kolom boleh banyak nilai (OR), antar kolom AND. */
export interface EventFilters {
  type?: EventType[]
  status?: EventStatus[]
  /** Menyapu nama, penyelenggara, lokasi, dan catatan sekaligus. */
  search?: string
  /** Format YYYY-MM-DD; batas akhir inklusif sampai akhir hari. */
  date_from?: string
  date_to?: string
  location?: string
  organizer?: string
  has_attachment?: boolean
  has_participant?: boolean
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
  participant_emails?: string[]
  attachments?: EventAttachment[]
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
    // Analisa berjalan di background dan dilaporkan Hermes lewat callback,
    // jadi halaman harus menjemput sendiri perubahannya.
    refetchInterval: (query) =>
      (query.state.data as Event | undefined)?.analysis_status === 'running' ? 5000 : false,
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

/** Unggah satu lampiran event. Berkas diunggah LEBIH DULU lalu URL-nya ikut
 * saat event disimpan — supaya lampiran juga bisa dipasang pada event yang
 * belum dibuat (form create), dan bisa dibatalkan sebelum menyimpan. */
export function useUploadEventAttachment() {
  return useMutation({
    mutationFn: (file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return apiFetch<EventAttachment>('/api/events/attachments', { method: 'POST', body: fd })
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


// ── Analisa Peserta Event (AI, on-demand) ─────────────────────────────────────

/** Bagian analisis. Judul DAN isi ditentukan AI mengikuti materi event;
 * `body` berisi markdown (bullet, penomoran, tebal, miring, tabel). */
export interface AnalysisSection {
  title: string
  body: string
}

/** Hasil Analisa AI. Seluruh field teks berisi markdown. */
export interface EventAnalysis {
  summary: string
  sections: AnalysisSection[]
  /** Yang bisa diolah untuk perusahaan sendiri (markdown). */
  internal_opportunities: string
  /** Peluang klien baru beserta cara masuknya (markdown). */
  client_opportunities: string
  /** Yang belum bisa disimpulkan — penangkal jawaban asal tambal. */
  data_gaps: string[]
}

/** Status Analisa AI. Selama 'running', event dikunci dari perubahan. */
export type AnalysisStatus = 'idle' | 'running' | 'success' | 'failed'

/** Jalankan Analisa AI. Tanpa payload — seluruh bahan (identitas event,
 * catatan, undangan, dan SEMUA lampiran) diambil server dari event itu
 * sendiri, sehingga tidak ada dokumen yang ikut dianalisa tapi tidak
 * tersimpan pada event-nya. */
export function useAnalyzeEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationKey: [...AI_MUTATION_KEYS.eventAnalysis],
    meta: {
      successToast: 'Analisa AI selesai.',
      errorToast: 'Analisa AI gagal, coba lagi nanti.',
    } satisfies AIMutationMeta,
    mutationFn: (id: string) =>
      apiFetch<Event>(`/api/events/${id}/analyze`, { method: 'POST' }),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['event', id] })
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
  })
}
