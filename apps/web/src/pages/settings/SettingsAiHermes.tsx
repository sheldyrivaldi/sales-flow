import { useEffect, useState } from 'react'
import { Activity, Sparkles, AlertTriangle, ShieldAlert, SlidersHorizontal, ExternalLink } from 'lucide-react'
import Card, { CardBody } from '../../components/ui/Card'
import Button from '../../components/ui/Button'
import Skeleton from '../../components/ui/Skeleton'
import ConfirmDialog from '../../components/ui/ConfirmDialog'
import { SectionHeader } from '../../components/ui/SectionHeader'
import { cn } from '../../lib/cn'
import { toast } from '../../lib/toast'
import { useAuthStore } from '../../store/auth'
import { can } from '../../lib/rbac'
import {
  useHermesStatus,
  useTestHermes,
  useResetHermesMemory,
  useIssueHermesTuiTicket,
  useEndHermesTuiSession,
} from '../../api/settings'

// ── Section 1: Status & Testing ─────────────────────────────────────────────

function StatusSection() {
  const role = useAuthStore((s) => s.user?.role)
  const isAdmin = can(role, 'ManageUsers')

  const { data: status, isLoading } = useHermesStatus()
  const testHermes = useTestHermes()
  const resetMemory = useResetHermesMemory()
  const [confirmReset, setConfirmReset] = useState(false)

  async function handleTest() {
    try {
      const res = await testHermes.mutateAsync()
      if (res.status === 'ok') {
        toast.success(`Koneksi berhasil${res.version ? ` • v${res.version}` : ''}.`)
      } else {
        toast.error('Koneksi gagal, periksa konfigurasi AI Agent.')
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Test koneksi gagal, coba lagi nanti.')
    }
  }

  async function handleResetMemory() {
    try {
      await resetMemory.mutateAsync()
      toast.success('Memory AI Agent berhasil direset.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Reset memory gagal, coba lagi nanti.')
    } finally {
      setConfirmReset(false)
    }
  }

  if (isLoading) {
    return (
      <Card className="max-w-xl">
        <CardBody className="flex flex-col gap-3">
          <Skeleton variant="text" className="h-5 w-1/2" />
          <Skeleton variant="text" className="h-4 w-3/4" />
        </CardBody>
      </Card>
    )
  }

  const connected = status?.status === 'connected'

  return (
    <Card className="max-w-xl">
      <CardBody className="flex flex-col gap-5">
        <div className="grid grid-cols-2 gap-4">
          {/* Connection */}
          <div className="flex flex-col gap-1.5">
            <p className="text-caption text-fg-muted">Status Koneksi</p>
            <span className="inline-flex items-center gap-2">
              <span
                className={cn(
                  'h-2.5 w-2.5 rounded-full',
                  connected ? 'bg-success shadow-[0_0_0_3px_rgba(5,150,105,0.15)]' : 'bg-danger shadow-[0_0_0_3px_rgba(225,29,72,0.15)]'
                )}
                aria-hidden="true"
              />
              <span className={cn('text-body font-semibold', connected ? 'text-success' : 'text-danger')}>
                {connected ? 'Terhubung' : 'Terputus'}
              </span>
              {connected && status?.version && (
                <span className="text-caption text-fg-subtle">v{status.version}</span>
              )}
            </span>
          </div>

          {/* Memory */}
          <div className="flex flex-col gap-1.5">
            <p className="text-caption text-fg-muted">Memory / Pembelajaran</p>
            <span className="inline-flex items-center gap-1.5 text-body text-fg">
              <Sparkles className="w-4 h-4 text-accent" aria-hidden="true" />
              {status?.memory_active ? 'Aktif' : 'Belum aktif'}
            </span>
          </div>
        </div>

        {!connected && (
          <p className="flex items-start gap-1.5 text-caption text-fg-muted rounded-btn bg-danger/5 border border-danger/20 px-3 py-2">
            <AlertTriangle className="w-3.5 h-3.5 text-danger mt-0.5 shrink-0" aria-hidden="true" />
            Asisten AI sedang tidak tersedia. Semua fitur data tetap berfungsi — hanya fitur AI yang terdampak.
          </p>
        )}

        <div className="flex gap-2">
          <Button variant="secondary" loading={testHermes.isPending} onClick={handleTest}>
            Test Koneksi
          </Button>
          {isAdmin && (
            <Button variant="danger" onClick={() => setConfirmReset(true)}>
              Reset Memory
            </Button>
          )}
        </div>
      </CardBody>

      <ConfirmDialog
        open={confirmReset}
        onCancel={() => setConfirmReset(false)}
        onConfirm={handleResetMemory}
        title="Reset memory AI Agent?"
        description="Seluruh riwayat pembelajaran AI (WON/LOST, alasan Tolak, konteks percakapan) akan dihapus dan tidak bisa dikembalikan."
        tone="danger"
        confirmLabel="Reset"
        loading={resetMemory.isPending}
      />
    </Card>
  )
}

// ── Section 2: AI Agent configuration console (admin only, auto-opens) ───────

type TuiStatus = 'connecting' | 'connected' | 'closed'

function AgentConsole() {
  const issueTicket = useIssueHermesTuiTicket()
  const endSession = useEndHermesTuiSession()

  const [status, setStatus] = useState<TuiStatus>('connecting')
  const [tuiUrl, setTuiUrl] = useState<string | null>(null)

  async function openSession() {
    setStatus('connecting')
    try {
      const { tui_url } = await issueTicket.mutateAsync()
      setTuiUrl(tui_url)
      setStatus('connected')
    } catch (err) {
      setStatus('closed')
      toast.error(err instanceof Error ? err.message : 'Gagal membuka sesi AI Agent.')
    }
  }

  // Auto-open on mount — visiting this page immediately opens the console
  // rather than gating it behind an extra click. Still audit-logged
  // server-side (open time, duration, IP) exactly like the manual flow.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- one-shot session bootstrap on mount, not a per-render reset.
    openSession()
    return () => {
      endSession.mutate()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- run once on mount; end on unmount only.
  }, [])

  async function handleOpenNewTab() {
    // A fresh ticket is required — tickets are single-use, and the one already
    // consumed by the iframe below cannot be reused for a second navigation.
    try {
      const { tui_url } = await issueTicket.mutateAsync()
      window.open(tui_url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal membuka tab baru.')
    }
  }

  async function handleEndSession() {
    try {
      await endSession.mutateAsync()
      toast.success('Sesi AI Agent ditutup.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal menutup sesi, coba lagi.')
    } finally {
      setTuiUrl(null)
      setStatus('closed')
    }
  }

  return (
    <Card className="w-full overflow-hidden">
      <CardBody className="flex flex-col gap-4">
        <div className="flex items-start gap-2 rounded-btn border border-warning/30 bg-warning/10 p-3">
          <ShieldAlert className="w-4 h-4 text-warning mt-0.5 shrink-0" aria-hidden="true" />
          <p className="text-caption text-fg-muted">
            Konsol ini memberi akses teknis langsung ke AI Agent (login provider, konfigurasi model,
            skills, memori). Aktivitas dicatat sebagai log audit (waktu buka, durasi, alamat IP),{' '}
            <strong>bukan</strong> transkrip percakapan atau isi layar.
          </p>
        </div>

        {status === 'closed' ? (
          <div>
            <Button
              variant="primary"
              leftIcon={<SlidersHorizontal className="w-4 h-4" aria-hidden="true" />}
              onClick={openSession}
            >
              Buka Konfigurasi AI Agent
            </Button>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between gap-2">
              <span className="inline-flex items-center gap-2 text-caption text-fg-muted">
                <span
                  className={cn(
                    'h-2 w-2 rounded-full',
                    status === 'connected' ? 'bg-success' : 'bg-warning animate-pulse'
                  )}
                  aria-hidden="true"
                />
                {status === 'connecting' ? 'Membuka konfigurasi…' : 'Konfigurasi AI Agent aktif'}
              </span>
              {status === 'connected' && (
                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    leftIcon={<ExternalLink className="w-4 h-4" aria-hidden="true" />}
                    onClick={handleOpenNewTab}
                  >
                    Buka di tab baru
                  </Button>
                  <Button variant="danger" size="sm" loading={endSession.isPending} onClick={handleEndSession}>
                    Tutup Sesi
                  </Button>
                </div>
              )}
            </div>
            {status === 'connecting' && <Skeleton className="h-[72vh] w-full rounded-btn" />}
            {tuiUrl && status === 'connected' && (
              <iframe
                src={tuiUrl}
                title="Konfigurasi AI Agent"
                className="h-[72vh] w-full rounded-btn border border-line bg-surface"
              />
            )}
          </>
        )}
      </CardBody>
    </Card>
  )
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function SettingsAiHermes() {
  const role = useAuthStore((s) => s.user?.role)
  const isAdmin = can(role, 'ManageUsers')

  return (
    <div className="flex flex-col gap-8 pb-24 max-w-6xl">
      <div>
        <h1 className="text-h2 font-semibold text-fg">AI Agent</h1>
        <p className="text-body text-fg-muted mt-1">
          Otak AI yang menjalankan pencarian tender, scoring, dan chat. Status koneksi dan konfigurasi teknis ada di sini.
        </p>
      </div>

      <section className="flex flex-col gap-4">
        <SectionHeader
          icon={Activity}
          tone="emerald"
          title="Status & Koneksi"
          description="Kondisi koneksi AI Agent dan status pembelajaran memori."
        />
        <StatusSection />
      </section>

      {isAdmin && (
        <>
          <div className="border-t border-line" role="separator" />
          <section className="flex flex-col gap-4">
            <SectionHeader
              icon={SlidersHorizontal}
              tone="ai"
              title="Konfigurasi AI Agent"
              description="Login provider, pilih model, kelola skills & memori — langsung di konsol AI Agent."
            />
            <AgentConsole />
          </section>
        </>
      )}
    </div>
  )
}
