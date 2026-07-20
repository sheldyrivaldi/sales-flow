import { useState, useId, useRef, useLayoutEffect } from 'react'
import { createPortal } from 'react-dom'
import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

type Side = 'top' | 'bottom' | 'left' | 'right'

// Transform anchoring the tooltip box to the computed point per side, so the
// point sits on the trigger's edge midpoint and the box grows away from it.
const sideTransform: Record<Side, string> = {
  top: 'translate(-50%, -100%)',
  bottom: 'translate(-50%, 0)',
  left: 'translate(-100%, -50%)',
  right: 'translate(0, -50%)',
}

export interface TooltipProps {
  content: ReactNode
  children: ReactNode
  side?: Side
  className?: string
}

/** Tooltip rendered in a portal with fixed positioning derived from the
 * trigger's viewport rect. Portaling is deliberate: inline tooltips get
 * clipped by any ancestor with `overflow` other than `visible` (e.g. the
 * scrollable sidebar rail), and no z-index can escape a clipping/scroll
 * container. Fixed + portal sidesteps both. */
export default function Tooltip({ content, children, side = 'top', className }: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const [coords, setCoords] = useState<{ top: number; left: number } | null>(null)
  const triggerRef = useRef<HTMLSpanElement>(null)
  const id = useId()

  useLayoutEffect(() => {
    if (!visible || !triggerRef.current) {
      setCoords(null)
      return
    }
    const r = triggerRef.current.getBoundingClientRect()
    const gap = 8
    switch (side) {
      case 'right':
        setCoords({ top: r.top + r.height / 2, left: r.right + gap })
        break
      case 'left':
        setCoords({ top: r.top + r.height / 2, left: r.left - gap })
        break
      case 'bottom':
        setCoords({ top: r.bottom + gap, left: r.left + r.width / 2 })
        break
      default:
        setCoords({ top: r.top - gap, left: r.left + r.width / 2 })
    }
  }, [visible, side])

  return (
    <span
      ref={triggerRef}
      className={cn('relative inline-flex items-center', className)}
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
      onFocus={() => setVisible(true)}
      onBlur={() => setVisible(false)}
    >
      <span aria-describedby={visible ? id : undefined}>{children}</span>
      {visible &&
        coords &&
        createPortal(
          <span
            id={id}
            role="tooltip"
            style={{ position: 'fixed', top: coords.top, left: coords.left, transform: sideTransform[side] }}
            className="z-[100] whitespace-nowrap rounded-btn bg-fg text-surface text-caption px-2 py-1 shadow-subtle pointer-events-none"
          >
            {content}
          </span>,
          document.body,
        )}
    </span>
  )
}
