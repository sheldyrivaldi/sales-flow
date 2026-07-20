import { useState } from 'react'
import { NavLink, useLocation } from 'react-router'
import { ChevronRight } from 'lucide-react'
import { cn } from '../lib/cn'
import { AiBadge } from '../components/ui/Badge'
import Badge from '../components/ui/Badge'
import Tooltip from '../components/ui/Tooltip'
import { LogoBadge, LogoWordmark } from '../components/Logo'
import { navItems } from './navItems'
import type { NavItem } from './navItems'
import { useAuthStore } from '../store/auth'
import { can } from '../lib/rbac'

export interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

function isPathActive(pathname: string, path: string, exact: boolean) {
  return exact ? pathname === path : pathname === path || pathname.startsWith(path + '/')
}

// Shared active/inactive treatment for every nav row (top-level, collapsed
// leaf, and expandable parent) so they read as one coherent system. Active =
// soft emerald tint + emerald text + semibold; inactive = muted slate that
// warms to full fg on hover. The left accent bar is added separately by the
// caller (it doesn't apply in the collapsed rail).
const rowBase =
  'flex items-center gap-2.5 w-full rounded-btn px-2.5 py-2 text-body transition-all duration-150 ' +
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 focus-visible:ring-offset-1'
const rowActive = 'bg-primary-subtle text-primary font-semibold'
const rowInactive = 'text-fg-muted font-medium hover:bg-surface-subtle hover:text-fg'

// ActiveBar is the emerald indicator on the left edge of an active pill.
function ActiveBar() {
  return (
    <span
      className="absolute left-0 top-1/2 -translate-y-1/2 h-5 w-1 rounded-r-full bg-primary"
      aria-hidden="true"
    />
  )
}

