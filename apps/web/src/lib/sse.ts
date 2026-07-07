import { currentAccessToken, ensureRefreshed } from './api'

export interface ToolCallEvent {
  id: string
  name: string
  arguments: unknown
}

export type ChatStreamEvent =
  | { type: 'delta'; content: string }
  | ({ type: 'tool_call' } & ToolCallEvent)
  | { type: 'done' }
  | { type: 'error'; message: string }

export interface ChatStreamHandlers {
  onDelta(text: string): void
  onToolCall(tc: ToolCallEvent): void
  onDone(): void
  onError(message: string): void
}

export async function streamChat(
  conversationId: string,
  content: string,
  handlers: ChatStreamHandlers,
  signal?: AbortSignal,
): Promise<void> {
  const doFetch = () =>
    fetch(`/api/conversations/${conversationId}/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${currentAccessToken() ?? ''}`,
      },
      body: JSON.stringify({ content }),
      signal,
    })

  let res: Response
  try {
    res = await doFetch()
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      handlers.onDone()
      return
    }
    handlers.onError('Koneksi ke agent terputus. Coba lagi.')
    return
  }

  if (res.status === 401) {
    try {
      await ensureRefreshed()
      res = await doFetch()
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        handlers.onDone()
        return
      }
      handlers.onError('Sesi berakhir, silakan login ulang.')
      return
    }
  }

  if (!res.ok) {
    let message = `HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body?.error?.message) message = body.error.message
    } catch {
      // use default message
    }
    handlers.onError(message)
    return
  }

  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // Split on double newline (SSE event boundary)
      const parts = buffer.split('\n\n')
      // Keep the last (potentially incomplete) chunk in the buffer
      buffer = parts.pop() ?? ''

      for (const part of parts) {
        for (const line of part.split('\n')) {
          if (!line.startsWith('data: ')) continue
          const payload = line.slice('data: '.length).trim()
          if (payload === '[DONE]') {
            handlers.onDone()
            return
          }
          try {
            const event: ChatStreamEvent = JSON.parse(payload)
            if (event.type === 'delta') {
              handlers.onDelta(event.content)
            } else if (event.type === 'tool_call') {
              handlers.onToolCall({ id: event.id, name: event.name, arguments: event.arguments })
            } else if (event.type === 'done') {
              handlers.onDone()
              return
            } else if (event.type === 'error') {
              handlers.onError(event.message)
              return
            }
          } catch {
            // malformed JSON frame — skip
          }
        }
      }
    }
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      handlers.onDone()
      return
    }
    handlers.onError('Stream terputus. Coba lagi.')
  }
}
