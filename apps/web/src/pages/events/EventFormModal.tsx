import { useState } from 'react'
import { CalendarDays, Users, Paperclip, AlertCircle } from 'lucide-react'

import Modal from '../../components/ui/Modal'
import Button from '../../components/ui/Button'
import Input from '../../components/ui/Input'
import Select from '../../components/ui/Select'
import Textarea from '../../components/ui/Textarea'
import Field from '../../components/ui/Field'
import DatePicker from '../../components/ui/DatePicker'
import EmailChipsInput from '../../components/events/EmailChipsInput'
import EventAttachmentsInput from '../../components/events/EventAttachmentsInput'
import { toast } from '../../lib/toast'
import { useCreateEvent, useUpdateEvent } from '../../api/events'
import type { Event, EventType, EventStatus, EventAttachment } from '../../api/events'

interface Props {
  open: boolean
  onClose: () => void
  event?: Event
  onSaved: () => void
}

const TYPE_OPTIONS: { value: EventType; label: string }[] = [
  { value: 'EXPO', label: 'Expo' },
  { value: 'CONFERENCE', label: 'Conference' },
  { value: 'SEMINAR', label: 'Seminar' },
  { value: 'WORKSHOP', label: 'Workshop' },
  { value: 'NETWORKING', label: 'Networking' },
  { value: 'OTHER', label: 'Lainnya' },
]

const STATUS_OPTIONS: { value: EventStatus; label: string }[] = [
  { value: 'PLANNED', label: 'Direncanakan' },
  { value: 'ATTENDED', label: 'Dihadiri' },
  { value: 'CANCELLED', label: 'Dibatalkan' },
]

function toDateInputValue(isoString: string | null | undefined): string {
  if (!isoString) return ''
  return isoString.slice(0, 10)
}

/** Judul seksi kecil di dalam modal — memisahkan detail acara, undangan, dan
 * lampiran supaya form panjang tetap terbaca sebagai tiga bagian pendek. */
function SectionLabel({ icon: Icon, children }: { icon: typeof Users; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2 pt-1">
      <span className="inline-flex items-center justify-center w-6 h-6 rounded-md bg-primary-subtle text-primary shrink-0">
        <Icon className="w-3.5 h-3.5" aria-hidden="true" />
      </span>
      <span className="text-caption font-semibold text-fg uppercase tracking-wide">{children}</span>
      <span className="flex-1 h-px bg-line" />
    </div>
  )
}

/**
 * Pembungkus tipis. Form hanya dipasang saat modal terbuka dan diberi `key`
 * per-event, sehingga nilai awal cukup diisi lewat useState initializer — tanpa
 * useEffect yang menyalin props ke state (sumber cascading render sekaligus
 * bug "isian event lama masih tertinggal" saat modal dibuka ulang).
 */
export default function EventFormModal({ open, onClose, event, onSaved }: Props) {
  if (!open) return null
  return <EventFormDialog key={event?.id ?? 'new'} onClose={onClose} event={event} onSaved={onSaved} />
}

/** Form buat/ubah event dalam modal (bukan drawer): isinya pendek dan terpusat,
 * jadi dialog di tengah layar lebih fokus daripada panel samping. */
