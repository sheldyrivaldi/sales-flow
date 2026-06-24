import type { ReactNode } from 'react'
import { Sparkles } from 'lucide-react'
import { cn } from '../../lib/cn'
import { toneClasses, actionColor, scoreColor } from '../../lib/score'
import type { Tone, RecommendedAction } from '../../lib/score'

export type Appearance = 'soft' | 'solid'

export interface BadgeProps {
  tone: Tone
  appearance?: Appearance
  children: ReactNode
  className?: string
}

export default function Badge({
  tone,
  appearance = 'soft',
  children,
  className,
}: BadgeProps) {
  const tc = toneClasses(tone)
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 px-2 py-0.5 rounded-pill text-caption font-medium',
        appearance === 'soft'
          ? `${tc.bgSoft} ${tc.text}`
          : `${tc.bg} text-white`,
        className
      )}
    >
      {children}
    </span>
  )
}

export function AiBadge({ className }: { className?: string }) {
  return (
    <Badge tone="accent" appearance="soft" className={cn('gap-0.5', className)}>
      <Sparkles className="w-3 h-3" aria-hidden="true" />
      AI
    </Badge>
  )
}

export interface ActionBadgeProps {
  action: RecommendedAction
  appearance?: Appearance
  className?: string
}

export function ActionBadge({ action, appearance = 'soft', className }: ActionBadgeProps) {
  return (
    <Badge tone={actionColor(action)} appearance={appearance} className={className}>
      {action}
    </Badge>
  )
}

export interface ScoreBadgeProps {
  score: number
  appearance?: Appearance
  className?: string
}

export function ScoreBadge({ score, appearance = 'soft', className }: ScoreBadgeProps) {
  return (
    <Badge tone={scoreColor(score)} appearance={appearance} className={className}>
      <span className="tabular-nums font-semibold">{score}</span>
    </Badge>
  )
}

type TenderStage = 'IDENTIFIED' | 'QUALIFYING' | 'BIDDING' | 'SUBMITTED' | 'WON' | 'LOST'
type ProspectStage = 'NEW' | 'QUALIFIED' | 'ENGAGED' | 'PROPOSAL' | 'WON' | 'LOST'

const stageClasses: Record<TenderStage | ProspectStage, string> = {
  IDENTIFIED: 'bg-surface-subtle text-fg-muted',
  NEW:        'bg-surface-subtle text-fg-muted',
  QUALIFYING: 'bg-info/10 text-info',
  QUALIFIED:  'bg-info/10 text-info',
  BIDDING:    'bg-primary/10 text-primary',
  ENGAGED:    'bg-primary/10 text-primary',
  SUBMITTED:  'bg-accent/10 text-accent',
  PROPOSAL:   'bg-accent/10 text-accent',
  WON:        'bg-success/10 text-success',
  LOST:       'bg-danger/10 text-danger',
}

export interface StagePillProps {
  stage: TenderStage | ProspectStage
  className?: string
}

export function StagePill({ stage, className }: StagePillProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2 py-0.5 rounded-pill text-caption font-medium',
        stageClasses[stage],
        className
      )}
    >
      {stage}
    </span>
  )
}
