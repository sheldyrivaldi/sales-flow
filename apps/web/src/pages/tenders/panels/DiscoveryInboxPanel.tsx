import { useState } from 'react'
import { Link } from 'react-router'
import { Sparkles, ChevronRight } from 'lucide-react'

import Button from '../../../components/ui/Button'
import Select from '../../../components/ui/Select'
import Card from '../../../components/ui/Card'
import ScoreRing from '../../../components/ui/ScoreRing'
import { ActionBadge } from '../../../components/ui/Badge'
import { RiskFlagList } from '../../../components/ui/RiskFlag'
import EmptyState from '../../../components/ui/EmptyState'
import Skeleton from '../../../components/ui/Skeleton'
import Modal from '../../../components/ui/Modal'
import Field from '../../../components/ui/Field'
import Textarea from '../../../components/ui/Textarea'

import { formatRupiahShort, formatTanggal } from '../../../lib/format'
import { cn } from '../../../lib/cn'
import { actionToLabel, usePromoteTender, useReviewTender } from '../../../api/tenders'
import type { Tender, TenderApiAction } from '../../../api/tenders'
import { useDiscoveryInbox, useDiscoveryRuns } from '../../../api/discovery'
import { useProfile, isProfileConfigured } from '../../../api/profile'
import { toast } from '../../../lib/toast'

function deadlineTone(deadline: string | null): 'normal' | 'warning' | 'danger' {
  if (!deadline) return 'normal'
  const diffMs = new Date(deadline).getTime() - Date.now()
  const diffDays = diffMs / (1000 * 60 * 60 * 24)
  if (diffMs < 0) return 'danger'
  if (diffDays <= 7) return 'warning'
  return 'normal'
}

function riskFlagItems(riskFlags: unknown): { label: string }[] {
  if (!Array.isArray(riskFlags)) return []
  return riskFlags.filter((f): f is string => typeof f === 'string').map((label) => ({ label }))
}

interface DiscoveryCardProps {
  tender: Tender
  onPursue: (tender: Tender) => void
  onWatchlist: (tender: Tender) => void
  onReject: (tender: Tender) => void
  pending: boolean
}

function DiscoveryCard({ tender, onPursue, onWatchlist, onReject, pending }: DiscoveryCardProps) {
  const tone = deadlineTone(tender.submission_deadline)
  return (
    <Card className="p-4 flex flex-col gap-2">
      <div className="flex items-start gap-3">
        {tender.fit_score != null && <ScoreRing score={tender.fit_score} size={48} strokeWidth={5} />}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            {tender.recommended_action && (
              <ActionBadge action={actionToLabel(tender.recommended_action)} />
            )}
            <h3 className="text-body font-semibold text-fg truncate">{tender.title}</h3>
          </div>
          {tender.buyer_name && (
            <p className="text-caption text-fg-muted truncate">{tender.buyer_name}</p>
          )}
          <div className="flex items-center gap-2 flex-wrap mt-1 text-caption text-fg-muted">
            {tender.source_name && <span>Sumber: {tender.source_name}</span>}
            {tender.submission_deadline && (
              <span
                className={cn(
                  'font-medium',
                  tone === 'danger' && 'text-danger',
                  tone === 'warning' && 'text-warning',
                )}
              >
                • {tone !== 'normal' && '⚠ '}Deadline {formatTanggal(tender.submission_deadline)}
              </span>
            )}
            {tender.value_estimate != null && <span>• {formatRupiahShort(tender.value_estimate)}</span>}
          </div>
        </div>
      </div>

      {tender.reasoning_summary && (
        <p className="text-caption text-fg-muted line-clamp-1">{tender.reasoning_summary}</p>
      )}

      <RiskFlagList items={riskFlagItems(tender.risk_flags)} />

      <div className="flex items-center gap-2 pt-1 flex-wrap">
        <Link
          to={`/tenders/${tender.id}`}
          className="inline-flex items-center gap-1 text-caption font-medium text-primary hover:underline mr-auto"
        >
          Tinjau <ChevronRight className="w-3.5 h-3.5" />
        </Link>
        <Button size="sm" variant="primary" disabled={pending} onClick={() => onPursue(tender)}>
          Pursue
        </Button>
        <Button size="sm" variant="secondary" disabled={pending} onClick={() => onWatchlist(tender)}>
          Watchlist
        </Button>
        <Button size="sm" variant="ghost" disabled={pending} onClick={() => onReject(tender)}>
          Tolak
        </Button>
      </div>
    </Card>
  )
}

const MIN_SCORE_OPTIONS = [
  { value: '', label: 'Semua Skor' },
  { value: '80', label: '≥ 80 (Pursue)' },
  { value: '65', label: '≥ 65 (Review)' },
  { value: '50', label: '≥ 50 (Watchlist)' },
]

/** Panel "Penemuan AI" — kartu triage tender temuan crawler yang belum
 * ditinjau (Pursue/Watchlist/Tolak). Self-contained: status run crawling
 * dibaca lewat useDiscoveryRuns (di-dedupe react-query dengan header
 * TendersPage), jadi panel ini tak butuh prop dari induk. Tombol "Cari Tender
 * dengan AI" dipindah ke header TendersPage. */
