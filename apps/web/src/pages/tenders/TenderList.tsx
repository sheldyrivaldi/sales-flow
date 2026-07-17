import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Plus, FileSearch, Sparkles } from 'lucide-react'

import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import { StagePill, ActionBadge } from '../../components/ui/Badge'
import ScoreRing from '../../components/ui/ScoreRing'
import EmptyState from '../../components/ui/EmptyState'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import Select from '../../components/ui/Select'
import Input from '../../components/ui/Input'
import { toast } from '../../lib/toast'
import { formatRupiahShort, formatTanggal } from '../../lib/format'
import { cn } from '../../lib/cn'

import {
  useTenders,
  useDeleteTender,
  usePromoteTender,
  actionToLabel,
} from '../../api/tenders'
import type { Tender, TenderFilters, TenderStatus, TenderApiAction, TenderOrigin } from '../../api/tenders'
import TenderFormDrawer from './TenderFormDrawer'
import ProposalDraftDrawer from '../../components/tenders/ProposalDraftDrawer'

function deadlineTone(deadline: string | null): 'normal' | 'warning' | 'danger' {
  if (!deadline) return 'normal'
  const diffMs = new Date(deadline).getTime() - Date.now()
  const diffDays = diffMs / (1000 * 60 * 60 * 24)
  if (diffMs < 0) return 'danger'
  if (diffDays <= 7) return 'warning'
  return 'normal'
}

