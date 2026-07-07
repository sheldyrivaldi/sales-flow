import { useDraggable } from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'

import Badge from '../ui/Badge'
import ScoreRing from '../ui/ScoreRing'
import Avatar from '../ui/Avatar'
import { formatRupiahShort } from '../../lib/format'
import { cn } from '../../lib/cn'
import type { Prospect } from '../../api/prospects'
import { SOURCE_LABELS } from '../../api/prospects'

export interface ProspectCardProps {
  prospect: Prospect
  /** Nama owner terselesaikan (dari useUsers()) — fallback ke owner_user_id
   * mentah hanya bila direktori user belum termuat. */
  ownerName?: string
  onClick?: () => void
}

/** Kartu prospect untuk kolom Kanban (Design §4.8). Skor (`ScoreRing`) baru
 * tersedia setelah EP-10 mengisi `prospect_score` — dirender kondisional.
 * Draggable antar-kolom via dnd-kit (ST-07.3.2). */
export default function ProspectCard({ prospect, ownerName, onClick }: ProspectCardProps) {
  const score: number | null = null // TODO(EP-10): isi dari prospect_score

  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: prospect.id,
  })
  const style = {
    transform: CSS.Translate.toString(transform),
  }

  return (
    <div ref={setNodeRef} style={style} {...listeners} {...attributes} className="touch-none">
      <button
        type="button"
        onClick={onClick}
        className={cn(
          'w-full text-left bg-surface border border-line rounded-card p-3 flex flex-col gap-2 shadow-subtle hover:border-primary/40 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
          isDragging && 'opacity-50'
        )}
      >
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <p className="text-body font-medium text-fg truncate">{prospect.name}</p>
            {prospect.company && (
              <p className="text-caption text-fg-muted truncate">{prospect.company}</p>
            )}
          </div>
          {score != null && <ScoreRing score={score} size={32} strokeWidth={4} showLabel />}
        </div>

        <div className="flex items-center justify-between gap-2">
          <Badge tone="info" appearance="soft">
            {SOURCE_LABELS[prospect.source_type]}
          </Badge>
          {prospect.owner_user_id && (
            <Avatar name={ownerName ?? prospect.owner_user_id} size="sm" />
          )}
        </div>

        {prospect.est_value != null && (
          <p className="text-caption font-semibold text-fg tabular-nums">
            {formatRupiahShort(prospect.est_value)}
          </p>
        )}
      </button>
    </div>
  )
}

