import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '../../lib/cn'

// Section identity: each thematic group gets a distinct colored icon chip so
// a page has visual rhythm without being "all one color". Emerald (brand) ·
// Teal (AI) · Amber (automation) · Sky (data) — plus neutral slate. Chip
// colors use the semantic subtle/strong ramps from tokens.css for contrast.
export type SectionTone = 'slate' | 'emerald' | 'ai' | 'amber' | 'sky'

export const chipTone: Record<SectionTone, string> = {
  slate: 'bg-surface-subtle text-fg-muted',
  emerald: 'bg-primary-subtle text-primary',
  ai: 'bg-accent-subtle text-accent-hover',
  amber: 'bg-warning-subtle text-warning-strong',
  sky: 'bg-info-subtle text-info-strong',
}

/** Page-level section header: colored icon chip + title + optional description
 *  and a right-aligned slot (badge, actions). */
export function SectionHeader({
  icon: Icon,
  title,
  description,
  tone = 'slate',
  right,
}: {
  icon: LucideIcon
  title: string
  description?: string
  tone?: SectionTone
  right?: ReactNode
}) {
  return (
    <div className="flex items-start justify-between gap-3">
      <div className="flex items-start gap-3 min-w-0">
        <span className={cn('flex h-10 w-10 shrink-0 items-center justify-center rounded-xl', chipTone[tone])}>
          <Icon className="h-5 w-5" aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <h2 className="text-h3 font-semibold text-fg leading-tight">{title}</h2>
          {description && <p className="text-caption text-fg-muted mt-0.5 max-w-2xl">{description}</p>}
        </div>
      </div>
      {right}
    </div>
  )
}

/** Compact subsection label: small colored chip + uppercase caption. */
export function GroupLabel({ icon: Icon, title, tone }: { icon: LucideIcon; title: string; tone: SectionTone }) {
  return (
    <div className="flex items-center gap-2">
      <span className={cn('flex h-6 w-6 shrink-0 items-center justify-center rounded-md', chipTone[tone])}>
        <Icon className="h-3.5 w-3.5" aria-hidden="true" />
      </span>
      <h3 className="text-caption font-semibold uppercase tracking-wide text-fg-muted">{title}</h3>
    </div>
  )
}
