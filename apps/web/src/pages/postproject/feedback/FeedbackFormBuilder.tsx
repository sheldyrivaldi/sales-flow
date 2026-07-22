import { useRef, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router'
import {
  ArrowUp, ArrowDown, GripVertical, Trash2, Plus, Sparkles, Hash,
  Type as TypeIcon, ListChecks, Gauge, X, Wand2, Check, Paperclip,
} from 'lucide-react'

import Button from '../../../components/ui/Button'
import Field from '../../../components/ui/Field'
import Input from '../../../components/ui/Input'
import Textarea from '../../../components/ui/Textarea'
import Select from '../../../components/ui/Select'
import Toggle from '../../../components/ui/Toggle'
import Badge from '../../../components/ui/Badge'
import Card, { CardBody } from '../../../components/ui/Card'
import Skeleton from '../../../components/ui/Skeleton'
import { toast } from '../../../lib/toast'
import { cn } from '../../../lib/cn'
import {
  useFeedbackForm,
  useCreateFeedbackForm,
  useUpdateFeedbackForm,
  usePublishFeedbackForm,
  useSuggestQuestions,
  useRefineQuestion,
  publicFormLink,
} from '../../../api/feedbackForms'
import type {
  FeedbackForm, FeedbackQuestion, QuestionType, SuggestedQuestion, FormLanguage,
} from '../../../api/feedbackForms'

// Batas lampiran saran AI — selaras dengan cap backend (10 MB per berkas).
const MAX_FILE_MB = 10
const MAX_FILE_BYTES = MAX_FILE_MB * 1024 * 1024
const MAX_FILES = 5

let idSeq = 0
function genId(): string {
  idSeq += 1
  const rnd = typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID().slice(0, 8) : String(Math.random()).slice(2, 10)
  return `q${Date.now().toString(36)}${idSeq}${rnd}`
}

const TYPE_META: Record<QuestionType, { label: string; icon: typeof Hash }> = {
  rating: { label: 'Rating (angka)', icon: Hash },
  text: { label: 'Teks bebas', icon: TypeIcon },
  choice: { label: 'Pilihan ganda', icon: ListChecks },
  nps: { label: 'NPS (0 sampai 10)', icon: Gauge },
}

// Label ujung skala rating bawaan (dipakai bila user mengosongkannya), mengikuti
// bahasa form.
function ratingDefaults(lang: FormLanguage): { min: string; max: string } {
  return lang === 'en'
    ? { min: 'Very poor', max: 'Excellent' }
    : { min: 'Sangat kurang', max: 'Sangat baik' }
}

function blankQuestion(type: QuestionType): FeedbackQuestion {
  const base: FeedbackQuestion = { id: genId(), type, label: '', required: false }
  if (type === 'rating') base.scale = 5
  if (type === 'choice') { base.options = ['', '']; base.multiple = false }
  return base
}

function fromSuggested(s: SuggestedQuestion): FeedbackQuestion {
  return {
    id: genId(),
    type: s.type,
    label: s.label,
    description: s.description,
    required: false,
    scale: s.type === 'rating' ? s.scale ?? 5 : undefined,
    options: s.type === 'choice' ? s.options ?? [] : undefined,
    multiple: s.type === 'choice' ? s.multiple ?? false : undefined,
    min_label: s.type === 'rating' ? s.min_label : undefined,
    max_label: s.type === 'rating' ? s.max_label : undefined,
  }
}

// Wrapper: tunggu data (mode edit) lalu remount BuilderInner dengan key stabil
// agar state awal di-derive dari data tanpa efek sinkronisasi.
export default function FeedbackFormBuilder() {
  const { id } = useParams<{ id: string }>()
  const { data: existing, isLoading } = useFeedbackForm(id)

  if (id && isLoading) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64" />
      </div>
    )
  }
  return <BuilderInner key={id ?? 'new'} existing={existing ?? null} paramId={id} />
}

