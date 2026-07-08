import { Info } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import ChipInput from '../ui/ChipInput'
import Tooltip from '../ui/Tooltip'
import { CAPABILITY_PRESETS } from '../../lib/profilePresets'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

export interface CapabilitiesCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
}

export default function CapabilitiesCard({ form, onChange, disabled }: CapabilitiesCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <h2 className="text-body font-semibold text-fg">Kapabilitas (yang dijual)</h2>
          <Tooltip content="Dipakai untuk generate keyword pencarian tender & scoring kecocokan.">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <Field label="Layanan">
          <ChipInput
            value={form.serviceCategories}
            onChange={(v) => onChange({ serviceCategories: v })}
            presets={CAPABILITY_PRESETS}
            disabled={disabled}
            placeholder="Tambah layanan…"
          />
        </Field>

        <Field label="Tech stack">
          <ChipInput
            value={form.techStack}
            onChange={(v) => onChange({ techStack: v })}
            disabled={disabled}
            placeholder="React, Go, Node…"
          />
        </Field>
      </CardBody>
    </Card>
  )
}
