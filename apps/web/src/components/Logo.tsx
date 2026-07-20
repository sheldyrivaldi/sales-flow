import type { CSSProperties } from 'react'
import { cn } from '../lib/cn'

/** Mark inti SalesFlow: anak panah kompas/pilot yang mengarah naik-kanan
 * (navigasi + pertumbuhan sekaligus) — dibelah dua secara diagonal dengan
 * opacity berbeda supaya terlihat seperti kertas terlipat (efek "pesawat
 * kertas"), bukan segitiga datar. Murni currentColor supaya bisa dipakai di
 * atas latar apa pun (gradient emerald di sidebar/login, solid di favicon). */
function LogoMark({ className, style }: { className?: string; style?: CSSProperties }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} style={style} aria-hidden="true">
      <path d="M3 11L22 2L11 13Z" fill="currentColor" />
      <path d="M22 2L13 21L11 13Z" fill="currentColor" fillOpacity="0.62" />
    </svg>
  )
}

export interface LogoProps {
  /** Ukuran sisi badge gradient dalam px. */
  size?: number
  className?: string
}

/** Badge gradient emerald→teal berisi LogoMark — dipakai di sidebar (rail
 * & expanded) serta halaman auth (Login/Onboarding). Satu sumber kebenaran
 * visual untuk identitas SalesFlow agar tidak lagi ada huruf "S" polos yang
 * tersebar dan tidak konsisten di beberapa halaman. */
export function LogoBadge({ size = 32, className }: LogoProps) {
  return (
    <span
      className={cn(
        'relative inline-flex items-center justify-center shrink-0 rounded-btn overflow-hidden',
        'bg-gradient-to-br from-primary-soft via-primary to-accent-hover',
        'shadow-[0_2px_10px_rgba(5,150,105,0.4)]',
        className
      )}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      {/* Kilau halus kiri-atas untuk kedalaman, konsisten dengan tone AI accent aplikasi. */}
      <span
        className="absolute inset-0 bg-gradient-to-br from-white/25 via-transparent to-transparent"
        aria-hidden="true"
      />
      <LogoMark className="relative text-white" style={{ width: size * 0.56, height: size * 0.56 }} />
    </span>
  )
}

/** Wordmark dua-nada: "Sales" netral + "Flow" emerald — dipakai berdampingan
 * dengan LogoBadge di header sidebar dan layar auth. */
export function LogoWordmark({ className }: { className?: string }) {
  return (
    <span className={cn('font-semibold tracking-tight truncate', className)}>
      <span className="text-fg">Sales</span>
      <span className="text-primary">Flow</span>
    </span>
  )
}

export default LogoBadge
