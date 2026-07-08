import { useMutation } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────
// Mirror internal/http/dto/keyword.go — keep in sync.

export interface KeywordGenerateBody {
  service_categories: string[]
  language?: string
}

export interface KeywordGenerateResult {
  keywords: string[]
  negative_keywords: string[]
  language: string
  degraded: boolean
}

// ── Mutation Hooks ────────────────────────────────────────────────────────────

/**
 * Generates a draft keyword set from capabilities via Hermes (not persisted —
 * caller merges the result into the profile form and saves via
 * useSaveProfile/PUT /api/profile). `degraded:true` means Hermes failed and
 * only the deterministic negative-keyword preset was returned.
 */
export function useGenerateKeywords() {
  return useMutation({
    mutationFn: (body: KeywordGenerateBody) =>
      apiFetch<KeywordGenerateResult>('/api/profile/keywords/generate', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
  })
}