export default function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const role = useAuthStore((s) => s.user?.role)
  const { pathname } = useLocation()
  // Filter item DAN sub-item berdasarkan capability; parent yang seluruh
  // anaknya tersaring ikut hilang.
  const visibleItems = navItems
    .filter((item) => !item.capability || can(role, item.capability))
    .map((item) =>
      item.children
        ? { ...item, children: item.children.filter((c) => !c.capability || can(role, c.capability)) }
        : item
    )
    .filter((item) => !item.children || item.children.length > 0)

  // Explicit open/closed overrides from clicking a parent row, keyed by item
  // path. Absent = "follow the route" (expanded iff a child is active).
  const [overrides, setOverrides] = useState<Record<string, boolean>>({})
  const [trackedActive, setTrackedActive] = useState<Record<string, boolean>>({})

  function isExpanded(item: NavItem): boolean {
    const branchActive = item.children?.some((c) => isPathActive(pathname, c.path, false)) ?? false
    if (trackedActive[item.path] !== branchActive) {
      if (item.path in overrides) {
        setOverrides((prev) => {
          const next = { ...prev }
          delete next[item.path]
          return next
        })
      }
      setTrackedActive((prev) => ({ ...prev, [item.path]: branchActive }))
      return branchActive
    }
    return overrides[item.path] ?? branchActive
  }

  function toggle(item: NavItem) {
    setOverrides((prev) => ({ ...prev, [item.path]: !isExpanded(item) }))
  }

  return (
    <aside
      className={cn(
        'flex flex-col h-full bg-surface border-r border-line shrink-0 transition-all duration-200',
        collapsed ? 'w-14' : 'w-60'
      )}
      aria-label="Navigasi utama"
    >
      {/* Logo */}
      <div
        className={cn(
          'flex items-center h-14 px-3 border-b border-line shrink-0',
          collapsed ? 'justify-center' : 'gap-2.5'
        )}
      >
        <LogoBadge size={32} />
        {!collapsed && <LogoWordmark className="text-body" />}
      </div>

      {/* Nav items. overflow-x-hidden is required alongside overflow-y-auto:
          otherwise the browser computes overflow-x to `auto` and a sub-pixel
          overflow paints a phantom horizontal scrollbar in the collapsed rail. */}
      <nav className="flex-1 overflow-y-auto overflow-x-hidden px-2 py-3">
        {visibleItems.map((item) => {
          const Icon = item.icon
          const hasChildren = !!item.children && item.children.length > 0
          const linkContent = (isActive: boolean, withBar = true) => (
            <span className={cn('relative', rowBase, isActive ? rowActive : rowInactive, collapsed && 'justify-center px-0')}>
              {isActive && withBar && !collapsed && <ActiveBar />}
              <Icon className={cn('w-4.5 h-4.5 shrink-0', isActive && 'text-primary')} aria-hidden="true" />
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

          // Collapsed rail (or any leaf item): no room for a nested tree.
          if (!hasChildren || collapsed) {
            const groupActive = hasChildren
              ? item.children!.some((c) => isPathActive(pathname, c.path, false))
              : undefined

            // A collapsed parent-with-children can't show its submenu inline,
            // so clicking it expands the whole rail and opens that group —
            // rather than silently jumping to a child page. Leaf items just
            // navigate.
            const row =
              collapsed && hasChildren ? (
                <button
                  type="button"
                  aria-label={item.label}
                  aria-expanded={false}
                  onClick={() => {
                    setOverrides((prev) => ({ ...prev, [item.path]: true }))
                    onToggle()
                  }}
                  className="block w-full py-0.5 focus-visible:outline-none"
                >
                  {linkContent(groupActive ?? false)}
                </button>
              ) : (
                <NavLink to={item.path} end={item.path === '/'} className="block py-0.5">
                  {({ isActive }) => linkContent(isActive)}
                </NavLink>
              )

            return (
              <div key={item.path}>
                {item.dividerBefore && (
                  <div className="border-t border-line mx-2 my-2.5" role="separator" />
                )}
                {collapsed ? (
                  // w-full + justify-center: the Tooltip wrapper is inline-flex
                  // and would otherwise shrink to content and left-align the
                  // icon in the rail. Full width + centering keeps every
                  // collapsed icon symmetric on the vertical axis.
                  <Tooltip content={item.label} side="right" className="w-full justify-center">
                    {row}
                  </Tooltip>
                ) : (
                  row
                )}
              </div>
            )
          }

          // Expandable parent with nested sub-items.
          const expanded = isExpanded(item)
          const parentActive =
            isPathActive(pathname, item.path, false) ||
            (item.children?.some((c) => isPathActive(pathname, c.path, false)) ?? false)

          return (
            <div key={item.path}>
              {item.dividerBefore && (
                <div className="border-t border-line mx-2 my-2.5" role="separator" />
              )}
              <button
                type="button"
                onClick={() => toggle(item)}
                aria-expanded={expanded}
                className={cn('relative py-0.5 w-full', 'focus-visible:outline-none')}
              >
                <span className={cn('relative', rowBase, parentActive ? rowActive : rowInactive)}>
                  {parentActive && <ActiveBar />}
                  <Icon className={cn('w-4.5 h-4.5 shrink-0', parentActive && 'text-primary')} aria-hidden="true" />
                  <span className="flex-1 truncate text-left">{item.label}</span>
                  <ChevronRight
                    className={cn('w-3.5 h-3.5 shrink-0 transition-transform duration-200', expanded && 'rotate-90')}
                    aria-hidden="true"
                  />
                </span>
              </button>

              {expanded && (
                <div className="mt-0.5 mb-0.5 flex flex-col gap-0.5 pl-5 relative">
                  {/* Guide rail connecting nested children to the parent. */}
                  <span className="absolute left-4 top-1 bottom-1 w-px bg-line" aria-hidden="true" />
                  {item.children!.map((child) => (
                    <NavLink
                      key={child.path}
                      to={child.path}
                      className={({ isActive }) =>
                        cn(
                          'flex items-center rounded-btn px-2.5 py-1.5 text-body transition-all duration-150',
                          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 focus-visible:ring-offset-1',
                          isActive
                            ? 'bg-primary-subtle text-primary font-semibold'
                            : 'text-fg-muted font-medium hover:bg-surface-subtle hover:text-fg'
                        )
                      }
                    >
                      <span className="truncate">{child.label}</span>
                    </NavLink>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </nav>
    </aside>
  )
}
