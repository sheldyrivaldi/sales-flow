import { useRef, useState, type ReactNode } from 'react'
import { Paperclip, SendHorizonal, Square, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { toast } from '../../lib/toast'
import type { ChatAttachment } from '../../lib/sse'
import Button from '../ui/Button'
import SuggestedPrompts from './SuggestedPrompts'

const MAX_ATTACHMENT_MB = 10
const ACCEPTED = '.pdf,.png,.jpg,.jpeg,.webp'

export interface ChatInputProps {
  onSend: (text: string, attachment?: ChatAttachment) => void
  onStop: () => void
  streaming: boolean
  disabled?: boolean
  contextChip?: ReactNode
  showSuggested?: boolean
  className?: string
}

/** Baca file jadi base64 murni (tanpa prefix data URL). */
function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      resolve(result.slice(result.indexOf(',') + 1))
    }
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

export default function ChatInput({
  onSend,
  onStop,
  streaming,
  disabled,
  contextChip,
  showSuggested,
  className,
}: ChatInputProps) {
  const [text, setText] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function submit() {
    const trimmed = text.trim()
    if ((!trimmed && !file) || disabled || streaming) return

    let attachment: ChatAttachment | undefined
    if (file) {
      try {
        attachment = { name: file.name, base64: await fileToBase64(file) }
      } catch {
        toast.error('Gagal membaca file lampiran.')
        return
      }
    }

    onSend(trimmed || `Tolong baca dan jelaskan isi dokumen ini.`, attachment)
    setText('')
    setFile(null)
  }

  function handleFilePick(f: File | undefined) {
    if (!f) return
    if (f.size > MAX_ATTACHMENT_MB * 1024 * 1024) {
      toast.error(`Ukuran lampiran maksimal ${MAX_ATTACHMENT_MB} MB.`)
      return
    }
    setFile(f)
  }

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      {/* Suggested prompts — shown when conversation is empty */}
      {showSuggested && !streaming && (
        <SuggestedPrompts onSelect={(p) => { setText(''); onSend(p) }} disabled={disabled} />
      )}

      {/* Context chip */}
      {contextChip && <div>{contextChip}</div>}

      {/* Attachment chip */}
      {file && (
        <div className="inline-flex items-center gap-2 self-start rounded-pill border border-accent/40 bg-accent-subtle px-3 py-1 text-caption text-fg">
          <Paperclip className="w-3.5 h-3.5 text-accent-hover" aria-hidden="true" />
          <span className="max-w-56 truncate font-medium">{file.name}</span>
          <button
            type="button"
            aria-label="Hapus lampiran"
            onClick={() => setFile(null)}
            className="text-fg-muted hover:text-danger transition-colors focus-visible:outline-none"
          >
            <X className="w-3.5 h-3.5" aria-hidden="true" />
          </button>
        </div>
      )}

      {/* Input row */}
      <div className="flex gap-2 items-end">
        {/* Attach file */}
        <input
          ref={fileInputRef}
          type="file"
          accept={ACCEPTED}
          className="sr-only"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => {
            handleFilePick(e.target.files?.[0])
            e.target.value = ''
          }}
        />
        <button
          type="button"
          aria-label="Lampirkan dokumen (PDF atau gambar)"
          disabled={disabled || streaming}
          onClick={() => fileInputRef.current?.click()}
          className={cn(
            'shrink-0 h-10 w-10 inline-flex items-center justify-center rounded-btn border border-line bg-surface',
            'text-fg-muted hover:text-primary hover:border-primary-border transition-colors',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1',
            'disabled:opacity-40 disabled:cursor-not-allowed',
          )}
        >
          <Paperclip className="w-4 h-4" aria-hidden="true" />
        </button>

        <textarea
          rows={1}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              void submit()
            }
          }}
          onInput={(e) => {
            const el = e.currentTarget
            el.style.height = 'auto'
            el.style.height = `${Math.min(el.scrollHeight, 160)}px`
          }}
          placeholder="Tanya tentang tender/prospek…"
          disabled={disabled}
          aria-label="Pesan ke agen AI"
          className={cn(
            'flex-1 resize-none rounded-btn border border-line px-3 py-2 text-body text-fg bg-surface',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1',
            'disabled:opacity-40 disabled:cursor-not-allowed overflow-y-auto',
          )}
        />

        {streaming ? (
          <Button
            variant="secondary"
            size="md"
            onClick={onStop}
            leftIcon={<Square className="w-4 h-4" aria-hidden="true" />}
            aria-label="Hentikan generasi"
          >
            Stop
          </Button>
        ) : (
          <Button
            variant="primary"
            size="md"
            onClick={() => void submit()}
            disabled={(!text.trim() && !file) || disabled}
            leftIcon={<SendHorizonal className="w-4 h-4" aria-hidden="true" />}
            aria-label="Kirim pesan"
          >
            Kirim
          </Button>
        )}
      </div>
    </div>
  )
}
