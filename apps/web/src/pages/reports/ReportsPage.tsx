import { useState } from 'react'
import { FileText, Sparkles } from 'lucide-react'

import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import Select from '../../components/ui/Select'
import Badge from '../../components/ui/Badge'
import EmptyState from '../../components/ui/EmptyState'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import { toast } from '../../lib/toast'
import { formatTanggal, formatRelative } from '../../lib/format'

import { useReports, useReport, useDeleteReport } from '../../api/reports'
import type { Report, ReportType } from '../../api/reports'
import { REPORT_TYPE_LABELS } from '../../api/reports'
import GenerateReportModal from './GenerateReportModal'
import ReportViewer from './ReportViewer'

export default function ReportsPage() {
  const [typeFilter, setTypeFilter] = useState<ReportType | ''>('')
  const [page, setPage] = useState(1)
  const [generateOpen, setGenerateOpen] = useState(false)
  const [viewingId, setViewingId] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Report | null>(null)

  const { data, isLoading } = useReports({
    type: typeFilter || undefined,
    page,
    page_size: 20,
  })
  const { data: viewingReport } = useReport(viewingId ?? undefined)
  const deleteReport = useDeleteReport()

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteReport.mutateAsync(deleteTarget.id)
      toast.success('Laporan dihapus.')
      if (viewingId === deleteTarget.id) setViewingId(null)
      setDeleteTarget(null)
    } catch {
      toast.error('Gagal menghapus laporan.')
    }
  }

  if (viewingId && viewingReport) {
    return (
      <div className="p-6">
        <ReportViewer
          report={viewingReport}
          onBack={() => setViewingId(null)}
          onDelete={() => setDeleteTarget(viewingReport)}
          deleting={deleteReport.isPending}
        />
        <ConfirmDialog
          open={!!deleteTarget}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={handleDelete}
          title="Hapus laporan?"
          description="Laporan yang dihapus tidak dapat dikembalikan."
          confirmLabel="Hapus"
          tone="danger"
          loading={deleteReport.isPending}
        />
      </div>
    )
  }

  const columns: Column<Report>[] = [
    {
      key: 'report_type',
      header: 'Tipe',
      render: (r) => <Badge tone="accent">{REPORT_TYPE_LABELS[r.report_type]}</Badge>,
    },
    { key: 'title', header: 'Judul', render: (r) => r.title },
    {
      key: 'period',
      header: 'Periode',
      render: (r) => `${formatTanggal(r.period_start)} – ${formatTanggal(r.period_end)}`,
    },
    {
      key: 'created_at',
      header: 'Dibuat',
      render: (r) => (
        <span className="text-caption text-fg-muted flex items-center gap-1">
          <Sparkles className="w-3 h-3 text-accent" aria-hidden="true" />
          {formatRelative(r.created_at)}
        </span>
      ),
    },
  ]

  return (
    <div className="p-6 flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-h2 font-semibold text-fg">Reports</h1>
        <Button leftIcon={<Sparkles className="w-4 h-4" />} onClick={() => setGenerateOpen(true)}>
          Generate Laporan
        </Button>
      </div>

      <div className="flex items-center gap-2">
        <Select
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value as ReportType | '')
            setPage(1)
          }}
          className="w-64"
        >
          <option value="">Semua tipe</option>
          {(Object.keys(REPORT_TYPE_LABELS) as ReportType[]).map((t) => (
            <option key={t} value={t}>
              {REPORT_TYPE_LABELS[t]}
            </option>
          ))}
        </Select>
      </div>

      <div className="bg-surface border border-line rounded-card overflow-hidden">
        <Table<Report>
          columns={columns}
          data={data?.items ?? []}
          rowKey={(r) => r.id}
          loading={isLoading}
          page={page}
          total={data?.total}
          pageSize={20}
          onPageChange={setPage}
          empty={
            <EmptyState
              icon={<FileText className="w-6 h-6" />}
              title="Belum ada laporan"
              description="Generate laporan pertama: Daily Digest, Weekly Pipeline, atau Per-Peluang."
              action={
                <Button size="sm" onClick={() => setGenerateOpen(true)}>
                  Generate Laporan
                </Button>
              }
            />
          }
          kebabActions={(row) => [
            { label: 'Lihat', onClick: () => setViewingId(row.id) },
            { label: 'Hapus', onClick: () => setDeleteTarget(row), danger: true },
          ]}
        />
      </div>

      <GenerateReportModal
        open={generateOpen}
        onClose={() => setGenerateOpen(false)}
        onGenerated={(id) => setViewingId(id)}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onCancel={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Hapus laporan?"
        description="Laporan yang dihapus tidak dapat dikembalikan."
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteReport.isPending}
      />
    </div>
  )
}
