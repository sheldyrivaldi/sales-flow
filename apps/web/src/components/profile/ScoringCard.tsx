import { Info, SlidersHorizontal } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import Input from '../ui/Input'
import Badge from '../ui/Badge'
import Tooltip from '../ui/Tooltip'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

export interface ScoringCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
}

// WEIGHT_FIELDS drives both the rubric grid and the total — order matches
// internal/ai/scoring.go's scoringRubric so the labels line up with what
// Hermes actually scores against.
const WEIGHT_FIELDS: { key: keyof OtakAgentFormState; label: string }[] = [
  { key: 'weightCapabilityFit', label: 'Capability fit' },
  { key: 'weightPortfolioMatch', label: 'Portfolio match' },
  { key: 'weightCommercialAttractiveness', label: 'Commercial attractiveness' },
  { key: 'weightEligibilityFit', label: 'Eligibility fit' },
  { key: 'weightDeadlineFeasibility', label: 'Deadline feasibility' },
  { key: 'weightStrategicAccountValue', label: 'Strategic account value' },
  { key: 'weightDeliveryRisk', label: 'Delivery risk' },
  { key: 'weightCompetitionWinProbability', label: 'Competition / win probability' },
]

function toNumber(v: string): number {
  const n = parseFloat(v)
  return isNaN(n) ? 0 : n
}

export default function ScoringCard({ form, onChange, disabled }: ScoringCardProps) {
  const total = WEIGHT_FIELDS.reduce((sum, f) => sum + toNumber(form[f.key] as string), 0)
  const totalOk = total === 100

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <SlidersHorizontal className="w-4 h-4 text-fg-muted" aria-hidden="true" />
          <h2 className="text-body font-semibold text-fg">Scoring</h2>
          <Tooltip content="Bobot & ambang batas ini dipakai langsung oleh AI saat menilai tender (rubrik §8) — bukan sekadar catatan.">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <p className="text-caption text-fg-muted">Bobot per dimensi (%) — total idealnya 100%</p>
          <Badge tone={totalOk ? 'success' : 'warning'}>Total: {total}%</Badge>
        </div>

        <div className="grid grid-cols-2 gap-3">
          {WEIGHT_FIELDS.map((f) => (
            <Field key={f.key} label={f.label}>
              <Input
                type="number"
                min="0"
                max="100"
                value={form[f.key] as string}
                onChange={(e) => onChange({ [f.key]: e.target.value } as OtakAgentFormPatch)}
                disabled={disabled}
              />
            </Field>
          ))}
        </div>

        <div className="flex flex-col gap-2 pt-2 border-t border-line">
          <p className="text-caption text-fg-muted">
            Ambang batas rekomendasi — skor ≥ Pursue jadi &ldquo;Pursue&rdquo;, di antara Pursue &amp; Review
            jadi &ldquo;Review&rdquo;, dst. Di bawah Watchlist otomatis &ldquo;Reject&rdquo;.
          </p>
          <div className="grid grid-cols-3 gap-3">
            <Field label="Pursue ≥">
              <Input
                type="number"
                min="0"
                max="100"
                value={form.thresholdPursue}
                onChange={(e) => onChange({ thresholdPursue: e.target.value })}
                disabled={disabled}
              />
            </Field>
            <Field label="Review ≥">
              <Input
                type="number"
                min="0"
                max="100"
                value={form.thresholdReview}
                onChange={(e) => onChange({ thresholdReview: e.target.value })}
                disabled={disabled}
              />
            </Field>
            <Field label="Watchlist ≥">
              <Input
                type="number"
                min="0"
                max="100"
                value={form.thresholdWatchlist}
                onChange={(e) => onChange({ thresholdWatchlist: e.target.value })}
                disabled={disabled}
              />
            </Field>
          </div>
        </div>
      </CardBody>
    </Card>
  )
}
