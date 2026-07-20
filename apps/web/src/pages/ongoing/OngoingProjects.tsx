import { useState } from 'react'
import { Plus, Trash2, Pencil, CheckCircle2, Circle, SendHorizonal } from 'lucide-react'

import Table from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import Badge from '../../components/ui/Badge'
import Modal from '../../components/ui/Modal'
import Drawer from '../../components/ui/Drawer'
import Field from '../../components/ui/Field'
import Input from '../../components/ui/Input'
import Select from '../../components/ui/Select'
import Textarea from '../../components/ui/Textarea'
import EmptyState from '../../components/ui/EmptyState'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import Skeleton from '../../components/ui/Skeleton'
import { toast } from '../../lib/toast'
import { formatRupiahShort, formatTanggal, formatRelative } from '../../lib/format'
import { cn } from '../../lib/cn'
import {
  useProjects,
  useCreateProject,
  useUpdateProject,
  useDeleteProject,
  useAddProjectActivity,
  PROJECT_STATUS_LABELS,
} from '../../api/projects'
import type { Project, ProjectStatus, ProjectUpsertBody, ProjectMilestone } from '../../api/projects'

const STATUS_TONE: Record<ProjectStatus, 'success' | 'warning' | 'danger' | 'info'> = {
  ON_TRACK: 'success',
  AT_RISK: 'warning',
  DELAYED: 'danger',
  COMPLETED: 'info',
}

interface FormState {
  name: string
  clientName: string
  contractValue: string
  startDate: string
  endDate: string
  status: ProjectStatus
  progress: string
  description: string
}

const emptyForm: FormState = {
  name: '', clientName: '', contractValue: '', startDate: '', endDate: '',
  status: 'ON_TRACK', progress: '0', description: '',
}

function projectToForm(p: Project): FormState {
  return {
    name: p.name,
    clientName: p.client_name ?? '',
    contractValue: p.contract_value != null ? String(p.contract_value) : '',
    startDate: p.start_date?.slice(0, 10) ?? '',
    endDate: p.end_date?.slice(0, 10) ?? '',
    status: p.status,
    progress: String(p.progress),
    description: p.description ?? '',
  }
}

function formToBody(f: FormState): ProjectUpsertBody {
  return {
    name: f.name.trim(),
    client_name: f.clientName.trim() || null,
    contract_value: f.contractValue !== '' ? Number(f.contractValue) : null,
    start_date: f.startDate || null,
    end_date: f.endDate || null,
    status: f.status,
    progress: Math.max(0, Math.min(100, Number(f.progress) || 0)),
    description: f.description.trim() || null,
  }
}

function ProgressBar({ value }: { value: number }) {
  return (
    <div className="h-1.5 w-24 rounded-pill bg-surface-subtle overflow-hidden">
      <div
        className={cn(
          'h-full rounded-pill',
          value >= 70 ? 'bg-success' : value >= 40 ? 'bg-primary' : 'bg-warning'
        )}
        style={{ width: `${Math.min(value, 100)}%` }}
      />
    </div>
  )
}

/** Daftar Proyek Berjalan: CRUD proyek + drawer detail berisi milestone
 * (checklist) dan catatan aktivitas (check-in) per proyek. */
