import { useState } from 'react'
import { useParams } from 'react-router'
import { CheckCircle2 } from 'lucide-react'

import Button from '../../components/ui/Button'
import Textarea from '../../components/ui/Textarea'
import Input from '../../components/ui/Input'
import Field from '../../components/ui/Field'
import Skeleton from '../../components/ui/Skeleton'
import { LogoBadge, LogoWordmark } from '../../components/Logo'
import { cn } from '../../lib/cn'
import { usePublicForm, useSubmitPublicForm } from '../../api/feedbackForms'
import type { FeedbackAnswer, FeedbackQuestion, FormLanguage } from '../../api/feedbackForms'

// State jawaban per pertanyaan (union longgar; dipetakan ke FeedbackAnswer saat submit).
type AnswerState = Record<string, { rating?: number; text?: string; choice?: string[] }>

// Teks UI per bahasa form (halaman publik mengikuti bahasa form).
const T = {
  id: {
    email: 'Email', name: 'Nama', division: 'Divisi', optional: 'opsional',
    text: 'Tulis jawaban Anda', submit: 'Kirim Feedback',
    fillEmail: 'Mohon isi email Anda.', fillName: 'Mohon isi nama Anda.',
    complete: (l: string) => `Mohon lengkapi: "${l}".`,
    thanks: 'Terima kasih!', received: 'sudah kami terima.',
    ratingMin: 'Sangat kurang', ratingMax: 'Sangat baik',
    footer: 'Feedback Anda membantu kami memberikan layanan yang lebih baik. Terima kasih.',
    notFound: 'Form tidak ditemukan', notFoundBody: 'Link form ini tidak valid atau belum aktif. Silakan hubungi tim kami untuk link baru.',
  },
  en: {
    email: 'Email', name: 'Name', division: 'Division', optional: 'optional',
    text: 'Type your answer', submit: 'Submit Feedback',
    fillEmail: 'Please enter your email.', fillName: 'Please enter your name.',
    complete: (l: string) => `Please complete: "${l}".`,
    thanks: 'Thank you!', received: 'has been received.',
    ratingMin: 'Very poor', ratingMax: 'Excellent',
    footer: 'Your feedback helps us deliver a better service. Thank you.',
    notFound: 'Form not found', notFoundBody: 'This form link is invalid or not active yet. Please contact us for a new link.',
  },
} satisfies Record<FormLanguage, Record<string, unknown>>

// NumberScaleInput: angka 1..scale tersebar rata dengan keterangan ujung kiri &
// kanan (bukan bintang).
function NumberScaleInput({
  scale, value, onChange, minLabel, maxLabel,
}: {
  scale: number
  value: number
  onChange: (v: number) => void
  minLabel: string
  maxLabel: string
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex justify-between gap-1.5" role="radiogroup">
        {Array.from({ length: scale }).map((_, i) => {
          const n = i + 1
          const on = n === value
          return (
            <button
              key={n}
              type="button"
              role="radio"
              aria-checked={on}
              aria-label={`${n}`}
              onClick={() => onChange(n === value ? 0 : n)}
              className={cn(
                'flex-1 h-10 rounded-btn border text-body font-medium tabular-nums transition-colors',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
                on
                  ? 'bg-primary text-white border-primary'
                  : 'bg-surface text-fg-muted border-line hover:border-primary-border hover:text-fg',
              )}
            >
              {n}
            </button>
          )
        })}
      </div>
      <div className="flex justify-between text-caption text-fg-subtle">
        <span>{minLabel}</span>
        <span>{maxLabel}</span>
      </div>
    </div>
  )
}

