import { useState, useRef, useEffect } from 'react'
import type { ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { ChevronUp, ChevronDown, ChevronsUpDown, MoreVertical } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useFloatingPosition } from '../../lib/useFloatingPosition'
import Button from './Button'

export interface Column<T> {
  key: string
  header: ReactNode
  render?: (row: T) => ReactNode
  sortable?: boolean
  align?: 'left' | 'right' | 'center'
  width?: string
}

export interface KebabAction {
  label: string
  onClick: () => void
  danger?: boolean
}

type SortDir = 'asc' | 'desc' | 'none'

interface SortState {
  key: string
  dir: Exclude<SortDir, 'none'>
}

export interface TableProps<T> {
  columns: Column<T>[]
  data: T[]
  rowKey: (row: T) => string
  kebabActions?: (row: T) => KebabAction[]
  stickyHeader?: boolean
  loading?: boolean
  empty?: ReactNode
  /** Controlled sort. Omit for uncontrolled. */
  sort?: SortState
  onSortChange?: (s: SortState | null) => void
  /** Controlled pagination. Omit for uncontrolled. */
  page?: number
  total?: number
  pageSize?: number
  onPageChange?: (page: number) => void
}

function SortIcon({ dir }: { dir: SortDir }) {
  if (dir === 'asc') return <ChevronUp className="w-3.5 h-3.5 ml-1 inline-block" aria-hidden="true" />
  if (dir === 'desc') return <ChevronDown className="w-3.5 h-3.5 ml-1 inline-block" aria-hidden="true" />
  return <ChevronsUpDown className="w-3.5 h-3.5 ml-1 inline-block text-fg-subtle" aria-hidden="true" />
}

function RowMenu({ actions }: { actions: KebabAction[] }) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLUListElement>(null)
  // align="end" (right-aligned to the trigger) — portaled + fixed-positioned
  // (see useFloatingPosition) so the menu isn't clipped by Table's own
  // `overflow-auto` wrapper (needed for horizontal scroll on wide tables),
  // which a plain `absolute` dropdown always was.
  const pos = useFloatingPosition(triggerRef, open, 'end')

  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      const target = e.target as Node
      if (triggerRef.current?.contains(target)) return
      if (menuRef.current?.contains(target)) return
      setOpen(false)
    }
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open])

  return (
    <div ref={triggerRef} className="relative inline-block">
      <button
        type="button"
        aria-label="Opsi"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className={cn(
          'p-1.5 rounded-btn text-fg-muted hover:text-fg hover:bg-surface-subtle transition-colors',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1'
        )}
      >
        <MoreVertical className="w-4 h-4" aria-hidden="true" />
      </button>
      {open &&
        pos &&
        createPortal(
          <ul
            ref={menuRef}
            role="menu"
            style={{
              position: 'fixed',
              top: pos.top,
              left: pos.left,
              transform: pos.alignEnd ? 'translateX(-100%)' : undefined,
            }}
            className="z-[9000] min-w-35 bg-surface border border-line rounded-btn shadow-lg py-1"
          >
            {actions.map((action, i) => (
              <li key={i} role="none">
                <button
                  role="menuitem"
                  type="button"
                  onClick={() => {
                    // try/finally: an action that throws synchronously (e.g.
                    // an unavailable browser API) must never leave the
                    // popover stuck open.
                    try {
                      action.onClick()
                    } finally {
                      setOpen(false)
                    }
                  }}
                  className={cn(
                    'w-full text-left px-3 py-2 text-body hover:bg-surface-subtle transition-colors',
                    action.danger ? 'text-danger' : 'text-fg'
                  )}
                >
                  {action.label}
                </button>
              </li>
            ))}
          </ul>,
          document.body
        )}
    </div>
  )
}

