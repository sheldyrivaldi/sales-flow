import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ChevronLeft, Copy, Download, Trash2 } from 'lucide-react'

import Button from '../../components/ui/Button'
import { formatTanggal, formatRelative } from '../../lib/format'
import { toast } from '../../lib/toast'
import { REPORT_TYPE_LABELS } from '../../api/reports'
import type { Report } from '../../api/reports'

export interface ReportViewerProps {
  report: Report
  onBack: () => void
  onDelete: () => void
  deleting?: boolean
}

function downloadMarkdown(filename: string, markdown: string) {
  const blob = new Blob([markdown], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export default function ReportViewer({ report, onBack, onDelete, deleting }: ReportViewerProps) {
  function handleCopy() {
    navigator.clipboard.writeText(report.content).then(
      () => toast.success('Laporan disalin ke clipboard.'),
      () => toast.error('Gagal menyalin ke clipboard.'),
    )
  }

  function handleExport() {
    downloadMarkdown(`${report.report_type}-${report.id}.md`, report.content)
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1 text-caption text-fg-muted hover:text-fg"
        >
          <ChevronLeft className="w-3.5 h-3.5" aria-hidden="true" />
          Kembali ke daftar
        </button>
        <div className="flex items-center gap-1.5">
          <Button size="sm" variant="ghost" leftIcon={<Copy className="w-3.5 h-3.5" />} onClick={handleCopy}>
            Salin
          </Button>
          <Button size="sm" variant="ghost" leftIcon={<Download className="w-3.5 h-3.5" />} onClick={handleExport}>
            Export
          </Button>
          <Button
            size="sm"
            variant="ghost"
            leftIcon={<Trash2 className="w-3.5 h-3.5" />}
            loading={deleting}
            onClick={onDelete}
          >
            Hapus
          </Button>
        </div>
      </div>

      <div className="rounded-card border border-line bg-surface p-5">
        <div className="mb-3 pb-3 border-b border-line">
          <h2 className="text-h3 font-semibold text-fg">{report.title}</h2>
          <p className="text-caption text-fg-muted mt-1">
            {REPORT_TYPE_LABELS[report.report_type]} • {formatTanggal(report.period_start)} –{' '}
            {formatTanggal(report.period_end)} • Dibuat AI{report.model ? ` • ${report.model}` : ''} •{' '}
            {formatRelative(report.created_at)}
          </p>
        </div>

        <div className="prose-custom">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              p: ({ children }) => <p className="mb-3 last:mb-0 text-body text-fg">{children}</p>,
              ul: ({ children }) => <ul className="list-disc pl-4 mb-3 space-y-0.5 text-body text-fg">{children}</ul>,
              ol: ({ children }) => <ol className="list-decimal pl-4 mb-3 space-y-0.5 text-body text-fg">{children}</ol>,
              li: ({ children }) => <li>{children}</li>,
              strong: ({ children }) => <strong className="font-semibold text-fg">{children}</strong>,
              h1: ({ children }) => <h1 className="text-h2 font-semibold mb-2 mt-1">{children}</h1>,
              h2: ({ children }) => <h2 className="text-h3 font-semibold mb-2 mt-4">{children}</h2>,
              h3: ({ children }) => <h3 className="text-body font-semibold mb-1 mt-3">{children}</h3>,
              table: ({ children }) => (
                <div className="overflow-x-auto mb-3">
                  <table className="min-w-full text-body text-fg border-collapse">{children}</table>
                </div>
              ),
              th: ({ children }) => (
                <th className="text-left border-b border-line py-1.5 pr-4 font-semibold">{children}</th>
              ),
              td: ({ children }) => <td className="border-b border-line py-1.5 pr-4">{children}</td>,
            }}
          >
            {report.content}
          </ReactMarkdown>
        </div>
      </div>
    </div>
  )
}
