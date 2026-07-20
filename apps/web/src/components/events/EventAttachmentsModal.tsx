import { useState } from 'react'
import {
  ExternalLink,
  Download,
  FileText,
  Image as ImageIcon,
  Sheet,
  File as FileIcon,
  Film,
  Music,
  Archive,
  Presentation,
} from 'lucide-react'

import Modal from '../ui/Modal'
import Button from '../ui/Button'
import ConfirmDialog from '../ui/ConfirmDialog'
import EmptyState from '../ui/EmptyState'
import { extOf, isViewableInBrowser, kindLabel, formatBytes, downloadFile } from '../../lib/fileKind'
import type { EventAttachment } from '../../api/events'

/** Ikon per keluarga berkas supaya daftar bisa dipindai sekilas. */
function iconFor(name: string) {
  const e = extOf(name)
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'avif', 'bmp', 'ico'].includes(e)) return ImageIcon
  if (['xlsx', 'xls', 'csv'].includes(e)) return Sheet
  if (['ppt', 'pptx'].includes(e)) return Presentation
  if (['mp4', 'webm', 'mov', 'avi', 'mkv'].includes(e)) return Film
  if (['mp3', 'wav', 'ogg', 'm4a'].includes(e)) return Music
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(e)) return Archive
  if (['pdf', 'doc', 'docx', 'txt', 'md', 'html', 'htm', 'json', 'xml'].includes(e)) return FileText
  return FileIcon
}

export interface EventAttachmentsModalProps {
  open: boolean
  onClose: () => void
  eventName: string
  attachments: EventAttachment[]
}

/**
 * Daftar lampiran sebuah event.
 *
 * Aturan membuka: berkas yang didukung browser (PDF, HTML, gambar, teks,
 * media) dibuka di tab baru. Yang tidak didukung — mis. .docx, .xlsx, .zip —
 * TIDAK dibuka diam-diam, karena browser hanya akan mengunduhnya atau
 * menampilkan halaman kosong. Untuk itu muncul konfirmasi lebih dulu supaya
 * user memilih sendiri mau mengunduh atau batal.
 */
export default function EventAttachmentsModal({
  open,
  onClose,
  eventName,
  attachments,
}: EventAttachmentsModalProps) {
  const [confirmTarget, setConfirmTarget] = useState<EventAttachment | null>(null)

  function handleOpen(a: EventAttachment) {
    if (isViewableInBrowser(a.name, a.mime)) {
      window.open(a.url, '_blank', 'noopener,noreferrer')
      return
    }
    setConfirmTarget(a)
  }

  return (
    <>
      <Modal
        open={open}
        onClose={onClose}
        size="lg"
        title={`Lampiran — ${eventName}`}
        footer={
          <div className="flex items-center justify-between gap-3 w-full">
            <span className="text-caption text-fg-subtle">
              {attachments.length} berkas
            </span>
            <Button variant="secondary" onClick={onClose}>
              Tutup
            </Button>
          </div>
        }
      >
        {attachments.length === 0 ? (
          <EmptyState
            icon={<FileIcon className="w-6 h-6" />}
            title="Belum ada lampiran"
            description="Tambahkan rundown, undangan, atau denah booth lewat menu Edit event."
          />
        ) : (
          <ul className="flex flex-col gap-2">
            {attachments.map((a, i) => {
              const Icon = iconFor(a.name)
              const viewable = isViewableInBrowser(a.name, a.mime)
              return (
                <li
                  key={`${a.url}-${i}`}
                  className="flex items-center gap-3 rounded-card border border-line bg-surface px-3 py-2.5 hover:border-primary-border transition-colors animate-row-in"
                >
                  <span className="inline-flex items-center justify-center w-9 h-9 rounded-btn bg-primary-subtle text-primary shrink-0">
                    <Icon className="w-4 h-4" aria-hidden="true" />
                  </span>

                  <div className="flex-1 min-w-0">
                    <p className="text-body text-fg truncate" title={a.name}>
                      {a.name}
                    </p>
                    <p className="text-caption text-fg-subtle">
                      {kindLabel(a.name)}
                      {formatBytes(a.size) && ` · ${formatBytes(a.size)}`}
                      {!viewable && ' · perlu diunduh'}
                    </p>
                  </div>

                  <div className="flex items-center gap-1 shrink-0">
                    <Button
                      size="sm"
                      variant="ghost"
                      leftIcon={<ExternalLink className="w-3.5 h-3.5" />}
                      onClick={() => handleOpen(a)}
                    >
                      Buka
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label={`Unduh ${a.name}`}
                      leftIcon={<Download className="w-3.5 h-3.5" />}
                      onClick={() => downloadFile(a.url, a.name)}
                    >
                      Unduh
                    </Button>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </Modal>

      {/* Jenis berkas yang tidak bisa ditampilkan browser — user yang memutuskan. */}
      <ConfirmDialog
        open={!!confirmTarget}
        title="Buka di browser tidak didukung"
        description={`Berkas ${kindLabel(confirmTarget?.name ?? '')} seperti "${confirmTarget?.name ?? ''}" tidak bisa ditampilkan langsung di browser. Unduh berkasnya sekarang?`}
        confirmLabel="Unduh"
        tone="primary"
        onConfirm={() => {
          if (confirmTarget) downloadFile(confirmTarget.url, confirmTarget.name)
          setConfirmTarget(null)
        }}
        onCancel={() => setConfirmTarget(null)}
      />
    </>
  )
}