function QuestionField({
  q, value, onChange, t,
}: {
  q: FeedbackQuestion
  value: { rating?: number; text?: string; choice?: string[] }
  onChange: (v: { rating?: number; text?: string; choice?: string[] }) => void
  t: (typeof T)[FormLanguage]
}) {
  return (
    <div className="flex flex-col gap-2">
      <p className="text-body font-medium text-fg">
        {q.label}
        {q.required && <span className="text-danger ml-0.5">*</span>}
      </p>
      {q.description && <p className="text-caption text-fg-muted -mt-1">{q.description}</p>}

      {q.type === 'rating' && (
        <NumberScaleInput
          scale={q.scale ?? 5}
          value={value.rating ?? 0}
          onChange={(v) => onChange({ rating: v })}
          minLabel={q.min_label || t.ratingMin}
          maxLabel={q.max_label || t.ratingMax}
        />
      )}

      {q.type === 'nps' && (
        <div className="flex flex-wrap gap-1.5">
          {Array.from({ length: 11 }).map((_, n) => (
            <button
              key={n}
              type="button"
              aria-pressed={value.rating === n}
              onClick={() => onChange({ rating: value.rating === n ? undefined : n })}
              className={cn(
                'w-8 h-8 rounded-btn border text-caption font-medium transition-colors',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
                value.rating === n
                  ? 'bg-primary text-white border-primary'
                  : 'bg-surface text-fg-muted border-line hover:border-primary-border hover:text-fg',
              )}
            >
              {n}
            </button>
          ))}
        </div>
      )}

      {q.type === 'text' && (
        <Textarea rows={3} value={value.text ?? ''} onChange={(e) => onChange({ text: e.target.value })} placeholder={t.text} />
      )}

      {q.type === 'choice' && (
        <div className="flex flex-col gap-1.5">
          {(q.options ?? []).map((opt) => {
            const current = value.choice ?? []
            const on = current.includes(opt)
            function toggle() {
              if (q.multiple) {
                onChange({ choice: on ? current.filter((c) => c !== opt) : [...current, opt] })
              } else {
                onChange({ choice: on ? [] : [opt] })
              }
            }
            return (
              <button
                key={opt}
                type="button"
                onClick={toggle}
                className={cn(
                  'flex items-center gap-2 p-2.5 rounded-btn border text-left transition-colors',
                  on ? 'border-primary bg-primary-subtle' : 'border-line hover:bg-surface-subtle',
                )}
              >
                <span className={cn(
                  'w-4 h-4 flex items-center justify-center border shrink-0',
                  q.multiple ? 'rounded' : 'rounded-full',
                  on ? 'bg-primary border-primary' : 'border-line-strong',
                )}>
                  {on && <span className={cn('bg-white', q.multiple ? 'w-2 h-2 rounded-[1px]' : 'w-1.5 h-1.5 rounded-full')} />}
                </span>
                <span className="text-body text-fg">{opt}</span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

/** Halaman publik /form/:slug — form feedback dinamis yang diisi client tanpa
 * login. Identitas (email & nama wajib, divisi opsional) di atas seperti
 * Google Form, lalu pertanyaan sesuai definisi builder. */
export default function PublicFeedbackForm() {
  const { slug } = useParams<{ slug: string }>()
  const { data: form, isLoading, isError } = usePublicForm(slug)
  const submitMutation = useSubmitPublicForm(slug)

  const [answers, setAnswers] = useState<AnswerState>({})
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [division, setDivision] = useState('')
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')

  const t = T[form?.language ?? 'id']

  function setAnswer(qid: string, v: { rating?: number; text?: string; choice?: string[] }) {
    setAnswers((a) => ({ ...a, [qid]: v }))
  }

  function isAnswered(q: FeedbackQuestion): boolean {
    const v = answers[q.id]
    if (!v) return false
    if (q.type === 'text') return !!v.text?.trim()
    if (q.type === 'rating' || q.type === 'nps') return v.rating != null
    if (q.type === 'choice') return (v.choice ?? []).length > 0
    return false
  }

  async function handleSubmit() {
    if (!form) return
    if (!email.trim()) {
      setError(t.fillEmail)
      return
    }
    if (!name.trim()) {
      setError(t.fillName)
      return
    }
    for (const q of form.questions) {
      if (q.required && !isAnswered(q)) {
        setError(t.complete(q.label))
        return
      }
    }
    setError('')
    const payload: FeedbackAnswer[] = form.questions
      .filter((q) => isAnswered(q))
      .map((q) => {
        const v = answers[q.id]
        if (q.type === 'text') return { question_id: q.id, text: v.text?.trim() }
        if (q.type === 'rating' || q.type === 'nps') return { question_id: q.id, rating: v.rating }
        return { question_id: q.id, choice: v.choice }
      })
    try {
      await submitMutation.mutateAsync({
        respondent_email: email.trim(),
        respondent_name: name.trim(),
        respondent_division: division.trim() || undefined,
        answers: payload,
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
        <p className="text-caption text-fg-subtle text-center">{t.footer}</p>
      </div>
    </div>
  )

  if (isLoading) return shell(<Skeleton className="h-48" />)

  if (isError || !form) {
    return shell(
      <div className="text-center py-6">
        <h1 className="text-h3 font-semibold text-fg mb-1">{t.notFound}</h1>
        <p className="text-body text-fg-muted">{t.notFoundBody}</p>
      </div>,
    )
  }

  if (done) {
    return shell(
      <div className="text-center py-6 flex flex-col items-center gap-3">
        <CheckCircle2 className="w-12 h-12 text-success" aria-hidden="true" />
        <h1 className="text-h3 font-semibold text-fg">{t.thanks}</h1>
        <p className="text-body text-fg-muted">
          <span className="font-medium text-fg">{form.title}</span> {t.received}
        </p>
      </div>,
    )
  }

  return shell(
    <div className="flex flex-col gap-6">
      <div className="text-center">
        <h1 className="text-h3 font-semibold text-fg">{form.title}</h1>
        {form.description && <p className="text-body text-fg-muted mt-1">{form.description}</p>}
      </div>

      {/* Identitas pengisi di atas (seperti Google Form). */}
      <div className="flex flex-col gap-3 border border-line rounded-card p-4 bg-surface-subtle">
        <Field label={t.email} required>
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder={t.email} />
        </Field>
        <Field label={t.name} required>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={t.name} />
        </Field>
        <Field label={`${t.division} (${t.optional})`}>
          <Input value={division} onChange={(e) => setDivision(e.target.value)} placeholder={t.division} />
        </Field>
      </div>

      <div className="flex flex-col gap-5">
        {form.questions.map((q) => (
          <QuestionField key={q.id} q={q} value={answers[q.id] ?? {}} onChange={(v) => setAnswer(q.id, v)} t={t} />
        ))}
      </div>

      {error && <p className="text-body text-danger text-center">{error}</p>}

      <Button size="lg" loading={submitMutation.isPending} onClick={() => void handleSubmit()}>
        {t.submit}
      </Button>
    </div>,
  )
}
