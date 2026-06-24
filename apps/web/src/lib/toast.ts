export type ToastTone = 'success' | 'error' | 'info' | 'warning'

export interface ToastItem {
  id: string
  tone: ToastTone
  message: string
  duration: number
}

type Listener = (items: ToastItem[]) => void

let items: ToastItem[] = []
const listeners = new Set<Listener>()
let counter = 0

function emit() {
  listeners.forEach((l) => l([...items]))
}

export function subscribe(fn: Listener): () => void {
  listeners.add(fn)
  fn([...items])
  return () => listeners.delete(fn)
}

function add(tone: ToastTone, message: string, duration = 4000) {
  const id = `toast-${++counter}`
  items = [...items, { id, tone, message, duration }]
  emit()
  setTimeout(() => remove(id), duration)
}

export function remove(id: string) {
  items = items.filter((t) => t.id !== id)
  emit()
}

export const toast = {
  success: (message: string, duration?: number) => add('success', message, duration),
  error: (message: string, duration?: number) => add('error', message, duration),
  info: (message: string, duration?: number) => add('info', message, duration),
  warning: (message: string, duration?: number) => add('warning', message, duration),
}
