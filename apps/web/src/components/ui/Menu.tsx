import { useRef } from 'react'
import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'
import type { Tone } from '../../lib/score'

export interface MenuItem {
  label: string
  icon?: ReactNode
  onSelect: () => void
  tone?: Tone
  disabled?: boolean
}

export interface MenuProps {
  items: MenuItem[]
  className?: string
}

const toneText: Partial<Record<Tone, string>> = {
  danger: 'text-danger',
  success: 'text-success',
  warning: 'text-warning',
  info: 'text-info',
  accent: 'text-accent',
}

export default function Menu({ items, className }: MenuProps) {
  const listRef = useRef<HTMLUListElement | null>(null)

  function handleKeyDown(e: React.KeyboardEvent, index: number) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      const next = listRef.current?.querySelectorAll<HTMLButtonElement>('button:not([disabled])')
      if (next) {
        const arr = Array.from(next)
        const nextIndex = arr.indexOf(e.currentTarget as HTMLButtonElement) + 1
        arr[nextIndex % arr.length]?.focus()
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      const all = listRef.current?.querySelectorAll<HTMLButtonElement>('button:not([disabled])')
      if (all) {
        const arr = Array.from(all)
        const curr = arr.indexOf(e.currentTarget as HTMLButtonElement)
        arr[(curr - 1 + arr.length) % arr.length]?.focus()
      }
    }
    void index
  }

  return (
    <ul ref={listRef} role="menu" className={cn('py-1', className)}>
      {items.map((item, i) => (
        <li key={i} role="none">
          <button
            role="menuitem"
            disabled={item.disabled}
            onClick={item.onSelect}
            onKeyDown={(e) => handleKeyDown(e, i)}
            className={cn(
              'w-full flex items-center gap-2 px-3 py-2 text-body transition-colors duration-150',
              'hover:bg-surface-subtle disabled:opacity-50 disabled:cursor-not-allowed',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1',
              item.tone ? toneText[item.tone] : 'text-fg'
            )}
          >
            {item.icon && (
              <span className="w-4 h-4 shrink-0" aria-hidden="true">
                {item.icon}
              </span>
            )}
            {item.label}
          </button>
        </li>
      ))}
    </ul>
  )
}
