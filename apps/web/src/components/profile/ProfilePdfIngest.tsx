import { useState } from 'react'
import { Sparkles, FileText } from 'lucide-react'

import FileDropzone from '../ui/FileDropzone'
import Button from '../ui/Button'
import Field from '../ui/Field'
import Input from '../ui/Input'
import ChipInput from '../ui/ChipInput'
import Badge from '../ui/Badge'
import { toast } from '../../lib/toast'
import { announce } from '../../lib/a11y'

import { useIngestProfilePdf, useSaveProfile } from '../../api/profile'
import type { Profile } from '../../api/profile'

export interface ProfilePdfIngestProps {
  /** Current profile (used to preserve fields the draft didn't touch, and to
   * append the new doc ref onto source_doc_refs rather than replacing it). */
  profile?: Profile
  disabled?: boolean
  onSaved?: (updated: Profile) => void
  className?: string
}

interface DraftForm {
  companyName: string
  oneLiner: string
  serviceCategories: string[]
  techStack: string[]
  // Which fields the AI itself actually populated (vs. falling back to the
  // existing profile value) — drives the "diisi AI ✨" chip per field.
  aiFilled: {
    companyName: boolean
    oneLiner: boolean
    serviceCategories: boolean
    techStack: boolean
  }
}

function AiFilledChip() {
  return (
    <Badge tone="accent" className="gap-1">
      <Sparkles className="w-3 h-3" aria-hidden="true" />
      diisi AI
    </Badge>
  )
}

/** Dropzone → AI extraction → editable review → confirm & save, for Company
 * Profile PDF ingest (EP-13 ST-13.3). Shared between Onboarding and Otak
 * Agent. Never auto-saves: the user always reviews/edits before the draft is
 * merged into the profile via PUT /api/profile (useSaveProfile). */
