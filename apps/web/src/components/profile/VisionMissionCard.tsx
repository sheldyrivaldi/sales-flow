import { useId } from 'react'
import { Info } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import Textarea from '../ui/Textarea'
import Tooltip from '../ui/Tooltip'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

export interface VisionMissionCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
}

export default function VisionMissionCard({ form, onChange, disabled }: VisionMissionCardProps) {
  const titleId = useId()

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <h2 className="text-body font-semibold text-fg">Visi &amp; Misi</h2>
          <Tooltip content="Dibaca AI saat menilai apakah sebuah tender sejalan dengan arah perusahaan — bukan cuma cocok kata kunci.">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <Field
          label="Visi"
          htmlFor={`${titleId}-vision`}
          helper="Arah jangka panjang perusahaan"
        >
          <Textarea
            id={`${titleId}-vision`}
            value={form.vision}
            onChange={(e) => onChange({ vision: e.target.value })}
            disabled={disabled}
            placeholder="Menjadi mitra transformasi digital terpercaya untuk instansi pemerintah di Indonesia…"
          />
        </Field>

        <Field
          label="Misi"
          htmlFor={`${titleId}-mission`}
          helper="Bagaimana visi itu dijalankan sehari-hari"
        >
          <Textarea
            id={`${titleId}-mission`}
            value={form.mission}
            onChange={(e) => onChange({ mission: e.target.value })}
            disabled={disabled}
            placeholder="Menyediakan solusi teknologi yang andal, aman, dan sesuai regulasi pengadaan…"
          />
        </Field>
      </CardBody>
    </Card>
  )
}
