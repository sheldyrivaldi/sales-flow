import { useRef, useState } from 'react'
import type { DragEvent, ChangeEvent, ReactNode } from 'react'
import { UploadCloud } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface FileDropzoneProps {
  onFiles: (files: File[]) => void
  accept?: string
  multiple?: boolean
  maxSizeMB?: number
  onError?: (message: string) => void
  children?: ReactNode
  disabled?: boolean
  className?: string
}

export default function FileDropzone({
  onFiles,
  accept = 'application/pdf',
  multiple = false,
  maxSizeMB = 10,
  onError,
  children,
  disabled = false,
  className,
}: FileDropzoneProps) {
  const [dragging, setDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  function validate(files: FileList | null): File[] {
    if (!files || files.length === 0) return []
    const valid: File[] = []
    const maxBytes = maxSizeMB * 1024 * 1024

    Array.from(files).forEach((file) => {
      if (accept && !file.type.match(accept.replace(/\*/g, '.*'))) {
        onError?.(`File "${file.name}" bukan tipe yang didukung (${accept}).`)
        return
      }
      if (file.size > maxBytes) {
        onError?.(`File "${file.name}" melebihi batas ukuran ${maxSizeMB} MB.`)
        return
      }
      valid.push(file)
    })
    return valid
  }

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setDragging(false)
    if (disabled) return
    const valid = validate(e.dataTransfer.files)
    if (valid.length > 0) onFiles(valid)
  }

  function handleDragOver(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    if (!disabled) setDragging(true)
  }

  function handleDragLeave() {
    setDragging(false)
  }

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    const valid = validate(e.target.files)
    if (valid.length > 0) onFiles(valid)
    // Reset input agar file yang sama bisa dipilih ulang
    e.target.value = ''
  }

  return (
    <div className={cn('flex flex-col gap-3', className)}>
      <div
        role="button"
        tabIndex={disabled ? -1 : 0}
        aria-label="Unggah file — klik atau seret file ke sini"
        aria-disabled={disabled || undefined}
        onClick={() => !disabled && inputRef.current?.click()}
        onKeyDown={(e) => {
          if (!disabled && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault()
            inputRef.current?.click()
          }
        }}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        className={cn(
          'flex flex-col items-center justify-center gap-2 rounded-card border-2 border-dashed p-8 text-center cursor-pointer transition-colors duration-150',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2',
          dragging
            ? 'border-primary bg-primary/5 text-primary'
            : 'border-line bg-surface hover:border-primary/40 hover:bg-primary/5 text-fg-muted',
          disabled && 'opacity-50 cursor-not-allowed pointer-events-none'
        )}
      >
        <UploadCloud
          className={cn('w-8 h-8', dragging ? 'text-primary' : 'text-fg-muted')}
          aria-hidden="true"
        />
        <div>
          <p className="text-body font-medium text-fg">
            Seret {multiple ? 'file' : 'PDF'} ke sini atau{' '}
            <span className="text-primary underline">klik untuk pilih</span>
          </p>
          <p className="text-caption text-fg-subtle mt-0.5">
            {accept === 'application/pdf' ? 'PDF' : accept} · maks. {maxSizeMB} MB
          </p>
        </div>
      </div>

      {children}

      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple={multiple}
        onChange={handleChange}
        disabled={disabled}
        className="sr-only"
        aria-hidden="true"
        tabIndex={-1}
      />
    </div>
  )
}
