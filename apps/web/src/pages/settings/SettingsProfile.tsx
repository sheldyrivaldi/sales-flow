import { useEffect, useState } from 'react'
import { Boxes, Building2, Download, Globe, Radar, Target, UploadCloud, UserCircle } from 'lucide-react'

import Card, { CardBody } from '../../components/ui/Card'
import Badge from '../../components/ui/Badge'
import Button from '../../components/ui/Button'
import { SectionHeader, GroupLabel } from '../../components/ui/SectionHeader'
import { SkeletonText } from '../../components/ui/Skeleton'
import ProfileCard from '../../components/profile/ProfileCard'
import CapabilitiesCard from '../../components/profile/CapabilitiesCard'
import SupportDocsCard from '../../components/profile/SupportDocsCard'
import VisionMissionCard from '../../components/profile/VisionMissionCard'
import TargetCard from '../../components/profile/TargetCard'
import NoGoCard from '../../components/profile/NoGoCard'
import SourcesKeywordCard from '../../components/profile/SourcesKeywordCard'
import ScoringCard from '../../components/profile/ScoringCard'
import SourcesTab from '../../components/profile/SourcesTab'
import ProfilePdfIngest from '../../components/profile/ProfilePdfIngest'
import type { OtakAgentFormState, OtakAgentFormPatch } from '../../components/profile/types'

import { useProfile, useSaveProfile, isProfileConfigured } from '../../api/profile'
import type { Profile, ProfileUpdateBody, KeywordSet } from '../../api/profile'
import { useSources } from '../../api/sources'
import { DEFAULT_VALUE_MIN, DEFAULT_DEADLINE_MIN_DAYS } from '../../lib/profilePresets'
import { dedupCaseInsensitive } from '../../lib/dedup'
import { formatRelative } from '../../lib/format'
import { exportProfilePdf } from '../../lib/exportProfilePdf'
import { toast } from '../../lib/toast'
import { useCan } from '../../lib/useCan'
import { useAuthStore } from '../../store/auth'

const ROLE_LABELS: Record<string, string> = {
  SALES: 'Sales',
  OPS: 'Operations',
  MANAGER: 'Manager',
  ADMIN: 'Admin',
}

// Mirrors internal/service/profile_service.go's defaultScoringConfig() —
// used whenever the profile has never touched the Scoring card.
const DEFAULT_SCORING = {
  weightCapabilityFit: '20',
  weightPortfolioMatch: '15',
  weightCommercialAttractiveness: '15',
  weightEligibilityFit: '15',
  weightDeadlineFeasibility: '10',
  weightStrategicAccountValue: '10',
  weightDeliveryRisk: '10',
  weightCompetitionWinProbability: '5',
  thresholdPursue: '80',
  thresholdReview: '65',
  thresholdWatchlist: '50',
}

// partitionKeywordSets splits a profile's keyword sets into the ones the flat
// editor round-trips (the default bucket: category-less, Indonesian/unset
// language) and the rest. Categorized / other-language sets are preserved
// verbatim on save so editing here never collapses them into one bucket.
function partitionKeywordSets(sets: KeywordSet[]): { editable: KeywordSet[]; preserved: KeywordSet[] } {
  const editable: KeywordSet[] = []
  const preserved: KeywordSet[] = []
  for (const s of sets) {
    if (!s.category && (!s.language || s.language === 'id')) {
      editable.push(s)
    } else {
      preserved.push(s)
    }
  }
  return { editable, preserved }
}

function mergeKeywordSets(sets: KeywordSet[]): { keywords: string[]; negativeKeywords: string[] } {
  return {
    keywords: dedupCaseInsensitive(sets.flatMap((s) => s.keywords)),
    negativeKeywords: dedupCaseInsensitive(sets.flatMap((s) => s.negative_keywords)),
  }
}

