import { useCallback } from 'react'
import { create } from 'zustand'
import { useQueryClient } from '@tanstack/react-query'
import { streamChat } from '../lib/sse'
import type { ToolCallEvent, ChatAttachment } from '../lib/sse'
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

// ── Per-conversation stream store ─────────────────────────────────────────────
//
// Streaming di-KEY per conversationId, bukan per komponen: pindah room saat
// respons masih berjalan tidak membawa spinner/draft ke room lain, dan stream
// yang ditinggal tetap berjalan di background — begitu selesai, hasilnya
// dipersist dan muncul saat room itu dibuka lagi.

export interface PendingUserMessage {
  content: string
  attachmentName?: string
}

export interface StreamState {
  streaming: boolean
  draft: string
  liveToolCalls: ToolCall[]
  pendingUserMessage: PendingUserMessage | null
}

const EMPTY_STREAM: StreamState = {
  streaming: false,
  draft: '',
  liveToolCalls: [],
  pendingUserMessage: null,
}

interface ChatStreamsStore {
  streams: Record<string, StreamState>
  patch: (id: string, p: Partial<StreamState>) => void
  clear: (id: string) => void
}

const useChatStreamsStore = create<ChatStreamsStore>((set) => ({
  streams: {},
  patch: (id, p) =>
    set((s) => ({
      streams: { ...s.streams, [id]: { ...(s.streams[id] ?? EMPTY_STREAM), ...p } },
    })),
  clear: (id) =>
    set((s) => {
      const next = { ...s.streams }
      delete next[id]
      return { streams: next }
    }),
}))

// AbortController per conversation — module-level, bukan state React, supaya
// stop() bisa membatalkan stream conversation tertentu dari komponen mana pun.
const controllers = new Map<string, AbortController>()

export interface UseChatStream {
  streaming: boolean
  draft: string
  liveToolCalls: ToolCall[]
  /** Pesan user yang baru dikirim — dirender optimistically oleh MessageList
   *  sampai hasil refetch dari server memuatnya. */
  pendingUserMessage: PendingUserMessage | null
  send: (conversationId: string, content: string, attachment?: ChatAttachment) => Promise<void>
  stop: () => void
}

/** State stream milik `conversationId` + aksi kirim/stop. Komponen hanya
 *  melihat stream conversation yang sedang dibukanya — terisolasi antar room. */
export function useChatStream(conversationId?: string): UseChatStream {
  const queryClient = useQueryClient()
  const { setDegraded } = useChatDegradeStore()

  const state = useChatStreamsStore((s) =>
    conversationId ? (s.streams[conversationId] ?? EMPTY_STREAM) : EMPTY_STREAM,
  )

  const stop = useCallback(() => {
    if (conversationId) controllers.get(conversationId)?.abort()
  }, [conversationId])

  const send = useCallback(
    async (convId: string, content: string, attachment?: ChatAttachment) => {
      const { patch, clear } = useChatStreamsStore.getState()

      // Satu stream aktif per conversation — kiriman ganda diabaikan
      // (ChatInput juga sudah disable saat streaming).
      if (useChatStreamsStore.getState().streams[convId]?.streaming) return

      patch(convId, {
        streaming: true,
        draft: '',
        liveToolCalls: [],
        pendingUserMessage: { content, attachmentName: attachment?.name },
      })
      setDegraded(false)

      const controller = new AbortController()
      controllers.set(convId, controller)

      // Akumulasi lokal (bukan baca-balik store) supaya update delta bebas
      // race antar-stream yang berjalan paralel di room berbeda.
      let draftAcc = ''
      const toolCallsAcc: ToolCall[] = []

      const settle = async () => {
        controllers.delete(convId)
        await Promise.allSettled([
          queryClient.invalidateQueries({ queryKey: ['conversation', convId] }),
          queryClient.invalidateQueries({ queryKey: ['conversations'] }),
        ])
        clear(convId)
      }

      await streamChat(
        convId,
        content,
        {
          onDelta(text) {
            draftAcc += text
            patch(convId, { draft: draftAcc })
          },
          onToolCall(tc: ToolCallEvent) {
            toolCallsAcc.push({ id: tc.id, type: 'function', name: tc.name, arguments: tc.arguments })
            patch(convId, { liveToolCalls: [...toolCallsAcc] })
          },
          onDone() {
            void settle()
          },
          onError(message) {
            setDegraded(true)
            toast.error(message)
            void settle()
          },
        },
        controller.signal,
        attachment,
      )
    },
    [queryClient, setDegraded],
  )

  return { ...state, send, stop }
}
