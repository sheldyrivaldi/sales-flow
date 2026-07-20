import { Star } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface StarRatingProps {
  value: number
  /** onChange menjadikan bintang interaktif; tanpa ini = display-only. */
  onChange?: (value: number) => void
  size?: 'sm' | 'lg'
  className?: string
}

/** Rating bintang 1-5 — dipakai form feedback publik (interaktif) dan
 * tampilan admin (read-only). */
export default function StarRating({ value, onChange, size = 'sm', className }: StarRatingProps) {
  const px = size === 'lg' ? 'w-8 h-8' : 'w-4 h-4'
  return (
    <div className={cn('flex items-center gap-1', className)} role={onChange ? 'radiogroup' : undefined}>
      {[1, 2, 3, 4, 5].map((n) => {
        const filled = n <= value
        const star = (
          <Star
            className={cn(
              px,
              'transition-colors',
              filled ? 'fill-amber-400 text-amber-400' : 'fill-none text-line-strong',
              onChange && 'hover:text-amber-400'
            )}
            aria-hidden="true"
          />
        )
        if (!onChange) return <span key={n}>{star}</span>
        return (
          <button
            key={n}
            type="button"
            role="radio"
            aria-checked={filled}
            aria-label={`${n} bintang`}
            onClick={() => onChange(n)}
            className="focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded"
          >
            {star}
          </button>
        )
      })}
    </div>
  )
}
