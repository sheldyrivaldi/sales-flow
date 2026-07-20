import { useState, useRef } from 'react'
import { Plus, Download, SendHorizonal, Sparkles, Trash2, Upload, X, Loader2, AlertCircle, Eye, RotateCcw } from 'lucide-react'

import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import Badge from '../../components/ui/Badge'
import Modal from '../../components/ui/Modal'
import Input from '../../components/ui/Input'
import Textarea from '../../components/ui/Textarea'
import EmptyState from '../../components/ui/EmptyState'
import Skeleton from '../../components/ui/Skeleton'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import PlaybookSlideViewer from '../../components/playbooks/PlaybookSlideViewer'
import AttachmentPreview from '../../components/chat/AttachmentPreview'
import { formatRelative } from '../../lib/format'
import { toast } from '../../lib/toast'
import { exportPlaybookPpt } from '../../lib/exportPlaybookPpt'
import {
  usePlaybookJobs,
  useCreatePlaybookJob,
  useRefinePlaybookJob,
  useRetryPlaybookJob,
  useDeletePlaybookJob,
  PLAYBOOK_STATUS_LABEL,
  isJobActive,
} from '../../api/playbookJobs'
import type { PlaybookJob, PlaybookJobStatus } from '../../api/playbookJobs'

const STATUS_TONE: Record<PlaybookJobStatus, 'info' | 'warning' | 'success' | 'danger'> = {
  in_progress: 'info',
  updating: 'warning',
  success: 'success',
  failed: 'danger',
}

function StatusBadge({ status }: { status: PlaybookJobStatus }) {
  const active = isJobActive(status)
  return (
    <Badge tone={STATUS_TONE[status]}>
      <span className="inline-flex items-center gap-1">
        {active && <Loader2 className="w-3 h-3 animate-spin" aria-hidden="true" />}
        {PLAYBOOK_STATUS_LABEL[status]}
      </span>
    </Badge>
  )
}

const MAX_MB = 10

/** Menu Playbooks: riwayat generate playbook async. Tombol Buat membuka modal
 * (prompt + lampiran), generate langsung masuk list berstatus Diproses,
 * berubah otomatis (polling) menjadi Selesai/Gagal. Buka detail untuk lihat
 * isi, unduh PPT, atau revisi via prompt (status Merevisi). */
