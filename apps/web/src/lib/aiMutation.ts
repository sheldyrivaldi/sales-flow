import { useIsMutating } from '@tanstack/react-query'

/** Meta yang dibaca MutationCache global (main.tsx): toast + invalidations
 * yang tetap berjalan walau komponen pemicunya sudah unmount. `invalidate`
 * menerima variables mutation supaya query key dinamis (per-target) tetap
 * bisa dibentuk dari level global. */
export interface AIMutationMeta {
  successToast?: string
  errorToast?: string
  invalidate?: (variables: unknown) => unknown[][]
  // Index signature: react-query mengetik `meta` sebagai Record<string,
  // unknown>; tanpa ini setiap assignment meta gagal typecheck.
  [key: string]: unknown
}

/** Kunci mutationKey standar untuk aksi AI — dipakai bersama useIsMutating
 * agar tombol pemicu tetap disabled saat aksi masih berjalan, termasuk
 * setelah user pindah halaman dan kembali (state komponen sudah reset, tapi
 * mutation-nya masih hidup di cache). */
export const AI_MUTATION_KEYS = {
  playbook: ['ai', 'playbook'] as const,
  docChecklist: ['ai', 'doc-checklist'] as const,
  proposal: ['ai', 'proposal'] as const,
  eventAnalysis: ['ai', 'event-analysis'] as const,
} as const

/** true bila masih ada aksi AI dengan kunci `key` yang sedang berjalan untuk
 * target tertentu (dicocokkan longgar terhadap variables mutation). */
export function useAIBusy(key: readonly string[], targetId?: string): boolean {
  const count = useIsMutating({
    mutationKey: key as unknown as string[],
    predicate: (mutation) => {
      if (!targetId) return true
      const v = mutation.state.variables as unknown
      if (typeof v === 'string') return v === targetId
      if (v && typeof v === 'object') {
        const rec = v as Record<string, unknown>
        return rec.id === targetId || rec.targetId === targetId || rec.playbookId === targetId
      }
      return true
    },
  })
  return count > 0
}
