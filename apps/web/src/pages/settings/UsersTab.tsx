import { useState } from 'react'
import { Plus, Copy } from 'lucide-react'
import Table from '../../components/ui/Table'
import type { Column, KebabAction } from '../../components/ui/Table'
import Button from '../../components/ui/Button'
import Badge from '../../components/ui/Badge'
import Select from '../../components/ui/Select'
import Modal from '../../components/ui/Modal'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import { toast } from '../../lib/toast'
import { useUsers, useUpdateUser, useResetPassword } from '../../api/users'
import type { User, UserRole } from '../../api/users'
import AddUserModal from './AddUserModal'

const ROLE_OPTIONS: UserRole[] = ['SALES', 'OPS', 'MANAGER', 'ADMIN']

export default function UsersTab() {
  const { data, isLoading } = useUsers()
  const updateUser = useUpdateUser()
  const resetPassword = useResetPassword()

  const [addOpen, setAddOpen] = useState(false)
  const [toggleTarget, setToggleTarget] = useState<User | null>(null)
  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [newPassword, setNewPassword] = useState<string | null>(null)

  const items = data?.items ?? []

  async function handleRoleChange(user: User, role: UserRole) {
    try {
      await updateUser.mutateAsync({ id: user.id, body: { role } })
      toast.success(`Role ${user.name} diubah menjadi ${role}.`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal mengubah role.')
    }
  }

  async function handleToggleActive() {
    if (!toggleTarget) return
    try {
      await updateUser.mutateAsync({ id: toggleTarget.id, body: { active: !toggleTarget.active } })
      toast.success(toggleTarget.active ? 'Akun dinonaktifkan.' : 'Akun diaktifkan kembali.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal mengubah status akun.')
    } finally {
      setToggleTarget(null)
    }
  }

  async function handleResetPassword() {
    if (!resetTarget) return
    try {
      const res = await resetPassword.mutateAsync(resetTarget.id)
      setNewPassword(res.password ?? null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal reset password.')
    } finally {
      setResetTarget(null)
    }
  }

  const columns: Column<User>[] = [
    { key: 'name', header: 'Nama', render: (u) => u.name },
    { key: 'email', header: 'Email', render: (u) => u.email },
    {
      key: 'role',
      header: 'Role',
      render: (u) => (
        <Select
          value={u.role}
          onChange={(e) => handleRoleChange(u, e.target.value as UserRole)}
          className="h-8 py-1 text-caption"
          disabled={updateUser.isPending}
        >
          {ROLE_OPTIONS.map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </Select>
      ),
    },
    {
      key: 'active',
      header: 'Status',
      render: (u) => (
        <Badge tone={u.active ? 'success' : 'danger'}>{u.active ? 'Aktif' : 'Nonaktif'}</Badge>
      ),
    },
  ]

  function kebabActions(u: User): KebabAction[] {
    return [
      { label: u.active ? 'Nonaktifkan' : 'Aktifkan', onClick: () => setToggleTarget(u), danger: u.active },
      { label: 'Reset Password', onClick: () => setResetTarget(u) },
    ]
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={() => setAddOpen(true)}>
          Tambah Akun
        </Button>
      </div>

      <Table
        columns={columns}
        data={items}
        rowKey={(u) => u.id}
        kebabActions={kebabActions}
        loading={isLoading}
        empty={<p className="text-body text-fg-muted py-8 text-center">Belum ada user.</p>}
      />

      <AddUserModal open={addOpen} onClose={() => setAddOpen(false)} />

      <ConfirmDialog
        open={!!toggleTarget}
        onCancel={() => setToggleTarget(null)}
        onConfirm={handleToggleActive}
        title={toggleTarget?.active ? 'Nonaktifkan akun?' : 'Aktifkan akun?'}
        description={
          toggleTarget?.active
            ? `${toggleTarget?.name} tidak akan bisa login setelah dinonaktifkan.`
            : `${toggleTarget?.name} akan bisa login kembali.`
        }
        tone={toggleTarget?.active ? 'danger' : 'primary'}
        confirmLabel={toggleTarget?.active ? 'Nonaktifkan' : 'Aktifkan'}
        loading={updateUser.isPending}
      />

      <ConfirmDialog
        open={!!resetTarget}
        onCancel={() => setResetTarget(null)}
        onConfirm={handleResetPassword}
        title="Reset password?"
        description={`Password baru akan dibuat otomatis untuk ${resetTarget?.name}.`}
        tone="danger"
        confirmLabel="Reset"
        loading={resetPassword.isPending}
      />

      <Modal
        open={!!newPassword}
        onClose={() => setNewPassword(null)}
        title="Password Baru"
        size="sm"
        footer={<Button onClick={() => setNewPassword(null)}>Selesai</Button>}
      >
        <div className="flex flex-col gap-3">
          <p className="text-body text-fg-muted">
            Salin password ini sekarang — tidak akan ditampilkan lagi.
          </p>
          <div className="flex items-center gap-2 rounded-btn border border-line bg-surface-subtle px-3 py-2">
            <code className="flex-1 text-body font-mono text-fg">{newPassword}</code>
            <button
              type="button"
              onClick={() => {
                if (!newPassword) return
                navigator.clipboard.writeText(newPassword).then(
                  () => toast.success('Password disalin.'),
                  () => toast.error('Gagal menyalin. Salin manual dari kotak di atas.'),
                )
              }}
              className="text-fg-muted hover:text-fg"
              aria-label="Salin password"
            >
              <Copy className="w-4 h-4" />
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
