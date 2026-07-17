export type Tone = 'success' | 'warning' | 'info' | 'danger' | 'accent'

export type RecommendedAction =
  | 'Pursue'
  | 'Review'
  | 'Watchlist'
  | 'Reject'
  | 'Need Partner'

export interface ToneClasses {
  text: string
  bg: string
  bgSoft: string
  border: string
}

/** Pemetaan fit_score → tone warna (Design §2.1):
 *  0–49 → danger (rose), 50–64 → info (sky), 65–79 → warning (amber), 80–100 → success (emerald). */
export function scoreColor(score: number): Tone {
  if (score < 50) return 'danger'
  if (score < 65) return 'info'
  if (score < 80) return 'warning'
  return 'success'
}

/** Pemetaan recommended_action → tone warna (Design §2.1). */
export function actionColor(action: RecommendedAction): Tone {
  switch (action) {
    case 'Pursue':       return 'success'
    case 'Review':       return 'warning'
    case 'Watchlist':    return 'info'
    case 'Reject':       return 'danger'
    case 'Need Partner': return 'accent'
  }
}

/** Utility classes Tailwind untuk sebuah tone — record statis (bukan template
 *  string) supaya Tailwind bisa melihat nama class lengkap saat scan source.
 *  Soft badge memakai ramp subtle/strong dari tokens.css agar teks tetap
 *  kontras di atas tint (mis. amber-800 di atas amber-50, bukan amber-500). */
const toneClassMap: Record<Tone, ToneClasses> = {
  success: { text: 'text-success-strong', bg: 'bg-success', bgSoft: 'bg-success-subtle', border: 'border-success-border' },
  warning: { text: 'text-warning-strong', bg: 'bg-warning', bgSoft: 'bg-warning-subtle', border: 'border-warning-border' },
  info:    { text: 'text-info-strong',    bg: 'bg-info',    bgSoft: 'bg-info-subtle',    border: 'border-info-border' },
  danger:  { text: 'text-danger-strong',  bg: 'bg-danger',  bgSoft: 'bg-danger-subtle',  border: 'border-danger-border' },
  accent:  { text: 'text-accent-hover',   bg: 'bg-accent',  bgSoft: 'bg-accent-subtle',  border: 'border-accent' },
}

export function toneClasses(tone: Tone): ToneClasses {
  return toneClassMap[tone]
}
