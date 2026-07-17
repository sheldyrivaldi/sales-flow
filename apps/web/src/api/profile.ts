import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────
// Mirror internal/http/dto/profile.go — keep in sync.

export interface TargetCriteria {
  countries: string[]
  industries: string[]
  value_min: number | null
  value_ideal: number | null
  value_max: number | null
  currency: string
  deadline_min_days: number | null
  procurement_types: string[]
  buyer_size_note: string | null
  document_languages: string[]
  work_model: string | null
  onsite_limit_note: string | null
  decision_maker_roles: string[]
}

export interface NoGoRule {
  preset_flags: string[]
  custom: string[]
}

export interface KeywordSet {
  id: string
  category: string | null
  keywords: string[]
  negative_keywords: string[]
  language: string
}

// ScoringConfig holds the configurable rubric weights + recommendation
// thresholds (RFI §8). Mirrors internal/http/dto/profile.go's
// ScoringConfigResponse/Request.
export interface ScoringConfig {
  weight_capability_fit: number
  weight_portfolio_match: number
  weight_commercial_attractiveness: number
  weight_eligibility_fit: number
  weight_deadline_feasibility: number
  weight_strategic_account_value: number
  weight_delivery_risk: number
  weight_competition_win_probability: number
  threshold_pursue: number
  threshold_review: number
  threshold_watchlist: number
}

export interface Profile {
  id: string
  company_name: string
  one_liner: string | null
  service_categories: string[]
  tech_stack: string[]
  products: string[]
  vision: string | null
  mission: string | null
  source_doc_refs: string[]
  portfolio_refs: string[]
  crawl_frequency: string
  crawl_enabled: boolean
  version: number
  is_current: boolean
  target: TargetCriteria | null
  nogo: NoGoRule | null
  keywords: KeywordSet[]
  scoring: ScoringConfig | null
  created_at: string
  updated_at: string
}

export interface TargetCriteriaBody {
  countries?: string[]
  industries?: string[]
  value_min?: number
  value_ideal?: number
  value_max?: number
  currency?: string
  deadline_min_days?: number
  procurement_types?: string[]
  buyer_size_note?: string
  document_languages?: string[]
  work_model?: string
  onsite_limit_note?: string
  decision_maker_roles?: string[]
}

export interface NoGoRuleBody {
  preset_flags?: string[]
  custom?: string[]
}

export interface KeywordSetBody {
  category?: string
  keywords?: string[]
  negative_keywords?: string[]
  language?: string
}

export type ScoringConfigBody = Partial<ScoringConfig>

export interface ProfileUpdateBody {
  company_name: string
  one_liner?: string
  service_categories?: string[]
  tech_stack?: string[]
  products?: string[]
  vision?: string
  mission?: string
  source_doc_refs?: string[]
  portfolio_refs?: string[]
  crawl_frequency?: string
  crawl_enabled?: boolean
  target?: TargetCriteriaBody
  nogo?: NoGoRuleBody
  keywords?: KeywordSetBody[]
  scoring?: ScoringConfigBody
}

// --- PDF Ingest (EP-13) ---

export interface ProfileDraftTarget {
  countries: string[]
  industries: string[]
  value_min: number | null
  value_ideal: number | null
  value_max: number | null
  deadline_min_days: number | null
  procurement_types: string[]
  buyer_size_note: string
  document_languages: string[]
  work_model: string
  onsite_limit_note: string
  decision_maker_roles: string[]
}

export interface ProfileDraft {
  company_name: string
  one_liner: string
  service_categories: string[]
  tech_stack: string[]
  products: string[]
  vision: string
  mission: string
  portfolio_refs: string[]
  keywords: string[]
  negative_keywords: string[]
  nogo_custom: string[]
  target: ProfileDraftTarget
}

export interface IngestResponse {
  doc_ref: string
  filename: string
  size: number
  draft: ProfileDraft | null
  degraded: boolean
}

// ── Helpers ───────────────────────────────────────────────────────────────────

/** version=0 is the never-saved default template (profile_service.go defaultAggregate). */
export function isProfileConfigured(p: Profile | undefined): boolean {
  return !!p && p.version > 0
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

export function useProfile() {
  return useQuery({
    queryKey: ['profile'],
    queryFn: () => apiFetch<Profile>('/api/profile'),
  })
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

export function useSaveProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: ProfileUpdateBody) =>
      apiFetch<Profile>('/api/profile', {
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(['profile'], data)
    },
  })
}

/** Uploads a PDF for AI-assisted Company Profile drafting (EP-13). Never
 * persists on its own — the caller reviews the returned draft and saves it
 * via useSaveProfile (PUT /api/profile). */
export function useIngestProfilePdf() {
  return useMutation({
    mutationFn: (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return apiFetch<IngestResponse>('/api/profile/ingest', {
        method: 'POST',
        body: formData,
      })
    },
  })
}
