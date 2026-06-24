// Formatter locale id-ID untuk Rupiah, tanggal, dan waktu relatif.
// Catatan: Intl.NumberFormat('id-ID', {currency:'IDR'}) menghasilkan NBSP (U+00A0) antara "Rp" dan angka
// -- gunakan normalizeSpace() agar output stabil di semua environment.

function normalizeSpace(str: string): string {
  return str.replace(/\u00A0/g, ' ')
}

const rupiahFormatter = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
})

export function formatRupiah(value: number): string {
  return normalizeSpace(rupiahFormatter.format(value))
}

export function formatRupiahShort(value: number): string {
  if (value >= 1e9) {
    const n = value / 1e9
    const formatted = n % 1 === 0 ? String(n) : n.toFixed(1).replace('.', ',')
    return `Rp ${formatted} M`
  }
  if (value >= 1e6) {
    const n = value / 1e6
    const formatted = n % 1 === 0 ? String(n) : n.toFixed(1).replace('.', ',')
    return `Rp ${formatted} jt`
  }
  return formatRupiah(value)
}

const tanggalFormatter = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
})

export function formatTanggal(input: Date | string): string {
  const date = typeof input === 'string' ? new Date(input) : input
  return tanggalFormatter.format(date)
}

export function formatRelative(input: Date | string, now?: Date): string {
  const date = typeof input === 'string' ? new Date(input) : input
  const base = now ?? new Date()
  const diffMs = base.getTime() - date.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)
  const diffDay = Math.floor(diffHour / 24)

  if (diffSec < 60) return 'baru saja'
  if (diffMin < 60) return `${diffMin} menit lalu`
  if (diffHour < 24) return `${diffHour} jam lalu`
  if (diffDay === 1) return 'kemarin'
  if (diffDay <= 7) return `${diffDay} hari lalu`
  return formatTanggal(date)
}
