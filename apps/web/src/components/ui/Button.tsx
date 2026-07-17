import { forwardRef } from 'react'
import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '../../lib/cn'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'
type Size = 'sm' | 'md' | 'lg'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  leftIcon?: ReactNode
  rightIcon?: ReactNode
  loading?: boolean
}

const base =
  'inline-flex items-center justify-center gap-2 font-medium rounded-btn transition-all duration-150 ' +
  'active:scale-[.97] disabled:opacity-50 disabled:cursor-not-allowed ' +
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'

// Mapping per design system: primary = emerald + glow; secondary = outline
// bertint brand (border emerald-200, teks emerald-800); ghost = teks emerald
// dengan hover tint; danger = merah semantic.
const variantClasses: Record<Variant, string> = {
  primary:
    'bg-primary text-white shadow-primary hover:bg-primary-hover active:bg-primary-active disabled:shadow-none',
  secondary:
    'bg-surface border border-primary-border text-primary-active hover:bg-primary-subtle',
  ghost:
    'bg-transparent text-primary hover:bg-primary-subtle',
  danger:
    'bg-danger text-white hover:opacity-90',
}

const sizeClasses: Record<Size, string> = {
  sm: 'h-8 px-3 text-caption',
  md: 'h-10 px-4 text-body',
  lg: 'h-12 px-6 text-body',
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = 'primary',
      size = 'md',
      leftIcon,
      rightIcon,
      loading = false,
      disabled,
      children,
      className,
      ...props
    },
    ref
  ) => {
    const isDisabled = disabled || loading
    return (
      <button
        ref={ref}
        disabled={isDisabled}
        aria-busy={loading || undefined}
        className={cn(base, variantClasses[variant], sizeClasses[size], className)}
        {...props}
      >
        {loading ? (
          <Loader2 className="w-4 h-4 animate-spin" />
        ) : (
          leftIcon
        )}
        {children}
        {!loading && rightIcon}
      </button>
    )
  }
)

Button.displayName = 'Button'

export default Button
