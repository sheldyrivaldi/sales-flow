import { useState } from 'react'
import { Users, Mail, Send, Copy, AlertTriangle } from 'lucide-react'

import Card, { CardHeader, CardBody } from '../ui/Card'
import Button from '../ui/Button'
import EmailChipsInput from './EmailChipsInput'
import { toast } from '../../lib/toast'
import { useUpdateEvent } from '../../api/events'
import type { Event } from '../../api/events'
import { buildInvite, mailtoUrl, isMailtoTooLong } from '../../lib/eventInvite'

export interface EventParticipantsSectionProps {
  event: Event
  /** true saat analisa berjalan — daftar peserta ikut dikunci. */
  locked?: boolean
}

/**
 * Kelola daftar undangan langsung dari halaman detail, plus kirim undangan
 * lewat aplikasi email milik user.
 *
 * Undangan dibuka sebagai draft `mailto:` (Outlook dan sejenisnya), BUKAN
 * dikirim dari server: undangan harus datang dari alamat pengundangnya,
 * tercatat di folder Sent-nya, dan bisa disunting sebelum dikirim.
 */
export default function EventParticipantsSection({ event, locked }: EventParticipantsSectionProps) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<string[]>(event.participant_emails ?? [])
  const update = useUpdateEvent()

  const saved = event.participant_emails ?? []
  const dirty = editing && JSON.stringify(draft) !== JSON.stringify(saved)

  async function save() {
    try {
      await update.mutateAsync({ id: event.id, body: { participant_emails: draft } })
      toast.success('Daftar peserta diperbarui.')
      setEditing(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal menyimpan peserta.')
    }
  }

  function sendInvite() {
    if (saved.length === 0) {
      toast.error('Belum ada peserta yang diundang.')
      return
    }
    const invite = buildInvite(event)
    const url = mailtoUrl(invite)
    if (isMailtoTooLong(url)) {
      // Sebagian klien memotong mailto yang terlalu panjang tanpa memberi tahu.
      toast.error('Daftar peserta terlalu panjang untuk dibuka otomatis, salin alamatnya lalu tempel di email.')
      return
    }
    window.location.href = url
  }

  async function copyAll() {
    try {
      await navigator.clipboard.writeText(saved.join('; '))
      toast.success('Alamat peserta disalin.')
    } catch {
      toast.error('Gagal menyalin alamat.')
    }
  }

  return (
    <Card>
      <CardHeader className="flex items-center gap-2 flex-wrap">
        <Users className="w-4 h-4 text-primary" aria-hidden="true" />
        <h3 className="text-body font-semibold text-fg">Peserta Diundang</h3>
        <span className="text-caption text-fg-subtle">{saved.length} orang</span>

        <div className="ml-auto flex items-center gap-2 flex-wrap">
          {saved.length > 0 && (
            <>
              <Button size="sm" variant="ghost" leftIcon={<Copy className="w-3.5 h-3.5" />} onClick={() => void copyAll()}>
                Salin Alamat
              </Button>
              <Button size="sm" leftIcon={<Send className="w-3.5 h-3.5" />} onClick={sendInvite}>
                Kirim Undangan
              </Button>
            </>
          )}
          <Button
            size="sm"
            variant="secondary"
            disabled={locked}
            title={locked ? 'Terkunci selama analisa berjalan' : undefined}
            onClick={() => {
              setDraft(saved)
              setEditing((v) => !v)
            }}
          >
            {locked ? 'Terkunci' : editing ? 'Batal' : 'Kelola Peserta'}
          </Button>
        </div>
      </CardHeader>

      <CardBody className="flex flex-col gap-3">
        {editing && !locked ? (
          <>
            <EmailChipsInput value={draft} onChange={setDraft} disabled={update.isPending} />
            <div className="flex items-center gap-2">
              <Button size="sm" loading={update.isPending} disabled={!dirty} onClick={() => void save()}>
                Simpan Peserta
              </Button>
              {dirty && <span className="text-caption text-fg-subtle">Ada perubahan yang belum disimpan.</span>}
            </div>
          </>
        ) : saved.length === 0 ? (
          <p className="text-caption text-fg-muted">
            Belum ada peserta. Klik Kelola Peserta untuk menambahkan alamat email — peserta tidak perlu
            punya akun di aplikasi ini.
          </p>
        ) : (
          <>
            <div className="flex flex-wrap gap-1.5">
              {saved.map((email) => (
                <a
                  key={email}
                  href={`mailto:${email}`}
                  className="inline-flex items-center gap-1 rounded-pill bg-primary-subtle border border-primary/25 px-2 py-0.5 text-caption text-fg hover:border-primary transition-colors"
                >
                  <Mail className="w-3 h-3 text-primary" aria-hidden="true" />
                  {email}
                </a>
              ))}
            </div>
            <p className="inline-flex items-start gap-1.5 text-caption text-fg-subtle">
              <AlertTriangle className="w-3.5 h-3.5 mt-px shrink-0" aria-hidden="true" />
              Kirim Undangan membuka draft di aplikasi email kamu (mis. Outlook) dengan subjek, penerima,
              dan isi yang sudah tersusun — kamu tetap bisa menyuntingnya sebelum mengirim.
            </p>
          </>
        )}
      </CardBody>
    </Card>
  )
}
