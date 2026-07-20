import { useState } from 'react'
import { useNavigate } from 'react-router'
import { UploadCloud, PenLine, Sparkles } from 'lucide-react'

import Card, { CardBody } from '../../components/ui/Card'
import Stepper from '../../components/ui/Stepper'
import type { StepItem } from '../../components/ui/Stepper'
import Button from '../../components/ui/Button'
import Field from '../../components/ui/Field'
import Input from '../../components/ui/Input'
import ChipInput from '../../components/ui/ChipInput'
import Badge from '../../components/ui/Badge'
import { toast } from '../../lib/toast'
import ProfilePdfIngest from '../../components/profile/ProfilePdfIngest'
import { LogoBadge, LogoWordmark } from '../../components/Logo'

import { useProfile, useSaveProfile } from '../../api/profile'
import { useKeywordGeneration } from '../../lib/useKeywordGeneration'
import { useCan } from '../../lib/useCan'
import { CAPABILITY_PRESETS, DEFAULT_VALUE_MIN } from '../../lib/profilePresets'
import type { OtakAgentFormPatch } from '../../components/profile/types'

const steps: StepItem[] = [
  { id: 'path', label: 'Cara mulai' },
  { id: 'form', label: 'Isi profil' },
  { id: 'activate', label: 'Aktifkan' },
]

type Path = 'pdf' | 'manual' | null

// Discovery (EP-12) doesn't exist yet — this is a best-effort no-op so
// activation never blocks on a run endpoint that isn't built. Replace the
// body with a real POST /api/discovery/run call once EP-12 ships.
async function triggerFirstDiscovery(): Promise<void> {
  return Promise.resolve()
}

