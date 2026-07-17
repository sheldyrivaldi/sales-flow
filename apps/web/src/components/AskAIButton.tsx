import { Sparkles } from 'lucide-react'
import { useLocation } from 'react-router'
import { useAskAIStore } from '../store/askAI'
import { cn } from '../lib/cn'

export default function AskAIButton() {
  const { openAskAI } = useAskAIStore()
  const { pathname } = useLocation()

  // Halaman Chat SUDAH berupa percakapan AI penuh — tombol mengambang di
  // atasnya redundan dan menutupi input.
  if (pathname.startsWith('/chat')) return null

  return (
    <button
      type="button"
      onClick={() => openAskAI()}
      aria-label="Tanya AI"
      className={cn(
        'fixed bottom-6 right-6 z-40',
        'flex items-center gap-2 px-4 py-2.5 rounded-pill',
        // Identitas AI: gradasi emerald→teal→cyan + glow berdenyut halus,
        // membedakan aksi AI dari tombol brand biasa (emerald solid).
        'bg-ai-gradient text-white animate-pulse-glow',
        'hover:brightness-110 active:scale-95 transition-all duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2',
      )}
    >
      <Sparkles className="w-4 h-4" aria-hidden="true" />
      <span className="text-sm font-medium">Tanya AI</span>
    </button>
  )
}
