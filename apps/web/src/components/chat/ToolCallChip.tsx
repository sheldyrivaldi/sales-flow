import { useState } from 'react'
import { Wrench, Check, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '../../lib/cn'

export const TOOL_LABELS: Record<string, string> = {
  list_tenders: 'Membaca data tender…',
  search_tenders: 'Mencari tender…',
  get_tender: 'Membuka detail tender…',
  list_prospects: 'Membaca data prospek…',
  get_prospect: 'Membuka detail prospek…',
  get_pipeline_summary: 'Merangkum pipeline…',
  get_revenue_summary: 'Membaca data pendapatan…',
  get_company_profile: 'Membaca profil perusahaan…',
  list_events: 'Membaca data event…',
  get_event: 'Membuka detail event…',
  update_prospect_stage: 'Memperbarui stage prospek…',
  save_playbook_draft: 'Menyimpan draft playbook…',
}

function toolLabel(name: string): string {
  return TOOL_LABELS[name] ?? `Menjalankan ${name}…`
}

export interface ToolCallChipProps {
  name: string
  arguments?: unknown
  status?: 'running' | 'done'
  resultCount?: number
}

export default function ToolCallChip({
  name,
  arguments: args,
  status = 'running',
  resultCount,
}: ToolCallChipProps) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="inline-flex flex-col">
      <button
        type="button"
        onClick={() => setExpanded((o) => !o)}
        className={cn(
          'inline-flex items-center gap-1.5 text-caption rounded-pill px-3 py-1 border transition-colors',
          status === 'running'
            ? 'border-line bg-surface text-fg-muted'
            : 'border-success/30 bg-success/5 text-success',
        )}
        aria-expanded={expanded}
      >
        {status === 'running' ? (
          <Wrench className="w-3.5 h-3.5 animate-pulse shrink-0" aria-hidden="true" />
        ) : (
          <Check className="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
        )}
        <span>
          {status === 'running'
            ? toolLabel(name)
            : resultCount !== undefined
              ? `✓ ${resultCount} data`
              : '✓ Selesai'}
        </span>
        {expanded ? (
          <ChevronUp className="w-3 h-3 ml-0.5 shrink-0" aria-hidden="true" />
        ) : (
          <ChevronDown className="w-3 h-3 ml-0.5 shrink-0" aria-hidden="true" />
        )}
      </button>

      {expanded && (
        <div className="mt-1 ml-2 text-caption text-fg-muted bg-surface-subtle border border-line rounded-card px-3 py-2">
          <p className="font-medium text-fg">{name}</p>
          {args !== undefined && (
            <pre className="mt-1 text-xs overflow-x-auto whitespace-pre-wrap break-all">
              {JSON.stringify(args, null, 2)}
            </pre>
          )}
        </div>
      )}
    </div>
  )
}
