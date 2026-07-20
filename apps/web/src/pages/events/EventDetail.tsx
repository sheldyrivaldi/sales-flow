import { useState } from 'react'
import { useParams, useNavigate } from 'react-router'
import { ArrowLeft, Edit2, CalendarDays, MapPin, Building2, Users, Lock } from 'lucide-react'

import Button from '../../components/ui/Button'
import Skeleton from '../../components/ui/Skeleton'
import EmptyState from '../../components/ui/EmptyState'
import { formatTanggal } from '../../lib/format'
import { cn } from '../../lib/cn'

import { useEvent, EVENT_TYPE_LABELS, EVENT_STATUS_LABELS } from '../../api/events'
import type { EventType, EventStatus } from '../../api/events'
import EventFormModal from './EventFormModal'
import EventAnalysisPanel from '../../components/events/EventAnalysisPanel'
import EventAttachmentsSection from '../../components/events/EventAttachmentsSection'
import EventParticipantsSection from '../../components/events/EventParticipantsSection'
import EventPlaybookCard from '../../components/events/EventPlaybookCard'

const TYPE_COLORS: Record<EventType, string> = {
  EXPO: 'bg-sky-100 text-sky-700',
  CONFERENCE: 'bg-indigo-100 text-indigo-700',
  SEMINAR: 'bg-violet-100 text-violet-700',
  WORKSHOP: 'bg-amber-100 text-amber-700',
  NETWORKING: 'bg-emerald-100 text-emerald-700',
  OTHER: 'bg-neutral-100 text-neutral-600',
}

const STATUS_COLORS: Record<EventStatus, string> = {
  PLANNED: 'bg-sky-100 text-sky-700',
  ATTENDED: 'bg-emerald-100 text-emerald-700',
  CANCELLED: 'bg-rose-100 text-rose-700',
}

/** Satu fakta ringkas pada bar identitas event. */
function Fact({ icon: Icon, label, value }: { icon: typeof MapPin; label: string; value: string }) {
  return (
    <div className="flex items-start gap-2 min-w-0">
      <Icon className="w-4 h-4 text-fg-subtle mt-0.5 shrink-0" aria-hidden="true" />
      <div className="min-w-0">
        <p className="text-caption text-fg-subtle">{label}</p>
        <p className="text-body text-fg truncate" title={value}>{value}</p>
      </div>
    </div>
  )
}

/**
 * Halaman detail event.
 *
 * Urutan seksi mengikuti bobot nilainya, bukan urutan teknis: Analisa AI
 * diletakkan PALING ATAS karena di situlah nilai menu ini — ringkasan, apa
 * yang bisa diolah di internal, dan peluang klien baru. Identitas event cukup
 * jadi bar tipis; ia konteks, bukan tujuan.
 *
 * Tidak ada konversi ke prospek di sini: event adalah manajemen acara, dan
 * jalur prospek berjalan lewat menu Tender. Organisasi yang layak didekati
 * muncul sebagai peluang klien di hasil analisa beserta cara masuknya.
 */
export default function EventDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: event, isLoading } = useEvent(id)
  const [editOpen, setEditOpen] = useState(false)
  // Selama analisa berjalan, event dikunci dari SEMUA perubahan: edit,
  // lampiran, dan peserta. Kalau datanya bergeser di tengah proses, hasil
  // analisa tidak lagi cocok dengan event yang tersimpan.
  const locked = event?.analysis_status === 'running'

  if (isLoading) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!event) {
    return (
      <div className="p-6">
        <EmptyState
          icon={<CalendarDays className="w-6 h-6" />}
          title="Event tidak ditemukan"
          description="Event mungkin sudah dihapus."
          action={<Button size="sm" onClick={() => navigate('/events')}>Kembali ke Events</Button>}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-5 p-6 animate-page-enter">
      {/* Header */}
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <div className="min-w-0">
          <button
            type="button"
            onClick={() => navigate('/events')}
            className="inline-flex items-center gap-1 text-caption text-fg-muted hover:text-primary transition-colors mb-1"
          >
            <ArrowLeft className="w-3.5 h-3.5" aria-hidden="true" /> Events
          </button>
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="text-h2 font-semibold text-fg">{event.name}</h1>
            <span className={cn('inline-flex items-center px-2 py-0.5 rounded-pill text-caption font-medium', TYPE_COLORS[event.type])}>
              {EVENT_TYPE_LABELS[event.type]}
            </span>
            <span className={cn('inline-flex items-center px-2 py-0.5 rounded-pill text-caption font-medium', STATUS_COLORS[event.status])}>
              {EVENT_STATUS_LABELS[event.status]}
            </span>
          </div>
        </div>
        <Button
          variant="secondary"
          size="sm"
          disabled={locked}
          title={locked ? 'Terkunci selama analisa berjalan' : undefined}
          leftIcon={locked ? <Lock className="w-3.5 h-3.5" /> : <Edit2 className="w-3.5 h-3.5" />}
          onClick={() => setEditOpen(true)}
        >
          Edit Event
        </Button>
      </div>

      {/* Bar identitas — konteks ringkas, sengaja tidak mendominasi halaman. */}
      <div className="bg-surface border border-line rounded-card p-4 grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Fact icon={CalendarDays} label="Tanggal" value={event.date ? formatTanggal(event.date) : '—'} />
        <Fact icon={MapPin} label="Lokasi" value={event.location ?? '—'} />
        <Fact icon={Building2} label="Penyelenggara" value={event.organizer ?? '—'} />
        <Fact icon={Users} label="Peserta diundang" value={`${event.participant_emails?.length ?? 0} orang`} />
      </div>

      {/* NILAI UTAMA — diletakkan paling atas dengan sengaja. */}
      <EventAnalysisPanel
        eventId={event.id}
        analysis={event.analysis}
        analyzedAt={event.analyzed_at}
        status={event.analysis_status}
        error={event.analysis_error}
        attachmentCount={event.attachments?.length ?? 0}
      />

      {/* Playbook event — hasilnya tampil di sini juga, bukan cuma di menu Playbooks. */}
      <EventPlaybookCard eventId={event.id} eventName={event.name} />

      {/* Lampiran: lihat, tambah, hapus langsung dari sini. */}
      <EventAttachmentsSection eventId={event.id} attachments={event.attachments ?? []} locked={locked} />

      {/* Peserta: kelola + kirim undangan lewat aplikasi email. */}
      <EventParticipantsSection event={event} locked={locked} />

      {event.notes && (
        <div className="bg-surface border border-line rounded-card p-4">
          <p className="text-caption font-medium text-fg-muted mb-1">Catatan</p>
          <p className="text-body text-fg whitespace-pre-wrap">{event.notes}</p>
        </div>
      )}

      <EventFormModal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        event={event}
        onSaved={() => setEditOpen(false)}
      />
    </div>
  )
}
