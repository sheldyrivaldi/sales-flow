import { describe, it, expect } from 'vitest'
import { extOf, isViewableInBrowser, kindLabel, formatBytes } from './fileKind'

describe('extOf', () => {
  it('mengambil ekstensi dan menormalkan huruf', () => {
    expect(extOf('Rundown.PDF')).toBe('pdf')
    expect(extOf('/uploads/event/abc-123.xlsx')).toBe('xlsx')
  })

  it('mengabaikan query string dan hash pada URL', () => {
    expect(extOf('/uploads/a.pdf?v=2')).toBe('pdf')
    expect(extOf('/uploads/a.png#page=1')).toBe('png')
  })

  it('mengembalikan kosong bila tidak ada ekstensi', () => {
    expect(extOf('README')).toBe('')
    expect(extOf('arsip.')).toBe('')
    expect(extOf('.gitignore')).toBe('') // titik di awal bukan ekstensi
  })
})

describe('isViewableInBrowser', () => {
  it('mengizinkan format yang memang dirender browser', () => {
    for (const n of ['a.pdf', 'a.html', 'a.htm', 'a.png', 'a.jpg', 'a.svg', 'a.txt', 'a.csv', 'a.json', 'a.mp4', 'a.mp3']) {
      expect(isViewableInBrowser(n)).toBe(true)
    }
  })

  it('menolak format yang hanya bisa diunduh', () => {
    // Inilah yang memicu konfirmasi "unduh atau batal".
    for (const n of ['a.docx', 'a.xlsx', 'a.pptx', 'a.zip', 'a.rar', 'a.exe', 'a.dwg']) {
      expect(isViewableInBrowser(n)).toBe(false)
    }
  })

  it('jatuh ke MIME saat nama berkas tanpa ekstensi', () => {
    expect(isViewableInBrowser('scan', 'image/png')).toBe(true)
    expect(isViewableInBrowser('dokumen', 'application/pdf')).toBe(true)
    expect(isViewableInBrowser('arsip', 'application/octet-stream')).toBe(false)
    expect(isViewableInBrowser('tanpa-apa-apa')).toBe(false)
  })

  it('ekstensi menang atas MIME yang menyesatkan', () => {
    // Server kerap mengirim octet-stream untuk unggahan; ekstensi lebih andal.
    expect(isViewableInBrowser('rundown.pdf', 'application/octet-stream')).toBe(true)
  })
})

describe('kindLabel & formatBytes', () => {
  it('memberi label jenis yang terbaca', () => {
    expect(kindLabel('a.xlsx')).toBe('XLSX')
    expect(kindLabel('tanpaekstensi')).toBe('FILE')
  })

  it('memformat ukuran dengan satuan yang wajar', () => {
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(2048)).toBe('2 KB')
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB')
    expect(formatBytes(0)).toBe('')
    expect(formatBytes(undefined)).toBe('')
  })
})
