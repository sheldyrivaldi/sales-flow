import { useCallback, useEffect, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '../../lib/cn'
import { useFloatingPosition } from '../../lib/useFloatingPosition'

type Align = 'start' | 'end'

export interface PopoverProps {
  trigger: ReactNode
  children: ReactNode
  align?: Align
  className?: string
}

export default function Popover({ trigger, children, align = 'end', className }: PopoverProps) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const pos = useFloatingPosition(triggerRef, open, align)

  const close = useCallback(() => setOpen(false), [])

  // Portaled to document.body (see useFloatingPosition's doc comment), so the
  // trigger's own ref no longer wraps the dropdown DOM-wise — outside-click
  // must check both trigger and portaled content explicitly.
  useEffect(() => {
    if (!open) return
    function handleMouseDown(e: MouseEvent) {
      const target = e.target as Node
      if (triggerRef.current?.contains(target)) return
      if (contentRef.current?.contains(target)) return
      close()
    }
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') close()
    }
    document.addEventListener('mousedown', handleMouseDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleMouseDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open, close])

  return (
    <div ref={triggerRef} className="relative inline-flex">
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
      {open &&
        pos &&
        createPortal(
          <div
            ref={contentRef}
            role="dialog"
            style={{
              position: 'fixed',
              top: pos.top,
              left: pos.left,
              transform: pos.alignEnd ? 'translateX(-100%)' : undefined,
            }}
            className={cn(
              'z-[9000] bg-surface border border-line rounded-card shadow-lg min-w-48',
              className
            )}
          >
            {children}
          </div>,
          document.body
        )}
    </div>
  )
}
