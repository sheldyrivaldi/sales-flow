import Card, { CardBody } from '../../components/ui/Card'
import Skeleton from '../../components/ui/Skeleton'
import EmptyState from '../../components/ui/EmptyState'
import { useProfile } from '../../api/profile'
import { formatRelative } from '../../lib/format'

export default function WorkspaceTab() {
  const { data: profile, isLoading } = useProfile()

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

  if (!profile) {
    return (
      <EmptyState
        title="Profil perusahaan belum diisi"
        description="Isi Otak Agent untuk melengkapi data workspace."
      />
    )
  }

  return (
    <Card className="max-w-md">
      <CardBody className="flex flex-col gap-4">
        <div>
          <p className="text-caption text-fg-muted">Nama Perusahaan</p>
          <p className="text-body font-medium text-fg">{profile.company_name}</p>
        </div>
        {profile.one_liner && (
          <div>
            <p className="text-caption text-fg-muted">Deskripsi Singkat</p>
            <p className="text-body text-fg">{profile.one_liner}</p>
          </div>
        )}
        <div>
          <p className="text-caption text-fg-muted">Versi Profil</p>
          <p className="text-body text-fg">
            v{profile.version} · diperbarui {formatRelative(profile.updated_at)}
          </p>
        </div>
        <p className="text-caption text-fg-subtle">
          SalesPilot adalah workspace tunggal (satu perusahaan, internal) — tidak ada multi-tenant.
        </p>
      </CardBody>
    </Card>
  )
}