function BuilderInner({ existing, paramId }: { existing: FeedbackForm | null; paramId?: string }) {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const createMutation = useCreateFeedbackForm()
  const updateMutation = useUpdateFeedbackForm()
  const publishMutation = usePublishFeedbackForm()

  const [title, setTitle] = useState(existing?.title ?? '')
  const [description, setDescription] = useState(existing?.description ?? '')
  const [slug, setSlug] = useState(existing?.slug ?? '')
  const [language, setLanguage] = useState<FormLanguage>(existing?.language ?? 'id')
  const [questions, setQuestions] = useState<FeedbackQuestion[]>(existing?.questions ?? [])
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [savedId, setSavedId] = useState<string | undefined>(existing?.id ?? paramId)

  // --- Question mutators ---
  function updateQuestion(qid: string, patch: Partial<FeedbackQuestion>) {
    setQuestions((qs) => qs.map((q) => (q.id === qid ? { ...q, ...patch } : q)))
  }
  function removeQuestion(qid: string) {
    setQuestions((qs) => qs.filter((q) => q.id !== qid))
  }
  function addQuestion(type: QuestionType) {
    setQuestions((qs) => [...qs, blankQuestion(type)])
    setAddOpen(false)
  }
  function move(index: number, dir: -1 | 1) {
    setQuestions((qs) => {
      const target = index + dir
      if (target < 0 || target >= qs.length) return qs
      const next = [...qs]
      ;[next[index], next[target]] = [next[target], next[index]]
      return next
    })
  }
  function dropOnto(target: number) {
    setQuestions((qs) => {
      if (dragIndex === null || dragIndex === target) return qs
      const next = [...qs]
      const [moved] = next.splice(dragIndex, 1)
      next.splice(target, 0, moved)
      return next
    })
    setDragIndex(null)
  }

  const [addOpen, setAddOpen] = useState(false)

  // --- Save / publish ---
  function buildBody() {
    return {
      title: title.trim(),
      description: description.trim() || null,
      slug: slug.trim() || undefined,
      language,
      questions,
    }
  }

  async function persist(): Promise<string | null> {
    if (!title.trim()) {
      toast.error('Judul form wajib diisi.')
      return null
    }
    for (const q of questions) {
      if (!q.label.trim()) {
        toast.error('Setiap pertanyaan wajib punya label.')
        return null
      }
      if (q.type === 'choice' && (q.options ?? []).filter((o) => o.trim()).length === 0) {
        toast.error(`Pertanyaan pilihan "${q.label || '(tanpa label)'}" butuh minimal satu opsi.`)
        return null
      }
    }
    try {
      if (savedId) {
        const updated = await updateMutation.mutateAsync({ id: savedId, body: buildBody() })
        setSlug(updated.slug)
        return updated.id
      }
      const created = await createMutation.mutateAsync(buildBody())
      setSavedId(created.id)
      setSlug(created.slug)
      // Ganti URL ke mode edit tanpa reload agar Save berikutnya meng-update.
      window.history.replaceState(null, '', `/postproject/feedback/${created.id}/edit`)
      return created.id
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Gagal menyimpan form.')
      return null
    }
  }

  async function handleSaveDraft() {
    const savedFormId = await persist()
    if (savedFormId) toast.success('Draft tersimpan.')
  }

  async function handlePublish() {
    if (questions.length === 0) {
      toast.error('Tambahkan minimal satu pertanyaan sebelum menerbitkan.')
      return
    }
    const savedFormId = await persist()
    if (!savedFormId) return
    try {
      const published = await publishMutation.mutateAsync(savedFormId)
      toast.success('Form diterbitkan.')
      await navigator.clipboard.writeText(publicFormLink(published.slug)).catch(() => {})
      navigate(`/postproject/feedback/${savedFormId}`)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Gagal menerbitkan form.')
    }
  }

  const busy = createMutation.isPending || updateMutation.isPending || publishMutation.isPending

  return (
    <div className="flex flex-col gap-6 p-6 max-w-3xl mx-auto w-full">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-h3 font-semibold text-fg">{savedId ? 'Edit Form' : 'Buat Form'}</h1>
        <div className="flex items-center gap-2">
          <Button variant="secondary" onClick={() => void handleSaveDraft()} loading={busy}>
            Simpan Draft
          </Button>
          <Button onClick={() => void handlePublish()} loading={busy}>
            Terbitkan
          </Button>
        </div>
      </div>

      {/* Meta form */}
      <Card>
        <CardBody className="flex flex-col gap-4">
          <Field label="Judul form" required>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Tulis judul form" />
          </Field>
          <Field label="Deskripsi" helper="Muncul di atas form yang dilihat client (opsional)">
            <Textarea rows={2} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Tulis keterangan singkat form" />
          </Field>
          <div className="grid sm:grid-cols-2 gap-4">
            <Field label="Bahasa form" required helper="Menentukan bahasa saran AI dan label bawaan.">
              <Select value={language} onChange={(e) => setLanguage(e.target.value as FormLanguage)}>
                <option value="id">Bahasa Indonesia</option>
                <option value="en">English</option>
              </Select>
            </Field>
            <Field label="Slug link publik" helper={`Link: ${publicFormLink(slug || 'otomatis')}`}>
              <Input value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="Tulis slug link publik" />
            </Field>
          </div>
        </CardBody>
      </Card>

      {/* Panel AI */}
      <AIPanel
        defaultOpen={searchParams.get('ai') === '1'}
        language={language}
        onAdd={(selected) => setQuestions((qs) => [...qs, ...selected.map(fromSuggested)])}
      />

      {/* Daftar pertanyaan */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-body font-semibold text-fg">Pertanyaan ({questions.length})</h2>
        </div>

        {questions.length === 0 && (
          <p className="text-caption text-fg-muted border border-dashed border-line rounded-card p-6 text-center">
            Belum ada pertanyaan. Tambah manual di bawah atau minta saran AI di atas.
          </p>
        )}

        {questions.map((q, i) => (
          <QuestionCard
            key={q.id}
            q={q}
            index={i}
            total={questions.length}
            language={language}
            isDragging={dragIndex === i}
            onDragStart={() => setDragIndex(i)}
            onDragEnd={() => setDragIndex(null)}
            onDropOnto={() => dropOnto(i)}
            onMoveUp={() => move(i, -1)}
            onMoveDown={() => move(i, 1)}
            onChange={(patch) => updateQuestion(q.id, patch)}
            onRemove={() => removeQuestion(q.id)}
          />
        ))}

        {/* Tambah pertanyaan */}
        <div className="relative">
          <Button variant="secondary" leftIcon={<Plus className="w-4 h-4" />} onClick={() => setAddOpen((o) => !o)}>
            Tambah Pertanyaan
          </Button>
          {addOpen && (
            <div className="absolute z-20 mt-1 w-56 bg-surface border border-line rounded-btn shadow-lg py-1">
              {(Object.keys(TYPE_META) as QuestionType[]).map((t) => {
                const Icon = TYPE_META[t].icon
                return (
                  <button
                    key={t}
                    type="button"
                    onClick={() => addQuestion(t)}
                    className="w-full flex items-center gap-2 px-3 py-2 text-body text-fg hover:bg-surface-subtle transition-colors"
                  >
                    <Icon className="w-4 h-4 text-fg-muted" /> {TYPE_META[t].label}
                  </button>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// ── Kartu pertanyaan (editor) ─────────────────────────────────────────────────

interface QuestionCardProps {
  q: FeedbackQuestion
  index: number
  total: number
  language: FormLanguage
  isDragging: boolean
  onDragStart: () => void
  onDragEnd: () => void
  onDropOnto: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  onChange: (patch: Partial<FeedbackQuestion>) => void
  onRemove: () => void
}

function QuestionCard({
  q, index, total, language, isDragging, onDragStart, onDragEnd, onDropOnto, onMoveUp, onMoveDown, onChange, onRemove,
}: QuestionCardProps) {
  const Icon = TYPE_META[q.type].icon
  const refine = useRefineQuestion()
  const [refineOpen, setRefineOpen] = useState(false)
  const [instruction, setInstruction] = useState('')

  async function runRefine() {
    if (!instruction.trim()) return
    try {
      const res = await refine.mutateAsync({
        question: {
          type: q.type, label: q.label, description: q.description,
          scale: q.scale, options: q.options, multiple: q.multiple,
          min_label: q.min_label, max_label: q.max_label,
        },
        instruction: instruction.trim(),
        language,
      })
      if (res.degraded) {
        toast.error('AI sedang tidak tersedia. Coba lagi nanti.')
        return
      }
      const s = res.question
      onChange({
        type: s.type,
        label: s.label,
        description: s.description,
        scale: s.type === 'rating' ? s.scale ?? 5 : undefined,
        options: s.type === 'choice' ? s.options ?? [] : undefined,
        multiple: s.type === 'choice' ? s.multiple ?? false : undefined,
        min_label: s.type === 'rating' ? s.min_label : undefined,
        max_label: s.type === 'rating' ? s.max_label : undefined,
      })
      toast.success('Pertanyaan diperbarui oleh AI.')
      setRefineOpen(false)
      setInstruction('')
    } catch {
      toast.error('Gagal meminta AI merevisi pertanyaan.')
    }
  }

  return (
    <div
      onDragOver={(e) => e.preventDefault()}
      onDrop={onDropOnto}
      className={cn(
        'rounded-card border bg-surface transition-colors',
        isDragging ? 'border-primary ring-2 ring-primary/25 opacity-60' : 'border-line',
      )}
    >
      <div className="flex items-start gap-2 p-3">
        {/* Handle drag + reorder */}
        <div className="flex flex-col items-center pt-1 shrink-0">
          <span
            draggable
            onDragStart={onDragStart}
            onDragEnd={onDragEnd}
            className="cursor-grab active:cursor-grabbing text-fg-subtle hover:text-fg"
            aria-label="Seret untuk memindah"
            title="Seret untuk memindah"
          >
            <GripVertical className="w-4 h-4" />
          </span>
          <button
            type="button"
            aria-label="Naikkan"
            disabled={index === 0}
            onClick={onMoveUp}
            className="text-fg-subtle hover:text-fg disabled:opacity-30"
          >
            <ArrowUp className="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            aria-label="Turunkan"
            disabled={index === total - 1}
            onClick={onMoveDown}
            className="text-fg-subtle hover:text-fg disabled:opacity-30"
          >
            <ArrowDown className="w-3.5 h-3.5" />
          </button>
        </div>

        <div className="flex-1 flex flex-col gap-2 min-w-0">
          <div className="flex items-center gap-2">
            <Badge tone="info" className="gap-1 shrink-0">
              <Icon className="w-3 h-3" /> {TYPE_META[q.type].label}
            </Badge>
            <div className="ml-auto flex items-center gap-1">
              <button
                type="button"
                onClick={() => setRefineOpen((o) => !o)}
                className="inline-flex items-center gap-1 px-2 py-1 rounded-btn text-caption text-accent hover:bg-accent/10 transition-colors"
              >
                <Wand2 className="w-3.5 h-3.5" /> Edit dengan AI
              </button>
              <button
                type="button"
                aria-label="Hapus pertanyaan"
                onClick={onRemove}
                className="p-1.5 rounded-btn text-fg-subtle hover:text-danger hover:bg-surface-subtle transition-colors"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          </div>

          <Input
            value={q.label}
            onChange={(e) => onChange({ label: e.target.value })}
            placeholder="Tulis pertanyaan"
          />

          {/* Config per tipe */}
          {q.type === 'rating' && (
            <RatingConfig q={q} language={language} onChange={onChange} />
          )}

          {q.type === 'choice' && (
            <ChoiceEditor
              options={q.options ?? []}
              multiple={q.multiple ?? false}
              onOptions={(options) => onChange({ options })}
              onMultiple={(multiple) => onChange({ multiple })}
            />
          )}

          {q.type === 'nps' && (
            <p className="text-caption text-fg-subtle">Skor 0 sampai 10 (kemungkinan merekomendasikan).</p>
          )}

          {refineOpen && (
            <div className="flex items-center gap-2 border-t border-line pt-2">
              <Input
                value={instruction}
                onChange={(e) => setInstruction(e.target.value)}
                placeholder="Tulis perubahan yang diinginkan"
                onKeyDown={(e) => e.key === 'Enter' && void runRefine()}
              />
              <Button size="sm" onClick={() => void runRefine()} loading={refine.isPending} leftIcon={<Sparkles className="w-3.5 h-3.5" />}>
                Revisi
              </Button>
            </div>
          )}

          <Toggle
            size="sm"
            checked={q.required}
            onChange={(v) => onChange({ required: v })}
            label="Wajib diisi"
          />
        </div>
      </div>
    </div>
  )
}

// RatingConfig: skala maksimum + label ujung kiri/kanan + pratinjau angka rata.
function RatingConfig({
  q, language, onChange,
}: {
  q: FeedbackQuestion
  language: FormLanguage
  onChange: (patch: Partial<FeedbackQuestion>) => void
}) {
  const scale = q.scale ?? 5
  const def = ratingDefaults(language)
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 text-caption text-fg-muted">
        <span>Skala maksimum</span>
        <Input
          type="number"
          min={2}
          max={10}
          value={scale}
          onChange={(e) => onChange({ scale: Math.max(2, Math.min(10, Number(e.target.value) || 5)) })}
          className="w-20"
        />
        <span>angka</span>
      </div>

      {/* Pratinjau angka tersebar rata dengan keterangan ujung kiri & kanan. */}
      <div className="flex items-center gap-3">
        <span className="text-caption text-fg-subtle shrink-0 w-28 truncate text-right" title={q.min_label || def.min}>
          {q.min_label || def.min}
        </span>
        <div className="flex-1 flex justify-between">
          {Array.from({ length: scale }).map((_, i) => (
            <span
              key={i}
              className="w-8 h-8 rounded-btn border border-line flex items-center justify-center text-caption tabular-nums text-fg-muted"
            >
              {i + 1}
            </span>
          ))}
        </div>
        <span className="text-caption text-fg-subtle shrink-0 w-28 truncate" title={q.max_label || def.max}>
          {q.max_label || def.max}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <Input
          value={q.min_label ?? ''}
          onChange={(e) => onChange({ min_label: e.target.value })}
          placeholder={`Keterangan kiri (${def.min})`}
        />
        <Input
          value={q.max_label ?? ''}
          onChange={(e) => onChange({ max_label: e.target.value })}
          placeholder={`Keterangan kanan (${def.max})`}
        />
      </div>
    </div>
  )
}

function ChoiceEditor({
  options, multiple, onOptions, onMultiple,
}: {
  options: string[]
  multiple: boolean
  onOptions: (o: string[]) => void
  onMultiple: (m: boolean) => void
}) {
  return (
    <div className="flex flex-col gap-1.5">
      {options.map((opt, i) => (
        <div key={i} className="flex items-center gap-2">
          <span className="text-fg-subtle text-caption w-5 text-right">{i + 1}.</span>
          <Input
            value={opt}
            onChange={(e) => onOptions(options.map((o, idx) => (idx === i ? e.target.value : o)))}
            placeholder="Tulis opsi jawaban"
          />
          <button
            type="button"
            aria-label="Hapus opsi"
            onClick={() => onOptions(options.filter((_, idx) => idx !== i))}
            className="p-1 rounded-btn text-fg-subtle hover:text-danger"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      ))}
      <div className="flex items-center justify-between mt-1">
        <button
          type="button"
          onClick={() => onOptions([...options, ''])}
          className="inline-flex items-center gap-1 text-caption text-primary hover:underline"
        >
          <Plus className="w-3.5 h-3.5" /> Tambah opsi
        </button>
        <Toggle size="sm" checked={multiple} onChange={onMultiple} label="Boleh pilih banyak" />
      </div>
    </div>
  )
}

// ── Panel AI (saran pertanyaan) ───────────────────────────────────────────────

function AIPanel({
  defaultOpen, language, onAdd,
}: {
  defaultOpen: boolean
  language: FormLanguage
  onAdd: (q: SuggestedQuestion[]) => void
}) {
  const [open, setOpen] = useState(defaultOpen)
  const [prompt, setPrompt] = useState('')
  const [files, setFiles] = useState<File[]>([])
  const [suggestions, setSuggestions] = useState<SuggestedQuestion[]>([])
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [degraded, setDegraded] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)
  const suggest = useSuggestQuestions()

  function addFiles(list: FileList | null) {
    if (!list || list.length === 0) return
    const incoming = Array.from(list)
    const tooBig = incoming.filter((f) => f.size > MAX_FILE_BYTES)
    if (tooBig.length > 0) {
      toast.error(`Ukuran lampiran maksimal ${MAX_FILE_MB} MB. Lewati: ${tooBig.map((f) => f.name).join(', ')}`)
    }
    const ok = incoming.filter((f) => f.size <= MAX_FILE_BYTES)
    setFiles((prev) => {
      const merged = [...prev]
      for (const f of ok) {
        if (merged.length >= MAX_FILES) break
        if (!merged.some((m) => m.name === f.name && m.size === f.size)) merged.push(f)
      }
      return merged
    })
    if (ok.length > 0) toast.success(`${ok.length} lampiran ditambahkan.`)
    if (fileRef.current) fileRef.current.value = ''
  }

  async function runSuggest() {
    try {
      const res = await suggest.mutateAsync({ prompt, files, language })
      setSuggestions(res.questions)
      setSelected(new Set(res.questions.map((_, i) => i))) // pilih semua default
      setDegraded(res.degraded)
      if (res.degraded) toast.error('AI sedang tidak tersedia. Susun pertanyaan manual dulu ya.')
      else if (res.questions.length === 0) toast.info('AI tidak menghasilkan saran. Coba perjelas kebutuhan.')
    } catch {
      toast.error('Gagal meminta saran AI.')
    }
  }

  function toggle(i: number) {
    setSelected((s) => {
      const next = new Set(s)
      if (next.has(i)) next.delete(i)
      else next.add(i)
      return next
    })
  }

  function addSelected() {
    const chosen = suggestions.filter((_, i) => selected.has(i))
    if (chosen.length === 0) return
    onAdd(chosen)
    setSuggestions([])
    setSelected(new Set())
    setPrompt('')
    setFiles([])
    toast.success(`${chosen.length} pertanyaan ditambahkan.`)
  }

  return (
    <Card className="border-accent/40">
      <CardBody className="flex flex-col gap-3">
        <button type="button" onClick={() => setOpen((o) => !o)} className="flex items-center gap-2 text-left">
          <Sparkles className="w-4 h-4 text-accent" />
          <span className="text-body font-semibold text-fg">Susun dengan bantuan AI</span>
          {degraded && <Badge tone="warning">AI tidak tersedia</Badge>}
          <span className="ml-auto text-caption text-fg-muted">{open ? 'Sembunyikan' : 'Buka'}</span>
        </button>

        {open && (
          <div className="flex flex-col gap-3">
            <Field label="Ceritakan kebutuhan kuesioner" helper="Jelaskan tujuan form dan hal yang ingin diukur dari client.">
              <Textarea rows={3} value={prompt} onChange={(e) => setPrompt(e.target.value)} placeholder="Tulis kebutuhan kuesioner Anda" />
            </Field>

            <div className="flex items-center gap-2 flex-wrap">
              <input
                ref={fileRef}
                type="file"
                accept="application/pdf,image/*,.pdf,.png,.jpg,.jpeg,.webp"
                multiple
                className="hidden"
                onChange={(e) => addFiles(e.target.files)}
              />
              <Button
                type="button"
                variant="secondary"
                size="sm"
                leftIcon={<Paperclip className="w-4 h-4" />}
                onClick={() => fileRef.current?.click()}
              >
                {files.length > 0 ? `Tambah lampiran (${files.length}/${MAX_FILES})` : 'Lampirkan dokumen'}
              </Button>
              <span className="text-caption text-fg-subtle">PDF atau gambar, maks {MAX_FILE_MB} MB/berkas</span>
              <Button
                className="ml-auto"
                size="sm"
                leftIcon={<Sparkles className="w-4 h-4" />}
                loading={suggest.isPending}
                onClick={() => void runSuggest()}
              >
                Minta saran AI
              </Button>
            </div>

            {files.length > 0 && (
              <div className="flex flex-col gap-1">
                {files.map((f, i) => (
                  <div key={`${f.name}-${i}`} className="flex items-center gap-2 text-caption text-fg-muted">
                    <Paperclip className="w-3 h-3 shrink-0" aria-hidden="true" />
                    <span className="truncate">{f.name}</span>
                    <span className="text-fg-subtle shrink-0">{(f.size / (1024 * 1024)).toFixed(1)} MB</span>
                    <button
                      type="button"
                      onClick={() => setFiles((prev) => prev.filter((_, idx) => idx !== i))}
                      className="text-fg-subtle hover:text-danger shrink-0"
                      aria-label="Hapus lampiran"
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            {suggestions.length > 0 && (
              <div className="flex flex-col gap-2 border-t border-line pt-3">
                <p className="text-caption text-fg-muted">Pilih pertanyaan yang mau dipakai:</p>
                {suggestions.map((s, i) => {
                  const Icon = TYPE_META[s.type].icon
                  const on = selected.has(i)
                  return (
                    <button
                      key={i}
                      type="button"
                      onClick={() => toggle(i)}
                      className={cn(
                        'flex items-start gap-2 p-2 rounded-btn border text-left transition-colors',
                        on ? 'border-primary bg-primary-subtle' : 'border-line hover:bg-surface-subtle',
                      )}
                    >
                      <span className={cn('mt-0.5 w-4 h-4 rounded border flex items-center justify-center shrink-0', on ? 'bg-primary border-primary text-white' : 'border-line')}>
                        {on && <Check className="w-3 h-3" />}
                      </span>
                      <span className="flex-1 min-w-0">
                        <span className="flex items-center gap-1.5">
                          <Icon className="w-3.5 h-3.5 text-fg-muted shrink-0" />
                          <span className="text-body text-fg">{s.label}</span>
                        </span>
                        {s.type === 'choice' && s.options && (
                          <span className="text-caption text-fg-subtle">Opsi: {s.options.join(', ')}</span>
                        )}
                      </span>
                    </button>
                  )
                })}
                <Button size="sm" className="self-start" leftIcon={<Plus className="w-4 h-4" />} onClick={addSelected}>
                  Tambahkan yang dipilih ({selected.size})
                </Button>
              </div>
            )}
          </div>
        )}
      </CardBody>
    </Card>
  )
}
