import { Sparkles } from 'lucide-react'

import ScoreRing from './ui/ScoreRing'
import { ActionBadge } from './ui/Badge'
import { RiskFlagList } from './ui/RiskFlag'
import AiCallout from './ui/AiCallout'
import Button from './ui/Button'
import Tooltip from './ui/Tooltip'
import { formatRelative } from '../lib/format'
import { actionToLabel } from '../api/tenders'
import type { Tender } from '../api/tenders'

type RiskFlagItem = { label: string; severity?: 'warning' | 'danger' }

function parseRiskFlags(raw: unknown): RiskFlagItem[] {
  if (!raw || !Array.isArray(raw)) return []
  return raw.flatMap((item): RiskFlagItem[] => {
    if (typeof item === 'string') return [{ label: item }]
    if (item && typeof item === 'object' && 'label' in item && typeof (item as { label: unknown }).label === 'string') {
      const flag = item as { label: string; severity?: unknown }
      const severity = flag.severity === 'danger' ? 'danger' : 'warning'
      return [{ label: flag.label, severity }]
    }
    return []
  })
}

export interface AiScorePanelProps {
  tender: Tender
}

export default function AiScorePanel({ tender }: AiScorePanelProps) {
  const riskFlags = parseRiskFlags(tender.risk_flags)
  const hasScore = tender.fit_score != null

  if (!hasScore) {
    return (
      <AiCallout title="Belum dianalisa AI">
        <p className="mt-1 text-body text-fg-muted">
          Jalankan analisa AI untuk mendapatkan Fit Score, rekomendasi, dan identifikasi risiko.
        </p>
        {tender.origin === 'discovery' && tender.source_url && (
          <a
            href={tender.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-2 inline-flex items-center gap-1 text-caption text-primary hover:underline"
          >
            Lihat sumber asli ↗
          </a>
        )}
        <div className="mt-3">
          <Tooltip content="Analisa AI tersedia di EP-10">
            <Button
              size="sm"
              variant="secondary"
              leftIcon={<Sparkles className="w-3.5 h-3.5" />}
              disabled
            >
              Analisa Sekarang
            </Button>
          </Tooltip>
        </div>
      </AiCallout>
    )
  }

  return (
    <div className="rounded-card border border-accent/20 bg-accent/5 p-4 flex flex-col gap-3">
      {/* Score + Action */}
      <div className="flex items-center gap-4">
        <ScoreRing score={tender.fit_score!} size={64} />
        <div className="flex flex-col gap-1.5">
          {tender.recommended_action && (
            <ActionBadge action={actionToLabel(tender.recommended_action)} />
          )}
          <p className="text-caption text-fg-muted">
            Dibuat AI • {formatRelative(tender.updated_at)}
          </p>
        </div>
      </div>

      {/* Reasoning */}
      {tender.reasoning_summary && (
        <p className="text-body text-fg-muted">{tender.reasoning_summary}</p>
      )}

      {/* Risk flags */}
      {riskFlags.length > 0 && <RiskFlagList items={riskFlags} />}

      {/* Source link for discovery */}
      {tender.origin === 'discovery' && tender.source_url && (
        <a
          href={tender.source_url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-caption text-primary hover:underline"
        >
          Lihat sumber asli ({tender.source_name ?? 'link'}) ↗
        </a>
      )}

      {/* Re-analyze (placeholder EP-10) */}
      <div>
        <Tooltip content="Analisa ulang tersedia di EP-10">
          <Button
            size="sm"
            variant="ghost"
            leftIcon={<Sparkles className="w-3.5 h-3.5" />}
            disabled
          >
            Analisa ulang
          </Button>
        </Tooltip>
      </div>
    </div>
  )
}
