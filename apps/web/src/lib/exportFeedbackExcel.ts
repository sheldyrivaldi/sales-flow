import type ExcelJS from 'exceljs'
import type { FeedbackForm, FeedbackSubmission } from '../api/feedbackForms'
import { slugify } from './format'

const EMERALD_600 = 'FF059669'
const SLATE_200 = 'FFE2E8F0'
const SLATE_900 = 'FF0F172A'
const WHITE = 'FFFFFFFF'

const thinBorder: Partial<ExcelJS.Borders> = {
  top: { style: 'thin', color: { argb: SLATE_200 } },
  bottom: { style: 'thin', color: { argb: SLATE_200 } },
  left: { style: 'thin', color: { argb: SLATE_200 } },
  right: { style: 'thin', color: { argb: SLATE_200 } },
}

// Ubah satu jawaban submission menjadi teks sel Excel sesuai tipe pertanyaan.
function answerText(form: FeedbackForm, sub: FeedbackSubmission, questionId: string): string {
  const q = form.questions.find((x) => x.id === questionId)
  const a = sub.answers.find((x) => x.question_id === questionId)
  if (!q || !a) return ''
  if (q.type === 'text') return a.text ?? ''
  if (q.type === 'rating') return a.rating != null ? `${a.rating}/${q.scale ?? 5}` : ''
  if (q.type === 'nps') return a.rating != null ? String(a.rating) : ''
  return (a.choice ?? []).join(', ')
}

/**
 * Ekspor semua respon satu form ke .xlsx: kolom = email/nama/tanggal + satu
 * kolom per pertanyaan, baris = tiap submission. Murni client-side (exceljs).
 */
export async function exportFeedbackExcel(form: FeedbackForm, submissions: FeedbackSubmission[]) {
  const { default: Excel } = await import('exceljs')
  const wb = new Excel.Workbook()
  wb.created = new Date()

  const ws = wb.addWorksheet('Feedback', { views: [{ state: 'frozen', ySplit: 2 }] })

  const questionHeaders = form.questions.map((q) => q.label)
  const headers = ['No', 'Email', 'Nama', 'Divisi', ...questionHeaders, 'Tanggal']

  ws.columns = headers.map((_, i) => ({
    key: `c${i}`,
    width: i === 0 ? 5 : i === headers.length - 1 ? 14 : 28,
  }))

  // Judul.
  const lastCol = ws.getColumn(headers.length).letter
  ws.mergeCells(`A1:${lastCol}1`)
  const title = ws.getCell('A1')
  title.value = `${form.title} — ${submissions.length} respon`
  title.font = { bold: true, size: 14, color: { argb: SLATE_900 } }
  title.alignment = { vertical: 'middle' }
  ws.getRow(1).height = 26

  // Header.
  const headerRow = ws.addRow(headers)
  headerRow.eachCell((cell) => {
    cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: EMERALD_600 } }
    cell.font = { bold: true, color: { argb: WHITE }, size: 11 }
    cell.alignment = { vertical: 'middle', horizontal: 'center', wrapText: true }
    cell.border = thinBorder
  })
  headerRow.height = 24

  // Data.
  submissions.forEach((sub, i) => {
    const row = ws.addRow([
      i + 1,
      sub.respondent_email ?? '—',
      sub.respondent_name ?? '—',
      sub.respondent_division ?? '—',
      ...form.questions.map((q) => answerText(form, sub, q.id)),
      new Date(sub.created_at),
    ])
    row.eachCell((cell, col) => {
      cell.border = thinBorder
      cell.alignment = { vertical: 'top', horizontal: col === 1 ? 'center' : 'left', wrapText: true }
    })
    row.getCell(headers.length).numFmt = 'dd/mm/yyyy hh:mm'
  })

  ws.autoFilter = { from: 'A2', to: `${lastCol}2` }

  const buf = await wb.xlsx.writeBuffer()
  const blob = new Blob([buf], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `feedback-${slugify(form.title)}-${new Date().toISOString().slice(0, 10)}.xlsx`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
