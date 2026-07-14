import { useState } from 'react'
import { Sparkles, Copy, Download } from 'lucide-react'

import Badge from './ui/Badge'
import Button from './ui/Button'
import Select from './ui/Select'
import AiCallout from './ui/AiCallout'
import { formatRelative } from '../lib/format'
import { toast } from '../lib/toast'
import { playbookToMarkdown } from '../lib/playbookFormat'
import { usePlaybooks, useGeneratePlaybook } from '../api/playbooks'
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

function PlaybookSections({ playbook }: { playbook: Playbook }) {
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
        <h4 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mb-1">Timeline</h4>
        <ol className="list-decimal pl-4 text-body text-fg space-y-0.5">
          {c.timeline.map((t, i) => (
            <li key={i}>{t}</li>
          ))}
        </ol>
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
  const [compareId, setCompareId] = useState<string>('')

  async function handleGenerate() {
    try {
      await generate.mutateAsync(targetId)
      toast.success('Playbook berhasil dibuat.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Generate playbook gagal, coba lagi nanti.')
    }
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
          stakeholder, strategi, timeline, risiko, dan next actions.
        </p>
        <div className="mt-3">
          <Button
            size="sm"
            variant="secondary"
            leftIcon={<Sparkles className="w-3.5 h-3.5" />}
            loading={generate.isPending}
            onClick={handleGenerate}
          >
            Generate Playbook
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

      <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-accent/10">
        <Button
          size="sm"
          variant="ghost"
          leftIcon={<Sparkles className="w-3.5 h-3.5" />}
          loading={generate.isPending}
          onClick={handleGenerate}
        >
          Generate versi baru
        </Button>

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
