import type { PlaybookContent, DeckSlideSpec, DeckLayout } from '../api/playbooks'
import { prepareAiSvg } from './playbookSvg'

/**
 * Mesin deck playbook. Sumber tunggal desain: tiap slide dihasilkan sebagai
 * SVG 1280×720 penuh, dipakai bersama oleh viewer in-app (render inline) dan
 * eksportir .pptx (raster ke PNG) — jadi preview & file identik.
 *
 * Dua tingkat desain, dari yang paling bebas:
 *
 * 1. SVG KARANGAN AI (`spec.svg`) — AI mendesain slide dari nol mengikuti
 *    topik: komposisi, palet, ilustrasi vektor. Inilah sumber variasi
 *    sesungguhnya; tanpa ini deck akan selalu terasa template karena hanya
 *    memilih dari katalog yang jumlahnya terbatas.
 * 2. KATALOG LAYOUT (LAYOUTS) — jaring pengaman. Dipakai bila SVG karangan AI
 *    gagal sanitasi/validasi, atau bila AI memang tidak mengirimnya.
 *
 * Playbook lama (tanpa `deck`) tetap tampil lewat konversi field datar.
 */

export const VW = 1280
export const VH = 720

export interface DeckSlide {
  key: string
  /** Judul untuk kontrol/label thumbnail. */
  title: string
  /** Seksi untuk aksi "edit slide ini". */
  section: string
  /** Markup <svg> penuh 1280×720. */
  svg: string
  /** true bila SVG-nya karangan AI (lolos validasi), false bila layout katalog. */
  aiDesigned: boolean
  /** Jenis transisi PowerPoint untuk slide ini. */
  transition: 'fade' | 'push' | 'wipe' | 'split' | 'zoom'
}

// ── Tema per topik ───────────────────────────────────────────────────────────
interface Theme {
  key: string
  deep: string
  dark: string
  base: string
  bright: string
  accent2: string
  tint: string
  line: string
  /** Gaya motif dekoratif — beda tema, beda karakter visual. */
  motif: 'arcs' | 'grid' | 'waves' | 'hex' | 'beams'
}

const THEMES: Record<string, Theme> = {
  emerald: { key: 'emerald', deep: '#04231b', dark: '#065f46', base: '#059669', bright: '#10b981', accent2: '#14b8a6', tint: '#ecfdf5', line: '#a7f3d0', motif: 'arcs' },
  teal: { key: 'teal', deep: '#042f2e', dark: '#0f766e', base: '#0d9488', bright: '#14b8a6', accent2: '#06b6d4', tint: '#f0fdfa', line: '#99f6e4', motif: 'waves' },
  indigo: { key: 'indigo', deep: '#1e1b4b', dark: '#3730a3', base: '#4f46e5', bright: '#6366f1', accent2: '#8b5cf6', tint: '#eef2ff', line: '#c7d2fe', motif: 'grid' },
  blue: { key: 'blue', deep: '#0c243b', dark: '#1e40af', base: '#2563eb', bright: '#3b82f6', accent2: '#06b6d4', tint: '#eff6ff', line: '#bfdbfe', motif: 'beams' },
  violet: { key: 'violet', deep: '#2e1065', dark: '#6d28d9', base: '#7c3aed', bright: '#8b5cf6', accent2: '#d946ef', tint: '#f5f3ff', line: '#ddd6fe', motif: 'hex' },
  cyan: { key: 'cyan', deep: '#083344', dark: '#0e7490', base: '#0891b2', bright: '#06b6d4', accent2: '#14b8a6', tint: '#ecfeff', line: '#a5f3fc', motif: 'waves' },
  amber: { key: 'amber', deep: '#451a03', dark: '#b45309', base: '#d97706', bright: '#f59e0b', accent2: '#ea580c', tint: '#fffbeb', line: '#fde68a', motif: 'beams' },
  rose: { key: 'rose', deep: '#4c0519', dark: '#be123c', base: '#e11d48', bright: '#f43f5e', accent2: '#ec4899', tint: '#fff1f2', line: '#fecdd3', motif: 'arcs' },
  slate: { key: 'slate', deep: '#0f172a', dark: '#334155', base: '#475569', bright: '#64748b', accent2: '#0ea5e9', tint: '#f8fafc', line: '#e2e8f0', motif: 'grid' },
}

const INK = '#0f172a'
const SUB = '#475569'
const MUT = '#94a3b8'
const PAPER = '#ffffff'
const PAPER2 = '#f8fafc'
const DANGER = '#e11d48'
const DANGER_BG = '#fff1f2'
const DANGER_INK = '#9f1239'

const ACCENT_KEYWORDS: [RegExp, string][] = [
  [/cyber|siber|security|keamanan|pentest|soc\b|threat|iso 27001/i, 'indigo'],
  [/bank|financ|keuangan|fintech|pajak|tax|akuntansi|invest|asuransi|insurance|perbankan/i, 'blue'],
  [/health|kesehatan|medis|medical|rumah sakit|hospital|farmasi|pharma|klinik/i, 'teal'],
  [/energy|energi|oil|gas|migas|tambang|mining|utilit|listrik|power|ebt/i, 'amber'],
  [/edu|pendidikan|kampus|universit|sekolah|training|akademi/i, 'violet'],
  [/retail|consumer|fmcg|ritel|ecommerce|marketplace|brand/i, 'rose'],
  [/logist|logistik|supply chain|transport|shipping|warehouse|gudang|fleet/i, 'cyan'],
  [/gov|pemerintah|kementerian|bumn|public sector|dinas|pemda/i, 'blue'],
  [/manufaktur|manufacturing|pabrik|industri|produksi|factory|smart factory/i, 'slate'],
  [/tech|digital|software|saas|\bai\b|data|cloud|platform|aplikasi|sistem/i, 'indigo'],
]

function pickTheme(content: PlaybookContent): Theme {
  const hint = (content.accent || '').trim().toLowerCase()
  if (hint && THEMES[hint]) return THEMES[hint]
  const hay = `${content.title ?? ''} ${content.subtitle ?? ''} ${content.summary ?? ''} ${content.value_prop ?? ''}`
  for (const [re, key] of ACCENT_KEYWORDS) if (re.test(hay)) return THEMES[key]
  return THEMES.emerald
}

