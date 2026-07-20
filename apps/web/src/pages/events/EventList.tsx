import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Plus, Calendar, Paperclip } from 'lucide-react'

import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import EmptyState from '../../components/ui/EmptyState'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import { toast } from '../../lib/toast'
import { formatTanggal } from '../../lib/format'
import { cn } from '../../lib/cn'

import {
  useEvents,
  useDeleteEvent,
  EVENT_TYPE_LABELS,
  EVENT_STATUS_LABELS,
} from '../../api/events'
import type { Event, EventFilters, EventType, EventStatus } from '../../api/events'
import EventFormModal from './EventFormModal'
import EventFilterBar from '../../components/events/EventFilterBar'
import EventAttachmentsModal from '../../components/events/EventAttachmentsModal'

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

export default function EventList() {
  const navigate = useNavigate()
  const [filters, setFilters] = useState<EventFilters>({ page: 1, page_size: 20 })
  const [page, setPage] = useState(1)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingEvent, setEditingEvent] = useState<Event | undefined>()
  const [deleteTarget, setDeleteTarget] = useState<Event | null>(null)
  const [attachmentTarget, setAttachmentTarget] = useState<Event | null>(null)

  const { data, isLoading } = useEvents({ ...filters, page })
  const deleteMutation = useDeleteEvent()

  const columns: Column<Event>[] = [
    {
      key: 'name',
      header: 'Nama',
      render: (row) => (
        <button
          type="button"
          className="text-left font-medium text-primary hover:underline max-w-xs truncate block"
          onClick={() => navigate(`/events/${row.id}`)}
        >
          {row.name}
        </button>
      ),
    },
    {
      key: 'type',
      header: 'Tipe',
      render: (row) => (
        <span className={cn('inline-flex items-center px-2 py-0.5 rounded-pill text-caption font-medium', TYPE_COLORS[row.type])}>
          {EVENT_TYPE_LABELS[row.type]}
        </span>
      ),
    },
    {
      key: 'date',
      header: 'Tanggal',
      render: (row) =>
        row.date ? (
          <span className="text-fg-muted">{formatTanggal(row.date)}</span>
        ) : (
          <span className="text-fg-subtle">—</span>
        ),
    },
    {
      key: 'location',
      header: 'Lokasi',
      render: (row) => <span className="text-fg-muted">{row.location ?? '—'}</span>,
    },
    {
      key: 'organizer',
      header: 'Organizer',
      render: (row) => <span className="text-fg-muted">{row.organizer ?? '—'}</span>,
    },
    {
      key: 'attachments',
      header: 'Lampiran',
      render: (row) => {
        const n = row.attachments?.length ?? 0
        if (n === 0) return <span className="text-fg-subtle">—</span>
        return (
          <button
            type="button"
            onClick={() => setAttachmentTarget(row)}
            className="inline-flex items-center gap-1.5 rounded-pill border border-line bg-surface-subtle px-2 py-0.5 text-caption text-fg hover:border-primary hover:text-primary transition-colors"
            title={`Lihat ${n} lampiran`}
          >
            <Paperclip className="w-3.5 h-3.5" aria-hidden="true" />
            {n}
          </button>
        )
      },
    },
    {
      key: 'status',
      header: 'Status',
      render: (row) => (
        <span className={cn('inline-flex items-center px-2 py-0.5 rounded-pill text-caption font-medium', STATUS_COLORS[row.status])}>
          {EVENT_STATUS_LABELS[row.status]}
        </span>
      ),
    },
  ]

  function openCreate() {
    setEditingEvent(undefined)
    setModalOpen(true)
  }

  function openEdit(ev: Event) {
    setEditingEvent(ev)
    setModalOpen(true)
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success('Event dihapus.')
    } catch {
      toast.error('Gagal menghapus event.')
    } finally {
      setDeleteTarget(null)
    }
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-h2 font-semibold text-fg">Events</h1>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
          Event Baru
        </Button>
      </div>

      {/* Filter multi-kolom bergaya Jira: pencarian lintas kolom, multi pilih
          per kolom, chip filter aktif, dan bersihkan semua. */}
      <EventFilterBar
        filters={filters}
        onChange={(next) => {
          setFilters(next)
          setPage(1)
        }}
        resultCount={data?.total}
      />

      {/* Table */}
      <div className="bg-surface border border-line rounded-card overflow-hidden">
        <Table<Event>
          columns={columns}
          data={data?.items ?? []}
          rowKey={(e) => e.id}
          loading={isLoading}
          page={page}
          total={data?.total}
          pageSize={filters.page_size ?? 20}
          onPageChange={setPage}
          empty={
            <EmptyState
              icon={<Calendar className="w-6 h-6" />}
              title="Belum ada event"
              description="Tambahkan event pameran, konferensi, atau networking yang dihadiri tim."
              action={
                <Button size="sm" onClick={openCreate}>
                  Event Baru
                </Button>
              }
            />
          }
          kebabActions={(row) => [
            {
              label: 'Lihat Detail',
              onClick: () => navigate(`/events/${row.id}`),
            },
            {
              label: 'Edit',
              onClick: () => openEdit(row),
            },
            {
              label: 'Lihat Lampiran',
              onClick: () => setAttachmentTarget(row),
            },
            {
              label: 'Hapus',
              onClick: () => setDeleteTarget(row),
              danger: true,
            },
          ]}
        />
      </div>

      {/* Form Drawer */}
      <EventFormModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        event={editingEvent}
        onSaved={() => setModalOpen(false)}
      />

      {/* Delete confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        title="Hapus Event?"
        description={`"${deleteTarget?.name}" akan dihapus permanen.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteMutation.isPending}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />

      {/* Daftar lampiran: buka di tab baru atau unduh */}
      <EventAttachmentsModal
        open={!!attachmentTarget}
        onClose={() => setAttachmentTarget(null)}
        eventName={attachmentTarget?.name ?? ''}
        attachments={attachmentTarget?.attachments ?? []}
      />
    </div>
  )
}
