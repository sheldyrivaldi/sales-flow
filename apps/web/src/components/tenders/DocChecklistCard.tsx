import { CheckCircle2, CircleAlert, CircleX, FileCheck2 } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Button from '../ui/Button'
import Badge from '../ui/Badge'
import { SkeletonText } from '../ui/Skeleton'
import { cn } from '../../lib/cn'
import { useAIBusy, AI_MUTATION_KEYS } from '../../lib/aiMutation'
import { useDocChecklist } from '../../api/tenders'
import type { DocChecklistStatus } from '../../api/tenders'

const STATUS_CONFIG: Record<
  DocChecklistStatus,
  { icon: typeof CheckCircle2; className: string; label: string }
> = {
  tersedia:          { icon: CheckCircle2, className: 'text-success', label: 'Tersedia' },
  perlu_verifikasi:  { icon: CircleAlert,  className: 'text-warning', label: 'Perlu verifikasi' },
  belum_ada:         { icon: CircleX,      className: 'text-danger',  label: 'Belum ada' },
}

function readinessTone(score: number): 'success' | 'warning' | 'danger' {
  if (score >= 75) return 'success'
  if (score >= 50) return 'warning'
  return 'danger'
}

/** Ceklis kelengkapan dokumen administrasi terhadap syarat tender — AI
 * membandingkan yang diminta tender dengan profil perusahaan dan memberi
 * saran konkret untuk dokumen yang kurang. */
export default function DocChecklistCard({ tenderId }: { tenderId: string }) {
  const checklist = useDocChecklist()
  const aiBusy = useAIBusy(AI_MUTATION_KEYS.docChecklist, tenderId)

  // Toast ditangani MutationCache global — tetap muncul walau pindah halaman.
  function handleCheck() {
    checklist.mutate(tenderId)
  }

  const result = checklist.data

  return (
    <Card>
      <CardHeader className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <FileCheck2 className="w-4 h-4 text-primary" aria-hidden="true" />
          <h3 className="text-body font-semibold text-fg">Kelengkapan Dokumen</h3>
        </div>
        {result && (
          <Badge tone={readinessTone(result.readiness_score)}>
            Kesiapan {result.readiness_score}%
          </Badge>
        )}
      </CardHeader>
      <CardBody className="flex flex-col gap-3">
        {!result && !checklist.isPending && !aiBusy && (
          <>
            <p className="text-caption text-fg-muted">
              AI memeriksa dokumen apa saja yang diminta tender ini, membandingkannya dengan profil
              perusahaan, dan memberi saran untuk yang belum ada.
            </p>
            <Button size="sm" className="self-start" disabled={aiBusy} onClick={handleCheck}>
              Periksa Kelengkapan Dokumen
            </Button>
          </>
        )}

        {(checklist.isPending || (aiBusy && !result)) && <SkeletonText lines={5} />}

        {result && (
          <>
            <p className="text-body text-fg">{result.summary}</p>

            <ul className="flex flex-col divide-y divide-line">
              {result.items.map((item, i) => {
                const cfg = STATUS_CONFIG[item.status] ?? STATUS_CONFIG.perlu_verifikasi
                const Icon = cfg.icon
                return (
                  <li key={i} className="py-2.5 flex items-start gap-2.5">
                    <Icon className={cn('w-4 h-4 mt-0.5 shrink-0', cfg.className)} aria-hidden="true" />
                    <div className="min-w-0 flex-1">
                      <p className="text-body font-medium text-fg">
                        {item.document}
                        {item.required && (
                          <span className="ml-1.5 text-caption font-semibold text-danger">wajib</span>
                        )}
                      </p>
                      <p className={cn('text-caption', cfg.className)}>{cfg.label}</p>
                      {item.suggestion && item.status !== 'tersedia' && (
                        <p className="text-caption text-fg-muted mt-0.5">{item.suggestion}</p>
                      )}
                    </div>
                  </li>
                )
              })}
            </ul>

            <Button
              variant="secondary"
              size="sm"
              className="self-start"
              loading={checklist.isPending}
              disabled={aiBusy}
              onClick={handleCheck}
            >
              Periksa Ulang
            </Button>
          </>
        )}
      </CardBody>
    </Card>
  )
}
