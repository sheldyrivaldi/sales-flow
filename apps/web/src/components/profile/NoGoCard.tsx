import { Info } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import Toggle from '../ui/Toggle'
import ChipInput from '../ui/ChipInput'
import Tooltip from '../ui/Tooltip'
import { NOGO_PRESET_FLAGS } from '../../lib/profilePresets'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

export interface NoGoCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
}

export default function NoGoCard({ form, onChange, disabled }: NoGoCardProps) {
  function toggleFlag(flag: string, checked: boolean) {
    const next = checked
      ? [...form.presetFlags, flag]
      : form.presetFlags.filter((f) => f !== flag)
    onChange({ presetFlags: next })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <h2 className="text-body font-semibold text-fg">Hindari (No-Go)</h2>
          <Tooltip content="Tender yang cocok dengan aturan ini akan otomatis ditandai No-Go/Need Partner (EP-10).">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          {NOGO_PRESET_FLAGS.map((flag) => (
            <Toggle
              key={flag}
              checked={form.presetFlags.includes(flag)}
              onChange={(checked) => toggleFlag(flag, checked)}
              label={flag}
              disabled={disabled}
              size="sm"
            />
          ))}
        </div>

        <Field label="Lainnya" helper="Aturan no-go kustom">
          <ChipInput
            value={form.customNoGo}
            onChange={(v) => onChange({ customNoGo: v })}
            disabled={disabled}
            placeholder="Tambah aturan…"
          />
        </Field>
      </CardBody>
    </Card>
  )
}
