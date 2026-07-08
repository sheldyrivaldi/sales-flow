import { useId } from 'react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import Input from '../ui/Input'
import Tooltip from '../ui/Tooltip'
import { Info } from 'lucide-react'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

export interface ProfileCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
  error?: string
}

export default function ProfileCard({ form, onChange, disabled, error }: ProfileCardProps) {
  const titleId = useId()

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <h2 className="text-body font-semibold text-fg">Profil Perusahaan</h2>
          <Tooltip content="Identitas dasar perusahaan yang dipakai AI untuk memperkenalkan diri & membangun konteks.">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <Field label="Nama perusahaan" required htmlFor={`${titleId}-name`} error={error}>
          <Input
            id={`${titleId}-name`}
            value={form.companyName}
            onChange={(e) => onChange({ companyName: e.target.value })}
            disabled={disabled}
            invalid={!!error}
            placeholder="PT Contoh Teknologi"
          />
        </Field>

        <Field label="One-liner" htmlFor={`${titleId}-oneliner`} helper="Deskripsi singkat perusahaan, 1 kalimat">
          <Input
            id={`${titleId}-oneliner`}
            value={form.oneLiner}
            onChange={(e) => onChange({ oneLiner: e.target.value })}
            disabled={disabled}
            placeholder="Kami membangun software untuk…"
          />
        </Field>
      </CardBody>
    </Card>
  )
}
