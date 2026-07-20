import { useState } from 'react'
import { Paperclip, Trash2 } from 'lucide-react'

import Card, { CardHeader, CardBody } from '../ui/Card'
import Button from '../ui/Button'
import ConfirmDialog from '../ui/ConfirmDialog'
import EventAttachmentsInput from './EventAttachmentsInput'
import { toast } from '../../lib/toast'
import { useUpdateEvent } from '../../api/events'
import type { EventAttachment } from '../../api/events'
import { extOf, isViewableInBrowser, kindLabel, formatBytes, downloadFile } from '../../lib/fileKind'

export interface EventAttachmentsSectionProps {
  eventId: string
  attachments: EventAttachment[]
  /** true saat analisa berjalan — tambah/hapus dilarang agar bahan analisa
   * tidak berubah di tengah proses. */
  locked?: boolean
}

/**
 * Kelola lampiran langsung dari halaman detail: lihat, tambah, hapus.
 *
 * Perubahan disimpan SEKETIKA lewat PUT event, bukan menunggu tombol simpan —
 * di halaman detail tidak ada konteks "form" yang sedang diisi, jadi menahan
 * perubahan justru membingungkan.
 */
export default function EventAttachmentsSection({ eventId, attachments, locked }: EventAttachmentsSectionProps) {
  const [adding, setAdding] = useState(false)
  const [removeTarget, setRemoveTarget] = useState<EventAttachment | null>(null)
  const update = useUpdateEvent()

  async function persist(next: EventAttachment[], successMsg: string) {
    try {
      await update.mutateAsync({ id: eventId, body: { attachments: next } })
      toast.success(successMsg)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal menyimpan lampiran.')
    }
  }

  return (
    <Card>
      <CardHeader className="flex items-center gap-2">
        <Paperclip className="w-4 h-4 text-primary" aria-hidden="true" />
        <h3 className="text-body font-semibold text-fg">Lampiran</h3>
        <span className="text-caption text-fg-subtle">{attachments.length} berkas</span>
        <Button
          size="sm"
          variant={adding ? 'ghost' : 'secondary'}
          className="ml-auto"
          disabled={locked}
          title={locked ? 'Terkunci selama analisa berjalan' : undefined}
          onClick={() => setAdding((v) => !v)}
        >
          {locked ? 'Terkunci' : adding ? 'Tutup' : 'Tambah Berkas'}
        </Button>
      </CardHeader>

      <CardBody className="flex flex-col gap-3">
        {adding && !locked && (
          <EventAttachmentsInput
            value={attachments}
            onChange={(next) => {
              // Komponen unggah mengembalikan daftar LENGKAP setelah berkas baru
              // selesai diunggah, jadi cukup disimpan apa adanya.
              void persist(next, 'Lampiran ditambahkan.')
            }}
            disabled={update.isPending}
          />
        )}

        {attachments.length === 0 ? (
          <p className="text-caption text-fg-muted">
            Belum ada lampiran. Tambahkan rundown, undangan, materi, atau denah booth.
          </p>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {attachments.map((a, i) => {
              const viewable = isViewableInBrowser(a.name, a.mime)
              return (
                <li
                  key={`${a.url}-${i}`}
                  className="flex items-center gap-2.5 rounded-card border border-line bg-surface px-3 py-2 hover:border-primary-border transition-colors animate-row-in"
                >
                  <span className="inline-flex items-center justify-center w-8 h-8 rounded-btn bg-primary-subtle text-primary text-[10px] font-bold shrink-0">
                    {extOf(a.name).slice(0, 4).toUpperCase() || 'FILE'}
                  </span>
                  <div className="flex-1 min-w-0">
                    <p className="text-body text-fg truncate" title={a.name}>{a.name}</p>
                    <p className="text-caption text-fg-subtle">
                      {kindLabel(a.name)}
                      {formatBytes(a.size) && ` · ${formatBytes(a.size)}`}
                      {!viewable && ' · perlu diunduh'}
                    </p>
                  </div>
                  {viewable && (
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => window.open(a.url, '_blank', 'noopener,noreferrer')}
                    >
                      Buka
                    </Button>
                  )}
                  <Button size="sm" variant="ghost" onClick={() => downloadFile(a.url, a.name)}>
                    Unduh
                  </Button>
                  <button
                    type="button"
                    aria-label={`Hapus ${a.name}`}
                    disabled={update.isPending || locked}
                    onClick={() => setRemoveTarget(a)}
                    className="p-1.5 rounded-btn text-fg-subtle hover:text-danger hover:bg-danger-subtle transition-colors shrink-0"
                  >
                    <Trash2 className="w-3.5 h-3.5" aria-hidden="true" />
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </CardBody>

      <ConfirmDialog
        open={!!removeTarget}
        title="Hapus lampiran?"
        description={`"${removeTarget?.name}" akan dilepas dari event ini.`}
        confirmLabel="Hapus"
        tone="danger"
        loading={update.isPending}
        onConfirm={() => {
          const next = attachments.filter((a) => a.url !== removeTarget?.url)
          setRemoveTarget(null)
          void persist(next, 'Lampiran dihapus.')
        }}
        onCancel={() => setRemoveTarget(null)}
      />
    </Card>
  )
}