// ── Util SVG ─────────────────────────────────────────────────────────────────
function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function fitLines(text: unknown, widthPx: number, fontSize: number, maxLines: number): string[] {
  const clean = String(text ?? '').replace(/\s+/g, ' ').trim()
  if (!clean) return []
  const cpl = Math.max(6, Math.floor(widthPx / (fontSize * 0.53)))
  const words = clean.split(' ')
  const lines: string[] = []
  let cur = ''
  for (const w of words) {
    const cand = cur ? `${cur} ${w}` : w
    if (cand.length <= cpl) {
      cur = cand
      continue
    }
    if (cur) lines.push(cur)
    if (lines.length >= maxLines) break
    if (w.length > cpl) {
      let rest = w
      while (rest.length > cpl && lines.length < maxLines) {
        lines.push(rest.slice(0, cpl - 1) + '-')
        rest = rest.slice(cpl - 1)
      }
      cur = rest
    } else {
      cur = w
    }
  }
  if (cur && lines.length < maxLines) lines.push(cur)
  if (lines.length > maxLines) lines.length = maxLines
  const consumed = lines.join(' ').replace(/-\s/g, '').length
  if (consumed < clean.length - 1 && lines.length) {
    const last = lines[lines.length - 1]
    lines[lines.length - 1] = last.replace(/[\s,;.]+$/, '') + '…'
  }
  return lines
}

interface TextOpts {
  size: number
  color: string
  weight?: number | 'normal' | 'bold'
  anchor?: 'start' | 'middle' | 'end'
  lineH?: number
  spacing?: number
  opacity?: number
  italic?: boolean
}

function textBlock(lines: string[], x: number, y: number, o: TextOpts): string {
  if (!lines.length) return ''
  const lh = o.lineH ?? o.size * 1.32
  const ls = o.spacing ? ` letter-spacing="${o.spacing}"` : ''
  const op = o.opacity != null ? ` opacity="${o.opacity}"` : ''
  const it = o.italic ? ` font-style="italic"` : ''
  const tspans = lines.map((ln, i) => `<tspan x="${x}" dy="${i === 0 ? 0 : lh}">${esc(ln)}</tspan>`).join('')
  return `<text x="${x}" y="${y}" font-size="${o.size}" font-weight="${o.weight ?? 'normal'}" fill="${o.color}" text-anchor="${o.anchor ?? 'start'}"${ls}${op}${it}>${tspans}</text>`
}

function line1(text: unknown, x: number, y: number, o: TextOpts): string {
  return textBlock([String(text ?? '')], x, y, o)
}

/** Teks multi-baris yang di-center vertikal pada titik cy. */
function centeredBlock(lines: string[], x: number, cy: number, o: TextOpts): string {
  const lh = o.lineH ?? o.size * 1.32
  const y = cy - ((lines.length - 1) * lh) / 2 + o.size * 0.35
  return textBlock(lines, x, y, o)
}

function checkIcon(cx: number, cy: number, r: number, color: string): string {
  const s = r * 0.95
  return `<path d="M ${cx - s * 0.55} ${cy} L ${cx - s * 0.1} ${cy + s * 0.45} L ${cx + s * 0.6} ${cy - s * 0.5}" fill="none" stroke="${color}" stroke-width="${Math.max(2, r * 0.34)}" stroke-linecap="round" stroke-linejoin="round"/>`
}
function arrowIcon(cx: number, cy: number, r: number, color: string): string {
  return `<path d="M ${cx - r * 0.6} ${cy} L ${cx + r * 0.35} ${cy} M ${cx - r * 0.05} ${cy - r * 0.45} L ${cx + r * 0.5} ${cy} L ${cx - r * 0.05} ${cy + r * 0.45}" fill="none" stroke="${color}" stroke-width="${Math.max(2, r * 0.3)}" stroke-linecap="round" stroke-linejoin="round"/>`
}
function warnIcon(cx: number, cy: number, r: number, color: string): string {
  return `<path d="M ${cx} ${cy - r} L ${cx + r * 0.95} ${cy + r * 0.72} L ${cx - r * 0.95} ${cy + r * 0.72} Z" fill="${color}"/>`
}

const MARGIN = 72
const DATE_ID = new Date().toLocaleDateString('id-ID', { day: '2-digit', month: 'long', year: 'numeric' })

function svgOpen(): string {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${VW}" height="${VH}" viewBox="0 0 ${VW} ${VH}" font-family="Arial, Helvetica, sans-serif">`
}

function defs(t: Theme): string {
  return (
    `<defs>` +
    `<linearGradient id="g1" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${t.deep}"/><stop offset="0.62" stop-color="${t.dark}"/><stop offset="1" stop-color="${t.base}"/></linearGradient>` +
    `<linearGradient id="g2" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="${PAPER}"/><stop offset="1" stop-color="${PAPER2}"/></linearGradient>` +
    `<linearGradient id="g3" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${t.base}"/><stop offset="1" stop-color="${t.dark}"/></linearGradient>` +
    `<linearGradient id="g4" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="${t.bright}"/><stop offset="1" stop-color="${t.accent2}"/></linearGradient>` +
    `</defs>`
  )
}

/** Motif dekoratif — karakternya berbeda per tema. */
function motif(t: Theme, dark: boolean): string {
  const c = dark ? t.bright : t.base
  const o = dark ? 1 : 0.5
  let s = ''
  switch (t.motif) {
    case 'arcs':
      for (let i = 0; i < 5; i++)
        s += `<circle cx="${VW - 40}" cy="${VH + 30}" r="${150 + i * 95}" fill="none" stroke="${c}" stroke-width="2" opacity="${(0.18 - i * 0.025) * o}"/>`
      s += `<circle cx="${VW - 170}" cy="150" r="16" fill="${t.accent2}" opacity="${0.85 * o}"/>`
      break
    case 'grid':
      for (let gx = 0; gx < 9; gx++)
        for (let gy = 0; gy < 6; gy++)
          s += `<circle cx="${VW - 330 + gx * 38}" cy="${60 + gy * 38}" r="2.8" fill="${c}" opacity="${0.45 * o}"/>`
      s += `<rect x="${VW - 250}" y="${VH - 190}" width="180" height="180" fill="none" stroke="${c}" stroke-width="2" opacity="${0.3 * o}" transform="rotate(15 ${VW - 160} ${VH - 100})"/>`
      break
    case 'waves':
      for (let i = 0; i < 4; i++)
        s += `<path d="M ${VW - 520} ${140 + i * 70} Q ${VW - 380} ${80 + i * 70} ${VW - 240} ${140 + i * 70} T ${VW + 40} ${140 + i * 70}" fill="none" stroke="${c}" stroke-width="2.5" opacity="${(0.3 - i * 0.05) * o}"/>`
      break
    case 'hex': {
      const hex = (cx: number, cy: number, r: number, op: number) => {
        const pts = Array.from({ length: 6 }, (_, i) => {
          const a = (Math.PI / 3) * i - Math.PI / 6
          return `${(cx + r * Math.cos(a)).toFixed(1)},${(cy + r * Math.sin(a)).toFixed(1)}`
        }).join(' ')
        return `<polygon points="${pts}" fill="none" stroke="${c}" stroke-width="2" opacity="${op * o}"/>`
      }
      s += hex(VW - 150, 130, 70, 0.35) + hex(VW - 260, 200, 46, 0.25) + hex(VW - 110, 265, 36, 0.2)
      s += hex(VW - 190, VH - 110, 90, 0.18)
      break
    }
    case 'beams':
      for (let i = 0; i < 6; i++)
        s += `<rect x="${VW - 420 + i * 68}" y="-60" width="16" height="360" rx="8" fill="${c}" opacity="${(0.22 - i * 0.028) * o}" transform="rotate(22 ${VW - 300} 120)"/>`
      break
  }
  return s
}

