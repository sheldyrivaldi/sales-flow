import { useState } from 'react'
import { Sparkles } from 'lucide-react'

import Button from '../../components/ui/Button'
import Tabs, { TabPanel } from '../../components/ui/Tabs'
import { useCan } from '../../lib/useCan'
import { formatRelative } from '../../lib/format'
import { toast } from '../../lib/toast'
import { useDiscoveryRuns, useRunDiscovery, useDiscoveryInbox } from '../../api/discovery'
import { useProfile, isProfileConfigured } from '../../api/profile'

import DiscoveryInboxPanel from './panels/DiscoveryInboxPanel'
import TenderTablePanel from './panels/TenderTablePanel'

type TabId = 'discovery' | 'tenders'

/** TendersPage — menu tunggal "Radar Tender" yang menyatukan penemuan tender
 * oleh AI (crawling via Hermes) dan pengelolaan tender lintas lifecycle
 * pra-deal. Dulu dua menu terpisah (/discovery + /tenders); kini satu halaman
 * dua tab. Fitur AI (tombol "Cari Tender dengan AI" + tab "Penemuan AI") hanya
 * untuk role dengan izin RunDiscovery; role lain tetap melihat tabel tender. */
export default function TendersPage() {
  const canDiscovery = useCan('RunDiscovery')
  const [tab, setTab] = useState<TabId>('discovery')

  const { data: profile } = useProfile()
  const profileConfigured = isProfileConfigured(profile)

  const { data: runsData } = useDiscoveryRuns({ refetchInterval: canDiscovery ? 3000 : false })
  const latestRun = runsData?.items[0]
  const isRunning = latestRun?.status === 'pending' || latestRun?.status === 'running'

  // Jumlah inbox (tanpa filter) untuk badge tab — di-dedupe react-query dengan
  // query berfilter di dalam DiscoveryInboxPanel bila keduanya kebetulan sama.
  const { data: inboxBadge } = useDiscoveryInbox()
  const inboxCount = inboxBadge?.total ?? 0

  const runMutation = useRunDiscovery()

  async function handleRun() {
    try {
      const run = await runMutation.mutateAsync()
      if (run.status === 'pending' || run.status === 'running') {
        toast.success('Crawling tender dimulai, hasil masuk otomatis saat selesai.')
      }
    } catch (err) {
      toast.error(err instanceof Error && err.message ? err.message : 'Gagal memulai crawling tender.')
    }
  }

  const statusText = !canDiscovery
    ? 'Kelola tender lintas tahap, dari identifikasi sampai submit.'
    : latestRun
      ? isRunning
        ? 'Crawling sedang berjalan — bisa memakan beberapa menit, kamu boleh tinggal ke halaman lain.'
        : `Terakhir dicari: ${formatRelative(latestRun.finished_at ?? latestRun.started_at)} • ${latestRun.found_count} baru`
      : 'Cari tender di internet yang cocok dengan profil perusahaan, atau kelola tender manual.'

  return (
    <div className="flex flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-h2 font-semibold text-fg flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-accent" /> Radar Tender
          </h1>
          <p className="text-caption text-fg-muted mt-0.5">{statusText}</p>
        </div>
        {canDiscovery && (
          <Button
            leftIcon={<Sparkles className="w-4 h-4" />}
            loading={runMutation.isPending || isRunning}
            disabled={!profileConfigured || isRunning}
            onClick={handleRun}
          >
            {isRunning ? 'Crawling berjalan…' : 'Cari Tender dengan AI'}
          </Button>
        )}
      </div>

      {/* Role tanpa izin discovery: langsung tabel tender tanpa tab. */}
      {!canDiscovery ? (
        <TenderTablePanel />
      ) : (
        <>
          <Tabs
            tabs={[
              { id: 'discovery', label: inboxCount > 0 ? `Penemuan AI (${inboxCount})` : 'Penemuan AI' },
              { id: 'tenders', label: 'Tender Aktif' },
            ]}
            value={tab}
            onChange={(id) => setTab(id as TabId)}
          />
          {tab === 'discovery' ? (
            <TabPanel id="discovery">
              <DiscoveryInboxPanel />
            </TabPanel>
          ) : (
            <TabPanel id="tenders">
              <TenderTablePanel />
            </TabPanel>
          )}
        </>
      )}
    </div>
  )
}
