import { useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { useConversations, useConversation, useCreateConversation, useDeleteConversation } from '../api/chat'
import type { ConversationSummary } from '../api/chat'
import { useChatStream, useChatDegradeStore } from '../store/chat'
import type { ChatAttachment } from '../lib/sse'
import { formatRelative } from '../lib/format'
import { cn } from '../lib/cn'
import { toast } from '../lib/toast'
import Skeleton from '../components/ui/Skeleton'
import EmptyState from '../components/ui/EmptyState'
import Tooltip from '../components/ui/Tooltip'
import ConfirmDialog from '../components/ui/ConfirmDialog'
import MessageList from '../components/chat/MessageList'
import ChatInput from '../components/chat/ChatInput'
import DegradeBanner from '../components/chat/DegradeBanner'

export default function Chat() {
  const [activeId, setActiveId] = useState<string | undefined>()
  const [deleteTarget, setDeleteTarget] = useState<ConversationSummary | undefined>()

  const { data: list, isLoading: loadingList } = useConversations()
  const { data: detail, isLoading: loadingDetail } = useConversation(activeId)
  const createConversation = useCreateConversation()
  const deleteConversation = useDeleteConversation()
  const { streaming, draft, liveToolCalls, pendingUserMessage, send, stop } = useChatStream(activeId)
  const { degraded } = useChatDegradeStore()

  const conversations = list?.items ?? []

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await deleteConversation.mutateAsync(deleteTarget.id)
      if (activeId === deleteTarget.id) setActiveId(undefined)
      toast.success('Percakapan dihapus.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal menghapus percakapan.')
    } finally {
      setDeleteTarget(undefined)
    }
  }

  async function handleSend(content: string, attachment?: ChatAttachment) {
    if (!content.trim()) return
    let convId = activeId

    if (!convId) {
      const conv = await createConversation.mutateAsync({ first_message: content })
      convId = conv.id
      setActiveId(convId)
    }

    await send(convId, content, attachment)
  }

  const activeTitle = detail?.title ?? (activeId ? '…' : 'Chat')

  return (
    <div className="flex -m-6 h-[calc(100%+3rem)] overflow-hidden">
      {/* ── Left panel: conversation list ──────────────────────────────── */}
      <aside
        className="w-64 shrink-0 border-r border-line bg-surface flex flex-col"
        aria-label="Daftar percakapan"
      >
        <div className="p-3 border-b border-line flex items-center justify-between gap-2">
          <h2 className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
            Percakapan
          </h2>
          <Tooltip content="Percakapan baru">
            <button
              type="button"
              aria-label="Percakapan baru"
              onClick={() => setActiveId(undefined)}
              className={cn(
                'p-1.5 rounded-btn text-fg-muted hover:bg-surface-subtle hover:text-fg transition-colors',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'
              )}
            >
              <Plus className="w-4 h-4" aria-hidden="true" />
            </button>
          </Tooltip>
        </div>

        <nav className="flex-1 overflow-y-auto py-2">
          {loadingList ? (
            <div className="px-3 space-y-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} variant="text" className="h-10 w-full" />
              ))}
            </div>
          ) : conversations.length === 0 ? (
            <EmptyState
              title="Belum ada percakapan"
              description="Ketik pesan di bawah untuk mulai"
              className="py-8 px-4"
            />
          ) : (
            conversations.map((conv) => (
              <div
                key={conv.id}
                className={cn(
                  'group relative w-full flex items-center hover:bg-surface-subtle transition-colors',
                  activeId === conv.id && 'bg-primary/5 border-l-2 border-primary',
                )}
              >
                <button
                  type="button"
                  onClick={() => setActiveId(conv.id)}
                  className="flex-1 min-w-0 text-left px-4 py-2.5 flex flex-col gap-0.5"
                >
                  <span className="text-sm font-medium text-fg truncate">{conv.title}</span>
                  <span className="text-caption text-fg-subtle">
                    {formatRelative(conv.updated_at)}
                  </span>
                </button>
                <button
                  type="button"
                  aria-label={`Hapus percakapan "${conv.title}"`}
                  onClick={(e) => {
                    e.stopPropagation()
                    setDeleteTarget(conv)
                  }}
                  className={cn(
                    'shrink-0 mr-2 p-1.5 rounded-btn text-fg-subtle opacity-0 group-hover:opacity-100',
                    'hover:bg-surface hover:text-danger transition-colors',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:opacity-100'
                  )}
                >
                  <Trash2 className="w-3.5 h-3.5" aria-hidden="true" />
                </button>
              </div>
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
              pendingUserMessage={pendingUserMessage}
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

      <ConfirmDialog
        open={!!deleteTarget}
        onCancel={() => setDeleteTarget(undefined)}
        onConfirm={handleDelete}
        title="Hapus percakapan?"
        description={`Percakapan "${deleteTarget?.title}" beserta seluruh isinya akan dihapus permanen.`}
        tone="danger"
        confirmLabel="Hapus"
        loading={deleteConversation.isPending}
      />
    </div>
  )
}