function profileToForm(p: Profile, editableSets: KeywordSet[]): OtakAgentFormState {
  const { keywords, negativeKeywords } = mergeKeywordSets(editableSets)
  const sc = p.scoring
  return {
    companyName: p.company_name,
    oneLiner: p.one_liner ?? '',
    serviceCategories: p.service_categories,
    techStack: p.tech_stack,
    products: p.products,
    portfolioRefs: p.portfolio_refs,
    supportDocuments: p.support_documents ?? [],
    vision: p.vision ?? '',
    mission: p.mission ?? '',
    countries: p.target?.countries ?? [],
    industries: p.target?.industries ?? [],
    valueMin: p.target?.value_min != null ? String(p.target.value_min) : String(DEFAULT_VALUE_MIN),
    valueIdeal: p.target?.value_ideal != null ? String(p.target.value_ideal) : '',
    valueMax: p.target?.value_max != null ? String(p.target.value_max) : '',
    deadlineMinDays:
      p.target?.deadline_min_days != null
        ? String(p.target.deadline_min_days)
        : String(DEFAULT_DEADLINE_MIN_DAYS),
    procurementTypes: p.target?.procurement_types ?? [],
    buyerSizeNote: p.target?.buyer_size_note ?? '',
    documentLanguages: p.target?.document_languages ?? [],
    workModel: p.target?.work_model ?? '',
    onsiteLimitNote: p.target?.onsite_limit_note ?? '',
    decisionMakerRoles: p.target?.decision_maker_roles ?? [],
    presetFlags: p.nogo?.preset_flags ?? [],
    customNoGo: p.nogo?.custom ?? [],
    keywords,
    negativeKeywords,
    crawlEnabled: p.crawl_enabled,
    crawlFrequency: p.crawl_frequency,
    sourceDocRefs: p.source_doc_refs,
    weightCapabilityFit: sc ? String(sc.weight_capability_fit) : DEFAULT_SCORING.weightCapabilityFit,
    weightPortfolioMatch: sc ? String(sc.weight_portfolio_match) : DEFAULT_SCORING.weightPortfolioMatch,
    weightCommercialAttractiveness: sc
      ? String(sc.weight_commercial_attractiveness)
      : DEFAULT_SCORING.weightCommercialAttractiveness,
    weightEligibilityFit: sc ? String(sc.weight_eligibility_fit) : DEFAULT_SCORING.weightEligibilityFit,
    weightDeadlineFeasibility: sc
      ? String(sc.weight_deadline_feasibility)
      : DEFAULT_SCORING.weightDeadlineFeasibility,
    weightStrategicAccountValue: sc
      ? String(sc.weight_strategic_account_value)
      : DEFAULT_SCORING.weightStrategicAccountValue,
    weightDeliveryRisk: sc ? String(sc.weight_delivery_risk) : DEFAULT_SCORING.weightDeliveryRisk,
    weightCompetitionWinProbability: sc
      ? String(sc.weight_competition_win_probability)
      : DEFAULT_SCORING.weightCompetitionWinProbability,
    thresholdPursue: sc ? String(sc.threshold_pursue) : DEFAULT_SCORING.thresholdPursue,
    thresholdReview: sc ? String(sc.threshold_review) : DEFAULT_SCORING.thresholdReview,
    thresholdWatchlist: sc ? String(sc.threshold_watchlist) : DEFAULT_SCORING.thresholdWatchlist,
  }
}

function toIntOrUndefined(s: string): number | undefined {
  const n = parseInt(s, 10)
  return isNaN(n) ? undefined : n
}

function formToBody(form: OtakAgentFormState, preservedKeywordSets: KeywordSet[]): ProfileUpdateBody {
  const valueMin = parseFloat(form.valueMin)
  const valueIdeal = parseFloat(form.valueIdeal)
  const valueMax = parseFloat(form.valueMax)
  const deadlineMinDays = parseInt(form.deadlineMinDays, 10)

  return {
    company_name: form.companyName,
    one_liner: form.oneLiner || undefined,
    service_categories: form.serviceCategories,
    tech_stack: form.techStack,
    products: form.products,
    portfolio_refs: form.portfolioRefs,
    support_documents: form.supportDocuments,
    vision: form.vision || undefined,
    mission: form.mission || undefined,
    source_doc_refs: form.sourceDocRefs,
    crawl_frequency: form.crawlFrequency,
    crawl_enabled: form.crawlEnabled,
    target: {
      countries: form.countries,
      industries: form.industries,
      value_min: !isNaN(valueMin) ? valueMin : undefined,
      value_ideal: form.valueIdeal !== '' && !isNaN(valueIdeal) ? valueIdeal : undefined,
      value_max: form.valueMax !== '' && !isNaN(valueMax) ? valueMax : undefined,
      deadline_min_days: !isNaN(deadlineMinDays) ? deadlineMinDays : undefined,
      procurement_types: form.procurementTypes,
      buyer_size_note: form.buyerSizeNote || undefined,
      document_languages: form.documentLanguages,
      work_model: form.workModel || undefined,
      onsite_limit_note: form.onsiteLimitNote || undefined,
      decision_maker_roles: form.decisionMakerRoles,
    },
    nogo: {
      preset_flags: form.presetFlags,
      custom: form.customNoGo,
    },
    keywords: [
      // Preserve categorized / other-language sets untouched, then the edited
      // default bucket. Replaces the whole set on the backend (mergeKeywords),
      // so both must be present or the preserved ones would be lost.
      ...preservedKeywordSets.map((s) => ({
        category: s.category ?? undefined,
        keywords: s.keywords,
        negative_keywords: s.negative_keywords,
        language: s.language,
      })),
      { keywords: form.keywords, negative_keywords: form.negativeKeywords },
    ],
    scoring: {
      weight_capability_fit: toIntOrUndefined(form.weightCapabilityFit),
      weight_portfolio_match: toIntOrUndefined(form.weightPortfolioMatch),
      weight_commercial_attractiveness: toIntOrUndefined(form.weightCommercialAttractiveness),
      weight_eligibility_fit: toIntOrUndefined(form.weightEligibilityFit),
      weight_deadline_feasibility: toIntOrUndefined(form.weightDeadlineFeasibility),
      weight_strategic_account_value: toIntOrUndefined(form.weightStrategicAccountValue),
      weight_delivery_risk: toIntOrUndefined(form.weightDeliveryRisk),
      weight_competition_win_probability: toIntOrUndefined(form.weightCompetitionWinProbability),
      threshold_pursue: toIntOrUndefined(form.thresholdPursue),
      threshold_review: toIntOrUndefined(form.thresholdReview),
      threshold_watchlist: toIntOrUndefined(form.thresholdWatchlist),
    },
  }
}

