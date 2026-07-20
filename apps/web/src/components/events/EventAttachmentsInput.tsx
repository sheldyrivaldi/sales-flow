import { useRef, useState } from 'react'
import type { DragEvent } from 'react'
import { Paperclip, X, Loader2, FileText, Image as ImageIcon, Sheet, File } from 'lucide-react'
import { cn } from '../../lib/cn'
import Button from '../ui/Button'
import { useUploadEventAttachment } from '../../api/events'
import type { EventAttachment } from '../../api/events'

const MAX_MB = 10

function formatSize(bytes?: number): string {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/** Ikon mengikuti jenis berkas supaya daftar bisa dipindai sekilas. */
function iconFor(a: EventAttachment) {
  const n = a.name.toLowerCase()
  if (/\.(png|jpe?g|webp|gif|svg)$/.test(n)) return ImageIcon
  if (/\.(xlsx?|csv)$/.test(n)) return Sheet
  if (/\.(pdf|docx?|txt)$/.test(n)) return FileText
  return File
}

export interface EventAttachmentsInputProps {
  value: EventAttachment[]
  onChange: (next: EventAttachment[]) => void
  disabled?: boolean
}

/**
 * Unggah lampiran event (rundown, undangan, denah booth).
 *
 * Berkas diunggah SEKETIKA saat dipilih, bukan saat form disimpan — supaya
 * user melihat progres per berkas dan bisa membatalkan sebelum menyimpan.
 * Yang disimpan pada event hanyalah metadata {name,url,mime,size}.
 */
export default function EventAttachmentsInput({ value, onChange, disabled }: EventAttachmentsInputProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [pending, setPending] = useState<string[]>([])
  const [error, setError] = useState('')
  const upload = useUploadEventAttachment()

  async function handleFiles(files: FileList | File[]) {
    const list = Array.from(files)
    if (list.length === 0) return
    setError('')

    const tooBig = list.filter((f) => f.size > MAX_MB * 1024 * 1024)
    if (tooBig.length > 0) {
      setError(`${tooBig.map((f) => f.name).join(', ')} melebihi ${MAX_MB} MB.`)
    }
    const ok = list.filter((f) => f.size <= MAX_MB * 1024 * 1024)
    if (ok.length === 0) return

    setPending((p) => [...p, ...ok.map((f) => f.name)])
    // Berurutan, bukan paralel: unggahan besar serentak membuat progres
    // terlihat macet dan membebani koneksi kantor.
    for (const f of ok) {
      try {
        const saved = await upload.mutateAsync(f)
        onChange([...value, saved])
      } catch {
        setError((e) => e || `Gagal mengunggah ${f.name}.`)
      } finally {
        setPending((p) => p.filter((n) => n !== f.name))
      }
    }
  }

  function onDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setDragging(false)
    if (disabled) return
    if (e.dataTransfer.files?.length) void handleFiles(e.dataTransfer.files)
  }

  return (
    <div className="flex flex-col gap-2">
      <div
        onDragOver={(e) => {
          e.preventDefault()
          if (!disabled) setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className={cn(
          'rounded-card border border-dashed px-3 py-3 transition-colors',
          dragging ? 'border-primary bg-primary-subtle' : 'border-line bg-surface-subtle',
          disabled && 'opacity-60',
        )}
      >
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <p className="text-caption text-fg-muted">
            Tarik berkas ke sini atau pilih manual. Maks {MAX_MB} MB per berkas.
          </p>
          <input
            ref={inputRef}
            type="file"
            multiple
            className="sr-only"
            tabIndex={-1}
            aria-hidden="true"
            onChange={(e) => {
              if (e.target.files) void handleFiles(e.target.files)
              e.target.value = ''
            }}
          />
          <Button
            type="button"
            variant="secondary"
            size="sm"
            disabled={disabled}
            leftIcon={<Paperclip className="w-3.5 h-3.5" />}
            onClick={() => inputRef.current?.click()}
          >
            Pilih Berkas
          </Button>
        </div>
      </div>

      {(value.length > 0 || pending.length > 0) && (
        <ul className="flex flex-col gap-1.5">
          {value.map((a, i) => {
            const Icon = iconFor(a)
            return (
              <li
                key={`${a.url}-${i}`}
                className="flex items-center gap-2 rounded-btn border border-line bg-surface px-2.5 py-1.5 animate-row-in"
              >
                <Icon className="w-4 h-4 text-primary shrink-0" aria-hidden="true" />
                <a
                  href={a.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex-1 min-w-0 text-body text-fg hover:text-primary hover:underline truncate"
                  title={a.name}
                >
                  {a.name}
                </a>
                {a.size ? <span className="text-caption text-fg-subtle shrink-0">{formatSize(a.size)}</span> : null}
                <button
                  type="button"
                  aria-label={`Hapus ${a.name}`}
                  disabled={disabled}
                  onClick={() => onChange(value.filter((_, idx) => idx !== i))}
                  className="p-1 rounded-btn text-fg-subtle hover:text-danger hover:bg-danger-subtle transition-colors shrink-0"
                >
                  <X className="w-3.5 h-3.5" aria-hidden="true" />
                </button>
              </li>
            )
          })}

          {pending.map((name) => (
            <li
              key={`pending-${name}`}
              className="flex items-center gap-2 rounded-btn border border-line bg-surface-subtle px-2.5 py-1.5"
            >
              <Loader2 className="w-4 h-4 text-primary shrink-0 animate-spin" aria-hidden="true" />
              <span className="flex-1 min-w-0 text-body text-fg-muted truncate">{name}</span>
              <span className="text-caption text-fg-subtle shrink-0">mengunggah…</span>
            </li>
          ))}
        </ul>
      )}

      {error && <p className="text-caption text-danger">{error}</p>}
    </div>
  )
}
