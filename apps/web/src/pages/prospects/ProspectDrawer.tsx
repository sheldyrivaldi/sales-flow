import { useState } from 'react'
import { Sparkles, ExternalLink } from 'lucide-react'
import { Link } from 'react-router'

import Drawer from '../../components/ui/Drawer'
import { StagePill } from '../../components/ui/Badge'
import Badge from '../../components/ui/Badge'
import Avatar from '../../components/ui/Avatar'
import Button from '../../components/ui/Button'
import Select from '../../components/ui/Select'
import EmptyState from '../../components/ui/EmptyState'
import Skeleton, { SkeletonText } from '../../components/ui/Skeleton'
import OutcomeNotesModal from '../../components/prospects/OutcomeNotesModal'
import AiScorePanel from '../../components/AiScorePanel'
import PlaybookPanel from '../../components/PlaybookPanel'
import ScoreRing from '../../components/ui/ScoreRing'

import { formatRupiahShort } from '../../lib/format'
import { toast } from '../../lib/toast'
import { useAskAIStore } from '../../store/askAI'

import {
  useProspect,
  useUpdateProspectStage,
  PROSPECT_STAGES,
  SOURCE_LABELS,
  isTerminalStage,
} from '../../api/prospects'
import type { ProspectStage } from '../../api/prospects'
import { useUsers } from '../../api/users'
import { useScore } from '../../api/scores'

export interface ProspectDrawerProps {
  open: boolean
  onClose: () => void
  prospectId?: string
}

function SummaryRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex gap-3 py-2 border-b border-line last:border-0">
      <span className="w-28 text-caption text-fg-muted shrink-0">{label}</span>
      <span className="text-body text-fg">{children}</span>
    </div>
  )
}

/** Detail drawer prospect (Design §4.9): header + Info + Analisa AI (EP-10)
 * + Playbook (EP-14) + Timeline (placeholder) + aksi cepat. */