export default function PlaybooksIndex() {
  const { data: jobs, isLoading } = usePlaybookJobs()
  const createMutation = useCreatePlaybookJob()
  const refineMutation = useRefinePlaybookJob()
  const retryMutation = useRetryPlaybookJob()
  const deleteMutation = useDeletePlaybookJob()

  const [createOpen, setCreateOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [prompt, setPrompt] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  // detail.mode: 'info' = klik judul (read-only: judul + prompt + riwayat),
  // 'view' = tombol Buka (preview slide + download + revisi).
  const [detail, setDetail] = useState<{ id: string; mode: 'info' | 'view' } | null>(null)
  const [instruction, setInstruction] = useState('')
  const [refineFile, setRefineFile] = useState<File | null>(null)
  const refineFileRef = useRef<HTMLInputElement>(null)
  const [deleteTarget, setDeleteTarget] = useState<PlaybookJob | null>(null)

  const items = jobs ?? []
  const current = items.find((j) => j.id === detail?.id) ?? null

  async function handleRetry(job: PlaybookJob) {
    try {
      await retryMutation.mutateAsync(job.id)
      toast.success('Playbook dicoba ulang, statusnya akan berubah otomatis.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal mencoba ulang.')
    }
  }

  async function handleCreate() {
    if (!prompt.trim() && !file) {
      toast.error('Isi prompt atau lampirkan dokumen dulu.')
      return
    }
    try {
      await createMutation.mutateAsync({ title: title.trim(), prompt: prompt.trim(), file })
      toast.success('Playbook sedang diproses, statusnya akan berubah otomatis.')
      setCreateOpen(false)
      setTitle('')
      setPrompt('')
      setFile(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal memulai generate playbook.')
    }
  }

  function pickFile(f: File | undefined) {
    if (!f) return
    if (f.size > MAX_MB * 1024 * 1024) {
      toast.error(`Ukuran lampiran maksimal ${MAX_MB} MB.`)
      return
    }
    setFile(f)
  }

  async function handleRefine() {
    if (!current) return
    const text = instruction.trim()
    if (!text) return
    try {
      await refineMutation.mutateAsync({ id: current.id, instruction: text, file: refineFile })
      toast.success('Revisi playbook sedang diproses, hasilnya diperbarui otomatis.')
      setInstruction('')
      setRefineFile(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal memulai revisi.')
    }
  }

  function pickRefineFile(f: File | undefined) {
    if (!f) return
    if (f.size > MAX_MB * 1024 * 1024) {
      toast.error(`Ukuran lampiran maksimal ${MAX_MB} MB.`)
      return
    }
    setRefineFile(f)
  }

  /** Arahkan prompt revisi ke seksi slide tertentu (dari tombol "Edit slide ini"). */
  function editSection(section: string) {
    setInstruction((prev) => (prev.trim() ? prev : `Perbaiki bagian ${section}: `))
    document.getElementById('pb-refine')?.focus()
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      if (detail?.id === deleteTarget.id) setDetail(null)
      toast.success('Playbook dihapus.')
    } catch {
      toast.error('Gagal menghapus playbook.')
    } finally {
      setDeleteTarget(null)
    }
  }

  /** Judul yang diketik user menang di mana pun — tabel, slide cover, dan
   * nama file — jadi judul karangan AI tidak boleh menggesernya. */
  function effectiveContent(job: PlaybookJob) {
    if (!job.content) return undefined
    if (!job.user_titled) return job.content
    return { ...job.content, title: job.title }
  }

  function openPpt(job: PlaybookJob) {
    const content = effectiveContent(job)
    if (!content) return
    void exportPlaybookPpt(content, job.title).catch(() => toast.error('Ekspor PPT gagal.'))
  }

  const columns: Column<PlaybookJob>[] = [
    {
      key: 'title',
      header: 'Judul',
      render: (row) => (
        <button
          type="button"
          className="text-left font-medium text-primary hover:underline max-w-sm truncate block"
          onClick={() => setDetail({ id: row.id, mode: 'info' })}
        >
          {row.title}
        </button>
      ),
    },
    {
      key: 'source',
      header: 'Sumber',
      render: (row) => (
        <Badge tone={row.source === 'event' ? 'accent' : 'info'}>
          {row.source === 'event' ? 'Event' : 'Custom'}
        </Badge>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row) => <StatusBadge status={row.status} />,
    },
    {
      key: 'error_message',
      header: 'Keterangan',
      render: (row) =>
        row.status === 'failed' && row.error_message ? (
          <span className="inline-flex items-center gap-1 text-caption text-danger max-w-xs">
            <AlertCircle className="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
            <span className="truncate" title={row.error_message}>{row.error_message}</span>
          </span>
        ) : (
          <span className="text-fg-subtle">—</span>
        ),
    },
    {
      key: 'created_at',
      header: 'Dibuat',
      render: (row) => <span className="text-fg-muted">{formatRelative(row.created_at)}</span>,
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => (
        <div className="flex items-center justify-end gap-1">
          {row.status === 'success' && (
            <Button size="sm" variant="ghost" leftIcon={<Eye className="w-3.5 h-3.5" />} onClick={() => setDetail({ id: row.id, mode: 'view' })}>
              Buka
            </Button>
          )}
          {row.status === 'failed' && (
            <Button
              size="sm"
              variant="ghost"
              leftIcon={<RotateCcw className="w-3.5 h-3.5" />}
              loading={retryMutation.isPending}
              onClick={() => void handleRetry(row)}
            >
              Retry
            </Button>
          )}
          <button
            type="button"
            aria-label="Hapus playbook"
            onClick={() => setDeleteTarget(row)}
            className="p-1.5 rounded-btn text-fg-subtle hover:text-danger hover:bg-surface-subtle transition-colors"
          >
            <Trash2 className="w-3.5 h-3.5" aria-hidden="true" />
          </button>
        </div>
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-h2 font-semibold text-fg">Playbooks</h1>
          <p className="text-caption text-fg-muted mt-0.5">
            Generate playbook dari prompt atau dokumen. Hasil berupa PPT yang bisa dibuka dan direvisi.
          </p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={() => setCreateOpen(true)}>
          Buat Playbook
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-64" />
      ) : items.length === 0 ? (
        <EmptyState
          icon={<Sparkles className="w-6 h-6" />}
          title="Belum ada playbook"
          description="Klik Buat Playbook untuk generate playbook pertama dari prompt atau dokumen."
          action={
            <Button size="sm" leftIcon={<Plus className="w-4 h-4" />} onClick={() => setCreateOpen(true)}>
              Buat Playbook
            </Button>
          }
        />
      ) : (
        <Table columns={columns} data={items} rowKey={(row) => row.id} />
      )}

      {/* Create modal — prompt + attachment (seperti chat AI) */}
      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Buat Playbook"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setCreateOpen(false)}>
              Batal
            </Button>
            <Button
              loading={createMutation.isPending}
              leftIcon={<Sparkles className="w-4 h-4" />}
              onClick={() => void handleCreate()}
            >
              Generate
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <label htmlFor="pb-title" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
              Judul playbook
            </label>
            <Input
              id="pb-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="mis. Strategi Masuk Sektor Kesehatan 2026"
            />
            <p className="text-caption text-fg-subtle">
              Judul dipakai apa adanya untuk nama file dan slide cover. Kosongkan bila ingin AI yang menyusunnya.
            </p>
          </div>

          <div className="flex flex-col gap-1.5">
            <label htmlFor="pb-prompt" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
              Prompt playbook
            </label>
            <Textarea
              id="pb-prompt"
              rows={4}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="mis. strategi masuk sektor kesehatan untuk produk absensi cloud, target BUMN…"
            />
          </div>

          <input
            ref={fileRef}
            type="file"
            accept=".pdf"
            className="sr-only"
            tabIndex={-1}
            aria-hidden="true"
            onChange={(e) => {
              pickFile(e.target.files?.[0])
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
            <Button
              variant="secondary"
              size="sm"
              className="self-start"
              leftIcon={<Upload className="w-3.5 h-3.5" />}
              onClick={() => fileRef.current?.click()}
            >
              Lampirkan Dokumen (PDF)
            </Button>
          )}
        </div>
      </Modal>

      {/* Modal detail — mode 'info' (klik judul, read-only) atau 'view'
          (tombol Buka, preview slide + download + revisi). */}
      <Modal open={!!detail} onClose={() => setDetail(null)} title={current?.title ?? 'Playbook'} size="lg">
        {current && (
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-2 flex-wrap">
              <StatusBadge status={current.status} />
              <Badge tone={current.source === 'event' ? 'accent' : 'info'}>
                {current.source === 'event' ? 'Event' : 'Custom'}
              </Badge>
              <span className="text-caption text-fg-subtle">{formatRelative(current.updated_at)}</span>
              {detail?.mode === 'view' && current.status === 'success' && current.content && (
                <Button
                  size="sm"
                  variant="secondary"
                  className="ml-auto"
                  leftIcon={<Download className="w-3.5 h-3.5" />}
                  onClick={() => openPpt(current)}
                >
                  Download PPT
                </Button>
              )}
            </div>

            {/* Status non-final juga tampil di kedua mode */}
            {isJobActive(current.status) && (
              <div className="flex items-center gap-2 text-body text-fg-muted py-6 justify-center">
                <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
                {current.status === 'updating' ? 'Sedang merevisi playbook…' : 'Sedang menyusun playbook…'} Kamu boleh tinggal halaman ini.
              </div>
            )}
            {current.status === 'failed' && (
              <div className="flex items-start gap-2 rounded-card border border-danger-border bg-danger-subtle p-3 text-body text-danger">
                <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" aria-hidden="true" />
                <span>{current.error_message ?? 'Generate gagal tanpa keterangan.'}</span>
              </div>
            )}

            {/* Mode VIEW: preview slide + revisi (hanya jika sukses) */}
            {detail?.mode === 'view' && current.status === 'success' && current.content && (
              <>
                <PlaybookSlideViewer content={effectiveContent(current)!} fallbackTitle={current.title} onEditSection={editSection} />

                <div className="flex flex-col gap-2 pt-3 border-t border-line">
                  <label htmlFor="pb-refine" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
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
                        pickRefineFile(e.target.files?.[0])
                        e.target.value = ''
                      }}
                    />
                    <Button
                      variant="secondary"
                      size="md"
                      aria-label="Lampirkan dokumen"
                      onClick={() => refineFileRef.current?.click()}
                      leftIcon={<Upload className="w-4 h-4" />}
                    >
                      Lampiran
                    </Button>
                    <Input
                      id="pb-refine"
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
                    <Button
                      loading={refineMutation.isPending}
                      leftIcon={<SendHorizonal className="w-3.5 h-3.5" />}
                      onClick={() => void handleRefine()}
                    >
                      Revisi
                    </Button>
                  </div>
                </div>
              </>
            )}

            {/* Prompt + riwayat revisi — read-only, tampil di kedua mode */}
            <div className="flex flex-col gap-2 pt-3 border-t border-line">
              <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Prompt &amp; Riwayat Revisi</p>
              <div className="flex flex-col gap-1.5">
                <p className="text-body text-fg whitespace-pre-wrap">{current.prompt}</p>
                {current.attachment_url && (
                  <AttachmentPreview url={current.attachment_url} name={current.attachment_name} align="start" />
                )}
              </div>
              {(current.revisions ?? []).map((r, i) => (
                <div key={i} className="flex flex-col gap-1.5 rounded-card bg-surface-subtle p-2.5">
                  <div className="flex items-center gap-2">
                    <Badge tone="warning">Revisi {i + 1}</Badge>
                    <span className="text-caption text-fg-subtle">{formatRelative(r.at)}</span>
                  </div>
                  <p className="text-body text-fg">{r.instruction}</p>
                  {r.attachment_url && (
                    <AttachmentPreview url={r.attachment_url} name={r.attachment_name} align="start" />
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        title="Hapus playbook?"
        description={`Playbook "${deleteTarget?.title}" akan dihapus permanen.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteMutation.isPending}
        onConfirm={() => void handleDelete()}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
