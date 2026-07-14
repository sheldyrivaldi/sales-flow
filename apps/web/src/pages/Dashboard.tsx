import { Link } from 'react-router'
import { Users, Wallet, Sparkles, Trophy } from 'lucide-react'

import OtakAgentBanner from '../components/OtakAgentBanner'
import StatCard from '../components/ui/StatCard'
import Card, { CardHeader, CardBody } from '../components/ui/Card'
import ScoreRing from '../components/ui/ScoreRing'
import { ActionBadge } from '../components/ui/Badge'
import AiCallout from '../components/ui/AiCallout'
import EmptyState from '../components/ui/EmptyState'
import Skeleton, { SkeletonText } from '../components/ui/Skeleton'

import { formatRupiahShort } from '../lib/format'
import { actionToLabel } from '../api/tenders'
import { useDashboardSummary } from '../api/dashboard'

export default function Dashboard() {
  const { data, isLoading, isError } = useDashboardSummary()

  return (
    <div className="flex flex-col gap-6">
      <OtakAgentBanner />

      <h1 className="text-h2 font-semibold text-fg">Dashboard</h1>

      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
      ) : isError ? (
        <EmptyState
          title="Gagal memuat dashboard"
          description="Terjadi kesalahan saat mengambil ringkasan. Coba muat ulang halaman."
        />
      ) : data ? (
        <>
          {/* Stat cards */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <StatCard
              label="Total Pipeline"
              value={data.total_pipeline_count}
              icon={<Users className="w-4 h-4" />}
              hint="prospek aktif"
            />
            <StatCard
              label="Estimasi Revenue"
              value={formatRupiahShort(data.total_pipeline_value)}
              icon={<Wallet className="w-4 h-4" />}
              hint="total pipeline"
            />
            <StatCard
              label="Penemuan AI Hari Ini"
              value={data.discovery_today_count}
              icon={<Sparkles className="w-4 h-4" />}
              hint="tender baru"
            />
          </div>

          {/* AI insight */}
          <AiCallout
            title="Ringkasan AI"
            meta={`${data.total_pipeline_count} prospek di pipeline • Rp senilai ${formatRupiahShort(data.total_pipeline_value)}`}
          >
            {data.priority_tenders.length > 0 ? (
              <p>
                Ada {data.priority_tenders.length} tender berskor tinggi yang layak diprioritaskan
                minggu ini.
              </p>
            ) : (
              <p>Belum ada tender berskor tinggi. Jalankan analisa AI pada tender yang tersedia.</p>
            )}
          </AiCallout>

          {/* Pipeline per stage */}
          <Card>
            <CardHeader>
              <h2 className="text-body font-semibold text-fg">Pipeline per Stage</h2>
            </CardHeader>
            <CardBody>
              {data.pipeline.length === 0 ? (
                <p className="text-body text-fg-muted">Belum ada prospek.</p>
              ) : (
                <div className="flex flex-col gap-2">
                  {data.pipeline.map((p) => (
                    <div key={p.stage} className="flex items-center justify-between text-body">
                      <span className="text-fg-muted">{p.stage}</span>
                      <span className="font-medium text-fg tabular-nums">
                        {p.count} • {formatRupiahShort(p.total_value)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </CardBody>
          </Card>

          {/* Priority tenders */}
          <Card>
            <CardHeader>
              <h2 className="text-body font-semibold text-fg flex items-center gap-1.5">
                <Trophy className="w-4 h-4 text-accent" /> Prioritas
              </h2>
            </CardHeader>
            <CardBody>
              {data.priority_tenders.length === 0 ? (
                <EmptyState
                  icon={<Sparkles className="w-6 h-6" />}
                  title="Belum ada tender berskor"
                  description="Jalankan analisa AI pada tender untuk melihat prioritas di sini."
                />
              ) : (
                <div className="flex flex-col gap-3">
                  {data.priority_tenders.map((t) => (
                    <Link
                      key={t.id}
                      to={`/tenders/${t.id}`}
                      className="flex items-center gap-3 rounded-btn p-2 -mx-2 hover:bg-surface-subtle transition-colors"
                    >
                      <ScoreRing score={t.fit_score ?? 0} size={40} strokeWidth={4} />
                      <div className="flex-1 min-w-0">
                        <p className="text-body font-medium text-fg truncate">{t.title}</p>
                        {t.buyer_name && (
                          <p className="text-caption text-fg-muted truncate">{t.buyer_name}</p>
                        )}
                      </div>
                      {t.recommended_action && (
                        <ActionBadge action={actionToLabel(t.recommended_action)} />
                      )}
                    </Link>
                  ))}
                </div>
              )}
            </CardBody>
          </Card>
        </>
      ) : (
        <SkeletonText lines={4} />
      )}
    </div>
  )
}
