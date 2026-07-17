import { useDraggable } from '@dnd-kit/core'

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
  className?: string
}

/** Tampilan murni kartu prospek — dipakai kartu draggable di kolom DAN clone
 * yang mengikuti kursor di <DragOverlay> (behavior board ala Jira). */
export function ProspectCardView({ prospect, ownerName, onClick, className }: ProspectCardProps) {
  const score: number | null = null // TODO(EP-10): isi dari prospect_score

  return (
    <button
      type="button"
      onClick={onClick}
      tabIndex={onClick ? 0 : -1}
      className={cn(
        'w-full text-left bg-surface border border-line rounded-card p-3 flex flex-col gap-2 shadow-subtle hover-lift hover:border-primary-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
        className
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
  )
}

/** Kartu prospect untuk kolom Kanban. Draggable antar-kolom via dnd-kit —
 * saat di-drag, kartu asal meredup + sedikit mengecil dan clone-nya mengikuti
 * kursor lewat <DragOverlay> di ProspectBoard (tanpa transform di sumber). */
export default function ProspectCard({ prospect, ownerName, onClick, className }: ProspectCardProps) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: prospect.id,
  })

  return (
    <div ref={setNodeRef} {...listeners} {...attributes} className="touch-none">
      <ProspectCardView
        prospect={prospect}
        ownerName={ownerName}
        onClick={onClick}
        className={cn(isDragging && 'opacity-40 scale-[.98]', className)}
      />
    </div>
  )
}
