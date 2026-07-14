import Card, { CardBody } from '../../components/ui/Card'
import Badge from '../../components/ui/Badge'
import { useAuthStore } from '../../store/auth'

const ROLE_LABELS: Record<string, string> = {
  SALES: 'Sales',
  OPS: 'Operations',
  MANAGER: 'Manager',
  ADMIN: 'Admin',
}

export default function ProfileTab() {
  const user = useAuthStore((s) => s.user)

  if (!user) return null

  return (
    <Card className="max-w-md">
      <CardBody className="flex flex-col gap-4">
        <div>
          <p className="text-caption text-fg-muted">Nama</p>
          <p className="text-body font-medium text-fg">{user.name}</p>
        </div>
        <div>
          <p className="text-caption text-fg-muted">Email</p>
          <p className="text-body font-medium text-fg">{user.email}</p>
        </div>
        <div>
          <p className="text-caption text-fg-muted">Role</p>
          <Badge tone="info">{ROLE_LABELS[user.role] ?? user.role}</Badge>
        </div>
        <p className="text-caption text-fg-subtle">
          Akun dikelola oleh Admin — hubungi Admin untuk mengubah nama, email, atau role.
        </p>
      </CardBody>
    </Card>
  )
}
