import {
  Sparkles,
  Loader2,
  Building2,
  Wrench,
  HelpCircle,
  RefreshCw,
  Paperclip,
  AlertCircle,
  Lock,
} from 'lucide-react'

import Card, { CardHeader, CardBody } from '../ui/Card'
import Button from '../ui/Button'
import AnalysisMarkdown from './AnalysisMarkdown'
import { toast } from '../../lib/toast'
import { formatRelative } from '../../lib/format'
import { useAnalyzeEvent } from '../../api/events'
import type { EventAnalysis, AnalysisStatus } from '../../api/events'

function Sub({ icon: Icon, title, hint }: { icon: typeof Wrench; title: string; hint?: string }) {
  return (
    <div className="flex items-baseline gap-2 flex-wrap">
      <span className="inline-flex items-center gap-1.5 text-body font-semibold text-fg">
        <Icon className="w-4 h-4 text-primary" aria-hidden="true" />
        {title}
      </span>
      {hint && <span className="text-caption text-fg-subtle">{hint}</span>}
    </div>
  )
}

export interface EventAnalysisPanelProps {
  eventId: string
  analysis?: EventAnalysis
  analyzedAt?: string
  status: AnalysisStatus
  error?: string
  attachmentCount: number
}

/**
 * Analisa AI event — inti nilai menu ini.
 *
 * Berjalan ASINKRON seperti generate playbook: menekan tombol hanya menitipkan
 * tugas, lalu Hermes melapor balik saat selesai. Selama status "running"
 * seluruh event dikunci (lihat EventDetail) supaya hasil tidak dihitung dari
 * data yang berubah di tengah jalan.
 *
 * Tidak ada tombol unggah di sini: bahan diambil dari SELURUH event, termasuk
 * semua lampirannya. Berkas ditambahkan lewat kartu Lampiran, satu tempat saja.
 */
