import { useLocation, useNavigate } from 'react-router'
import { Menu as MenuIcon, Bell } from 'lucide-react'
import { cn } from '../lib/cn'
import Avatar from '../components/ui/Avatar'
import Popover from '../components/ui/Popover'
import Menu from '../components/ui/Menu'
import EmptyState from '../components/ui/EmptyState'
import { navItems } from './navItems'
import { useAuthStore } from '../store/auth'

export interface TopbarProps {
  collapsed: boolean
  onToggleSidebar: () => void
}

function usePageTitle(): string {
  const { pathname } = useLocation()
  for (const item of navItems) {
    // Sub-item dicek lebih dulu supaya judul lebih spesifik (mis. "Daftar
    // Proyek" alih-alih "Proyek Berjalan").
    const child = item.children?.find(
      (c) => pathname === c.path || pathname.startsWith(c.path + '/')
    )
    if (child) return child.label
    if (item.path === '/' ? pathname === '/' : pathname.startsWith(item.path)) {
      return item.label
    }
  }
  return 'SalesFlow'
}

export default function Topbar({ onToggleSidebar }: TopbarProps) {
  const title = usePageTitle()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  const avatarMenuItems = [
    { label: 'Profil', onSelect: () => {} },
    { label: 'Settings', onSelect: () => navigate('/settings') },
    {
      label: 'Keluar',
      onSelect: () => {
        logout()
        navigate('/login', { replace: true })
      },
      tone: 'danger' as const,
    },
  ]

  return (
    <header
      className={cn(
        'sticky top-0 z-10 flex items-center gap-3 h-14 px-5',
        'bg-surface/85 backdrop-blur-md border-b border-line shadow-subtle shrink-0'
      )}
    >
      {/* Toggle sidebar */}
      <button
        onClick={onToggleSidebar}
        aria-label="Toggle sidebar"
        className={cn(
          'p-1.5 rounded-btn text-fg-muted hover:bg-surface-subtle transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'
        )}
      >
        <MenuIcon className="w-4.5 h-4.5" aria-hidden="true" />
      </button>

      {/* Page title */}
      <h1 className="text-h3 font-semibold text-fg flex-1 truncate">{title}</h1>

      {/* Bell notification */}
      <Popover
        align="end"
        trigger={
          <button
            aria-label="Notifikasi"
            className={cn(
              'p-1.5 rounded-btn text-fg-muted hover:bg-surface-subtle transition-colors',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'
            )}
          >
            <Bell className="w-4.5 h-4.5" aria-hidden="true" />
          </button>
        }
        className="w-72"
      >
        <div className="px-3 py-2 border-b border-line">
          <p className="text-body font-semibold text-fg">Notifikasi</p>
        </div>
        <EmptyState
          title="Belum ada notifikasi"
          className="py-6"
        />
      </Popover>

      {/* Avatar menu */}
      <Popover
        align="end"
        trigger={<Avatar name={user?.name ?? 'U'} size="sm" />}
      >
        <div className="px-3 py-2 border-b border-line">
          <p className="text-body font-semibold text-fg">{user?.name ?? '—'}</p>
          <p className="text-caption text-fg-muted">{user?.email ?? ''}</p>
        </div>
        <Menu items={avatarMenuItems} />
      </Popover>
    </header>
  )
}
