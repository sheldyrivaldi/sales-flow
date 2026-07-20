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
  // Elemen panel disimpan sebagai STATE, bukan ref: perubahannya harus memicu
  // penghitungan ulang posisi, karena lebar panel baru terukur setelah ia
  // menempel di DOM — dan lebar itulah dasar penjepitan ke dalam viewport.
  const [contentEl, setContentEl] = useState<HTMLDivElement | null>(null)
  const pos = useFloatingPosition(triggerRef, open, align, contentEl)

  const close = useCallback(() => setOpen(false), [])

  // Portaled to document.body (see useFloatingPosition's doc comment), so the
  // trigger's own ref no longer wraps the dropdown DOM-wise — outside-click
  // must check both trigger and portaled content explicitly.
  useEffect(() => {
    if (!open) return
    function handleMouseDown(e: MouseEvent) {
      const target = e.target as Node
      if (triggerRef.current?.contains(target)) return
      if (contentEl?.contains(target)) return
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
  }, [open, close, contentEl])

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
            ref={setContentEl}
            role="dialog"
            style={{
              position: 'fixed',
              top: pos.top,
              left: pos.left,
              transform: pos.alignEnd ? 'translateX(-100%)' : undefined,
            }}
            className={cn(
              // max-w mengunci panel agar tidak pernah melewati tepi layar:
              // posisinya `fixed`, jadi tanpa batas ini isi yang lebar akan
              // terpotong di viewport sempit.
              'z-[9000] bg-surface border border-line rounded-card shadow-lg min-w-48',
              'max-w-[calc(100vw-1rem)] overflow-x-hidden',
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
