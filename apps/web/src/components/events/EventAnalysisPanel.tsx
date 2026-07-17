import { useRef, useState } from 'react'
import { CalendarClock, FileSpreadsheet, Sparkles, Upload, X } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Button from '../ui/Button'
import Badge from '../ui/Badge'
import { SkeletonText } from '../ui/Skeleton'
import { cn } from '../../lib/cn'
import { toast } from '../../lib/toast'
import { useAnalyzeEvent } from '../../api/events'
import type { EventCompanyInsight, EventQuadrant } from '../../api/events'

const QUADRANTS: { key: EventQuadrant; title: string; hint: string; className: string }[] = [
  { key: 'prioritas_utama', title: 'Prioritas Utama', hint: 'Potensi tinggi · minat tinggi', className: 'border-success-border bg-success-subtle' },
  { key: 'perlu_digarap',   title: 'Perlu Digarap',   hint: 'Potensi tinggi · minat rendah', className: 'border-info-border bg-info-subtle' },
  { key: 'quick_win',       title: 'Quick Win',       hint: 'Potensi rendah · minat tinggi', className: 'border-warning-border bg-warning-subtle' },
  { key: 'dipantau',        title: 'Dipantau',        hint: 'Potensi rendah · minat rendah', className: 'border-line bg-surface-subtle' },
]

/** Konversi file Excel (.xlsx) menjadi teks CSV di browser — hasilnya
 * dikirim sebagai teks ke AI (exceljs sudah jadi dependency untuk export). */
async function xlsxToCsv(file: File): Promise<string> {
  const { default: Excel } = await import('exceljs')
  const wb = new Excel.Workbook()
  await wb.xlsx.load(await file.arrayBuffer())
  const lines: string[] = []
  const ws = wb.worksheets[0]
  ws?.eachRow((row) => {
    const cells: string[] = []
    row.eachCell({ includeEmpty: true }, (cell) => {
      cells.push(String(cell.value ?? '').replace(/[\r\n,]+/g, ' ').trim())
    })
    lines.push(cells.join(','))
  })
  return lines.join('\n')
}

function CompanyChip({ company }: { company: EventCompanyInsight }) {
  return (
    <div className="rounded-btn bg-surface border border-line px-2.5 py-1.5 shadow-subtle">
      <p className="text-caption font-semibold text-fg truncate">{company.name}</p>
      {company.industry && <p className="text-caption text-fg-muted truncate">{company.industry}</p>}
      {company.note && <p className="text-caption text-fg-subtle mt-0.5 line-clamp-2">{company.note}</p>}
    </div>
  )
}

/** Analisa peserta event pasca-acara: unggah daftar peserta (PDF/Excel), AI
 * mengekstrak perusahaan + riset web, memetakan ke kuadran 2x2 (potensi ×
 * minat), lalu memberi ringkasan dan timeline follow-up sales otomatis. */
export default function EventAnalysisPanel({ eventId }: { eventId: string }) {
  const analyze = useAnalyzeEvent()
  const [file, setFile] = useState<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleAnalyze() {
    if (!file) return
    // Toast sukses/gagal ditangani MutationCache global (main.tsx).
    if (file.name.toLowerCase().endsWith('.xlsx')) {
      const tableText = await xlsxToCsv(file)
      analyze.mutate({ id: eventId, tableText })
    } else {
      analyze.mutate({ id: eventId, file })
    }
  }

  function handlePick(f: File | undefined) {
    if (!f) return
    if (f.size > 10 * 1024 * 1024) {
      toast.error('Ukuran dokumen maksimal 10 MB.')
      return
    }
    setFile(f)
  }

  const result = analyze.data
  const byQuadrant = (q: EventQuadrant) => result?.companies.filter((c) => c.quadrant === q) ?? []

  return (
    <Card>
      <CardHeader className="flex items-center gap-2">
        <Sparkles className="w-4 h-4 text-accent" aria-hidden="true" />
        <h3 className="text-body font-semibold text-fg">Analisa Peserta (AI)</h3>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        {/* Input */}
        <div className="flex flex-wrap items-center gap-2">
          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf,.xlsx"
            className="sr-only"
            tabIndex={-1}
            aria-hidden="true"
            onChange={(e) => {
              handlePick(e.target.files?.[0])
              e.target.value = ''
            }}
          />
          <Button
            variant="secondary"
            size="sm"
            leftIcon={file?.name.toLowerCase().endsWith('.xlsx') ? <FileSpreadsheet className="w-3.5 h-3.5" /> : <Upload className="w-3.5 h-3.5" />}
            disabled={analyze.isPending}
            onClick={() => fileInputRef.current?.click()}
          >
            {file ? file.name : 'Pilih dokumen peserta (PDF/Excel)'}
          </Button>
          {file && (
            <button
              type="button"
              aria-label="Hapus file"
              onClick={() => setFile(null)}
              className="text-fg-muted hover:text-danger transition-colors"
            >
              <X className="w-4 h-4" aria-hidden="true" />
            </button>
          )}
          <Button size="sm" loading={analyze.isPending} disabled={!file} onClick={() => void handleAnalyze()}>
            Analisa
          </Button>
        </div>
        {!result && !analyze.isPending && (
          <p className="text-caption text-fg-muted">
            AI membaca daftar peserta, mencari info tiap perusahaan di internet, memetakan ke kuadran
            potensi × minat, dan menyusun timeline follow-up otomatis.
          </p>
        )}

        {analyze.isPending && (
          <>
            <p className="text-caption text-fg-muted">
              AI membaca dokumen dan meriset perusahaan peserta — bisa memakan beberapa menit…
            </p>
            <SkeletonText lines={6} />
          </>
        )}

        {result && (
          <>
            {/* Ringkasan */}
            <p className="text-body text-fg">{result.summary}</p>

            {/* Kuadran 2x2 */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {QUADRANTS.map((q) => {
                const companies = byQuadrant(q.key)
                return (
                  <div key={q.key} className={cn('rounded-card border p-3 flex flex-col gap-2', q.className)}>
                    <div className="flex items-center justify-between gap-2">
                      <div>
                        <p className="text-body font-semibold text-fg">{q.title}</p>
                        <p className="text-caption text-fg-muted">{q.hint}</p>
                      </div>
                      <Badge tone={q.key === 'prioritas_utama' ? 'success' : q.key === 'dipantau' ? 'info' : 'warning'}>
                        {companies.length}
                      </Badge>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      {companies.length === 0 ? (
                        <p className="text-caption text-fg-subtle">Tidak ada.</p>
                      ) : (
                        companies.map((c, i) => <CompanyChip key={i} company={c} />)
                      )}
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Timeline follow-up */}
            {result.timeline_suggestions.length > 0 && (
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-1.5">
                  <CalendarClock className="w-4 h-4 text-primary" aria-hidden="true" />
                  <h4 className="text-body font-semibold text-fg">Timeline Follow-up</h4>
                </div>
                <ol className="relative flex flex-col gap-2.5 pl-5 border-l-2 border-primary-border ml-1.5">
                  {result.timeline_suggestions.map((t, i) => (
                    <li key={i} className="relative text-body text-fg">
                      <span
                        className="absolute -left-[1.44rem] top-1.5 h-2.5 w-2.5 rounded-full bg-primary ring-4 ring-primary-subtle"
                        aria-hidden="true"
                      />
                      {t}
                    </li>
                  ))}
                </ol>
              </div>
            )}
          </>
        )}
      </CardBody>
    </Card>
  )
}
