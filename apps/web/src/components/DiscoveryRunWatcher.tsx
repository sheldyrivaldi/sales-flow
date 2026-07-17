import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useDiscoveryRuns } from '../api/discovery'
import { toast } from '../lib/toast'

/** Pengawas global run Radar Tender — dipasang di AppShell sehingga:
 *  - saat ada run pending/running, status dipoll tiap 5 detik di halaman
 *    MANA PUN user berada (crawling jalan terus di backend);
 *  - begitu run selesai, user dapat toast + data inbox/tender di-refresh,
 *    tanpa harus berdiri di halaman Radar Tender.
 *  Saat tidak ada run aktif, polling mati total (refetchInterval false). */
export default function DiscoveryRunWatcher() {
  const queryClient = useQueryClient()
  const { data } = useDiscoveryRuns({ refetchInterval: undefined })
  const latest = data?.items[0]
  const isActive = latest?.status === 'pending' || latest?.status === 'running'

  // Poll hanya saat ada run aktif.
  useDiscoveryRuns({ refetchInterval: isActive ? 5000 : false })

  // Deteksi transisi running → selesai. Ref (bukan state) supaya efek tidak
  // memicu render tambahan.
  const prevRef = useRef<{ id: string; active: boolean } | null>(null)
  useEffect(() => {
    if (!latest) return
    const prev = prevRef.current
    if (prev && prev.id === latest.id && prev.active && !isActive) {
      if (latest.status === 'success') {
        toast.success(`Crawling tender selesai, ${latest.found_count} tender baru masuk Radar Tender.`)
      } else if (latest.status === 'failed') {
        toast.error('Crawling tender gagal, coba jalankan ulang dari Radar Tender.')
      }
      void queryClient.invalidateQueries({ queryKey: ['discovery-inbox'] })
      void queryClient.invalidateQueries({ queryKey: ['tenders'] })
    }
    prevRef.current = { id: latest.id, active: isActive }
  }, [latest, isActive, queryClient])

  return null
}
