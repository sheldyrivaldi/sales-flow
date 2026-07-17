import { useEffect, useRef } from 'react'
import { Paperclip } from 'lucide-react'
import type { Message, ToolCall } from '../../api/chat'
import type { PendingUserMessage } from '../../store/chat'
import MessageBubble from './MessageBubble'
import ToolCallChip from './ToolCallChip'

export interface MessageListProps {
  messages: Message[]
  draft: string
  liveToolCalls: ToolCall[]
  streaming: boolean
  /** Pesan user yang baru dikirim, dirender langsung (optimistic) sampai
   *  versi persisten dari server tiba lewat refetch. */
  pendingUserMessage?: PendingUserMessage | null
}

export default function MessageList({
  messages,
  draft,
  liveToolCalls,
  streaming,
  pendingUserMessage,
}: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when messages, pending message, or draft changes
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length, draft, pendingUserMessage])

  // Dedup guard: begitu refetch memuat versi persisten dari pesan pending
  // (kontennya sama, atau sama + suffix "[Lampiran: …]" yang ditambah
  // server), jangan render bubble optimistic kedua kalinya.
  const last = messages[messages.length - 1]
  const showPending =
    !!pendingUserMessage &&
    !(last && last.role === 'user' && last.content.startsWith(pendingUserMessage.content))

  return (
    <div className="flex flex-col gap-4">
      {messages.map((msg) => (
        <MessageBubble
          key={msg.id}
          role={msg.role}
          content={msg.content}
          toolCalls={msg.tool_calls}
        />
      ))}

      {/* Pesan user optimistic — tampil seketika saat dikirim. Dirender
          langsung (bukan dibungkus flex items-end) supaya layout max-width
          internal MessageBubble tidak rusak/terpotong. */}
      {showPending && pendingUserMessage && (
        <>
          <MessageBubble role="user" content={pendingUserMessage.content} />
          {pendingUserMessage.attachmentName && (
            <div className="flex justify-end -mt-2">
              <span className="inline-flex items-center gap-1 text-caption text-fg-muted pr-1">
                <Paperclip className="w-3 h-3" aria-hidden="true" />
                {pendingUserMessage.attachmentName}
              </span>
            </div>
          )}
        </>
      )}

      {/* Live tool calls during streaming */}
      {streaming && liveToolCalls.length > 0 && (
        <div className="flex justify-start gap-2 pl-8">
          <div className="flex flex-col gap-1">
            {liveToolCalls.map((tc) => (
              <ToolCallChip key={tc.id} name={tc.name} arguments={tc.arguments} status="running" />
            ))}
          </div>
        </div>
      )}

      {/* Typing dots — AI sedang "berpikir", belum ada token pertama */}
      {streaming && !draft && (
        <div className="flex justify-start">
          <div
            className="inline-flex items-center gap-1.5 rounded-card bg-surface border border-line px-4 py-3 shadow-subtle"
            role="status"
            aria-label="AI sedang mengetik"
          >
            <span className="typing-dot inline-block h-2 w-2 rounded-full bg-accent" />
            <span className="typing-dot inline-block h-2 w-2 rounded-full bg-accent" />
            <span className="typing-dot inline-block h-2 w-2 rounded-full bg-accent" />
          </div>
        </div>
      )}

      {/* Streaming draft (live assistant text) */}
      {streaming && draft && (
        <MessageBubble role="assistant" content={draft} streaming />
      )}

      {/* Anchor for auto-scroll */}
      <div ref={bottomRef} />
    </div>
  )
}
