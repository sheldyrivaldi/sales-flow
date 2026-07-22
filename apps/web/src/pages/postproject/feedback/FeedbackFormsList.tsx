import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Plus, ExternalLink, FileText } from 'lucide-react'

import Table from '../../../components/ui/Table'
import type { Column, KebabAction } from '../../../components/ui/Table'
import Button from '../../../components/ui/Button'
import Badge from '../../../components/ui/Badge'
import type { Tone } from '../../../lib/score'
import EmptyState from '../../../components/ui/EmptyState'
import ConfirmDialog from '../../../components/ui/ConfirmDialog'
import Skeleton from '../../../components/ui/Skeleton'
import { toast } from '../../../lib/toast'
import { formatTanggal } from '../../../lib/format'
import {
  useFeedbackForms,
  useDeleteFeedbackForm,
  useCreateFeedbackForm,
  publicFormLink,
} from '../../../api/feedbackForms'
import type { FeedbackForm, FeedbackFormStatus, FormLanguage } from '../../../api/feedbackForms'

const STATUS: Record<FeedbackFormStatus, { label: string; tone: Tone; solid?: boolean }> = {
  draft: { label: 'Draft', tone: 'info' },
  published: { label: 'Terbit', tone: 'success', solid: true },
  closed: { label: 'Ditutup', tone: 'warning' },
}

const LANG: Record<FormLanguage, string> = {
  id: 'Indonesia',
  en: 'English',
}

/** Feedback Client — daftar form kuesioner dinamis. Buat form (manual atau
 * dengan bantuan AI di dalam builder), terbitkan link publik, dan pantau
 * respon. Klik judul untuk membuka halaman detail. */
export default function FeedbackFormsList() {
  const navigate = useNavigate()
  const { data: forms, isLoading } = useFeedbackForms()
  const createMutation = useCreateFeedbackForm()
  const deleteMutation = useDeleteFeedbackForm()
  const [deleteTarget, setDeleteTarget] = useState<FeedbackForm | null>(null)

  function copyLink(form: FeedbackForm) {
    navigator.clipboard.writeText(publicFormLink(form.slug)).then(
      () => toast.success('Link form disalin.'),
      () => toast.error('Gagal menyalin link.'),
    )
  }

  async function handleDuplicate(form: FeedbackForm) {
    try {
      const copy = await createMutation.mutateAsync({
        title: `${form.title} (salinan)`,
        description: form.description,
        language: form.language,
        collect_email: form.collect_email,
        // Kosongkan ID agar service men-generate ID baru per pertanyaan.
        questions: form.questions.map((q) => ({ ...q, id: '' })),
      })
      toast.success('Form diduplikasi sebagai draft baru.')
      navigate(`/postproject/feedback/${copy.id}/edit`)
    } catch {
      toast.error('Gagal menduplikasi form.')
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success('Form dihapus.')
    } catch {
      toast.error('Gagal menghapus form.')
    } finally {
      setDeleteTarget(null)
    }
  }

  const columns: Column<FeedbackForm>[] = [
    {
      key: 'title',
      header: 'Form',
      width: '34%',
      render: (row) => (
        <div className="flex flex-col gap-0.5 min-w-0">
          <button
            type="button"
            onClick={() => navigate(`/postproject/feedback/${row.id}`)}
            className="font-medium text-fg hover:text-primary text-left break-words"
          >
            {row.title}
          </button>
          {row.description && (
            <span className="text-caption text-fg-subtle line-clamp-1">{row.description}</span>
          )}
          {row.status === 'published' ? (
            <a
              href={publicFormLink(row.slug)}
              target="_blank"
              rel="noreferrer"
              onClick={(e) => e.stopPropagation()}
              className="inline-flex items-center gap-1 text-caption text-primary hover:underline w-fit"
            >
              <ExternalLink className="w-3 h-3" aria-hidden="true" /> /form/{row.slug}
            </a>
          ) : (
            <span className="text-caption text-fg-subtle">/form/{row.slug}</span>
          )}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row) => (
        <Badge tone={STATUS[row.status].tone} appearance={STATUS[row.status].solid ? 'solid' : 'soft'}>
          {STATUS[row.status].label}
        </Badge>
      ),
    },
    {
      key: 'language',
      header: 'Bahasa',
      render: (row) => <span className="text-fg-muted">{LANG[row.language] ?? row.language}</span>,
    },
    {
      key: 'questions',
      header: 'Pertanyaan',
      align: 'center',
      render: (row) => <span className="tabular-nums text-fg-muted">{row.questions.length}</span>,
    },
    {
      key: 'submission_count',
      header: 'Respon',
      align: 'center',
      render: (row) => <span className="tabular-nums font-medium text-fg">{row.submission_count}</span>,
    },
    {
      key: 'created_by',
      header: 'Dibuat oleh',
      render: (row) => (
        <div className="flex flex-col">
          <span className="text-fg-muted">{row.created_by_name ?? '—'}</span>
          <span className="text-caption text-fg-subtle">{formatTanggal(row.created_at)}</span>
        </div>
      ),
    },
  ]

  const kebabActions = (row: FeedbackForm): KebabAction[] => {
    const actions: KebabAction[] = [
      { label: 'Buka detail', onClick: () => navigate(`/postproject/feedback/${row.id}`) },
      { label: 'Edit', onClick: () => navigate(`/postproject/feedback/${row.id}/edit`) },
    ]
    if (row.status === 'published') {
      actions.push({ label: 'Salin link publik', onClick: () => copyLink(row) })
    }
    actions.push({ label: 'Duplikat', onClick: () => void handleDuplicate(row) })
    actions.push({ label: 'Hapus', onClick: () => setDeleteTarget(row), danger: true })
    return actions
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-h2 font-semibold text-fg">Feedback Client</h1>
          <p className="text-caption text-fg-muted mt-0.5">
            Susun kuesioner sendiri (rating, teks, pilihan) atau dengan bantuan AI, terbitkan link publik, dan kumpulkan jawaban client.
          </p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={() => navigate('/postproject/feedback/new')}>
          Buat Form
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-64" />
      ) : !forms || forms.length === 0 ? (
        <EmptyState
          icon={<FileText className="w-6 h-6" />}
          title="Belum ada form feedback"
          description="Buat form kuesioner untuk client — mulai dari nol atau minta bantuan AI menyusun pertanyaannya di dalam builder."
          action={
            <Button size="sm" leftIcon={<Plus className="w-4 h-4" />} onClick={() => navigate('/postproject/feedback/new')}>
              Buat Form
            </Button>
          }
        />
      ) : (
        <Table columns={columns} data={forms} rowKey={(row) => row.id} kebabActions={kebabActions} />
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        title="Hapus form feedback?"
        description={`Form "${deleteTarget?.title}" beserta ${deleteTarget?.submission_count ?? 0} jawaban akan dihapus permanen. Link publiknya berhenti berfungsi.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteMutation.isPending}
        onConfirm={() => void handleDelete()}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