function UserSection() {
  const user = useAuthStore((s) => s.user)
  if (!user) return null

  return (
    <Card className="max-w-md">
      <CardBody className="flex flex-col gap-4">
        <div>
          <p className="text-caption text-fg-muted">Nama</p>
          <p className="text-body font-medium text-fg">{user.name}</p>
        </div>
        <div>
          <p className="text-caption text-fg-muted">Email</p>
          <p className="text-body font-medium text-fg">{user.email}</p>
        </div>
        <div>
          <p className="text-caption text-fg-muted">Role</p>
          <Badge tone="info">{ROLE_LABELS[user.role] ?? user.role}</Badge>
        </div>
        <p className="text-caption text-fg-subtle">
          Akun dikelola oleh Admin — hubungi Admin untuk mengubah nama, email, atau role.
        </p>
      </CardBody>
    </Card>
  )
}

export default function SettingsProfile() {
  const { data: profile, isLoading } = useProfile()
  const saveProfile = useSaveProfile()
  const canEdit = useCan('EditProfile')
  // page_size high enough to cover the whole list for export — this page
  // doesn't paginate sources itself (SourcesTab does its own fetch).
  const { data: sourcesPage } = useSources({ page_size: 200 })

  const [form, setForm] = useState<OtakAgentFormState | null>(null)
  const [preservedKeywordSets, setPreservedKeywordSets] = useState<KeywordSet[]>([])
  const [initialized, setInitialized] = useState(false)
  const [companyNameError, setCompanyNameError] = useState<string | undefined>()
  const [valueMinError, setValueMinError] = useState<string | undefined>()
  const [showPdfIngest, setShowPdfIngest] = useState(false)
  // Snapshot form persis seperti terakhir disimpan/dimuat — tombol Simpan
  // hanya aktif bila form menyimpang dari snapshot ini (dirty check).
  const [savedSnapshot, setSavedSnapshot] = useState<string>('')

  function syncFromProfile(p: Profile) {
    const { editable, preserved } = partitionKeywordSets(p.keywords)
    const next = profileToForm(p, editable)
    setForm(next)
    setPreservedKeywordSets(preserved)
    setSavedSnapshot(JSON.stringify(next))
  }

  useEffect(() => {
    if (profile && !initialized) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- init form once from loaded profile (same justified pattern as ProspectFormDrawer), not a per-render reset.
      syncFromProfile(profile)
      setInitialized(true)
    }
  }, [profile, initialized])

  function patchForm(patch: OtakAgentFormPatch) {
    setForm((f) => (f ? { ...f, ...patch } : f))
    if (patch.companyName !== undefined) setCompanyNameError(undefined)
    if (patch.valueMin !== undefined) setValueMinError(undefined)
  }

  function handleDraftApplied(patch: OtakAgentFormPatch, docRef: string) {
    setForm((f) => (f ? { ...f, ...patch, sourceDocRefs: [...f.sourceDocRefs, docRef] } : f))
  }

  function handleExport() {
    if (!profile) return
    exportProfilePdf(profile, sourcesPage?.items ?? [])
  }

  async function handleSave() {
    if (!form) return
    let invalid = false
    if (!form.companyName.trim()) {
      setCompanyNameError('Nama perusahaan wajib diisi.')
      invalid = true
    }
    if (form.valueMin.trim() === '' || isNaN(parseFloat(form.valueMin))) {
      setValueMinError('Nilai minimum wajib diisi.')
      invalid = true
    }
    if (invalid) return
    try {
      const updated = await saveProfile.mutateAsync(formToBody(form, preservedKeywordSets))
      // Re-sync from the persisted result so the form shows exactly what the
      // server stored — the PUT merges over the prior version (a blank field
      // keeps its old value), and without this re-sync the UI would silently
      // diverge from what was actually saved.
      syncFromProfile(updated)
      toast.success('Profil perusahaan berhasil disimpan.')
    } catch {
      toast.error('Gagal menyimpan profil perusahaan.')
    }
  }

  const disabled = !canEdit || saveProfile.isPending
  const isDirty = !!form && JSON.stringify(form) !== savedSnapshot

  return (
    <div className="flex flex-col gap-8 pb-24 max-w-6xl">
      <div>
        <h1 className="text-h2 font-semibold text-fg">Profile</h1>
        <p className="text-body text-fg-muted mt-1">Akun kamu dan profil perusahaan yang dipakai AI mencari tender.</p>
      </div>

      {/* ── Section 1: Profil User ──────────────────────────────────── */}
      <section className="flex flex-col gap-4">
        <SectionHeader
          icon={UserCircle}
          tone="slate"
          title="Profil User"
          description="Akun kamu di SalesFlow — dikelola oleh Admin."
        />
        <UserSection />
      </section>

      <div className="border-t border-line" role="separator" />

      {/* ── Section 2: Profil Perusahaan ────────────────────────────── */}
      <section className="flex flex-col gap-5">
        <SectionHeader
          icon={Building2}
          tone="emerald"
          title="Profil Perusahaan"
          description="Dipakai AI untuk mencari & menilai kecocokan tender. Makin lengkap, makin akurat."
          right={
            profile ? (
              <Badge tone={isProfileConfigured(profile) ? 'success' : 'warning'} className="shrink-0 whitespace-nowrap">
                {isProfileConfigured(profile)
                  ? `Diperbarui ${formatRelative(profile.updated_at)}`
                  : 'Belum dikonfigurasi'}
              </Badge>
            ) : undefined
          }
        />

        {isLoading || !form ? (
          <SkeletonText lines={6} />
        ) : (
          <>
            {canEdit && (
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  leftIcon={<Download className="w-4 h-4" />}
                  onClick={handleExport}
                  disabled={!profile}
                >
                  Ekspor PDF
                </Button>
                {!showPdfIngest && (
                  <Button
                    variant="secondary"
                    size="sm"
                    leftIcon={<UploadCloud className="w-4 h-4" />}
                    onClick={() => setShowPdfIngest(true)}
                  >
                    Isi dari PDF
                  </Button>
                )}
              </div>
            )}

            {canEdit && showPdfIngest && <ProfilePdfIngest onDraftApplied={handleDraftApplied} />}

            <div className="flex flex-col gap-3">
              <GroupLabel icon={Boxes} tone="emerald" title="Identitas & Kapabilitas" />
              <div className="grid lg:grid-cols-2 gap-4">
                <ProfileCard form={form} onChange={patchForm} disabled={disabled} error={companyNameError} />
                <CapabilitiesCard form={form} onChange={patchForm} disabled={disabled} />
                <VisionMissionCard form={form} onChange={patchForm} disabled={disabled} />
                <SupportDocsCard form={form} onChange={patchForm} disabled={disabled} />
              </div>
            </div>

            <div className="flex flex-col gap-3">
              <GroupLabel icon={Target} tone="ai" title="Target & Kriteria" />
              <div className="grid lg:grid-cols-2 gap-4">
                <TargetCard form={form} onChange={patchForm} disabled={disabled} valueMinError={valueMinError} />
                <NoGoCard form={form} onChange={patchForm} disabled={disabled} />
              </div>
            </div>

            <div className="flex flex-col gap-3">
              <GroupLabel icon={Radar} tone="amber" title="Sumber & Otomasi" />
              <div className="grid lg:grid-cols-2 gap-4">
                <SourcesKeywordCard form={form} onChange={patchForm} disabled={disabled} />
                <ScoringCard form={form} onChange={patchForm} disabled={disabled} />
              </div>
            </div>

            <div className="flex flex-col gap-3">
              <GroupLabel icon={Globe} tone="sky" title="Website Dipantau" />
              <p className="text-caption text-fg-muted -mt-1.5 ml-8">
                Sumber yang dikunjungi AI saat mencari tender. Hanya sumber publik yang bisa di-crawl otomatis.
              </p>
              <SourcesTab canEdit={canEdit} />
            </div>

            {canEdit && (
              <div className="flex items-center justify-end gap-3 mt-2 pt-4 border-t border-line">
                <span className="text-caption text-fg-subtle">
                  {isDirty ? 'Ada perubahan yang belum disimpan.' : 'Semua perubahan sudah tersimpan.'}
                </span>
                {/* Disabled saat form identik dengan snapshot tersimpan —
                    tidak ada yang berubah berarti tidak ada yang perlu
                    disimpan. */}
                <Button loading={saveProfile.isPending} onClick={handleSave} disabled={!form || !isDirty}>
                  Simpan Perubahan
                </Button>
              </div>
            )}
          </>
        )}
      </section>
    </div>
  )
}
