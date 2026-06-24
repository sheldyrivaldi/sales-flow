import { useState, useCallback } from 'react'
import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { useClickOutside } from '../../lib/useClickOutside'

type Align = 'start' | 'end'

export interface PopoverProps {
  trigger: ReactNode
  children: ReactNode
  align?: Align
  className?: string
}

export default function Popover({ trigger, children, align = 'end', className }: PopoverProps) {
  const [open, setOpen] = useState(false)

  const close = useCallback(() => setOpen(false), [])
  const containerRef = useClickOutside<HTMLDivElement>(close)

  return (
    <div ref={containerRef} className="relative inline-flex">
      <div
        onClick={() => setOpen((prev) => !prev)}
        aria-expanded={open}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            setOpen((prev) => !prev)
          }
        }}
        className="focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 rounded-btn"
      >
        {trigger}
      </div>
      {open && (
        <div
          role="dialog"
          className={cn(
            'absolute top-full mt-1 z-50 bg-surface border border-line rounded-card shadow-subtle min-w-48',
            align === 'end' ? 'right-0' : 'left-0',
            className
          )}
        >
          {children}
        </div>
      )}
    </div>
  )
}
