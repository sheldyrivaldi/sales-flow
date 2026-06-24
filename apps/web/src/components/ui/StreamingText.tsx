import { cn } from '../../lib/cn'

export interface StreamingTextProps {
  text: string
  streaming?: boolean
  className?: string
}

export default function StreamingText({ text, streaming = false, className }: StreamingTextProps) {
  return (
    <span
      aria-live="polite"
      aria-atomic="false"
      className={cn('whitespace-pre-wrap', className)}
    >
      {text}
      {streaming && (
        <span
          aria-hidden="true"
          className="inline-block w-0.5 h-[1em] bg-current align-middle ml-0.5 animate-pulse"
        />
      )}
    </span>
  )
}