export default function OngoingProjects() {
  const { data, isLoading } = useProjects()
  const createMutation = useCreateProject()
  const updateMutation = useUpdateProject()
  const deleteMutation = useDeleteProject()
  const activityMutation = useAddProjectActivity()

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Project | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [detailId, setDetailId] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null)
  const [newMilestone, setNewMilestone] = useState('')
  const [newNote, setNewNote] = useState('')

  const items = data?.items ?? []
  const detail = items.find((p) => p.id === detailId) ?? null

  function openCreate() {
    setEditing(null)
    setForm(emptyForm)
    setModalOpen(true)
  }

  function openEdit(p: Project) {
    setEditing(p)
    setForm(projectToForm(p))
    setModalOpen(true)
  }

  async function handleSave() {
    if (!form.name.trim()) {
      toast.error('Nama proyek wajib diisi.')
      return
    }
    try {
      if (editing) {
        await updateMutation.mutateAsync({ id: editing.id, ...formToBody(form), milestones: editing.milestones })
        toast.success('Proyek diperbarui.')
      } else {
        await createMutation.mutateAsync(formToBody(form))
        toast.success('Proyek ditambahkan.')
      }
      setModalOpen(false)
    } catch {
      toast.error('Gagal menyimpan proyek.')
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      if (detailId === deleteTarget.id) setDetailId(null)
      toast.success('Proyek dihapus.')
    } catch {
      toast.error('Gagal menghapus proyek.')
    } finally {
      setDeleteTarget(null)
    }
  }

  async function toggleMilestone(p: Project, index: number) {
    const milestones: ProjectMilestone[] = p.milestones.map((m, i) =>
      i === index ? { ...m, done: !m.done } : m
    )
    try {
      await updateMutation.mutateAsync({ id: p.id, ...formToBody(projectToForm(p)), milestones })
    } catch {
      toast.error('Gagal memperbarui milestone.')
    }
  }

  async function addMilestone(p: Project) {
    const title = newMilestone.trim()
    if (!title) return
    const milestones = [...p.milestones, { title, done: false }]
    try {
      await updateMutation.mutateAsync({ id: p.id, ...formToBody(projectToForm(p)), milestones })
      setNewMilestone('')
    } catch {
      toast.error('Gagal menambah milestone.')
    }
  }

  async function addNote(p: Project) {
    const note = newNote.trim()
    if (!note) return
    try {
      await activityMutation.mutateAsync({ id: p.id, note })
      setNewNote('')
      toast.success('Catatan ditambahkan.')
    } catch {
      toast.error('Gagal menambah catatan.')
    }
  }

  const columns: Column<Project>[] = [
    {
      key: 'name',
      header: 'Proyek',
      render: (row) => (
        <button
          type="button"
          className="text-left font-medium text-primary hover:underline max-w-xs truncate block"
          onClick={() => setDetailId(row.id)}
        >
          {row.name}
        </button>
      ),
    },
    {
      key: 'client_name',
      header: 'Client',
      render: (row) => <span className="text-fg-muted">{row.client_name ?? '—'}</span>,
    },
    {
      key: 'contract_value',
      header: 'Nilai',
      align: 'right',
      render: (row) =>
        row.contract_value != null ? (
          <span className="tabular-nums">{formatRupiahShort(row.contract_value)}</span>
        ) : (
          <span className="text-fg-subtle">—</span>
        ),
    },
    {
      key: 'end_date',
      header: 'Target Selesai',
      render: (row) =>
        row.end_date ? formatTanggal(row.end_date) : <span className="text-fg-subtle">—</span>,
    },
    {
      key: 'progress',
      header: 'Progress',
      render: (row) => (
        <div className="flex items-center gap-2">
          <ProgressBar value={row.progress} />
          <span className="text-caption text-fg-muted tabular-nums">{row.progress}%</span>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row) => <Badge tone={STATUS_TONE[row.status]}>{PROJECT_STATUS_LABELS[row.status]}</Badge>,
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (row) => (
        <div className="flex items-center justify-end gap-1">
          <button
            type="button"
            aria-label="Edit proyek"
            onClick={(e) => {
              e.stopPropagation()
              openEdit(row)
            }}
            className="p-1.5 rounded-btn text-fg-subtle hover:text-fg hover:bg-surface-subtle transition-colors"
          >
            <Pencil className="w-3.5 h-3.5" aria-hidden="true" />
          </button>
          <button
            type="button"
            aria-label="Hapus proyek"
            onClick={(e) => {
              e.stopPropagation()
              setDeleteTarget(row)
            }}
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
          <h1 className="text-h2 font-semibold text-fg">Daftar Proyek</h1>
          <p className="text-caption text-fg-muted mt-0.5">
            Semua proyek berjalan beserta progress dan statusnya.
          </p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
          Proyek Baru
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-64" />
      ) : items.length === 0 ? (
        <EmptyState
          title="Belum ada proyek berjalan"
          description="Tambahkan proyek yang sudah dimenangkan untuk mulai memantau progress delivery."
          action={
            <Button size="sm" leftIcon={<Plus className="w-4 h-4" />} onClick={openCreate}>
              Proyek Baru
            </Button>
          }
        />
      ) : (
        <Table columns={columns} data={items} rowKey={(row) => row.id} />
      )}

      {/* Create/Edit modal */}
      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editing ? 'Edit Proyek' : 'Proyek Baru'}
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setModalOpen(false)}>
              Batal
            </Button>
            <Button
              loading={createMutation.isPending || updateMutation.isPending}
              onClick={() => void handleSave()}
            >
              Simpan
            </Button>
          </div>
        }
      >
        <div className="grid grid-cols-2 gap-4">
          <div className="col-span-2">
            <Field label="Nama Proyek" required>
              <Input value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
            </Field>
          </div>
          <Field label="Client">
            <Input value={form.clientName} onChange={(e) => setForm((f) => ({ ...f, clientName: e.target.value }))} />
          </Field>
          <Field label="Nilai Kontrak (Rp)">
            <Input
              type="number"
              min="0"
              value={form.contractValue}
              onChange={(e) => setForm((f) => ({ ...f, contractValue: e.target.value }))}
            />
          </Field>
          <Field label="Mulai">
            <Input type="date" value={form.startDate} onChange={(e) => setForm((f) => ({ ...f, startDate: e.target.value }))} />
          </Field>
          <Field label="Target Selesai">
            <Input type="date" value={form.endDate} onChange={(e) => setForm((f) => ({ ...f, endDate: e.target.value }))} />
          </Field>
          <Field label="Status">
            <Select
              value={form.status}
              onChange={(e) => setForm((f) => ({ ...f, status: e.target.value as ProjectStatus }))}
            >
              {(Object.keys(PROJECT_STATUS_LABELS) as ProjectStatus[]).map((st) => (
                <option key={st} value={st}>
                  {PROJECT_STATUS_LABELS[st]}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Progress (%)">
            <Input
              type="number"
              min="0"
              max="100"
              value={form.progress}
              onChange={(e) => setForm((f) => ({ ...f, progress: e.target.value }))}
            />
          </Field>
          <div className="col-span-2">
            <Field label="Deskripsi">
              <Textarea
                rows={3}
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              />
            </Field>
          </div>
        </div>
      </Modal>

      {/* Detail drawer */}
      <Drawer open={!!detail} onClose={() => setDetailId(null)} title={detail?.name ?? ''} width="w-[480px]">
        {detail && (
          <div className="flex flex-col gap-5 p-1">
            <div className="flex items-center gap-2 flex-wrap">
              <Badge tone={STATUS_TONE[detail.status]}>{PROJECT_STATUS_LABELS[detail.status]}</Badge>
              {detail.contract_value != null && (
                <span className="text-caption text-fg-muted">{formatRupiahShort(detail.contract_value)}</span>
              )}
              {detail.end_date && (
                <span className="text-caption text-fg-muted">target {formatTanggal(detail.end_date)}</span>
              )}
            </div>

            <div>
              <div className="flex items-center justify-between mb-1">
                <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Progress</p>
                <span className="text-caption text-fg tabular-nums">{detail.progress}%</span>
              </div>
              <div className="h-2 rounded-pill bg-surface-subtle overflow-hidden">
                <div className="h-full rounded-pill bg-primary" style={{ width: `${detail.progress}%` }} />
              </div>
            </div>

            {detail.description && <p className="text-body text-fg">{detail.description}</p>}

            {/* Milestones */}
            <div className="flex flex-col gap-2">
              <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Milestone</p>
              {detail.milestones.length === 0 && (
                <p className="text-caption text-fg-subtle">Belum ada milestone.</p>
              )}
              {detail.milestones.map((m, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => void toggleMilestone(detail, i)}
                  className="flex items-center gap-2 text-left text-body hover:bg-surface-subtle rounded-btn px-2 py-1 -mx-2 transition-colors"
                >
                  {m.done ? (
                    <CheckCircle2 className="w-4 h-4 text-success shrink-0" aria-hidden="true" />
                  ) : (
                    <Circle className="w-4 h-4 text-line-strong shrink-0" aria-hidden="true" />
                  )}
                  <span className={cn('flex-1', m.done && 'line-through text-fg-subtle')}>{m.title}</span>
                </button>
              ))}
              <div className="flex gap-2">
                <Input
                  value={newMilestone}
                  onChange={(e) => setNewMilestone(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      void addMilestone(detail)
                    }
                  }}
                  placeholder="Tambah milestone…"
                />
                <Button size="sm" variant="secondary" onClick={() => void addMilestone(detail)}>
                  Tambah
                </Button>
              </div>
            </div>

            {/* Catatan aktivitas */}
            <div className="flex flex-col gap-2">
              <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Catatan Aktivitas</p>
              <div className="flex gap-2">
                <Input
                  value={newNote}
                  onChange={(e) => setNewNote(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      void addNote(detail)
                    }
                  }}
                  placeholder="mis. kickoff meeting dengan client…"
                />
                <Button
                  size="sm"
                  leftIcon={<SendHorizonal className="w-3.5 h-3.5" />}
                  loading={activityMutation.isPending}
                  onClick={() => void addNote(detail)}
                >
                  Catat
                </Button>
              </div>
              {detail.activities.length === 0 ? (
                <p className="text-caption text-fg-subtle">Belum ada catatan.</p>
              ) : (
                <ol className="relative flex flex-col gap-2.5 pl-5 border-l-2 border-line ml-1.5 mt-1">
                  {detail.activities.map((a, i) => (
                    <li key={i} className="relative">
                      <span
                        className="absolute -left-[1.44rem] top-1.5 h-2.5 w-2.5 rounded-full bg-primary ring-4 ring-primary-subtle"
                        aria-hidden="true"
                      />
                      <p className="text-body text-fg">{a.note}</p>
                      <p className="text-caption text-fg-subtle">{formatRelative(a.date)}</p>
                    </li>
                  ))}
                </ol>
              )}
            </div>
          </div>
        )}
      </Drawer>

      <ConfirmDialog
        open={!!deleteTarget}
        title="Hapus proyek?"
        description={`Proyek "${deleteTarget?.name}" beserta milestone dan catatannya akan dihapus permanen.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={deleteMutation.isPending}
        onConfirm={() => void handleDelete()}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
