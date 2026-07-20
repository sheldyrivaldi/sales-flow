import { useState } from 'react'
import { useParams } from 'react-router'
import { CheckCircle2 } from 'lucide-react'

import Button from '../../components/ui/Button'
import Textarea from '../../components/ui/Textarea'
import Input from '../../components/ui/Input'
import StarRating from '../../components/ui/StarRating'
import Skeleton from '../../components/ui/Skeleton'
import { LogoBadge, LogoWordmark } from '../../components/Logo'
import { cn } from '../../lib/cn'
import { usePublicFeedbackInfo, useSubmitPublicFeedback } from '../../api/feedback'

function AspectStars({
  label,
  value,
  onChange,
}: {
  label: string
  value: number
  onChange: (v: number) => void
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-body text-fg">{label}</span>
      <StarRating value={value} onChange={onChange} />
    </div>
  )
}

/** Halaman publik /f/:token — form feedback singkat yang diisi client tanpa
 * login. Sengaja ringkas (1 rating wajib + 3 aspek & komentar opsional)
 * supaya client tidak malas mengisi. */
export default function PublicFeedback() {
  const { token } = useParams<{ token: string }>()
  const { data: info, isLoading, isError } = usePublicFeedbackInfo(token)
  const submitMutation = useSubmitPublicFeedback(token)

  const [overall, setOverall] = useState(0)
  const [quality, setQuality] = useState(0)
  const [communication, setCommunication] = useState(0)
  const [timeliness, setTimeliness] = useState(0)
  const [nps, setNps] = useState<number | null>(null)
  const [comment, setComment] = useState('')
  const [name, setName] = useState('')
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit() {
    if (overall === 0) {
      setError('Mohon pilih rating keseluruhan dulu ya.')
      return
    }
    setError('')
    try {
      await submitMutation.mutateAsync({
        overall_rating: overall,
        quality_rating: quality || undefined,
        communication_rating: communication || undefined,
        timeliness_rating: timeliness || undefined,
        nps: nps ?? undefined,
        comment: comment.trim() || undefined,
        respondent_name: name.trim() || undefined,
      })
      setDone(true)
    } catch (err) {
      setError(err instanceof Error && err.message ? err.message : 'Gagal mengirim, coba lagi.')
    }
  }

  const shell = (children: React.ReactNode) => (
    <div className="min-h-screen bg-surface-muted flex items-start justify-center px-4 py-10">
      <div className="w-full max-w-lg flex flex-col gap-6">
        <div className="flex items-center justify-center gap-2.5">
          <LogoBadge size={36} />
          <LogoWordmark className="text-h3" />
        </div>
        <div className="bg-surface rounded-card shadow-lg border border-line p-6 sm:p-8">{children}</div>
        <p className="text-caption text-fg-subtle text-center">
          Feedback Anda membantu kami memberikan layanan yang lebih baik. Terima kasih.
        </p>
      </div>
    </div>
  )

  if (isLoading) {
    return shell(<Skeleton className="h-48" />)
  }

  if (isError || !info) {
    return shell(
      <div className="text-center py-6">
        <h1 className="text-h3 font-semibold text-fg mb-1">Link tidak ditemukan</h1>
        <p className="text-body text-fg-muted">
          Link feedback ini tidak valid atau sudah tidak berlaku. Silakan hubungi tim kami untuk link baru.
        </p>
      </div>,
    )
  }

  if (info.submitted || done) {
    return shell(
      <div className="text-center py-6 flex flex-col items-center gap-3">
        <CheckCircle2 className="w-12 h-12 text-success" aria-hidden="true" />
        <h1 className="text-h3 font-semibold text-fg">Terima kasih!</h1>
        <p className="text-body text-fg-muted">
          Feedback untuk proyek <span className="font-medium text-fg">{info.project_name}</span> sudah kami
          terima{done ? '' : ' sebelumnya'}. Senang bekerja sama dengan Anda.
        </p>
      </div>,
    )
  }

  return shell(
    <div className="flex flex-col gap-6">
      <div className="text-center">
        <h1 className="text-h3 font-semibold text-fg">Bagaimana pengalaman Anda?</h1>
        <p className="text-body text-fg-muted mt-1">
          Proyek <span className="font-medium text-fg">{info.project_name}</span>
          {info.client_name && ` — ${info.client_name}`}
        </p>
        <p className="text-caption text-fg-subtle mt-0.5">Hanya butuh ± 1 menit.</p>
      </div>

      {/* Rating keseluruhan (wajib) */}
      <div className="flex flex-col items-center gap-2">
        <p className="text-body font-medium text-fg">Kepuasan keseluruhan</p>
        <StarRating value={overall} onChange={setOverall} size="lg" />
      </div>

      {/* Aspek (opsional) */}
      <div className="flex flex-col gap-3 border-t border-line pt-4">
        <p className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Opsional</p>
        <AspectStars label="Kualitas hasil" value={quality} onChange={setQuality} />
        <AspectStars label="Komunikasi tim" value={communication} onChange={setCommunication} />
        <AspectStars label="Ketepatan waktu" value={timeliness} onChange={setTimeliness} />

        <div className="flex flex-col gap-1.5 mt-1">
          <p className="text-body text-fg">
            Seberapa besar kemungkinan Anda merekomendasikan kami? <span className="text-fg-subtle">(0-10)</span>
          </p>
          <div className="flex flex-wrap gap-1.5">
            {Array.from({ length: 11 }).map((_, n) => (
              <button
                key={n}
                type="button"
                aria-pressed={nps === n}
                onClick={() => setNps(nps === n ? null : n)}
                className={cn(
                  'w-8 h-8 rounded-btn border text-caption font-medium transition-colors',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
                  nps === n
                    ? 'bg-primary text-white border-primary'
                    : 'bg-surface text-fg-muted border-line hover:border-primary-border hover:text-fg',
                )}
              >
                {n}
              </button>
            ))}
          </div>
        </div>

        <Textarea
          rows={3}
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="Saran atau komentar (opsional)…"
        />
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Nama Anda (opsional)" />
      </div>

      {error && <p className="text-body text-danger text-center">{error}</p>}

      <Button size="lg" loading={submitMutation.isPending} onClick={() => void handleSubmit()}>
        Kirim Feedback
      </Button>
    </div>,
  )
}
