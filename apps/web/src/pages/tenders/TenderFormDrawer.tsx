import { useState, useEffect, useId } from 'react'
import { Sparkles } from 'lucide-react'

import Drawer from '../../components/ui/Drawer'
import Field from '../../components/ui/Field'
import Input from '../../components/ui/Input'
import Textarea from '../../components/ui/Textarea'
import Select from '../../components/ui/Select'
import Button from '../../components/ui/Button'
import { toast } from '../../lib/toast'

import { useCreateTender, useUpdateTender } from '../../api/tenders'
import type { Tender, TenderStatus, TenderCreateBody } from '../../api/tenders'

export interface TenderFormDrawerProps {
  open: boolean
  onClose: () => void
  tender?: Tender
  onSaved?: (t: Tender) => void
}

interface FormState {
  title: string
  buyer_name: string
  buyer_country: string
  buyer_industry: string
  value_estimate: string
  currency: string
  submission_deadline: string
  published_date: string
  source_name: string
  source_url: string
  service_category: string
  scope_summary: string
  eligibility_requirements: string
  technical_requirements: string
  status: TenderStatus
}

const emptyForm: FormState = {
  title: '',
  buyer_name: '',
  buyer_country: '',
  buyer_industry: '',
  value_estimate: '',
  currency: 'IDR',
  submission_deadline: '',
  published_date: '',
  source_name: '',
  source_url: '',
  service_category: '',
  scope_summary: '',
  eligibility_requirements: '',
  technical_requirements: '',
  status: 'IDENTIFIED',
}

function tenderToForm(t: Tender): FormState {
  return {
    title: t.title,
    buyer_name: t.buyer_name ?? '',
    buyer_country: t.buyer_country ?? '',
    buyer_industry: t.buyer_industry ?? '',
    value_estimate: t.value_estimate != null ? String(t.value_estimate) : '',
    currency: t.currency,
    submission_deadline: t.submission_deadline ? t.submission_deadline.slice(0, 10) : '',
    published_date: t.published_date ? t.published_date.slice(0, 10) : '',
    source_name: t.source_name ?? '',
    source_url: t.source_url ?? '',
    service_category: t.service_category ?? '',
    scope_summary: t.scope_summary ?? '',
    eligibility_requirements: t.eligibility_requirements ?? '',
    technical_requirements: t.technical_requirements ?? '',
    status: t.status,
  }
}

function formToBody(form: FormState): TenderCreateBody {
  const body: TenderCreateBody = { title: form.title }
  if (form.buyer_name) body.buyer_name = form.buyer_name
  if (form.buyer_country) body.buyer_country = form.buyer_country
  if (form.buyer_industry) body.buyer_industry = form.buyer_industry
  if (form.value_estimate !== '') {
    const n = parseFloat(form.value_estimate)
    if (!isNaN(n)) body.value_estimate = n
  }
  if (form.currency) body.currency = form.currency
  if (form.submission_deadline) body.submission_deadline = form.submission_deadline
  if (form.published_date) body.published_date = form.published_date
  if (form.source_name) body.source_name = form.source_name
  if (form.source_url) body.source_url = form.source_url
  if (form.service_category) body.service_category = form.service_category
  if (form.scope_summary) body.scope_summary = form.scope_summary
  if (form.eligibility_requirements) body.eligibility_requirements = form.eligibility_requirements
  if (form.technical_requirements) body.technical_requirements = form.technical_requirements
  if (form.status) body.status = form.status
  return body
}

