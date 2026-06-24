import type { ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'
import { cn } from '../../lib/cn'

type Severity = 'warning' | 'danger'

const severityClasses: Record<Severity, string> = {
  warning: 'bg-warning/10 text-warning border-warning/20',
  danger:  'bg-danger/10 text-danger border-danger/20',
}

export interface RiskFlagProps {
  label: string
  severity?: Severity
  className?: string
}

export default function RiskFlag({ label, severity = 'warning', className }: RiskFlagProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 px-2 py-0.5 rounded-pill text-caption font-medium border',
        severityClasses[severity],
        className
      )}
    >
      <AlertTriangle className="w-3 h-3 shrink-0" aria-hidden="true" />
      {label}
    </span>
  )
}

export interface RiskFlagListProps {
  items: { label: string; severity?: Severity }[]
  className?: string
}

export function RiskFlagList({ items, className }: RiskFlagListProps) {
  if (items.length === 0) return null
  return (
    <div className={cn('flex flex-wrap gap-1.5', className)}>
      {items.map((item, i) => (
        <RiskFlag key={i} label={item.label} severity={item.severity} />
      ))}
    </div>
  )
}

export interface StreamingRiskFlagProps {
  children: ReactNode
}
