import { Construction } from 'lucide-react'
import EmptyState from './ui/EmptyState'

export interface PagePlaceholderProps {
  title: string
}

export default function PagePlaceholder({ title }: PagePlaceholderProps) {
  return (
    <EmptyState
      icon={<Construction className="w-7 h-7" aria-hidden="true" />}
      title={title}
      description="Halaman ini akan dibangun pada epik berikutnya."
    />
  )
}
