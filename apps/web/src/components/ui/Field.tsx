import type { ReactNode } from 'react'

export interface FieldProps {
  label?: string
  helper?: string
  error?: string
  required?: boolean
  htmlFor?: string
  children: ReactNode
}

export default function Field({ label, helper, error, required, htmlFor, children }: FieldProps) {
  const helperId = htmlFor ? `${htmlFor}-helper` : undefined
  const message = error ?? helper

  return (
    <div className="flex flex-col gap-1">
      {label && (
        <label htmlFor={htmlFor} className="text-caption font-medium text-fg">
          {label}
          {required && <span className="text-danger ml-0.5" aria-hidden="true">*</span>}
        </label>
      )}
      {children}
      {message && (
        <p
          id={helperId}
          className={error ? 'text-caption text-danger' : 'text-caption text-fg-muted'}
        >
          {message}
        </p>
      )}
    </div>
  )
}
