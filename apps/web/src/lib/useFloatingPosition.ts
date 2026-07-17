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
 */
export function useFloatingPosition(
  triggerRef: RefObject<HTMLElement | null>,
  open: boolean,
  align: 'start' | 'end' = 'start'
): FloatingPosition | null {
  const [pos, setPos] = useState<FloatingPosition | null>(null)

  const recalc = useCallback(() => {
    const el = triggerRef.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    setPos({
      top: rect.bottom + 4,
      left: align === 'end' ? rect.right : rect.left,
      alignEnd: align === 'end',
    })
  }, [triggerRef, align])

  useEffect(() => {
    if (!open) {
      setPos(null)
      return
    }
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
