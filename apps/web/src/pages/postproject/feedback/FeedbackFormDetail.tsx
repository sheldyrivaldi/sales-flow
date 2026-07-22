import { useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import {
  Pencil, Link as LinkIcon, FileSpreadsheet, Send, Inbox, Sparkles,
  ThumbsUp, TrendingDown, Lightbulb, Tags, Hash, Type as TypeIcon, ListChecks, Gauge,
} from 'lucide-react'

import Card, { CardHeader, CardBody } from '../../../components/ui/Card'
import Button from '../../../components/ui/Button'
import Badge from '../../../components/ui/Badge'
import Table from '../../../components/ui/Table'
import type { Column } from '../../../components/ui/Table'
import Tabs, { TabPanel } from '../../../components/ui/Tabs'
import EmptyState from '../../../components/ui/EmptyState'
import Skeleton from '../../../components/ui/Skeleton'
import StatCard from '../../../components/ui/StatCard'
import { toast } from '../../../lib/toast'
import { formatTanggal } from '../../../lib/format'
import { cn } from '../../../lib/cn'
import { copyToClipboard } from '../../../lib/clipboard'
import type { Tone } from '../../../lib/score'
import {
  useFeedbackForm,
  useFeedbackSubmissions,
  useFeedbackFormAnalytics,
  usePublishFeedbackForm,
  useAnalyzeFeedback,
  publicFormLink,
} from '../../../api/feedbackForms'
import type {
  FeedbackForm, FeedbackFormStatus, FeedbackSubmission, FormAnalytics, QuestionStat, FeedbackInsight,
  FeedbackQuestion, QuestionType, FormLanguage,
} from '../../../api/feedbackForms'
import { exportFeedbackExcel } from '../../../lib/exportFeedbackExcel'

const TYPE_ICON: Record<QuestionType, typeof Hash> = {
  rating: Hash, text: TypeIcon, choice: ListChecks, nps: Gauge,
}
const TYPE_LABEL: Record<QuestionType, string> = {
  rating: 'Rating (angka)', text: 'Teks bebas', choice: 'Pilihan ganda', nps: 'NPS (0 sampai 10)',
}

const STATUS_META: Record<FeedbackFormStatus, { label: string; tone: Tone; solid?: boolean }> = {
  draft: { label: 'Draft', tone: 'info' },
  published: { label: 'Terbit', tone: 'success', solid: true },
  closed: { label: 'Ditutup', tone: 'warning' },
  processing_ai: { label: 'AI Memproses…', tone: 'accent' },
  need_clarification: { label: 'Perlu Klarifikasi', tone: 'warning', solid: true },
}

function ratingDefaults(lang: FormLanguage): { min: string; max: string } {
  return lang === 'en'
    ? { min: 'Very poor', max: 'Excellent' }
    : { min: 'Sangat kurang', max: 'Sangat baik' }
}

// ── Read-only preview (tab "Pertanyaan") ──────────────────────────────────────

function QuestionPreview({ q, lang }: { q: FeedbackQuestion; lang: FormLanguage }) {
  const Icon = TYPE_ICON[q.type]
  const def = ratingDefaults(lang)
  return (
    <div className="rounded-card border border-line bg-surface p-3 flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <Badge tone="info" className="gap-1 shrink-0">
          <Icon className="w-3 h-3" /> {TYPE_LABEL[q.type]}
        </Badge>
        {q.required && <Badge tone="danger">Wajib</Badge>}
      </div>
      <p className="text-body font-medium text-fg">{q.label || <span className="text-fg-subtle italic">(tanpa label)</span>}</p>
      {q.description && <p className="text-caption text-fg-muted">{q.description}</p>}

      {q.type === 'rating' && (
        <div className="flex items-center gap-3">
          <span className="text-caption text-fg-subtle w-28 truncate text-right">{q.min_label || def.min}</span>
          <div className="flex-1 flex justify-between">
            {Array.from({ length: q.scale ?? 5 }).map((_, i) => (
              <span key={i} className="w-7 h-7 rounded-btn border border-line flex items-center justify-center text-caption tabular-nums text-fg-muted">
                {i + 1}
              </span>
            ))}
          </div>
          <span className="text-caption text-fg-subtle w-28 truncate">{q.max_label || def.max}</span>
        </div>
      )}
      {q.type === 'nps' && <p className="text-caption text-fg-subtle">Skor 0 sampai 10.</p>}
      {q.type === 'choice' && (
        <ul className="flex flex-col gap-1">
          {(q.options ?? []).map((o, i) => (
            <li key={i} className="text-caption text-fg-muted flex items-center gap-2">
              <span className={cn('w-3.5 h-3.5 border border-line-strong shrink-0', q.multiple ? 'rounded' : 'rounded-full')} />
              {o}
            </li>
          ))}
          {q.multiple && <li className="text-caption text-fg-subtle">Boleh pilih lebih dari satu.</li>}
        </ul>
      )}
    </div>
  )
}

// ── Charts (tab "Hasil & Analisa") ────────────────────────────────────────────

function RatingBars({ stat }: { stat: QuestionStat }) {
  const dist = stat.distribution ?? []
  const max = Math.max(...dist, 1)
  const rows = dist.map((_, i) => dist.length - i)
  return (
    <div className="flex flex-col gap-1.5">
      {rows.map((n) => {
        const c = dist[n - 1] ?? 0
        return (
          <div key={n} className="flex items-center gap-3">
            <span className="w-6 shrink-0 text-caption text-fg tabular-nums text-right">{n}</span>
            <div className="flex-1 h-3 rounded-pill bg-surface-subtle overflow-hidden">
              <div className="h-full rounded-pill bg-amber-400" style={{ width: `${(c / max) * 100}%` }} />
            </div>
            <span className="w-6 text-right text-caption text-fg-muted tabular-nums">{c}</span>
          </div>
        )
      })}
    </div>
  )
}

function ChoiceBars({ stat }: { stat: QuestionStat }) {
  const entries = Object.entries(stat.choices ?? {})
  const max = Math.max(...entries.map(([, v]) => v), 1)
  if (entries.length === 0) return <p className="text-caption text-fg-subtle">Belum ada jawaban.</p>
  return (
    <div className="flex flex-col gap-1.5">
      {entries.map(([opt, count]) => (
        <div key={opt} className="flex items-center gap-3">
          <span className="w-32 shrink-0 text-caption text-fg truncate" title={opt}>{opt}</span>
          <div className="flex-1 h-3 rounded-pill bg-surface-subtle overflow-hidden">
            <div className="h-full rounded-pill bg-primary" style={{ width: `${(count / max) * 100}%` }} />
          </div>
          <span className="w-6 text-right text-caption text-fg-muted tabular-nums">{count}</span>
        </div>
      ))}
    </div>
  )
}

function QuestionStatCard({ stat }: { stat: QuestionStat }) {
  return (
    <Card>
      <CardHeader className="flex items-center justify-between gap-2">
        <h3 className="text-body font-medium text-fg">{stat.label}</h3>
        <Badge tone="info">{stat.responses} respon</Badge>
      </CardHeader>
      <CardBody>
        {stat.type === 'rating' && (
          <div className="flex flex-col gap-3">
            <p className="text-h3 font-semibold text-fg">
              {stat.average ? stat.average.toFixed(1) : '—'}
              <span className="text-caption text-fg-muted font-normal"> rata-rata</span>
            </p>
            <RatingBars stat={stat} />
          </div>
        )}
        {stat.type === 'nps' && (
          <p className="text-h3 font-semibold text-fg">
            {stat.average ? stat.average.toFixed(1) : '—'}
            <span className="text-caption text-fg-muted font-normal"> rata-rata (0 sampai 10)</span>
          </p>
        )}
        {stat.type === 'choice' && <ChoiceBars stat={stat} />}
        {stat.type === 'text' && (
          <div className="flex flex-col gap-2 max-h-64 overflow-auto">
            {(stat.texts ?? []).length === 0 ? (
              <p className="text-caption text-fg-subtle">Belum ada jawaban.</p>
            ) : (
              (stat.texts ?? []).map((t, i) => (
                <p key={i} className="text-body text-fg-muted border-l-2 border-line pl-3">"{t}"</p>
              ))
            )}
          </div>
        )}
      </CardBody>
    </Card>
  )
}

function InsightList({ title, items, icon, tone }: { title: string; items: string[]; icon: React.ReactNode; tone: string }) {
  if (items.length === 0) return null
  return (
    <div className="flex flex-col gap-1.5">
      <p className={cn('text-caption font-semibold uppercase tracking-wide flex items-center gap-1.5', tone)}>
        {icon} {title}
      </p>
      <ul className="flex flex-col gap-1">
        {items.map((it, i) => (
          <li key={i} className="text-body text-fg flex gap-2">
            <span className="text-fg-subtle">•</span> {it}
          </li>
        ))}
      </ul>
    </div>
  )
}

function answerSummary(form: FeedbackForm, sub: FeedbackSubmission): Record<string, string> {
  const byQ = new Map(sub.answers.map((a) => [a.question_id, a]))
  const out: Record<string, string> = {}
  for (const q of form.questions) {
    const a = byQ.get(q.id)
    if (!a) { out[q.id] = '—'; continue }
    if (q.type === 'text') out[q.id] = a.text || '—'
    else if (q.type === 'rating') out[q.id] = a.rating != null ? `${a.rating}/${q.scale ?? 5}` : '—'
    else if (q.type === 'nps') out[q.id] = a.rating != null ? String(a.rating) : '—'
    else out[q.id] = (a.choice ?? []).join(', ') || '—'
  }
  return out
}

// ── Halaman detail ────────────────────────────────────────────────────────────

export default function FeedbackFormDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: form, isLoading } = useFeedbackForm(id)
  const { data: submissions } = useFeedbackSubmissions(id)
  const { data: analytics } = useFeedbackFormAnalytics(id)
  const publishMutation = usePublishFeedbackForm()
  const analyzeMutation = useAnalyzeFeedback()

  const [tab, setTab] = useState('form')
  const [insight, setInsight] = useState<FeedbackInsight | null>(null)

  function copyLink() {
    if (!form) return
    void copyToClipboard(publicFormLink(form.slug)).then((ok) => {
      if (ok) toast.success('Link form disalin.')
      else toast.error('Gagal menyalin link.')
    })
  }

  async function handlePublish() {
    if (!form) return
    if (form.questions.length === 0) {
      toast.error('Tambahkan minimal satu pertanyaan sebelum menerbitkan.')
      return
    }
    try {
      const published = await publishMutation.mutateAsync(form.id)
      toast.success('Form diterbitkan.')
      await copyToClipboard(publicFormLink(published.slug))
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Gagal menerbitkan form.')
    }
  }

  async function handleExport() {
    if (!form || !submissions) return
    try {
      await exportFeedbackExcel(form, submissions)
    } catch {
      toast.error('Gagal mengekspor Excel.')
    }
  }

  async function runAnalyze() {
    if (!id) return
    try {
      setInsight(await analyzeMutation.mutateAsync(id))
    } catch {
      setInsight(null)
      toast.error('Gagal menganalisa feedback.')
    }
  }

  if (isLoading || !form) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64" />
      </div>
    )
  }

  const subs = submissions ?? []
  const summary: FormAnalytics | undefined = analytics

  const columns: Column<FeedbackSubmission>[] = [
    {
      key: 'respondent',
      header: 'Pengisi',
      render: (row) => (
        <div className="flex flex-col">
          <span className="font-medium text-fg">{row.respondent_email ?? '—'}</span>
          <span className="text-caption text-fg-muted">
            {[row.respondent_name, row.respondent_division].filter(Boolean).join(' · ') || '—'}
          </span>
        </div>
      ),
    },
    ...form.questions.map((q): Column<FeedbackSubmission> => ({
      key: q.id,
      header: q.label,
      render: (row) => {
        const s = answerSummary(form, row)
        return <span className="text-fg-muted whitespace-pre-wrap">{s[q.id]}</span>
      },
    })),
    {
      key: 'created_at',
      header: 'Tanggal',
      render: (row) => <span className="text-fg-muted">{formatTanggal(row.created_at)}</span>,
    },
  ]

  return (
    <div className="flex flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <div className="flex flex-col gap-1">
          <h1 className="text-h2 font-semibold text-fg">{form.title}</h1>
          <div className="flex items-center gap-2 flex-wrap">
            <Badge tone={STATUS_META[form.status].tone} appearance={STATUS_META[form.status].solid ? 'solid' : 'soft'}>
              {STATUS_META[form.status].label}
            </Badge>
            <span className="text-caption text-fg-muted">{form.language === 'en' ? 'English' : 'Indonesia'}</span>
            {form.status === 'published' ? (
              <a href={publicFormLink(form.slug)} target="_blank" rel="noreferrer" className="text-caption text-primary hover:underline">
                /form/{form.slug}
              </a>
            ) : (
              <span className="text-caption text-fg-subtle">/form/{form.slug}</span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" leftIcon={<Pencil className="w-4 h-4" />} onClick={() => navigate(`/postproject/feedback/${form.id}/edit`)}>
            Edit
          </Button>
          {form.status === 'published' ? (
            <Button variant="secondary" leftIcon={<LinkIcon className="w-4 h-4" />} onClick={copyLink}>
              Salin Link
            </Button>
          ) : form.status === 'processing_ai' || form.status === 'need_clarification' ? null : (
            <Button leftIcon={<Send className="w-4 h-4" />} loading={publishMutation.isPending} onClick={() => void handlePublish()}>
              Terbitkan
            </Button>
          )}
        </div>
      </div>

      <Tabs
        tabs={[
          { id: 'form', label: 'Pertanyaan' },
          { id: 'results', label: `Hasil & Analisa${subs.length ? ` (${subs.length})` : ''}` },
        ]}
        value={tab}
        onChange={setTab}
      />

      {tab === 'form' && (
        <TabPanel id="form" className="flex flex-col gap-3 max-w-2xl">
          {form.description && <p className="text-body text-fg-muted">{form.description}</p>}
          {form.questions.length === 0 ? (
            <EmptyState title="Belum ada pertanyaan" description="Klik Edit untuk menambahkan pertanyaan." />
          ) : (
            form.questions.map((q) => <QuestionPreview key={q.id} q={q} lang={form.language} />)
          )}
        </TabPanel>
      )}

      {tab === 'results' && (
        <TabPanel id="results" className="flex flex-col gap-6">
          {subs.length === 0 ? (
            <EmptyState
              icon={<Inbox className="w-6 h-6" />}
              title="Belum ada respon"
              description={form.status === 'published' ? 'Bagikan link form ke client; hasil akan muncul di sini.' : 'Terbitkan form ini lalu bagikan linknya ke client.'}
            />
          ) : (
            <>
              <div className="flex items-center justify-end">
                <Button variant="secondary" leftIcon={<FileSpreadsheet className="w-4 h-4" />} onClick={() => void handleExport()}>
                  Export Excel
                </Button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <StatCard label="Total Respon" value={subs.length} icon={<Inbox className="w-4 h-4" />} />
                {summary && summary.avg_rating > 0 && (
                  <StatCard label="Rata-rata Rating" value={summary.avg_rating.toFixed(1)} icon={<Hash className="w-4 h-4" />} hint="dari 5" />
                )}
                {summary && summary.questions.some((q) => q.type === 'nps') && (
                  <StatCard label="NPS" value={summary.nps} hint="-100 s/d 100" />
                )}
              </div>

              {/* Analisa AI */}
              <Card className="border-accent/40">
                <CardHeader className="flex items-center justify-between gap-2">
                  <h2 className="text-body font-semibold text-fg flex items-center gap-2">
                    <Sparkles className="w-4 h-4 text-accent" /> Analisa AI
                  </h2>
                  <Button
                    size="sm"
                    leftIcon={<Sparkles className="w-4 h-4" />}
                    loading={analyzeMutation.isPending}
                    onClick={() => void runAnalyze()}
                  >
                    {insight ? 'Analisa ulang' : 'Analisa dengan AI'}
                  </Button>
                </CardHeader>
                <CardBody className="flex flex-col gap-4">
                  {!insight && (
                    <p className="text-body text-fg-muted">
                      Klik "Analisa dengan AI" untuk ringkasan kekuatan, kekurangan, saran perbaikan, dan tema komentar dari feedback client.
                    </p>
                  )}
                  {insight?.degraded && <Badge tone="warning">AI sedang tidak tersedia — coba lagi nanti.</Badge>}
                  {insight && !insight.degraded && (
                    <>
                      {insight.summary && <p className="text-body text-fg">{insight.summary}</p>}
                      <div className="grid sm:grid-cols-2 gap-4">
                        <InsightList title="Kekuatan" items={insight.strengths} icon={<ThumbsUp className="w-3.5 h-3.5" />} tone="text-success" />
                        <InsightList title="Kekurangan" items={insight.weaknesses} icon={<TrendingDown className="w-3.5 h-3.5" />} tone="text-danger" />
                        <InsightList title="Saran Perbaikan" items={insight.improvements} icon={<Lightbulb className="w-3.5 h-3.5" />} tone="text-primary" />
                        <InsightList title="Tema Komentar" items={insight.themes} icon={<Tags className="w-3.5 h-3.5" />} tone="text-fg-muted" />
                      </div>
                    </>
                  )}
                </CardBody>
              </Card>

              {summary && (
                <div className="grid lg:grid-cols-2 gap-4 items-start">
                  {summary.questions.map((stat) => (
                    <QuestionStatCard key={stat.question_id} stat={stat} />
                  ))}
                </div>
              )}

              <Card>
                <CardHeader>
                  <h2 className="text-body font-semibold text-fg">Semua Jawaban</h2>
                </CardHeader>
                <div className="overflow-x-auto">
                  <Table columns={columns} data={subs} rowKey={(row) => row.id} pageSize={20} />
                </div>
              </Card>
            </>
          )}
        </TabPanel>
      )}
    </div>
  )
}
