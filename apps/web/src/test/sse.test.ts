import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { ChatStreamHandlers } from '../lib/sse'

// Mock the api module so currentAccessToken() returns a stub token
vi.mock('../lib/api', () => ({
  currentAccessToken: () => 'test-token',
}))

// Import AFTER mocking
const { streamChat } = await import('../lib/sse')

function makeStream(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  let i = 0
  return new ReadableStream({
    pull(controller) {
      if (i < chunks.length) {
        controller.enqueue(encoder.encode(chunks[i++]))
      } else {
        controller.close()
      }
    },
  })
}

function makeFetch(chunks: string[], status = 200) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: async () => ({ error: { message: 'AI error' } }),
    body: makeStream(chunks),
  })
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('streamChat', () => {
  it('dispatches delta, tool_call, then done events', async () => {
    const handlers: ChatStreamHandlers = {
      onDelta: vi.fn(),
      onToolCall: vi.fn(),
      onDone: vi.fn(),
      onError: vi.fn(),
    }

    vi.stubGlobal('fetch', makeFetch([
      'data: {"type":"delta","content":"Hai"}\n\n',
      'data: {"type":"tool_call","id":"c1","name":"list_tenders","arguments":{"limit":10}}\n\n',
      'data: {"type":"done"}\n\n',
    ]))

    await streamChat('conv-1', 'test', handlers)

    expect(handlers.onDelta).toHaveBeenCalledWith('Hai')
    expect(handlers.onToolCall).toHaveBeenCalledWith({ id: 'c1', name: 'list_tenders', arguments: { limit: 10 } })
    expect(handlers.onDone).toHaveBeenCalled()
    expect(handlers.onError).not.toHaveBeenCalled()
  })

  it('handles [DONE] literal end marker', async () => {
    const handlers: ChatStreamHandlers = {
      onDelta: vi.fn(),
      onToolCall: vi.fn(),
      onDone: vi.fn(),
      onError: vi.fn(),
    }

    vi.stubGlobal('fetch', makeFetch([
      'data: {"type":"delta","content":"OK"}\n\n',
      'data: [DONE]\n\n',
    ]))

    await streamChat('conv-1', 'test', handlers)

    expect(handlers.onDelta).toHaveBeenCalledWith('OK')
    expect(handlers.onDone).toHaveBeenCalled()
  })

  it('calls onError when SSE error event received', async () => {
    const handlers: ChatStreamHandlers = {
      onDelta: vi.fn(),
      onToolCall: vi.fn(),
      onDone: vi.fn(),
      onError: vi.fn(),
    }

    vi.stubGlobal('fetch', makeFetch([
      'data: {"type":"error","message":"Agent tidak tersedia"}\n\n',
    ]))

    await streamChat('conv-1', 'test', handlers)

    expect(handlers.onError).toHaveBeenCalledWith('Agent tidak tersedia')
    expect(handlers.onDone).not.toHaveBeenCalled()
  })

  it('calls onError when HTTP response is not ok', async () => {
    const handlers: ChatStreamHandlers = {
      onDelta: vi.fn(),
      onToolCall: vi.fn(),
      onDone: vi.fn(),
      onError: vi.fn(),
    }

    vi.stubGlobal('fetch', makeFetch([], 401))

    await streamChat('conv-1', 'test', handlers)

    expect(handlers.onError).toHaveBeenCalled()
    expect(handlers.onDone).not.toHaveBeenCalled()
  })

  it('does not call onError when aborted', async () => {
    const handlers: ChatStreamHandlers = {
      onDelta: vi.fn(),
      onToolCall: vi.fn(),
      onDone: vi.fn(),
      onError: vi.fn(),
    }

    const controller = new AbortController()

    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(
      Object.assign(new Error('aborted'), { name: 'AbortError' })
    ))

    controller.abort()
    await streamChat('conv-1', 'test', handlers, controller.signal)

    expect(handlers.onError).not.toHaveBeenCalled()
    expect(handlers.onDone).not.toHaveBeenCalled()
  })
})
