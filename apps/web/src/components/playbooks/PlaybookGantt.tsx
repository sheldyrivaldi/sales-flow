import { cn } from '../../lib/cn'
import type { PlaybookTimelineItem } from '../../api/playbooks'

/** Gantt sederhana untuk rencana kerja playbook: satu baris per aktivitas,
 * bar berdasarkan start_day/duration_days, penggaris hari/minggu di atas.
 * Murni CSS (tanpa library chart) supaya ringan dan konsisten token. */
export default function PlaybookGantt({ items }: { items: PlaybookTimelineItem[] }) {
  if (items.length === 0) return null

  const totalDays = Math.max(...items.map((i) => i.start_day + Math.max(i.duration_days, 1)), 7)
  // Tick mingguan (setiap 7 hari), minimal penggaris 1 minggu.
  const weeks = Math.ceil(totalDays / 7)

  return (
    <div className="flex flex-col gap-1 overflow-x-auto scrollbar-thin">
      <div className="min-w-[480px]">
        {/* Penggaris minggu */}
        <div className="flex ml-40 border-b border-line text-caption text-fg-subtle select-none">
          {Array.from({ length: weeks }).map((_, w) => (
            <div
              key={w}
              className="border-l border-line pl-1 py-0.5"
              style={{ width: `${(7 / totalDays) * 100}%` }}
            >
              M{w + 1}
            </div>
          ))}
        </div>

        {/* Baris aktivitas */}
        <div className="flex flex-col">
          {items.map((item, i) => {
            const left = (item.start_day / totalDays) * 100
            const width = (Math.max(item.duration_days, 1) / totalDays) * 100
            return (
              <div key={i} className="flex items-center gap-0 py-1 group">
                <div
                  className="w-40 shrink-0 pr-2 text-caption text-fg truncate"
                  title={item.activity}
                >
                  {item.activity}
                </div>
                <div className="relative flex-1 h-5 rounded bg-surface-subtle/60">
                  <div
                    className={cn(
                      'absolute top-0.5 bottom-0.5 rounded-pill bg-primary/80 group-hover:bg-primary transition-colors',
                      'flex items-center justify-center'
                    )}
                    style={{ left: `${left}%`, width: `${width}%`, minWidth: 10 }}
                    title={`Hari ${item.start_day + 1} — ${item.duration_days} hari`}
                  >
                    {width > 12 && (
                      <span className="text-[10px] font-semibold text-white truncate px-1">
                        {item.duration_days}h
                      </span>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
