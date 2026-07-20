import { describe, it, expect } from 'vitest'
import { isValidEmail, splitEmails } from './email'

describe('isValidEmail', () => {
  it('menerima alamat yang wajar', () => {
    for (const e of ['a@b.co', 'nama.lengkap@perusahaan.co.id', 'user+tag@sub.domain.com']) {
      expect(isValidEmail(e)).toBe(true)
    }
  })

  it('menolak yang jelas salah', () => {
    for (const e of ['', 'tanpa-at', 'a@b', 'a@@b.com', 'a b@c.com', '@b.com', 'a@.com']) {
      expect(isValidEmail(e)).toBe(false)
    }
  })

  it('tidak peduli spasi pinggir dan huruf besar', () => {
    expect(isValidEmail('  Nama@Perusahaan.COM  ')).toBe(true)
  })

  it('menolak alamat yang masih mengandung pemisah daftar', () => {
    // Kalau ini lolos, satu chip bisa berisi dua alamat sekaligus.
    expect(isValidEmail('a@b.com,c@d.com')).toBe(false)
    expect(isValidEmail('a@b.com;c@d.com')).toBe(false)
  })
})

describe('splitEmails', () => {
  it('memecah berbagai pemisah yang biasa dipakai orang', () => {
    expect(splitEmails('a@b.com, c@d.com; e@f.com')).toEqual(['a@b.com', 'c@d.com', 'e@f.com'])
  })

  it('memecah tempelan multi-baris dari Excel/Outlook', () => {
    expect(splitEmails('a@b.com\nc@d.com\r\ne@f.com')).toEqual(['a@b.com', 'c@d.com', 'e@f.com'])
  })

  it('membuang bagian kosong dan menormalkan huruf', () => {
    expect(splitEmails(' ,,  A@B.com ;; ')).toEqual(['a@b.com'])
  })

  it('mengembalikan array kosong untuk teks kosong', () => {
    expect(splitEmails('   ')).toEqual([])
  })
})
