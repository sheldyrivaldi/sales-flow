import { useState, useId } from 'react'
import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

type Side = 'top' | 'bottom' | 'left' | 'right'

const positionClasses: Record<Side, string> = {
  top: 'bottom-full left-1/2 -translate-x-1/2 mb-1.5',
  bottom: 'top-full left-1/2 -translate-x-1/2 mt-1.5',
  left: 'right-full top-1/2 -translate-y-1/2 mr-1.5',
  right: 'left-full top-1/2 -translate-y-1/2 ml-1.5',
}

export interface TooltipProps {
  content: ReactNode
  children: ReactNode
  side?: Side
  className?: string
}

export default function Tooltip({ content, children, side = 'top', className }: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const id = useId()

  return (
    <span
      className={cn('relative inline-flex items-center', className)}
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
      onFocus={() => setVisible(true)}
      onBlur={() => setVisible(false)}
    >
      <span aria-describedby={visible ? id : undefined}>
        {children}
      </span>
      {visible && (
        <span
          id={id}
          role="tooltip"
          className={cn(
            'absolute z-50 whitespace-nowrap rounded-btn bg-fg text-surface text-caption px-2 py-1 shadow-subtle pointer-events-none',
            positionClasses[side]
          )}
        >
          {content}
        </span>
      )}
    </span>
  )
}
