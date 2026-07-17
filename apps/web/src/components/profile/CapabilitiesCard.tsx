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
          <h2 className="text-body font-semibold text-fg">Produk &amp; Layanan</h2>
          <Tooltip content="Dipakai AI untuk menilai relevansi tender (bukan cuma cocok kata kunci) & generate keyword pencarian.">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <Field label="Produk" helper="Nama produk/solusi spesifik yang kamu tawarkan">
          <ChipInput
            value={form.products}
            onChange={(v) => onChange({ products: v })}
            disabled={disabled}
            placeholder="mis. Sistem Absensi Cloud, ERP Retail…"
          />
        </Field>

        <Field label="Layanan" helper="Kategori jasa yang kamu kerjakan">
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

        <Field
          label="Bukti / Portfolio"
          helper="Nama client, nama project, atau link case study yang membuktikan kapabilitas di atas"
        >
          <ChipInput
            value={form.portfolioRefs}
            onChange={(v) => onChange({ portfolioRefs: v })}
            disabled={disabled}
            placeholder="CTP PLTE, Case Study Dashboard BUMN…"
          />
        </Field>
      </CardBody>
    </Card>
  )
}
