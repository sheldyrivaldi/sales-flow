import { useState } from 'react'
import { Plus, Link as LinkIcon, Trash2, Eye } from 'lucide-react'

import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import Badge from '../../components/ui/Badge'
import Modal from '../../components/ui/Modal'
import Drawer from '../../components/ui/Drawer'
import Field from '../../components/ui/Field'
import Input from '../../components/ui/Input'
import Select from '../../components/ui/Select'
import StarRating from '../../components/ui/StarRating'
import EmptyState from '../../components/ui/EmptyState'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import Skeleton from '../../components/ui/Skeleton'
import { toast } from '../../lib/toast'
import { formatTanggal } from '../../lib/format'
import {
  useFeedbackRequests,
  useCreateFeedbackRequest,
  useDeleteFeedbackRequest,
} from '../../api/feedback'
import type { FeedbackRequest } from '../../api/feedback'
import { useProjects } from '../../api/projects'

function publicLink(token: string): string {
  return `${window.location.origin}/f/${token}`
}

/** Feedback Client (Pasca-Proyek): buat link feedback per proyek, bagikan ke
 * client, dan pantau mana yang sudah diisi. Form untuk client sengaja dibuat
 * singkat supaya tingkat pengisian tinggi. */
export default function PostFeedback() {
  const { data: requests, isLoading } = useFeedbackRequests()
  const { data: projectsData } = useProjects()
  const createMutation = useCreateFeedbackRequest()
  const deleteMutation = useDeleteFeedbackRequest()

  const [modalOpen, setModalOpen] = useState(false)
  const [selectedProjectId, setSelectedProjectId] = useState('')
  const [projectName, setProjectName] = useState('')
  const [clientName, setClientName] = useState('')
  const [viewTarget, setViewTarget] = useState<FeedbackRequest | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<FeedbackRequest | null>(null)

  const projects = projectsData?.items ?? []

  function openCreate() {
    setSelectedProjectId('')
    setProjectName('')
    setClientName('')
    setModalOpen(true)
  }

  function handlePickProject(id: string) {
    setSelectedProjectId(id)
    const p = projects.find((x) => x.id === id)
    if (p) {
      setProjectName(p.name)
      setClientName(p.client_name ?? '')
    }
  }

  async function handleCreate() {
    const name = projectName.trim()
    if (!name) {
      toast.error('Nama proyek wajib diisi.')
      return
    }
    try {
      const created = await createMutation.mutateAsync({
        project_name: name,
        client_name: clientName.trim() || undefined,
        project_id: selectedProjectId || undefined,
      })
      await navigator.clipboard.writeText(publicLink(created.token)).catch(() => {})
      toast.success('Link feedback dibuat dan disalin ke clipboard.')
      setModalOpen(false)
    } catch {
      toast.error('Gagal membuat link feedback.')
    }
  }

  function copyLink(req: FeedbackRequest) {
    navigator.clipboard.writeText(publicLink(req.token)).then(
      () => toast.success('Link feedback disalin.'),
      () => toast.error('Gagal menyalin link.'),
    )
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success('Link feedback dihapus.')
    } catch {
      toast.error('Gagal menghapus link.')
    } finally {
      setDeleteTarget(null)
    }
  }

  const columns: Column<FeedbackRequest>[] = [
    {
      key: 'project_name',
      header: 'Proyek',
      render: (row) => <span className="font-medium text-fg">{row.project_name}</span>,
    },
    {
      key: 'client_name',
      header: 'Client',
      render: (row) => <span className="text-fg-muted">{row.client_name ?? '—'}</span>,
    },
    {
      key: 'created_at',
      header: 'Dibuat',
      render: (row) => formatTanggal(row.created_at),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row) =>
        row.response ? (
          <div className="flex items-center gap-2">
            <Badge tone="success">Terisi</Badge>
            <StarRating value={row.response.overall_rating} />
          </div>
        ) : (
          <Badge tone="warning">Menunggu</Badge>
        ),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => (
        <div className="flex items-center justify-end gap-1">
          <Button
            size="sm"
            variant="ghost"
            leftIcon={<LinkIcon className="w-3.5 h-3.5" />}
            onClick={() => copyLink(row)}
          >
            Salin Link
          </Button>
          {row.response && (
            <Button
              size="sm"
              variant="ghost"
              leftIcon={<Eye className="w-3.5 h-3.5" />}
              onClick={() => setViewTarget(row)}
            >
              Lihat
            </Button>
          )}
          <button
            type="button"
            aria-label="Hapus link"
            onClick={() => setDeleteTarget(row)}
            className="p-1.5 rounded-btn text-fg-subtle hover:text-danger hover:bg-surface-subtle transition-colors"
          >
            <Trash2 className="w-3.5 h-3.5" aria-hidden="true" />
          </button>
        </div>
      ),
    },
  ]

  const resp = viewTarget?.response

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-h2 font-semibold text-fg">Feedback Client</h1>
          <p className="text-caption text-fg-muted mt-0.5">
            Buat link feedback singkat, bagikan ke client, dan pantau hasilnya.
          </p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
          Buat Link Feedback
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-64" />
      ) : !requests || requests.length === 0 ? (
        <EmptyState
          title="Belum ada permintaan feedback"
          description="Buat link feedback untuk proyek yang sudah selesai, lalu bagikan ke client."
          action={
            <Button size="sm" leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
              Buat Link Feedback
            </Button>
          }
        />
      ) : (
        <Table columns={columns} data={requests} rowKey={(row) => row.id} />
      )}

      {/* Create modal */}
      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title="Buat Link Feedback"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setModalOpen(false)}>
              Batal
            </Button>
            <Button loading={createMutation.isPending} onClick={() => void handleCreate()}>
              Buat &amp; Salin Link
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-4">
          {projects.length > 0 && (
            <Field label="Ambil dari Proyek Berjalan" helper="Opsional, mengisi nama proyek & client otomatis">
              <Select value={selectedProjectId} onChange={(e) => handlePickProject(e.target.value)}>
                <option value="">Pilih proyek…</option>
                {projects.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </Select>
            </Field>
          )}
          <Field label="Nama Proyek" required>
            <Input value={projectName} onChange={(e) => setProjectName(e.target.value)} />
          </Field>
          <Field label="Nama Client">
            <Input value={clientName} onChange={(e) => setClientName(e.target.value)} />
          </Field>
        </div>
      </Modal>

      {/* Response drawer */}
      <Drawer
        open={!!viewTarget}
        onClose={() => setViewTarget(null)}
        title={`Feedback: ${viewTarget?.project_name ?? ''}`}
        width="w-[420px]"
      >
        {resp && (
          <div className="flex flex-col gap-4 p-1">
            <div>
              <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Keseluruhan</p>
              <StarRating value={resp.overall_rating} size="lg" />
            </div>
            {(
              [
                ['Kualitas hasil', resp.quality_rating],
                ['Komunikasi', resp.communication_rating],
                ['Ketepatan waktu', resp.timeliness_rating],
              ] as [string, number | null][]
            ).map(
              ([label, v]) =>
                v != null && (
                  <div key={label} className="flex items-center justify-between">
                    <span className="text-body text-fg">{label}</span>
                    <StarRating value={v} />
                  </div>
                ),
            )}
            {resp.nps != null && (
              <div className="flex items-center justify-between">
                <span className="text-body text-fg">Skor rekomendasi (0-10)</span>
                <Badge tone={resp.nps >= 9 ? 'success' : resp.nps >= 7 ? 'warning' : 'danger'}>{resp.nps}</Badge>
              </div>
            )}
            {resp.comment && (
              <div>
                <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Komentar</p>
                <p className="text-body text-fg whitespace-pre-wrap">{resp.comment}</p>
              </div>
            )}
            <p className="text-caption text-fg-subtle">
              Diisi {resp.respondent_name ? `oleh ${resp.respondent_name} ` : ''}pada {formatTanggal(resp.created_at)}.
            </p>
          </div>
        )}
      </Drawer>

      <ConfirmDialog
        open={!!deleteTarget}
        title="Hapus link feedback?"
        description={`Link untuk proyek "${deleteTarget?.project_name}" akan berhenti berfungsi${deleteTarget?.response ? ' dan jawaban client ikut terhapus' : ''}.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteMutation.isPending}
        onConfirm={() => void handleDelete()}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
