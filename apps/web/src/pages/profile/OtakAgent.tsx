import { useEffect, useState } from 'react'
import { Brain } from 'lucide-react'

import Tabs, { TabPanel } from '../../components/ui/Tabs'
import Button from '../../components/ui/Button'
import { SkeletonText } from '../../components/ui/Skeleton'
import ProfileCard from '../../components/profile/ProfileCard'
import CapabilitiesCard from '../../components/profile/CapabilitiesCard'
import TargetCard from '../../components/profile/TargetCard'
import NoGoCard from '../../components/profile/NoGoCard'
import SourcesKeywordCard from '../../components/profile/SourcesKeywordCard'
import ScoringCard from '../../components/profile/ScoringCard'
import SourcesTab from '../../components/profile/SourcesTab'
import type { OtakAgentFormState, OtakAgentFormPatch } from '../../components/profile/types'

import { useProfile, useSaveProfile, isProfileConfigured } from '../../api/profile'
import type { Profile, ProfileUpdateBody, KeywordSet } from '../../api/profile'
import { DEFAULT_VALUE_MIN, DEFAULT_DEADLINE_MIN_DAYS } from '../../lib/profilePresets'
import { dedupCaseInsensitive } from '../../lib/dedup'
import { formatRelative } from '../../lib/format'
import { toast } from '../../lib/toast'
import { useCan } from '../../lib/useCan'

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
  return {
    companyName: p.company_name,
    oneLiner: p.one_liner ?? '',
    serviceCategories: p.service_categories,
    techStack: p.tech_stack,
    countries: p.target?.countries ?? [],
    industries: p.target?.industries ?? [],
    valueMin: p.target?.value_min != null ? String(p.target.value_min) : String(DEFAULT_VALUE_MIN),
    valueIdeal: p.target?.value_ideal != null ? String(p.target.value_ideal) : '',
    deadlineMinDays:
      p.target?.deadline_min_days != null
        ? String(p.target.deadline_min_days)
        : String(DEFAULT_DEADLINE_MIN_DAYS),
    procurementTypes: p.target?.procurement_types ?? [],
    presetFlags: p.nogo?.preset_flags ?? [],
    customNoGo: p.nogo?.custom ?? [],
    keywords,
    negativeKeywords,
  }
}

function formToBody(form: OtakAgentFormState, preservedKeywordSets: KeywordSet[]): ProfileUpdateBody {
  const valueMin = parseFloat(form.valueMin)
  const valueIdeal = parseFloat(form.valueIdeal)
  const deadlineMinDays = parseInt(form.deadlineMinDays, 10)

  return {
    company_name: form.companyName,
    one_liner: form.oneLiner || undefined,
    service_categories: form.serviceCategories,
    tech_stack: form.techStack,
    target: {
      countries: form.countries,
      industries: form.industries,
      value_min: !isNaN(valueMin) ? valueMin : undefined,
      value_ideal: form.valueIdeal !== '' && !isNaN(valueIdeal) ? valueIdeal : undefined,
      deadline_min_days: !isNaN(deadlineMinDays) ? deadlineMinDays : undefined,
      procurement_types: form.procurementTypes,
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
  }
}

export default function OtakAgent() {
  const { data: profile, isLoading } = useProfile()
  const saveProfile = useSaveProfile()
  const canEdit = useCan('EditProfile')

  const [tab, setTab] = useState('profil')
  const [form, setForm] = useState<OtakAgentFormState | null>(null)
  const [preservedKeywordSets, setPreservedKeywordSets] = useState<KeywordSet[]>([])
  const [initialized, setInitialized] = useState(false)
  const [companyNameError, setCompanyNameError] = useState<string | undefined>()
  const [valueMinError, setValueMinError] = useState<string | undefined>()

  function syncFromProfile(p: Profile) {
    const { editable, preserved } = partitionKeywordSets(p.keywords)
    setForm(profileToForm(p, editable))
    setPreservedKeywordSets(preserved)
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
    if (invalid) {
      setTab('profil')
      return
    }
    try {
      const updated = await saveProfile.mutateAsync(formToBody(form, preservedKeywordSets))
      // Re-sync from the persisted result so the form shows exactly what the
      // server stored — the PUT merges over the prior version (a blank field
      // keeps its old value), and without this re-sync the UI would silently
      // diverge from what was actually saved.
      syncFromProfile(updated)
      toast.success('Otak agent diperbarui, discovery berikutnya pakai ini')
    } catch {
      toast.error('Gagal menyimpan Otak Agent.')
    }
  }

  const disabled = !canEdit || saveProfile.isPending

  return (
    <div className="flex flex-col gap-4 pb-20">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Brain className="w-5 h-5 text-primary" aria-hidden="true" />
          <h1 className="text-h2 font-semibold text-fg">Otak Agent</h1>
        </div>
        {profile && (
          <span className="text-caption text-fg-muted">
            {isProfileConfigured(profile)
              ? `Profil dipakai agent • diperbarui ${formatRelative(profile.updated_at)}`
              : 'Profil belum dikonfigurasi — nilai default ditampilkan'}
          </span>
        )}
      </div>

      <Tabs
        tabs={[
          { id: 'profil', label: 'Profil' },
          { id: 'sumber', label: 'Sumber' },
        ]}
        value={tab}
        onChange={setTab}
      />

      {isLoading || !form ? (
        <SkeletonText lines={6} />
      ) : (
        <>
          <TabPanel id="profil" className={tab === 'profil' ? 'flex flex-col gap-4' : 'hidden'}>
            <div className="grid lg:grid-cols-2 gap-4">
              <ProfileCard form={form} onChange={patchForm} disabled={disabled} error={companyNameError} />
              <CapabilitiesCard form={form} onChange={patchForm} disabled={disabled} />
              <TargetCard form={form} onChange={patchForm} disabled={disabled} valueMinError={valueMinError} />
              <NoGoCard form={form} onChange={patchForm} disabled={disabled} />
              <SourcesKeywordCard form={form} onChange={patchForm} disabled={disabled} />
              <ScoringCard />
            </div>
          </TabPanel>

          <TabPanel id="sumber" className={tab === 'sumber' ? 'flex flex-col gap-4' : 'hidden'}>
            <SourcesTab canEdit={canEdit} />
          </TabPanel>
        </>
      )}

      {canEdit && (
        <div className="sticky bottom-0 -mx-6 -mb-6 mt-2 px-6 py-3 border-t border-line bg-surface/95 backdrop-blur flex justify-end">
          <Button loading={saveProfile.isPending} onClick={handleSave} disabled={!form}>
            Simpan
          </Button>
        </div>
      )}
    </div>
  )
}
