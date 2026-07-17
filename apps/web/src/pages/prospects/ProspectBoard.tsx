import { useMemo, useState } from 'react'
import { FileSpreadsheet, LayoutGrid, List as ListIcon, Plus, Search, X } from 'lucide-react'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useDroppable,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core'

import { StagePill } from '../../components/ui/Badge'
import Badge from '../../components/ui/Badge'
import EmptyState from '../../components/ui/EmptyState'
import Skeleton from '../../components/ui/Skeleton'
import Modal from '../../components/ui/Modal'
import Field from '../../components/ui/Field'
import Button from '../../components/ui/Button'
import Select from '../../components/ui/Select'
import Input from '../../components/ui/Input'
import Avatar from '../../components/ui/Avatar'
import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import ProspectCard, { ProspectCardView } from '../../components/prospects/ProspectCard'
import OutcomeNotesModal from '../../components/prospects/OutcomeNotesModal'
import ProspectFormDrawer from './ProspectFormDrawer'
import ProspectDrawer from './ProspectDrawer'
import { formatRupiahShort } from '../../lib/format'
import { exportPipelineExcel } from '../../lib/exportPipelineExcel'
import { toast } from '../../lib/toast'
import { cn } from '../../lib/cn'

import {
  useProspects,
  useUpdateProspectStage,
  useDeleteProspect,
  PROSPECT_STAGES,
  SOURCE_LABELS,
  isTerminalStage,
} from '../../api/prospects'
import type { Prospect, ProspectStage, ProspectSource } from '../../api/prospects'
import { useUsers } from '../../api/users'

// Melebihi pagination.MaxSize backend (500) supaya board tetap memuat seluruh
// pipeline dalam satu request; ditandingkan dengan `data.total` di bawah agar
// pemotongan data tidak lagi diam-diam (lihat banner peringatan pada render).
const BOARD_PAGE_SIZE = 500

interface StageColumnProps {
  stage: ProspectStage
  cards: Prospect[]
  ownerNames: Record<string, string>
  onCardClick: (id: string) => void
}

/** Kolom board ala Jira: panel abu lembut setinggi board, header tetap di
 * atas (stage + jumlah + total nilai), daftar kartu scroll sendiri per kolom,
 * dan seluruh kolom menyala saat kartu diseret di atasnya. */
function StageColumn({ stage, cards, ownerNames, onCardClick }: StageColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id: stage })
  const totalValue = cards.reduce((sum, p) => sum + (p.est_value ?? 0), 0)

  return (
    <div
      ref={setNodeRef}
      className={cn(
        'w-72 flex-shrink-0 flex flex-col rounded-card border bg-surface-subtle/70 transition-all duration-150 min-h-0',
        isOver ? 'border-primary bg-primary-subtle ring-2 ring-primary/25' : 'border-line'
      )}
    >
      <div className="flex items-center justify-between gap-2 px-3 py-2.5 border-b border-line/70 shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <StagePill stage={stage} />
          <span className="inline-flex items-center justify-center min-w-5 h-5 px-1.5 rounded-pill bg-surface text-caption font-semibold text-fg-muted tabular-nums">
            {cards.length}
          </span>
        </div>
        <span className="text-caption font-semibold text-fg-muted tabular-nums truncate">
          {formatRupiahShort(totalValue)}
        </span>
      </div>

      <div className="flex-1 min-h-24 overflow-y-auto scrollbar-thin p-2 flex flex-col gap-2">
        {cards.map((p) => (
          <ProspectCard
            key={p.id}
            prospect={p}
            ownerName={p.owner_user_id ? ownerNames[p.owner_user_id] : undefined}
            onClick={() => onCardClick(p.id)}
          />
        ))}
        {cards.length === 0 && (
          <div className="flex items-center justify-center rounded-btn border border-dashed border-line-strong/60 py-6 text-caption text-fg-subtle select-none">
            Seret kartu ke sini
          </div>
        )}
      </div>
    </div>
  )
}

