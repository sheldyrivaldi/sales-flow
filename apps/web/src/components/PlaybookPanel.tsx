import { useRef, useState } from 'react'
import { Sparkles, Copy, Download, Presentation, SendHorizonal, Upload } from 'lucide-react'

import Badge from './ui/Badge'
import Button from './ui/Button'
import Select from './ui/Select'
import Input from './ui/Input'
import AiCallout from './ui/AiCallout'
import { formatRelative } from '../lib/format'
import { toast } from '../lib/toast'
import { playbookToMarkdown } from '../lib/playbookFormat'
import { exportPlaybookPpt } from '../lib/exportPlaybookPpt'
import { useAIBusy, AI_MUTATION_KEYS } from '../lib/aiMutation'
import PlaybookGantt from './playbooks/PlaybookGantt'
import {
  usePlaybooks,
  useGeneratePlaybook,
  useGeneratePlaybookFromDocument,
  useRefinePlaybook,
} from '../api/playbooks'
import type { PlaybookTargetType, Playbook } from '../api/playbooks'

export interface PlaybookPanelProps {
  targetType: PlaybookTargetType
  targetId: string
}

function downloadMarkdown(filename: string, markdown: string) {
  const blob = new Blob([markdown], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export function PlaybookSections({ playbook }: { playbook: Playbook }) {
  const c = playbook.content
  return (
    <div className="flex flex-col gap-3">
      <section>
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Ringkasan</h4>
        <p className="text-body text-fg">{c.summary}</p>
      </section>
      <section>
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Value Proposition</h4>
        <p className="text-body text-fg">{c.value_prop}</p>
      </section>
      <section>
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Stakeholders</h4>
        <ul className="list-disc pl-4 text-body text-fg space-y-0.5">
          {c.stakeholders.map((s, i) => (
            <li key={i}>{s}</li>
          ))}
        </ul>
      </section>
      <section>
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Strategi</h4>
        <ul className="flex flex-col gap-1">
          {c.strategy_checklist.map((s, i) => (
            <li key={i} className="flex items-start gap-2 text-body text-fg">
              <span className="mt-1 w-3.5 h-3.5 shrink-0 rounded border border-line" aria-hidden="true" />
              {s}
            </li>
          ))}
        </ul>
      </section>
      <section>
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Rencana Kerja (Timeline)</h4>
        {c.timeline_plan && c.timeline_plan.length > 0 ? (
          <PlaybookGantt items={c.timeline_plan} />
        ) : (
          <ol className="list-decimal pl-4 text-body text-fg space-y-0.5">
            {c.timeline.map((t, i) => (
              <li key={i}>{t}</li>
            ))}
          </ol>
        )}
      </section>
      <section>
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Risiko</h4>
        <ul className="list-disc pl-4 text-body text-danger space-y-0.5">
          {c.risks.map((r, i) => (
            <li key={i}>{r}</li>
          ))}
        </ul>
      </section>
      <section>
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Next Actions</h4>
        <ul className="list-disc pl-4 text-body text-fg space-y-0.5">
          {c.next_actions.map((a, i) => (
            <li key={i}>{a}</li>
          ))}
        </ul>
      </section>
    </div>
  )
}

/** AI-generated, immutable-versioned playbook viewer for one tender/prospect
 * (EP-14). Mirrors AiScorePanel's shape: empty state with a generate CTA,
 * populated state with the latest version + actions (generate new version,
 * compare against an older version, copy/export markdown). */
export default function PlaybookPanel({ targetType, targetId }: PlaybookPanelProps) {
  const { data: playbooks, isLoading } = usePlaybooks(targetType, targetId)
  const generate = useGeneratePlaybook(targetType)
  const fromDocument = useGeneratePlaybookFromDocument(targetType)
  const refine = useRefinePlaybook()
  const [compareId, setCompareId] = useState<string>('')
  const [instruction, setInstruction] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const aiBusy = useAIBusy(AI_MUTATION_KEYS.playbook, targetId)
  const busy = generate.isPending || fromDocument.isPending || refine.isPending || aiBusy

  // Toast sukses/gagal ditangani MutationCache global (lihat main.tsx) —
  // tetap muncul meski user sudah pindah halaman saat AI selesai.
  function handleGenerate() {
    generate.mutate(targetId)
  }

  function handleFromDocument(file: File | undefined) {
    if (!file) return
    if (file.size > 10 * 1024 * 1024) {
      toast.error('Ukuran dokumen maksimal 10 MB.')
      return
    }
    fromDocument.mutate({ id: targetId, file })
  }

  function handleRefine(latest: Playbook) {
    const text = instruction.trim()
    if (!text) return
    refine.mutate({ playbookId: latest.id, instruction: text, targetId })
    setInstruction('')
  }

  function handleCopy(playbook: Playbook) {
    const md = playbookToMarkdown(playbook.content, playbook.version)
    navigator.clipboard.writeText(md).then(
      () => toast.success('Playbook disalin ke clipboard.'),
      () => toast.error('Gagal menyalin ke clipboard.'),
    )
  }

  function handleExport(playbook: Playbook) {
    const md = playbookToMarkdown(playbook.content, playbook.version)
    downloadMarkdown(`playbook-${targetType}-${targetId}-v${playbook.version}.md`, md)
  }

  if (isLoading) {
    return (
      <div className="rounded-card border border-line bg-surface p-4 animate-pulse">
        <div className="h-4 w-40 rounded bg-surface-subtle" />
      </div>
    )
  }

  if (!playbooks || playbooks.length === 0) {
    return (
      <AiCallout title="Belum ada playbook">
        <p className="mt-1 text-body text-fg-muted">
          Generate playbook untuk mendapatkan strategi terstruktur: ringkasan, value proposition,
          stakeholder, strategi, timeline, risiko, dan next actions — atau susun dari dokumen yang
          sudah kamu punya (proposal lama, playbook existing, notulen strategi).
        </p>
        <input
          ref={fileInputRef}
          type="file"
          accept=".pdf"
          className="sr-only"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => {
            handleFromDocument(e.target.files?.[0])
            e.target.value = ''
          }}
        />
        <div className="mt-3 flex flex-wrap gap-2">
          <Button
            size="sm"
            variant="secondary"
            leftIcon={<Sparkles className="w-3.5 h-3.5" />}
            loading={generate.isPending}
            disabled={busy}
            onClick={handleGenerate}
          >
            Generate Playbook
          </Button>
          <Button
            size="sm"
            variant="secondary"
            leftIcon={<Upload className="w-3.5 h-3.5" />}
            loading={fromDocument.isPending}
            disabled={busy}
            onClick={() => fileInputRef.current?.click()}
          >
            Dari Dokumen (PDF)
          </Button>
        </div>
      </AiCallout>
    )
  }

  const latest = playbooks[0]
  const olderVersions = playbooks.slice(1)
  const compared = compareId ? playbooks.find((p) => p.id === compareId) : undefined

  return (
    <div className="rounded-card border border-accent/20 bg-accent/5 p-4 flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Badge tone="accent">v{latest.version}</Badge>
          <p className="text-caption text-fg-muted">
            Dibuat AI{latest.model ? ` • ${latest.model}` : ''} • {formatRelative(latest.created_at)}
          </p>
        </div>
        <div className="flex items-center gap-1.5">
          <Button size="sm" variant="ghost" leftIcon={<Copy className="w-3.5 h-3.5" />} onClick={() => handleCopy(latest)}>
            Salin
          </Button>
          <Button size="sm" variant="ghost" leftIcon={<Download className="w-3.5 h-3.5" />} onClick={() => handleExport(latest)}>
            Export
          </Button>
          {targetType === 'event' && (
            <Button
              size="sm"
              variant="ghost"
              leftIcon={<Presentation className="w-3.5 h-3.5" />}
              onClick={() => void exportPlaybookPpt(latest.content, `Playbook Event`).catch(() => toast.error('Ekspor PPT gagal.'))}
            >
              PPT
            </Button>
          )}
        </div>
      </div>

      <div className={compared ? 'grid md:grid-cols-2 gap-4' : undefined}>
        <div className={compared ? 'border-r border-accent/10 md:pr-4' : undefined}>
          {compared && <p className="text-caption font-semibold text-fg-muted mb-2">Versi terbaru (v{latest.version})</p>}
          <PlaybookSections playbook={latest} />
        </div>
        {compared && (
          <div>
            <p className="text-caption font-semibold text-fg-muted mb-2">Dibandingkan (v{compared.version})</p>
            <PlaybookSections playbook={compared} />
          </div>
        )}
      </div>

      {/* Revisi via prompt — bagian yang tidak disinggung dipertahankan;
          hasil selalu dipersist sebagai versi baru. */}
      <div className="flex flex-col gap-1.5 pt-2 border-t border-accent/10">
        <label htmlFor={`refine-${targetId}`} className="text-caption font-semibold text-fg-muted uppercase tracking-wide">
          Revisi dengan prompt
        </label>
        <div className="flex gap-2">
          <Input
            id={`refine-${targetId}`}
            value={instruction}
            onChange={(e) => setInstruction(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                handleRefine(latest)
              }
            }}
            disabled={busy}
            placeholder="mis. tambahkan mitigasi risiko keamanan data, persingkat timeline jadi 3 bulan…"
          />
          <Button
            size="md"
            loading={refine.isPending}
            disabled={busy || !instruction.trim()}
            onClick={() => handleRefine(latest)}
            leftIcon={<SendHorizonal className="w-3.5 h-3.5" />}
            aria-label="Kirim instruksi revisi"
          >
            Revisi
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-accent/10">
        <div className="flex flex-wrap items-center gap-1.5">
          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf"
            className="sr-only"
            tabIndex={-1}
            aria-hidden="true"
            onChange={(e) => {
              handleFromDocument(e.target.files?.[0])
              e.target.value = ''
            }}
          />
          <Button
            size="sm"
            variant="ghost"
            leftIcon={<Sparkles className="w-3.5 h-3.5" />}
            loading={generate.isPending}
            disabled={busy}
            onClick={handleGenerate}
          >
            Generate versi baru
          </Button>
          <Button
            size="sm"
            variant="ghost"
            leftIcon={<Upload className="w-3.5 h-3.5" />}
            loading={fromDocument.isPending}
            disabled={busy}
            onClick={() => fileInputRef.current?.click()}
          >
            Dari Dokumen
          </Button>
        </div>

        {olderVersions.length > 0 && (
          <div className="flex items-center gap-2">
            <span className="text-caption text-fg-muted">Bandingkan:</span>
            <Select value={compareId} onChange={(e) => setCompareId(e.target.value)} className="w-auto text-caption">
              <option value="">— pilih versi —</option>
              {olderVersions.map((p) => (
                <option key={p.id} value={p.id}>
                  v{p.version} • {formatRelative(p.created_at)}
                </option>
              ))}
            </Select>
          </div>
        )}
      </div>
    </div>
  )
}
