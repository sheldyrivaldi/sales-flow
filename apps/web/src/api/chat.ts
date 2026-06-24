import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface ToolCall {
  id: string
  type: string
  name: string
  arguments: unknown
}

export interface ConversationSummary {
  id: string
  title: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  conversation_id: string
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  tool_calls?: ToolCall[]
  created_at: string
}

export interface ConversationDetail extends ConversationSummary {
  messages: Message[]
}

interface ConversationListResponse {
  items: ConversationSummary[]
  total: number
  page: number
  page_size: number
}

// ── Hooks ──────────────────────────────────────────────────────────────────────

export function useConversations() {
  return useQuery({
    queryKey: ['conversations'],
    queryFn: () =>
      apiFetch<ConversationListResponse>('/api/conversations?page=1&page_size=50'),
  })
}

export function useConversation(id?: string) {
  return useQuery({
    queryKey: ['conversation', id],
    queryFn: () => apiFetch<ConversationDetail>(`/api/conversations/${id}`),
    enabled: !!id,
  })
}

export function useCreateConversation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: { title?: string; first_message?: string }) =>
      apiFetch<ConversationSummary>('/api/conversations', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['conversations'] })
    },
  })
}
