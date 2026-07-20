import type { PlaybookContent } from '../api/playbooks'
import { buildDeck, pptFileName, VW, VH } from './playbookDeck'
import type { DeckSlide } from './playbookDeck'

/** Ukuran slide PowerPoint 16:9 (inci). */
const W_IN = 13.333
const H_IN = 7.5

/** Skala raster — 2× supaya teks tetap tajam saat diproyeksikan. */
const SCALE = 2

/** Raster satu SVG menjadi data URI PNG lewat canvas. Blob URL bersifat
 * same-origin sehingga canvas tidak ter-taint. */
async function svgToPng(svg: string): Promise<string> {
  const blob = new Blob([svg], { type: 'image/svg+xml;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  try {
    const img = new Image()
    img.decoding = 'sync'
    await new Promise<void>((resolve, reject) => {
      img.onload = () => resolve()
      img.onerror = () => reject(new Error('SVG slide gagal dirender'))
      img.src = url
    })
    const canvas = document.createElement('canvas')
    canvas.width = VW * SCALE
    canvas.height = VH * SCALE
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('Canvas tidak tersedia')
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
    return canvas.toDataURL('image/png')
  } finally {
    URL.revokeObjectURL(url)
  }
}

/** XML transisi PowerPoint per jenis animasi slide. */
function transitionXml(kind: DeckSlide['transition']): string {
  const inner =
    kind === 'fade'
      ? '<p:fade/>'
      : kind === 'push'
        ? '<p:push dir="u"/>'
        : kind === 'wipe'
          ? '<p:wipe dir="r"/>'
          : kind === 'split'
            ? '<p:split orient="horz" dir="out"/>'
            : '<p:zoom dir="in"/>'
  return `<p:transition spd="med">${inner}</p:transition>`
}

/**
 * Sisipkan elemen <p:transition> ke tiap slide XML di dalam paket .pptx.
 * pptxgenjs tidak punya API transisi, jadi paket di-repack: elemen ditaruh
 * tepat sebelum </p:sld> — posisi itu sudah sesuai urutan skema OOXML
 * (setelah <p:clrMapOvr>, sebelum <p:timing>).
 */
async function injectTransitions(pptxBlob: Blob, deck: DeckSlide[]): Promise<Blob> {
  const { default: JSZip } = await import('jszip')
  const zip = await JSZip.loadAsync(pptxBlob)

  await Promise.all(
    deck.map(async (slide, i) => {
      const path = `ppt/slides/slide${i + 1}.xml`
      const file = zip.file(path)
      if (!file) return
      const xml = await file.async('string')
      if (xml.includes('<p:transition')) return
      zip.file(path, xml.replace('</p:sld>', `${transitionXml(slide.transition)}</p:sld>`))
    }),
  )

  return zip.generateAsync({
    type: 'blob',
    mimeType: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    compression: 'DEFLATE',
  })
}

function triggerDownload(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
  // Beri browser waktu memulai unduhan sebelum URL dilepas.
  setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

/**
 * Rakit paket .pptx di memori. Dipisah dari aksi unduh supaya bisa diuji.
 * Tiap slide dirancang AI (layout dipilih per topik), digambar sebagai SVG
 * oleh playbookDeck, lalu diraster penuh ke PNG 2× dan dipasang full-bleed —
 * sehingga file .pptx persis sama dengan preview di aplikasi.
 */
export async function buildPptxBlob(
  content: PlaybookContent,
  fallbackTitle: string,
): Promise<{ blob: Blob; deck: DeckSlide[]; fileName: string }> {
  const deck = buildDeck(content, fallbackTitle)
  const title = (content.title || fallbackTitle || 'Playbook Strategis').trim()

  const { default: PptxGen } = await import('pptxgenjs')
  const pptx = new PptxGen()
  pptx.defineLayout({ name: 'DECK169', width: W_IN, height: H_IN })
  pptx.layout = 'DECK169'
  pptx.title = title
  pptx.company = 'SalesFlow'

  // Raster paralel — jauh lebih cepat daripada berurutan untuk deck 10+ slide.
  const images = await Promise.all(deck.map((s) => svgToPng(s.svg)))

  images.forEach((data, i) => {
    const slide = pptx.addSlide()
    slide.addImage({ data, x: 0, y: 0, w: W_IN, h: H_IN })
    slide.addNotes(deck[i].title)
  })

  const raw = (await pptx.write({ outputType: 'blob' })) as Blob
  const blob = await injectTransitions(raw, deck)
  return { blob, deck, fileName: pptFileName(title) }
}

/** Ekspor playbook menjadi file PowerPoint dan mulai unduhan. */
export async function exportPlaybookPpt(content: PlaybookContent, fallbackTitle: string) {
  const { blob, fileName } = await buildPptxBlob(content, fallbackTitle)
  triggerDownload(blob, fileName)
}
