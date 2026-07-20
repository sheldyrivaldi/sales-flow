import { Link } from 'react-router'
import { BookOpen, Loader2, Presentation, Sparkles, ArrowRight, AlertCircle } from 'lucide-react'

import Card, { CardHeader, CardBody } from '../ui/Card'
import Button from '../ui/Button'
import Badge from '../ui/Badge'
import { toast } from '../../lib/toast'
import { formatRelative } from '../../lib/format'
import { exportPlaybookPpt } from '../../lib/exportPlaybookPpt'
import {
  usePlaybookJobs,
  useCreateEventPlaybookJob,
  isJobActive,
  PLAYBOOK_STATUS_LABEL,
} from '../../api/playbookJobs'

export interface EventPlaybookCardProps {
  eventId: string
  eventName: string
}

/** Kartu generate playbook di halaman detail Event. Tombol memicu generate
 * TERSTANDARISASI (riset internet + seluruh konteks event) secara async;
 * hasilnya masuk riwayat di menu Playbooks. Status job untuk event ini
 * ditampilkan inline bila ada. */
export default function EventPlaybookCard({ eventId, eventName }: EventPlaybookCardProps) {
  const { data: jobs } = usePlaybookJobs()
  const createMutation = useCreateEventPlaybookJob()

  // Job untuk event ini dikenali dari judul baku "Playbook Event: {nama}".
  const title = `Playbook Event: ${eventName}`
  const job = jobs?.find((j) => j.source === 'event' && j.title === title)

  async function handleGenerate() {
    try {
      await createMutation.mutateAsync(eventId)
      toast.success('Playbook event sedang diproses, hasilnya muncul di menu Playbooks.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal memulai generate playbook.')
    }
  }

  const busy = createMutation.isPending || (job ? isJobActive(job.status) : false)

  return (
    <Card>
      <CardHeader className="flex items-center gap-2">
        <BookOpen className="w-4 h-4 text-primary" aria-hidden="true" />
        <h3 className="text-body font-semibold text-fg">Playbook Event</h3>
      </CardHeader>
      <CardBody className="flex flex-col gap-3">
        <p className="text-caption text-fg-muted">
          AI menyusun playbook terstandarisasi untuk memaksimalkan peluang dari event ini — memakai
          seluruh informasi event, riset internet terbaru tentang penyelenggara, dan profil perusahaan.
        </p>

        {job && (
          <div className="flex items-center gap-2 flex-wrap rounded-card border border-line bg-surface-subtle px-3 py-2">
            <Badge
              tone={
                job.status === 'success' ? 'success' : job.status === 'failed' ? 'danger' : job.status === 'updating' ? 'warning' : 'info'
              }
            >
              <span className="inline-flex items-center gap-1">
                {isJobActive(job.status) && <Loader2 className="w-3 h-3 animate-spin" aria-hidden="true" />}
                {PLAYBOOK_STATUS_LABEL[job.status]}
              </span>
            </Badge>
            <span className="text-caption text-fg-subtle">{formatRelative(job.updated_at)}</span>
            {job.status === 'failed' && job.error_message && (
              <span className="inline-flex items-center gap-1 text-caption text-danger">
                <AlertCircle className="w-3.5 h-3.5" aria-hidden="true" />
                {job.error_message}
              </span>
            )}
            {job.status === 'success' && job.content && (
              <Button
                size="sm"
                variant="ghost"
                leftIcon={<Presentation className="w-3.5 h-3.5" />}
                onClick={() => void exportPlaybookPpt(job.content!, job.title).catch(() => toast.error('Ekspor PPT gagal.'))}
              >
                Buka PPT
              </Button>
            )}
            <Link to="/playbooks" className="ml-auto text-caption text-primary inline-flex items-center gap-1 hover:underline">
              Lihat di Playbooks <ArrowRight className="w-3 h-3" aria-hidden="true" />
            </Link>
          </div>
        )}

        <Button
          size="sm"
          className="self-start"
          leftIcon={<Sparkles className="w-4 h-4" />}
          loading={busy}
          disabled={busy}
          onClick={() => void handleGenerate()}
        >
          {job ? 'Generate Ulang Playbook' : 'Generate Playbook'}
        </Button>
      </CardBody>
    </Card>
  )
}
