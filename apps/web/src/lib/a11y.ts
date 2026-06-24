// Helper aksesibilitas: live-region untuk streaming chat & ekstraksi PDF,
// konstanta kelas focus-ring dan sr-only (selaras idiom Button.tsx).

// Focus ring — idiom proyek (samakan dengan Button.tsx)
export const focusRing =
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2'

// Screen-reader only (Tailwind utility bawaan)
export const srOnly = 'sr-only'

let liveRegionPolite: HTMLElement | null = null
let liveRegionAssertive: HTMLElement | null = null

function getOrCreateRegion(politeness: 'polite' | 'assertive'): HTMLElement {
  const existing = politeness === 'polite' ? liveRegionPolite : liveRegionAssertive
  if (existing) return existing

  const el = document.createElement('div')
  el.setAttribute('aria-live', politeness)
  el.setAttribute('aria-atomic', 'true')
  el.setAttribute('aria-relevant', 'additions text')
  // sr-only: visually hidden but accessible
  el.style.cssText =
    'position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0'
  document.body.appendChild(el)

  if (politeness === 'polite') {
    liveRegionPolite = el
  } else {
    liveRegionAssertive = el
  }
  return el
}

/**
 * Kirim pesan ke aria-live region.
 * Gunakan 'polite' untuk streaming chat & ekstraksi PDF (tidak interupsi),
 * 'assertive' untuk error penting.
 */
export function announce(message: string, politeness: 'polite' | 'assertive' = 'polite'): void {
  if (typeof document === 'undefined') return
  const region = getOrCreateRegion(politeness)
  // Reset lalu set agar screen reader selalu membacakan ulang
  region.textContent = ''
  // Timeout kecil agar perubahan dipicu sebagai event terpisah
  setTimeout(() => {
    region.textContent = message
  }, 50)
}
