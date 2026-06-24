import { useState } from 'react'
import { Search, Plus } from 'lucide-react'
import { useConversations, useConversation, useCreateConversation } from '../api/chat'
import { useChatStream, useChatDegradeStore } from '../store/chat'
import { formatRelative } from '../lib/format'
import { cn } from '../lib/cn'
import Skeleton from '../components/ui/Skeleton'
import EmptyState from '../components/ui/EmptyState'
import Button from '../components/ui/Button'
import Input from '../components/ui/Input'
import MessageList from '../components/chat/MessageList'
import ChatInput from '../components/chat/ChatInput'
import DegradeBanner from '../components/chat/DegradeBanner'

export default function Chat() {
  const [activeId, setActiveId] = useState<string | undefined>()
  const [search, setSearch] = useState('')

  const { data: list, isLoading: loadingList } = useConversations()
  const { data: detail, isLoading: loadingDetail } = useConversation(activeId)
  const createConversation = useCreateConversation()
  const { streaming, draft, liveToolCalls, send, stop } = useChatStream()
  const { degraded } = useChatDegradeStore()

  const filtered = (list?.items ?? []).filter((c) =>
    c.title.toLowerCase().includes(search.toLowerCase()),
  )

  async function handleSend(content: string) {
    if (!content.trim()) return
    let convId = activeId

    if (!convId) {
      const conv = await createConversation.mutateAsync({ first_message: content })
      convId = conv.id
      setActiveId(convId)
    }

    await send(convId, content)
  }

  const activeTitle = detail?.title ?? (activeId ? '…' : 'Chat')

  return (
    <div className="flex h-full -m-6 overflow-hidden">
      {/* ── Left panel: conversation list ──────────────────────────────── */}
      <aside className="w-64 shrink-0 border-r border-line bg-surface flex flex-col">
        <div className="p-3 border-b border-line flex items-center gap-2">
          <Button
            variant="primary"
            size="sm"
            leftIcon={<Plus className="w-4 h-4" />}
            onClick={() => setActiveId(undefined)}
            className="flex-1"
          >
            Baru
          </Button>
        </div>

        <div className="p-3 border-b border-line">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-fg-subtle pointer-events-none" />
            <Input
              placeholder="Cari…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8 text-sm"
            />
          </div>
        </div>

        <nav className="flex-1 overflow-y-auto py-2">
          {loadingList ? (
            <div className="px-3 space-y-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} variant="text" className="h-10 w-full" />
              ))}
            </div>
          ) : filtered.length === 0 ? (
            <EmptyState
              title="Belum ada percakapan"
              description='Klik "+ Baru" untuk mulai'
              className="py-8 px-4"
            />
          ) : (
            filtered.map((conv) => (
              <button
                key={conv.id}
                type="button"
                onClick={() => setActiveId(conv.id)}
                className={cn(
                  'w-full text-left px-4 py-2.5 flex flex-col gap-0.5 hover:bg-surface-subtle transition-colors',
                  activeId === conv.id && 'bg-primary/5 border-l-2 border-primary',
                )}
              >
                <span className="text-sm font-medium text-fg truncate">{conv.title}</span>
                <span className="text-caption text-fg-subtle">
                  {formatRelative(conv.updated_at)}
                </span>
              </button>
            ))
          )}
        </nav>
      </aside>

      {/* ── Right panel: message area ───────────────────────────────────── */}
      <div className="flex-1 flex flex-col min-w-0 bg-surface-muted">
        {/* Header */}
        <header className="px-5 py-3 border-b border-line bg-surface flex items-center gap-3 shrink-0">
          <h1 className="text-h3 font-semibold text-fg flex-1 truncate">{activeTitle}</h1>
          <span
            className={cn(
              'flex items-center gap-1.5 text-caption font-medium',
              degraded ? 'text-danger' : 'text-success',
            )}
          >
            <span
              className={cn('w-2 h-2 rounded-full', degraded ? 'bg-danger' : 'bg-success')}
            />
            {degraded ? 'Agent tidak tersedia' : 'Terhubung ke AI'}
          </span>
        </header>

        {/* Degrade banner */}
        <DegradeBanner className="mx-5 mt-4" />

        {/* Message thread */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {!activeId ? (
            <div className="flex flex-col items-center justify-center h-full gap-4">
              <EmptyState
                title="Tanya Agen AI"
                description="Pilih percakapan atau mulai yang baru"
              />
            </div>
          ) : loadingDetail ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} variant="text" className="h-12 w-3/4" />
              ))}
            </div>
          ) : (
            <MessageList
              messages={detail?.messages ?? []}
              draft={draft}
              liveToolCalls={liveToolCalls}
              streaming={streaming}
            />
          )}
        </div>

        {/* Input area */}
        <div className="px-5 py-4 border-t border-line bg-surface shrink-0">
          <ChatInput
            onSend={handleSend}
            onStop={stop}
            streaming={streaming}
            disabled={degraded || createConversation.isPending}
            showSuggested={!activeId}
          />
        </div>
      </div>
    </div>
  )
}
