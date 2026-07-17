import { useState } from 'react'
import { FileText, X } from 'lucide-react'

import FileDropzone from '../ui/FileDropzone'
import Button from '../ui/Button'
import { announce } from '../../lib/a11y'

import { useIngestProfilePdf } from '../../api/profile'
import type { OtakAgentFormPatch } from './types'

export interface ProfilePdfIngestProps {
  disabled?: boolean
  /** Called once with the fields the AI actually found (already merged with
   * the doc ref) — the caller (SettingsProfile) patches the live edit form
   * with it so the user reviews everything via the normal cards, then saves
   * with the single existing "Simpan" action (no separate save here). */
  onDraftApplied: (patch: OtakAgentFormPatch, docRef: string) => void
  className?: string
}

/** Dropzone → AI extraction → merge into the live Otak Agent form (EP-13
 * ST-13.3). Never saves on its own: extraction only produces a form patch,
 * which the user reviews/edits via the normal cards before the one "Simpan"
 * button persists it via PUT /api/profile. */
export default function ProfilePdfIngest({ disabled, onDraftApplied, className }: ProfilePdfIngestProps) {
  const ingest = useIngestProfilePdf()

  const [degraded, setDegraded] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [appliedSections, setAppliedSections] = useState<string[] | null>(null)

  async function handleFiles(files: File[]) {
    const file = files[0]
    if (!file) return

    setUploadError(null)
    setDegraded(false)
    setAppliedSections(null)
    announce('AI membaca dokumen…')

    try {
      const res = await ingest.mutateAsync(file)

      if (!res.draft) {
        setDegraded(true)
        announce('AI tidak dapat membaca dokumen ini, silakan isi manual.', 'assertive')
        return
      }

      const d = res.draft
      const t = d.target
      const patch: OtakAgentFormPatch = {}
      const sections: string[] = []

      if (d.company_name || d.one_liner) {
        if (d.company_name) patch.companyName = d.company_name
        if (d.one_liner) patch.oneLiner = d.one_liner
        sections.push('Identitas')
      }
      if (d.service_categories.length || d.tech_stack.length || d.products.length) {
        if (d.service_categories.length) patch.serviceCategories = d.service_categories
        if (d.tech_stack.length) patch.techStack = d.tech_stack
        if (d.products.length) patch.products = d.products
        sections.push('Produk & Layanan')
      }
      if (d.portfolio_refs.length) {
        patch.portfolioRefs = d.portfolio_refs
        sections.push('Bukti/Portfolio')
      }
      if (d.vision || d.mission) {
        if (d.vision) patch.vision = d.vision
        if (d.mission) patch.mission = d.mission
        sections.push('Visi & Misi')
      }
      if (
        t.countries.length ||
        t.industries.length ||
        t.value_min != null ||
        t.value_ideal != null ||
        t.value_max != null ||
        t.deadline_min_days != null ||
        t.procurement_types.length ||
        t.buyer_size_note ||
        t.document_languages.length ||
        t.work_model ||
        t.onsite_limit_note ||
        t.decision_maker_roles.length
      ) {
        if (t.countries.length) patch.countries = t.countries
        if (t.industries.length) patch.industries = t.industries
        if (t.value_min != null) patch.valueMin = String(t.value_min)
        if (t.value_ideal != null) patch.valueIdeal = String(t.value_ideal)
        if (t.value_max != null) patch.valueMax = String(t.value_max)
        if (t.deadline_min_days != null) patch.deadlineMinDays = String(t.deadline_min_days)
        if (t.procurement_types.length) patch.procurementTypes = t.procurement_types
        if (t.buyer_size_note) patch.buyerSizeNote = t.buyer_size_note
        if (t.document_languages.length) patch.documentLanguages = t.document_languages
        if (t.work_model) patch.workModel = t.work_model
        if (t.onsite_limit_note) patch.onsiteLimitNote = t.onsite_limit_note
        if (t.decision_maker_roles.length) patch.decisionMakerRoles = t.decision_maker_roles
        sections.push('Target Peluang')
      }
      if (d.nogo_custom.length) {
        patch.customNoGo = d.nogo_custom
        sections.push('No-Go')
      }
      if (d.keywords.length || d.negative_keywords.length) {
        if (d.keywords.length) patch.keywords = d.keywords
        if (d.negative_keywords.length) patch.negativeKeywords = d.negative_keywords
        sections.push('Keyword')
      }

      onDraftApplied(patch, res.doc_ref)
      setAppliedSections(sections)
      announce(
        sections.length
          ? `AI selesai membaca dokumen — ${sections.length} bagian terisi. Tinjau lalu simpan.`
          : 'AI selesai membaca dokumen, namun tidak ada field yang ditemukan.'
      )
    } catch {
      setUploadError('Gagal mengunggah PDF. Coba lagi atau isi manual.')
      announce('Gagal mengunggah PDF.', 'assertive')
    }
  }

  const isBusy = ingest.isPending

  return (
    <div className={className}>
      {appliedSections === null && !degraded && (
        <div className="flex flex-col gap-2">
          <FileDropzone
            onFiles={handleFiles}
            disabled={disabled}
            loading={isBusy}
            loadingLabel="AI membaca dokumen — bisa memakan beberapa menit untuk dokumen panjang…"
            maxSizeMB={10}
          />
          {uploadError && <p className="text-caption text-danger">{uploadError}</p>}
        </div>
      )}

      {degraded && (
        <div className="flex flex-col gap-2 p-4 rounded-card border border-line bg-surface-subtle">
          <p className="text-caption text-fg-muted">
            AI tidak dapat membaca dokumen ini (kemungkinan hasil scan atau tidak tersedia).
            Silakan isi profil secara manual.
          </p>
          <Button variant="secondary" size="sm" className="self-start" onClick={() => setDegraded(false)}>
            Coba unggah lagi
          </Button>
        </div>
      )}

      {appliedSections !== null && (
        <div className="flex items-start gap-2 p-3 rounded-card border border-l-4 border-line border-l-warning bg-surface text-caption text-fg shadow-subtle">
          <FileText className="w-4 h-4 mt-0.5 shrink-0 text-warning" aria-hidden="true" />
          <div className="flex-1">
            {appliedSections.length > 0 ? (
              <>
                Terisi dari PDF: <span className="font-medium">{appliedSections.join(', ')}</span>. Tinjau di
                kartu terkait di bawah — <span className="font-semibold">perubahan ini belum tersimpan</span>,
                klik <span className="font-semibold">Simpan</span> di bagian bawah halaman untuk menyimpannya.
              </>
            ) : (
              'AI tidak menemukan field yang bisa diisi dari dokumen ini.'
            )}
          </div>
          <button
            type="button"
            onClick={() => setAppliedSections(null)}
            className="text-fg-subtle hover:text-fg shrink-0"
            aria-label="Tutup"
          >
            <X className="w-3.5 h-3.5" aria-hidden="true" />
          </button>
        </div>
      )}
    </div>
  )
}
