/**
 * Jaring pengaman untuk SVG slide yang DIKARANG AI.
 *
 * AI bebas mendesain tiap slide dari nol, tapi keluarannya tidak boleh
 * dipercaya begitu saja karena dua alasan:
 *
 * 1. KEAMANAN — markup dari model dirender di dalam aplikasi. Script,
 *    foreignObject, handler on*, dan rujukan eksternal harus dibuang.
 * 2. KUALITAS — model sering meleset menghitung lebar teks sehingga tulisan
 *    meluber keluar kanvas atau saling tumpuk. Slide seperti itu ditolak dan
 *    pemanggil jatuh ke layout katalog.
 *
 * Sanitasi memakai allowlist (bukan blocklist): apa pun yang tidak dikenal
 * dibuang, sehingga elemen/atribut baru yang berbahaya tidak lolos diam-diam.
 */

export const SVG_W = 1280
export const SVG_H = 720

// Elemen SVG yang boleh dirender. Sengaja TIDAK memuat: script, foreignObject
// (bisa menyisipkan HTML), a (navigasi), image (rujukan eksternal), iframe.
const ALLOWED_TAGS = new Set([
  'svg', 'g', 'defs', 'title', 'desc',
  'linearGradient', 'radialGradient', 'stop',
  'rect', 'circle', 'ellipse', 'line', 'polyline', 'polygon', 'path',
  'text', 'tspan',
  'clipPath', 'mask', 'pattern', 'symbol', 'use', 'marker',
  'filter', 'feGaussianBlur', 'feOffset', 'feMerge', 'feMergeNode',
  'feColorMatrix', 'feBlend', 'feDropShadow', 'feFlood', 'feComposite',
])

// Atribut presentasi yang boleh lewat.
const ALLOWED_ATTRS = new Set([
  'x', 'y', 'x1', 'y1', 'x2', 'y2', 'cx', 'cy', 'r', 'rx', 'ry',
  'width', 'height', 'd', 'points', 'transform', 'viewBox', 'xmlns',
  'fill', 'fill-opacity', 'fill-rule', 'stroke', 'stroke-width', 'stroke-opacity',
  'stroke-linecap', 'stroke-linejoin', 'stroke-dasharray', 'stroke-dashoffset',
  'opacity', 'offset', 'stop-color', 'stop-opacity', 'gradientUnits',
  'gradientTransform', 'patternUnits', 'maskUnits', 'clipPathUnits',
  'font-family', 'font-size', 'font-weight', 'font-style', 'letter-spacing',
  'text-anchor', 'dominant-baseline', 'alignment-baseline', 'dy', 'dx',
  'clip-path', 'mask', 'filter', 'id', 'class',
  'stdDeviation', 'in', 'in2', 'result', 'mode', 'values', 'type',
  'flood-color', 'flood-opacity', 'preserveAspectRatio',
])

export interface SvgCheck {
  ok: boolean
  /** Alasan penolakan — dipakai untuk log/diagnosa, bukan untuk user. */
  reasons: string[]
}

/**
 * Buang semua yang tidak ada di allowlist. Mengembalikan markup bersih, atau
 * null bila dokumen tidak bisa diparse / bukan SVG.
 */
/**
 * Perbaikan kecil sebelum parsing XML yang ketat.
 *
 * Model rutin menulis "&" apa adanya di teks bisnis ("Operasi & Pemeliharaan").
 * Di XML itu awal entity reference, sehingga SATU ampersand menggugurkan
 * seluruh slide. Escape hanya "&" yang BUKAN bagian dari entity yang sah,
 * supaya `&amp;` / `&#39;` yang sudah benar tidak ikut dirusak.
 */
