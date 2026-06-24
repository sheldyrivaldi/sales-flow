import { Sparkles } from 'lucide-react'
import { useAskAIStore } from '../store/askAI'
import { cn } from '../lib/cn'

export default function AskAIButton() {
  const { openAskAI } = useAskAIStore()

  return (
    <button
      type="button"
      onClick={() => openAskAI()}
      aria-label="Tanya AI"
      className={cn(
        'fixed bottom-6 right-6 z-40',
        'flex items-center gap-2 px-4 py-2.5 rounded-pill',
        'bg-accent text-white shadow-subtle',
        'hover:bg-accent/90 active:scale-95 transition-all',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2',
      )}
    >
      <Sparkles className="w-4 h-4" aria-hidden="true" />
      <span className="text-sm font-medium">Tanya AI</span>
    </button>
  )
}
