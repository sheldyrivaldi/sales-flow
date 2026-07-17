import { useState } from 'react'
import { BookOpen, Presentation, SendHorizonal, Sparkles } from 'lucide-react'

import Card, { CardHeader, CardBody } from '../../components/ui/Card'
import Button from '../../components/ui/Button'
import Badge from '../../components/ui/Badge'
import Input from '../../components/ui/Input'
import EmptyState from '../../components/ui/EmptyState'
import { SkeletonText } from '../../components/ui/Skeleton'
import { PlaybookSections } from '../../components/PlaybookPanel'
import { formatRelative } from '../../lib/format'
import { toast } from '../../lib/toast'
import { cn } from '../../lib/cn'
import { exportPlaybookPpt } from '../../lib/exportPlaybookPpt'
import { useAIBusy, AI_MUTATION_KEYS } from '../../lib/aiMutation'
import {
  useCustomPlaybooks,
  useGenerateCustomPlaybook,
  useRefinePlaybook,
} from '../../api/playbooks'
import type { Playbook } from '../../api/playbooks'

/** Menu Playbooks: playbook CUSTOM yang berdiri sendiri (bukan turunan
 * tender/prospek/event) — dibuat dari topik bebas, direvisi via prompt, dan
 * bisa diekspor ke PowerPoint. Playbook per-peluang tetap digenerate dari
 * halaman masing-masing (detail tender/prospek/event). */
export default function PlaybooksIndex() {
  const { data: playbooks, isLoading } = useCustomPlaybooks()
  const generate = useGenerateCustomPlaybook()
  const refine = useRefinePlaybook()
  const aiBusy = useAIBusy(AI_MUTATION_KEYS.playbook)

  const [topic, setTopic] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [instruction, setInstruction] = useState('')

  const busy = generate.isPending || refine.isPending || aiBusy
  const items = playbooks ?? []
  const selected: Playbook | undefined =
    items.find((p) => p.id === selectedId) ?? items[0]

  function handleGenerate() {
    const t = topic.trim()
    if (!t) return
    generate.mutate({ topic: t })
    setTopic('')
  }

  function handleRefine() {
    if (!selected) return
    const text = instruction.trim()
    if (!text) return
    refine.mutate({ playbookId: selected.id, instruction: text, targetId: selected.target_id })
    setInstruction('')
  }

  return (
    <div className="flex flex-col gap-6 p-6 max-w-5xl">
      <div>
        <h1 className="text-h2 font-semibold text-fg">Playbooks</h1>
        <p className="text-body text-fg-muted mt-1">
          Playbook custom untuk inisiatif apa pun — playbook per-peluang dibuat dari halaman detail
          tender, prospek, atau event masing-masing.
        </p>
      </div>

      {/* Buat playbook custom dari topik */}
      <Card>
        <CardBody className="flex flex-col gap-2">
          <label htmlFor="playbook-topic" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
            Buat playbook custom
          </label>
          <div className="flex gap-2">
            <Input
              id="playbook-topic"
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  handleGenerate()
                }
              }}
              disabled={busy}
              placeholder="mis. strategi masuk sektor kesehatan, rencana partnership cloud provider…"
            />
            <Button
              loading={generate.isPending}
              disabled={busy || !topic.trim()}
              leftIcon={<Sparkles className="w-4 h-4" />}
              onClick={handleGenerate}
            >
              Generate
            </Button>
          </div>
        </CardBody>
      </Card>

      {isLoading ? (
        <SkeletonText lines={6} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={<BookOpen className="w-6 h-6" />}
          title="Belum ada playbook custom"
          description="Tulis topik di atas untuk membuat playbook pertama."
        />
      ) : (
        <div className="grid lg:grid-cols-[260px_1fr] gap-4 items-start">
          {/* Daftar */}
          <div className="flex flex-col gap-1.5">
            {items.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => setSelectedId(p.id)}
                className={cn(
                  'text-left rounded-card border px-3 py-2.5 transition-colors',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
                  selected?.id === p.id
                    ? 'border-primary-border bg-primary-subtle'
                    : 'border-line bg-surface hover:bg-surface-subtle',
                )}
              >
                <p className="text-body font-medium text-fg line-clamp-2">{p.content.summary || 'Playbook'}</p>
                <p className="text-caption text-fg-subtle mt-0.5">
                  v{p.version} • {formatRelative(p.created_at)}
                </p>
              </button>
            ))}
          </div>

          {/* Detail terpilih */}
          {selected && (
            <Card>
              <CardHeader className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <Badge tone="accent">v{selected.version}</Badge>
                  <span className="text-caption text-fg-muted">{formatRelative(selected.created_at)}</span>
                </div>
                <Button
                  size="sm"
                  variant="secondary"
                  leftIcon={<Presentation className="w-3.5 h-3.5" />}
                  onClick={() =>
                    void exportPlaybookPpt(selected.content, selected.content.summary.slice(0, 60) || 'Playbook Custom')
                      .catch(() => toast.error('Ekspor PPT gagal.'))
                  }
                >
                  Export PPT
                </Button>
              </CardHeader>
              <CardBody className="flex flex-col gap-4">
                <PlaybookSections playbook={selected} />

                {/* Revisi via prompt — versi baru menggantikan tampilan */}
                <div className="flex flex-col gap-1.5 pt-3 border-t border-line">
                  <label htmlFor="playbook-refine" className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
                    Revisi dengan prompt
                  </label>
                  <div className="flex gap-2">
                    <Input
                      id="playbook-refine"
                      value={instruction}
                      onChange={(e) => setInstruction(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault()
                          handleRefine()
                        }
                      }}
                      disabled={busy}
                      placeholder="mis. tambahkan strategi pricing, fokuskan ke BUMN…"
                    />
                    <Button
                      loading={refine.isPending}
                      disabled={busy || !instruction.trim()}
                      leftIcon={<SendHorizonal className="w-3.5 h-3.5" />}
                      onClick={handleRefine}
                    >
                      Revisi
                    </Button>
                  </div>
                </div>
              </CardBody>
            </Card>
          )}
        </div>
      )}
    </div>
  )
}