function lightBase(t: Theme): string {
  return (
    `<rect width="${VW}" height="${VH}" fill="url(#g2)"/>` +
    `<path d="M ${VW} 0 L ${VW} 200 Q ${VW - 160} 78 ${VW - 350} 0 Z" fill="${t.tint}"/>` +
    `<rect x="0" y="0" width="10" height="${VH}" fill="${t.base}"/>`
  )
}

function header(t: Theme, num: number, eyebrow: string, heading: string): string {
  return (
    `<rect x="${MARGIN}" y="56" width="52" height="52" rx="13" fill="${t.base}"/>` +
    line1(String(num).padStart(2, '0'), MARGIN + 26, 90, { size: 22, color: PAPER, weight: 'bold', anchor: 'middle' }) +
    line1(eyebrow.toUpperCase(), MARGIN + 70, 76, { size: 13, color: t.base, weight: 'bold', spacing: 3 }) +
    textBlock(fitLines(heading, VW - MARGIN * 2 - 90, 30, 1), MARGIN + 70, 105, { size: 30, color: INK, weight: 'bold' }) +
    `<rect x="${MARGIN + 70}" y="119" width="64" height="5" rx="2.5" fill="${t.bright}"/>`
  )
}

function footer(t: Theme, page: number): string {
  return (
    `<rect x="${MARGIN}" y="668" width="${VW - MARGIN * 2}" height="1" fill="#e2e8f0"/>` +
    `<text x="${MARGIN}" y="692" font-size="13" font-weight="bold"><tspan fill="${SUB}">Sales</tspan><tspan fill="${t.base}">Flow</tspan></text>` +
    line1(`${DATE_ID}   ·   ${String(page).padStart(2, '0')}`, VW - MARGIN, 692, { size: 12, color: MUT, anchor: 'end' })
  )
}

/** Strip insight "so what" di bawah konten. */
function noteStrip(t: Theme, note?: string): string {
  if (!note?.trim()) return ''
  const y = 596
  return (
    `<rect x="${MARGIN}" y="${y}" width="${VW - MARGIN * 2}" height="52" rx="12" fill="${t.tint}"/>` +
    `<rect x="${MARGIN}" y="${y}" width="6" height="52" rx="3" fill="${t.base}"/>` +
    line1('INSIGHT', MARGIN + 26, y + 22, { size: 10, color: t.base, weight: 'bold', spacing: 2 }) +
    textBlock(fitLines(note, VW - MARGIN * 2 - 130, 14, 1), MARGIN + 26, y + 40, { size: 14, color: INK })
  )
}

/** Batas bawah area konten (menyempit bila ada note strip). */
function contentBottom(spec: DeckSlideSpec): number {
  return spec.note?.trim() ? 582 : 648
}

const CONTENT_TOP = 168

interface Ctx {
  num: number
  page: number
  title: string
}

// ══ LAYOUT ═══════════════════════════════════════════════════════════════════

function layCover(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const title = spec.heading || ctx.title
  const tl = fitLines(title, 950, 58, 3)
  const yTitle = 330 - (tl.length - 1) * 34
  const eyebrow = (spec.eyebrow || 'Playbook Strategis').toUpperCase()
  return (
    svgOpen() +
    defs(t) +
    `<rect width="${VW}" height="${VH}" fill="url(#g1)"/>` +
    motif(t, true) +
    `<rect x="${MARGIN}" y="200" width="70" height="7" rx="3.5" fill="${t.bright}"/>` +
    line1(eyebrow, MARGIN, 182, { size: 16, color: t.tint, weight: 'bold', spacing: 5 }) +
    textBlock(tl, MARGIN, yTitle, { size: 58, color: PAPER, weight: 'bold', lineH: 68 }) +
    textBlock(fitLines(spec.body, 920, 24, 2), MARGIN, yTitle + tl.length * 68 + 20, { size: 24, color: t.tint, lineH: 34 }) +
    `<rect x="${MARGIN}" y="622" width="380" height="1.5" fill="${t.bright}" opacity="0.4"/>` +
    `<text x="${MARGIN}" y="660" font-size="17" font-weight="bold"><tspan fill="${PAPER}">Sales</tspan><tspan fill="${t.bright}">Flow</tspan><tspan fill="${t.tint}" font-weight="normal">   ·   ${DATE_ID}</tspan></text>` +
    `</svg>`
  )
}

function layClosing(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const head = spec.heading || 'Terima Kasih'
  const body = spec.body || `Mari eksekusi dan menangkan ${ctx.title} bersama.`
  // Judul penutup bisa 2 baris — badan teks mengikuti tingginya.
  const headLines = fitLines(head, 900, 52, 2)
  const headY = 366
  const bodyY = headY + headLines.length * 60 + 4
  return (
    svgOpen() +
    defs(t) +
    `<rect width="${VW}" height="${VH}" fill="url(#g1)"/>` +
    motif(t, true) +
    `<rect x="${MARGIN}" y="288" width="70" height="7" rx="3.5" fill="${t.bright}"/>` +
    textBlock(headLines, MARGIN, headY, { size: 52, color: PAPER, weight: 'bold', lineH: 60 }) +
    textBlock(fitLines(body, 880, 22, 3), MARGIN, bodyY, { size: 22, color: t.tint, lineH: 31 }) +
    `<text x="${MARGIN}" y="644" font-size="18" font-weight="bold"><tspan fill="${PAPER}">Sales</tspan><tspan fill="${t.bright}">Flow</tspan></text>` +
    `</svg>`
  )
}

