import { useRef, useState } from 'react'
import type { ClipboardEvent, KeyboardEvent } from 'react'
import { X, Mail, AlertCircle } from 'lucide-react'
import { cn } from '../../lib/cn'
import { isValidEmail, splitEmails } from '../../lib/email'

export interface EmailChipsInputProps {
  value: string[]
  onChange: (next: string[]) => void
  disabled?: boolean
  max?: number
  id?: string
  /** Ditandai invalid oleh form induk (mis. saat submit). */
  invalid?: boolean
}

/**
 * Input daftar email peserta berbentuk chip.
 *
 * Perilaku yang disengaja:
 * - Enter, koma, titik koma, atau Tab menutup satu chip.
 * - Tempel banyak email sekaligus langsung terpecah jadi banyak chip.
 * - Email tidak valid TIDAK dibuang diam-diam; ia ditahan di kotak isian dan
 *   ditandai merah supaya user bisa memperbaikinya, bukan kehilangan ketikan.
 * - Duplikat diabaikan tanpa pesan error (bukan kesalahan user).
 * - Backspace pada kotak kosong menghapus chip terakhir.
 */
export default function EmailChipsInput({
  value,
  onChange,
  disabled,
  max = 200,
  id,
  invalid,
}: EmailChipsInputProps) {
  const [draft, setDraft] = useState('')
  const [focused, setFocused] = useState(false)
  const [rejected, setRejected] = useState<string[]>([])
  const inputRef = useRef<HTMLInputElement>(null)

  const atLimit = value.length >= max
  const draftInvalid = draft.trim() !== '' && !isValidEmail(draft)

  /** Tambah banyak kandidat sekaligus; kembalikan yang ditolak. */
  function commit(candidates: string[]): string[] {
    const bad: string[] = []
    const next = [...value]
    for (const c of candidates) {
      if (!c) continue
      if (!isValidEmail(c)) {
        bad.push(c)
        continue
      }
      if (next.length >= max) break
      if (!next.includes(c)) next.push(c)
    }
    if (next.length !== value.length) onChange(next)
    return bad
  }

  function commitDraft() {
    const parts = splitEmails(draft)
    if (parts.length === 0) {
      setDraft('')
      return
    }
    const bad = commit(parts)
    // Yang gagal dikembalikan ke kotak isian agar bisa dibetulkan.
    setDraft(bad.join(', '))
    setRejected(bad)
  }

  function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' || e.key === ',' || e.key === ';' || e.key === 'Tab') {
      if (draft.trim() === '') return // Tab tetap boleh pindah fokus
      e.preventDefault()
      commitDraft()
      return
    }
    if (e.key === 'Backspace' && draft === '' && value.length > 0) {
      onChange(value.slice(0, -1))
    }
  }

  function handlePaste(e: ClipboardEvent<HTMLInputElement>) {
    const text = e.clipboardData.getData('text')
    if (!/[\s,;]/.test(text)) return // satu email saja — biarkan perilaku normal
    e.preventDefault()
    const bad = commit(splitEmails(text))
    setDraft(bad.join(', '))
    setRejected(bad)
  }

  function removeAt(i: number) {
    onChange(value.filter((_, idx) => idx !== i))
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div
        onClick={() => inputRef.current?.focus()}
        className={cn(
          'flex flex-wrap items-center gap-1.5 rounded-btn border bg-surface px-2 py-2 transition-colors cursor-text min-h-[42px]',
          focused ? 'border-primary ring-2 ring-primary/20' : 'border-line hover:border-primary-border',
          (invalid || draftInvalid) && 'border-danger ring-2 ring-danger/15',
          disabled && 'opacity-60 cursor-not-allowed',
        )}
      >
        {value.map((email, i) => (
          <span
            key={email}
            className="inline-flex items-center gap-1 rounded-pill bg-primary-subtle border border-primary/25 pl-2 pr-1 py-0.5 text-caption text-fg animate-chip-in max-w-full"
          >
            <Mail className="w-3 h-3 text-primary shrink-0" aria-hidden="true" />
            <span className="truncate">{email}</span>
            <button
              type="button"
              aria-label={`Hapus ${email}`}
              disabled={disabled}
              onClick={(e) => {
                e.stopPropagation()
                removeAt(i)
              }}
              className="p-0.5 rounded-full text-fg-muted hover:text-danger hover:bg-danger-subtle transition-colors"
            >
              <X className="w-3 h-3" aria-hidden="true" />
            </button>
          </span>
        ))}

        <input
          ref={inputRef}
          id={id}
          type="text"
          inputMode="email"
          autoComplete="off"
          disabled={disabled || atLimit}
          value={draft}
          onChange={(e) => {
            setDraft(e.target.value)
            if (rejected.length) setRejected([])
          }}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
          onFocus={() => setFocused(true)}
          onBlur={() => {
            setFocused(false)
            commitDraft()
          }}
          placeholder={
            atLimit
              ? `Batas ${max} peserta tercapai`
              : value.length === 0
                ? 'ketik email lalu Enter, atau tempel banyak sekaligus'
                : 'tambah lagi…'
          }
          className="flex-1 min-w-[180px] bg-transparent outline-none text-body text-fg placeholder:text-fg-subtle disabled:cursor-not-allowed"
        />
      </div>

      <div className="flex items-start justify-between gap-2">
        {draftInvalid || rejected.length > 0 ? (
          <p className="inline-flex items-start gap-1 text-caption text-danger">
            <AlertCircle className="w-3.5 h-3.5 mt-px shrink-0" aria-hidden="true" />
            <span>
              {rejected.length > 1
                ? `${rejected.length} alamat belum valid dan masih di kotak isian.`
                : 'Format email belum valid, contoh nama@perusahaan.com.'}
            </span>
          </p>
        ) : (
          <p className="text-caption text-fg-subtle">
            Peserta tidak perlu punya akun di aplikasi ini. Pisahkan dengan Enter atau koma.
          </p>
        )}
        {value.length > 0 && (
          <span className="text-caption text-fg-subtle tabular-nums shrink-0">
            {value.length}
            {max ? `/${max}` : ''}
          </span>
        )}
      </div>
    </div>
  )
}