export default function Onboarding() {
  const navigate = useNavigate()
  const canEdit = useCan('EditProfile')
  const { data: profile } = useProfile()
  const [stepIndex, setStepIndex] = useState(0)
  const [path, setPath] = useState<Path>(null)

  const [companyName, setCompanyName] = useState('')
  const [capabilities, setCapabilities] = useState<string[]>([])
  const [valueMin, setValueMin] = useState(String(DEFAULT_VALUE_MIN))
  const [errors, setErrors] = useState<{ companyName?: string }>({})

  const saveProfile = useSaveProfile()
  const { generate, degraded: keywordsDegraded, isPending: generatingKeywords } = useKeywordGeneration()
  const [generatedKeywords, setGeneratedKeywords] = useState<string[]>([])
  const [generatedNegativeKeywords, setGeneratedNegativeKeywords] = useState<string[]>([])

  function choosePath(next: Exclude<Path, null>) {
    setPath(next)
    setStepIndex(1)
  }

  function skip() {
    navigate('/')
  }

  // ProfilePdfIngest only merges the AI's findings into a form-shaped patch
  // (it never saves on its own) — the fast onboarding path has no full form
  // to review it in, so save immediately here with the same fields the
  // manual path below saves (company_name, service_categories), then run the
  // same activation follow-through: best-effort discovery kick + redirect.
  async function handleDraftApplied(patch: OtakAgentFormPatch, docRef: string) {
    try {
      await saveProfile.mutateAsync({
        company_name: patch.companyName || companyName || 'Perusahaan',
        one_liner: patch.oneLiner,
        service_categories: patch.serviceCategories,
        tech_stack: patch.techStack,
        source_doc_refs: [...(profile?.source_doc_refs ?? []), docRef],
      })
      toast.success('Profil diperbarui dari PDF.')
    } catch {
      toast.error('Gagal menyimpan profil.')
      return
    }
    try {
      await triggerFirstDiscovery()
    } catch {
      // best-effort — discovery (EP-12) not built yet, never block activation
    }
    navigate('/discovery')
  }

  async function handleGenerateKeywords() {
    const res = await generate(capabilities)
    if (!res) return
    setGeneratedKeywords(res.keywords)
    // Keep the negative keywords too — the degrade path returns the preset
    // negatives, and dropping them here would save a profile with none.
    setGeneratedNegativeKeywords(res.negative_keywords)
  }

  function validateForm(): boolean {
    const next: typeof errors = {}
    if (!companyName.trim()) next.companyName = 'Nama perusahaan wajib diisi.'
    setErrors(next)
    return Object.keys(next).length === 0
  }

  function goToActivate() {
    if (!validateForm()) return
    setStepIndex(2)
  }

  async function activateAgent() {
    if (!validateForm()) {
      setStepIndex(1)
      return
    }
    const parsedValueMin = parseFloat(valueMin)
    try {
      await saveProfile.mutateAsync({
        company_name: companyName,
        service_categories: capabilities,
        target: {
          value_min: !isNaN(parsedValueMin) ? parsedValueMin : undefined,
        },
        keywords:
          generatedKeywords.length > 0 || generatedNegativeKeywords.length > 0
            ? [{ keywords: generatedKeywords, negative_keywords: generatedNegativeKeywords }]
            : undefined,
      })
      try {
        await triggerFirstDiscovery()
      } catch {
        // best-effort — discovery (EP-12) not built yet, never block activation
      }
      toast.success('Otak agent berhasil diaktifkan.')
      navigate('/discovery')
    } catch {
      toast.error('Gagal menyimpan profil, coba lagi.')
    }
  }

  return (
    <div className="min-h-screen bg-surface-subtle flex flex-col items-center px-4 py-10">
      <div className="w-full max-w-2xl flex flex-col gap-6">
        <div className="flex flex-col items-center gap-2 text-center">
          <LogoBadge size={40} className="mb-1" />
          <h1 className="text-h2 font-semibold text-fg">
            Selamat datang di <LogoWordmark />
          </h1>
          <p className="text-body text-fg-muted">
            Atur "Otak Agent" secepat mungkin (&lt; 2 menit) agar AI bisa mulai mencari tender.
          </p>
        </div>

        <Stepper steps={steps} current={stepIndex} className="px-4" />

        {stepIndex === 0 && (
          <div className="flex flex-col gap-4">
            <div className="grid sm:grid-cols-2 gap-4">
              <Card className="flex flex-col">
                <CardBody className="flex flex-col gap-3 items-center text-center flex-1 justify-between">
                  <div className="flex flex-col items-center gap-3">
                    <UploadCloud className="w-7 h-7 text-primary" aria-hidden="true" />
                    <div>
                      <h2 className="text-body font-semibold text-fg">Cara cepat</h2>
                      <p className="text-caption text-fg-muted mt-1">
                        Upload PDF company profile / capability deck → AI isi otomatis
                      </p>
                    </div>
                  </div>
                  <Button onClick={() => choosePath('pdf')} className="w-full">
                    Mulai unggah
                  </Button>
                </CardBody>
              </Card>

              <Card className="flex flex-col">
                <CardBody className="flex flex-col gap-3 items-center text-center flex-1 justify-between">
                  <div className="flex flex-col items-center gap-3">
                    <PenLine className="w-7 h-7 text-primary" aria-hidden="true" />
                    <div>
                      <h2 className="text-body font-semibold text-fg">Isi manual</h2>
                      <p className="text-caption text-fg-muted mt-1">
                        Isi beberapa pilihan (chip &amp; angka), kurang dari 2 menit
                      </p>
                    </div>
                  </div>
                  <Button onClick={() => choosePath('manual')} className="w-full">
                    Mulai isi
                  </Button>
                </CardBody>
              </Card>
            </div>

            <button
              type="button"
              onClick={skip}
              className="text-caption text-fg-muted hover:text-fg hover:underline mx-auto"
            >
              Lewati, atur nanti
            </button>
          </div>
        )}

        {stepIndex === 1 && path === 'pdf' && (
          <Card>
            <CardBody className="flex flex-col gap-4">
              <ProfilePdfIngest onDraftApplied={handleDraftApplied} />
              <div className="flex items-center justify-between pt-2">
                <button
                  type="button"
                  onClick={() => setStepIndex(0)}
                  className="text-caption text-fg-muted hover:text-fg hover:underline"
                >
                  Kembali
                </button>
                <button
                  type="button"
                  onClick={skip}
                  className="text-caption text-fg-muted hover:text-fg hover:underline"
                >
                  Lewati, atur nanti
                </button>
              </div>
            </CardBody>
          </Card>
        )}

        {stepIndex === 1 && path === 'manual' && (
          <Card>
            <CardBody className="flex flex-col gap-4">
              <Field label="Nama perusahaan" required error={errors.companyName}>
                <Input
                  value={companyName}
                  onChange={(e) => setCompanyName(e.target.value)}
                  invalid={!!errors.companyName}
                  placeholder="PT Contoh Teknologi"
                />
              </Field>

              <Field label="Kapabilitas (yang dijual)" helper="Pilih preset atau tambah sendiri">
                <ChipInput
                  value={capabilities}
                  onChange={setCapabilities}
                  presets={CAPABILITY_PRESETS}
                  placeholder="Tambah kapabilitas…"
                />
              </Field>

              <Field label="Nilai minimum" helper="Nilai tender minimum yang relevan (Rp)">
                <Input
                  type="number"
                  min="0"
                  value={valueMin}
                  onChange={(e) => setValueMin(e.target.value)}
                />
              </Field>

              <div className="flex flex-col gap-2">
                <Button
                  variant="secondary"
                  leftIcon={<Sparkles className="w-4 h-4" />}
                  loading={generatingKeywords}
                  onClick={handleGenerateKeywords}
                  disabled={!canEdit}
                  className="self-start"
                >
                  Generate keyword dari kapabilitas
                </Button>
                {generatedKeywords.length > 0 && (
                  <div className="flex flex-wrap items-center gap-1.5">
                    {keywordsDegraded && <Badge tone="warning">AI tidak tersedia</Badge>}
                    {generatedKeywords.map((k) => (
                      <span
                        key={k}
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded-pill bg-accent/10 text-accent border border-accent text-caption font-medium"
                      >
                        <Sparkles className="w-3 h-3" aria-hidden="true" />
                        {k}
                      </span>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex items-center justify-between pt-2">
                <button
                  type="button"
                  onClick={skip}
                  className="text-caption text-fg-muted hover:text-fg hover:underline"
                >
                  Lewati, atur nanti
                </button>
                <Button onClick={goToActivate}>Lanjut</Button>
              </div>
            </CardBody>
          </Card>
        )}

        {stepIndex === 2 && (
          <Card>
            <CardBody className="flex flex-col gap-4 items-center text-center">
              <Sparkles className="w-8 h-8 text-primary" aria-hidden="true" />
              <div>
                <h2 className="text-body font-semibold text-fg">Siap mengaktifkan agent</h2>
                <p className="text-caption text-fg-muted mt-1">
                  Profil akan disimpan dan AI mulai mencari peluang yang relevan.
                </p>
              </div>
              <Button
                loading={saveProfile.isPending}
                onClick={activateAgent}
                disabled={!canEdit}
                className="w-full sm:w-auto"
              >
                Aktifkan Agent
              </Button>
              {!canEdit && (
                <p className="text-caption text-danger">
                  Peran Anda tidak punya akses mengubah Otak Agent — hubungi Ops/Admin.
                </p>
              )}
              <button
                type="button"
                onClick={skip}
                className="text-caption text-fg-muted hover:text-fg hover:underline"
              >
                Lewati, atur nanti
              </button>
            </CardBody>
          </Card>
        )}
      </div>
    </div>
  )
}
