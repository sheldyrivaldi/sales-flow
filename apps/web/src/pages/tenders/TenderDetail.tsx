import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router'
import { ChevronRight, Sparkles, ExternalLink, AlertTriangle, FileText } from 'lucide-react'

import Tabs, { TabPanel } from '../../components/ui/Tabs'
import { StagePill } from '../../components/ui/Badge'
import Button from '../../components/ui/Button'
import Modal from '../../components/ui/Modal'
import Select from '../../components/ui/Select'
import Textarea from '../../components/ui/Textarea'
import Field from '../../components/ui/Field'
import EmptyState from '../../components/ui/EmptyState'
import Skeleton, { SkeletonText } from '../../components/ui/Skeleton'
import AiScorePanel from '../../components/AiScorePanel'
import PlaybookPanel from '../../components/PlaybookPanel'
import DocChecklistCard from '../../components/tenders/DocChecklistCard'
import ProposalDraftDrawer from '../../components/tenders/ProposalDraftDrawer'
import TenderFormDrawer from './TenderFormDrawer'

import { toast } from '../../lib/toast'
import { formatRupiah, formatTanggal, formatRelative } from '../../lib/format'
import { cn } from '../../lib/cn'

import {
  useTender,
  useUpdateTenderStatus,
  useRecordOutcome,
} from '../../api/tenders'
import type { Tender, TenderStatus } from '../../api/tenders'

const TENDER_TABS = [
  { id: 'ringkasan', label: 'Ringkasan' },
  { id: 'analisa', label: 'Analisa AI' },
  { id: 'playbook', label: 'Playbook' },
  { id: 'timeline', label: 'Timeline' },
]

const VALID_TRANSITIONS: Record<TenderStatus, TenderStatus[]> = {
  IDENTIFIED: ['QUALIFYING'],
  QUALIFYING: ['BIDDING'],
  BIDDING: ['SUBMITTED'],
  SUBMITTED: [],
  WON: [],
  LOST: [],
}

function deadlineTone(deadline: string | null): 'normal' | 'warning' | 'danger' {
  if (!deadline) return 'normal'
  const diffMs = new Date(deadline).getTime() - Date.now()
  const diffDays = diffMs / (1000 * 60 * 60 * 24)
  if (diffMs < 0) return 'danger'
  if (diffDays <= 7) return 'warning'
  return 'normal'
}

function SummaryRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex gap-3 py-2 border-b border-line last:border-0">
      <span className="w-36 text-caption text-fg-muted shrink-0">{label}</span>
      <span className="text-body text-fg">{children}</span>
    </div>
  )
}