export default function TenderList() {
  const navigate = useNavigate()
  const [filters, setFilters] = useState<TenderFilters>({ page: 1, page_size: 20 })
  const [page, setPage] = useState(1)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editingTender, setEditingTender] = useState<Tender | undefined>()
  const [deleteTarget, setDeleteTarget] = useState<Tender | null>(null)
  const [proposalTender, setProposalTender] = useState<Tender | null>(null)

  const { data, isLoading } = useTenders({ ...filters, page })
  const deleteMutation = useDeleteTender()
  const promoteMutation = usePromoteTender()

  const columns: Column<Tender>[] = [
    {
      key: 'title',
      header: 'Judul',
      render: (row) => (
        <button
          type="button"
          className="text-left font-medium text-primary hover:underline max-w-xs truncate block"
          onClick={() => navigate(`/tenders/${row.id}`)}
        >
          {row.title}
        </button>
      ),
    },
    {
      key: 'buyer_name',
      header: 'Buyer',
      render: (row) => <span className="text-fg-muted">{row.buyer_name ?? '—'}</span>,
    },
    {
      key: 'value_estimate',
      header: 'Nilai',
      align: 'right',
      render: (row) =>
        row.value_estimate != null
          ? <span className="tabular-nums">{formatRupiahShort(row.value_estimate)}</span>
          : <span className="text-fg-subtle">—</span>,
    },
    {
      key: 'submission_deadline',
      header: 'Deadline',
      render: (row) => {
        if (!row.submission_deadline) return <span className="text-fg-subtle">—</span>
        const tone = deadlineTone(row.submission_deadline)
        return (
          <span className={cn(
            'text-caption font-medium',
            tone === 'danger' && 'text-danger',
            tone === 'warning' && 'text-warning',
          )}>
            {tone !== 'normal' && '⚠ '}
            {formatTanggal(row.submission_deadline)}
          </span>
        )
      },
    },
    {
      key: 'status',
      header: 'Status',
      render: (row) => <StagePill stage={row.status} />,
    },
    {
      key: 'fit_score',
      header: 'Fit Score',
      align: 'center',
      render: (row) =>
        row.fit_score != null
          ? <ScoreRing score={row.fit_score} size={32} strokeWidth={4} showLabel />
          : <span className="text-fg-subtle">—</span>,
    },
    {
      key: 'recommended_action',
      header: 'Rekomendasi',
      render: (row) =>
        row.recommended_action
          ? <ActionBadge action={actionToLabel(row.recommended_action)} />
          : <span className="text-fg-subtle">—</span>,
    },
    {
      key: 'origin',
      header: 'Origin',
      align: 'center',
      render: (row) =>
        row.origin === 'discovery'
          ? <Sparkles className="w-4 h-4 text-accent mx-auto" aria-label="Ditemukan AI" />
          : null,
    },
  ]

  function openCreate() {
    setEditingTender(undefined)
    setDrawerOpen(true)
  }

  function openEdit(tender: Tender) {
    setEditingTender(tender)
    setDrawerOpen(true)
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success('Tender dihapus.')
    } catch {
      toast.error('Gagal menghapus tender.')
    } finally {
      setDeleteTarget(null)
    }
  }

  async function handlePromote(tender: Tender) {
    try {
      await promoteMutation.mutateAsync(tender.id)
      toast.success('Tender dipromosikan ke pipeline.')
    } catch {
      toast.error('Gagal mempromosikan tender.')
    }
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-h2 font-semibold text-fg">Tenders</h1>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
          + Tender Baru
        </Button>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap gap-3">
        <Select
          className="w-40"
          value={filters.status ?? ''}
          onChange={(e) => {
            const v = e.target.value as TenderStatus | ''
            setFilters((f) => ({ ...f, status: v || undefined }))
            setPage(1)
          }}
        >
          <option value="">Semua Status</option>
          <option value="IDENTIFIED">IDENTIFIED</option>
          <option value="QUALIFYING">QUALIFYING</option>
          <option value="BIDDING">BIDDING</option>
          <option value="SUBMITTED">SUBMITTED</option>
          <option value="WON">WON</option>
          <option value="LOST">LOST</option>
        </Select>

        <Select
          className="w-44"
          value={filters.recommended_action ?? ''}
          onChange={(e) => {
            const v = e.target.value as TenderApiAction | ''
            setFilters((f) => ({ ...f, recommended_action: v || undefined }))
            setPage(1)
          }}
        >
          <option value="">Semua Rekomendasi</option>
          <option value="PURSUE">Pursue</option>
          <option value="REVIEW">Review</option>
          <option value="WATCHLIST">Watchlist</option>
          <option value="REJECT">Reject</option>
          <option value="NEED_PARTNER">Need Partner</option>
        </Select>

        <Select
          className="w-36"
          value={filters.origin ?? ''}
          onChange={(e) => {
            const v = e.target.value as TenderOrigin | ''
            setFilters((f) => ({ ...f, origin: v || undefined }))
            setPage(1)
          }}
        >
          <option value="">Semua Origin</option>
          <option value="manual">Manual</option>
          <option value="discovery">AI Discovery</option>
        </Select>

        <Input
          className="w-48"
          placeholder="Cari judul/buyer…"
          value={filters.search ?? ''}
          onChange={(e) => {
            setFilters((f) => ({ ...f, search: e.target.value || undefined }))
            setPage(1)
          }}
        />
      </div>

      {/* Table */}
      <div className="bg-surface border border-line rounded-card overflow-hidden">
        <Table<Tender>
          columns={columns}
          data={data?.items ?? []}
          rowKey={(t) => t.id}
          loading={isLoading}
          page={page}
          total={data?.total}
          pageSize={filters.page_size ?? 20}
          onPageChange={setPage}
          empty={
            <EmptyState
              icon={<FileSearch className="w-6 h-6" />}
              title="Belum ada tender"
              description="Mulai tambahkan tender baru atau jalankan Radar Tender."
              action={
                <Button size="sm" onClick={openCreate}>
                  + Tender Baru
                </Button>
              }
            />
          }
          kebabActions={(row) => [
            {
              label: 'Lihat Detail',
              onClick: () => navigate(`/tenders/${row.id}`),
            },
            {
              label: 'Edit',
              onClick: () => openEdit(row),
            },
            {
              label: 'Generate Proposal',
              onClick: () => setProposalTender(row),
            },
            ...(row.origin === 'discovery' && row.status === 'IDENTIFIED'
              ? [{ label: '✨ Promote ke Pipeline', onClick: () => handlePromote(row) }]
              : []),
            {
              label: 'Hapus',
              onClick: () => setDeleteTarget(row),
              danger: true,
            },
          ]}
        />
      </div>

      {/* Form Drawer */}
      <TenderFormDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        tender={editingTender}
        onSaved={() => setDrawerOpen(false)}
      />

      {/* Delete confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        title="Hapus Tender?"
        description={`"${deleteTarget?.title}" akan dihapus permanen.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteMutation.isPending}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />

      {/* Draf Proposal (kebab "Generate Proposal") */}
      <ProposalDraftDrawer
        open={!!proposalTender}
        onClose={() => setProposalTender(null)}
        tenderId={proposalTender?.id}
        tenderTitle={proposalTender?.title}
      />
    </div>
  )
}
