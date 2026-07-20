import { Link } from 'react-router'
import { Briefcase, Wallet, TrendingUp, CalendarClock, ArrowRight } from 'lucide-react'

import StatCard from '../../components/ui/StatCard'
import Card, { CardHeader, CardBody } from '../../components/ui/Card'
import Badge from '../../components/ui/Badge'
import Button from '../../components/ui/Button'
import EmptyState from '../../components/ui/EmptyState'
import Skeleton from '../../components/ui/Skeleton'
import { formatRupiahShort, formatTanggal } from '../../lib/format'
import { cn } from '../../lib/cn'
import { useProjects, useProjectSummary, PROJECT_STATUS_LABELS } from '../../api/projects'
import type { Project, ProjectStatus } from '../../api/projects'

const STATUS_TONE: Record<ProjectStatus, 'success' | 'warning' | 'danger' | 'info'> = {
  ON_TRACK: 'success',
  AT_RISK: 'warning',
  DELAYED: 'danger',
  COMPLETED: 'info',
}

function ProgressBar({ value }: { value: number }) {
  return (
    <div className="h-1.5 w-full rounded-pill bg-surface-subtle overflow-hidden">
      <div
        className={cn(
          'h-full rounded-pill transition-all',
          value >= 70 ? 'bg-success' : value >= 40 ? 'bg-primary' : 'bg-warning'
        )}
        style={{ width: `${Math.min(value, 100)}%` }}
      />
    </div>
  )
}

/** Ringkasan Proyek Berjalan: kesehatan portofolio proyek dalam sekali
 * pandang — jumlah & nilai proyek aktif, sebaran status, dan daftar proyek
 * yang butuh perhatian (berisiko/terlambat). */
export default function OngoingSummary() {
  const { data: summary, isLoading: loadingSummary } = useProjectSummary()
  const { data: listData, isLoading: loadingList } = useProjects()

  const needAttention: Project[] =
    listData?.items.filter((p) => p.status === 'AT_RISK' || p.status === 'DELAYED') ?? []

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-h2 font-semibold text-fg">Ringkasan Proyek Berjalan</h1>
          <p className="text-caption text-fg-muted mt-0.5">
            Kesehatan seluruh proyek yang sedang dikerjakan.
          </p>
        </div>
        <Link to="/ongoing/projects">
          <Button variant="secondary" size="sm" rightIcon={<ArrowRight className="w-4 h-4" />}>
            Daftar Proyek
          </Button>
        </Link>
      </div>

      {loadingSummary ? (
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      ) : summary ? (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              label="Proyek Aktif"
              value={summary.total_active}
              icon={<Briefcase className="w-4 h-4" />}
              hint={`${summary.completed} selesai`}
            />
            <StatCard
              label="Nilai Kontrak Aktif"
              value={formatRupiahShort(summary.total_value)}
              icon={<Wallet className="w-4 h-4" />}
              hint="total berjalan"
            />
            <StatCard
              label="Rata-rata Progress"
              value={`${summary.avg_progress}%`}
              icon={<TrendingUp className="w-4 h-4" />}
              hint="proyek aktif"
            />
            <StatCard
              label="Berakhir ≤ 30 Hari"
              value={summary.ending_soon}
              icon={<CalendarClock className="w-4 h-4" />}
              hint="perlu persiapan closing"
            />
          </div>

          {/* Sebaran status */}
          <Card>
            <CardHeader>
              <h2 className="text-body font-semibold text-fg">Sebaran Status</h2>
            </CardHeader>
            <CardBody className="flex flex-wrap gap-3">
              {(
                [
                  ['ON_TRACK', summary.on_track],
                  ['AT_RISK', summary.at_risk],
                  ['DELAYED', summary.delayed],
                  ['COMPLETED', summary.completed],
                ] as [ProjectStatus, number][]
              ).map(([st, count]) => (
                <div
                  key={st}
                  className="flex items-center gap-2 rounded-card border border-line bg-surface px-3 py-2"
                >
                  <Badge tone={STATUS_TONE[st]}>{PROJECT_STATUS_LABELS[st]}</Badge>
                  <span className="text-h3 font-semibold text-fg tabular-nums">{count}</span>
                </div>
              ))}
            </CardBody>
          </Card>
        </>
      ) : null}

      {/* Butuh perhatian */}
      <Card>
        <CardHeader>
          <h2 className="text-body font-semibold text-fg">Butuh Perhatian</h2>
        </CardHeader>
        <CardBody>
          {loadingList ? (
            <Skeleton className="h-20" />
          ) : needAttention.length === 0 ? (
            <EmptyState
              title="Semua proyek sehat"
              description="Tidak ada proyek berstatus berisiko atau terlambat."
              className="py-6"
            />
          ) : (
            <div className="flex flex-col divide-y divide-line">
              {needAttention.map((p) => (
                <Link
                  key={p.id}
                  to="/ongoing/projects"
                  className="flex items-center gap-4 py-3 hover:bg-surface-subtle rounded-btn px-2 -mx-2 transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-body font-medium text-fg truncate">{p.name}</p>
                    <p className="text-caption text-fg-muted truncate">
                      {p.client_name ?? '—'}
                      {p.end_date && ` • target selesai ${formatTanggal(p.end_date)}`}
                    </p>
                  </div>
                  <div className="w-32 shrink-0">
                    <ProgressBar value={p.progress} />
                    <p className="text-caption text-fg-subtle mt-0.5 text-right tabular-nums">{p.progress}%</p>
                  </div>
                  <Badge tone={STATUS_TONE[p.status]}>{PROJECT_STATUS_LABELS[p.status]}</Badge>
                </Link>
              ))}
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