function layStatement(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const bottom = contentBottom(spec)
  const h = bottom - CONTENT_TOP
  const lines = fitLines(spec.body, VW - MARGIN * 2 - 170, 34, 6)
  return (
    svgOpen() +
    defs(t) +
    lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Inti Gagasan', spec.heading || 'Statement') +
    `<rect x="${MARGIN}" y="${CONTENT_TOP}" width="${VW - MARGIN * 2}" height="${h}" rx="20" fill="url(#g1)"/>` +
    `<g opacity="0.5">${motif(t, true)}</g>` +
    `<rect x="${MARGIN + 50}" y="${CONTENT_TOP + 46}" width="7" height="${h - 92}" rx="3.5" fill="${t.bright}"/>` +
    centeredBlock(lines, MARGIN + 84, CONTENT_TOP + h / 2, { size: 34, color: PAPER, weight: 'bold', lineH: 48 }) +
    noteStrip(t, spec.note) +
    footer(t, ctx.page) +
    `</svg>`
  )
}

function layQuote(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const bottom = contentBottom(spec)
  const h = bottom - CONTENT_TOP
  const lines = fitLines(spec.body, VW - MARGIN * 2 - 260, 30, 5)
  const cy = CONTENT_TOP + h / 2 - (spec.attribution ? 22 : 0)
  return (
    svgOpen() +
    defs(t) +
    lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Suara Pasar', spec.heading || 'Kutipan') +
    `<rect x="${MARGIN}" y="${CONTENT_TOP}" width="${VW - MARGIN * 2}" height="${h}" rx="20" fill="${PAPER}" stroke="${t.line}" stroke-width="2"/>` +
    `<circle cx="${VW - MARGIN - 90}" cy="${CONTENT_TOP + 90}" r="90" fill="${t.tint}"/>` +
    `<text x="${MARGIN + 62}" y="${CONTENT_TOP + 150}" font-family="Georgia, serif" font-size="150" fill="${t.base}" opacity="0.22">“</text>` +
    centeredBlock(lines, MARGIN + 130, cy, { size: 30, color: INK, weight: 'bold', lineH: 44, italic: true }) +
    (spec.attribution
      ? `<rect x="${MARGIN + 130}" y="${bottom - 74}" width="46" height="4" rx="2" fill="${t.bright}"/>` +
        line1(spec.attribution, MARGIN + 130, bottom - 40, { size: 17, color: t.base, weight: 'bold' })
      : '') +
    noteStrip(t, spec.note) +
    footer(t, ctx.page) +
    `</svg>`
  )
}

