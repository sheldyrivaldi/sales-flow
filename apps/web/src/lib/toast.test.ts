import { describe, it, expect, vi, beforeEach } from 'vitest'

// Reset module between tests to clear singleton state
beforeEach(async () => {
  vi.resetModules()
})

describe('toast store', () => {
  it('toast.success menambah item ke subscriber', async () => {
    const { toast, subscribe } = await import('./toast')
    const listener = vi.fn()
    const unsub = subscribe(listener)

    toast.success('Berhasil disimpan')

    // Panggilan kedua berisi item
    const calls = listener.mock.calls
    const lastItems = calls[calls.length - 1][0]
    expect(lastItems.length).toBe(1)
    expect(lastItems[0].tone).toBe('success')
    expect(lastItems[0].message).toBe('Berhasil disimpan')

    unsub()
  })

  it('subscribe langsung dipanggil dengan state awal', async () => {
    const { subscribe } = await import('./toast')
    const listener = vi.fn()
    const unsub = subscribe(listener)
    expect(listener).toHaveBeenCalledOnce()
    unsub()
  })

  it('remove menghapus item', async () => {
    const { toast, subscribe, remove } = await import('./toast')
    const listener = vi.fn()
    const unsub = subscribe(listener)

    toast.error('Terjadi kesalahan')
    const callsAfterAdd = listener.mock.calls
    const addedId = callsAfterAdd[callsAfterAdd.length - 1][0][0]?.id as string

    remove(addedId)
    const callsAfterRemove = listener.mock.calls
    const itemsAfterRemove = callsAfterRemove[callsAfterRemove.length - 1][0]
    expect(itemsAfterRemove.find((t: { id: string }) => t.id === addedId)).toBeUndefined()

    unsub()
  })

  it('auto-dismiss setelah duration', async () => {
    vi.useFakeTimers()
    const { toast, subscribe } = await import('./toast')
    const listener = vi.fn()
    const unsub = subscribe(listener)

    toast.warning('Peringatan', 1000)

    let items = listener.mock.calls[listener.mock.calls.length - 1][0]
    expect(items.length).toBeGreaterThan(0)

    vi.advanceTimersByTime(1100)

    items = listener.mock.calls[listener.mock.calls.length - 1][0]
    expect(items.length).toBe(0)

    unsub()
    vi.useRealTimers()
  })
})
