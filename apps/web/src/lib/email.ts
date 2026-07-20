/** Validasi bentuk alamat email, BUKAN keberadaan akun — peserta event memang
 * tidak perlu terdaftar di aplikasi ini. Pola sengaja konservatif: satu @,
 * domain bertitik, tanpa spasi maupun pemisah daftar. */
const EMAIL_RE = /^[^\s@,;]+@[^\s@,;]+\.[^\s@,;]{2,}$/

export function isValidEmail(v: string): boolean {
  return EMAIL_RE.test(v.trim().toLowerCase())
}

/** Pecah teks tempelan menjadi calon email. Menerima pemisah koma, titik koma,
 * spasi, dan baris baru sekaligus — daftar undangan biasanya disalin dari
 * Excel atau Outlook dalam bentuk yang tidak seragam. */
export function splitEmails(raw: string): string[] {
  return raw
    .split(/[\s,;]+/)
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean)
}
