import type { KeyboardEvent } from 'react'
import { cn } from '../../lib/cn'

export interface ToggleProps {
  checked: boolean
  onChange: (next: boolean) => void
  label?: string
  disabled?: boolean
  size?: 'sm' | 'md'
  className?: string
}

export default function Toggle({
  checked,
  onChange,
  label,
  disabled,
  size = 'md',
  className,
}: ToggleProps) {
  function handleKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      if (!disabled) onChange(!checked)
    }
  }

  const trackSize = size === 'sm' ? 'w-8 h-4' : 'w-11 h-6'
  const thumbSize = size === 'sm' ? 'w-3 h-3' : 'w-4 h-4'
  const thumbTranslate = size === 'sm'
    ? (checked ? 'translate-x-4' : 'translate-x-0.5')
    : (checked ? 'translate-x-6' : 'translate-x-1')

  return (
    <label className={cn('inline-flex items-center gap-2 cursor-pointer', disabled && 'opacity-50 cursor-not-allowed', className)}>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => !disabled && onChange(!checked)}
        onKeyDown={handleKeyDown}
        className={cn(
          'relative inline-flex items-center rounded-pill transition-colors duration-200',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2',
          'disabled:cursor-not-allowed',
          trackSize,
          checked ? 'bg-primary' : 'bg-surface-subtle border border-line'
        )}
      >
        <span
          className={cn(
            'inline-block rounded-full bg-white shadow-subtle transition-transform duration-200',
            thumbSize,
            thumbTranslate
          )}
        />
      </button>
      {label && <span className="text-body text-fg select-none">{label}</span>}
    </label>
  )
}
