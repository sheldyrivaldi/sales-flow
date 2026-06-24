import { forwardRef } from 'react'
import type { InputHTMLAttributes } from 'react'
import { cn } from '../../lib/cn'
import { inputBase } from './Input'

export interface DatePickerProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean
}

const DatePicker = forwardRef<HTMLInputElement, DatePickerProps>(
  ({ invalid, className, ...props }, ref) => (
    <input
      ref={ref}
      type="date"
      aria-invalid={invalid || undefined}
      className={cn(inputBase, className)}
      {...props}
    />
  )
)

DatePicker.displayName = 'DatePicker'

export default DatePicker
