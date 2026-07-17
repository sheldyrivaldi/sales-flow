import { useState } from 'react'
import { Sparkles } from 'lucide-react'
import Drawer from './ui/Drawer'
import { useAskAIStore } from '../store/askAI'
import { useCreateConversation, useConversation } from '../api/chat'
import { useChatStream, useChatDegradeStore } from '../store/chat'
import type { ChatAttachment } from '../lib/sse'
import MessageList from './chat/MessageList'
import ChatInput from './chat/ChatInput'
import ContextChip from './chat/ContextChip'
import DegradeBanner from './chat/DegradeBanner'

export default function AskAIDrawer() {
  const { open, context, close } = useAskAIStore()
  const { degraded } = useChatDegradeStore()
  const createConversation = useCreateConversation()

  // Drawer memakai pipeline chat yang SAMA dengan halaman Chat: percakapan
  // nyata yang dipersist + streaming hermes — bukan state lokal yang hilang.
  // Percakapannya juga muncul di halaman Chat untuk dilanjutkan kapan pun.
  const [activeId, setActiveId] = useState<string | undefined>()
  const { streaming, draft, liveToolCalls, pendingUserMessage, send, stop } = useChatStream(activeId)
  const { data: detail } = useConversation(activeId)
  const [localContext, setLocalContext] = useState(context)

  const effectiveContext = localContext ?? context

  function handleClearContext() {
    setLocalContext(null)
  }

  async function handleSend(content: string, attachment?: ChatAttachment) {
    if (!content.trim()) return

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

    await send(convId, content, attachment)
  }

  function handleClose() {
    close()
    // Reset ephemeral state when drawer closes — percakapannya sendiri tetap
    // tersimpan dan bisa dibuka lagi dari halaman Chat.
    setActiveId(undefined)
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
            messages={detail?.messages ?? []}
            draft={draft}
            liveToolCalls={liveToolCalls}
            streaming={streaming}
            pendingUserMessage={pendingUserMessage}
          />
        </div>

        {/* Input */}
        <ChatInput
          onSend={handleSend}
          onStop={stop}
          streaming={streaming}
          disabled={degraded || createConversation.isPending}
          showSuggested={!activeId}
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
