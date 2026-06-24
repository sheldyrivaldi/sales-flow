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

/** Utility classes Tailwind untuk sebuah tone (pakai token, bukan hex). */
export function toneClasses(tone: Tone): ToneClasses {
  return {
    text:   `text-${tone}`,
    bg:     `bg-${tone}`,
    bgSoft: `bg-${tone}/10`,
    border: `border-${tone}`,
  }
}