function EventFormDialog({ onClose, event, onSaved }: Omit<Props, 'open'>) {
  const isEdit = !!event

  const [name, setName] = useState(event?.name ?? '')
  const [type, setType] = useState<EventType>(event?.type ?? 'EXPO')
  const [date, setDate] = useState(toDateInputValue(event?.date))
  const [location, setLocation] = useState(event?.location ?? '')
  const [organizer, setOrganizer] = useState(event?.organizer ?? '')
  const [notes, setNotes] = useState(event?.notes ?? '')
  const [status, setStatus] = useState<EventStatus>(event?.status ?? 'PLANNED')
  const [emails, setEmails] = useState<string[]>(event?.participant_emails ?? [])
  const [attachments, setAttachments] = useState<EventAttachment[]>(event?.attachments ?? [])
  const [errors, setErrors] = useState<Record<string, string>>({})

  const createMutation = useCreateEvent()
  const updateMutation = useUpdateEvent()

  function validate(): boolean {
    const e: Record<string, string> = {}
    if (!name.trim()) e.name = 'Nama event wajib diisi'
    else if (name.trim().length < 3) e.name = 'Nama event minimal 3 karakter'
    if (!type) e.type = 'Tipe event wajib dipilih'
    // Status "Dihadiri" tanpa tanggal membuat laporan tidak bisa diurutkan.
    if (status === 'ATTENDED' && !date) e.date = 'Event yang sudah dihadiri wajib punya tanggal'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  async function handleSubmit(ev: React.FormEvent) {
    ev.preventDefault()
    if (!validate()) return

    const body = {
      name: name.trim(),
      type,
      date: date ? new Date(date).toISOString() : undefined,
      location: location.trim() || undefined,
      organizer: organizer.trim() || undefined,
      notes: notes.trim() || undefined,
      status,
      participant_emails: emails,
      attachments,
    }

    try {
      if (isEdit && event) {
        await updateMutation.mutateAsync({ id: event.id, body })
        toast.success('Event diperbarui.')
      } else {
        await createMutation.mutateAsync(body)
        toast.success('Event ditambahkan.')
      }
      onSaved()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : isEdit ? 'Gagal memperbarui event.' : 'Gagal menambahkan event.')
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending
  const errorCount = Object.keys(errors).length

  return (
    <Modal
      open
      onClose={onClose}
      size="xl"
      title={isEdit ? 'Edit Event' : 'Tambah Event Baru'}
      footer={
        <div className="flex items-center justify-between gap-3 w-full">
          {errorCount > 0 ? (
            <span className="inline-flex items-center gap-1.5 text-caption text-danger">
              <AlertCircle className="w-3.5 h-3.5" aria-hidden="true" />
              {errorCount === 1 ? '1 isian perlu diperbaiki' : `${errorCount} isian perlu diperbaiki`}
            </span>
          ) : (
            <span className="text-caption text-fg-subtle">
              {emails.length > 0 && `${emails.length} peserta`}
              {emails.length > 0 && attachments.length > 0 && ' · '}
              {attachments.length > 0 && `${attachments.length} lampiran`}
            </span>
          )}
          <div className="flex gap-2 shrink-0">
            <Button variant="secondary" onClick={onClose} disabled={isPending}>
              Batal
            </Button>
            <Button type="submit" form="event-form" loading={isPending}>
              {isEdit ? 'Simpan Perubahan' : 'Tambah Event'}
            </Button>
          </div>
        </div>
      }
    >
      <form id="event-form" onSubmit={handleSubmit} className="flex flex-col gap-4">
        <SectionLabel icon={CalendarDays}>Detail Acara</SectionLabel>

        <Field label="Nama Event" required htmlFor="ev-name" error={errors.name}>
          <Input
            id="ev-name"
            placeholder="Indo Security Expo 2026"
            value={name}
            onChange={(e) => setName(e.target.value)}
            invalid={!!errors.name}
            autoFocus
          />
        </Field>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field label="Tipe" required htmlFor="ev-type" error={errors.type}>
            <Select id="ev-type" value={type} onChange={(e) => setType(e.target.value as EventType)}>
              {TYPE_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </Select>
          </Field>

          <Field label="Status" htmlFor="ev-status">
            <Select id="ev-status" value={status} onChange={(e) => setStatus(e.target.value as EventStatus)}>
              {STATUS_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </Select>
          </Field>

          <Field label="Tanggal" htmlFor="ev-date" error={errors.date}>
            <DatePicker id="ev-date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>

          <Field label="Lokasi" htmlFor="ev-location">
            <Input
              id="ev-location"
              placeholder="JCC Jakarta"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
            />
          </Field>
        </div>

        <Field label="Penyelenggara" htmlFor="ev-organizer">
          <Input
            id="ev-organizer"
            placeholder="Nama penyelenggara"
            value={organizer}
            onChange={(e) => setOrganizer(e.target.value)}
          />
        </Field>

        <SectionLabel icon={Users}>Peserta yang Diundang</SectionLabel>
        <EmailChipsInput id="ev-emails" value={emails} onChange={setEmails} disabled={isPending} />

        <SectionLabel icon={Paperclip}>Lampiran</SectionLabel>
        <EventAttachmentsInput value={attachments} onChange={setAttachments} disabled={isPending} />

        <Field label="Catatan" htmlFor="ev-notes">
          <Textarea
            id="ev-notes"
            rows={3}
            placeholder="Catatan tambahan…"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </Field>
      </form>
    </Modal>
  )
}
