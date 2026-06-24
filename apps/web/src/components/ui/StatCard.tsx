import type { ReactNode } from 'react'
import { TrendingUp, TrendingDown } from 'lucide-react'
import { cn } from '../../lib/cn'
import Card, { CardBody } from './Card'

export interface StatCardProps {
  label: string
  value: ReactNode
  icon?: ReactNode
  delta?: {
    value: string
    trend: 'up' | 'down'
  }
  hint?: string
  className?: string
}

export default function StatCard({ label, value, icon, delta, hint, className }: StatCardProps) {
  return (
    <Card className={cn('min-w-48', className)}>
      <CardBody className="flex flex-col gap-2">
        <div className="flex items-start justify-between gap-2">
          <span className="text-caption font-medium text-fg-muted">{label}</span>
          {icon && (
            <span className="text-fg-muted" aria-hidden="true">{icon}</span>
          )}
        </div>
        <div className="text-h2 font-semibold tabular-nums text-fg leading-tight">
          {value}
        </div>
        {(delta || hint) && (
          <div className="flex items-center gap-2">
            {delta && (
              <span
                className={cn(
                  'inline-flex items-center gap-0.5 text-caption font-medium',
                  delta.trend === 'up' ? 'text-success' : 'text-danger'
                )}
              >
                {delta.trend === 'up' ? (
                  <TrendingUp className="w-3.5 h-3.5" aria-hidden="true" />
                ) : (
                  <TrendingDown className="w-3.5 h-3.5" aria-hidden="true" />
                )}
                {delta.value}
              </span>
            )}
            {hint && (
              <span className="text-caption text-fg-subtle">{hint}</span>
            )}
          </div>
        )}
      </CardBody>
    </Card>
  )
}