function layMetrics(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.metrics ?? []).slice(0, 4)
  const bottom = contentBottom(spec)
  const gap = 26
  const cw = (VW - MARGIN * 2 - gap * (items.length - 1)) / Math.max(items.length, 1)
  const y = CONTENT_TOP + 26
  const H = bottom - y
  let cards = ''
  items.forEach((m, i) => {
    const x = MARGIN + i * (cw + gap)
    const valLines = fitLines(m.value, cw - 56, 46, 2)
    cards +=
      `<rect x="${x}" y="${y}" width="${cw}" height="${H}" rx="18" fill="${PAPER}" stroke="${t.line}" stroke-width="1.5"/>` +
      `<rect x="${x}" y="${y}" width="${cw}" height="7" rx="3.5" fill="url(#g4)"/>` +
      `<circle cx="${x + 44}" cy="${y + 56}" r="21" fill="${t.tint}"/>` +
      line1(String(i + 1).padStart(2, '0'), x + 44, y + 63, { size: 17, color: t.base, weight: 'bold', anchor: 'middle' }) +
      textBlock(valLines, x + 28, y + 150, { size: 46, color: t.base, weight: 'bold', lineH: 50 }) +
      `<rect x="${x + 28}" y="${y + 150 + (valLines.length - 1) * 50 + 22}" width="44" height="4" rx="2" fill="${t.bright}"/>` +
      textBlock(fitLines(m.label, cw - 56, 18, 2), x + 28, y + 150 + (valLines.length - 1) * 50 + 62, { size: 18, color: INK, weight: 'bold', lineH: 24 }) +
      textBlock(fitLines(m.caption, cw - 56, 14, 4), x + 28, y + 150 + (valLines.length - 1) * 50 + 118, { size: 14, color: MUT, lineH: 21 })
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Bukti & Skala', spec.heading || 'Angka Kunci') +
    cards + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function layPillars(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.cards ?? []).slice(0, 4)
  const bottom = contentBottom(spec)
  const gap = 24
  const cw = (VW - MARGIN * 2 - gap * (items.length - 1)) / Math.max(items.length, 1)
  const y = CONTENT_TOP + 22
  const H = bottom - y
  const band = 96
  let cards = ''
  items.forEach((c, i) => {
    const x = MARGIN + i * (cw + gap)
    cards +=
      `<rect x="${x}" y="${y}" width="${cw}" height="${H}" rx="18" fill="${PAPER}" stroke="${t.line}" stroke-width="1.5"/>` +
      `<path d="M ${x} ${y + 18} a 18 18 0 0 1 18 -18 h ${cw - 36} a 18 18 0 0 1 18 18 v ${band - 18} h -${cw} Z" fill="${i % 2 === 0 ? t.base : t.dark}"/>` +
      `<circle cx="${x + 44}" cy="${y + 48}" r="23" fill="${PAPER}" opacity="0.2"/>` +
      line1(String(i + 1).padStart(2, '0'), x + 44, y + 56, { size: 20, color: PAPER, weight: 'bold', anchor: 'middle' }) +
      textBlock(fitLines(c.title, cw - 120, 19, 2), x + 78, y + 42, { size: 19, color: PAPER, weight: 'bold', lineH: 24 }) +
      (c.tag
        ? `<rect x="${x + 24}" y="${y + band + 18}" width="${Math.min(cw - 48, 8 + String(c.tag).length * 8)}" height="24" rx="12" fill="${t.tint}"/>` +
          line1(clip(c.tag, 22), x + 36, y + band + 34, { size: 12, color: t.base, weight: 'bold' })
        : '') +
      textBlock(fitLines(c.detail, cw - 52, 16, 9), x + 26, y + band + (c.tag ? 74 : 44), { size: 16, color: SUB, lineH: 25 })
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Keunggulan', spec.heading || 'Pilar Utama') +
    cards + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function layBullets(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.bullets ?? []).slice(0, 10)
  const bottom = contentBottom(spec)
  const twoCol = items.length > 5
  const colW = twoCol ? (VW - MARGIN * 2 - 44) / 2 : VW - MARGIN * 2
  const perCol = twoCol ? Math.ceil(items.length / 2) : items.length
  const y0 = CONTENT_TOP + 14
  const rowH = Math.min(80, (bottom - y0) / Math.max(perCol, 1))
  let rows = ''
  items.forEach((it, i) => {
    const col = twoCol && i >= perCol ? 1 : 0
    const idx = twoCol && i >= perCol ? i - perCol : i
    const x = MARGIN + col * (colW + 44)
    const y = y0 + idx * rowH
    const cy = y + rowH / 2 - 4
    rows +=
      `<circle cx="${x + 20}" cy="${cy}" r="17" fill="${t.tint}"/>` +
      checkIcon(x + 20, cy, 11, t.base) +
      centeredBlock(fitLines(it, colW - 62, 17, 3), x + 50, cy, { size: 17, color: SUB, lineH: 23 })
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Cara Menang', spec.heading || 'Poin Kunci') +
    rows + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function laySteps(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.bullets ?? []).slice(0, 6)
  const bottom = contentBottom(spec)
  const y0 = CONTENT_TOP + 18
  const rowH = Math.min(84, (bottom - y0) / Math.max(items.length, 1))
  const railX = MARGIN + 30
  let out = ''
  if (items.length > 1)
    out += `<rect x="${railX - 2}" y="${y0 + 8}" width="4" height="${(items.length - 1) * rowH}" rx="2" fill="${t.line}"/>`
  items.forEach((it, i) => {
    const y = y0 + i * rowH
    out +=
      `<circle cx="${railX}" cy="${y + 8}" r="21" fill="${t.base}"/>` +
      line1(String(i + 1), railX, y + 15, { size: 17, color: PAPER, weight: 'bold', anchor: 'middle' }) +
      centeredBlock(fitLines(it, VW - railX - MARGIN - 60, 17, 3), railX + 44, y + 8, { size: 17, color: SUB, lineH: 24 })
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Langkah', spec.heading || 'Tahapan') +
    out + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function layProcess(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.cards ?? []).slice(0, 5)
  const bottom = contentBottom(spec)
  const n = Math.max(items.length, 1)
  const gap = 16
  const cw = (VW - MARGIN * 2 - gap * (n - 1)) / n
  const y = CONTENT_TOP + 52
  const H = Math.min(300, bottom - y - 20)
  const notch = 26
  let out = ''
  items.forEach((c, i) => {
    const x = MARGIN + i * (cw + gap)
    const last = i === items.length - 1
    // chevron
    const d = last
      ? `M ${x} ${y} H ${x + cw} V ${y + H} H ${x} L ${x + notch} ${y + H / 2} Z`
      : `M ${x} ${y} H ${x + cw - notch} L ${x + cw} ${y + H / 2} L ${x + cw - notch} ${y + H} H ${x} L ${x + notch} ${y + H / 2} Z`
    const first = i === 0
    const dFirst = last
      ? `M ${x} ${y} H ${x + cw} V ${y + H} H ${x} Z`
      : `M ${x} ${y} H ${x + cw - notch} L ${x + cw} ${y + H / 2} L ${x + cw - notch} ${y + H} H ${x} Z`
    const fill = i % 2 === 0 ? t.base : t.dark
    out +=
      `<path d="${first ? dFirst : d}" fill="${fill}"/>` +
      line1(`0${i + 1}`, x + (first ? 30 : 46), y + 42, { size: 15, color: PAPER, weight: 'bold', opacity: 0.7 }) +
      textBlock(fitLines(c.title, cw - 70, 18, 2), x + (first ? 30 : 46), y + 78, { size: 18, color: PAPER, weight: 'bold', lineH: 23 }) +
      textBlock(fitLines(c.detail, cw - 76, 13, 5), x + (first ? 30 : 46), y + 132, { size: 13, color: t.tint, lineH: 19 })
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Alur', spec.heading || 'Proses') +
    out + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function layComparison(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const cols = (spec.columns ?? []).slice(0, 2)
  const bottom = contentBottom(spec)
  const gap = 34
  const cw = (VW - MARGIN * 2 - gap) / 2
  const y = CONTENT_TOP + 16
  const H = bottom - y
  let out = ''
  cols.forEach((c, i) => {
    const x = MARGIN + i * (cw + gap)
    const isRight = i === 1
    const bg = isRight ? t.tint : DANGER_BG
    const accent = isRight ? t.base : DANGER
    const ink = isRight ? INK : DANGER_INK
    out +=
      `<rect x="${x}" y="${y}" width="${cw}" height="${H}" rx="18" fill="${bg}"/>` +
      `<rect x="${x}" y="${y}" width="${cw}" height="62" rx="18" fill="${accent}"/>` +
      `<rect x="${x}" y="${y + 40}" width="${cw}" height="22" fill="${accent}"/>` +
      textBlock(fitLines(c.title, cw - 44, 19, 1), x + 24, y + 38, { size: 19, color: PAPER, weight: 'bold' }) +
      (() => {
        const list = (c.items ?? []).slice(0, 6)
        const iy = y + 92
        const rh = Math.min(64, (H - 108) / Math.max(list.length, 1))
        return list
          .map((it, k) => {
            const cy = iy + k * rh + rh / 2 - 8
            const ico = isRight ? checkIcon(x + 34, cy, 10, accent) : warnIcon(x + 34, cy, 9, accent)
            return ico + centeredBlock(fitLines(it, cw - 82, 15, 3), x + 58, cy, { size: 15, color: ink, lineH: 21 })
          })
          .join('')
      })()
  })
  // panah pemisah di tengah
  out += `<circle cx="${VW / 2}" cy="${y + H / 2}" r="24" fill="${PAPER}" stroke="${t.line}" stroke-width="2"/>` + arrowIcon(VW / 2, y + H / 2, 13, t.base)
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Kontras', spec.heading || 'Sebelum vs Sesudah') +
    out + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function layMatrix(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const q = (spec.quadrants ?? []).slice(0, 4)
  const bottom = contentBottom(spec)
  const gap = 20
  const cw = (VW - MARGIN * 2 - gap) / 2
  const y = CONTENT_TOP + 14
  const ch = (bottom - y - gap) / 2
  const fills = [t.base, t.dark, t.accent2, t.bright]
  let out = ''
  q.forEach((quad, i) => {
    const col = i % 2
    const row = Math.floor(i / 2)
    const x = MARGIN + col * (cw + gap)
    const yy = y + row * (ch + gap)
    const c = fills[i % 4]
    out +=
      `<rect x="${x}" y="${yy}" width="${cw}" height="${ch}" rx="16" fill="${PAPER}" stroke="${t.line}" stroke-width="1.5"/>` +
      `<rect x="${x}" y="${yy}" width="${cw}" height="6" rx="3" fill="${c}"/>` +
      `<circle cx="${x + 30}" cy="${yy + 42}" r="14" fill="${c}"/>` +
      line1(String.fromCharCode(65 + i), x + 30, yy + 48, { size: 13, color: PAPER, weight: 'bold', anchor: 'middle' }) +
      textBlock(fitLines(quad.title, cw - 80, 18, 1), x + 54, yy + 48, { size: 18, color: INK, weight: 'bold' }) +
      (() => {
        const list = (quad.items ?? []).slice(0, 3)
        return list
          .map((it, k) => {
            const ly = yy + 78 + k * 34
            return `<circle cx="${x + 32}" cy="${ly + 2}" r="3.5" fill="${c}"/>` + textBlock(fitLines(it, cw - 76, 14, 2), x + 46, ly + 7, { size: 14, color: SUB, lineH: 19 })
          })
          .join('')
      })()
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Pemetaan', spec.heading || 'Matriks') +
    out + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function influenceRank(v?: string): number {
  const s = (v ?? '').toLowerCase()
  if (/tinggi|high|kunci|key|utama|decision/.test(s)) return 3
  if (/rendah|low|minor/.test(s)) return 1
  return 2
}

function layPeople(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.people ?? []).slice(0, 6)
  const bottom = contentBottom(spec)
  const perRow = items.length <= 4 ? 2 : 3
  const gap = 22
  const cw = (VW - MARGIN * 2 - gap * (perRow - 1)) / perRow
  const rows = Math.ceil(items.length / perRow)
  const y0 = CONTENT_TOP + 12
  const ch = Math.min(200, (bottom - y0 - gap * (rows - 1)) / Math.max(rows, 1))
  const col = (r: number) => (r === 3 ? t.base : r === 2 ? t.accent2 : MUT)
  const lbl = (r: number) => (r === 3 ? 'Pengaruh tinggi' : r === 2 ? 'Pengaruh sedang' : 'Pengaruh rendah')
  let out = ''
  items.forEach((p, i) => {
    const c0 = i % perRow
    const r0 = Math.floor(i / perRow)
    const x = MARGIN + c0 * (cw + gap)
    const y = y0 + r0 * (ch + gap)
    const rank = influenceRank(p.influence)
    const c = col(rank)
    const initials = String(p.name || '?')
      .split(/\s+/)
      .slice(0, 2)
      .map((w) => w[0])
      .join('')
      .toUpperCase()
    // Alir vertikal: nama bisa 2 baris, jadi role/badge/angle mengikuti tinggi
    // nama sebenarnya (bukan offset tetap) agar tidak pernah bertumpuk.
    const nameLines = fitLines(p.name, cw - 108, 19, 2)
    const nameBottom = y + 42 + (nameLines.length - 1) * 23
    const roleY = p.role ? nameBottom + 21 : nameBottom
    const badgeY = roleY + 13
    const angleY = badgeY + 44
    const angleLines = p.angle ? fitLines('▸ ' + p.angle, cw - 52, 13, 2) : []
    const angleFits = angleY + (angleLines.length - 1) * 19 <= y + ch - 10
    out +=
      `<rect x="${x}" y="${y}" width="${cw}" height="${ch}" rx="16" fill="${PAPER}" stroke="${t.line}" stroke-width="1.5"/>` +
      `<rect x="${x}" y="${y}" width="6" height="${ch}" rx="3" fill="${c}"/>` +
      `<circle cx="${x + 50}" cy="${y + 48}" r="25" fill="${t.tint}"/>` +
      line1(initials || '•', x + 50, y + 55, { size: 17, color: t.base, weight: 'bold', anchor: 'middle' }) +
      textBlock(nameLines, x + 88, y + 42, { size: 19, color: INK, weight: 'bold', lineH: 23 }) +
      (p.role ? textBlock(fitLines(p.role, cw - 108, 14, 1), x + 88, roleY, { size: 14, color: SUB }) : '') +
      `<rect x="${x + 26}" y="${badgeY}" width="${Math.min(cw - 52, 150)}" height="24" rx="12" fill="${c}" opacity="0.14"/>` +
      `<circle cx="${x + 42}" cy="${badgeY + 12}" r="4" fill="${c}"/>` +
      line1(lbl(rank), x + 54, badgeY + 16, { size: 12, color: c, weight: 'bold' }) +
      (angleFits ? textBlock(angleLines, x + 26, angleY, { size: 13, color: MUT, lineH: 19 }) : '')
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Peta Pengaruh', spec.heading || 'Stakeholder Kunci') +
    out + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function layRisks(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.risks ?? []).slice(0, 5)
  const bottom = contentBottom(spec)
  const y0 = CONTENT_TOP + 30
  const gap = 12
  const rowH = Math.min(100, (bottom - y0 - gap * (items.length - 1)) / Math.max(items.length, 1))
  const riskW = (VW - MARGIN * 2) * 0.44
  const mitX = MARGIN + riskW + 44
  const mitW = VW - MARGIN - mitX
  const impCol = (v?: string) => {
    const s = (v ?? '').toLowerCase()
    if (/tinggi|high/.test(s)) return DANGER
    if (/rendah|low/.test(s)) return '#d97706'
    return '#ea580c'
  }
  let out =
    line1('RISIKO', MARGIN + 14, y0 - 12, { size: 12, color: DANGER_INK, weight: 'bold', spacing: 2 }) +
    line1('MITIGASI', mitX + 14, y0 - 12, { size: 12, color: t.base, weight: 'bold', spacing: 2 })
  items.forEach((r, i) => {
    const y = y0 + i * (rowH + gap)
    const c = impCol(r.impact)
    const cy = y + rowH / 2
    out +=
      `<rect x="${MARGIN}" y="${y}" width="${riskW}" height="${rowH}" rx="12" fill="${DANGER_BG}"/>` +
      `<rect x="${MARGIN}" y="${y}" width="6" height="${rowH}" rx="3" fill="${c}"/>` +
      warnIcon(MARGIN + 32, cy, 10, c) +
      centeredBlock(fitLines(r.risk, riskW - 70, 15, 3), MARGIN + 54, cy, { size: 15, color: DANGER_INK, lineH: 20 }) +
      `<circle cx="${MARGIN + riskW + 22}" cy="${cy}" r="16" fill="${PAPER}" stroke="${t.line}" stroke-width="1.5"/>` +
      arrowIcon(MARGIN + riskW + 22, cy, 9, t.base) +
      `<rect x="${mitX}" y="${y}" width="${mitW}" height="${rowH}" rx="12" fill="${t.tint}"/>` +
      `<rect x="${mitX}" y="${y}" width="6" height="${rowH}" rx="3" fill="${t.base}"/>` +
      checkIcon(mitX + 32, cy, 10, t.base) +
      centeredBlock(fitLines(r.mitigation, mitW - 70, 15, 3), mitX + 54, cy, { size: 15, color: INK, lineH: 20 })
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Kewaspadaan', spec.heading || 'Risiko & Mitigasi') +
    out + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function layTimeline(t: Theme, spec: DeckSlideSpec, ctx: Ctx): string {
  const items = (spec.timeline_plan ?? []).slice(0, 8)
  const bottom = contentBottom(spec)
  const totalDays = Math.max(...items.map((p) => (p.start_day ?? 0) + Math.max(p.duration_days ?? 1, 1)), 7)
  const labelW = 292
  const chartX = MARGIN + labelW
  const chartW = VW - MARGIN - chartX
  const top = CONTENT_TOP + 44
  const rowH = Math.min(60, (bottom - top) / Math.max(items.length, 1))
  const weeks = Math.ceil(totalDays / 7)
  let out = ''
  for (let w = 0; w < weeks; w++) {
    const gx = chartX + ((w * 7) / totalDays) * chartW
    out +=
      `<line x1="${gx}" y1="${top - 10}" x2="${gx}" y2="${top + rowH * items.length + 4}" stroke="#e2e8f0" stroke-width="1"/>` +
      line1(`M${w + 1}`, gx + 6, top - 16, { size: 12, color: MUT, weight: 'bold' })
  }
  items.forEach((p, i) => {
    const y = top + i * rowH
    const bx = chartX + ((p.start_day ?? 0) / totalDays) * chartW
    const bw = Math.max((Math.max(p.duration_days ?? 1, 1) / totalDays) * chartW, 14)
    const fill = i % 2 === 0 ? t.base : t.accent2
    if (i % 2 === 0) out += `<rect x="${chartX}" y="${y}" width="${chartW}" height="${rowH - 8}" rx="6" fill="${PAPER2}"/>`
    out +=
      centeredBlock(fitLines(p.activity, labelW - 20, 15, 2), MARGIN, y + (rowH - 8) / 2, { size: 15, color: INK, lineH: 19 }) +
      `<rect x="${bx}" y="${y + 5}" width="${bw}" height="${rowH - 18}" rx="7" fill="${fill}"/>` +
      (bw > 52 ? line1(`${p.duration_days ?? 1}h`, bx + bw / 2, y + (rowH - 8) / 2 + 5, { size: 12, color: PAPER, weight: 'bold', anchor: 'middle' }) : '')
  })
  return (
    svgOpen() + defs(t) + lightBase(t) +
    header(t, ctx.num, spec.eyebrow || 'Eksekusi', spec.heading || 'Rencana Kerja') +
    out + noteStrip(t, spec.note) + footer(t, ctx.page) + `</svg>`
  )
}

function clip(s: unknown, max: number): string {
  const v = String(s ?? '').trim()
  return v.length > max ? v.slice(0, max - 1) + '…' : v
}

// ── Registry ─────────────────────────────────────────────────────────────────
type LayoutFn = (t: Theme, spec: DeckSlideSpec, ctx: Ctx) => string

const LAYOUTS: Record<DeckLayout, LayoutFn> = {
  cover: layCover,
  closing: layClosing,
  statement: layStatement,
  quote: layQuote,
  metrics: layMetrics,
  pillars: layPillars,
  bullets: layBullets,
  steps: laySteps,
  process: layProcess,
  comparison: layComparison,
  matrix: layMatrix,
  people: layPeople,
  risks: layRisks,
  timeline: layTimeline,
}

const TRANSITIONS: Record<DeckLayout, DeckSlide['transition']> = {
  cover: 'fade',
  closing: 'fade',
  statement: 'zoom',
  quote: 'fade',
  metrics: 'wipe',
  pillars: 'push',
  bullets: 'wipe',
  steps: 'wipe',
  process: 'push',
  comparison: 'split',
  matrix: 'zoom',
  people: 'push',
  risks: 'wipe',
  timeline: 'push',
}

/** Layout tak dikenal / data kosong → pilih pengganti yang masuk akal. */
function resolveLayout(spec: DeckSlideSpec): DeckLayout {
  const l = spec.layout
  if (l && LAYOUTS[l]) {
    // Turunkan layout bila datanya tidak tersedia agar slide tak kosong.
    const has = (v?: unknown[], min = 1) => Array.isArray(v) && v.length >= min
    // Metrics/comparison butuh minimal 2 entri agar komposisinya tidak timpang.
    if (l === 'metrics' && !has(spec.metrics, 2)) return spec.body ? 'statement' : 'bullets'
    if ((l === 'pillars' || l === 'process') && !has(spec.cards)) return has(spec.bullets) ? 'bullets' : 'statement'
    if (l === 'people' && !has(spec.people)) return 'bullets'
    if (l === 'risks' && !has(spec.risks)) return 'bullets'
    if (l === 'comparison' && !has(spec.columns, 2)) return has(spec.bullets) ? 'bullets' : 'statement'
    if (l === 'matrix' && !has(spec.quadrants)) return 'bullets'
    if (l === 'timeline' && !has(spec.timeline_plan)) return 'steps'
    if ((l === 'bullets' || l === 'steps') && !has(spec.bullets)) return 'statement'
    if ((l === 'statement' || l === 'quote') && !spec.body?.trim()) return 'bullets'
    return l
  }
  if (has2(spec.metrics)) return 'metrics'
  if (has2(spec.risks)) return 'risks'
  if (has2(spec.people)) return 'people'
  if (has2(spec.cards)) return 'pillars'
  if (has2(spec.bullets)) return 'bullets'
  return 'statement'
}
function has2(v?: unknown[]): boolean {
  return Array.isArray(v) && v.length > 0
}

// ── Konversi playbook lama (tanpa deck) menjadi spec ─────────────────────────
function nonEmpty(a?: string[]): string[] {
  return (a ?? []).map((s) => String(s ?? '').trim()).filter(Boolean)
}

function legacySpecs(c: PlaybookContent): DeckSlideSpec[] {
  const out: DeckSlideSpec[] = []
  out.push({ layout: 'cover', heading: c.title, body: c.subtitle })

  const metrics = (c.metrics ?? []).filter((m) => m && (m.value || m.label))
  if (c.summary?.trim())
    out.push({ layout: 'statement', eyebrow: 'Ikhtisar', heading: 'Ringkasan Eksekutif', body: c.summary })
  if (metrics.length >= 2) out.push({ layout: 'metrics', heading: 'Angka Kunci', metrics })

  const diffs = (c.differentiators ?? []).filter((d) => d && (d.title || d.detail))
  if (diffs.length >= 2)
    out.push({ layout: 'pillars', eyebrow: 'Keunggulan', heading: 'Yang Membedakan Kami', cards: diffs.map((d) => ({ title: d.title, detail: d.detail })) })
  else if (c.value_prop?.trim())
    out.push({ layout: 'statement', eyebrow: 'Mengapa Kami', heading: 'Value Proposition', body: c.value_prop })

  const people = (c.stakeholder_map ?? []).filter((s) => s && s.name)
  const peopleFallback = nonEmpty(c.stakeholders).map((s) => ({ name: s }))
  if (people.length || peopleFallback.length)
    out.push({ layout: 'people', heading: 'Stakeholder Kunci', people: people.length ? people : peopleFallback })

  if (nonEmpty(c.strategy_checklist).length)
    out.push({ layout: 'bullets', eyebrow: 'Cara Menang', heading: 'Strategi Pemenangan', bullets: nonEmpty(c.strategy_checklist) })

  if (c.timeline_plan?.length) out.push({ layout: 'timeline', heading: 'Rencana Kerja', timeline_plan: c.timeline_plan })
  else if (nonEmpty(c.timeline).length) out.push({ layout: 'steps', eyebrow: 'Eksekusi', heading: 'Rencana Kerja', bullets: nonEmpty(c.timeline) })

  const risks = (c.risk_matrix ?? []).filter((r) => r && (r.risk || r.mitigation))
  if (risks.length) out.push({ layout: 'risks', heading: 'Risiko & Mitigasi', risks })
  else if (nonEmpty(c.risks).length)
    out.push({ layout: 'risks', heading: 'Risiko & Mitigasi', risks: nonEmpty(c.risks).map((r) => ({ risk: r, mitigation: '—' })) })

  if (nonEmpty(c.next_actions).length)
    out.push({ layout: 'steps', eyebrow: 'Langkah Berikutnya', heading: 'Next Actions', bullets: nonEmpty(c.next_actions) })

  out.push({ layout: 'closing' })
  return out
}

/** Spec final: pakai deck rancangan AI bila ada, jika tidak susun dari field datar. */
function toSpecs(c: PlaybookContent): DeckSlideSpec[] {
  const deck = (c.deck ?? []).filter((s) => s && typeof s === 'object')
  if (deck.length >= 3) {
    const specs = [...deck]
    // Pastikan selalu ada cover & closing walau AI lupa.
    if (specs[0].layout !== 'cover') specs.unshift({ layout: 'cover', heading: c.title, body: c.subtitle })
    if (specs[specs.length - 1].layout !== 'closing') specs.push({ layout: 'closing' })
    return specs
  }
  return legacySpecs(c)
}

/** Bangun seluruh slide deck sebagai SVG. */
export function buildDeck(content: PlaybookContent, fallbackTitle: string): DeckSlide[] {
  const t = pickTheme(content)
  const title = (content.title || fallbackTitle || 'Playbook Strategis').trim()
  const specs = toSpecs(content)

  let num = 0
  return specs.map((spec, i) => {
    const layout = resolveLayout(spec)
    const numbered = layout !== 'cover' && layout !== 'closing'
    if (numbered) num += 1
    const ctx: Ctx = { num, page: i + 1, title }
    const heading =
      spec.heading?.trim() ||
      (layout === 'cover' ? title : layout === 'closing' ? 'Penutup' : defaultHeading(layout))
    // Utamakan desain karangan AI; layout katalog hanya dipakai bila SVG-nya
    // gagal sanitasi/validasi. Inilah yang membuat tiap deck benar-benar beda
    // tanpa risiko slide rusak sampai ke user.
    const authored = prepareAiSvg(spec.svg)
    return {
      key: `${layout}-${i}`,
      title: heading,
      section: heading,
      svg: authored ?? LAYOUTS[layout](t, spec, ctx),
      aiDesigned: authored !== null,
      transition: TRANSITIONS[layout],
    }
  })
}

function defaultHeading(l: DeckLayout): string {
  const m: Record<DeckLayout, string> = {
    cover: 'Cover',
    closing: 'Penutup',
    statement: 'Inti Gagasan',
    quote: 'Kutipan',
    metrics: 'Angka Kunci',
    pillars: 'Pilar Utama',
    bullets: 'Poin Kunci',
    steps: 'Tahapan',
    process: 'Proses',
    comparison: 'Perbandingan',
    matrix: 'Matriks',
    people: 'Stakeholder',
    risks: 'Risiko & Mitigasi',
    timeline: 'Rencana Kerja',
  }
  return m[l]
}

/** Nama file dari judul: lowercase, spasi → underscore, tanpa simbol, plus
 * kode unik berbasis timestamp. */
export function pptFileName(title: string): string {
  const base =
    (title || 'playbook')
      .toLowerCase()
      .normalize('NFKD')
      .replace(/[^\w\s-]/g, '')
      .trim()
      .replace(/[\s-]+/g, '_')
      .replace(/_+/g, '_')
      .replace(/^_|_$/g, '')
      .slice(0, 60) || 'playbook'
  const d = new Date()
  const p = (nn: number) => String(nn).padStart(2, '0')
  const code = `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}_${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`
  return `${base}_${code}.pptx`
}
