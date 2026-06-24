import { NavLink } from 'react-router'
import { cn } from '../lib/cn'
import { AiBadge } from '../components/ui/Badge'
import Badge from '../components/ui/Badge'
import Tooltip from '../components/ui/Tooltip'
import { navItems } from './navItems'
import { useAuthStore } from '../store/auth'
import { can } from '../lib/rbac'

export interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

export default function Sidebar({ collapsed }: SidebarProps) {
  const role = useAuthStore((s) => s.user?.role)
  const visibleItems = navItems.filter((item) => !item.capability || can(role, item.capability))

  return (
    <aside
      className={cn(
        'flex flex-col h-full bg-surface border-r border-line shrink-0 transition-all duration-200',
        collapsed ? 'w-14' : 'w-56'
      )}
      aria-label="Navigasi utama"
    >
      {/* Logo */}
      <div
        className={cn(
          'flex items-center h-14 px-3 border-b border-line shrink-0',
          collapsed ? 'justify-center' : 'gap-2'
        )}
      >
        <span className="w-7 h-7 rounded-btn bg-primary flex items-center justify-center text-white font-bold text-caption shrink-0">
          S
        </span>
        {!collapsed && (
          <span className="font-semibold text-body text-fg truncate">SalesPilot</span>
        )}
      </div>

      {/* Nav items */}
      <nav className="flex-1 overflow-y-auto py-2">
        {visibleItems.map((item) => {
          const Icon = item.icon
          const linkContent = (isActive: boolean) => (
            <span
              className={cn(
                'flex items-center gap-2.5 w-full rounded-btn px-2.5 py-2 text-body font-medium transition-colors duration-150',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2',
                isActive
                  ? 'bg-primary/10 text-primary'
                  : 'text-fg-muted hover:bg-surface-subtle hover:text-fg',
                collapsed && 'justify-center px-0'
              )}
            >
              <Icon className="w-4.5 h-4.5 shrink-0" aria-hidden="true" />
              {!collapsed && (
                <>
                  <span className="flex-1 truncate">{item.label}</span>
                  {item.badge === 'ai' && <AiBadge />}
                  {item.badge === 'count' && (
                    <Badge tone="danger" appearance="solid" className="text-caption px-1.5 py-0">
                      0
                    </Badge>
                  )}
                </>
              )}
            </span>
          )

          return (
            <div key={item.path}>
              {item.dividerBefore && (
                <div className="border-t border-line mx-3 my-2" role="separator" />
              )}
              {collapsed ? (
                <Tooltip content={item.label} side="right">
                  <NavLink
                    to={item.path}
                    end={item.path === '/'}
                    className={cn('block px-1.5 py-0.5')}
                  >
                    {({ isActive }) => linkContent(isActive)}
                  </NavLink>
                </Tooltip>
              ) : (
                <NavLink
                  to={item.path}
                  end={item.path === '/'}
                  className={cn('block px-1.5 py-0.5')}
                >
                  {({ isActive }) => linkContent(isActive)}
                </NavLink>
              )}
            </div>
          )
        })}
      </nav>
    </aside>
  )
}
