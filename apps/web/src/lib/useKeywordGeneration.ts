import { useState } from 'react'

import { useGenerateKeywords } from '../api/keywords'
import type { KeywordGenerateResult } from '../api/keywords'
import { toast } from './toast'

// useKeywordGeneration wraps the raw useGenerateKeywords mutation with the
// shared guard + degrade-toast UX used by both the onboarding wizard and the
// Otak Agent editor, so that behaviour lives in one place (previously it was
// copy-pasted, and the copies had already drifted — one dropped the returned
// negative_keywords). Callers decide how to merge the returned result into
// their own form state.
export function useKeywordGeneration() {
  const mutation = useGenerateKeywords()
  const [degraded, setDegraded] = useState(false)

  async function generate(categories: string[]): Promise<KeywordGenerateResult | null> {
    if (categories.length === 0) {
      toast.warning('Pilih kapabilitas terlebih dahulu.')
      return null
    }
    try {
      const res = await mutation.mutateAsync({ service_categories: categories })
      setDegraded(res.degraded)
      if (res.degraded) {
        toast.warning('AI tidak tersedia — hanya keyword negatif preset yang terisi.')
      } else {
        toast.success('Keyword berhasil dibuat, silakan tinjau.')
      }
      return res
    } catch {
      toast.error('Gagal membuat keyword.')
      return null
    }
  }

  return { generate, degraded, isPending: mutation.isPending }
}
