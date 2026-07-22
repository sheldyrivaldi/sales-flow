import { useRef, useState } from 'react'
import {
  BookOpen,
  Loader2,
  Eye,
  Sparkles,
  AlertCircle,
  Download,
  Upload,
  X,
  SendHorizonal,
  RotateCcw,
  Paperclip,
  RefreshCw,
} from 'lucide-react'

import Card, { CardHeader, CardBody } from '../ui/Card'
import Button from '../ui/Button'
import Badge from '../ui/Badge'
import Modal from '../ui/Modal'
import Input from '../ui/Input'
import Textarea from '../ui/Textarea'
import PlaybookSlideViewer from '../playbooks/PlaybookSlideViewer'
import AttachmentPreview from '../chat/AttachmentPreview'
import { toast } from '../../lib/toast'
import { formatRelative } from '../../lib/format'
import { exportPlaybookPpt } from '../../lib/exportPlaybookPpt'
import {
  usePlaybookJobs,
  useCreateEventPlaybookJob,
  useRefinePlaybookJob,
  useRetryPlaybookJob,
  isJobActive,
  PLAYBOOK_STATUS_LABEL,
} from '../../api/playbookJobs'
import type { PlaybookJob } from '../../api/playbookJobs'
import type { EventAttachment } from '../../api/events'

export interface EventPlaybookCardProps {
  eventId: string
  eventName: string
  /** Lampiran event — otomatis ikut sebagai konteks saat generate playbook. */
  attachments: EventAttachment[]
}

const MAX_MB = 10

/** Judul yang diketik user menang di mana pun (slide cover, nama file). */
function effectiveContent(job: PlaybookJob) {
  if (!job.content) return undefined
  if (!job.user_titled) return job.content
  return { ...job.content, title: job.title }
}

/**
 * Kartu Playbook di halaman detail Event.
 *
 * Playbook yang di-generate dari sebuah event TERTAUT ke event itu (event_id),
 * jadi hasilnya tampil, bisa dibuka, direvisi, dan di-generate ulang langsung
 * di sini — bukan cuma nyangkut di menu Playbooks. SATU event = SATU playbook:
 * generate ulang melepas yang lama dan menautkan yang baru. Seluruh lampiran
 * event otomatis ikut sebagai konteks saat generate.
 */
