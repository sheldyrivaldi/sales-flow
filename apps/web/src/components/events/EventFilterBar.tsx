import { useMemo } from 'react'
import { Search, Plus, X, Check } from 'lucide-react'

import Popover from '../ui/Popover'
import Button from '../ui/Button'
import Input from '../ui/Input'
import DatePicker from '../ui/DatePicker'
import { cn } from '../../lib/cn'
import { EVENT_TYPE_LABELS, EVENT_STATUS_LABELS } from '../../api/events'
import type { EventFilters, EventType, EventStatus } from '../../api/events'

/**
 * Filter multi-kolom bergaya Jira.
 *
 * Pola yang diikuti (lihat Atlassian "Filter work items"):
 * - satu kolom boleh dipilih BEBERAPA nilai sekaligus → OR di dalam kolom,
 *   AND antar kolom;
 * - kolom ditambahkan sesuai kebutuhan lewat menu "Filter", bukan semuanya
 *   ditampilkan sekaligus;
 * - tiap filter aktif tampil sebagai chip dengan tombol (x) sendiri;
 * - "Bersihkan semua" muncul hanya ketika ada filter aktif;
 * - hasil diperbarui seketika, tanpa tombol "Terapkan".
 */

const TYPE_VALUES: EventType[] = ['EXPO', 'CONFERENCE', 'SEMINAR', 'WORKSHOP', 'NETWORKING', 'OTHER']
const STATUS_VALUES: EventStatus[] = ['PLANNED', 'ATTENDED', 'CANCELLED']

export interface EventFilterBarProps {
  filters: EventFilters
  onChange: (next: EventFilters) => void
  /** Jumlah hasil saat ini — ditampilkan agar dampak filter terasa langsung. */
  resultCount?: number
}

/** Satu baris pilihan multi-select di dalam popover. */
function CheckRow({
  label,
  checked,
  onToggle,
}: {
  label: string
  checked: boolean
  onToggle: () => void
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="flex items-center gap-2 w-full px-2 py-1.5 rounded-btn text-left text-body text-fg hover:bg-surface-subtle transition-colors"
    >
      <span
        className={cn(
          'inline-flex items-center justify-center w-4 h-4 rounded border shrink-0 transition-colors',
          checked ? 'bg-primary border-primary text-white' : 'border-line bg-surface',
        )}
      >
        {checked && <Check className="w-3 h-3" aria-hidden="true" />}
      </span>
      {label}
    </button>
  )
}

function FilterChip({ label, onRemove }: { label: string; onRemove: () => void }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-pill bg-primary-subtle border border-primary/25 pl-2.5 pr-1 py-1 text-caption text-fg animate-chip-in">
      {label}
      <button
        type="button"
        aria-label={`Hapus filter ${label}`}
        onClick={onRemove}
        className="p-0.5 rounded-full text-fg-muted hover:text-danger hover:bg-danger-subtle transition-colors"
      >
        <X className="w-3 h-3" aria-hidden="true" />
      </button>
    </span>
  )
}

