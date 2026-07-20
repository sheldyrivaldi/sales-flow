import { useCallback, useEffect, useState } from 'react'
import type { RefObject } from 'react'

export interface FloatingPosition {
  top: number
  left: number
  /** true when `left` marks the trigger's right edge (align="end") — pair
   * with `transform: translateX(-100%)` so the floated element's right edge
   * lands there instead of its left edge. */
  alignEnd: boolean
}

/** Jarak aman minimum dari tepi layar. */
const VIEWPORT_MARGIN = 8

/**
 * Computes a `position: fixed` anchor point under a trigger element,
 * recalculated whenever it opens and whenever the trigger might have moved
 * (window resize, or scroll on ANY ancestor — capture:true, since scroll on
 * a nested scroll container doesn't bubble to window).
 *
 * Pair with `createPortal(..., document.body)`: a plain `absolute` dropdown
 * gets clipped by the first ancestor with `overflow-hidden`/`overflow-auto`
 * (Modal's scrollable body, Table's horizontal-scroll wrapper, etc.) no
 * matter how high its z-index is — portaling to body plus fixed coordinates
 * escapes that clipping entirely.
 *
 * `contentEl` opsional: berikan ELEMEN panel (bukan ref) supaya perubahannya
 * ikut jadi dependensi dan posisi dihitung ulang tepat saat panel menempel di
 * DOM. Hanya sejak saat itu lebarnya terukur, dan tanpa lebar itu posisi cuma
 * menempel pada tepi trigger sehingga panel lebar menjulur keluar layar pada
 * viewport sempit.
 */
export function useFloatingPosition(
  triggerRef: RefObject<HTMLElement | null>,
  open: boolean,
  align: 'start' | 'end' = 'start',
  contentEl?: HTMLElement | null
): FloatingPosition | null {
  const [pos, setPos] = useState<FloatingPosition | null>(null)

  const recalc = useCallback(() => {
    const el = triggerRef.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    const vw = window.innerWidth
    const width = contentEl?.offsetWidth ?? 0

    let left = align === 'end' ? rect.right : rect.left
    let alignEnd = align === 'end'

    // Jepit ke dalam viewport begitu lebar panel diketahui.
    if (width > 0) {
      if (alignEnd) {
        // Panel menempati [left - width, left].
        if (left - width < VIEWPORT_MARGIN) {
          // Tidak muat ke kiri: balik jadi rata-kiri lalu jepit.
          alignEnd = false
          left = Math.max(VIEWPORT_MARGIN, Math.min(rect.left, vw - VIEWPORT_MARGIN - width))
        }
      } else if (left + width > vw - VIEWPORT_MARGIN) {
        // Panel menempati [left, left + width].
        left = Math.max(VIEWPORT_MARGIN, vw - VIEWPORT_MARGIN - width)
      }
    }

    const next: FloatingPosition = { top: rect.bottom + 4, left, alignEnd }
    setPos((prev) =>
      prev && prev.top === next.top && prev.left === next.left && prev.alignEnd === next.alignEnd
        ? prev // nilai sama — jangan picu render ulang
        : next
    )
  }, [triggerRef, align, contentEl])

  useEffect(() => {
    if (!open) {
      setPos(null)
      return
    }
    // contentEl ada di dependensi recalc, jadi effect ini berjalan lagi begitu
    // panel menempel di DOM — saat itulah penjepitan bisa dihitung.
    recalc()
    window.addEventListener('resize', recalc)
    window.addEventListener('scroll', recalc, true)
    return () => {
      window.removeEventListener('resize', recalc)
      window.removeEventListener('scroll', recalc, true)
    }
  }, [open, recalc])

  return pos
}
