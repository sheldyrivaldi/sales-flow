import { useRef, useState, type CSSProperties } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router'
import {
  DndContext, DragOverlay, closestCenter, PointerSensor, useSensor, useSensors,
  type DragEndEvent, type DragStartEvent,
} from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy, useSortable, arrayMove } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  ArrowUp, ArrowDown, GripVertical, Trash2, Plus, Sparkles, Hash,
  Type as TypeIcon, ListChecks, Gauge, X, Wand2, Check, Paperclip,
  Loader2, AlertTriangle, Settings2, RotateCcw,
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
import { copyToClipboard } from '../../../lib/clipboard'
import {
  useFeedbackForm,
  useCreateFeedbackForm,
  useUpdateFeedbackForm,
  usePublishFeedbackForm,
  useSuggestQuestions,
  useSubmitClarifyAnswers,
  useClearAIJob,
  useRefineQuestion,
  publicFormLink,
} from '../../../api/feedbackForms'
import type {
  FeedbackForm, FeedbackQuestion, QuestionType, SuggestedQuestion, FormLanguage, TypeCountsInput,
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
  // Wajib diisi secara default — user cukup mematikan toggle untuk pertanyaan
  // yang memang opsional, alih-alih menyalakannya satu per satu.
  const base: FeedbackQuestion = { id: genId(), type, label: '', required: true }
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
    required: true,
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
  const [activeId, setActiveId] = useState<string | null>(null)
  const [savedId, setSavedId] = useState<string | undefined>(existing?.id ?? paramId)
  const dndSensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 8 } }))

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
  function handleDragStart(e: DragStartEvent) {
    setActiveId(String(e.active.id))
  }
  function handleDragEnd(e: DragEndEvent) {
    const { active, over } = e
    setActiveId(null)
    if (!over || active.id === over.id) return
    setQuestions((qs) => {
      const oldIndex = qs.findIndex((q) => q.id === active.id)
      const newIndex = qs.findIndex((q) => q.id === over.id)
      if (oldIndex === -1 || newIndex === -1) return qs
      return arrayMove(qs, oldIndex, newIndex)
    })
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
      await copyToClipboard(publicFormLink(published.slug))
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
            <Field label="Alamat link publik" helper={`Link: ${publicFormLink(slug || 'otomatis')}`}>
              <Input value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="Tulis alamat singkat untuk link ini" />
            </Field>
          </div>
        </CardBody>
      </Card>

      {/* Panel AI */}
      <AIPanel
        defaultOpen={searchParams.get('ai') === '1'}
        formId={savedId}
        language={language}
        onFormCreated={(id, derivedTitle) => {
          setSavedId(id)
          if (!title.trim()) setTitle(derivedTitle)
          window.history.replaceState(null, '', `/postproject/feedback/${id}/edit`)
        }}
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

        <DndContext
          sensors={dndSensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={() => setActiveId(null)}
        >
          <SortableContext items={questions.map((q) => q.id)} strategy={verticalListSortingStrategy}>
            <div className="flex flex-col gap-3">
              {questions.map((q, i) => (
                <QuestionCard
                  key={q.id}
                  q={q}
                  index={i}
                  total={questions.length}
                  language={language}
                  onMoveUp={() => move(i, -1)}
                  onMoveDown={() => move(i, 1)}
                  onChange={(patch) => updateQuestion(q.id, patch)}
                  onRemove={() => removeQuestion(q.id)}
                />
              ))}
            </div>
          </SortableContext>
          <DragOverlay>
            {activeId ? <QuestionCardOverlay q={questions.find((q) => q.id === activeId) ?? null} /> : null}
          </DragOverlay>
        </DndContext>

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
  onMoveUp: () => void
  onMoveDown: () => void
  onChange: (patch: Partial<FeedbackQuestion>) => void
  onRemove: () => void
}

function QuestionCard({
  q, index, total, language, onMoveUp, onMoveDown, onChange, onRemove,
}: QuestionCardProps) {
  const Icon = TYPE_META[q.type].icon
  const refine = useRefineQuestion()
  const [refineOpen, setRefineOpen] = useState(false)
  const [instruction, setInstruction] = useState('')

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: q.id })
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 10 : undefined,
  }

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
      ref={setNodeRef}
      style={style}
      className={cn(
        'rounded-card border bg-surface transition-shadow',
        isDragging ? 'border-primary/50 shadow-lg opacity-40' : 'border-line',
      )}
    >
      <div className="flex items-start gap-2 p-3">
        {/* Handle drag + reorder */}
        <div className="flex flex-col items-center pt-1 shrink-0">
          <span
            {...attributes}
            {...listeners}
            className="cursor-grab active:cursor-grabbing text-fg-subtle hover:text-fg touch-none"
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

