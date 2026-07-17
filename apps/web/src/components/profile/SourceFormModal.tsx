import { useState, useEffect, useId } from 'react'
import type { ChangeEvent } from 'react'
import Modal from '../ui/Modal'
import Field from '../ui/Field'
import Input from '../ui/Input'
import Select from '../ui/Select'
import Button from '../ui/Button'
import ChipInput from '../ui/ChipInput'
import { toast } from '../../lib/toast'
import {
  useCreateSource,
  useUpdateSource,
  ACCESS_LABELS,
  FREQUENCY_LABELS,
  PRIORITY_TIERS,
} from '../../api/sources'
import type { Source, SourceAccess, SourceFrequency } from '../../api/sources'

export interface SourceFormModalProps {
  open: boolean
  onClose: () => void
  source?: Source
}

interface FormState {
  name: string
  url: string
  country: string
  access: SourceAccess
  legalNote: string
  priority: number
  frequency: SourceFrequency
  dataTypes: string[]
}

const emptyForm: FormState = {
  name: '',
  url: '',
  country: '',
  access: 'publik',
  legalNote: '',
  priority: PRIORITY_TIERS[1].value,
  frequency: 'harian',
  dataTypes: [],
}

function sourceToForm(s: Source): FormState {
  return {
    name: s.name,
    url: s.url,
    country: s.country ?? '',
    access: s.access,
    legalNote: s.legal_note ?? '',
    priority: s.priority,
    frequency: s.frequency,
    dataTypes: s.data_types,
  }
}

export default function SourceFormModal({ open, onClose, source }: SourceFormModalProps) {
  const titleId = useId()
  const isEdit = !!source
  const [form, setForm] = useState<FormState>(emptyForm)
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({})

  const createMutation = useCreateSource()
  const updateMutation = useUpdateSource()
  const isPending = createMutation.isPending || updateMutation.isPending

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- reset form saat modal dibuka (open/source berubah), pola sama seperti ProspectFormDrawer.
      setForm(source ? sourceToForm(source) : emptyForm)
      setErrors({})
    }
  }, [open, source])

  function set(field: keyof FormState) {
    return (e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
      setForm((f) => ({ ...f, [field]: e.target.value }))
      setErrors((err) => ({ ...err, [field]: undefined }))
    }
  }

  function validate(): boolean {
    const next: typeof errors = {}
    if (!form.name.trim()) next.name = 'Nama wajib diisi.'
    if (!form.url.trim()) next.url = 'URL wajib diisi.'
    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function submit() {
    if (!validate()) return
    const body = {
      name: form.name,
      url: form.url,
      country: form.country || undefined,
      access: form.access,
      legal_note: form.legalNote || undefined,
      priority: form.priority,
      frequency: form.frequency,
      data_types: form.dataTypes,
    }
    try {
      if (isEdit && source) {
        await updateMutation.mutateAsync({ id: source.id, body })
      } else {
        await createMutation.mutateAsync(body)
      }
      toast.success(isEdit ? 'Sumber diperbarui.' : 'Sumber ditambahkan.')
      onClose()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Gagal menyimpan sumber.'
      setErrors((e) => ({ ...e, url: message }))
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit Sumber' : 'Tambah Sumber'}
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
          <Input id={`${titleId}-name`} value={form.name} onChange={set('name')} invalid={!!errors.name} />
        </Field>
        <Field label="URL" required htmlFor={`${titleId}-url`} error={errors.url}>
          <Input
            id={`${titleId}-url`}
            value={form.url}
            onChange={set('url')}
            invalid={!!errors.url}
            placeholder="https://…"
          />
        </Field>
        <Field label="Negara" htmlFor={`${titleId}-country`}>
          <Input id={`${titleId}-country`} value={form.country} onChange={set('country')} />
        </Field>
        <Field label="Akses" htmlFor={`${titleId}-access`}>
          <Select id={`${titleId}-access`} value={form.access} onChange={set('access')}>
            {(Object.keys(ACCESS_LABELS) as SourceAccess[]).map((a) => (
              <option key={a} value={a}>
                {ACCESS_LABELS[a]}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Legal note" htmlFor={`${titleId}-legal`}>
          <Input id={`${titleId}-legal`} value={form.legalNote} onChange={set('legalNote')} />
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Prioritas" htmlFor={`${titleId}-priority`}>
            <Select
              id={`${titleId}-priority`}
              value={form.priority}
              onChange={(e) => setForm((f) => ({ ...f, priority: Number(e.target.value) }))}
            >
              {PRIORITY_TIERS.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Frekuensi" htmlFor={`${titleId}-frequency`}>
            <Select
              id={`${titleId}-frequency`}
              value={form.frequency}
              onChange={(e) => setForm((f) => ({ ...f, frequency: e.target.value as typeof form.frequency }))}
            >
              {(Object.keys(FREQUENCY_LABELS) as (typeof form.frequency)[]).map((f) => (
                <option key={f} value={f}>
                  {FREQUENCY_LABELS[f]}
                </option>
              ))}
            </Select>
          </Field>
        </div>
        <Field label="Jenis data" helper="Tender pemerintah, RFP/RFQ, pengumuman lelang…">
          <ChipInput value={form.dataTypes} onChange={(v) => setForm((f) => ({ ...f, dataTypes: v }))} />
        </Field>
      </div>
    </Modal>
  )
}