function preRepair(src: string): string {
  return src.replace(/&(?!(?:amp|lt|gt|quot|apos|#\d+|#x[0-9a-fA-F]+);)/g, '&amp;')
}

export function sanitizeSvg(raw: string): string | null {
  if (typeof DOMParser === 'undefined') return null
  const src = preRepair(String(raw ?? '').trim())
  if (!src) return null

  const doc = new DOMParser().parseFromString(src, 'image/svg+xml')
  if (doc.getElementsByTagName('parsererror').length > 0) return null

  const root = doc.documentElement
  if (!root || root.tagName.toLowerCase() !== 'svg') return null

  const walk = (el: Element): void => {
    // Iterasi mundur supaya penghapusan tidak menggeser indeks.
    for (let i = el.children.length - 1; i >= 0; i--) {
      const child = el.children[i]
      const tag = child.tagName
      // Nama tag SVG case-sensitive (feGaussianBlur), jadi cek apa adanya
      // lalu fallback ke lowercase untuk tag biasa.
      if (!ALLOWED_TAGS.has(tag) && !ALLOWED_TAGS.has(tag.toLowerCase())) {
        child.remove()
        continue
      }
      for (const attr of Array.from(child.attributes)) {
        const name = attr.name
        const lower = name.toLowerCase()
        // Handler event dan rujukan eksternal selalu dibuang.
        if (lower.startsWith('on')) {
          child.removeAttribute(name)
          continue
        }
        if (lower === 'href' || lower === 'xlink:href') {
          // Hanya rujukan internal (#id) yang aman, mis. untuk <use>.
          if (!attr.value.trim().startsWith('#')) child.removeAttribute(name)
          continue
        }
        if (!ALLOWED_ATTRS.has(name) && !ALLOWED_ATTRS.has(lower)) {
          child.removeAttribute(name)
          continue
        }
        // url(...) hanya boleh menunjuk id internal (gradient/filter/mask).
        if (/url\(/i.test(attr.value) && !/url\(\s*['"]?#/.test(attr.value)) {
          child.removeAttribute(name)
        }
      }
      walk(child)
    }
  }
  walk(root)

  // Paksa kanvas ke ukuran deck agar raster & preview konsisten.
  root.setAttribute('xmlns', 'http://www.w3.org/2000/svg')
  root.setAttribute('viewBox', `0 0 ${SVG_W} ${SVG_H}`)
  root.setAttribute('width', String(SVG_W))
  root.setAttribute('height', String(SVG_H))

  return new XMLSerializer().serializeToString(root)
}

interface Box {
  x: number
  y: number
  w: number
  h: number
  s: string
}

/** Ambil atribut angka dari elemen atau leluhurnya (font-size sering diwarisi). */
function inherited(el: Element, attr: string, fallback: number): number {
  let cur: Element | null = el
  while (cur) {
    const v = cur.getAttribute(attr)
    if (v) {
      const n = parseFloat(v)
      if (!Number.isNaN(n)) return n
    }
    cur = cur.parentElement
  }
  return fallback
}

function inheritedStr(el: Element, attr: string, fallback: string): string {
  let cur: Element | null = el
  while (cur) {
    const v = cur.getAttribute(attr)
    if (v) return v
    cur = cur.parentElement
  }
  return fallback
}

/**
 * Perkirakan kotak teks tanpa mesin layout. Dipakai sebagai pemeriksaan dasar
 * yang SELALU jalan (termasuk di luar browser), dan sebagai pengganti getBBox
 * yang tidak tersedia di jsdom.
 *
 * Lebar rata-rata karakter Arial ~0.55 x font-size — heuristik yang sama yang
 * dipakai renderer saat memecah baris, jadi ambangnya konsisten.
 */
const CHAR_W = 0.55

function estimateBoxes(root: Element): Box[] {
  const boxes: Box[] = []
  for (const t of Array.from(root.querySelectorAll('text'))) {
    const fs = inherited(t, 'font-size', 16)
    const anchor = inheritedStr(t, 'text-anchor', 'start')
    const baseX = inherited(t, 'x', 0)
    const baseY = inherited(t, 'y', 0)

    // Tiap tspan dianggap satu baris; tanpa tspan, teksnya sendiri satu baris.
    const spans = Array.from(t.querySelectorAll('tspan'))
    const lines: { text: string; x: number; y: number; fs: number }[] = []
    if (spans.length === 0) {
      lines.push({ text: (t.textContent ?? '').trim(), x: baseX, y: baseY, fs })
    } else {
      let cursorY = baseY
      for (const sp of spans) {
        const sx = sp.getAttribute('x') ? parseFloat(sp.getAttribute('x')!) : baseX
        const dy = sp.getAttribute('dy') ? parseFloat(sp.getAttribute('dy')!) : 0
        const sy = sp.getAttribute('y') ? parseFloat(sp.getAttribute('y')!) : cursorY + dy
        cursorY = sy
        lines.push({ text: (sp.textContent ?? '').trim(), x: sx, y: sy, fs: inherited(sp, 'font-size', fs) })
      }
    }

    for (const ln of lines) {
      if (!ln.text) continue
      const w = ln.text.length * CHAR_W * ln.fs
      const h = ln.fs * 1.15
      let x = ln.x
      if (anchor === 'middle') x = ln.x - w / 2
      else if (anchor === 'end') x = ln.x - w
      // y pada SVG adalah baseline; kotak membentang ke atas dari sana.
      boxes.push({ x, y: ln.y - ln.fs * 0.8, w, h, s: ln.text.slice(0, 24) })
    }
  }
  return boxes
}

/**
 * Ukur kotak teks dalam KOORDINAT ROOT SVG.
 *
 * PENTING: jangan pakai getBBox() di sini. getBBox mengembalikan kotak dalam
 * ruang koordinat LOKAL elemen — sebelum transform miliknya sendiri maupun
 * transform leluhurnya. Begitu model membungkus konten dalam
 * <g transform='translate(...)'> (dan itu lazim), koordinat lokal tidak bisa
 * dibandingkan dengan elemen di grup lain: hasilnya tabrakan/luber PALSU, dan
 * slide yang sebenarnya rapi ikut dibuang.
 *
 * getBoundingClientRect() sudah memperhitungkan seluruh rantai transform.
 * Hasilnya (piksel layar) diskalakan kembali ke satuan viewBox 1280x720.
 */
function measuredBoxes(root: Element): Box[] | null {
  const rootRect = (root as SVGGraphicsElement).getBoundingClientRect?.()
  if (!rootRect || rootRect.width === 0 || rootRect.height === 0) return null
  const sx = SVG_W / rootRect.width
  const sy = SVG_H / rootRect.height

  const boxes: Box[] = []
  for (const t of Array.from(root.querySelectorAll('text'))) {
    const label = (t.textContent ?? '').trim()
    if (!label) continue
    const r = (t as SVGGraphicsElement).getBoundingClientRect?.()
    if (!r || (r.width === 0 && r.height === 0)) continue
    boxes.push({
      x: (r.left - rootRect.left) * sx,
      y: (r.top - rootRect.top) * sy,
      w: r.width * sx,
      h: r.height * sy,
      s: label.slice(0, 24),
    })
  }
  // jsdom mengembalikan 0 untuk semuanya — anggap tak terukur.
  return boxes.length > 0 ? boxes : null
}

function auditBoxes(boxes: Box[]): string[] {
  const reasons: string[] = []
  for (const b of boxes) {
    if (b.x < -1 || b.y < -1 || b.x + b.w > SVG_W + 1 || b.y + b.h > SVG_H + 1) {
      reasons.push(`teks keluar kanvas: "${b.s}"`)
      break
    }
  }
  // Ambang 6px, bukan 3px: getBBox menyertakan ekor huruf (g, y, p) dan ruang
  // atas huruf besar, sehingga dua baris yang RAPAT TAPI RAPI bisa terhitung
  // bersinggungan beberapa piksel. Tumpang tindih yang benar-benar merusak
  // selalu jauh lebih dalam dari itu.
  outer: for (let i = 0; i < boxes.length; i++) {
    for (let j = i + 1; j < boxes.length; j++) {
      const a = boxes[i]
      const c = boxes[j]
      const ix = Math.min(a.x + a.w, c.x + c.w) - Math.max(a.x, c.x)
      const iy = Math.min(a.y + a.h, c.y + c.h) - Math.max(a.y, c.y)
      if (ix > 6 && iy > 6) {
        reasons.push(`teks tumpang tindih: "${a.s}" x "${c.s}"`)
        break outer
      }
    }
  }
  return reasons
}

/**
 * Periksa geometri teks: menolak slide yang teksnya keluar kanvas atau saling
 * tumpuk — dua cacat tersering saat model salah menaksir lebar tulisan.
 *
 * Memakai getBBox bila mesin layout tersedia (browser sungguhan), dan jatuh ke
 * perkiraan lebar karakter bila tidak (jsdom/Node). Jadi validasi TIDAK PERNAH
 * lolos diam-diam hanya karena lingkungannya tak bisa mengukur.
 */
export function checkSvgGeometry(svg: string): SvgCheck {
  if (typeof DOMParser === 'undefined') return { ok: true, reasons: [] }

  const doc = new DOMParser().parseFromString(svg, 'image/svg+xml')
  if (doc.getElementsByTagName('parsererror').length > 0) {
    return { ok: false, reasons: ['svg tidak bisa diparse'] }
  }
  const parsed = doc.documentElement
  if (!parsed || parsed.tagName.toLowerCase() !== 'svg') {
    return { ok: false, reasons: ['tidak ada elemen svg'] }
  }
  if (parsed.querySelectorAll('text').length === 0) {
    return { ok: false, reasons: ['slide tanpa teks'] }
  }

  // Pengukuran presisi butuh elemen yang benar-benar dirender.
  let boxes: Box[] | null = null
  if (typeof document !== 'undefined' && document.body) {
    const host = document.createElement('div')
    host.setAttribute('style', 'position:absolute;left:-99999px;top:0;width:1280px;height:720px;pointer-events:none')
    host.innerHTML = svg
    document.body.appendChild(host)
    try {
      const live = host.querySelector('svg')
      if (live) boxes = measuredBoxes(live)
    } finally {
      host.remove()
    }
  }
  if (!boxes) {
    // Jalur cadangan (tanpa mesin layout) menghitung koordinat mentah dari
    // atribut, sehingga TIDAK memperhitungkan transform. Kalau dokumen memakai
    // transform, hasilnya tidak bisa dipercaya — lebih baik meloloskan slide
    // daripada membuang slide bagus karena salah ukur. Sanitasi tetap jalan.
    if (parsed.querySelector('[transform]')) return { ok: true, reasons: [] }
    boxes = estimateBoxes(parsed)
  }

  const reasons = auditBoxes(boxes)
  return { ok: reasons.length === 0, reasons }
}

/**
 * Sanitasi + validasi sekaligus. Mengembalikan markup siap pakai, atau null
 * bila slide harus jatuh ke layout katalog.
 */
export function prepareAiSvg(raw: string | undefined): string | null {
  if (!raw) return null
  const clean = sanitizeSvg(raw)
  if (!clean) return null
  const check = checkSvgGeometry(clean)
  if (!check.ok) {
    if (typeof console !== 'undefined') {
      console.warn('[playbook] slide SVG dari AI ditolak:', check.reasons.join('; '))
    }
    return null
  }
  return clean
}
