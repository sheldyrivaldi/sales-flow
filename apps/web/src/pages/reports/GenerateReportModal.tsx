import { useState } from 'react'
import Modal from '../../components/ui/Modal'
import Button from '../../components/ui/Button'
import Field from '../../components/ui/Field'
import Select from '../../components/ui/Select'
import DatePicker from '../../components/ui/DatePicker'
import { toast } from '../../lib/toast'
import { useGenerateReport } from '../../api/reports'
import type { ReportType } from '../../api/reports'
import { REPORT_TYPE_LABELS } from '../../api/reports'

export interface GenerateReportModalProps {
  open: boolean
  onClose: () => void
  onGenerated?: (reportId: string) => void
}

function toRFC3339Start(dateStr: string): string {
  return `${dateStr}T00:00:00Z`
}
function toRFC3339End(dateStr: string): string {
  return `${dateStr}T23:59:59Z`
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

export default function GenerateReportModal({ open, onClose, onGenerated }: GenerateReportModalProps) {
  const [type, setType] = useState<ReportType>('daily_digest')
  const [start, setStart] = useState(todayISO())
  const [end, setEnd] = useState(todayISO())
  const generate = useGenerateReport()

  async function handleGenerate() {
    if (!start || !end) {
      toast.error('Periode wajib diisi.')
      return
    }
    if (start > end) {
      toast.error('Tanggal mulai harus sebelum atau sama dengan tanggal akhir.')
      return
    }
    try {
      const report = await generate.mutateAsync({
        type,
        period_start: toRFC3339Start(start),
        period_end: toRFC3339End(end),
      })
      toast.success('Laporan berhasil dibuat.')
      onGenerated?.(report.id)
      onClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Generate laporan gagal, coba lagi nanti.')
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Generate Laporan"
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={generate.isPending}>
            Batal
          </Button>
          <Button loading={generate.isPending} onClick={handleGenerate}>
            Generate
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Tipe Laporan" htmlFor="report-type">
          <Select id="report-type" value={type} onChange={(e) => setType(e.target.value as ReportType)}>
            {(Object.keys(REPORT_TYPE_LABELS) as ReportType[]).map((t) => (
              <option key={t} value={t}>
                {REPORT_TYPE_LABELS[t]}
              </option>
            ))}
          </Select>
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Periode mulai" htmlFor="report-start">
            <DatePicker id="report-start" value={start} onChange={(e) => setStart(e.target.value)} />
          </Field>
          <Field label="Periode akhir" htmlFor="report-end">
            <DatePicker id="report-end" value={end} onChange={(e) => setEnd(e.target.value)} />
          </Field>
        </div>
      </div>
    </Modal>
  )
}