// Kartu "melayang" yang mengikuti kursor selama drag (efek angkat ala Jira).
// Tampilan ringkas saja — interaksi form tidak perlu di sini.
function QuestionCardOverlay({ q }: { q: FeedbackQuestion | null }) {
  if (!q) return null
  const Icon = TYPE_META[q.type].icon
  return (
    <div className="flex items-center gap-2 px-3 py-3 rounded-card border border-primary bg-surface shadow-2xl scale-[1.02] cursor-grabbing">
      <GripVertical className="w-4 h-4 text-fg-subtle shrink-0" />
      <Icon className="w-4 h-4 text-fg-muted shrink-0" />
      <span className="text-body text-fg truncate">{q.label || 'Pertanyaan'}</span>
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

// ── Panel AI (saran pertanyaan, async + persisten) ────────────────────────────
//
// Generate bisa makan waktu lama (banyak lampiran/konteks), jadi alurnya tidak
// menahan koneksi: sekali diminta, form langsung tersimpan dengan status
// processing_ai/need_clarification, dan panel ini merekonstruksi tampilannya
// dari data form yang di-poll (bukan state lokal) — pindah halaman lalu
// kembali tidak menghilangkan progres.

const QUESTION_TYPE_ORDER: QuestionType[] = ['rating', 'text', 'choice', 'nps']

function AIPanel({
  defaultOpen, formId, language, onFormCreated, onAdd,
}: {
  defaultOpen: boolean
  formId: string | undefined
  language: FormLanguage
  onFormCreated: (id: string, derivedTitle: string) => void
  onAdd: (q: SuggestedQuestion[]) => void
}) {
  const [open, setOpen] = useState(defaultOpen)
  const [prompt, setPrompt] = useState('')
  const [files, setFiles] = useState<File[]>([])
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [clarifyAnswers, setClarifyAnswers] = useState<string[]>([])
  const [typeConfigOpen, setTypeConfigOpen] = useState(false)
  const [typeCounts, setTypeCounts] = useState<TypeCountsInput>({})
  const [localFormId, setLocalFormId] = useState<string | undefined>(formId)
  const fileRef = useRef<HTMLInputElement>(null)

  const { data: form } = useFeedbackForm(localFormId)
  const suggest = useSuggestQuestions()
  const submitClarify = useSubmitClarifyAnswers()
  const clearJob = useClearAIJob()

  const job = form?.ai_job ?? null
  const isProcessing = form?.status === 'processing_ai'
  const isClarifying = form?.status === 'need_clarification' && (job?.clarifying_questions.length ?? 0) > 0
  const pending = job?.pending_questions ?? []

  // Tiga penyesuaian "derived state" di bawah dilakukan LANGSUNG di badan
  // render (pola resmi React untuk "sesuaikan state saat prop/data berubah"),
  // bukan lewat useEffect — memanggil setState sinkron di dalam efek memicu
  // render tambahan yang tak perlu.

  // formId milik parent menang begitu tersedia (mis. user sudah Simpan Draft
  // sebelum membuka panel ini) — supaya saran AI menempel ke form yang sama,
  // bukan membuat form baru.
  const [prevFormIdProp, setPrevFormIdProp] = useState(formId)
  if (formId && formId !== prevFormIdProp) {
    setPrevFormIdProp(formId)
    setLocalFormId(formId)
  }

  // Sinkronkan panjang array jawaban dengan jumlah pertanyaan klarifikasi
  // terbaru (berubah tiap kali form ter-poll dengan job baru).
  const clarifyCount = isClarifying ? job!.clarifying_questions.length : 0
  const [prevClarifyCount, setPrevClarifyCount] = useState(-1)
  if (isClarifying && clarifyCount !== prevClarifyCount) {
    setPrevClarifyCount(clarifyCount)
    setClarifyAnswers((prev) => job!.clarifying_questions.map((_, i) => prev[i] ?? ''))
  }

  // Pilih semua opsi secara default begitu daftar pending muncul/berubah.
  const [prevPendingCount, setPrevPendingCount] = useState(0)
  if (pending.length !== prevPendingCount) {
    setPrevPendingCount(pending.length)
    setSelected(pending.length > 0 ? new Set(pending.map((_, i) => i)) : new Set())
  }

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

  function openTypeConfig() {
    setTypeConfigOpen(true)
    // Default tiap tipe: acak (bebas jumlahnya) — hanya diisi sekali saat
    // pertama dibuka; kalau user sudah pernah mengatur, jangan ditimpa.
    setTypeCounts((prev) => (Object.keys(prev).length === 0
      ? { rating: 'random', text: 'random', choice: 'random', nps: 'random' }
      : prev))
  }

  function resetTypeConfig() {
    setTypeConfigOpen(false)
    setTypeCounts({})
  }

  async function runSuggest() {
    try {
      const created = await suggest.mutateAsync({ formId: localFormId, prompt, files, language, typeCounts })
      setLocalFormId(created.id)
      if (!formId) onFormCreated(created.id, created.title)
      setFiles([])
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Gagal meminta saran AI.')
    }
  }

  async function submitAnswers() {
    if (!localFormId) return
    try {
      await submitClarify.mutateAsync({ formId: localFormId, answers: clarifyAnswers })
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Gagal mengirim jawaban klarifikasi.')
    }
  }

  async function cancelJob() {
    if (!localFormId) return
    try {
      await clearJob.mutateAsync(localFormId)
      setPrompt('')
      setFiles([])
    } catch {
      toast.error('Gagal membatalkan.')
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

  async function addSelected() {
    const chosen = pending.filter((_, i) => selected.has(i))
    if (chosen.length === 0 || !localFormId) return
    onAdd(chosen)
    setSelected(new Set())
    setPrompt('')
    setFiles([])
    toast.success(`${chosen.length} pertanyaan ditambahkan.`)
    try {
      await clearJob.mutateAsync(localFormId)
    } catch {
      // Best-effort: pertanyaan sudah masuk ke form lokal: sisa job sementara
      // yang gagal dibersihkan tidak fatal, cuma tampil lagi saat dibuka nanti.
    }
  }

  return (
    <Card className="border-accent/40">
      <CardBody className="flex flex-col gap-3">
        <button type="button" onClick={() => setOpen((o) => !o)} className="flex items-center gap-2 text-left">
          <Sparkles className="w-4 h-4 text-accent" />
          <span className="text-body font-semibold text-fg">Susun dengan bantuan AI</span>
          {isProcessing && <Badge tone="info">Memproses…</Badge>}
          {isClarifying && <Badge tone="warning">Perlu klarifikasi</Badge>}
          <span className="ml-auto text-caption text-fg-muted">{open ? 'Sembunyikan' : 'Buka'}</span>
        </button>

        {open && (
          <div className="flex flex-col gap-3">
            {job?.error && !isProcessing && !isClarifying && pending.length === 0 && (
              <div className="flex items-center gap-2 rounded-btn border border-danger/30 bg-danger/5 px-3 py-2 text-caption text-danger">
                <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
                <span>{job.error}</span>
              </div>
            )}

            {isProcessing ? (
              <div className="flex items-center gap-2 py-4 text-body text-fg-muted">
                <Loader2 className="w-4 h-4 animate-spin shrink-0" />
                <span>AI sedang menyusun pertanyaan. Boleh tinggalkan halaman ini — hasilnya tersimpan otomatis.</span>
              </div>
            ) : isClarifying ? (
              <div className="flex flex-col gap-3 border-t border-line pt-3">
                <p className="text-body text-fg font-medium">Bantu perjelas dulu, supaya saran AI lebih tepat sasaran</p>
                {job!.clarifying_questions.map((q, i) => (
                  <Field key={i} label={q}>
                    <Input
                      value={clarifyAnswers[i] ?? ''}
                      onChange={(e) => setClarifyAnswers((prev) => prev.map((a, idx) => (idx === i ? e.target.value : a)))}
                      placeholder="Tulis jawaban"
                      onKeyDown={(e) => e.key === 'Enter' && void submitAnswers()}
                    />
                  </Field>
                ))}
                <div className="flex items-center gap-3">
                  <Button size="sm" leftIcon={<Sparkles className="w-4 h-4" />} loading={submitClarify.isPending} onClick={() => void submitAnswers()}>
                    Lanjutkan
                  </Button>
                  <button type="button" onClick={() => void cancelJob()} className="text-caption text-fg-subtle hover:text-fg underline">
                    Batalkan
                  </button>
                </div>
              </div>
            ) : pending.length > 0 ? (
              <div className="flex flex-col gap-2 border-t border-line pt-3">
                <p className="text-caption text-fg-muted">Pilih pertanyaan yang mau dipakai:</p>
                {pending.map((s, i) => {
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
                <div className="flex items-center gap-3">
                  <Button size="sm" leftIcon={<Plus className="w-4 h-4" />} loading={clearJob.isPending} onClick={() => void addSelected()}>
                    Tambahkan yang dipilih ({selected.size})
                  </Button>
                  <button type="button" onClick={() => void cancelJob()} className="text-caption text-fg-subtle hover:text-fg underline">
                    Buang saran ini
                  </button>
                </div>
              </div>
            ) : (
              <>
                <Field label="Ceritakan kebutuhan kuesioner" helper="Jelaskan tujuan form dan hal yang ingin diukur dari client.">
                  <Textarea rows={3} value={prompt} onChange={(e) => setPrompt(e.target.value)} placeholder="Tulis kebutuhan kuesioner Anda" />
                </Field>

                {!typeConfigOpen ? (
                  <button
                    type="button"
                    onClick={openTypeConfig}
                    className="inline-flex items-center gap-1.5 self-start text-caption text-fg-muted hover:text-fg"
                  >
                    <Settings2 className="w-3.5 h-3.5" /> Atur tipe & jumlah pertanyaan
                  </button>
                ) : (
                  <div className="flex flex-col gap-2 rounded-btn border border-line p-3">
                    <div className="flex items-center justify-between">
                      <span className="text-caption text-fg-muted">
                        Atur tipe & jumlah pertanyaan (kosongkan/acak = AI bebas menentukan)
                      </span>
                      <button
                        type="button"
                        onClick={resetTypeConfig}
                        className="inline-flex items-center gap-1 text-caption text-fg-subtle hover:text-fg"
                      >
                        <RotateCcw className="w-3 h-3" /> Reset ke otomatis
                      </button>
                    </div>
                    {QUESTION_TYPE_ORDER.map((t) => (
                      <TypeCountRow
                        key={t}
                        label={TYPE_META[t].label}
                        value={typeCounts[t]}
                        onChange={(v) => setTypeCounts((prev) => ({ ...prev, [t]: v }))}
                      />
                    ))}
                  </div>
                )}

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
              </>
            )}
          </div>
        )}
      </CardBody>
    </Card>
  )
}

// Satu baris konfigurasi tipe & jumlah pertanyaan: tombol "Acak" atau angka
// pasti (0 = tidak disertakan sama sekali). Kosong (undefined) berarti tipe
// itu belum disebutkan sama sekali — dibiarkan sepenuhnya bebas untuk AI.
function TypeCountRow({
  label, value, onChange,
}: {
  label: string
  value: number | 'random' | undefined
  onChange: (v: number | 'random' | undefined) => void
}) {
  const isRandom = value === 'random'
  const isFixed = typeof value === 'number'
  return (
    <div className="flex items-center gap-2">
      <span className="text-caption text-fg w-32 shrink-0">{label}</span>
      <button
        type="button"
        onClick={() => onChange('random')}
        className={cn(
          'px-2.5 py-1 rounded-btn text-caption border transition-colors',
          isRandom ? 'border-primary bg-primary-subtle text-primary' : 'border-line text-fg-muted hover:bg-surface-subtle',
        )}
      >
        Acak
      </button>
      <Input
        type="number"
        min={0}
        max={10}
        value={isFixed ? value : ''}
        placeholder="jumlah"
        onChange={(e) => {
          const raw = e.target.value
          if (raw === '') { onChange(undefined); return }
          onChange(Math.max(0, Math.min(10, Number(raw) || 0)))
        }}
        className="w-20"
      />
      <span className="text-caption text-fg-subtle">
        {isFixed ? (value === 0 ? 'tidak disertakan' : `tepat ${value} pertanyaan`) : isRandom ? 'jumlah bebas' : 'bebas ditentukan AI'}
      </span>
    </div>
  )
}
