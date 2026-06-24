import { create } from 'zustand'

export interface AskAIContext {
  type: 'tender' | 'prospect'
  id: string
  label: string
}

interface AskAIStore {
  open: boolean
  context: AskAIContext | null
  openAskAI: (ctx?: AskAIContext) => void
  close: () => void
}

export const useAskAIStore = create<AskAIStore>((set) => ({
  open: false,
  context: null,
  openAskAI: (ctx) => set({ open: true, context: ctx ?? null }),
  close: () => set({ open: false }),
}))
