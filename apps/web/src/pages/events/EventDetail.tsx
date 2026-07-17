import { useState } from 'react'
import { useParams, useNavigate } from 'react-router'
import { ArrowLeft, Edit2, RefreshCw } from 'lucide-react'

import Button from '../../components/ui/Button'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import Skeleton from '../../components/ui/Skeleton'
import { toast } from '../../lib/toast'
import { formatTanggal } from '../../lib/format'
import { cn } from '../../lib/cn'

import { useEvent, useConvertEvent, EVENT_TYPE_LABELS, EVENT_STATUS_LABELS } from '../../api/events'
import type { EventType, EventStatus } from '../../api/events'
import EventFormDrawer from './EventFormDrawer'
import EventAnalysisPanel from '../../components/events/EventAnalysisPanel'
import PlaybookPanel from '../../components/PlaybookPanel'

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

export default function EventDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: event, isLoading } = useEvent(id)
  const convertMutation = useConvertEvent()

  const [editOpen, setEditOpen] = useState(false)
  const [confirmConvert, setConfirmConvert] = useState(false)

  async function handleConvert() {
    if (!id) return
    try {
      await convertMutation.mutateAsync(id)
      toast.success('Event berhasil dikonversi ke prospek.')
      setConfirmConvert(false)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : ''
      toast.error(msg.includes('sudah dikonversi') ? 'Event ini sudah pernah dikonversi ke prospek.' : 'Gagal mengonversi event.')
      setConfirmConvert(false)
    }
  }

  if (isLoading) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  }

  if (!event) {
    return (
      <div className="p-6">
        <p className="text-fg-muted">Event tidak ditemukan.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 p-6 max-w-2xl">
      {/* Breadcrumb / back */}
      <button
        type="button"
        onClick={() => navigate('/events')}
        className="inline-flex items-center gap-1.5 text-caption text-fg-muted hover:text-fg transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        Kembali ke Events
      </button>

      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-2">
          <h1 className="text-h2 font-semibold text-fg">{event.name}</h1>
          <div className="flex items-center gap-2 flex-wrap">
            <span className={cn('inline-flex items-center px-2 py-0.5 rounded-pill text-caption font-medium', TYPE_COLORS[event.type])}>
              {EVENT_TYPE_LABELS[event.type]}
            </span>
            <span className={cn('inline-flex items-center px-2 py-0.5 rounded-pill text-caption font-medium', STATUS_COLORS[event.status])}>
              {EVENT_STATUS_LABELS[event.status]}
            </span>
          </div>
        </div>

        <div className="flex gap-2 flex-shrink-0">
          <Button
            variant="secondary"
            size="sm"
            leftIcon={<Edit2 className="w-4 h-4" />}
            onClick={() => setEditOpen(true)}
          >
            Edit
          </Button>
          <Button
            size="sm"
            leftIcon={<RefreshCw className="w-4 h-4" />}
            onClick={() => setConfirmConvert(true)}
          >
            + Konversi ke Prospek
          </Button>
        </div>
      </div>

      {/* Detail card */}
      <div className="bg-surface border border-line rounded-card p-5 flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-x-8 gap-y-3">
          <InfoRow label="Tanggal" value={event.date ? formatTanggal(event.date) : '—'} />
          <InfoRow label="Lokasi" value={event.location ?? '—'} />
          <InfoRow label="Organizer" value={event.organizer ?? '—'} />
          <InfoRow label="Ditambahkan" value={formatTanggal(event.created_at)} />
        </div>

        {event.notes && (
          <div>
            <p className="text-caption font-medium text-fg-muted mb-1">Catatan</p>
            <p className="text-body text-fg whitespace-pre-wrap">{event.notes}</p>
          </div>
        )}
      </div>

      {/* Analisa peserta pasca-event: upload daftar peserta → kuadran + timeline */}
      <EventAnalysisPanel eventId={event.id} />

      {/* Playbook khusus event ini — bisa digenerate, dari dokumen, direvisi
          via prompt, dan diekspor ke PPT */}
      <PlaybookPanel targetType="event" targetId={event.id} />

      {/* Prospek dari event (placeholder — EP-07 akan mengisi list prospek tertaut) */}
      <div className="bg-surface border border-line rounded-card p-5">
        <h2 className="text-h3 font-semibold text-fg mb-1">Kontak / Prospek dari Event</h2>
        <p className="text-caption text-fg-muted">
          Daftar prospek yang bersumber dari event ini akan tampil di sini setelah EP-07 selesai.
        </p>
      </div>

      {/* Edit drawer */}
      <EventFormDrawer
        open={editOpen}
        onClose={() => setEditOpen(false)}
        event={event}
        onSaved={() => setEditOpen(false)}
      />

      {/* Convert confirm */}
      <ConfirmDialog
        open={confirmConvert}
        title="Konversi ke Prospek?"
        description={`Event "${event.name}" akan dikonversi menjadi prospek baru dengan source dari event ini.`}
        confirmLabel="Konversi"
        tone="primary"
        loading={convertMutation.isPending}
        onConfirm={handleConvert}
        onCancel={() => setConfirmConvert(false)}
      />
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-caption font-medium text-fg-muted">{label}</p>
      <p className="text-body text-fg">{value}</p>
    </div>
  )
}
