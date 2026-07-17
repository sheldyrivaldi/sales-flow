import { Info, Sparkles } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import ChipInput from '../ui/ChipInput'
import Button from '../ui/Button'
import Badge from '../ui/Badge'
import Select from '../ui/Select'
import Toggle from '../ui/Toggle'
import Tooltip from '../ui/Tooltip'
import { useKeywordGeneration } from '../../lib/useKeywordGeneration'
import { dedupCaseInsensitive } from '../../lib/dedup'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

const CRAWL_FREQUENCY_OPTIONS: { value: string; label: string }[] = [
  { value: 'harian', label: 'Harian' },
  { value: '2-3x', label: '2-3x seminggu' },
  { value: 'mingguan', label: 'Mingguan' },
]

export interface SourcesKeywordCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
}

export default function SourcesKeywordCard({ form, onChange, disabled }: SourcesKeywordCardProps) {
  const { generate, degraded, isPending } = useKeywordGeneration()

  async function handleGenerate() {
    const res = await generate(form.serviceCategories)
    if (!res) return
    onChange({
      keywords: dedupCaseInsensitive([...form.keywords, ...res.keywords]),
      negativeKeywords: dedupCaseInsensitive([...form.negativeKeywords, ...res.negative_keywords]),
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <h2 className="text-body font-semibold text-fg">Sumber &amp; Keyword</h2>
          <Tooltip content="Keyword dipakai AI untuk mencari tender (EP-12). Kelola daftar sumber di tab 'Sumber'.">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <p className="text-caption text-fg-muted">
          Kelola daftar sumber crawling di tab &ldquo;Sumber&rdquo; di atas.
        </p>

        <div className="flex flex-col gap-2">
          <Button
            variant="secondary"
            size="sm"
            leftIcon={<Sparkles className="w-4 h-4" />}
            loading={isPending}
            onClick={handleGenerate}
            disabled={disabled}
            className="self-start"
          >
            Generate dari kapabilitas
          </Button>
          {degraded && <Badge tone="warning">AI tidak tersedia saat generate terakhir</Badge>}
        </div>

        <Field label="Keyword">
          <ChipInput
            value={form.keywords}
            onChange={(v) => onChange({ keywords: v })}
            disabled={disabled}
            placeholder="pengadaan aplikasi, integrasi sistem…"
          />
        </Field>

        <Field label="Keyword negatif">
          <ChipInput
            value={form.negativeKeywords}
            onChange={(v) => onChange({ negativeKeywords: v })}
            disabled={disabled}
            placeholder="hardware only, pengadaan laptop…"
          />
        </Field>

        <Field
          label="Crawl otomatis"
          helper="Jadwal disinkronkan ke penjadwal AI saat disimpan — berjalan otomatis tanpa perlu dipicu manual."
        >
          <div className="flex items-center gap-3">
            <Toggle
              checked={form.crawlEnabled}
              onChange={(checked) => onChange({ crawlEnabled: checked })}
              label={form.crawlEnabled ? 'Aktif' : 'Nonaktif'}
              disabled={disabled}
            />
            <Select
              value={form.crawlFrequency}
              onChange={(e) => onChange({ crawlFrequency: e.target.value })}
              disabled={disabled || !form.crawlEnabled}
              className="max-w-[10rem]"
            >
              {CRAWL_FREQUENCY_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </Select>
          </div>
        </Field>
      </CardBody>
    </Card>
  )
}
