import { useState, type ReactNode } from 'react'
import { SendHorizonal, Square } from 'lucide-react'
import { cn } from '../../lib/cn'
import Button from '../ui/Button'
import SuggestedPrompts from './SuggestedPrompts'

export interface ChatInputProps {
  onSend: (text: string) => void
  onStop: () => void
  streaming: boolean
  disabled?: boolean
  contextChip?: ReactNode
  showSuggested?: boolean
  className?: string
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

  function submit() {
    const trimmed = text.trim()
    if (!trimmed || disabled || streaming) return
    onSend(trimmed)
    setText('')
  }

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      {/* Suggested prompts — shown when conversation is empty */}
      {showSuggested && !streaming && (
        <SuggestedPrompts onSelect={(p) => { setText(''); onSend(p) }} disabled={disabled} />
      )}

      {/* Context chip */}
      {contextChip && <div>{contextChip}</div>}

      {/* Input row */}
      <div className="flex gap-2 items-end">
        <textarea
          rows={1}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              submit()
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
            onClick={submit}
            disabled={!text.trim() || disabled}
            leftIcon={<SendHorizonal className="w-4 h-4" aria-hidden="true" />}
            aria-label="Kirim pesan"
          >
            Kirim
          </Button>
        )}
      </div>

      <p className="text-caption text-fg-subtle">
        Asisten belajar dari aktivitas &amp; hasil kamu
      </p>
    </div>
  )
}
