import { useState, useEffect, useId } from 'react'

import Drawer from '../../components/ui/Drawer'
import Field from '../../components/ui/Field'
import Input from '../../components/ui/Input'
import Textarea from '../../components/ui/Textarea'
import Select from '../../components/ui/Select'
import Button from '../../components/ui/Button'
import { toast } from '../../lib/toast'

import { useCreateProspect, useUpdateProspect, PROSPECT_STAGES, SOURCE_LABELS } from '../../api/prospects'
import type { Prospect, ProspectStage, ProspectSource, ProspectCreateBody } from '../../api/prospects'
import { useUsers } from '../../api/users'

export interface ProspectFormDrawerProps {
  open: boolean
  onClose: () => void
  prospect?: Prospect
  onSaved?: (p: Prospect) => void
}

interface FormState {
  name: string
  company: string
  contact_info: string
  source_type: ProspectSource
  est_value: string
  owner_user_id: string
  stage: ProspectStage
}

const emptyForm: FormState = {
  name: '',
  company: '',
  contact_info: '',
  source_type: 'manual',
  est_value: '',
  owner_user_id: '',
  stage: 'NEW',
}

function prospectToForm(p: Prospect): FormState {
  return {
    name: p.name,
    company: p.company ?? '',
    contact_info: p.contact_info ?? '',
    source_type: p.source_type,
    est_value: p.est_value != null ? String(p.est_value) : '',
    owner_user_id: p.owner_user_id ?? '',
    stage: p.stage,
  }
}

function formToBody(form: FormState): ProspectCreateBody {
  const body: ProspectCreateBody = { name: form.name }
  // company/contact_info are always sent (even empty) so clearing a field in
  // the edit form actually clears it server-side — omitting them here would
  // mean "don't touch", silently keeping the old value.
  body.company = form.company
  body.contact_info = form.contact_info
  if (form.source_type) body.source_type = form.source_type
  if (form.est_value !== '') {
    const n = parseFloat(form.est_value)
    if (!isNaN(n)) body.est_value = n
  }
  // owner_user_id is a UUID column — only send it when a real user is
  // selected; sending '' would fail the UUID column constraint server-side.
  if (form.owner_user_id) body.owner_user_id = form.owner_user_id
  if (form.stage) body.stage = form.stage
  return body
}

export default function ProspectFormDrawer({ open, onClose, prospect, onSaved }: ProspectFormDrawerProps) {
  const titleId = useId()
  const isEdit = !!prospect
  const [form, setForm] = useState<FormState>(emptyForm)
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({})

  const createMutation = useCreateProspect()
  const updateMutation = useUpdateProspect()
  const isPending = createMutation.isPending || updateMutation.isPending
  const { data: usersData } = useUsers()

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- reset form saat drawer dibuka (open/prospect berubah), pola sama seperti TenderFormDrawer/EventFormDrawer.
      setForm(prospect ? prospectToForm(prospect) : emptyForm)
      setErrors({})
    }
  }, [open, prospect])

  function set(field: keyof FormState) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
      setForm((f) => ({ ...f, [field]: e.target.value }))
      setErrors((err) => ({ ...err, [field]: undefined }))
    }
  }

  function validate(): boolean {
    const newErrors: typeof errors = {}
    if (!form.name.trim()) newErrors.name = 'Nama wajib diisi.'
    if (form.est_value !== '' && parseFloat(form.est_value) < 0) {
      newErrors.est_value = 'Nilai tidak boleh negatif.'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  async function submit() {
    if (!validate()) return
    const body = formToBody(form)
    try {
      let saved: Prospect
      if (isEdit && prospect) {
        saved = await updateMutation.mutateAsync({ id: prospect.id, body })
      } else {
        saved = await createMutation.mutateAsync(body)
      }
      toast.success(isEdit ? 'Prospek diperbarui.' : 'Prospek dibuat.')
      onSaved?.(saved)
      onClose()
    } catch {
      toast.error('Gagal menyimpan prospek.')
    }
  }

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit Prospek' : 'Prospek Baru'}
      width="w-[480px]"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isPending}>
            Batal
          </Button>
          <Button loading={isPending} onClick={submit}>
            Simpan
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama" required htmlFor={`${titleId}-name`} error={errors.name}>
          <Input
            id={`${titleId}-name`}
            value={form.name}
            onChange={set('name')}
            invalid={!!errors.name}
            placeholder="Nama prospek / kontak"
          />
        </Field>

        <Field label="Perusahaan" htmlFor={`${titleId}-company`}>
          <Input
            id={`${titleId}-company`}
            value={form.company}
            onChange={set('company')}
            placeholder="Nama perusahaan"
          />
        </Field>

        <Field label="Info Kontak" htmlFor={`${titleId}-contact`}>
          <Textarea
            id={`${titleId}-contact`}
            value={form.contact_info}
            onChange={set('contact_info')}
            rows={2}
            placeholder="Email, telepon, alamat…"
          />
        </Field>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Sumber" htmlFor={`${titleId}-source`}>
            <Select id={`${titleId}-source`} value={form.source_type} onChange={set('source_type')}>
              {(Object.keys(SOURCE_LABELS) as ProspectSource[]).map((s) => (
                <option key={s} value={s}>
                  {SOURCE_LABELS[s]}
                </option>
              ))}
            </Select>
          </Field>

          <Field label="Stage" htmlFor={`${titleId}-stage`}>
            <Select id={`${titleId}-stage`} value={form.stage} onChange={set('stage')}>
              {PROSPECT_STAGES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Nilai Estimasi" htmlFor={`${titleId}-value`} error={errors.est_value}>
            <Input
              id={`${titleId}-value`}
              type="number"
              min="0"
              value={form.est_value}
              onChange={set('est_value')}
              invalid={!!errors.est_value}
              placeholder="0"
            />
          </Field>

          <Field label="Owner" htmlFor={`${titleId}-owner`}>
            <Select id={`${titleId}-owner`} value={form.owner_user_id} onChange={set('owner_user_id')}>
              <option value="">Kosongkan = saya</option>
              {usersData?.items.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name}
                </option>
              ))}
            </Select>
          </Field>
        </div>
      </div>
    </Drawer>
  )
}
