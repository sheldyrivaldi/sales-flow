import { Check } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface StepItem {
  id: string
  label: string
}

export interface StepperProps {
  steps: StepItem[]
  current: number
  className?: string
}

export default function Stepper({ steps, current, className }: StepperProps) {
  return (
    <nav aria-label="Langkah" className={cn('flex items-center', className)}>
      {steps.map((step, i) => {
        const isDone = i < current
        const isActive = i === current
        const isUpcoming = i > current
        const isLast = i === steps.length - 1

        return (
          <div key={step.id} className="flex items-center flex-1 last:flex-none">
            {/* Circle */}
            <div className="flex flex-col items-center">
              <div
                aria-current={isActive ? 'step' : undefined}
                className={cn(
                  'w-8 h-8 rounded-pill flex items-center justify-center text-caption font-semibold transition-colors',
                  isDone && 'bg-primary text-white',
                  isActive && 'border-2 border-primary text-primary bg-surface',
                  isUpcoming && 'bg-surface-subtle text-fg-muted'
                )}
              >
                {isDone ? (
                  <Check className="w-4 h-4" aria-hidden="true" />
                ) : (
                  <span aria-hidden="true">{i + 1}</span>
                )}
              </div>
              <span
                className={cn(
                  'mt-1 text-caption text-center max-w-20 leading-tight',
                  isActive ? 'text-fg font-medium' : isDone ? 'text-fg-muted' : 'text-fg-subtle'
                )}
              >
                {step.label}
              </span>
            </div>

            {/* Connector */}
            {!isLast && (
              <div
                aria-hidden="true"
                className={cn(
                  'flex-1 h-0.5 mx-2 mb-5 transition-colors',
                  i < current ? 'bg-primary' : 'bg-line'
                )}
              />
            )}
          </div>
        )
      })}
    </nav>
  )
}
