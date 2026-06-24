import { useState, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { CheckCircle2, AlertCircle, Info, AlertTriangle, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { subscribe, remove } from '../../lib/toast'
import type { ToastItem, ToastTone } from '../../lib/toast'

const toneConfig: Record<ToastTone, { icon: typeof CheckCircle2; classes: string }> = {
  success: { icon: CheckCircle2, classes: 'bg-success/10 border-success/30 text-success' },
  error:   { icon: AlertCircle,  classes: 'bg-danger/10 border-danger/30 text-danger' },
  warning: { icon: AlertTriangle, classes: 'bg-warning/10 border-warning/30 text-warning' },
  info:    { icon: Info,          classes: 'bg-info/10 border-info/30 text-info' },
}

function ToastItem({ item }: { item: ToastItem }) {
  const { icon: Icon, classes } = toneConfig[item.tone]
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        'flex items-start gap-3 px-4 py-3 rounded-card border shadow-subtle bg-surface min-w-64 max-w-sm',
        classes
      )}
    >
      <Icon className="w-5 h-5 flex-shrink-0 mt-0.5" aria-hidden="true" />
      <p className="flex-1 text-body text-fg">{item.message}</p>
      <button
        type="button"
        aria-label="Tutup notifikasi"
        onClick={() => remove(item.id)}
        className={cn(
          'flex-shrink-0 p-0.5 rounded hover:bg-surface-subtle transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary'
        )}
      >
        <X className="w-4 h-4 text-fg-muted" aria-hidden="true" />
      </button>
    </div>
  )
}

export default function ToastViewport() {
  const [items, setItems] = useState<ToastItem[]>([])

  useEffect(() => subscribe(setItems), [])

  if (items.length === 0) return null

  return createPortal(
    <div
      className="fixed top-4 right-4 z-[9999] flex flex-col gap-2 pointer-events-none"
      aria-label="Notifikasi"
    >
      {items.map((item) => (
        <div key={item.id} className="pointer-events-auto">
          <ToastItem item={item} />
        </div>
      ))}
    </div>,
    document.body
  )
}
