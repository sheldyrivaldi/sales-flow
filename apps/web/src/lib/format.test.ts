import { describe, it, expect } from 'vitest'
import { formatRupiah, formatRupiahShort, formatTanggal, formatRelative } from './format'

// Normalizer: ganti NBSP (U+00A0) dan spasi biasa agar perbandingan stabil
function norm(s: string): string {
  return s.replace(/ /g, ' ')
}

describe('formatRupiah', () => {
  it('2.500.000.000 -> Rp 2.500.000.000', () => {
    expect(norm(formatRupiah(2_500_000_000))).toBe('Rp 2.500.000.000')
  })
  it('0 -> Rp 0', () => {
    expect(norm(formatRupiah(0))).toBe('Rp 0')
  })
  it('1.000 -> Rp 1.000', () => {
    expect(norm(formatRupiah(1_000))).toBe('Rp 1.000')
  })
})

describe('formatRupiahShort', () => {
  it('2.500.000.000 -> Rp 2,5 M', () => {
    expect(formatRupiahShort(2_500_000_000)).toBe('Rp 2,5 M')
  })
  it('1.000.000.000 -> Rp 1 M (tanpa koma)', () => {
    expect(formatRupiahShort(1_000_000_000)).toBe('Rp 1 M')
  })
  it('300.000.000 -> Rp 300 jt', () => {
    expect(formatRupiahShort(300_000_000)).toBe('Rp 300 jt')
  })
  it('500.000 (< 1 jt) -> fallback formatRupiah', () => {
    expect(norm(formatRupiahShort(500_000))).toBe('Rp 500.000')
  })
})

describe('formatTanggal', () => {
  it('Date object -> 24 Jun 2026', () => {
    expect(formatTanggal(new Date('2026-06-24'))).toBe('24 Jun 2026')
  })
  it('ISO string -> 1 Jan 2025', () => {
    expect(formatTanggal('2025-01-01')).toBe('1 Jan 2025')
  })
})

describe('formatRelative', () => {
  const now = new Date('2026-06-22T12:00:00Z')

  it('30 detik lalu -> baru saja', () => {
    const d = new Date(now.getTime() - 30_000)
    expect(formatRelative(d, now)).toBe('baru saja')
  })
  it('2 menit lalu -> 2 menit lalu', () => {
    const d = new Date(now.getTime() - 2 * 60_000)
    expect(formatRelative(d, now)).toBe('2 menit lalu')
  })
  it('2 jam lalu -> 2 jam lalu', () => {
    const d = new Date(now.getTime() - 2 * 3600_000)
    expect(formatRelative(d, now)).toBe('2 jam lalu')
  })
  it('1 hari lalu -> kemarin', () => {
    const d = new Date(now.getTime() - 24 * 3600_000)
    expect(formatRelative(d, now)).toBe('kemarin')
  })
  it('3 hari lalu -> 3 hari lalu', () => {
    const d = new Date(now.getTime() - 3 * 24 * 3600_000)
    expect(formatRelative(d, now)).toBe('3 hari lalu')
  })
  it('8 hari lalu -> fallback tanggal', () => {
    const d = new Date(now.getTime() - 8 * 24 * 3600_000)
    const result = formatRelative(d, now)
    expect(result).not.toContain('hari lalu')
    expect(result).toMatch(/\d{1,2} \w+ \d{4}/)
  })
})
