import { useEffect, useState } from 'react'
import Drawer from '../../components/ui/Drawer'
import Button from '../../components/ui/Button'
import Input from '../../components/ui/Input'
import Select from '../../components/ui/Select'
import Textarea from '../../components/ui/Textarea'
import Field from '../../components/ui/Field'
import DatePicker from '../../components/ui/DatePicker'
import { toast } from '../../lib/toast'
import { useCreateEvent, useUpdateEvent } from '../../api/events'
import type { Event, EventType, EventStatus } from '../../api/events'

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

export default function EventFormDrawer({ open, onClose, event, onSaved }: Props) {
  const isEdit = !!event

  const [name, setName] = useState('')
  const [type, setType] = useState<EventType>('EXPO')
  const [date, setDate] = useState('')
  const [location, setLocation] = useState('')
  const [organizer, setOrganizer] = useState('')
  const [notes, setNotes] = useState('')
  const [status, setStatus] = useState<EventStatus>('PLANNED')
  const [errors, setErrors] = useState<Record<string, string>>({})

  const createMutation = useCreateEvent()
  const updateMutation = useUpdateEvent()

  useEffect(() => {
    if (open) {
      setName(event?.name ?? '')
      setType(event?.type ?? 'EXPO')
      setDate(toDateInputValue(event?.date))
      setLocation(event?.location ?? '')
      setOrganizer(event?.organizer ?? '')
      setNotes(event?.notes ?? '')
      setStatus(event?.status ?? 'PLANNED')
      setErrors({})
    }
  }, [open, event])

  function validate(): boolean {
    const e: Record<string, string> = {}
    if (!name.trim()) e.name = 'Nama event wajib diisi'
    if (!type) e.type = 'Tipe event wajib dipilih'
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
    } catch {
      toast.error(isEdit ? 'Gagal memperbarui event.' : 'Gagal menambahkan event.')
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit Event' : 'Tambah Event Baru'}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isPending}>
            Batal
          </Button>
          <Button type="submit" form="event-form" loading={isPending}>
            {isEdit ? 'Simpan Perubahan' : 'Tambah Event'}
          </Button>
        </>
      }
    >
      <form id="event-form" onSubmit={handleSubmit} className="flex flex-col gap-4">
        <Field label="Nama Event" required htmlFor="ev-name" error={errors.name}>
          <Input
            id="ev-name"
            placeholder="Indo Security Expo 2026"
            value={name}
            onChange={(e) => setName(e.target.value)}
            invalid={!!errors.name}
          />
        </Field>

        <Field label="Tipe" required htmlFor="ev-type" error={errors.type}>
          <Select
            id="ev-type"
            value={type}
            onChange={(e) => setType(e.target.value as EventType)}
          >
            {TYPE_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </Select>
        </Field>

        <Field label="Tanggal" htmlFor="ev-date">
          <DatePicker
            id="ev-date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
          />
        </Field>

        <Field label="Lokasi" htmlFor="ev-location">
          <Input
            id="ev-location"
            placeholder="JCC Jakarta"
            value={location}
            onChange={(e) => setLocation(e.target.value)}
          />
        </Field>

        <Field label="Organizer" htmlFor="ev-organizer">
          <Input
            id="ev-organizer"
            placeholder="Nama penyelenggara"
            value={organizer}
            onChange={(e) => setOrganizer(e.target.value)}
          />
        </Field>

        <Field label="Status" htmlFor="ev-status">
          <Select
            id="ev-status"
            value={status}
            onChange={(e) => setStatus(e.target.value as EventStatus)}
          >
            {STATUS_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </Select>
        </Field>

        <Field label="Catatan" htmlFor="ev-notes">
          <Textarea
            id="ev-notes"
            placeholder="Catatan tambahan…"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </Field>
      </form>
    </Drawer>
  )
}
