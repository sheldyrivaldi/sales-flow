/**
 * Menentukan apakah sebuah berkas bisa DITAMPILKAN browser di tab baru, atau
 * hanya bisa diunduh.
 *
 * Dasarnya ekstensi, bukan MIME dari server: MIME sering kosong atau
 * generik (application/octet-stream) untuk berkas yang diunggah user,
 * sementara ekstensi selalu ada pada nama berkas.
 */

/** Format yang dirender sendiri oleh semua browser modern. */
const VIEWABLE = new Set([
  // Dokumen
  'pdf', 'txt', 'csv', 'log', 'md', 'json', 'xml',
  // Web
  'html', 'htm',
  // Gambar
  'png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'avif', 'bmp', 'ico',
  // Media
  'mp4', 'webm', 'ogg', 'mp3', 'wav', 'm4a', 'mov',
])

/** Ekstensi berkas dalam huruf kecil, tanpa titik. '' bila tidak ada. */
export function extOf(nameOrUrl: string): string {
  const clean = String(nameOrUrl ?? '').split(/[?#]/)[0]
  const base = clean.substring(clean.lastIndexOf('/') + 1)
  const dot = base.lastIndexOf('.')
  if (dot <= 0 || dot === base.length - 1) return ''
  return base.slice(dot + 1).toLowerCase()
}

/** true bila browser bisa menampilkannya langsung di tab baru. */
export function isViewableInBrowser(nameOrUrl: string, mime?: string): boolean {
  const ext = extOf(nameOrUrl)
  if (ext) return VIEWABLE.has(ext)
  // Tanpa ekstensi, MIME jadi petunjuk terakhir.
  const m = (mime ?? '').toLowerCase()
  return m.startsWith('image/') || m.startsWith('video/') || m.startsWith('audio/') || m.startsWith('text/') || m === 'application/pdf'
}

/** Label jenis berkas untuk ditampilkan (mis. "PDF", "XLSX"). */
export function kindLabel(nameOrUrl: string): string {
  const ext = extOf(nameOrUrl)
  return ext ? ext.toUpperCase() : 'FILE'
}

export function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/** Paksa unduh lewat atribut `download`, tanpa berpindah halaman. */
export function downloadFile(url: string, filename: string): void {
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
}
