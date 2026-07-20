import { FileCheck2, Info } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import ChipInput from '../ui/ChipInput'
import Tooltip from '../ui/Tooltip'
import { SUPPORT_DOCUMENT_PRESETS } from '../../lib/profilePresets'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

export interface SupportDocsCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
}

/** Kartu "Dokumen Pendukung Tender": daftar dokumen administrasi yang SUDAH
 * dimiliki perusahaan. AI memakainya saat scoring tender dan saat ceklis
 * kelengkapan dokumen — kesiapan administrasi dinilai dari fakta di sini,
 * bukan tebakan. */
export default function SupportDocsCard({ form, onChange, disabled }: SupportDocsCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <FileCheck2 className="w-4 h-4 text-primary" aria-hidden="true" />
          <h2 className="text-body font-semibold text-fg">Dokumen Pendukung Tender</h2>
          <Tooltip content="Dipakai AI saat mencari & menganalisa tender: ceklis kelengkapan dokumen membandingkan syarat tender dengan daftar ini.">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody>
        <Field
          label="Dokumen yang sudah dimiliki"
          helper="Legalitas, sertifikasi, dan dokumen administrasi yang siap dilampirkan saat ikut tender"
        >
          <ChipInput
            value={form.supportDocuments}
            onChange={(v) => onChange({ supportDocuments: v })}
            presets={SUPPORT_DOCUMENT_PRESETS}
            disabled={disabled}
            placeholder="mis. NIB, ISO 9001, Laporan Keuangan Audited…"
          />
        </Field>
      </CardBody>
    </Card>
  )
}
