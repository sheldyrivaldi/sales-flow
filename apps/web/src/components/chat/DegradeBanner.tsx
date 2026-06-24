import { AlertTriangle, RefreshCw } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useChatDegradeStore } from '../../store/chat'

export interface DegradeBannerProps {
  className?: string
}

export default function DegradeBanner({ className }: DegradeBannerProps) {
  const { degraded, setDegraded } = useChatDegradeStore()

  if (!degraded) return null

  return (
    <div
      role="alert"
      className={cn(
        'flex items-start gap-3 px-4 py-3 rounded-card',
        'border border-danger/30 bg-danger/5 text-danger',
        className,
      )}
    >
      <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" aria-hidden="true" />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium">Agent tidak tersedia saat ini.</p>
        <p className="text-caption text-danger/80 mt-0.5">
          Semua data &amp; fitur CRUD tetap berfungsi normal. Hanya fitur AI yang terdampak.
        </p>
      </div>
      <button
        type="button"
        onClick={() => setDegraded(false)}
        aria-label="Coba lagi"
        className={cn(
          'shrink-0 flex items-center gap-1 text-caption font-medium px-2.5 py-1 rounded-btn',
          'border border-danger/30 hover:bg-danger/10 transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger focus-visible:ring-offset-1',
        )}
      >
        <RefreshCw className="w-3 h-3" aria-hidden="true" />
        Coba lagi
      </button>
    </div>
  )
}
