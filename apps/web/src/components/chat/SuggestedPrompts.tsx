import { cn } from '../../lib/cn'

const PROMPTS = [
  'Tender prioritas minggu ini?',
  'Ringkas pipeline',
  'Buatkan playbook prospek teratas',
  'Kenapa tender X skornya rendah?',
  'Cari tender baru sekarang',
]

export interface SuggestedPromptsProps {
  onSelect: (prompt: string) => void
  disabled?: boolean
  className?: string
}

export default function SuggestedPrompts({ onSelect, disabled, className }: SuggestedPromptsProps) {
  return (
    <div className={cn('flex flex-wrap gap-2', className)}>
      {PROMPTS.map((p) => (
        <button
          key={p}
          type="button"
          disabled={disabled}
          onClick={() => onSelect(p)}
          className={cn(
            'text-caption px-3 py-1.5 rounded-pill border border-primary/40 text-primary',
            'hover:bg-primary/5 transition-colors',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1',
            'disabled:opacity-40 disabled:cursor-not-allowed',
          )}
        >
          {p}
        </button>
      ))}
    </div>
  )
}
