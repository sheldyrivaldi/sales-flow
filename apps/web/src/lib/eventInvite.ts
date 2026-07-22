import type { Event } from '../api/events'
import { formatTanggal } from './format'

/**
 * Menyusun undangan event menjadi tautan `mailto:` yang siap dibuka aplikasi
 * email default (Outlook, Mail, Gmail) di Windows, Linux, Mac, maupun mobile.
 *
 * Kenapa mailto, bukan kirim dari server: undangan seperti ini harus terkirim
 * DARI alamat orang yang mengundang, muncul di folder Sent-nya, dan bisa
 * disunting sebelum dikirim.
 *
 * Catatan penting soal lampiran: `mailto:` TIDAK bisa membawa berkas (batasan
 * RFC 6068 di semua browser/OS). Karena itu berkas event disertakan sebagai
 * TAUTAN UNDUH di badan email — penerima mengklik untuk mengunduhnya.
 */

/** Batas aman panjang URL mailto. Di atas ini sebagian klien (khususnya
 * Outlook di Windows, yang memicu mailto lewat shell) memotong isinya diam-diam.
 * Undangan formal + tautan lampiran lebih panjang, jadi ambangnya dinaikkan
 * sedikit namun tetap konservatif. */
const MAILTO_SAFE_LEN = 2000

export interface InviteAttachment {
  name: string
  url: string
}

export interface InviteOptions {
  /** Nama pengundang untuk tanda tangan. */
  senderName?: string
  /** Nama perusahaan pengundang, tampil di bawah nama pengirim. */
  companyName?: string
  /** Label tipe kegiatan yang manusiawi (mis. "Pameran", "Konferensi"). */
  typeLabel?: string
  /** Basis URL (window.location.origin) untuk membangun tautan unduh absolut. */
  baseUrl?: string
  /** Berkas yang disertakan sebagai tautan unduh (lampiran event, dll). */
  attachments?: InviteAttachment[]
}

export interface InviteDraft {
  to: string[]
  subject: string
  body: string
}

/** Bangun URL absolut dari path lampiran ("/uploads/…") memakai baseUrl. */
function absoluteUrl(url: string, baseUrl?: string): string {
  if (/^https?:\/\//i.test(url)) return url
  if (!baseUrl) return url
  return baseUrl.replace(/\/+$/, '') + url
}

/** Satu kalimat konteks acara — nada rekan kerja, bukan vendor yang menyembah. */
function eventContextSentence(event: Event, typeLabel: string): string {
  const jenis = (typeLabel || 'kegiatan').toLowerCase()
  const penyelenggara =
    event.organizer && event.organizer.trim() ? ` yang digelar ${event.organizer.trim()}` : ''
  return (
    `Ini ${jenis}${penyelenggara} yang jadi peluang bagus untuk memperluas jejaring ` +
    `dan menjajaki kerja sama baru. Kehadiran Anda akan sangat berarti.`
  )
}

/**
 * Susun subjek + isi undangan.
 *
 * Nada: seorang atasan/rekan yang mengundang tim untuk ikut acara — percaya
 * diri, ringkas, hangat; BUKAN vendor yang merendah. Tanpa tanda hubung sebagai
 * penanda daftar. Struktur dibuat jelas dengan header bagian huruf kapital dan
 * daftar lampiran bernomor (URL di baris sendiri agar mudah diklik/disalin).
 */
export function buildInvite(event: Event, opts: InviteOptions = {}): InviteDraft {
  const { senderName, companyName, typeLabel = 'Kegiatan', baseUrl, attachments = [] } = opts
  const tanggal = event.date ? formatTanggal(event.date) : null
  const perusahaan = companyName?.trim()
  const lokasi = event.location?.trim()
  const penyelenggara = event.organizer?.trim()

  const lines: string[] = []
  lines.push('Halo Rekan-rekan,')
  lines.push('')

  // Pembuka langsung: siapa mengundang, acara apa, kapan, di mana.
  let pembuka = `Saya mengundang Anda untuk hadir di ${event.name}`
  if (tanggal) pembuka += ` pada ${tanggal}`
  if (lokasi) pembuka += `, di ${lokasi}`
  pembuka += '.'
  lines.push(pembuka)
  lines.push('')
  lines.push(eventContextSentence(event, typeLabel))
  lines.push('')

  // Rincian acara — header menonjol, "Label: value", tanpa tanda hubung.
  lines.push('DETAIL ACARA')
  lines.push(`Acara: ${event.name}`)
  lines.push(`Jenis: ${typeLabel}`)
  if (tanggal) lines.push(`Hari/Tanggal: ${tanggal}`)
  if (lokasi) lines.push(`Tempat: ${lokasi}`)
  if (penyelenggara) lines.push(`Penyelenggara: ${penyelenggara}`)
  lines.push('')

  // Catatan tim, bila ada.
  if (event.notes && event.notes.trim()) {
    lines.push('CATATAN')
    lines.push(event.notes.trim())
    lines.push('')
  }

  // Lampiran sebagai daftar bernomor + tautan unduh (mailto tak bisa bawa berkas).
  if (attachments.length > 0) {
    lines.push('MATERI PENDUKUNG')
    lines.push('Klik atau salin tautan berikut untuk mengunduh.')
    lines.push('')
    attachments.forEach((att, i) => {
      lines.push(`${i + 1}. ${att.name}`)
      lines.push(`   ${absoluteUrl(att.url, baseUrl)}`)
      if (i < attachments.length - 1) lines.push('')
    })
    lines.push('')
  }

  // Penutup hangat dan setara, bukan merendah.
  lines.push('Mohon konfirmasi kehadiran Anda lewat balasan email ini. Sampai jumpa di acara.')
  lines.push('')
  lines.push('Salam,')
  if (senderName?.trim()) lines.push(senderName.trim())
  if (perusahaan) lines.push(perusahaan)
  if (!senderName?.trim() && !perusahaan) lines.push(`Panitia ${event.name}`)

  return {
    to: event.participant_emails ?? [],
    subject: `Undangan: ${event.name}${tanggal ? ` — ${tanggal}` : ''}`,
    body: lines.join('\n'),
  }
}

/** Bangun URL mailto lengkap dari draft. */
export function mailtoUrl(draft: InviteDraft): string {
  const params = new URLSearchParams()
  params.set('subject', draft.subject)
  params.set('body', draft.body)
  // encodeURIComponent per alamat agar tanda '+' pada alamat tidak rusak;
  // URLSearchParams mengubah spasi jadi '+', yang salah untuk mailto.
  const qs = params.toString().replace(/\+/g, '%20')
  return `mailto:${draft.to.map(encodeURIComponent).join(',')}?${qs}`
}

/** true bila URL kemungkinan terlalu panjang untuk klien email tertentu. */
export function isMailtoTooLong(url: string): boolean {
  return url.length > MAILTO_SAFE_LEN
}
