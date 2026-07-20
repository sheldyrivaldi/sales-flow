import type { Event } from '../api/events'
import { formatTanggal } from './format'

/**
 * Menyusun undangan event menjadi tautan `mailto:` yang siap dibuka aplikasi
 * email default (Outlook di kebanyakan perusahaan).
 *
 * Kenapa mailto, bukan kirim dari server: undangan seperti ini harus terkirim
 * DARI alamat orang yang mengundang, muncul di folder Sent-nya, dan bisa
 * disunting sebelum dikirim. Mengirim lewat server justru membuatnya datang
 * dari alamat aplikasi dan gampang masuk spam.
 */

/** Batas aman panjang URL mailto. Di atas ini sebagian klien memotong isi. */
const MAILTO_SAFE_LEN = 1800

export interface InviteDraft {
  to: string[]
  subject: string
  body: string
}

/** Susun subjek + isi email yang rapi dan profesional dari data event. */
export function buildInvite(event: Event, senderName?: string): InviteDraft {
  const tanggal = event.date ? formatTanggal(event.date) : null

  const lines: string[] = []
  lines.push('Halo,')
  lines.push('')
  lines.push(
    tanggal
      ? `Dengan ini kami mengundang Anda untuk menghadiri ${event.name} pada ${tanggal}.`
      : `Dengan ini kami mengundang Anda untuk menghadiri ${event.name}.`,
  )
  lines.push('')
  lines.push('Detail acara:')
  lines.push(`- Acara   : ${event.name}`)
  if (tanggal) lines.push(`- Tanggal : ${tanggal}`)
  if (event.location) lines.push(`- Lokasi  : ${event.location}`)
  if (event.organizer) lines.push(`- Penyelenggara : ${event.organizer}`)
  lines.push('')
  if (event.notes?.trim()) {
    lines.push('Catatan:')
    lines.push(event.notes.trim())
    lines.push('')
  }
  lines.push('Mohon konfirmasi kehadiran Anda dengan membalas email ini.')
  lines.push('')
  lines.push('Terima kasih atas perhatian dan kerja samanya.')
  lines.push('')
  lines.push('Salam,')
  lines.push(senderName?.trim() || '')

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
