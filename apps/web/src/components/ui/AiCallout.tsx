import { useState } from 'react'
import type { ReactNode } from 'react'
import { Sparkles, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface AiCalloutProps {
  title?: string
  children?: ReactNode
  meta?: string
  reason?: ReactNode
  className?: string
}

export default function AiCallout({ title, children, meta, reason, className }: AiCalloutProps) {
  const [reasonOpen, setReasonOpen] = useState(false)

  return (
    <div
      className={cn(
        'rounded-card border border-accent/20 bg-accent/5 p-4',
        className
      )}
    >
      {/* Header */}
      <div className="flex items-start gap-2">
        <Sparkles
          className="w-4 h-4 text-accent mt-0.5 shrink-0"
          aria-hidden="true"
        />
        <div className="flex-1 min-w-0">
          {title && (
            <p className="text-body font-semibold text-fg">{title}</p>
          )}
          {children && (
            <div className="text-body text-fg-muted mt-0.5">{children}</div>
          )}
          {meta && (
            <p className="text-caption text-fg-subtle mt-1">{meta}</p>
          )}
        </div>
      </div>

      {/* "Lihat alasan" disclosure */}
      {reason && (
        <div className="mt-3">
          <button
            type="button"
            onClick={() => setReasonOpen((o) => !o)}
            className={cn(
              'inline-flex items-center gap-1 text-caption font-medium text-accent',
              'hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-1 rounded'
            )}
            aria-expanded={reasonOpen}
          >
            {reasonOpen ? (
              <>
                <ChevronUp className="w-3.5 h-3.5" aria-hidden="true" />
                Sembunyikan alasan
              </>
            ) : (
              <>
                <ChevronDown className="w-3.5 h-3.5" aria-hidden="true" />
                Lihat alasan
              </>
            )}
          </button>
          {reasonOpen && (
            <div className="mt-2 text-body text-fg-muted border-t border-accent/10 pt-2">
              {reason}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