export default function ProspectDrawer({ open, onClose, prospectId }: ProspectDrawerProps) {
  const { data: prospect, isLoading } = useProspect(prospectId)
  const { data: score } = useScore('prospect', prospectId)
  const updateStageMutation = useUpdateProspectStage()
  const openAskAI = useAskAIStore((s) => s.openAskAI)
  const { data: usersData } = useUsers()
  const ownerName = usersData?.items.find((u) => u.id === prospect?.owner_user_id)?.name

  const [outcomeTarget, setOutcomeTarget] = useState<'WON' | 'LOST' | null>(null)
  const [outcomeNotes, setOutcomeNotes] = useState('')

  async function handleStageChange(stage: ProspectStage) {
    if (!prospect) return
    if (isTerminalStage(stage)) {
      setOutcomeNotes('')
      setOutcomeTarget(stage)
      return
    }
    try {
      await updateStageMutation.mutateAsync({ id: prospect.id, stage })
      toast.success(`Stage diubah ke ${stage}.`)
    } catch {
      // onError hook sudah menampilkan toast.error
    }
  }

  async function confirmOutcome() {
    if (!prospect || !outcomeTarget) return
    try {
      await updateStageMutation.mutateAsync({
        id: prospect.id,
        stage: outcomeTarget,
        notes: outcomeNotes || undefined,
      })
      toast.success(`Prospek ditandai ${outcomeTarget}. AI akan belajar dari hasil ini.`)
    } catch {
      // onError hook sudah menampilkan toast.error
    } finally {
      setOutcomeTarget(null)
      setOutcomeNotes('')
    }
  }

  function handleAskAI() {
    if (!prospect) return
    openAskAI({ type: 'prospect', id: prospect.id, label: prospect.name })
  }

  const sourceLink =
    prospect?.source_type === 'tender' && prospect.source_id
      ? `/tenders/${prospect.source_id}`
      : prospect?.source_type === 'event' && prospect.source_id
        ? `/events/${prospect.source_id}`
        : null

  return (
    <Drawer open={open} onClose={onClose} title="Detail Prospek" width="w-[480px]">
      {isLoading ? (
        <div className="flex flex-col gap-4 p-1">
          <Skeleton className="h-8 w-48" />
          <SkeletonText lines={4} />
        </div>
      ) : !prospect ? (
        <EmptyState title="Prospek tidak ditemukan" description="Prospek ini mungkin sudah dihapus." />
      ) : (
        <div className="flex flex-col gap-6">
          {/* Header */}
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-3 min-w-0">
              {score && <ScoreRing score={score.fit_score} size={40} strokeWidth={4} />}
              <div className="min-w-0">
                <h2 className="text-h3 font-semibold text-fg truncate">{prospect.name}</h2>
                {prospect.company && (
                  <p className="text-caption text-fg-muted truncate">{prospect.company}</p>
                )}
                <div className="mt-1.5">
                  <StagePill stage={prospect.stage} />
                </div>
              </div>
            </div>
            {prospect.owner_user_id && <Avatar name={ownerName ?? prospect.owner_user_id} size="md" />}
          </div>

          {/* Info */}
          <div>
            <h3 className="text-body font-semibold text-fg mb-2">Info</h3>
            <SummaryRow label="Sumber">
              <div className="flex items-center gap-2">
                <Badge tone="info" appearance="soft">
                  {SOURCE_LABELS[prospect.source_type]}
                </Badge>
                {sourceLink && (
                  <Link to={sourceLink} className="inline-flex items-center gap-1 text-primary hover:underline text-caption">
                    Lihat sumber <ExternalLink className="w-3.5 h-3.5" />
                  </Link>
                )}
              </div>
            </SummaryRow>
            {prospect.contact_info && (
              <SummaryRow label="Kontak">{prospect.contact_info}</SummaryRow>
            )}
            <SummaryRow label="Nilai Estimasi">
              {prospect.est_value != null ? (
                <span className="font-medium">{formatRupiahShort(prospect.est_value)}</span>
              ) : (
                <span className="text-fg-subtle">—</span>
              )}
            </SummaryRow>
          </div>

          {/* Analisa AI */}
          <div>
            <h3 className="text-body font-semibold text-fg mb-2 flex items-center gap-1.5">
              <Sparkles className="w-3.5 h-3.5 text-accent" /> Analisa AI
            </h3>
            <AiScorePanel targetType="prospect" targetId={prospect.id} />
          </div>

          {/* Playbook */}
          <div>
            <h3 className="text-body font-semibold text-fg mb-2">Playbook</h3>
            <PlaybookPanel targetType="prospect" targetId={prospect.id} />
          </div>

          {/* Timeline (placeholder) */}
          <div>
            <h3 className="text-body font-semibold text-fg mb-2">Timeline</h3>
            <p className="text-caption text-fg-muted">Timeline aktivitas akan ditambahkan di sprint berikutnya.</p>
          </div>

          {/* Aksi cepat */}
          <div className="flex flex-col gap-3 pt-2 border-t border-line">
            <h3 className="text-body font-semibold text-fg">Aksi Cepat</h3>

            <Select
              value={prospect.stage}
              onChange={(e) => handleStageChange(e.target.value as ProspectStage)}
              disabled={updateStageMutation.isPending}
            >
              {PROSPECT_STAGES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </Select>

            <div className="flex gap-2">
              <Button
                variant="secondary"
                className="flex-1"
                disabled={prospect.stage === 'WON'}
                onClick={() => {
                  setOutcomeNotes('')
                  setOutcomeTarget('WON')
                }}
              >
                WON
              </Button>
              <Button
                variant="secondary"
                className="flex-1"
                disabled={prospect.stage === 'LOST'}
                onClick={() => {
                  setOutcomeNotes('')
                  setOutcomeTarget('LOST')
                }}
              >
                LOST
              </Button>
            </div>

            <Button
              variant="primary"
              leftIcon={<Sparkles className="w-4 h-4" />}
              onClick={handleAskAI}
            >
              Tanya AI tentang prospek ini
            </Button>
          </div>
        </div>
      )}

      {/* WON/LOST — modal catatan opsional (komponen bersama, lihat ProspectBoard) */}
      <OutcomeNotesModal
        open={!!outcomeTarget}
        stage={outcomeTarget}
        notes={outcomeNotes}
        onNotesChange={setOutcomeNotes}
        loading={updateStageMutation.isPending}
        onConfirm={confirmOutcome}
        onCancel={() => setOutcomeTarget(null)}
      />
    </Drawer>
  )
}
