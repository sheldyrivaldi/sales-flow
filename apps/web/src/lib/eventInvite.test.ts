import { describe, it, expect } from 'vitest'
import { buildInvite, mailtoUrl, isMailtoTooLong } from './eventInvite'
import type { Event } from '../api/events'

function makeEvent(over: Partial<Event> = {}): Event {
  return {
    id: 'e1',
    name: 'Indo Security Expo 2026',
    type: 'EXPO',
    date: '2026-07-20T00:00:00Z',
    location: 'JCC Jakarta',
    organizer: 'Dyandra',
    notes: null,
    status: 'PLANNED',
    analysis_status: 'idle',
    participant_emails: ['a@contoh.com', 'b@contoh.com'],
    attachments: [],
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
    ...over,
  }
}

describe('buildInvite', () => {
  it('menyusun subjek yang menyebut acara dan tanggal', () => {
    const inv = buildInvite(makeEvent())
    expect(inv.subject).toContain('Undangan: Indo Security Expo 2026')
    expect(inv.subject).toMatch(/2026/)
  })

  it('memuat detail acara yang penting di badan email', () => {
    const body = buildInvite(makeEvent()).body
    expect(body).toContain('Indo Security Expo 2026')
    expect(body).toContain('JCC Jakarta')
    expect(body).toContain('Dyandra')
    expect(body).toMatch(/konfirmasi kehadiran/i)
  })

  it('memakai nada rekan/atasan, bukan vendor yang merendah', () => {
    const body = buildInvite(makeEvent()).body
    expect(body).toContain('Halo Rekan-rekan,')
    expect(body).not.toContain('Hormat kami')
    expect(body).not.toContain('perkenankan')
  })

  it('tidak memakai tanda hubung sebagai penanda daftar (tell AI)', () => {
    const body = buildInvite(makeEvent({ notes: 'Bawa kartu nama.' })).body
    expect(body).not.toMatch(/(^|\n)\s*-\s/)
  })

  it('menyertakan seluruh penerima', () => {
    expect(buildInvite(makeEvent()).to).toEqual(['a@contoh.com', 'b@contoh.com'])
  })

  it('tidak menampilkan baris tanggal saat event belum bertanggal', () => {
    const body = buildInvite(makeEvent({ date: null })).body
    expect(body).not.toContain('Hari/Tanggal')
    expect(buildInvite(makeEvent({ date: null })).subject).toBe('Undangan: Indo Security Expo 2026')
  })

  it('menyisipkan catatan tim bila ada', () => {
    const body = buildInvite(makeEvent({ notes: 'Bawa kartu nama.' })).body
    expect(body).toContain('CATATAN')
    expect(body).toContain('Bawa kartu nama.')
  })

  it('memakai nama pengirim dan perusahaan pada tanda tangan', () => {
    const body = buildInvite(makeEvent(), { senderName: 'Sheldy', companyName: 'PT Moonlay Technologies' }).body
    const tail = body.trimEnd()
    expect(tail.endsWith('PT Moonlay Technologies')).toBe(true)
    expect(tail).toContain('Sheldy')
  })

  it('menyertakan lampiran sebagai daftar bernomor + tautan unduh absolut', () => {
    const body = buildInvite(makeEvent(), {
      baseUrl: 'https://app.contoh.com',
      attachments: [
        { name: 'Rundown.pdf', url: '/uploads/event/abc.pdf' },
        { name: 'Denah.png', url: '/uploads/event/def.png' },
      ],
    }).body
    expect(body).toContain('MATERI PENDUKUNG')
    expect(body).toContain('1. Rundown.pdf')
    expect(body).toContain('https://app.contoh.com/uploads/event/abc.pdf')
    expect(body).toContain('2. Denah.png')
  })
})

describe('mailtoUrl', () => {
  it('menempatkan penerima setelah mailto: dan dipisah koma', () => {
    const url = mailtoUrl(buildInvite(makeEvent()))
    expect(url.startsWith('mailto:a%40contoh.com,b%40contoh.com?')).toBe(true)
  })

  it('meng-encode subjek dan body, memakai %20 bukan tanda plus', () => {
    // Klien email memperlakukan '+' sebagai karakter literal, bukan spasi —
    // memakai bentuk URLSearchParams apa adanya membuat isi email penuh '+'.
    const url = mailtoUrl(buildInvite(makeEvent()))
    const qs = url.slice(url.indexOf('?') + 1)
    expect(qs).toContain('%20')
    expect(qs).not.toContain('+')
  })

  it('menyertakan subject dan body sekaligus', () => {
    const url = mailtoUrl(buildInvite(makeEvent()))
    expect(url).toContain('subject=')
    expect(url).toContain('body=')
  })
})

describe('isMailtoTooLong', () => {
  it('menerima undangan berukuran wajar', () => {
    expect(isMailtoTooLong(mailtoUrl(buildInvite(makeEvent())))).toBe(false)
  })

  it('menandai daftar penerima yang sangat panjang', () => {
    const many = Array.from({ length: 120 }, (_, i) => `peserta${i}@perusahaan-contoh.co.id`)
    const url = mailtoUrl(buildInvite(makeEvent({ participant_emails: many })))
    expect(isMailtoTooLong(url)).toBe(true)
  })
})
