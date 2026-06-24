import { createPortal } from 'react-dom'
import { useId } from 'react'
import type { ReactNode } from 'react'
import { X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useOverlay } from '../../lib/useOverlay'

type DrawerSide = 'right' | 'left'

export interface DrawerProps {
  open: boolean
  onClose: () => void
  title?: ReactNode
  children?: ReactNode
  footer?: ReactNode
  side?: DrawerSide
  width?: string
  className?: string
}

export default function Drawer({
  open,
  onClose,
  title,
  children,
  footer,
  side = 'right',
  width = 'w-[420px]',
  className,
}: DrawerProps) {
  const titleId = useId()
  const panelRef = useOverlay(open, onClose)

  if (!open) return null

  return createPortal(
    <div className="fixed inset-0 z-50 flex">
      {/* Overlay */}
      <div
        className="absolute inset-0 bg-fg/40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef as React.RefObject<HTMLDivElement>}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        className={cn(
          'relative z-10 flex flex-col bg-surface shadow-subtle h-full max-w-full',
          width,
          side === 'right' ? 'ml-auto' : 'mr-auto',
          className
        )}
      >
        {/* Header */}
        {title && (
          <div className="flex items-center justify-between px-5 py-4 border-b border-line flex-shrink-0">
            <h2 id={titleId} className="text-h3 font-semibold text-fg">{title}</h2>
            <button
              type="button"
              aria-label="Tutup"
              onClick={onClose}
              className={cn(
                'p-1.5 rounded-btn text-fg-muted hover:text-fg hover:bg-surface-subtle transition-colors',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1'
              )}
            >
              <X className="w-5 h-5" aria-hidden="true" />
            </button>
          </div>
        )}

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {children}
        </div>

        {/* Footer */}
        {footer && (
          <div className="px-5 py-4 border-t border-line flex justify-end gap-3 flex-shrink-0">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body
  )
}
