import { describe, it, expect } from 'vitest'
import { buildQueryString } from './api'

describe('buildQueryString', () => {
  it('menghasilkan parameter BERULANG untuk array, bukan gabungan koma', () => {
    // Sisi Go membaca c.QueryParams()["type"] → []string. Kalau digabung jadi
    // "EXPO,SEMINAR", nilainya dianggap satu dan tidak pernah cocok.
    const qs = buildQueryString({ type: ['EXPO', 'SEMINAR'] })
    expect(qs).toBe('?type=EXPO&type=SEMINAR')
  })

  it('melewati array kosong sepenuhnya', () => {
    expect(buildQueryString({ type: [], search: 'a' })).toBe('?search=a')
  })

  it('membuang nilai kosong di dalam array', () => {
    expect(buildQueryString({ status: ['PLANNED', ''] })).toBe('?status=PLANNED')
  })

  it('tetap menangani nilai tunggal seperti sebelumnya', () => {
    const qs = buildQueryString({ search: 'expo jakarta', page: 2, has_attachment: true })
    expect(qs).toContain('search=expo+jakarta')
    expect(qs).toContain('page=2')
    expect(qs).toContain('has_attachment=true')
  })

  it('membuang undefined, null, dan string kosong', () => {
    expect(buildQueryString({ a: undefined, b: null, c: '', d: 'ok' })).toBe('?d=ok')
  })

  it('mengembalikan string kosong bila tidak ada parameter', () => {
    expect(buildQueryString({})).toBe('')
    expect(buildQueryString({ a: undefined })).toBe('')
  })

  it('menggabungkan filter multi-kolom seperti pemakaian nyata', () => {
    const qs = buildQueryString({
      type: ['EXPO', 'CONFERENCE'],
      status: ['PLANNED'],
      search: 'security',
      has_attachment: true,
      page: 1,
    })
    expect(qs).toContain('type=EXPO&type=CONFERENCE')
    expect(qs).toContain('status=PLANNED')
    expect(qs).toContain('search=security')
  })
})
