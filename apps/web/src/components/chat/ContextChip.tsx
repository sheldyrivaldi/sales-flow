import { X } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface ContextChipProps {
  label: string
  onClear: () => void
  className?: string
}

export default function ContextChip({ label, onClear, className }: ContextChipProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 text-caption font-medium',
        'px-2.5 py-1 rounded-pill border border-accent/30 bg-accent/5 text-accent',
        className,
      )}
    >
      Konteks: {label}
      <button
        type="button"
        aria-label="Hapus konteks"
        onClick={onClear}
        className="ml-0.5 hover:text-accent/70 transition-colors focus-visible:outline-none"
      >
        <X className="w-3 h-3" aria-hidden="true" />
      </button>
    </span>
  )
}