export default function DiscoveryInboxPanel() {
  const [recommendedAction, setRecommendedAction] = useState<TenderApiAction | ''>('')
  const [minScore, setMinScore] = useState('')
  const [rejectTarget, setRejectTarget] = useState<Tender | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [pendingId, setPendingId] = useState<string | null>(null)

  const { data: profile } = useProfile()
  const profileConfigured = isProfileConfigured(profile)

  const { data: runsData } = useDiscoveryRuns({ refetchInterval: 3000 })
  const latestRun = runsData?.items[0]
  const isRunning = latestRun?.status === 'pending' || latestRun?.status === 'running'

  const { data: inboxData, isLoading: inboxLoading } = useDiscoveryInbox({
    recommended_action: recommendedAction || undefined,
    min_score: minScore ? Number(minScore) : undefined,
  })

  const promoteMutation = usePromoteTender()
  const reviewMutation = useReviewTender()

  async function handlePursue(tender: Tender) {
    setPendingId(tender.id)
    try {
      await promoteMutation.mutateAsync(tender.id)
      toast.success('Tender dipromosikan ke pipeline.')
    } catch {
      toast.error('Gagal mempromosikan tender.')
    } finally {
      setPendingId(null)
    }
  }

  async function handleWatchlist(tender: Tender) {
    setPendingId(tender.id)
    try {
      await reviewMutation.mutateAsync({ id: tender.id })
      toast.success('Ditandai untuk dipantau (Watchlist).')
    } catch {
      toast.error('Gagal menandai tender.')
    } finally {
      setPendingId(null)
    }
  }

  function openReject(tender: Tender) {
    setRejectReason('')
    setRejectTarget(tender)
  }

  async function confirmReject() {
    if (!rejectTarget) return
    setPendingId(rejectTarget.id)
    try {
      await reviewMutation.mutateAsync({ id: rejectTarget.id, reason: rejectReason || undefined })
      toast.success('Tender ditolak dan alasannya dipelajari AI.')
    } catch {
      toast.error('Gagal menolak tender.')
    } finally {
      setPendingId(null)
      setRejectTarget(null)
      setRejectReason('')
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Filter bar */}
      <div className="flex flex-wrap gap-3">
        <Select
          className="w-48"
          value={recommendedAction}
          onChange={(e) => setRecommendedAction(e.target.value as TenderApiAction | '')}
        >
          <option value="">Semua Rekomendasi</option>
          <option value="PURSUE">Pursue</option>
          <option value="REVIEW">Review</option>
          <option value="WATCHLIST">Watchlist</option>
          <option value="REJECT">Reject</option>
          <option value="NEED_PARTNER">Need Partner</option>
        </Select>

        <Select className="w-44" value={minScore} onChange={(e) => setMinScore(e.target.value)}>
          {MIN_SCORE_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </Select>
      </div>

      {/* Content */}
      {!profileConfigured ? (
        <EmptyState
          icon={<Sparkles className="w-6 h-6" />}
          title="Lengkapi Otak Agent agar AI mulai mencari"
          description="Isi profil perusahaan, kriteria target, dan sumber pencarian terlebih dahulu."
          action={
            <Link to="/onboarding">
              <Button size="sm">Lengkapi sekarang</Button>
            </Link>
          }
        />
      ) : isRunning ? (
        <div className="flex flex-col gap-3">
          <p className="text-body text-fg-muted flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-accent animate-pulse" />
            AI sedang mencari di {latestRun?.source_ids.length ?? 0} sumber…
          </p>
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
      ) : inboxLoading ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
      ) : !inboxData || inboxData.items.length === 0 ? (
        <EmptyState
          icon={<Sparkles className="w-6 h-6" />}
          title="Belum ada peluang baru"
          description="Coba longgarkan kriteria atau tambah sumber pencarian, lalu jalankan “Cari Tender dengan AI”."
        />
      ) : (
        <div className="flex flex-col gap-3">
          {inboxData.items.map((tender) => (
            <DiscoveryCard
              key={tender.id}
              tender={tender}
              onPursue={handlePursue}
              onWatchlist={handleWatchlist}
              onReject={openReject}
              pending={pendingId === tender.id}
            />
          ))}
        </div>
      )}

      {/* Tolak — alasan singkat (opsional, untuk pembelajaran EP-16) */}
      <Modal
        open={!!rejectTarget}
        onClose={() => setRejectTarget(null)}
        title={`Tolak "${rejectTarget?.title ?? ''}"`}
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setRejectTarget(null)}>
              Batal
            </Button>
            <Button variant="danger" loading={reviewMutation.isPending} onClick={confirmReject}>
              Tolak
            </Button>
          </>
        }
      >
        <Field label="Alasan (opsional)" htmlFor="reject-reason-textarea">
          <Textarea
            id="reject-reason-textarea"
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            rows={3}
            placeholder="Mis. nilai terlalu kecil, tidak sesuai kapabilitas…"
          />
        </Field>
        <p className="mt-2 text-caption text-fg-muted flex items-center gap-1">
          <Sparkles className="w-3.5 h-3.5 text-accent" />
          AI akan belajar dari alasan ini untuk penemuan berikutnya.
        </p>
      </Modal>
    </div>
  )
}
