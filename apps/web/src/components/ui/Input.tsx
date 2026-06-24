import { forwardRef } from 'react'
import type { InputHTMLAttributes } from 'react'
import { cn } from '../../lib/cn'

export const inputBase =
  'w-full rounded-btn border border-line bg-surface px-3 py-2 text-body text-fg ' +
  'placeholder:text-fg-subtle ' +
  'transition-colors duration-150 ' +
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 ' +
  'disabled:opacity-50 disabled:cursor-not-allowed ' +
  'aria-invalid:border-danger aria-invalid:focus-visible:ring-danger'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ invalid, className, ...props }, ref) => (
    <input
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn(inputBase, className)}
      {...props}
    />
  )
)

Input.displayName = 'Input'

export default Input
