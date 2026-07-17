import { useLocation, useNavigate } from 'react-router'
import { ChevronLeft } from 'lucide-react'
import Breadcrumb from '../components/ui/Breadcrumb'
import { cn } from '../lib/cn'
import { buildTrail } from './breadcrumbTrail'

/** Thin bar under the Topbar: breadcrumb trail + a Back button that always
 * navigates to the parent crumb's path — never `navigate(-1)`, which would
 * replay arbitrary browser history instead of the app's actual hierarchy.
 * Renders nothing on top-level pages (a lone crumb would just duplicate the
 * Topbar's page title). */
export default function Breadcrumbs() {
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const trail = buildTrail(pathname)

  if (trail.length < 2) return null

  const parent = trail[trail.length - 2]

  return (
    <div
      className={cn(
        'flex items-center gap-3 h-10 px-4 shrink-0',
        'bg-surface border-b border-line'
      )}
    >
      <button
        type="button"
        onClick={() => navigate(parent.href ?? '/')}
        aria-label={`Kembali ke ${parent.label}`}
        className={cn(
          'flex items-center gap-1 p-1 -ml-1 rounded-btn text-fg-muted hover:bg-surface-subtle hover:text-fg transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'
        )}
      >
        <ChevronLeft className="w-4 h-4" aria-hidden="true" />
      </button>
      <Breadcrumb items={trail} />
    </div>
  )
}
