import { useState } from 'react'
import { Sparkles, AlertTriangle } from 'lucide-react'
import Card, { CardBody } from '../../components/ui/Card'
import Badge from '../../components/ui/Badge'
import Button from '../../components/ui/Button'
import Skeleton from '../../components/ui/Skeleton'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import { toast } from '../../lib/toast'
import { useAuthStore } from '../../store/auth'
import { can } from '../../lib/rbac'
import { useHermesStatus, useTestHermes, useResetHermesMemory } from '../../api/settings'

export default function AiHermesTab() {
  const role = useAuthStore((s) => s.user?.role)
  const isAdmin = can(role, 'ManageUsers')

  const { data: status, isLoading } = useHermesStatus()
  const testHermes = useTestHermes()
  const resetMemory = useResetHermesMemory()
  const [confirmReset, setConfirmReset] = useState(false)

  async function handleTest() {
    try {
      const res = await testHermes.mutateAsync()
      if (res.status === 'ok') {
        toast.success(`Koneksi berhasil${res.version ? ` • v${res.version}` : ''}.`)
      } else {
        toast.error('Koneksi gagal. Periksa konfigurasi Hermes.')
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Test koneksi gagal, coba lagi nanti.')
    }
  }

  async function handleResetMemory() {
    try {
      await resetMemory.mutateAsync()
      toast.success('Memory Hermes berhasil di-reset.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Reset memory gagal, coba lagi nanti.')
    } finally {
      setConfirmReset(false)
    }
  }

  if (isLoading) {
    return (
      <Card className="max-w-md">
        <CardBody className="flex flex-col gap-3">
          <Skeleton variant="text" className="h-5 w-1/2" />
          <Skeleton variant="text" className="h-4 w-3/4" />
        </CardBody>
      </Card>
    )
  }

  const connected = status?.status === 'connected'

  return (
    <Card className="max-w-md">
      <CardBody className="flex flex-col gap-4">
        <div>
          <p className="text-caption text-fg-muted mb-1">Status Koneksi</p>
          {connected ? (
            <Badge tone="success">Connected{status?.version ? ` • v${status.version}` : ''}</Badge>
          ) : (
            <div className="flex flex-col gap-1">
              <Badge tone="danger">Terputus</Badge>
              <p className="flex items-center gap-1 text-caption text-fg-muted">
                <AlertTriangle className="w-3.5 h-3.5 text-danger" aria-hidden="true" />
                Asisten AI sedang tidak tersedia. Periksa konfigurasi provider di tab ini (admin) atau
                coba lagi nanti.
              </p>
            </div>
          )}
        </div>

        <div>
          <p className="text-caption text-fg-muted mb-1">Memory</p>
          <p className="flex items-center gap-1.5 text-body text-fg">
            <Sparkles className="w-4 h-4 text-accent" aria-hidden="true" />
            {status?.memory_active ? 'Pembelajaran aktif' : 'Pembelajaran belum aktif'}
          </p>
        </div>

        <div className="flex gap-2">
          <Button variant="secondary" loading={testHermes.isPending} onClick={handleTest}>
            Test Koneksi
          </Button>
          {isAdmin && (
            <Button variant="danger" onClick={() => setConfirmReset(true)}>
              Reset Memory
            </Button>
          )}
        </div>
      </CardBody>

      <ConfirmDialog
        open={confirmReset}
        onCancel={() => setConfirmReset(false)}
        onConfirm={handleResetMemory}
        title="Reset memory Hermes?"
        description="Seluruh riwayat pembelajaran AI (WON/LOST, alasan Tolak, konteks percakapan) akan dihapus dan tidak bisa dikembalikan."
        tone="danger"
        confirmLabel="Reset"
        loading={resetMemory.isPending}
      />
    </Card>
  )
}
