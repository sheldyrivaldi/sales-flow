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

export interface Profile {
  id: string
  company_name: string
  one_liner: string | null
  service_categories: string[]
  tech_stack: string[]
  source_doc_refs: string[]
  version: number
  is_current: boolean
  target: TargetCriteria | null
  nogo: NoGoRule | null
  keywords: KeywordSet[]
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

export interface ProfileUpdateBody {
  company_name: string
  one_liner?: string
  service_categories?: string[]
  tech_stack?: string[]
  source_doc_refs?: string[]
  target?: TargetCriteriaBody
  nogo?: NoGoRuleBody
  keywords?: KeywordSetBody[]
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