export default function ProfilePdfIngest({ profile, disabled, onSaved, className }: ProfilePdfIngestProps) {
  const ingest = useIngestProfilePdf()
  const saveProfile = useSaveProfile()

  const [docRef, setDocRef] = useState<string | null>(null)
  const [degraded, setDegraded] = useState(false)
  const [draft, setDraft] = useState<DraftForm | null>(null)
  const [uploadError, setUploadError] = useState<string | null>(null)

  async function handleFiles(files: File[]) {
    const file = files[0]
    if (!file) return

    setUploadError(null)
    setDraft(null)
    setDegraded(false)
    setDocRef(null)
    announce('AI membaca dokumen…')

    try {
      const res = await ingest.mutateAsync(file)
      setDocRef(res.doc_ref)

      if (!res.draft) {
        setDegraded(true)
        announce('AI tidak dapat membaca dokumen ini, silakan isi manual.', 'assertive')
        return
      }

      setDraft({
        companyName: res.draft.company_name || profile?.company_name || '',
        oneLiner: res.draft.one_liner || profile?.one_liner || '',
        serviceCategories: res.draft.service_categories.length
          ? res.draft.service_categories
          : (profile?.service_categories ?? []),
        techStack: res.draft.tech_stack.length ? res.draft.tech_stack : (profile?.tech_stack ?? []),
        aiFilled: {
          companyName: !!res.draft.company_name,
          oneLiner: !!res.draft.one_liner,
          serviceCategories: res.draft.service_categories.length > 0,
          techStack: res.draft.tech_stack.length > 0,
        },
      })
      announce('AI selesai membaca dokumen — silakan tinjau hasilnya di bawah.')
    } catch {
      setUploadError('Gagal mengunggah PDF. Coba lagi atau isi manual.')
      announce('Gagal mengunggah PDF.', 'assertive')
    }
  }

  function patchDraft(patch: Partial<Pick<DraftForm, 'companyName' | 'oneLiner' | 'serviceCategories' | 'techStack'>>) {
    setDraft((d) => (d ? { ...d, ...patch } : d))
  }

  function handleDiscard() {
    setDraft(null)
    setDegraded(false)
    setDocRef(null)
    setUploadError(null)
  }

  async function handleUseAndSave() {
    if (!draft || !docRef) return
    try {
      const updated = await saveProfile.mutateAsync({
        company_name: draft.companyName || 'Perusahaan',
        one_liner: draft.oneLiner || undefined,
        service_categories: draft.serviceCategories,
        tech_stack: draft.techStack,
        source_doc_refs: [...(profile?.source_doc_refs ?? []), docRef],
      })
      toast.success('Profil diperbarui dari PDF.')
      handleDiscard()
      onSaved?.(updated)
    } catch {
      toast.error('Gagal menyimpan profil.')
    }
  }

  const isBusy = ingest.isPending

  return (
    <div className={className}>
      {!draft && !degraded && (
        <div className="flex flex-col gap-2">
          <FileDropzone onFiles={handleFiles} disabled={disabled || isBusy} maxSizeMB={10} />
          {isBusy && (
            <p className="text-caption text-fg-muted flex items-center gap-1.5" role="status">
              <Sparkles className="w-3.5 h-3.5 text-accent animate-pulse" aria-hidden="true" />
              AI membaca dokumen…
            </p>
          )}
          {uploadError && <p className="text-caption text-danger">{uploadError}</p>}
        </div>
      )}

      {degraded && (
        <div className="flex flex-col gap-2 p-4 rounded-card border border-line bg-surface-subtle">
          <p className="text-caption text-fg-muted">
            AI tidak dapat membaca dokumen ini (kemungkinan hasil scan atau tidak tersedia).
            Silakan isi profil secara manual.
          </p>
          <Button variant="secondary" size="sm" className="self-start" onClick={handleDiscard}>
            Coba unggah lagi
          </Button>
        </div>
      )}

      {draft && (
        <div className="flex flex-col gap-4 p-4 rounded-card border border-line bg-surface">
          <div className="flex items-center gap-1.5 text-caption text-fg-muted">
            <FileText className="w-3.5 h-3.5" aria-hidden="true" />
            Hasil ekstraksi — tinjau &amp; edit sebelum disimpan
          </div>

          <Field
            label="Nama perusahaan"
            required
            helper={draft.aiFilled.companyName ? undefined : 'Tidak ditemukan di dokumen'}
          >
            <div className="flex items-center gap-2">
              <Input
                value={draft.companyName}
                onChange={(e) => patchDraft({ companyName: e.target.value })}
                placeholder="PT Contoh Teknologi"
                className="flex-1"
              />
              {draft.aiFilled.companyName && <AiFilledChip />}
            </div>
          </Field>

          <Field label="One-liner" helper={draft.aiFilled.oneLiner ? undefined : 'Tidak ditemukan di dokumen'}>
            <div className="flex items-center gap-2">
              <Input
                value={draft.oneLiner}
                onChange={(e) => patchDraft({ oneLiner: e.target.value })}
                placeholder="Kami membangun software untuk…"
                className="flex-1"
              />
              {draft.aiFilled.oneLiner && <AiFilledChip />}
            </div>
          </Field>

          <Field label="Kategori layanan">
            <div className="flex flex-col gap-1.5">
              <ChipInput value={draft.serviceCategories} onChange={(v) => patchDraft({ serviceCategories: v })} />
              {draft.aiFilled.serviceCategories && <AiFilledChip />}
            </div>
          </Field>

          <Field label="Tech stack">
            <div className="flex flex-col gap-1.5">
              <ChipInput value={draft.techStack} onChange={(v) => patchDraft({ techStack: v })} />
              {draft.aiFilled.techStack && <AiFilledChip />}
            </div>
          </Field>

          <div className="flex items-center justify-end gap-2 pt-1">
            <Button variant="secondary" size="sm" onClick={handleDiscard} disabled={saveProfile.isPending}>
              Buang
            </Button>
            <Button size="sm" loading={saveProfile.isPending} onClick={handleUseAndSave}>
              Gunakan &amp; Simpan
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
