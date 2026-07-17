import { useEffect, useRef } from 'react'
import { Copy, Download, FileText, RefreshCw } from 'lucide-react'
import Drawer from '../ui/Drawer'
import Button from '../ui/Button'
import { SkeletonText } from '../ui/Skeleton'
import EmptyState from '../ui/EmptyState'
import { toast } from '../../lib/toast'
import { slugify } from '../../lib/format'
import { useProposalDraft } from '../../api/tenders'
import type { ProposalDraft } from '../../api/tenders'

function draftToMarkdown(draft: ProposalDraft): string {
  const parts = [`# ${draft.title}`, '']
  for (const s of draft.sections) {
    parts.push(`## ${s.title}`, '', s.content, '')
  }
  parts.push('---', `> ${draft.disclaimer}`)
  return parts.join('\n')
}

export interface ProposalDraftDrawerProps {
  open: boolean
  onClose: () => void
  tenderId?: string
  tenderTitle?: string
}

/** Draf proposal terstandarisasi (10 bagian baku) untuk satu tender —
 * digenerate otomatis saat drawer dibuka; bisa disalin sebagai markdown atau
 * diunduh .md sebagai bahan awal tim proposal. */
export default function ProposalDraftDrawer({ open, onClose, tenderId, tenderTitle }: ProposalDraftDrawerProps) {
  const proposal = useProposalDraft()
  // Satu generate per pembukaan drawer per tender — bukan per render.
  const generatedForRef = useRef<string | null>(null)

  useEffect(() => {
    if (open && tenderId && generatedForRef.current !== tenderId) {
      generatedForRef.current = tenderId
      proposal.mutate(tenderId, {
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : 'Generate proposal gagal, coba lagi nanti.'),
      })
    }
    if (!open) generatedForRef.current = null
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mutate identity stabil dari react-query; trigger hanya open/tenderId.
  }, [open, tenderId])

  const draft = proposal.data

  function handleCopy() {
    if (!draft) return
    navigator.clipboard.writeText(draftToMarkdown(draft)).then(
      () => toast.success('Draf proposal disalin ke clipboard.'),
      () => toast.error('Gagal menyalin ke clipboard.'),
    )
  }

  function handleDownload() {
    if (!draft) return
    const blob = new Blob([draftToMarkdown(draft)], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `proposal-${slugify(tenderTitle ?? draft.title)}.md`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width="w-[640px]"
      title={
        <span className="flex items-center gap-2">
          <FileText className="w-4 h-4 text-primary" aria-hidden="true" />
          Draf Proposal
        </span>
      }
      footer={
        draft ? (
          <>
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<RefreshCw className="w-3.5 h-3.5" />}
              loading={proposal.isPending}
              onClick={() => tenderId && proposal.mutate(tenderId)}
            >
              Generate Ulang
            </Button>
            <Button variant="secondary" size="sm" leftIcon={<Copy className="w-3.5 h-3.5" />} onClick={handleCopy}>
              Salin
            </Button>
            <Button size="sm" leftIcon={<Download className="w-3.5 h-3.5" />} onClick={handleDownload}>
              Unduh .md
            </Button>
          </>
        ) : undefined
      }
    >
      {proposal.isPending && (
        <div className="flex flex-col gap-4">
          <p className="text-caption text-fg-muted">
            AI menyusun draf proposal terstandarisasi untuk tender ini…
          </p>
          <SkeletonText lines={12} />
        </div>
      )}

      {!proposal.isPending && !draft && (
        <EmptyState
          title="Draf belum tersedia"
          description="Generate gagal atau belum dijalankan."
          action={
            <Button size="sm" onClick={() => tenderId && proposal.mutate(tenderId)}>
              Coba Lagi
            </Button>
          }
        />
      )}

      {!proposal.isPending && draft && (
        <article className="flex flex-col gap-5">
          <h2 className="text-h3 font-semibold text-fg">{draft.title}</h2>
          {draft.sections.map((s, i) => (
            <section key={i}>
              <h3 className="text-body font-semibold text-fg mb-1">
                {i + 1}. {s.title}
              </h3>
              <p className="text-body text-fg-muted whitespace-pre-line">{s.content}</p>
            </section>
          ))}
          <p className="text-caption text-fg-subtle border-t border-line pt-3">{draft.disclaimer}</p>
        </article>
      )}
    </Drawer>
  )
}
