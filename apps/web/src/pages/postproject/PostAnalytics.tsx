import { Star, MessagesSquare, ThumbsUp, Inbox } from 'lucide-react'

import StatCard from '../../components/ui/StatCard'
import Card, { CardHeader, CardBody } from '../../components/ui/Card'
import StarRating from '../../components/ui/StarRating'
import EmptyState from '../../components/ui/EmptyState'
import Skeleton from '../../components/ui/Skeleton'
import { cn } from '../../lib/cn'
import { useFeedbackAnalytics } from '../../api/feedback'

function AspectBar({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex items-center gap-3">
      <span className="w-36 shrink-0 text-body text-fg">{label}</span>
      <div className="flex-1 h-2 rounded-pill bg-surface-subtle overflow-hidden">
        <div
          className={cn('h-full rounded-pill', value >= 4 ? 'bg-success' : value >= 3 ? 'bg-primary' : 'bg-warning')}
          style={{ width: `${(value / 5) * 100}%` }}
        />
      </div>
      <span className="w-8 text-right text-body font-medium text-fg tabular-nums">
        {value > 0 ? value.toFixed(1) : '—'}
      </span>
    </div>
  )
}

/** Analisa Feedback (Pasca-Proyek): agregat semua jawaban client — rating
 * rata-rata, NPS, distribusi bintang, rata-rata per aspek, dan komentar. */
export default function PostAnalytics() {
  const { data, isLoading } = useFeedbackAnalytics()

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 p-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      </div>
    )
  }

  if (!data || data.total_responses === 0) {
    return (
      <div className="flex flex-col gap-6 p-6">
        <h1 className="text-h2 font-semibold text-fg">Analisa Feedback</h1>
        <EmptyState
          icon={<Inbox className="w-6 h-6" />}
          title="Belum ada feedback masuk"
          description="Bagikan link feedback dari halaman Feedback Client; analisa akan muncul setelah client mengisi."
        />
      </div>
    )
  }

  const maxDist = Math.max(...data.rating_distribution, 1)
  const responseRate =
    data.total_requests > 0 ? Math.round((data.total_responses / data.total_requests) * 100) : 0

  return (
    <div className="flex flex-col gap-6 p-6">
      <div>
        <h1 className="text-h2 font-semibold text-fg">Analisa Feedback</h1>
        <p className="text-caption text-fg-muted mt-0.5">
          Ringkasan kepuasan client dari seluruh feedback pasca-proyek.
        </p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Rating Rata-rata"
          value={data.avg_overall.toFixed(1)}
          icon={<Star className="w-4 h-4" />}
          hint="dari 5"
        />
        <StatCard
          label="NPS"
          value={data.nps}
          icon={<ThumbsUp className="w-4 h-4" />}
          hint="skor -100 sampai 100"
        />
        <StatCard
          label="Respon Masuk"
          value={`${data.total_responses}/${data.total_requests}`}
          icon={<MessagesSquare className="w-4 h-4" />}
          hint={`${responseRate}% response rate`}
        />
        <StatCard
          label="Komentar"
          value={data.comments?.length ?? 0}
          icon={<MessagesSquare className="w-4 h-4" />}
          hint="dengan tulisan"
        />
      </div>

      <div className="grid lg:grid-cols-2 gap-4 items-start">
        {/* Distribusi rating */}
        <Card>
          <CardHeader>
            <h2 className="text-body font-semibold text-fg">Distribusi Rating</h2>
          </CardHeader>
          <CardBody className="flex flex-col gap-2">
            {[5, 4, 3, 2, 1].map((n) => {
              const count = data.rating_distribution[n - 1] ?? 0
              return (
                <div key={n} className="flex items-center gap-3">
                  <span className="w-10 shrink-0 text-body text-fg tabular-nums flex items-center gap-1">
                    {n} <Star className="w-3 h-3 fill-amber-400 text-amber-400" aria-hidden="true" />
                  </span>
                  <div className="flex-1 h-3 rounded-pill bg-surface-subtle overflow-hidden">
                    <div
                      className="h-full rounded-pill bg-amber-400"
                      style={{ width: `${(count / maxDist) * 100}%` }}
                    />
                  </div>
                  <span className="w-6 text-right text-caption text-fg-muted tabular-nums">{count}</span>
                </div>
              )
            })}
          </CardBody>
        </Card>

        {/* Rata-rata aspek */}
        <Card>
          <CardHeader>
            <h2 className="text-body font-semibold text-fg">Rata-rata per Aspek</h2>
          </CardHeader>
          <CardBody className="flex flex-col gap-3">
            <AspectBar label="Kualitas hasil" value={data.avg_quality} />
            <AspectBar label="Komunikasi" value={data.avg_communication} />
            <AspectBar label="Ketepatan waktu" value={data.avg_timeliness} />
          </CardBody>
        </Card>
      </div>

      {/* Komentar */}
      {data.comments && data.comments.length > 0 && (
        <Card>
          <CardHeader>
            <h2 className="text-body font-semibold text-fg">Komentar Client</h2>
          </CardHeader>
          <CardBody className="flex flex-col divide-y divide-line">
            {data.comments.map((c, i) => (
              <div key={i} className="py-3 first:pt-0 last:pb-0">
                <div className="flex items-center justify-between gap-2 mb-1">
                  <p className="text-body font-medium text-fg truncate">
                    {c.project_name}
                    {c.client_name && <span className="text-fg-muted font-normal"> • {c.client_name}</span>}
                  </p>
                  <StarRating value={c.rating} />
                </div>
                <p className="text-body text-fg-muted">"{c.comment}"</p>
              </div>
            ))}
          </CardBody>
        </Card>
      )}
    </div>
  )
}
