import { useState } from 'react'
import { ExternalLink, Plus } from 'lucide-react'
import Card, { CardBody } from '../ui/Card'
import Badge from '../ui/Badge'
import Button from '../ui/Button'
import Toggle from '../ui/Toggle'
import Table from '../ui/Table'
import type { Column, KebabAction } from '../ui/Table'
import ConfirmDialog from '../ui/ConfirmDialog'
import Tooltip from '../ui/Tooltip'
import { toast } from '../../lib/toast'
import type { Tone } from '../../lib/score'
import {
  useSources,
  useSourcePresets,
  useUpdateSource,
  useDeleteSource,
  useActivatePreset,
  ACCESS_LABELS,
  FREQUENCY_LABELS,
  priorityLabel,
} from '../../api/sources'
import type { Source, SourceAccess } from '../../api/sources'
import SourceFormModal from './SourceFormModal'

const ACCESS_TONE: Record<SourceAccess, Tone> = {
  publik: 'success',
  login: 'warning',
  manual: 'warning',
}

export interface SourcesTabProps {
  canEdit: boolean
}

export default function SourcesTab({ canEdit }: SourcesTabProps) {
  const { data, isLoading } = useSources({ page_size: 100 })
  const { data: presets } = useSourcePresets()
  const activatePreset = useActivatePreset()
  const updateSource = useUpdateSource()
  const deleteSource = useDeleteSource()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Source | undefined>()
  const [deleteTarget, setDeleteTarget] = useState<Source | undefined>()

  async function handleActivate(key: string) {
    try {
      await activatePreset.mutateAsync(key)
      toast.success('Sumber diaktifkan.')
    } catch {
      toast.error('Gagal mengaktifkan sumber.')
    }
  }

  async function handleToggleEnabled(source: Source, enabled: boolean) {
    try {
      await updateSource.mutateAsync({ id: source.id, body: { enabled } })
    } catch {
      toast.error('Gagal mengubah status sumber.')
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteSource.mutateAsync(deleteTarget.id)
      toast.success('Sumber dihapus.')
      setDeleteTarget(undefined)
    } catch {
      toast.error('Gagal menghapus sumber.')
    }
  }

  const columns: Column<Source>[] = [
    { key: 'name', header: 'Nama' },
    {
      key: 'url',
      header: 'URL',
      render: (s) => (
        <a
          href={s.url}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1 text-primary hover:underline"
        >
          {s.url}
          <ExternalLink className="w-3 h-3" aria-hidden="true" />
        </a>
      ),
    },
    { key: 'country', header: 'Negara', render: (s) => s.country ?? '—' },
    {
      key: 'access',
      header: 'Akses',
      render: (s) => <Badge tone={ACCESS_TONE[s.access]}>{ACCESS_LABELS[s.access]}</Badge>,
    },
    { key: 'legal_note', header: 'Legal note', render: (s) => s.legal_note ?? '—' },
    { key: 'priority', header: 'Prioritas', render: (s) => priorityLabel(s.priority) },
    { key: 'frequency', header: 'Frekuensi', render: (s) => FREQUENCY_LABELS[s.frequency] },
    {
      key: 'enabled',
      header: 'Aktif',
      render: (s) => (
        <Toggle
          checked={s.enabled}
          onChange={(checked) => handleToggleEnabled(s, checked)}
          disabled={!canEdit}
          size="sm"
        />
      ),
    },
  ]

  function kebabActions(source: Source): KebabAction[] {
    if (!canEdit) return []
    return [
      {
        label: 'Edit',
        onClick: () => {
          setEditing(source)
          setFormOpen(true)
        },
      },
      { label: 'Hapus', onClick: () => setDeleteTarget(source), danger: true },
    ]
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardBody className="flex flex-col gap-3">
          <h3 className="text-body font-semibold text-fg">Preset sumber (Indonesia)</h3>
          <div className="flex flex-wrap gap-2">
            {presets?.map((p) => (
              <Tooltip key={p.key} content={p.legal_note}>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={p.activated || !canEdit}
                  loading={activatePreset.isPending}
                  onClick={() => handleActivate(p.key)}
                >
                  {p.activated ? `✓ ${p.name}` : `Aktifkan ${p.name}`}
                </Button>
              </Tooltip>
            ))}
          </div>
        </CardBody>
      </Card>

      <div className="flex items-center justify-between">
        <h3 className="text-body font-semibold text-fg">Daftar sumber</h3>
        {canEdit && (
          <Button
            size="sm"
            leftIcon={<Plus className="w-4 h-4" />}
            onClick={() => {
              setEditing(undefined)
              setFormOpen(true)
            }}
          >
            Tambah sumber
          </Button>
        )}
      </div>

      <Table
        columns={columns}
        data={data?.items ?? []}
        rowKey={(s) => s.id}
        kebabActions={canEdit ? kebabActions : undefined}
        loading={isLoading}
        empty="Belum ada sumber."
      />

      <SourceFormModal open={formOpen} onClose={() => setFormOpen(false)} source={editing} />

      <ConfirmDialog
        open={!!deleteTarget}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(undefined)}
        title="Hapus sumber?"
        description={`Sumber "${deleteTarget?.name}" akan dihapus permanen.`}
        loading={deleteSource.isPending}
      />
    </div>
  )
}
