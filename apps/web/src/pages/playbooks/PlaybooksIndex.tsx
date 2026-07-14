import { Sparkles } from 'lucide-react'
import { Link } from 'react-router'

import EmptyState from '../../components/ui/EmptyState'
import Button from '../../components/ui/Button'

/** Playbook selalu digenerate dari konteks satu peluang (tab Playbook di
 * Tender Detail / drawer Prospect) — halaman ini bukan generator berdiri
 * sendiri, hanya mengarahkan pengguna ke tempat yang benar. */
export default function PlaybooksIndex() {
  return (
    <div className="p-6">
      <EmptyState
        icon={<Sparkles className="w-6 h-6" />}
        title="Playbook dibuat per peluang"
        description="Buka detail tender atau prospek, lalu buka tab/bagian Playbook untuk generate strategi terstruktur."
        action={
          <div className="flex items-center gap-2">
            <Link to="/tenders">
              <Button variant="secondary" size="sm">Buka Tenders</Button>
            </Link>
            <Link to="/prospects">
              <Button variant="secondary" size="sm">Buka Prospects</Button>
            </Link>
          </div>
        }
      />
    </div>
  )
}