export default function ProspectBoard() {
  const [view, setView] = useState<'board' | 'table'>('board')
  const [sourceFilter, setSourceFilter] = useState<ProspectSource | ''>('')
  const [ownerFilter, setOwnerFilter] = useState('')
  const [searchText, setSearchText] = useState('')
  const [stageFilter, setStageFilter] = useState<ProspectStage[]>([])
  const [minValue, setMinValue] = useState('')

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [editingProspect, setEditingProspect] = useState<Prospect | undefined>()
  const [deleteTarget, setDeleteTarget] = useState<Prospect | null>(null)

  // Kartu yang sedang diseret — dirender sebagai clone di <DragOverlay>
  // (behavior board ala Jira: kartu asal meredup, clone mengikuti kursor).
  const [activeProspect, setActiveProspect] = useState<Prospect | null>(null)

  // Drag-to-WON/LOST & kebab "Ubah Stage" berbagi modal catatan opsional yang sama.
  const [pendingMove, setPendingMove] = useState<{ id: string; stage: ProspectStage } | null>(null)
  const [outcomeNotes, setOutcomeNotes] = useState('')
  const [stageChangeRow, setStageChangeRow] = useState<Prospect | null>(null)
  const [stageChangeValue, setStageChangeValue] = useState<ProspectStage>('NEW')

  const { data, isLoading } = useProspects({
    page_size: BOARD_PAGE_SIZE,
    source_type: sourceFilter || undefined,
  })
  const { data: usersData } = useUsers()
  const updateStageMutation = useUpdateProspectStage()
  const deleteMutation = useDeleteProspect()

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } })
  )

  const ownerNames = useMemo(() => {
    const map: Record<string, string> = {}
    usersData?.items.forEach((u) => {
      map[u.id] = u.name
    })
    return map
  }, [usersData])

  const allItems = useMemo(() => data?.items ?? [], [data])
  const isTruncated = !!data && data.total > data.items.length
  const ownerOptions = useMemo(
    () => Array.from(new Set(allItems.map((p) => p.owner_user_id).filter((v): v is string => !!v))),
    [allItems]
  )
  // Filter gabungan ala Jira: semua kriteria di-AND-kan; tiap filter berdiri
  // sendiri dan bisa dikombinasikan bebas.
  const minValueNum = parseFloat(minValue)
  const searchLower = searchText.trim().toLowerCase()
  const items = allItems.filter((p) => {
    if (ownerFilter && p.owner_user_id !== ownerFilter) return false
    if (stageFilter.length > 0 && !stageFilter.includes(p.stage)) return false
    if (!isNaN(minValueNum) && minValue !== '' && (p.est_value ?? 0) < minValueNum) return false
    if (searchLower) {
      const hay = `${p.name} ${p.company ?? ''} ${p.contact_info ?? ''}`.toLowerCase()
      if (!hay.includes(searchLower)) return false
    }
    return true
  })

  const activeFilterCount =
    (searchLower ? 1 : 0) + (sourceFilter ? 1 : 0) + (ownerFilter ? 1 : 0) +
    (stageFilter.length > 0 ? 1 : 0) + (minValue !== '' && !isNaN(minValueNum) ? 1 : 0)

  function resetFilters() {
    setSearchText('')
    setSourceFilter('')
    setOwnerFilter('')
    setStageFilter([])
    setMinValue('')
  }

  function toggleStage(stage: ProspectStage) {
    setStageFilter((prev) =>
      prev.includes(stage) ? prev.filter((x) => x !== stage) : [...prev, stage],
    )
  }

  const byStage: Record<ProspectStage, Prospect[]> = {
    NEW: [],
    QUALIFIED: [],
    ENGAGED: [],
    PROPOSAL: [],
    WON: [],
    LOST: [],
  }
  for (const p of items) {
    byStage[p.stage]?.push(p)
  }

  function openCreate() {
    setEditingProspect(undefined)
    setFormOpen(true)
  }

  function openEdit(p: Prospect) {
    setEditingProspect(p)
    setFormOpen(true)
  }

  function openStageChange(p: Prospect) {
    setStageChangeValue(p.stage)
    setStageChangeRow(p)
  }

  async function confirmStageChange() {
    if (!stageChangeRow) return
    if (stageChangeValue === stageChangeRow.stage) {
      setStageChangeRow(null)
      return
    }
    if (isTerminalStage(stageChangeValue)) {
      setOutcomeNotes('')
      setPendingMove({ id: stageChangeRow.id, stage: stageChangeValue })
      setStageChangeRow(null)
      return
    }
    try {
      await updateStageMutation.mutateAsync({ id: stageChangeRow.id, stage: stageChangeValue })
      toast.success(`Stage diubah ke ${stageChangeValue}.`)
    } catch {
      // onError hook sudah menampilkan toast.error
    } finally {
      setStageChangeRow(null)
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success('Prospek dihapus.')
    } catch {
      toast.error('Gagal menghapus prospek.')
    } finally {
      setDeleteTarget(null)
    }
  }

  function handleDragStart(event: DragStartEvent) {
    setActiveProspect(items.find((p) => p.id === String(event.active.id)) ?? null)
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveProspect(null)
    const { active, over } = event
    if (!over) return

    const prospectId = String(active.id)
    const targetStage = over.id as ProspectStage
    const prospect = items.find((p) => p.id === prospectId)
    if (!prospect || prospect.stage === targetStage) return

    if (isTerminalStage(targetStage)) {
      setOutcomeNotes('')
      setPendingMove({ id: prospectId, stage: targetStage })
      return
    }

    updateStageMutation.mutate({ id: prospectId, stage: targetStage })
  }

  function handleExportExcel() {
    void exportPipelineExcel(items, ownerNames)
      .then(() => toast.success('Pipeline berhasil diekspor ke Excel.'))
      .catch(() => toast.error('Gagal mengekspor pipeline.'))
  }

  async function confirmPendingMove() {
    if (!pendingMove) return
    try {
      await updateStageMutation.mutateAsync({
        id: pendingMove.id,
        stage: pendingMove.stage,
        notes: outcomeNotes || undefined,
      })
      toast.success(`Prospek dipindahkan ke ${pendingMove.stage}.`)
    } catch {
      // onError hook sudah menampilkan toast.error
    } finally {
      setPendingMove(null)
      setOutcomeNotes('')
    }
  }

  const tableColumns: Column<Prospect>[] = [
    {
      key: 'name',
      header: 'Nama',
      render: (row) => (
        <button
          type="button"
          className="text-left font-medium text-primary hover:underline max-w-xs truncate block"
          onClick={() => setSelectedId(row.id)}
        >
          {row.name}
        </button>
      ),
    },
    {
      key: 'company',
      header: 'Perusahaan',
      render: (row) => <span className="text-fg-muted">{row.company ?? '—'}</span>,
    },
    {
      key: 'stage',
      header: 'Stage',
      render: (row) => <StagePill stage={row.stage} />,
    },
    {
      key: 'est_value',
      header: 'Nilai',
      align: 'right',
      render: (row) =>
        row.est_value != null ? (
          <span className="tabular-nums">{formatRupiahShort(row.est_value)}</span>
        ) : (
          <span className="text-fg-subtle">—</span>
        ),
    },
    {
      key: 'owner_user_id',
      header: 'Owner',
      align: 'center',
      render: (row) =>
        row.owner_user_id ? (
          <Avatar name={ownerNames[row.owner_user_id] ?? row.owner_user_id} size="sm" />
        ) : (
          <span className="text-fg-subtle">—</span>
        ),
    },
    {
      key: 'source_type',
      header: 'Sumber',
      render: (row) => (
        <Badge tone="info" appearance="soft">
          {SOURCE_LABELS[row.source_type]}
        </Badge>
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-4 h-full min-h-0">
      <div className="flex items-center justify-between shrink-0">
        <h1 className="text-h2 font-semibold text-fg">Pipeline</h1>
        <div className="flex items-center gap-2">
          {/* Toggle Board / Table */}
          <div className="inline-flex rounded-btn border border-line overflow-hidden">
            <button
              type="button"
              onClick={() => setView('board')}
              className={cn(
                'px-3 py-1.5 text-caption font-medium inline-flex items-center gap-1.5 transition-colors',
                view === 'board' ? 'bg-primary text-white' : 'bg-surface text-fg-muted hover:text-fg'
              )}
            >
              <LayoutGrid className="w-3.5 h-3.5" /> Board
            </button>
            <button
              type="button"
              onClick={() => setView('table')}
              className={cn(
                'px-3 py-1.5 text-caption font-medium inline-flex items-center gap-1.5 transition-colors',
                view === 'table' ? 'bg-primary text-white' : 'bg-surface text-fg-muted hover:text-fg'
              )}
            >
              <ListIcon className="w-3.5 h-3.5" /> Table
            </button>
          </div>
          <Button
            variant="secondary"
            leftIcon={<FileSpreadsheet className="w-4 h-4" />}
            onClick={handleExportExcel}
            disabled={items.length === 0}
          >
            Export Excel
          </Button>
          <Button leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
            Prospek Baru
          </Button>
        </div>
      </div>

      {isTruncated && data && (
        <p className="text-caption text-warning bg-warning/10 rounded-card px-3 py-2">
          Menampilkan {data.items.length} dari {data.total} prospek. Persempit dengan filter untuk melihat sisanya.
        </p>
      )}

      {/* Filter bar ala Jira: search + multi-stage + sumber + owner + min nilai,
          semua bisa dikombinasikan; chip stage toggle + tombol reset. */}
      <div className="flex flex-col gap-2 shrink-0">
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-fg-subtle pointer-events-none" aria-hidden="true" />
            <Input
              className="w-56 pl-8"
              placeholder="Cari nama, perusahaan, kontak…"
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              aria-label="Cari prospek"
            />
          </div>

          <Select
            className="w-40"
            value={sourceFilter}
            onChange={(e) => setSourceFilter(e.target.value as ProspectSource | '')}
          >
            <option value="">Semua Sumber</option>
            {(Object.keys(SOURCE_LABELS) as ProspectSource[]).map((s) => (
              <option key={s} value={s}>
                {SOURCE_LABELS[s]}
              </option>
            ))}
          </Select>

          <Select className="w-44" value={ownerFilter} onChange={(e) => setOwnerFilter(e.target.value)}>
            <option value="">Semua Owner</option>
            {ownerOptions.map((o) => (
              <option key={o} value={o}>
                {ownerNames[o] ?? o}
              </option>
            ))}
          </Select>

          <Input
            className="w-40"
            type="number"
            min="0"
            placeholder="Min nilai (Rp)"
            value={minValue}
            onChange={(e) => setMinValue(e.target.value)}
            aria-label="Nilai minimum"
          />

          {activeFilterCount > 0 && (
            <Button size="sm" variant="ghost" leftIcon={<X className="w-3.5 h-3.5" />} onClick={resetFilters}>
              Reset ({activeFilterCount})
            </Button>
          )}
        </div>

        {/* Chip stage multi-select */}
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-caption text-fg-muted mr-1">Stage:</span>
          {PROSPECT_STAGES.map((stage) => {
            const active = stageFilter.includes(stage)
            return (
              <button
                key={stage}
                type="button"
                aria-pressed={active}
                onClick={() => toggleStage(stage)}
                className={cn(
                  'px-2.5 py-1 rounded-pill text-caption font-medium border transition-colors duration-150',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1',
                  active
                    ? 'bg-primary-subtle text-primary-active border-primary-border'
                    : 'bg-surface text-fg-muted border-line hover:border-primary-border hover:text-fg',
                )}
              >
                {stage}
              </button>
            )
          })}
        </div>
      </div>

      {isLoading ? (
        <div className="flex gap-3 overflow-x-auto scrollbar-thin pb-2">
          {PROSPECT_STAGES.map((stage) => (
            <div key={stage} className="w-72 flex-shrink-0 flex flex-col gap-3">
              <Skeleton className="h-10" />
              <Skeleton className="h-24" />
              <Skeleton className="h-24" />
            </div>
          ))}
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          title="Belum ada prospek"
          description="Buat prospek baru atau konversi dari event/tender untuk mulai mengelola pipeline."
          action={
            <Button size="sm" onClick={openCreate}>
              Prospek Baru
            </Button>
          }
        />
      ) : view === 'board' ? (
        <DndContext
          sensors={sensors}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={() => setActiveProspect(null)}
        >
          {/* Board mengisi sisa tinggi halaman; kolom setinggi board dan
              scroll kartunya sendiri-sendiri — persis behavior papan Jira. */}
          <div className="flex-1 min-h-0 flex gap-3 overflow-x-auto scrollbar-thin pb-2 items-stretch">
            {PROSPECT_STAGES.map((stage) => (
              <StageColumn
                key={stage}
                stage={stage}
                cards={byStage[stage]}
                ownerNames={ownerNames}
                onCardClick={setSelectedId}
              />
            ))}
          </div>

          {/* Clone kartu yang mengikuti kursor saat drag — sedikit miring +
              bayangan tebal supaya terasa "terangkat" dari papan. */}
          <DragOverlay dropAnimation={{ duration: 180, easing: 'cubic-bezier(0.16, 1, 0.3, 1)' }}>
            {activeProspect ? (
              <ProspectCardView
                prospect={activeProspect}
                ownerName={activeProspect.owner_user_id ? ownerNames[activeProspect.owner_user_id] : undefined}
                className="rotate-2 shadow-lg cursor-grabbing ring-1 ring-primary-border"
              />
            ) : null}
          </DragOverlay>
        </DndContext>
      ) : (
        <div className="bg-surface border border-line rounded-card overflow-hidden">
          <Table<Prospect>
            columns={tableColumns}
            data={items}
            rowKey={(p) => p.id}
            kebabActions={(row) => [
              { label: 'Lihat Detail', onClick: () => setSelectedId(row.id) },
              { label: 'Edit', onClick: () => openEdit(row) },
              { label: 'Ubah Stage', onClick: () => openStageChange(row) },
              { label: 'Hapus', onClick: () => setDeleteTarget(row), danger: true },
            ]}
          />
        </div>
      )}

      {/* Create / Edit drawer */}
      <ProspectFormDrawer
        open={formOpen}
        onClose={() => setFormOpen(false)}
        prospect={editingProspect}
        onSaved={() => setFormOpen(false)}
      />

      {/* Detail drawer */}
      <ProspectDrawer
        open={!!selectedId}
        onClose={() => setSelectedId(null)}
        prospectId={selectedId ?? undefined}
      />

      {/* Delete confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        title="Hapus Prospek?"
        description={`"${deleteTarget?.name}" akan dihapus permanen.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteMutation.isPending}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />

      {/* Kebab "Ubah Stage" — pilih stage, notes muncul via modal WON/LOST di bawah bila perlu */}
      <Modal
        open={!!stageChangeRow}
        onClose={() => setStageChangeRow(null)}
        title="Ubah Stage"
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setStageChangeRow(null)}>
              Batal
            </Button>
            <Button loading={updateStageMutation.isPending} onClick={confirmStageChange}>
              Simpan
            </Button>
          </>
        }
      >
        <Field label="Stage baru" htmlFor="prospect-stage-select">
          <Select
            id="prospect-stage-select"
            value={stageChangeValue}
            onChange={(e) => setStageChangeValue(e.target.value as ProspectStage)}
          >
            {PROSPECT_STAGES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </Select>
        </Field>
      </Modal>

      {/* WON/LOST — modal catatan opsional (komponen bersama, lihat ProspectDrawer) */}
      <OutcomeNotesModal
        open={!!pendingMove}
        stage={pendingMove?.stage ?? null}
        notes={outcomeNotes}
        onNotesChange={setOutcomeNotes}
        loading={updateStageMutation.isPending}
        onConfirm={confirmPendingMove}
        onCancel={() => setPendingMove(null)}
      />
    </div>
  )
}


