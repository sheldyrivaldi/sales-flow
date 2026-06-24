import { cn } from '../../lib/cn'

type SkeletonVariant = 'text' | 'rect' | 'circle'

export interface SkeletonProps {
  variant?: SkeletonVariant
  className?: string
}

export default function Skeleton({ variant = 'rect', className }: SkeletonProps) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        'animate-pulse bg-surface-subtle',
        variant === 'circle' ? 'rounded-pill' : 'rounded-btn',
        variant === 'text' ? 'h-4' : variant === 'rect' ? 'h-10' : 'w-10 h-10',
        className
      )}
    />
  )
}

export interface SkeletonTextProps {
  lines?: number
  className?: string
}

export function SkeletonText({ lines = 3, className }: SkeletonTextProps) {
  return (
    <div className={cn('flex flex-col gap-2', className)} aria-hidden="true">
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          variant="text"
          className={i === lines - 1 && lines > 1 ? 'w-3/4' : 'w-full'}
        />
      ))}
    </div>
  )
}
