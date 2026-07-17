import type { PlaybookContent } from '../api/playbooks'
import { slugify } from './format'

const EMERALD = '059669'
const SLATE_900 = '0F172A'
const SLATE_600 = '475569'

/** Ekspor playbook menjadi file PowerPoint (.pptx) — satu slide per bagian,
 * gaya konsisten (judul emerald, isi slate). Hanya ditawarkan untuk playbook
 * event & custom. pptxgenjs di-lazy-load saat tombol diklik. */
export async function exportPlaybookPpt(content: PlaybookContent, title: string) {
  const { default: PptxGen } = await import('pptxgenjs')
  const pptx = new PptxGen()
  pptx.defineLayout({ name: 'WIDE', width: 13.33, height: 7.5 })
  pptx.layout = 'WIDE'

  function addTitleSlide() {
    const s = pptx.addSlide()
    s.background = { color: 'FFFFFF' }
    s.addShape('rect', { x: 0, y: 0, w: 13.33, h: 0.35, fill: { color: EMERALD } })
    s.addText('PLAYBOOK', {
      x: 0.8, y: 2.4, w: 11.7, h: 0.6,
      fontSize: 20, color: EMERALD, bold: true, charSpacing: 4,
    })
    s.addText(title, {
      x: 0.8, y: 3.0, w: 11.7, h: 1.6,
      fontSize: 36, color: SLATE_900, bold: true,
    })
    s.addText(new Date().toLocaleDateString('id-ID', { day: '2-digit', month: 'long', year: 'numeric' }), {
      x: 0.8, y: 4.7, w: 11.7, h: 0.5, fontSize: 14, color: SLATE_600,
    })
  }

  function addSectionSlide(heading: string, body: string | string[], numbered = false) {
    const s = pptx.addSlide()
    s.background = { color: 'FFFFFF' }
    s.addShape('rect', { x: 0, y: 0, w: 0.25, h: 7.5, fill: { color: EMERALD } })
    s.addText(heading, {
      x: 0.8, y: 0.5, w: 11.7, h: 0.8,
      fontSize: 26, color: SLATE_900, bold: true,
    })
    if (typeof body === 'string') {
      s.addText(body, { x: 0.8, y: 1.6, w: 11.7, h: 5.2, fontSize: 16, color: SLATE_600, valign: 'top' })
    } else {
      s.addText(
        body.map((t, i) => ({
          text: numbered ? `${i + 1}. ${t}` : t,
          options: { bullet: !numbered, fontSize: 15, color: SLATE_600, breakLine: true, paraSpaceAfter: 8 },
        })),
        { x: 0.8, y: 1.6, w: 11.7, h: 5.4, valign: 'top' },
      )
    }
  }

  addTitleSlide()
  addSectionSlide('Ringkasan', content.summary)
  addSectionSlide('Value Proposition', content.value_prop)
  addSectionSlide('Stakeholder Kunci', content.stakeholders)
  addSectionSlide('Strategi', content.strategy_checklist)
  if (content.timeline_plan && content.timeline_plan.length > 0) {
    addSectionSlide(
      'Rencana Kerja (Timeline)',
      content.timeline_plan.map(
        (t) => `${t.activity} — mulai hari ke-${t.start_day + 1}, durasi ${t.duration_days} hari`,
      ),
      true,
    )
  } else if (content.timeline.length > 0) {
    addSectionSlide('Timeline', content.timeline, true)
  }
  addSectionSlide('Risiko', content.risks)
  addSectionSlide('Next Actions', content.next_actions, true)

  await pptx.writeFile({ fileName: `playbook-${slugify(title)}.pptx` })
}
