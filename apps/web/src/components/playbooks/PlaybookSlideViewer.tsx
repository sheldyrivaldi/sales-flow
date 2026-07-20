import { useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, Pencil } from 'lucide-react'
import { cn } from '../../lib/cn'
import { buildDeck } from '../../lib/playbookDeck'
import type { PlaybookContent } from '../../api/playbooks'

export interface PlaybookSlideViewerProps {
  content: PlaybookContent
  fallbackTitle: string
  /** Dipanggil saat user menekan "Edit slide ini" — mengarahkan prompt revisi
   * ke seksi terkait. */
  onEditSection?: (section: string) => void
}

/**
 * Preview deck slide-per-slide. Menampilkan SVG yang PERSIS SAMA dengan yang
 * dipakai eksportir .pptx (lihat lib/playbookDeck), jadi apa yang dilihat di
 * sini adalah apa yang keluar di file PowerPoint.
 */
export default function PlaybookSlideViewer({ content, fallbackTitle, onEditSection }: PlaybookSlideViewerProps) {
  const slides = useMemo(() => buildDeck(content, fallbackTitle), [content, fallbackTitle])

  const [idx, setIdx] = useState(0)
  // Deck bisa menyusut setelah revisi — jepit saat baca, bukan lewat effect.
  const safeIdx = Math.min(idx, Math.max(slides.length - 1, 0))
  const cur = slides[safeIdx]
  if (!cur) return null

  return (
    <div className="flex flex-col gap-3">
      {/* Kanvas slide 16:9 — SVG identik dengan hasil ekspor */}
      <div
        key={cur.key}
        className={cn(
          'relative w-full rounded-card overflow-hidden border border-line shadow-elevated bg-white',
          'animate-slide-enter [&>svg]:block [&>svg]:w-full [&>svg]:h-auto',
        )}
        style={{ aspectRatio: '16 / 9' }}
        dangerouslySetInnerHTML={{ __html: cur.svg }}
      />

      {/* Kontrol */}
      <div className="flex items-center gap-2">
        <button
          type="button"
          aria-label="Slide sebelumnya"
          disabled={safeIdx === 0}
          onClick={() => setIdx((i) => Math.max(0, i - 1))}
          className="p-1.5 rounded-btn border border-line text-fg-muted hover:bg-surface-subtle disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <ChevronLeft className="w-4 h-4" aria-hidden="true" />
        </button>
        <span className="text-caption text-fg-muted tabular-nums">
          {safeIdx + 1} / {slides.length}
        </span>
        <button
          type="button"
          aria-label="Slide berikutnya"
          disabled={safeIdx >= slides.length - 1}
          onClick={() => setIdx((i) => Math.min(slides.length - 1, i + 1))}
          className="p-1.5 rounded-btn border border-line text-fg-muted hover:bg-surface-subtle disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <ChevronRight className="w-4 h-4" aria-hidden="true" />
        </button>

        <span className="text-caption text-fg-subtle truncate ml-1 flex-1">{cur.title}</span>

        {onEditSection && (
          <button
            type="button"
            onClick={() => onEditSection(cur.section)}
            className="inline-flex items-center gap-1 text-caption text-primary hover:underline"
          >
            <Pencil className="w-3.5 h-3.5" aria-hidden="true" /> Edit slide ini
          </button>
        )}
      </div>

      {/* Strip thumbnail — render SVG mini supaya variasi layout terlihat */}
      <div className="flex gap-1.5 overflow-x-auto scrollbar-thin pb-1">
        {slides.map((s, i) => (
          <button
            key={s.key}
            type="button"
            onClick={() => setIdx(i)}
            title={s.title}
            aria-label={`Slide ${i + 1}: ${s.title}`}
            className={cn(
              'shrink-0 w-24 rounded overflow-hidden border transition-colors',
              '[&>svg]:block [&>svg]:w-full [&>svg]:h-auto',
              i === safeIdx ? 'border-primary ring-1 ring-primary' : 'border-line hover:border-primary-border',
            )}
            style={{ aspectRatio: '16 / 9' }}
            dangerouslySetInnerHTML={{ __html: s.svg }}
          />
        ))}
      </div>
    </div>
  )
}
