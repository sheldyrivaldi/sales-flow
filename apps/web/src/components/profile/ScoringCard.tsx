import { ChevronDown, SlidersHorizontal } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'

// Placeholder for the scoring weights/threshold editor — backend fields for
// this (EP-10) don't exist yet, so this card renders collapsed and inert.
// Replace the body once EP-10 ships prospect_score weight configuration.
export default function ScoringCard() {
  return (
    <Card>
      <details>
        <summary className="list-none cursor-pointer">
          <CardHeader className="flex items-center justify-between border-b-0">
            <div className="flex items-center gap-1.5">
              <SlidersHorizontal className="w-4 h-4 text-fg-muted" aria-hidden="true" />
              <h2 className="text-body font-semibold text-fg">Scoring (Advanced)</h2>
            </div>
            <ChevronDown className="w-4 h-4 text-fg-muted" aria-hidden="true" />
          </CardHeader>
        </summary>
        <CardBody className="border-t border-line">
          <p className="text-caption text-fg-muted">
            Bobot scoring &amp; threshold rekomendasi — segera hadir (EP-10). Saat ini agent memakai
            rubrik default.
          </p>
        </CardBody>
      </details>
    </Card>
  )
}