export default function TenderFormDrawer({ open, onClose, tender, onSaved }: TenderFormDrawerProps) {
  const titleId = useId()
  const isEdit = !!tender
  const [form, setForm] = useState<FormState>(emptyForm)
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({})

  const createMutation = useCreateTender()
  const updateMutation = useUpdateTender()
  const isPending = createMutation.isPending || updateMutation.isPending

  useEffect(() => {
    if (open) {
      setForm(tender ? tenderToForm(tender) : emptyForm)
      setErrors({})
    }
  }, [open, tender])

  function set(field: keyof FormState) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
      setForm((f) => ({ ...f, [field]: e.target.value }))
      setErrors((err) => ({ ...err, [field]: undefined }))
    }
  }

  function validate(): boolean {
    const newErrors: typeof errors = {}
    if (!form.title.trim()) newErrors.title = 'Judul wajib diisi.'
    if (form.value_estimate !== '' && parseFloat(form.value_estimate) < 0) {
      newErrors.value_estimate = 'Nilai tidak boleh negatif.'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  async function submit(withAnalyze = false) {
    if (!validate()) return
    const body = formToBody(form)
    try {
      let saved: Tender
      if (isEdit && tender) {
        saved = await updateMutation.mutateAsync({ id: tender.id, body })
      } else {
        saved = await createMutation.mutateAsync(body)
      }
      toast.success(isEdit ? 'Tender diperbarui.' : 'Tender dibuat.')
      if (withAnalyze) {
        toast.info('Analisa AI akan tersedia di EP-10.')
      }
      onSaved?.(saved)
      onClose()
    } catch {
      toast.error('Gagal menyimpan tender.')
    }
  }

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit Tender' : 'Tender Baru'}
      width="w-[520px]"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isPending}>
            Batal
          </Button>
          <Button
            variant="secondary"
            leftIcon={<Sparkles className="w-4 h-4" />}
            loading={isPending}
            onClick={() => submit(true)}
          >
            Simpan & Analisa AI
          </Button>
          <Button loading={isPending} onClick={() => submit(false)}>
            Simpan
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Judul" required htmlFor={`${titleId}-title`} error={errors.title}>
          <Input
            id={`${titleId}-title`}
            value={form.title}
            onChange={set('title')}
            invalid={!!errors.title}
            placeholder="Nama tender / proyek"
          />
        </Field>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Buyer / Instansi" htmlFor={`${titleId}-buyer`}>
            <Input
              id={`${titleId}-buyer`}
              value={form.buyer_name}
              onChange={set('buyer_name')}
              placeholder="Nama instansi"
            />
          </Field>

          <Field label="Negara" htmlFor={`${titleId}-country`}>
            <Input
              id={`${titleId}-country`}
              value={form.buyer_country}
              onChange={set('buyer_country')}
              placeholder="Indonesia"
            />
          </Field>
        </div>

        <Field label="Industri" htmlFor={`${titleId}-industry`}>
          <Input
            id={`${titleId}-industry`}
            value={form.buyer_industry}
            onChange={set('buyer_industry')}
            placeholder="Government, Healthcare, …"
          />
        </Field>

        <div className="grid grid-cols-3 gap-4">
          <div className="col-span-2">
            <Field label="Nilai Estimasi" htmlFor={`${titleId}-value`} error={errors.value_estimate}>
              <Input
                id={`${titleId}-value`}
                type="number"
                min="0"
                value={form.value_estimate}
                onChange={set('value_estimate')}
                invalid={!!errors.value_estimate}
                placeholder="0"
              />
            </Field>
          </div>
          <Field label="Mata Uang" htmlFor={`${titleId}-currency`}>
            <Select
              id={`${titleId}-currency`}
              value={form.currency}
              onChange={set('currency')}
            >
              <option value="IDR">IDR</option>
              <option value="USD">USD</option>
              <option value="EUR">EUR</option>
            </Select>
          </Field>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Deadline Pengajuan" htmlFor={`${titleId}-deadline`}>
            <Input
              id={`${titleId}-deadline`}
              type="date"
              value={form.submission_deadline}
              onChange={set('submission_deadline')}
            />
          </Field>

          <Field label="Tanggal Terbit" htmlFor={`${titleId}-published`}>
            <Input
              id={`${titleId}-published`}
              type="date"
              value={form.published_date}
              onChange={set('published_date')}
            />
          </Field>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Nama Sumber" htmlFor={`${titleId}-srcname`}>
            <Input
              id={`${titleId}-srcname`}
              value={form.source_name}
              onChange={set('source_name')}
              placeholder="SPSE, LKPP, …"
            />
          </Field>

          <Field label="URL Sumber" htmlFor={`${titleId}-srcurl`}>
            <Input
              id={`${titleId}-srcurl`}
              value={form.source_url}
              onChange={set('source_url')}
              placeholder="https://…"
            />
          </Field>
        </div>

        <Field label="Kategori Layanan" htmlFor={`${titleId}-category`}>
          <Input
            id={`${titleId}-category`}
            value={form.service_category}
            onChange={set('service_category')}
            placeholder="Web app, System Integrator, …"
          />
        </Field>

        <Field label="Status" htmlFor={`${titleId}-status`}>
          <Select
            id={`${titleId}-status`}
            value={form.status}
            onChange={set('status')}
          >
            <option value="IDENTIFIED">IDENTIFIED</option>
            <option value="QUALIFYING">QUALIFYING</option>
            <option value="BIDDING">BIDDING</option>
            <option value="SUBMITTED">SUBMITTED</option>
            <option value="WON">WON</option>
            <option value="LOST">LOST</option>
          </Select>
        </Field>

        <Field label="Ringkasan Scope" htmlFor={`${titleId}-scope`}>
          <Textarea
            id={`${titleId}-scope`}
            value={form.scope_summary}
            onChange={set('scope_summary')}
            rows={3}
            placeholder="Deskripsi singkat lingkup pekerjaan…"
          />
        </Field>

        <Field label="Syarat Kelayakan" htmlFor={`${titleId}-elig`}>
          <Textarea
            id={`${titleId}-elig`}
            value={form.eligibility_requirements}
            onChange={set('eligibility_requirements')}
            rows={2}
            placeholder="NIB, NPWP, pengalaman 3 tahun, …"
          />
        </Field>

        <Field label="Persyaratan Teknis" htmlFor={`${titleId}-tech`}>
          <Textarea
            id={`${titleId}-tech`}
            value={form.technical_requirements}
            onChange={set('technical_requirements')}
            rows={2}
            placeholder="Stack, sertifikasi, spesifikasi teknis…"
          />
        </Field>
      </div>
    </Drawer>
  )
}
