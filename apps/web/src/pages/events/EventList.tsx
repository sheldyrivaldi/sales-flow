import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Plus, Calendar } from 'lucide-react'

import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import EmptyState from '../../components/ui/EmptyState'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import Select from '../../components/ui/Select'
import Input from '../../components/ui/Input'
import { toast } from '../../lib/toast'
import { formatTanggal } from '../../lib/format'
import { cn } from '../../lib/cn'

import {
  useEvents,
  useDeleteEvent,
  useConvertEvent,
  EVENT_TYPE_LABELS,
  EVENT_STATUS_LABELS,
} from '../../api/events'
import type { Event, EventFilters, EventType, EventStatus } from '../../api/events'
import EventFormDrawer from './EventFormDrawer'

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
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editingEvent, setEditingEvent] = useState<Event | undefined>()
  const [deleteTarget, setDeleteTarget] = useState<Event | null>(null)
  const [convertTarget, setConvertTarget] = useState<Event | null>(null)

  const { data, isLoading } = useEvents({ ...filters, page })
  const deleteMutation = useDeleteEvent()
  const convertMutation = useConvertEvent()

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
    setDrawerOpen(true)
  }

  function openEdit(ev: Event) {
    setEditingEvent(ev)
    setDrawerOpen(true)
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

  async function handleConvert() {
    if (!convertTarget) return
    try {
      await convertMutation.mutateAsync(convertTarget.id)
      toast.success('Event berhasil dikonversi ke prospek.')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : ''
      toast.error(msg.includes('sudah dikonversi') ? 'Event ini sudah pernah dikonversi ke prospek.' : 'Gagal mengonversi event.')
    } finally {
      setConvertTarget(null)
    }
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-h2 font-semibold text-fg">Events</h1>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
          + Event Baru
        </Button>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap gap-3">
        <Select
          className="w-44"
          value={filters.type ?? ''}
          onChange={(e) => {
            const v = e.target.value as EventType | ''
            setFilters((f) => ({ ...f, type: v || undefined }))
            setPage(1)
          }}
        >
          <option value="">Semua Tipe</option>
          <option value="EXPO">Expo</option>
          <option value="CONFERENCE">Conference</option>
          <option value="SEMINAR">Seminar</option>
          <option value="WORKSHOP">Workshop</option>
          <option value="NETWORKING">Networking</option>
          <option value="OTHER">Lainnya</option>
        </Select>

        <Select
          className="w-44"
          value={filters.status ?? ''}
          onChange={(e) => {
            const v = e.target.value as EventStatus | ''
            setFilters((f) => ({ ...f, status: v || undefined }))
            setPage(1)
          }}
        >
          <option value="">Semua Status</option>
          <option value="PLANNED">Direncanakan</option>
          <option value="ATTENDED">Dihadiri</option>
          <option value="CANCELLED">Dibatalkan</option>
        </Select>

        <Input
          className="w-52"
          placeholder="Cari nama/organizer…"
          value={filters.search ?? ''}
          onChange={(e) => {
            setFilters((f) => ({ ...f, search: e.target.value || undefined }))
            setPage(1)
          }}
        />
      </div>

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
                  + Event Baru
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
              label: '+ Konversi ke Prospek',
              onClick: () => setConvertTarget(row),
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
      <EventFormDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        event={editingEvent}
        onSaved={() => setDrawerOpen(false)}
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

      {/* Convert confirm */}
      <ConfirmDialog
        open={!!convertTarget}
        title="Konversi ke Prospek?"
        description={`Event "${convertTarget?.name}" akan dikonversi menjadi prospek baru.`}
        confirmLabel="Konversi"
        tone="primary"
        loading={convertMutation.isPending}
        onConfirm={handleConvert}
        onCancel={() => setConvertTarget(null)}
      />
    </div>
  )
}
