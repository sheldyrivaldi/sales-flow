import { useState } from 'react'
import { Sparkles, CheckCircle2, AlertTriangle, XCircle, ChevronDown, ChevronUp } from 'lucide-react'

import ScoreRing from './ui/ScoreRing'
import { ActionBadge } from './ui/Badge'
import { RiskFlagList } from './ui/RiskFlag'
import AiCallout from './ui/AiCallout'
import Button from './ui/Button'
import { formatRelative } from '../lib/format'
import { toast } from '../lib/toast'
import { actionToLabel } from '../api/tenders'
import type { Tender } from '../api/tenders'
import { useScore, useRunScore } from '../api/scores'
import type { ScoreTargetType, EvidenceItem } from '../api/scores'

function EvidenceIcon({ verdict }: { verdict: EvidenceItem['verdict'] }) {
  if (verdict === 'pass') return <CheckCircle2 className="w-3.5 h-3.5 text-success mt-0.5 shrink-0" aria-hidden="true" />
  if (verdict === 'fail') return <XCircle className="w-3.5 h-3.5 text-danger mt-0.5 shrink-0" aria-hidden="true" />
  return <AlertTriangle className="w-3.5 h-3.5 text-warning mt-0.5 shrink-0" aria-hidden="true" />
}

export interface AiScorePanelProps {
  targetType: ScoreTargetType
  targetId: string
  /** Optional — only tenders carry an origin/source link to show while empty. */
  tender?: Tender
}

export default function AiScorePanel({ targetType, targetId, tender }: AiScorePanelProps) {
  const { data: score, isLoading } = useScore(targetType, targetId)
  const runScore = useRunScore(targetType)
  const [reasonOpen, setReasonOpen] = useState(false)

  async function handleAnalyze() {
    try {
      await runScore.mutateAsync(targetId)
      toast.success('Analisa AI selesai.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Analisa AI gagal, coba lagi nanti.')
    }
  }

  if (isLoading) {
    return (
      <div className="rounded-card border border-line bg-surface p-4 animate-pulse">
        <div className="h-16 w-16 rounded-full bg-surface-subtle" />
      </div>
    )
  }

  if (!score) {
    return (
      <AiCallout title="Belum dianalisa AI">
        <p className="mt-1 text-body text-fg-muted">
          Jalankan analisa AI untuk mendapatkan Fit Score, rekomendasi, dan identifikasi risiko.
        </p>
        {tender?.origin === 'discovery' && tender.source_url && (
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
          <Button
            size="sm"
            variant="secondary"
            leftIcon={<Sparkles className="w-3.5 h-3.5" />}
            loading={runScore.isPending}
            onClick={handleAnalyze}
          >
            Analisa Sekarang
          </Button>
        </div>
      </AiCallout>
    )
  }

  const riskFlags = (score.risk_flags ?? []).map((label) => ({ label }))
  const confidencePct = score.confidence != null ? Math.round(score.confidence * 100) : null

  return (
    <div className="rounded-card border border-accent/20 bg-accent/5 p-4 flex flex-col gap-3">
      {/* Score + Action */}
      <div className="flex items-center gap-4">
        <ScoreRing score={score.fit_score} size={64} />
        <div className="flex flex-col gap-1.5">
          <ActionBadge action={actionToLabel(score.recommended_action)} />
          <p className="text-caption text-fg-muted">
            Dibuat AI{confidencePct != null ? ` • ${confidencePct}%` : ''} • {formatRelative(score.created_at)}
          </p>
        </div>
      </div>

      {/* Evidence per dimensi */}
      {score.evidence && score.evidence.length > 0 && (
        <div className="flex flex-col gap-1.5">
          {score.evidence.map((e, i) => (
            <div key={i} className="flex items-start gap-2 text-caption">
              <EvidenceIcon verdict={e.verdict} />
              <span className="text-fg-muted">
                <span className="font-medium text-fg">{e.dimension}:</span> {e.note}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Risk flags */}
      {riskFlags.length > 0 && <RiskFlagList items={riskFlags} />}

      {/* Lihat alasan */}
      {score.reasoning && (
        <div>
          <button
            type="button"
            onClick={() => setReasonOpen((o) => !o)}
            className="inline-flex items-center gap-1 text-caption font-medium text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-1 rounded"
            aria-expanded={reasonOpen}
          >
            {reasonOpen ? (
              <>
                <ChevronUp className="w-3.5 h-3.5" aria-hidden="true" /> Sembunyikan alasan
              </>
            ) : (
              <>
                <ChevronDown className="w-3.5 h-3.5" aria-hidden="true" /> Lihat alasan
              </>
            )}
          </button>
          {reasonOpen && (
            <p className="mt-2 text-body text-fg-muted border-t border-accent/10 pt-2">{score.reasoning}</p>
          )}
        </div>
      )}

      {/* Source link for discovery tenders */}
      {tender?.origin === 'discovery' && tender.source_url && (
        <a
          href={tender.source_url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-caption text-primary hover:underline"
        >
          Lihat sumber asli ({tender.source_name ?? 'link'}) ↗
        </a>
      )}

      {/* Re-analyze */}
      <div>
        <Button
          size="sm"
          variant="ghost"
          leftIcon={<Sparkles className="w-3.5 h-3.5" />}
          loading={runScore.isPending}
          onClick={handleAnalyze}
        >
          Analisa ulang
        </Button>
      </div>
    </div>
  )
}
