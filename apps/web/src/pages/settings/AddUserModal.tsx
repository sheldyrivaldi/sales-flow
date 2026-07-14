import { useState } from 'react'
import Modal from '../../components/ui/Modal'
import Button from '../../components/ui/Button'
import Field from '../../components/ui/Field'
import Input from '../../components/ui/Input'
import Select from '../../components/ui/Select'
import { toast } from '../../lib/toast'
import { useCreateUser } from '../../api/users'
import type { UserRole } from '../../api/users'

const ROLE_OPTIONS: UserRole[] = ['SALES', 'OPS', 'MANAGER', 'ADMIN']

export interface AddUserModalProps {
  open: boolean
  onClose: () => void
}

export default function AddUserModal({ open, onClose }: AddUserModalProps) {
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [role, setRole] = useState<UserRole>('SALES')
  const [password, setPassword] = useState('')
  const createUser = useCreateUser()

  function reset() {
    setEmail('')
    setName('')
    setRole('SALES')
    setPassword('')
  }

  async function handleCreate() {
    if (!email.trim() || !name.trim() || !password) {
      toast.error('Email, nama, dan password wajib diisi.')
      return
    }
    if (password.length < 8) {
      toast.error('Password minimal 8 karakter.')
      return
    }
    try {
      await createUser.mutateAsync({ email: email.trim(), name: name.trim(), role, password })
      toast.success('Akun berhasil dibuat.')
      reset()
      onClose()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal membuat akun, coba lagi nanti.')
    }
  }

  return (
    <Modal
      open={open}
      onClose={() => {
        reset()
        onClose()
      }}
      title="Tambah Akun"
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={createUser.isPending}>
            Batal
          </Button>
          <Button loading={createUser.isPending} onClick={handleCreate}>
            Buat Akun
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama" htmlFor="user-name" required>
          <Input id="user-name" value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="Email" htmlFor="user-email" required>
          <Input id="user-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </Field>
        <Field label="Role" htmlFor="user-role" required>
          <Select id="user-role" value={role} onChange={(e) => setRole(e.target.value as UserRole)}>
            {ROLE_OPTIONS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Password Awal" htmlFor="user-password" required helper="Minimal 8 karakter.">
          <Input
            id="user-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>
      </div>
    </Modal>
  )
}
