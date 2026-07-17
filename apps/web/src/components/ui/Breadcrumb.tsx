import { ChevronRight } from 'lucide-react'
import { Link } from 'react-router'
import { cn } from '../../lib/cn'

export interface BreadcrumbItem {
  label: string
  href?: string
}

export interface BreadcrumbProps {
  items: BreadcrumbItem[]
  className?: string
}

export default function Breadcrumb({ items, className }: BreadcrumbProps) {
  return (
    <nav aria-label="Breadcrumb" className={cn('flex items-center', className)}>
      <ol className="flex items-center gap-1">
        {items.map((item, i) => {
          const isLast = i === items.length - 1
          return (
            <li key={i} className="flex items-center gap-1">
              {i > 0 && (
                <ChevronRight
                  className="w-3.5 h-3.5 text-fg-subtle flex-shrink-0"
                  aria-hidden="true"
                />
              )}
              {isLast ? (
                <span
                  aria-current="page"
                  className="text-body text-fg font-medium"
                >
                  {item.label}
                </span>
              ) : item.href ? (
                <Link
                  to={item.href}
                  className={cn(
                    'text-body text-fg-muted hover:text-fg transition-colors duration-150',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1 rounded'
                  )}
                >
                  {item.label}
                </Link>
              ) : (
                <span className="text-body text-fg-muted">{item.label}</span>
              )}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