export default function EventFilterBar({ filters, onChange, resultCount }: EventFilterBarProps) {
  const types = filters.type ?? []
  const statuses = filters.status ?? []

  function patch(p: Partial<EventFilters>) {
    onChange({ ...filters, ...p, page: 1 })
  }

  function toggleType(t: EventType) {
    patch({ type: types.includes(t) ? types.filter((x) => x !== t) : [...types, t] })
  }

  function toggleStatus(s: EventStatus) {
    patch({ status: statuses.includes(s) ? statuses.filter((x) => x !== s) : [...statuses, s] })
  }

  /** Chip untuk tiap filter aktif — sumber tunggal supaya jumlah & tampilan konsisten. */
  const chips = useMemo(() => {
    const out: { key: string; label: string; clear: () => void }[] = []

    for (const t of types) {
      out.push({
        key: `type-${t}`,
        label: `Tipe: ${EVENT_TYPE_LABELS[t]}`,
        clear: () => patch({ type: types.filter((x) => x !== t) }),
      })
    }
    for (const s of statuses) {
      out.push({
        key: `status-${s}`,
        label: `Status: ${EVENT_STATUS_LABELS[s]}`,
        clear: () => patch({ status: statuses.filter((x) => x !== s) }),
      })
    }
    if (filters.date_from) {
      out.push({ key: 'from', label: `Dari: ${filters.date_from}`, clear: () => patch({ date_from: undefined }) })
    }
    if (filters.date_to) {
      out.push({ key: 'to', label: `Sampai: ${filters.date_to}`, clear: () => patch({ date_to: undefined }) })
    }
    if (filters.location) {
      out.push({ key: 'loc', label: `Lokasi: ${filters.location}`, clear: () => patch({ location: undefined }) })
    }
    if (filters.organizer) {
      out.push({ key: 'org', label: `Penyelenggara: ${filters.organizer}`, clear: () => patch({ organizer: undefined }) })
    }
    if (filters.has_attachment !== undefined) {
      out.push({
        key: 'att',
        label: filters.has_attachment ? 'Punya lampiran' : 'Tanpa lampiran',
        clear: () => patch({ has_attachment: undefined }),
      })
    }
    if (filters.has_participant !== undefined) {
      out.push({
        key: 'par',
        label: filters.has_participant ? 'Punya peserta' : 'Tanpa peserta',
        clear: () => patch({ has_participant: undefined }),
      })
    }
    return out
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters])

  const hasAny = chips.length > 0 || !!filters.search

  function clearAll() {
    onChange({ page: 1, page_size: filters.page_size })
  }

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex flex-wrap items-center gap-2">
        {/* Pencarian menyapu nama, penyelenggara, lokasi, dan catatan sekaligus. */}
        <div className="relative w-64">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-fg-subtle pointer-events-none" aria-hidden="true" />
          <Input
            className="pl-8"
            placeholder="Cari nama, penyelenggara, lokasi…"
            value={filters.search ?? ''}
            onChange={(e) => patch({ search: e.target.value || undefined })}
          />
        </div>

        {/* Tipe — multi pilih */}
        <Popover
          align="start"
          trigger={
            <Button variant="secondary" size="sm">
              Tipe{types.length > 0 && ` (${types.length})`}
            </Button>
          }
        >
          <div className="w-52 p-1">
            {TYPE_VALUES.map((t) => (
              <CheckRow key={t} label={EVENT_TYPE_LABELS[t]} checked={types.includes(t)} onToggle={() => toggleType(t)} />
            ))}
          </div>
        </Popover>

        {/* Status — multi pilih */}
        <Popover
          align="start"
          trigger={
            <Button variant="secondary" size="sm">
              Status{statuses.length > 0 && ` (${statuses.length})`}
            </Button>
          }
        >
          <div className="w-52 p-1">
            {STATUS_VALUES.map((s) => (
              <CheckRow key={s} label={EVENT_STATUS_LABELS[s]} checked={statuses.includes(s)} onToggle={() => toggleStatus(s)} />
            ))}
          </div>
        </Popover>

        {/* Kolom lain disembunyikan di balik satu tombol agar bar tetap ringkas. */}
        <Popover
          align="start"
          trigger={
            <Button variant="secondary" size="sm" leftIcon={<Plus className="w-3.5 h-3.5" />}>
              Filter
            </Button>
          }
        >
          <div className="w-[min(20rem,calc(100vw-1.5rem))] p-3 flex flex-col gap-3">
            <div className="flex flex-col gap-1.5">
              <span className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Rentang Tanggal</span>
              {/* Dua kolom grid, BUKAN flex: input[type=date] punya lebar
                  intrinsik bawaan browser dan flex item ber-min-width:auto
                  menolak menyusut di bawahnya, sehingga input kedua meluber
                  keluar popover. Kolom grid 1fr + min-w-0 memaksa keduanya
                  berbagi lebar secara merata. Label "Dari"/"Sampai" dipakai
                  menggantikan pemisah "–" agar hemat tempat sekaligus jelas. */}
              <div className="grid grid-cols-2 gap-2">
                <div className="flex flex-col gap-1 min-w-0">
                  <label htmlFor="ev-f-from" className="text-caption text-fg-subtle">Dari</label>
                  <DatePicker
                    id="ev-f-from"
                    className="min-w-0 px-2"
                    value={filters.date_from ?? ''}
                    onChange={(e) => patch({ date_from: e.target.value || undefined })}
                  />
                </div>
                <div className="flex flex-col gap-1 min-w-0">
                  <label htmlFor="ev-f-to" className="text-caption text-fg-subtle">Sampai</label>
                  <DatePicker
                    id="ev-f-to"
                    className="min-w-0 px-2"
                    value={filters.date_to ?? ''}
                    onChange={(e) => patch({ date_to: e.target.value || undefined })}
                  />
                </div>
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <span className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Lokasi</span>
              <Input
                placeholder="mis. Jakarta"
                value={filters.location ?? ''}
                onChange={(e) => patch({ location: e.target.value || undefined })}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <span className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Penyelenggara</span>
              <Input
                placeholder="mis. Dyandra"
                value={filters.organizer ?? ''}
                onChange={(e) => patch({ organizer: e.target.value || undefined })}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <span className="text-caption font-semibold text-fg-muted uppercase tracking-wide">Kelengkapan</span>
              <CheckRow
                label="Punya lampiran"
                checked={filters.has_attachment === true}
                onToggle={() => patch({ has_attachment: filters.has_attachment === true ? undefined : true })}
              />
              <CheckRow
                label="Tanpa lampiran"
                checked={filters.has_attachment === false}
                onToggle={() => patch({ has_attachment: filters.has_attachment === false ? undefined : false })}
              />
              <CheckRow
                label="Punya peserta"
                checked={filters.has_participant === true}
                onToggle={() => patch({ has_participant: filters.has_participant === true ? undefined : true })}
              />
            </div>
          </div>
        </Popover>

        {hasAny && (
          <Button variant="ghost" size="sm" onClick={clearAll}>
            Bersihkan semua
          </Button>
        )}

        {resultCount !== undefined && (
          <span className="text-caption text-fg-subtle ml-auto tabular-nums">{resultCount} event</span>
        )}
      </div>

      {chips.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5">
          {chips.map((c) => (
            <FilterChip key={c.key} label={c.label} onRemove={c.clear} />
          ))}
        </div>
      )}
    </div>
  )
}
