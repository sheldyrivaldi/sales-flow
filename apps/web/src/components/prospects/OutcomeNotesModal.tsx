import { Sparkles } from 'lucide-react'
import Modal from '../ui/Modal'
import Field from '../ui/Field'
import Textarea from '../ui/Textarea'
import Button from '../ui/Button'
import type { ProspectStage } from '../../api/prospects'

export interface OutcomeNotesModalProps {
  open: boolean
  stage: ProspectStage | null
  notes: string
  onNotesChange: (notes: string) => void
  loading?: boolean
  onConfirm: () => void
  onCancel: () => void
}

/** Modal catatan opsional sebelum mencatat outcome WON/LOST (Design §4.8/4.9).
 * Dipakai bersama oleh ProspectBoard (drag/kebab) dan ProspectDrawer agar
 * gating & UX WON/LOST tidak bisa drift antar-file. */
export default function OutcomeNotesModal({
  open,
  stage,
  notes,
  onNotesChange,
  loading,
  onConfirm,
  onCancel,
}: OutcomeNotesModalProps) {
  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={`Tandai ${stage ?? ''}`}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onCancel}>
            Batal
          </Button>
          <Button loading={loading} onClick={onConfirm}>
            Simpan
          </Button>
        </>
      }
    >
      <Field label="Catatan (opsional)" htmlFor="outcome-notes-textarea">
        <Textarea
          id="outcome-notes-textarea"
          value={notes}
          onChange={(e) => onNotesChange(e.target.value)}
          rows={3}
          placeholder="Alasan, pembelajaran, atau catatan tambahan…"
        />
      </Field>
      <p className="mt-2 text-caption text-fg-muted flex items-center gap-1">
        <Sparkles className="w-3.5 h-3.5 text-accent" />
        Asisten belajar dari aktivitas & hasil kamu.
      </p>
    </Modal>
  )
}
