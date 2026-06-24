import { useState, useRef, useCallback } from 'react'
import { create } from 'zustand'
import { useQueryClient } from '@tanstack/react-query'
import { streamChat } from '../lib/sse'
import type { ToolCallEvent } from '../lib/sse'
import type { ToolCall } from '../api/chat'
import { toast } from '../lib/toast'

// ── Global degrade state (shared between Chat page and AskAIDrawer) ───────────

interface ChatDegradeStore {
  degraded: boolean
  setDegraded: (v: boolean) => void
}

export const useChatDegradeStore = create<ChatDegradeStore>((set) => ({
  degraded: false,
  setDegraded: (v) => set({ degraded: v }),
}))

// ── Per-surface streaming hook ────────────────────────────────────────────────

export interface ChatStreamState {
  streaming: boolean
  draft: string
  liveToolCalls: ToolCall[]
}

export interface UseChatStream {
  streaming: boolean
  draft: string
  liveToolCalls: ToolCall[]
  send: (conversationId: string, content: string) => Promise<void>
  stop: () => void
}

export function useChatStream(): UseChatStream {
  const [streaming, setStreaming] = useState(false)
  const [draft, setDraft] = useState('')
  const [liveToolCalls, setLiveToolCalls] = useState<ToolCall[]>([])
  const abortRef = useRef<AbortController | null>(null)
  const queryClient = useQueryClient()
  const { setDegraded } = useChatDegradeStore()

  const stop = useCallback(() => {
    abortRef.current?.abort()
  }, [])

  const send = useCallback(
    async (conversationId: string, content: string) => {
      // Reset state for new message
      setStreaming(true)
      setDraft('')
      setLiveToolCalls([])
      setDegraded(false)

      const controller = new AbortController()
      abortRef.current = controller

      await streamChat(
        conversationId,
        content,
        {
          onDelta(text) {
            setDraft((prev) => prev + text)
          },
          onToolCall(tc: ToolCallEvent) {
            setLiveToolCalls((prev) => [
              ...prev,
              { id: tc.id, type: 'function', name: tc.name, arguments: tc.arguments },
            ])
          },
          onDone() {
            setStreaming(false)
            setDraft('')
            setLiveToolCalls([])
            // Refetch conversation to get persisted messages
            queryClient.invalidateQueries({ queryKey: ['conversation', conversationId] })
            queryClient.invalidateQueries({ queryKey: ['conversations'] })
          },
          onError(message) {
            setStreaming(false)
            setDegraded(true)
            toast.error(message)
          },
        },
        controller.signal,
      )
    },
    [queryClient, setDegraded],
  )

  return { streaming, draft, liveToolCalls, send, stop }
}
