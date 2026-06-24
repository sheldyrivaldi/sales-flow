import { cn } from '../../lib/cn'

type Size = 'sm' | 'md' | 'lg'

const sizeClasses: Record<Size, string> = {
  sm: 'w-7 h-7 text-caption',
  md: 'w-9 h-9 text-body',
  lg: 'w-12 h-12 text-h3',
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase()
  return name.slice(0, 2).toUpperCase()
}

export interface AvatarProps {
  name: string
  src?: string
  size?: Size
  className?: string
}

export default function Avatar({ name, src, size = 'md', className }: AvatarProps) {
  const base = cn(
    'inline-flex items-center justify-center rounded-pill flex-shrink-0 font-semibold select-none',
    sizeClasses[size],
    className
  )

  if (src) {
    return (
      <img
        src={src}
        alt={name}
        className={cn(base, 'object-cover')}
      />
    )
  }

  return (
    <span
      aria-label={name}
      className={cn(base, 'bg-primary/10 text-primary')}
    >
      {initials(name)}
    </span>
  )
}
