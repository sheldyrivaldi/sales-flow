import type ExcelJS from 'exceljs'
import type { Prospect, ProspectStage } from '../api/prospects'
import { SOURCE_LABELS } from '../api/prospects'
import { slugify } from './format'

// Warna selaras design token (ARGB, tanpa '#'):
const EMERALD_600 = 'FF059669'
const EMERALD_50 = 'FFECFDF5'
const SLATE_100 = 'FFF1F5F9'
const SLATE_200 = 'FFE2E8F0'
const SLATE_900 = 'FF0F172A'
const WHITE = 'FFFFFFFF'

const STAGE_FILL: Record<ProspectStage, string> = {
  NEW: SLATE_100,
  QUALIFIED: 'FFEFF6FF',  // info-subtle
  ENGAGED: EMERALD_50,
  PROPOSAL: 'FFCCFBF1',   // accent-subtle
  WON: EMERALD_50,
  LOST: 'FFFEF2F2',       // danger-subtle
}

const thinBorder: Partial<ExcelJS.Borders> = {
  top: { style: 'thin', color: { argb: SLATE_200 } },
  bottom: { style: 'thin', color: { argb: SLATE_200 } },
  left: { style: 'thin', color: { argb: SLATE_200 } },
  right: { style: 'thin', color: { argb: SLATE_200 } },
}

/**
 * Ekspor pipeline ke file .xlsx berformat rapi: header emerald, freeze pane,
 * autofilter, format Rupiah, tint per stage, dan baris total. Murni
 * client-side (exceljs), tanpa AI.
 */
export async function exportPipelineExcel(
  prospects: Prospect[],
  ownerNames: Record<string, string>,
  companyName = 'SalesFlow',
) {
  // Dynamic import: exceljs ~1 MB — dimuat hanya saat tombol export diklik,
  // bukan ikut bundle awal aplikasi.
  const { default: Excel } = await import('exceljs')
  const wb = new Excel.Workbook()
  wb.creator = companyName
  wb.created = new Date()

  const ws = wb.addWorksheet('Pipeline', {
    views: [{ state: 'frozen', ySplit: 2 }],
  })

  ws.columns = [
    { key: 'no', width: 5 },
    { key: 'name', width: 32 },
    { key: 'company', width: 26 },
    { key: 'stage', width: 14 },
    { key: 'value', width: 18 },
    { key: 'source', width: 12 },
    { key: 'owner', width: 20 },
    { key: 'contact', width: 26 },
    { key: 'created', width: 14 },
  ]

  // ── Baris judul ──
  ws.mergeCells('A1:I1')
  const title = ws.getCell('A1')
  title.value = `Pipeline Prospek — diekspor ${new Date().toLocaleDateString('id-ID', { day: '2-digit', month: 'long', year: 'numeric' })}`
  title.font = { bold: true, size: 14, color: { argb: SLATE_900 } }
  title.alignment = { vertical: 'middle' }
  ws.getRow(1).height = 26

  // ── Header ──
  const header = ws.addRow(['No', 'Nama Prospek', 'Perusahaan', 'Stage', 'Estimasi Nilai (Rp)', 'Sumber', 'Owner', 'Kontak', 'Dibuat'])
  header.eachCell((cell) => {
    cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: EMERALD_600 } }
    cell.font = { bold: true, color: { argb: WHITE }, size: 11 }
    cell.alignment = { vertical: 'middle', horizontal: 'center' }
    cell.border = thinBorder
  })
  header.height = 22

  // ── Data ──
  prospects.forEach((p, i) => {
    const row = ws.addRow([
      i + 1,
      p.name,
      p.company ?? '—',
      p.stage,
      p.est_value ?? 0,
      SOURCE_LABELS[p.source_type] ?? p.source_type,
      p.owner_user_id ? (ownerNames[p.owner_user_id] ?? p.owner_user_id) : '—',
      p.contact_info ?? '—',
      new Date(p.created_at),
    ])
    row.eachCell((cell, col) => {
      cell.border = thinBorder
      cell.alignment = { vertical: 'middle', horizontal: col === 1 || col === 4 ? 'center' : col === 5 ? 'right' : 'left' }
    })
    row.getCell(5).numFmt = '#,##0'
    row.getCell(9).numFmt = 'dd/mm/yyyy'
    const stageCell = row.getCell(4)
    stageCell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: STAGE_FILL[p.stage] ?? SLATE_100 } }
    stageCell.font = { bold: true, size: 10, color: { argb: SLATE_900 } }
  })

  // ── Baris total ──
  const total = ws.addRow(['', 'TOTAL', '', '', prospects.reduce((s, p) => s + (p.est_value ?? 0), 0), '', '', '', ''])
  total.eachCell((cell) => {
    cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: SLATE_100 } }
    cell.font = { bold: true, color: { argb: SLATE_900 } }
    cell.border = thinBorder
  })
  total.getCell(5).numFmt = '#,##0'
  total.getCell(5).alignment = { horizontal: 'right' }

  // Autofilter di header (baris 2), kolom penuh.
  ws.autoFilter = { from: 'A2', to: 'I2' }

  // ── Unduh ──
  const buf = await wb.xlsx.writeBuffer()
  const blob = new Blob([buf], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `pipeline-${slugify(companyName)}-${new Date().toISOString().slice(0, 10)}.xlsx`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