function TenderSummaryCard({ tender }: { tender: Tender }) {
  const tone = deadlineTone(tender.submission_deadline)
  return (
    <div className="bg-surface border border-line rounded-card p-4 flex flex-col">
      <h3 className="text-h3 font-semibold text-fg mb-3">Ringkasan</h3>
      <SummaryRow label="Buyer">
        {tender.buyer_name ?? <span className="text-fg-subtle">—</span>}
      </SummaryRow>
      {(tender.buyer_country || tender.buyer_industry) && (
        <SummaryRow label="Negara / Industri">
          {[tender.buyer_country, tender.buyer_industry].filter(Boolean).join(' / ')}
        </SummaryRow>
      )}
      <SummaryRow label="Nilai Estimasi">
        {tender.value_estimate != null
          ? <span className="font-medium">{formatRupiah(tender.value_estimate)} {tender.currency}</span>
          : <span className="text-fg-subtle">—</span>}
      </SummaryRow>
      <SummaryRow label="Deadline">
        {tender.submission_deadline ? (
          <span className={cn(
            'font-medium',
            tone === 'danger' && 'text-danger',
            tone === 'warning' && 'text-warning',
          )}>
            {tone !== 'normal' && <AlertTriangle className="w-3.5 h-3.5 inline mr-1" />}
            {formatTanggal(tender.submission_deadline)}
            <span className="text-fg-muted font-normal ml-1">
              ({formatRelative(tender.submission_deadline)})
            </span>
          </span>
        ) : <span className="text-fg-subtle">—</span>}
      </SummaryRow>
      <SummaryRow label="Sumber">
        {tender.source_url ? (
          <a
            href={tender.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-primary hover:underline"
          >
            {tender.source_name ?? tender.source_url}
            <ExternalLink className="w-3.5 h-3.5" />
          </a>
        ) : tender.source_name ?? <span className="text-fg-subtle">—</span>}
      </SummaryRow>
      {tender.service_category && (
        <SummaryRow label="Kategori">{tender.service_category}</SummaryRow>
      )}
      <SummaryRow label="Origin">
        {tender.origin === 'discovery' ? (
          <span className="inline-flex items-center gap-1 text-accent">
            <Sparkles className="w-3.5 h-3.5" />
            Ditemukan AI{tender.source_name ? ` (${tender.source_name})` : ''}
          </span>
        ) : 'Manual'}
      </SummaryRow>
      {tender.scope_summary && (
        <SummaryRow label="Scope">{tender.scope_summary}</SummaryRow>
      )}
      {tender.eligibility_requirements && (
        <SummaryRow label="Syarat Kelayakan">{tender.eligibility_requirements}</SummaryRow>
      )}
      {tender.technical_requirements && (
        <SummaryRow label="Syarat Teknis">{tender.technical_requirements}</SummaryRow>
      )}
    </div>
  )
}

export default function TenderDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('ringkasan')
  const [editOpen, setEditOpen] = useState(false)
  const [proposalOpen, setProposalOpen] = useState(false)
  const [outcomeOpen, setOutcomeOpen] = useState(false)
  const [outcomeResult, setOutcomeResult] = useState<'WON' | 'LOST'>('WON')
  const [outcomeNotes, setOutcomeNotes] = useState('')

  const { data: tender, isLoading, error } = useTender(id)
  const updateStatusMutation = useUpdateTenderStatus()
  const recordOutcomeMutation = useRecordOutcome()

  async function handleStatusChange(status: TenderStatus) {
    if (!id) return
    try {
      await updateStatusMutation.mutateAsync({ id, status })
      toast.success(`Status diubah ke ${status}.`)
    } catch {
      toast.error('Gagal mengubah status.')
    }
  }

  async function handleOutcome() {
    if (!id) return
    try {
      await recordOutcomeMutation.mutateAsync({
        id,
        result: outcomeResult,
        notes: outcomeNotes || undefined,
      })
      toast.success(`Tender ditandai ${outcomeResult} dan hasilnya dipelajari AI.`)
      setOutcomeOpen(false)
      setOutcomeNotes('')
    } catch {
      toast.error('Gagal merekam outcome.')
    }
  }

  if (isLoading) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <Skeleton className="h-8 w-64" />
        <div className="grid grid-cols-2 gap-6">
          <div className="flex flex-col gap-2">
            {Array.from({ length: 6 }).map((_, i) => <SkeletonText key={i} lines={1} />)}
          </div>
          <Skeleton className="h-48" />
        </div>
      </div>
    )
  }

  if (error || !tender) {
    return (
      <div className="p-6">
        <EmptyState
          title="Tender tidak ditemukan"
          description="Tender ini mungkin sudah dihapus atau URL tidak valid."
          action={
            <Button variant="secondary" onClick={() => navigate('/tenders')}>
              Kembali ke daftar
            </Button>
          }
        />
      </div>
    )
  }

  const nextStatuses = VALID_TRANSITIONS[tender.status] ?? []
  const isTerminal = tender.status === 'WON' || tender.status === 'LOST'

  return (
    <div className="flex flex-col gap-0">
      {/* Header */}
      <div className="px-6 pt-5 pb-4 border-b border-line">
        {/* Breadcrumb */}
        <div className="flex items-center gap-1.5 text-caption text-fg-muted mb-2">
          <Link to="/tenders" className="hover:text-primary">Tenders</Link>
          <ChevronRight className="w-3.5 h-3.5" />
          <span className="text-fg truncate max-w-xs">{tender.title}</span>
        </div>

        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div className="flex items-center gap-3">
            <h1 className="text-h2 font-semibold text-fg">{tender.title}</h1>
            <StagePill stage={tender.status} />
          </div>

          <div className="flex items-center gap-2 flex-wrap">
            <Button variant="secondary" size="sm" onClick={() => setEditOpen(true)}>
              Edit
            </Button>

            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Sparkles className="w-3.5 h-3.5" />}
              onClick={() => setActiveTab('playbook')}
            >
              Playbook
            </Button>

            <Button
              variant="secondary"
              size="sm"
              leftIcon={<FileText className="w-3.5 h-3.5" />}
              onClick={() => setProposalOpen(true)}
            >
              Generate Proposal
            </Button>

            {/* Status change */}
            {nextStatuses.length > 0 && (
              <div className="flex gap-1">
                {nextStatuses.map((s) => (
                  <Button
                    key={s}
                    variant="secondary"
                    size="sm"
                    loading={updateStatusMutation.isPending}
                    onClick={() => handleStatusChange(s)}
                  >
                    → {s}
                  </Button>
                ))}
              </div>
            )}

            {/* WON / LOST */}
            {!isTerminal && (
              <Button
                size="sm"
                variant="primary"
                onClick={() => setOutcomeOpen(true)}
              >
                WON / LOST
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="px-6 pt-4">
        <Tabs tabs={TENDER_TABS} value={activeTab} onChange={setActiveTab} />
      </div>

      {/* Tab content */}
      <div className="px-6 py-4">
        <TabPanel id="ringkasan">
          {activeTab === 'ringkasan' && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <TenderSummaryCard tender={tender} />
              <AiScorePanel targetType="tender" targetId={tender.id} tender={tender} />
            </div>
          )}
        </TabPanel>

        <TabPanel id="analisa">
          {activeTab === 'analisa' && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
              <AiScorePanel targetType="tender" targetId={tender.id} tender={tender} />
              <DocChecklistCard tenderId={tender.id} />
            </div>
          )}
        </TabPanel>

        <TabPanel id="playbook">
          {activeTab === 'playbook' && (
            <div className="max-w-2xl">
              <PlaybookPanel targetType="tender" targetId={tender.id} />
            </div>
          )}
        </TabPanel>

        <TabPanel id="timeline">
          {activeTab === 'timeline' && (
            <EmptyState
              title="Timeline belum tersedia"
              description="Timeline aktivitas akan ditambahkan di sprint berikutnya."
            />
          )}
        </TabPanel>
      </div>

      {/* Form Drawer */}
      <TenderFormDrawer
        open={editOpen}
        onClose={() => setEditOpen(false)}
        tender={tender}
        onSaved={() => setEditOpen(false)}
      />

      {/* Draf Proposal */}
      <ProposalDraftDrawer
        open={proposalOpen}
        onClose={() => setProposalOpen(false)}
        tenderId={tender.id}
        tenderTitle={tender.title}
      />

      {/* Outcome Modal */}
      <Modal
        open={outcomeOpen}
        onClose={() => setOutcomeOpen(false)}
        title="Rekam Hasil Tender"
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setOutcomeOpen(false)}>
              Batal
            </Button>
            <Button
              loading={recordOutcomeMutation.isPending}
              onClick={handleOutcome}
            >
              Simpan
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-4">
          <Field label="Hasil" htmlFor="outcome-result">
            <Select
              id="outcome-result"
              value={outcomeResult}
              onChange={(e) => setOutcomeResult(e.target.value as 'WON' | 'LOST')}
            >
              <option value="WON">WON — Berhasil</option>
              <option value="LOST">LOST — Tidak berhasil</option>
            </Select>
          </Field>
          <Field label="Catatan (opsional)" htmlFor="outcome-notes">
            <Textarea
              id="outcome-notes"
              value={outcomeNotes}
              onChange={(e) => setOutcomeNotes(e.target.value)}
              rows={3}
              placeholder="Alasan, pembelajaran, atau catatan tambahan…"
            />
          </Field>
          <p className="text-caption text-fg-muted flex items-center gap-1">
            <Sparkles className="w-3.5 h-3.5 text-accent" />
            AI akan belajar dari hasil ini untuk meningkatkan rekomendasi berikutnya.
          </p>
        </div>
      </Modal>
    </div>
  )
}
