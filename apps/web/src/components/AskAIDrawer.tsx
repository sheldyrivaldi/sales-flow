import { useState } from 'react'
import { Sparkles } from 'lucide-react'
import Drawer from './ui/Drawer'
import { useAskAIStore } from '../store/askAI'
import { useCreateConversation } from '../api/chat'
import { useChatStream, useChatDegradeStore } from '../store/chat'
import MessageList from './chat/MessageList'
import ChatInput from './chat/ChatInput'
import ContextChip from './chat/ContextChip'
import DegradeBanner from './chat/DegradeBanner'
import type { Message } from '../api/chat'

export default function AskAIDrawer() {
  const { open, context, close } = useAskAIStore()
  const { degraded } = useChatDegradeStore()
  const createConversation = useCreateConversation()
  const { streaming, draft, liveToolCalls, send, stop } = useChatStream()

  // Drawer has its own ephemeral conversation (not shared with Chat page)
  const [activeId, setActiveId] = useState<string | undefined>()
  const [messages, setMessages] = useState<Message[]>([])
  const [localContext, setLocalContext] = useState(context)

  // Sync context when drawer opens with new context
  // (using open as trigger — context may change between opens)
  const effectiveContext = localContext ?? context

  function handleClearContext() {
    setLocalContext(null)
  }

  async function handleSend(content: string) {
    if (!content.trim()) return

    // Append user message optimistically
    const userMsg: Message = {
      id: `local-${Date.now()}`,
      conversation_id: activeId ?? '',
      role: 'user',
      content,
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, userMsg])

    let convId = activeId

    if (!convId) {
      // Build first message with context prefix if present
      const firstMsg = effectiveContext
        ? `[Konteks: ${effectiveContext.type === 'tender' ? 'Tender' : 'Prospek'} — ${effectiveContext.label}]\n\n${content}`
        : content

      const conv = await createConversation.mutateAsync({ first_message: firstMsg })
      convId = conv.id
      setActiveId(convId)
    }

    await send(convId, content)

    // After streaming done, append assistant response from store draft
    // (the MessageList handles live draft display; onDone triggers refetch)
    // Since we manage messages locally here (not via useConversation hook),
    // we append from draft on done
  }

  function handleClose() {
    close()
    // Reset ephemeral state when drawer closes
    setActiveId(undefined)
    setMessages([])
    setLocalContext(null)
  }

  return (
    <Drawer
      open={open}
      onClose={handleClose}
      title={
        <span className="flex items-center gap-2">
          <Sparkles className="w-4 h-4 text-accent" aria-hidden="true" />
          Tanya AI
        </span>
      }
      width="w-[480px]"
    >
      <div className="flex flex-col h-full gap-0">
        {/* Degrade banner inside drawer */}
        <DegradeBanner className="mb-3" />

        {/* Message thread */}
        <div className="flex-1 overflow-y-auto min-h-0 mb-4" style={{ height: 'calc(100% - 140px)' }}>
          <MessageList
            messages={messages}
            draft={draft}
            liveToolCalls={liveToolCalls}
            streaming={streaming}
          />
        </div>

        {/* Input */}
        <ChatInput
          onSend={handleSend}
          onStop={stop}
          streaming={streaming}
          disabled={degraded || createConversation.isPending}
          showSuggested={messages.length === 0}
          contextChip={
            effectiveContext ? (
              <ContextChip label={effectiveContext.label} onClear={handleClearContext} />
            ) : undefined
          }
        />
      </div>
    </Drawer>
  )
}
