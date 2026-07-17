import { useEffect, useRef, useState } from 'react'

/**
 * Menganimasikan sebuah angka dari nilai sebelumnya menuju `target` dengan
 * ease-out (~800ms) — dipakai untuk metric/stat card. Menghormati
 * prefers-reduced-motion (langsung lompat ke nilai akhir tanpa animasi).
 * Visual murni: nilai yang dirender selalu berakhir tepat di `target`.
 */
export function useCountUp(target: number, durationMs = 800): number {
  // Mulai dari 0 supaya load pertama pun ikut teranimasi (0 → nilai),
  // bukan hanya update berikutnya.
  const [display, setDisplay] = useState(0)
  const fromRef = useRef(0)
  const frameRef = useRef(0)

  useEffect(() => {
    const from = fromRef.current
    if (from === target) return

    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      fromRef.current = target
      setDisplay(target)
      return
    }

    const start = performance.now()
    function tick(now: number) {
      const t = Math.min((now - start) / durationMs, 1)
      const eased = 1 - Math.pow(1 - t, 3) // easeOutCubic
      const value = from + (target - from) * eased
      setDisplay(t < 1 ? value : target)
      if (t < 1) {
        frameRef.current = requestAnimationFrame(tick)
      } else {
        fromRef.current = target
      }
    }
    frameRef.current = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(frameRef.current)
  }, [target, durationMs])

  return display
}

/** Varian komponen: render angka count-up dengan formatter opsional. */
export function CountUp({
  value,
  format,
}: {
  value: number
  format?: (n: number) => string
}) {
  const display = useCountUp(value)
  // Selama animasi, nilai antara dibulatkan supaya tidak menampilkan desimal
  // aneh; formatter (mis. formatRupiahShort) menerima angka bulat.
  const rounded = Math.round(display)
  return <>{format ? format(rounded) : rounded.toLocaleString('id-ID')}</>
}
