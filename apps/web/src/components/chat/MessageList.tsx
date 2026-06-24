import { useEffect, useRef } from 'react'
import type { Message, ToolCall } from '../../api/chat'
import MessageBubble from './MessageBubble'
import ToolCallChip from './ToolCallChip'

export interface MessageListProps {
  messages: Message[]
  draft: string
  liveToolCalls: ToolCall[]
  streaming: boolean
}

export default function MessageList({ messages, draft, liveToolCalls, streaming }: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when messages or streaming draft changes
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length, draft])

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

      {/* Streaming draft (live assistant text) */}
      {streaming && draft && (
        <MessageBubble role="assistant" content={draft} streaming />
      )}

      {/* Anchor for auto-scroll */}
      <div ref={bottomRef} />
    </div>
  )
}