export default function EventPlaybookCard({ eventId, eventName, attachments }: EventPlaybookCardProps) {
  const { data: jobs } = usePlaybookJobs()
  const createMutation = useCreateEventPlaybookJob()
  const refineMutation = useRefinePlaybookJob()
  const retryMutation = useRetryPlaybookJob()

  // Tautan yang benar: berdasar event_id, bukan tebakan judul.
  const job = jobs?.find((j) => j.event_id === eventId) ?? null

  // Modal generate (title + prompt + lampiran tambahan) — sama seperti Playbooks.
  const [createOpen, setCreateOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [prompt, setPrompt] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  // Modal buka/revisi hasil.
  const [viewOpen, setViewOpen] = useState(false)
  const [instruction, setInstruction] = useState('')
  const [refineFile, setRefineFile] = useState<File | null>(null)
  const refineFileRef = useRef<HTMLInputElement>(null)

  const busy = createMutation.isPending || (job ? isJobActive(job.status) : false)

  function openCreate() {
    setTitle(job ? job.title : `Playbook Event: ${eventName}`)
    setPrompt('')
    setFile(null)
    setCreateOpen(true)
  }

  function pick(f: File | undefined, set: (f: File) => void) {
    if (!f) return
    if (f.size > MAX_MB * 1024 * 1024) {
      toast.error(`Ukuran lampiran maksimal ${MAX_MB} MB.`)
      return
    }
    set(f)
  }

  async function handleGenerate() {
    try {
      await createMutation.mutateAsync({ eventId, title: title.trim(), prompt: prompt.trim(), file })
      toast.success(
        job
          ? 'Generate ulang dimulai, playbook lama dilepas dan yang baru akan muncul di sini.'
          : 'Playbook event sedang diproses, hasilnya muncul di sini otomatis.',
      )
      setCreateOpen(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal memulai generate playbook.')
    }
  }

  async function handleRefine() {
    if (!job) return
    const text = instruction.trim()
    if (!text) return
    try {
      await refineMutation.mutateAsync({ id: job.id, instruction: text, file: refineFile })
      toast.success('Revisi playbook sedang diproses, hasilnya diperbarui otomatis.')
      setInstruction('')
      setRefineFile(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal memulai revisi.')
    }
  }

  async function handleRetry() {
    if (!job) return
    try {
      await retryMutation.mutateAsync(job.id)
      toast.success('Playbook dicoba ulang, statusnya berubah otomatis.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal mencoba ulang.')
    }
  }

  function openPpt() {
    const content = job && effectiveContent(job)
    if (!content) return
    void exportPlaybookPpt(content, job!.title).catch(() => toast.error('Ekspor PPT gagal.'))
  }

  return (
    <Card>
      <CardHeader className="flex items-center gap-2 flex-wrap">
        <BookOpen className="w-4 h-4 text-primary" aria-hidden="true" />
        <h3 className="text-body font-semibold text-fg">Playbook Event</h3>
        {job && <span className="text-caption text-fg-subtle">{formatRelative(job.updated_at)}</span>}
        <div className="ml-auto flex items-center gap-2">
          {job?.status === 'success' && (
            <Button size="sm" variant="ghost" leftIcon={<Eye className="w-3.5 h-3.5" />} onClick={() => setViewOpen(true)}>
              Buka
            </Button>
          )}
          {job?.status === 'failed' && (
            <Button
              size="sm"
              variant="ghost"
              leftIcon={<RotateCcw className="w-3.5 h-3.5" />}
              loading={retryMutation.isPending}
              onClick={() => void handleRetry()}
            >
              Coba Lagi
            </Button>
          )}
          <Button
            size="sm"
            leftIcon={job ? <RefreshCw className="w-3.5 h-3.5" /> : <Sparkles className="w-3.5 h-3.5" />}
            loading={busy}
            disabled={busy}
            onClick={openCreate}
          >
            {job ? 'Generate Ulang' : 'Generate Playbook'}
          </Button>
        </div>
      </CardHeader>

      <CardBody className="flex flex-col gap-3">
        <p className="text-caption text-fg-muted">
          AI menyusun playbook untuk memaksimalkan peluang dari event ini — memakai seluruh informasi
          event, {attachments.length > 0 ? `${attachments.length} lampiran, ` : ''}riset internet terbaru
          tentang penyelenggara, dan profil perusahaan.
        </p>

        {job ? (
          <div className="flex items-center gap-2 flex-wrap rounded-card border border-line bg-surface-subtle px-3 py-2">
            <Badge
              tone={
                job.status === 'success'
                  ? 'success'
                  : job.status === 'failed'
                    ? 'danger'
                    : job.status === 'updating'
                      ? 'warning'
                      : 'info'
              }
            >
              <span className="inline-flex items-center gap-1">
                {isJobActive(job.status) && <Loader2 className="w-3 h-3 animate-spin" aria-hidden="true" />}
                {PLAYBOOK_STATUS_LABEL[job.status]}
              </span>
            </Badge>
            <span className="text-caption text-fg truncate max-w-xs" title={job.title}>
              {job.title}
            </span>
            {job.status === 'failed' && job.error_message && (
              <span className="inline-flex items-center gap-1 text-caption text-danger">
                <AlertCircle className="w-3.5 h-3.5" aria-hidden="true" />
                {job.error_message}
              </span>
            )}
          </div>
        ) : (
          <p className="text-caption text-fg-subtle">Belum ada playbook untuk event ini.</p>
        )}
      </CardBody>

      {/* Modal generate — identik dengan menu Playbooks (title + prompt + lampiran). */}
      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={job ? 'Generate Ulang Playbook' : 'Generate Playbook Event'}
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setCreateOpen(false)}>
              Batal
            </Button>
            <Button loading={createMutation.isPending} leftIcon={<Sparkles className="w-4 h-4" />} onClick={() => void handleGenerate()}>
              Generate
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-3">
          {job && (
            <div className="flex items-start gap-2 rounded-card border border-warning-border bg-warning-subtle p-2.5 text-caption text-fg">
              <AlertCircle className="w-3.5 h-3.5 mt-0.5 shrink-0 text-warning" aria-hidden="true" />
              Event ini sudah punya playbook. Generate baru akan melepas tautan playbook lama (tetap ada di
              menu Playbooks) dan menautkan hasil yang baru.
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <label htmlFor="ev-pb-title" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
              Judul playbook
            </label>
            <Input id="ev-pb-title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder={`Playbook Event: ${eventName}`} />
            <p className="text-caption text-fg-subtle">Dipakai apa adanya untuk nama file dan slide cover.</p>
          </div>

          <div className="flex flex-col gap-1.5">
            <label htmlFor="ev-pb-prompt" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
              Arahan tambahan <span className="normal-case text-fg-subtle font-normal">(opsional)</span>
            </label>
            <Textarea
              id="ev-pb-prompt"
              rows={3}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="mis. fokus ke peluang cross-sell produk absensi, tekankan keunggulan harga…"
            />
            <p className="text-caption text-fg-subtle">
              Konteks event (nama, tanggal, penyelenggara, catatan) sudah otomatis dipakai. Isi ini bila ingin
              menajamkan arah playbook.
            </p>
          </div>

          {/* Lampiran event — otomatis ikut, ditampilkan agar user yakin. */}
          {attachments.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
                Lampiran event (otomatis disertakan)
              </span>
              <div className="flex flex-wrap gap-1.5">
                {attachments.map((a) => (
                  <span
                    key={a.url}
                    className="inline-flex items-center gap-1 rounded-pill border border-line bg-surface px-2 py-0.5 text-caption text-fg"
                  >
                    <Paperclip className="w-3 h-3 text-primary" aria-hidden="true" />
                    <span className="max-w-40 truncate">{a.name}</span>
                  </span>
                ))}
              </div>
            </div>
          )}

          <input
            ref={fileRef}
            type="file"
            accept=".pdf"
            className="sr-only"
            tabIndex={-1}
            aria-hidden="true"
            onChange={(e) => {
              pick(e.target.files?.[0], setFile)
              e.target.value = ''
            }}
          />
          {file ? (
            <div className="inline-flex items-center gap-2 self-start rounded-pill border border-accent/40 bg-accent-subtle px-3 py-1 text-caption text-fg">
              <Upload className="w-3.5 h-3.5 text-accent-hover" aria-hidden="true" />
              <span className="max-w-56 truncate font-medium">{file.name}</span>
              <button type="button" aria-label="Hapus lampiran" onClick={() => setFile(null)} className="text-fg-muted hover:text-danger">
                <X className="w-3.5 h-3.5" aria-hidden="true" />
              </button>
            </div>
          ) : (
            <Button variant="secondary" size="sm" className="self-start" leftIcon={<Upload className="w-3.5 h-3.5" />} onClick={() => fileRef.current?.click()}>
              Lampirkan Dokumen Tambahan (PDF)
            </Button>
          )}
        </div>
      </Modal>

      {/* Modal buka/revisi hasil — preview slide + download + revisi via prompt. */}
      <Modal open={viewOpen} onClose={() => setViewOpen(false)} title={job?.title ?? 'Playbook'} size="lg">
        {job && (
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-2 flex-wrap">
              <Badge tone="accent">Event</Badge>
              <span className="text-caption text-fg-subtle">{formatRelative(job.updated_at)}</span>
              {job.status === 'success' && job.content && (
                <Button size="sm" variant="secondary" className="ml-auto" leftIcon={<Download className="w-3.5 h-3.5" />} onClick={openPpt}>
                  Download PPT
                </Button>
              )}
            </div>

            {isJobActive(job.status) && (
              <div className="flex items-center gap-2 text-body text-fg-muted py-6 justify-center">
                <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
                {job.status === 'updating' ? 'Sedang merevisi playbook…' : 'Sedang menyusun playbook…'} Kamu boleh tinggal halaman ini.
              </div>
            )}

            {job.status === 'success' && job.content && (
              <>
                <PlaybookSlideViewer content={effectiveContent(job)!} fallbackTitle={job.title} />

                <div className="flex flex-col gap-2 pt-3 border-t border-line">
                  <label htmlFor="ev-pb-refine" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
                    Revisi dengan prompt
                  </label>
                  {refineFile && (
                    <div className="inline-flex items-center gap-2 self-start rounded-pill border border-accent/40 bg-accent-subtle px-3 py-1 text-caption text-fg">
                      <Upload className="w-3.5 h-3.5 text-accent-hover" aria-hidden="true" />
                      <span className="max-w-56 truncate font-medium">{refineFile.name}</span>
                      <button type="button" aria-label="Hapus lampiran" onClick={() => setRefineFile(null)} className="text-fg-muted hover:text-danger">
                        <X className="w-3.5 h-3.5" aria-hidden="true" />
                      </button>
                    </div>
                  )}
                  <div className="flex gap-2">
                    <input
                      ref={refineFileRef}
                      type="file"
                      accept=".pdf,.png,.jpg,.jpeg,.webp"
                      className="sr-only"
                      tabIndex={-1}
                      aria-hidden="true"
                      onChange={(e) => {
                        pick(e.target.files?.[0], setRefineFile)
                        e.target.value = ''
                      }}
                    />
                    <Button variant="secondary" size="md" aria-label="Lampirkan dokumen" onClick={() => refineFileRef.current?.click()} leftIcon={<Upload className="w-4 h-4" />}>
                      Lampiran
                    </Button>
                    <Input
                      id="ev-pb-refine"
                      className="flex-1"
                      value={instruction}
                      onChange={(e) => setInstruction(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault()
                          void handleRefine()
                        }
                      }}
                      placeholder="mis. tambahkan strategi pricing, perbaiki slide risiko…"
                    />
                    <Button loading={refineMutation.isPending} leftIcon={<SendHorizonal className="w-3.5 h-3.5" />} onClick={() => void handleRefine()}>
                      Revisi
                    </Button>
                  </div>
                </div>
              </>
            )}

            {/* Prompt + riwayat revisi */}
            <div className="flex flex-col gap-2 pt-3 border-t border-line">
              <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Prompt &amp; Riwayat Revisi</p>
              <p className="text-body text-fg whitespace-pre-wrap">{job.prompt}</p>
              {job.attachment_url && <AttachmentPreview url={job.attachment_url} name={job.attachment_name} align="start" />}
              {(job.revisions ?? []).map((r, i) => (
                <div key={i} className="flex flex-col gap-1.5 rounded-card bg-surface-subtle p-2.5">
                  <div className="flex items-center gap-2">
                    <Badge tone="warning">Revisi {i + 1}</Badge>
                    <span className="text-caption text-fg-subtle">{formatRelative(r.at)}</span>
                  </div>
                  <p className="text-body text-fg">{r.instruction}</p>
                  {r.attachment_url && <AttachmentPreview url={r.attachment_url} name={r.attachment_name} align="start" />}
                </div>
              ))}
            </div>
          </div>
        )}
      </Modal>
    </Card>
  )
}
