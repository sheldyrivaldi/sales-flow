import { FileText, FileImage, File as FileIcon } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface AttachmentPreviewProps {
  url: string
  name?: string
  mime?: string
  /** Selaraskan ke kanan untuk pesan user, kiri untuk balasan AI. */
  align?: 'start' | 'end'
}

function isImage(mime?: string, url?: string): boolean {
  if (mime?.startsWith('image/')) return true
  return /\.(png|jpe?g|webp|gif|svg)$/i.test(url ?? '')
}

/** Preview lampiran yang bisa diklik untuk membuka file di tab baru. Gambar
 * tampil sebagai thumbnail; jenis lain tampil sebagai chip berikon dengan
 * nama file. Dipakai untuk lampiran user maupun file yang dikembalikan AI. */
export default function AttachmentPreview({ url, name, mime, align = 'start' }: AttachmentPreviewProps) {
  const label = name ?? url.split('/').pop() ?? 'Lampiran'
  const image = isImage(mime, url)
  const pdf = mime === 'application/pdf' || /\.pdf$/i.test(url)

  if (image) {
    return (
      <a
        href={url}
        target="_blank"
        rel="noopener noreferrer"
        title={`Buka ${label}`}
        className={cn(
          'block max-w-[220px] overflow-hidden rounded-card border border-line bg-surface',
          'transition-shadow hover:shadow-elevated focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
          align === 'end' ? 'self-end' : 'self-start',
        )}
      >
        <img src={url} alt={label} className="max-h-56 w-full object-cover" loading="lazy" />
      </a>
    )
  }

  const Icon = pdf ? FileText : mime?.startsWith('image/') ? FileImage : FileIcon
  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      title={`Buka ${label}`}
      className={cn(
        'inline-flex items-center gap-2 rounded-btn border border-line bg-surface px-3 py-2 max-w-[240px]',
        'text-body text-fg transition-colors hover:border-primary-border hover:bg-surface-subtle',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
        align === 'end' ? 'self-end' : 'self-start',
      )}
    >
      <span className={cn('shrink-0 grid place-items-center w-8 h-8 rounded-btn', pdf ? 'bg-danger-subtle text-danger' : 'bg-primary-subtle text-primary')}>
        <Icon className="w-4 h-4" aria-hidden="true" />
      </span>
      <span className="min-w-0 flex flex-col">
        <span className="truncate font-medium">{label}</span>
        <span className="text-caption text-fg-subtle">Klik untuk membuka</span>
      </span>
    </a>
  )
}
