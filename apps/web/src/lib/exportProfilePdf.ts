import { jsPDF } from 'jspdf'
import type { Profile } from '../api/profile'
import type { Source } from '../api/sources'
import { slugify } from './format'

const MARGIN = 18
const PAGE_WIDTH = 210 // A4 mm
const CONTENT_WIDTH = PAGE_WIDTH - MARGIN * 2
const LINE_HEIGHT = 5.2

function formatCurrency(n: number | null, currency: string): string {
  if (n == null) return '—'
  return `${currency} ${n.toLocaleString('id-ID')}`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('id-ID', { day: '2-digit', month: 'long', year: 'numeric' })
}

/** Builds and downloads a formatted PDF snapshot of the current Company
 * Profile — replaces the old raw-JSON export. Client-side (jsPDF) since the
 * data is already loaded on the page; no backend round-trip needed. */
export function exportProfilePdf(profile: Profile, sources: Source[]) {
  const doc = new jsPDF({ unit: 'mm', format: 'a4' })
  let y = MARGIN

  function ensureSpace(next: number) {
    if (y + next > 297 - MARGIN) {
      doc.addPage()
      y = MARGIN
    }
  }

  function heading(text: string) {
    ensureSpace(10)
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(13)
    doc.setTextColor(16, 94, 62) // emerald-ish, matches app theme
    doc.text(text, MARGIN, y)
    y += 2
    doc.setDrawColor(16, 94, 62)
    doc.line(MARGIN, y, PAGE_WIDTH - MARGIN, y)
    y += 6
    doc.setTextColor(20, 20, 20)
  }

  function label(text: string) {
    ensureSpace(LINE_HEIGHT)
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(10)
    doc.text(text, MARGIN, y)
    y += LINE_HEIGHT
  }

  function body(text: string) {
    const lines = doc.splitTextToSize(text || '—', CONTENT_WIDTH)
    ensureSpace(lines.length * LINE_HEIGHT)
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(10)
    doc.text(lines, MARGIN, y)
    y += lines.length * LINE_HEIGHT + 2
  }

  function chipList(items: string[]) {
    body(items.length ? items.join(', ') : '—')
  }

  // ── Title ──
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(18)
  doc.setTextColor(16, 94, 62)
  doc.text(profile.company_name || 'Profil Perusahaan', MARGIN, y)
  y += 8
  doc.setTextColor(90, 90, 90)
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(9)
  doc.text(`Diekspor ${formatDate(new Date().toISOString())} — versi profil #${profile.version}`, MARGIN, y)
  y += 10
  doc.setTextColor(20, 20, 20)

  // ── Identitas & Kapabilitas ──
  heading('Identitas Perusahaan')
  label('One-liner')
  body(profile.one_liner ?? '')
  label('Visi')
  body(profile.vision ?? '')
  label('Misi')
  body(profile.mission ?? '')

  heading('Produk & Layanan')
  label('Produk')
  chipList(profile.products)
  label('Layanan')
  chipList(profile.service_categories)
  label('Tech Stack')
  chipList(profile.tech_stack)
  label('Portfolio / Bukti')
  chipList(profile.portfolio_refs)

  // ── Target Peluang ──
  if (profile.target) {
    const t = profile.target
    heading('Target Peluang')
    label('Negara Prioritas')
    chipList(t.countries)
    label('Industri Prioritas')
    chipList(t.industries)
    label('Nilai Project')
    body(
      `Min: ${formatCurrency(t.value_min, t.currency)}   Ideal: ${formatCurrency(t.value_ideal, t.currency)}   Maks: ${formatCurrency(t.value_max, t.currency)}`
    )
    label('Deadline Proposal Minimum')
    body(t.deadline_min_days != null ? `${t.deadline_min_days} hari kerja` : '—')
    label('Jenis Pengadaan')
    chipList(t.procurement_types)
    label('Bahasa Dokumen')
    chipList(t.document_languages)
    label('Model Kerja')
    body(t.work_model ?? '')
    label('Batasan Onsite')
    body(t.onsite_limit_note ?? '')
    label('Target Decision Maker')
    chipList(t.decision_maker_roles)
    label('Catatan Ukuran Buyer')
    body(t.buyer_size_note ?? '')
  }

  // ── No-Go ──
  if (profile.nogo) {
    heading('Kriteria Hindari (No-Go)')
    label('Preset')
    chipList(profile.nogo.preset_flags)
    label('Kustom')
    chipList(profile.nogo.custom)
  }

  // ── Keywords ──
  if (profile.keywords.length > 0) {
    heading('Keyword Pencarian')
    for (const k of profile.keywords) {
      label(k.category ? `Kategori: ${k.category}` : 'Kata Kunci Umum')
      body(`Positif: ${k.keywords.join(', ') || '—'}`)
      body(`Negatif: ${k.negative_keywords.join(', ') || '—'}`)
    }
  }

  // ── Scoring ──
  if (profile.scoring) {
    const s = profile.scoring
    heading('Bobot Scoring & Threshold')
    body(
      `Capability Fit ${s.weight_capability_fit}% · Portfolio Match ${s.weight_portfolio_match}% · Commercial ${s.weight_commercial_attractiveness}% · ` +
        `Eligibility ${s.weight_eligibility_fit}% · Deadline ${s.weight_deadline_feasibility}% · Strategic Value ${s.weight_strategic_account_value}% · ` +
        `Delivery Risk ${s.weight_delivery_risk}% · Competition ${s.weight_competition_win_probability}%`
    )
    body(
      `Threshold — Pursue ≥ ${s.threshold_pursue} · Review ≥ ${s.threshold_review} · Watchlist ≥ ${s.threshold_watchlist}`
    )
  }

  // ── Sources ──
  if (sources.length > 0) {
    heading('Sumber Crawl')
    for (const src of sources) {
      label(`${src.name}${src.enabled ? '' : ' (nonaktif)'}`)
      body(`${src.url}${src.country ? ` — ${src.country}` : ''}`)
      body(`Akses: ${src.access} · Prioritas: ${src.priority} · Frekuensi: ${src.frequency}`)
      if (src.legal_note) body(`Catatan legal: ${src.legal_note}`)
    }
  }

  doc.save(`profil-perusahaan-${slugify(profile.company_name)}-v${profile.version}.pdf`)
}
