import { useState, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { AlertCircle, Info, AlertTriangle, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { subscribe, remove } from '../../lib/toast'
import type { ToastItem, ToastTone } from '../../lib/toast'

// Solid surface + left accent bar (bukan tint transparan) supaya toast selalu
// terbaca di atas konten apa pun. Warna tone dibawa oleh bar kiri, ikon, dan
// progress bar — bukan seluruh background.
const toneConfig: Record<
  ToastTone,
  { icon: typeof AlertCircle | null; iconClass: string; borderClass: string; progressClass: string }
> = {
  success: { icon: null,          iconClass: 'text-success', borderClass: 'border-l-success', progressClass: 'bg-success' },
  error:   { icon: AlertCircle,   iconClass: 'text-danger',  borderClass: 'border-l-danger',  progressClass: 'bg-danger' },
  warning: { icon: AlertTriangle, iconClass: 'text-warning', borderClass: 'border-l-warning', progressClass: 'bg-warning' },
  info:    { icon: Info,          iconClass: 'text-info',    borderClass: 'border-l-info',    progressClass: 'bg-info' },
}

/** Ikon check yang "digambar" (stroke draw-in) khusus toast success. */
function DrawnCheck() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      className="w-5 h-5 flex-shrink-0 mt-0.5 text-success"
      aria-hidden="true"
    >
      <circle
        cx="12" cy="12" r="10"
        stroke="currentColor" strokeWidth="2" pathLength={1}
        style={{ strokeDasharray: 1, strokeDashoffset: 1, animation: 'stroke-draw 300ms ease-out forwards' }}
      />
      <path
        d="M8 12.5l2.5 2.5L16 9.5"
        stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" pathLength={1}
        style={{ strokeDasharray: 1, strokeDashoffset: 1, animation: 'stroke-draw 250ms ease-out 200ms forwards' }}
      />
    </svg>
  )
}

function ToastCard({ item }: { item: ToastItem }) {
  const { icon: Icon, iconClass, borderClass, progressClass } = toneConfig[item.tone]

  // Rangkaian animasi dibangun per-tone: masuk (slide+fade dari kanan),
  // shake sekali untuk error, lalu keluar otomatis tepat sebelum store
  // menghapus item (duration - 200ms). Inline style karena delay-nya dinamis.
  const animations = [
    'toast-in 250ms ease-out',
    ...(item.tone === 'error' ? ['toast-shake 400ms ease-in-out 250ms'] : []),
    `toast-out 200ms ease-in ${Math.max(item.duration - 200, 300)}ms forwards`,
  ].join(', ')

  return (
    <div
      role="status"
      aria-live="polite"
      style={{ animation: animations }}
      className={cn(
        'relative overflow-hidden flex items-start gap-3 px-4 py-3 rounded-card border border-line border-l-4 shadow-lg bg-surface min-w-64 max-w-sm',
        borderClass
      )}
    >
      {item.tone === 'success' ? (
        <DrawnCheck />
      ) : (
        Icon && <Icon className={cn('w-5 h-5 flex-shrink-0 mt-0.5', iconClass)} aria-hidden="true" />
      )}
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

      {/* Progress bar auto-dismiss — menyusut linear selama durasi toast. */}
      <span
        aria-hidden="true"
        className={cn('absolute bottom-0 left-0 right-0 h-0.5 origin-left', progressClass)}
        style={{ animation: `toast-progress ${item.duration}ms linear forwards` }}
      />
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
          <ToastCard item={item} />
        </div>
      ))}
    </div>,
    document.body
  )
}