export default function EventAnalysisPanel({
  eventId,
  analysis,
  analyzedAt,
  status,
  error,
  attachmentCount,
}: EventAnalysisPanelProps) {
  const analyze = useAnalyzeEvent()
  const running = status === 'running'
  const has = !!analysis

  async function run() {
    try {
      await analyze.mutateAsync(eventId)
      toast.success('Analisa dimulai, hasilnya muncul otomatis di sini.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Analisa AI gagal dimulai.')
    }
  }

  return (
    <Card>
      <CardHeader className="flex items-center gap-2 flex-wrap">
        <Sparkles className="w-4 h-4 text-primary" aria-hidden="true" />
        <h3 className="text-body font-semibold text-fg">Analisa AI</h3>
        {analyzedAt && !running && (
          <span className="text-caption text-fg-subtle">diperbarui {formatRelative(analyzedAt)}</span>
        )}
        <Button
          size="sm"
          className="ml-auto"
          loading={analyze.isPending}
          disabled={running || analyze.isPending}
          leftIcon={has ? <RefreshCw className="w-3.5 h-3.5" /> : <Sparkles className="w-3.5 h-3.5" />}
          onClick={() => void run()}
        >
          {running ? 'Sedang Berjalan…' : has ? 'Analisa Ulang' : 'Jalankan Analisa'}
        </Button>
      </CardHeader>

      <CardBody className="flex flex-col gap-5">
        <p className="inline-flex items-center gap-1.5 text-caption text-fg-muted">
          <Paperclip className="w-3.5 h-3.5" aria-hidden="true" />
          Membaca seluruh data event ini
          {attachmentCount > 0
            ? ` termasuk ${attachmentCount} lampiran.`
            : '. Tambahkan berkas di kartu Lampiran agar analisa lebih dalam.'}
        </p>

        {running && (
          <div className="rounded-card border border-primary-border bg-primary-subtle p-4 flex items-start gap-3">
            <Loader2 className="w-5 h-5 text-primary animate-spin shrink-0 mt-0.5" aria-hidden="true" />
            <div>
              <p className="text-body font-semibold text-fg">Analisa sedang berjalan</p>
              <p className="text-caption text-fg-muted mt-0.5">
                AI sedang membaca lampiran dan meriset di internet. Ini bisa belasan menit — kamu boleh
                meninggalkan halaman ini, hasilnya tersimpan otomatis saat selesai.
              </p>
              <p className="inline-flex items-center gap-1.5 text-caption text-fg-subtle mt-2">
                <Lock className="w-3.5 h-3.5" aria-hidden="true" />
                Event dikunci selama proses ini agar hasilnya konsisten dengan datanya.
              </p>
            </div>
          </div>
        )}

        {status === 'failed' && error && (
          <div className="rounded-card border border-danger-border bg-danger-subtle p-3 flex items-start gap-2">
            <AlertCircle className="w-4 h-4 text-danger mt-0.5 shrink-0" aria-hidden="true" />
            <div>
              <p className="text-body text-fg">{error}</p>
              <p className="text-caption text-fg-muted mt-0.5">Klik Analisa Ulang untuk mencoba lagi.</p>
            </div>
          </div>
        )}

        {!has && !running && status !== 'failed' && (
          <div className="rounded-card border border-dashed border-line bg-surface-subtle p-4 text-center">
            <p className="text-body text-fg">Belum ada analisa untuk event ini.</p>
            <p className="text-caption text-fg-muted mt-1 max-w-2xl mx-auto">
              AI membaca identitas event, catatan tim, daftar undangan, dan isi seluruh lampiran, lalu
              meriset di internet — menghasilkan ringkasan, apa yang bisa diolah di internal perusahaan,
              dan peluang klien baru.
            </p>
          </div>
        )}

        {has && analysis && (
          <>
            {analysis.summary && (
              <div className="rounded-card border border-primary/25 bg-primary-subtle p-3.5">
                <AnalysisMarkdown>{analysis.summary}</AnalysisMarkdown>
              </div>
            )}

            {analysis.sections?.map((sec, i) => (
              <div key={i} className="flex flex-col gap-2 animate-row-in">
                <Sub icon={Sparkles} title={sec.title} />
                <div className="rounded-card border border-line bg-surface p-3.5">
                  <AnalysisMarkdown>{sec.body}</AnalysisMarkdown>
                </div>
              </div>
            ))}

            {analysis.internal_opportunities && (
              <div className="flex flex-col gap-2">
                <Sub icon={Wrench} title="Untuk Internal Perusahaan" hint="yang bisa diolah sendiri" />
                <div className="rounded-card border border-line bg-surface p-3.5">
                  <AnalysisMarkdown>{analysis.internal_opportunities}</AnalysisMarkdown>
                </div>
              </div>
            )}

            {analysis.client_opportunities && (
              <div className="flex flex-col gap-2">
                <Sub icon={Building2} title="Peluang Klien Baru" hint="organisasi dan cara masuknya" />
                <div className="rounded-card border border-primary/25 bg-primary-subtle p-3.5">
                  <AnalysisMarkdown>{analysis.client_opportunities}</AnalysisMarkdown>
                </div>
              </div>
            )}

            {/* Kejujuran data sengaja DITAMPILKAN: analisa yang mengaku tidak
                tahu jauh lebih berguna daripada yang menambal kekosongan. */}
            {analysis.data_gaps?.length > 0 && (
              <div className="rounded-card border border-line bg-surface-subtle p-3">
                <Sub icon={HelpCircle} title="Yang Belum Bisa Disimpulkan" hint="perlu data tambahan" />
                {/* Tiap butir bisa memuat markdown (mis. **nama file** ditebalkan).
                    Dirender lewat AnalysisMarkdown sebagai daftar agar tidak tampil
                    mentah sebagai "**teks**". */}
                <div className="mt-1.5">
                  <AnalysisMarkdown>
                    {analysis.data_gaps.map((g) => `- ${g}`).join('\n')}
                  </AnalysisMarkdown>
                </div>
              </div>
            )}
          </>
        )}
      </CardBody>
    </Card>
  )
}