export default function Table<T>({
  columns,
  data,
  rowKey,
  kebabActions,
  stickyHeader = false,
  loading = false,
  empty,
  sort: controlledSort,
  onSortChange,
  page: controlledPage,
  total: controlledTotal,
  pageSize = 10,
  onPageChange,
}: TableProps<T>) {
  const isControlledSort = controlledSort !== undefined
  const isControlledPage = controlledPage !== undefined

  const [internalSort, setInternalSort] = useState<SortState | null>(null)
  const [internalPage, setInternalPage] = useState(1)

  const activeSort = isControlledSort ? (controlledSort ?? null) : internalSort
  const activePage = isControlledPage ? controlledPage : internalPage

  function handleSort(col: Column<T>) {
    if (!col.sortable) return
    let next: SortState | null
    if (activeSort?.key === col.key) {
      next = activeSort.dir === 'asc' ? { key: col.key, dir: 'desc' } : null
    } else {
      next = { key: col.key, dir: 'asc' }
    }
    if (isControlledSort) {
      onSortChange?.(next)
    } else {
      setInternalSort(next)
    }
  }

  function handlePage(p: number) {
    if (isControlledPage) {
      onPageChange?.(p)
    } else {
      setInternalPage(p)
    }
  }

  // Local sort
  const sortedData = [...data]
  if (activeSort) {
    sortedData.sort((a, b) => {
      const aVal = String((a as Record<string, unknown>)[activeSort.key] ?? '')
      const bVal = String((b as Record<string, unknown>)[activeSort.key] ?? '')
      const cmp = aVal.localeCompare(bVal, 'id')
      return activeSort.dir === 'asc' ? cmp : -cmp
    })
  }

  // Local pagination
  const total = isControlledPage ? (controlledTotal ?? data.length) : data.length
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const pagedData = isControlledPage
    ? sortedData
    : sortedData.slice((activePage - 1) * pageSize, activePage * pageSize)

  const hasKebab = !!kebabActions

  return (
    <div className="flex flex-col gap-0">
      <div className="overflow-auto">
        <table className="w-full text-body text-fg border-collapse">
          <thead className={cn('bg-surface-subtle', stickyHeader && 'sticky top-0 z-10')}>
            <tr className="border-b border-line">
              {columns.map((col) => {
                const dir: SortDir =
                  activeSort?.key === col.key ? activeSort.dir : 'none'
                return (
                  <th
                    key={col.key}
                    scope="col"
                    style={col.width ? { width: col.width } : undefined}
                    className={cn(
                      'px-3 py-2.5 text-caption font-semibold text-fg-muted text-left select-none',
                      col.align === 'right' && 'text-right',
                      col.align === 'center' && 'text-center',
                      col.sortable && 'cursor-pointer hover:text-fg'
                    )}
                    onClick={() => col.sortable && handleSort(col)}
                    aria-sort={
                      dir === 'asc' ? 'ascending' : dir === 'desc' ? 'descending' : undefined
                    }
                  >
                    {col.header}
                    {col.sortable && <SortIcon dir={dir} />}
                  </th>
                )
              })}
              {hasKebab && <th scope="col" className="w-10" />}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <tr key={i} className="border-b border-line">
                  {columns.map((col) => (
                    <td key={col.key} className="px-3 py-3">
                      <div className="h-4 rounded bg-surface-subtle animate-pulse" />
                    </td>
                  ))}
                  {hasKebab && <td />}
                </tr>
              ))
            ) : pagedData.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length + (hasKebab ? 1 : 0)}
                  className="px-3 py-12 text-center text-fg-muted text-body"
                >
                  {empty ?? 'Tidak ada data.'}
                </td>
              </tr>
            ) : (
              pagedData.map((row) => (
                <tr
                  key={rowKey(row)}
                  className="border-b border-line hover:bg-primary-subtle transition-colors duration-150"
                >
                  {columns.map((col) => (
                    <td
                      key={col.key}
                      className={cn(
                        'px-3 py-3',
                        col.align === 'right' && 'text-right',
                        col.align === 'center' && 'text-center'
                      )}
                    >
                      {col.render
                        ? col.render(row)
                        : String((row as Record<string, unknown>)[col.key] ?? '')}
                    </td>
                  ))}
                  {hasKebab && (
                    <td className="px-2 py-2 text-right">
                      <RowMenu actions={kebabActions!(row)} />
                    </td>
                  )}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between px-3 py-2.5 border-t border-line">
          <span className="text-caption text-fg-muted">
            Hal {activePage} / {totalPages}
          </span>
          <div className="flex gap-2">
            <Button
              variant="ghost"
              size="sm"
              disabled={activePage <= 1}
              onClick={() => handlePage(activePage - 1)}
            >
              Sebelumnya
            </Button>
            <Button
              variant="ghost"
              size="sm"
              disabled={activePage >= totalPages}
              onClick={() => handlePage(activePage + 1)}
            >
              Berikutnya
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
