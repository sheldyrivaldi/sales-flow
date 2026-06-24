import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

export interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export default function EmptyState({
  icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center text-center gap-3 py-12 px-6', className)}>
      {icon && (
        <div
          className="w-14 h-14 rounded-pill bg-surface-subtle flex items-center justify-center text-fg-muted"
          aria-hidden="true"
        >
          {icon}
        </div>
      )}
      <div className="space-y-1">
        <h3 className="text-h3 font-semibold text-fg">{title}</h3>
        {description && (
          <p className="text-body text-fg-muted max-w-sm mx-auto">{description}</p>
        )}
      </div>
      {action && <div className="mt-1">{action}</div>}
    </div>
  )
}
