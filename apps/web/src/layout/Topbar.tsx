import { useLocation, useNavigate } from 'react-router'
import { Menu as MenuIcon, Plus, Bell, Search } from 'lucide-react'
import { cn } from '../lib/cn'
import Button from '../components/ui/Button'
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
  const match = navItems.find((item) =>
    item.path === '/' ? pathname === '/' : pathname.startsWith(item.path)
  )
  return match?.label ?? 'SalesPilot'
}

export default function Topbar({ onToggleSidebar }: TopbarProps) {
  const title = usePageTitle()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  const newMenuItems = [
    { label: 'Tender baru', onSelect: () => {} },
    { label: 'Event baru', onSelect: () => {} },
    { label: 'Prospek baru', onSelect: () => {} },
  ]

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
        'sticky top-0 z-10 flex items-center gap-3 h-14 px-4',
        'bg-surface border-b border-line shrink-0'
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

      {/* Search ⌘K placeholder (P1) */}
      <button
        aria-label="Cari (⌘K)"
        className={cn(
          'hidden sm:flex items-center gap-2 px-3 h-8 rounded-btn border border-line',
          'text-body text-fg-muted hover:bg-surface-subtle transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'
        )}
        onClick={() => {}}
      >
        <Search className="w-3.5 h-3.5" aria-hidden="true" />
        <span className="text-caption">Cari…</span>
        <kbd className="text-caption bg-surface-subtle border border-line rounded px-1">⌘K</kbd>
      </button>

      {/* + New */}
      <Popover
        align="end"
        trigger={
          <Button size="sm" leftIcon={<Plus className="w-3.5 h-3.5" aria-hidden="true" />}>
            Baru
          </Button>
        }
      >
        <Menu items={newMenuItems} />
      </Popover>

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
