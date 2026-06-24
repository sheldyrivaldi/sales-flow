import { useState } from 'react'
import type { ReactNode } from 'react'
import { Plus, ArrowRight, LayoutGrid, FileText } from 'lucide-react'
import Button from '../components/ui/Button'
import Field from '../components/ui/Field'
import Input from '../components/ui/Input'
import Textarea from '../components/ui/Textarea'
import Select from '../components/ui/Select'
import DatePicker from '../components/ui/DatePicker'
import Combobox from '../components/ui/Combobox'
import ChipInput from '../components/ui/ChipInput'
import Toggle from '../components/ui/Toggle'
import Badge, { AiBadge, ActionBadge, ScoreBadge, StagePill } from '../components/ui/Badge'
import Card, { CardHeader, CardBody, CardFooter } from '../components/ui/Card'
import Tabs, { TabPanel } from '../components/ui/Tabs'
import Breadcrumb from '../components/ui/Breadcrumb'
import Avatar from '../components/ui/Avatar'
import Tooltip from '../components/ui/Tooltip'
import Table from '../components/ui/Table'
import Modal from '../components/ui/Modal'
import Drawer from '../components/ui/Drawer'
import ConfirmDialog from '../components/ui/ConfirmDialog'
import { toast } from '../lib/toast'
import Skeleton, { SkeletonText } from '../components/ui/Skeleton'
import EmptyState from '../components/ui/EmptyState'
import { Search } from 'lucide-react'
import ScoreRing from '../components/ui/ScoreRing'
import StatCard from '../components/ui/StatCard'
import AiCallout from '../components/ui/AiCallout'
import { BarChart2, Target } from 'lucide-react'
import StreamingText from '../components/ui/StreamingText'
import RiskFlag, { RiskFlagList } from '../components/ui/RiskFlag'
import FileDropzone from '../components/ui/FileDropzone'
import Stepper from '../components/ui/Stepper'
import Popover from '../components/ui/Popover'
import Menu from '../components/ui/Menu'

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-4">
      <h2 className="text-h2 font-semibold text-fg border-b border-line pb-2">{title}</h2>
      {children}
    </section>
  )
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1">
      <p className="text-caption text-fg-muted font-medium uppercase tracking-wide">{label}</p>
      <div className="flex flex-wrap items-center gap-3">{children}</div>
    </div>
  )
}

const cityOptions = [
  { label: 'Jakarta', value: 'jkt' },
  { label: 'Bandung', value: 'bdg' },
  { label: 'Surabaya', value: 'sby' },
  { label: 'Medan', value: 'mdn' },
  { label: 'Makassar', value: 'mksr' },
]

function FormFieldsSection() {
  const [comboVal, setComboVal] = useState('')
  return (
    <Section title="Form Fields">
      <Row label="Input — normal / helper / error / disabled">
        <div className="w-64">
          <Field label="Nama perusahaan" htmlFor="input-normal">
            <Input id="input-normal" placeholder="PT Maju Jaya" />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Nama perusahaan" helper="Gunakan nama resmi" htmlFor="input-helper">
            <Input id="input-helper" placeholder="PT Maju Jaya" />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Nama perusahaan" error="Nama wajib diisi" htmlFor="input-error">
            <Input id="input-error" placeholder="PT Maju Jaya" invalid />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Nama perusahaan" htmlFor="input-disabled">
            <Input id="input-disabled" placeholder="PT Maju Jaya" disabled />
          </Field>
        </div>
      </Row>

      <Row label="Textarea — normal / error / disabled">
        <div className="w-64">
          <Field label="Deskripsi" htmlFor="textarea-normal">
            <Textarea id="textarea-normal" placeholder="Tuliskan deskripsi…" />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Deskripsi" error="Deskripsi terlalu pendek" htmlFor="textarea-error">
            <Textarea id="textarea-error" placeholder="Tuliskan deskripsi…" invalid />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Deskripsi" htmlFor="textarea-disabled">
            <Textarea id="textarea-disabled" placeholder="Tuliskan deskripsi…" disabled />
          </Field>
        </div>
      </Row>

      <Row label="Select — normal / error / disabled">
        <div className="w-64">
          <Field label="Kategori" htmlFor="select-normal">
            <Select id="select-normal">
              <option value="">Pilih kategori</option>
              <option value="it">IT</option>
              <option value="konstruksi">Konstruksi</option>
            </Select>
          </Field>
        </div>
        <div className="w-64">
          <Field label="Kategori" error="Pilih kategori" htmlFor="select-error">
            <Select id="select-error" invalid>
              <option value="">Pilih kategori</option>
            </Select>
          </Field>
        </div>
        <div className="w-64">
          <Field label="Kategori" htmlFor="select-disabled">
            <Select id="select-disabled" disabled>
              <option>Tidak aktif</option>
            </Select>
          </Field>
        </div>
      </Row>

      <Row label="DatePicker — normal / error / disabled">
        <div className="w-64">
          <Field label="Deadline" htmlFor="date-normal">
            <DatePicker id="date-normal" />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Deadline" error="Tanggal wajib diisi" htmlFor="date-error">
            <DatePicker id="date-error" invalid />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Deadline" htmlFor="date-disabled">
            <DatePicker id="date-disabled" disabled />
          </Field>
        </div>
      </Row>

      <Row label="Combobox — normal / error / disabled">
        <div className="w-64">
          <Field label="Kota" htmlFor="combo-normal">
            <Combobox
              options={cityOptions}
              value={comboVal}
              onChange={setComboVal}
              placeholder="Cari kota…"
            />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Kota" error="Pilih kota" htmlFor="combo-error">
            <Combobox
              options={cityOptions}
              value=""
              onChange={() => {}}
              placeholder="Cari kota…"
              invalid
            />
          </Field>
        </div>
        <div className="w-64">
          <Field label="Kota" htmlFor="combo-disabled">
            <Combobox
              options={cityOptions}
              value=""
              onChange={() => {}}
              placeholder="Cari kota…"
              disabled
            />
          </Field>
        </div>
      </Row>
    </Section>
  )
}

function ToggleSection() {
  const [a, setA] = useState(false)
  const [b, setB] = useState(true)
  return (
    <Section title="Toggle / Switch">
      <Row label="md — off / on">
        <Toggle checked={a} onChange={setA} label="Notifikasi email" />
        <Toggle checked={b} onChange={setB} label="Aktifkan discovery" />
      </Row>
      <Row label="sm">
        <Toggle checked={a} onChange={setA} size="sm" label="Kecil" />
        <Toggle checked={b} onChange={setB} size="sm" label="Kecil aktif" />
      </Row>
      <Row label="disabled">
        <Toggle checked={false} onChange={() => {}} disabled label="Nonaktif" />
        <Toggle checked={true} onChange={() => {}} disabled label="Nonaktif aktif" />
      </Row>
    </Section>
  )
}

function BadgeSection() {
  return (
    <Section title="Badge / Pill">
      <Row label="tone — soft">
        <Badge tone="success">success</Badge>
        <Badge tone="warning">warning</Badge>
        <Badge tone="danger">danger</Badge>
        <Badge tone="info">info</Badge>
        <Badge tone="accent">accent</Badge>
      </Row>
      <Row label="tone — solid">
        <Badge tone="success" appearance="solid">success</Badge>
        <Badge tone="warning" appearance="solid">warning</Badge>
        <Badge tone="danger" appearance="solid">danger</Badge>
        <Badge tone="info" appearance="solid">info</Badge>
        <Badge tone="accent" appearance="solid">accent</Badge>
      </Row>
      <Row label="AI badge">
        <AiBadge />
      </Row>
      <Row label="ActionBadge — semua 5 action">
        <ActionBadge action="Pursue" />
        <ActionBadge action="Review" />
        <ActionBadge action="Watchlist" />
        <ActionBadge action="Reject" />
        <ActionBadge action="Need Partner" />
      </Row>
      <Row label="ScoreBadge — contoh skor">
        <ScoreBadge score={48} />
        <ScoreBadge score={56} />
        <ScoreBadge score={72} />
        <ScoreBadge score={86} />
      </Row>
      <Row label="StagePill — tender">
        <StagePill stage="IDENTIFIED" />
        <StagePill stage="QUALIFYING" />
        <StagePill stage="BIDDING" />
        <StagePill stage="SUBMITTED" />
        <StagePill stage="WON" />
        <StagePill stage="LOST" />
      </Row>
      <Row label="StagePill — prospect">
        <StagePill stage="NEW" />
        <StagePill stage="QUALIFIED" />
        <StagePill stage="ENGAGED" />
        <StagePill stage="PROPOSAL" />
      </Row>
    </Section>
  )
}

const CAPABILITY_PRESETS = ['IT', 'Konstruksi', 'Konsultansi', 'Pengadaan', 'Infrastruktur']

function ChipInputSection() {
  const [freeVal, setFreeVal] = useState<string[]>([])
  const [presetVal, setPresetVal] = useState<string[]>(['IT'])
  return (
    <Section title="ChipInput">
      <Row label="Free-form">
        <div className="w-full max-w-md">
          <Field label="Keyword pencarian">
            <ChipInput
              value={freeVal}
              onChange={setFreeVal}
              placeholder="Ketik lalu Enter…"
            />
          </Field>
        </div>
      </Row>
      <Row label="Dengan preset">
        <div className="w-full max-w-md">
          <Field label="Kapabilitas perusahaan">
            <ChipInput
              value={presetVal}
              onChange={setPresetVal}
              presets={CAPABILITY_PRESETS}
              placeholder="Tambah kapabilitas…"
            />
          </Field>
        </div>
      </Row>
      <Row label="Disabled">
        <div className="w-full max-w-md">
          <Field label="Kapabilitas (read-only)">
            <ChipInput
              value={['IT', 'Konstruksi']}
              onChange={() => {}}
              presets={CAPABILITY_PRESETS}
              disabled
            />
          </Field>
        </div>
      </Row>
    </Section>
  )
}

export default function ComponentsGallery() {
  return (
    <div className="max-w-7xl mx-auto p-6 space-y-12">
      <div>
        <h1 className="text-h1 font-bold text-fg">Component Gallery</h1>
        <p className="text-body text-fg-muted mt-1">Semua komponen UI ST-02.2 + ST-02.3 + ST-02.4</p>
      </div>

      {/* ── Form Fields ────────────────────────── */}
      <FormFieldsSection />

      {/* ── ChipInput ──────────────────────────── */}
      <ChipInputSection />

      {/* ── Toggle ─────────────────────────────── */}
      <ToggleSection />

      {/* ── Badge ──────────────────────────────── */}
      <BadgeSection />

      {/* ── Button ─────────────────────────────── */}
      <Section title="Button">
        <Row label="primary">
          <Button size="sm" variant="primary">Small</Button>
          <Button size="md" variant="primary">Medium</Button>
          <Button size="lg" variant="primary">Large</Button>
        </Row>
        <Row label="secondary">
          <Button size="sm" variant="secondary">Small</Button>
          <Button size="md" variant="secondary">Medium</Button>
          <Button size="lg" variant="secondary">Large</Button>
        </Row>
        <Row label="ghost">
          <Button size="sm" variant="ghost">Small</Button>
          <Button size="md" variant="ghost">Medium</Button>
          <Button size="lg" variant="ghost">Large</Button>
        </Row>
        <Row label="danger">
          <Button size="sm" variant="danger">Small</Button>
          <Button size="md" variant="danger">Medium</Button>
          <Button size="lg" variant="danger">Large</Button>
        </Row>
        <Row label="with icon">
          <Button leftIcon={<Plus className="w-4 h-4" />}>Add Item</Button>
          <Button variant="secondary" rightIcon={<ArrowRight className="w-4 h-4" />}>Next</Button>
          <Button variant="ghost" leftIcon={<Plus className="w-4 h-4" />} rightIcon={<ArrowRight className="w-4 h-4" />}>Both</Button>
        </Row>
        <Row label="loading">
          <Button loading>Saving…</Button>
          <Button loading variant="secondary">Loading</Button>
        </Row>
        <Row label="disabled">
          <Button disabled>Disabled primary</Button>
          <Button disabled variant="secondary">Disabled secondary</Button>
          <Button disabled variant="ghost">Disabled ghost</Button>
          <Button disabled variant="danger">Disabled danger</Button>
        </Row>
      </Section>

      {/* ── ST-02.3 Struktur ───────────────────── */}
      <StructureSection />

      {/* ── Table ──────────────────────────────── */}
      <TableSection />

      {/* ── Modal / Drawer / Toast / Confirm ── */}
      <OverlaySection />

      {/* ── Skeleton + EmptyState ───────────── */}
      <SkeletonSection />

      {/* ── ST-02.4 AI Components ─────────── */}
      <ScoreRingSection />
      <StatCardSection />
      <StreamingRiskSection />
      <DropzoneStepperSection />

      {/* ── ST-02.5 Shell Primitives ──────────── */}
      <PopoverMenuSection />
    </div>
  )
}

function StructureSection() {
  const [activeTab, setActiveTab] = useState('ringkasan')
  return (
    <Section title="Struktur & Feedback (ST-02.3)">
      {/* Card */}
      <Row label="Card — polos / header-body-footer">
        <Card className="w-48 p-4">
          <p className="text-body text-fg">Card sederhana</p>
        </Card>
        <Card className="w-56">
          <CardHeader><span className="font-semibold text-body text-fg">Header</span></CardHeader>
          <CardBody><p className="text-body text-fg-muted">Konten body kartu</p></CardBody>
          <CardFooter><span className="text-caption text-fg-subtle">Footer</span></CardFooter>
        </Card>
      </Row>

      {/* Tabs */}
      <Row label="Tabs">
        <div className="w-full max-w-lg">
          <Tabs
            tabs={[
              { id: 'ringkasan', label: 'Ringkasan', icon: <LayoutGrid className="w-4 h-4" /> },
              { id: 'analisaai', label: 'Analisa AI' },
              { id: 'playbook', label: 'Playbook', icon: <FileText className="w-4 h-4" /> },
              { id: 'timeline', label: 'Timeline' },
            ]}
            value={activeTab}
            onChange={setActiveTab}
          />
          <TabPanel id={activeTab} className="pt-3">
            <p className="text-body text-fg-muted">Konten tab: <strong>{activeTab}</strong></p>
          </TabPanel>
        </div>
      </Row>

      {/* Breadcrumb */}
      <Row label="Breadcrumb">
        <Breadcrumb
          items={[
            { label: 'Dashboard', href: '/' },
            { label: 'Tenders', href: '/tenders' },
            { label: 'Pengadaan Server Pusat Data' },
          ]}
        />
        <Breadcrumb items={[{ label: 'Dashboard', href: '/' }, { label: 'Chat' }]} />
      </Row>

      {/* Avatar */}
      <Row label="Avatar — inisial sm/md/lg">
        <Avatar name="Andi Kurniawan" size="sm" />
        <Avatar name="Andi Kurniawan" size="md" />
        <Avatar name="Andi Kurniawan" size="lg" />
        <Avatar name="Sales Pilot" size="md" />
      </Row>

      {/* Tooltip */}
      <Row label="Tooltip — top / bottom / right">
        <Tooltip content="Tooltip dari atas">
          <Button variant="secondary" size="sm">Hover saya (atas)</Button>
        </Tooltip>
        <Tooltip content="Tooltip dari bawah" side="bottom">
          <Button variant="secondary" size="sm">Hover saya (bawah)</Button>
        </Tooltip>
        <Tooltip content="Tooltip dari kanan" side="right">
          <Button variant="secondary" size="sm">Hover saya (kanan)</Button>
        </Tooltip>
      </Row>
    </Section>
  )
}

interface TenderRow {
  id: string
  judul: string
  buyer: string
  nilai: number
  status: string
  score: number
}

const tenderData: TenderRow[] = [
  { id: '1', judul: 'Pengadaan Server Pusat Data', buyer: 'Kemenkominfo', nilai: 5000000000, status: 'BIDDING', score: 86 },
  { id: '2', judul: 'Sistem Informasi Kesehatan', buyer: 'Kemenkes', nilai: 2500000000, status: 'QUALIFYING', score: 72 },
  { id: '3', judul: 'Digitalisasi Perpustakaan', buyer: 'Kemendikbud', nilai: 800000000, status: 'IDENTIFIED', score: 48 },
  { id: '4', judul: 'Aplikasi Absensi Pegawai', buyer: 'Kemenpan RB', nilai: 1200000000, status: 'SUBMITTED', score: 65 },
  { id: '5', judul: 'Infrastruktur Jaringan WAN', buyer: 'Pemerintah Daerah Jatim', nilai: 3000000000, status: 'BIDDING', score: 78 },
  { id: '6', judul: 'Platform E-Procurement', buyer: 'LKPP', nilai: 4200000000, status: 'QUALIFYING', score: 81 },
]

function TableSection() {
  return (
    <Section title="Table (TK-02.3.2)">
      <Row label="Sortable + pagination + kebab + sticky header">
        <div className="w-full border border-line rounded-card overflow-hidden">
          <Table<TenderRow>
            stickyHeader
            columns={[
              { key: 'judul', header: 'Judul', sortable: true },
              { key: 'buyer', header: 'Buyer', sortable: true },
              { key: 'status', header: 'Status' },
              { key: 'score', header: 'Fit Score', align: 'right', sortable: true,
                render: (r) => <ScoreBadge score={r.score} /> },
            ]}
            data={tenderData}
            rowKey={(r) => r.id}
            pageSize={3}
            kebabActions={(r) => [
              { label: 'Lihat detail', onClick: () => alert(`Lihat ${r.judul}`) },
              { label: 'Edit', onClick: () => alert(`Edit ${r.judul}`) },
              { label: 'Hapus', onClick: () => alert(`Hapus ${r.judul}`), danger: true },
            ]}
          />
        </div>
      </Row>
      <Row label="Loading state">
        <div className="w-full border border-line rounded-card overflow-hidden">
          <Table<TenderRow>
            columns={[{ key: 'judul', header: 'Judul' }, { key: 'buyer', header: 'Buyer' }]}
            data={[]}
            rowKey={(r) => r.id}
            loading
          />
        </div>
      </Row>
      <Row label="Empty state">
        <div className="w-full border border-line rounded-card overflow-hidden">
          <Table<TenderRow>
            columns={[{ key: 'judul', header: 'Judul' }, { key: 'buyer', header: 'Buyer' }]}
            data={[]}
            rowKey={(r) => r.id}
            empty="Belum ada tender. Coba ubah filter atau tambah tender baru."
          />
        </div>
      </Row>
    </Section>
  )
}

function OverlaySection() {
  const [modalOpen, setModalOpen] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)

  return (
    <Section title="Modal / Drawer / Toast / ConfirmDialog (TK-02.3.3)">
      <Row label="Modal">
        <Button onClick={() => setModalOpen(true)}>Buka Modal</Button>
        <Modal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          title="Contoh Modal"
          footer={
            <>
              <Button variant="secondary" onClick={() => setModalOpen(false)}>Batal</Button>
              <Button onClick={() => setModalOpen(false)}>Simpan</Button>
            </>
          }
        >
          <p className="text-body text-fg-muted">Ini adalah konten modal. Tekan Escape atau klik overlay untuk menutup.</p>
        </Modal>
      </Row>

      <Row label="Drawer (slide-over)">
        <Button variant="secondary" onClick={() => setDrawerOpen(true)}>Buka Drawer</Button>
        <Drawer
          open={drawerOpen}
          onClose={() => setDrawerOpen(false)}
          title="Form Tender"
          footer={
            <>
              <Button variant="secondary" onClick={() => setDrawerOpen(false)}>Batal</Button>
              <Button onClick={() => setDrawerOpen(false)}>Simpan</Button>
            </>
          }
        >
          <p className="text-body text-fg-muted">Konten drawer di sini. Formulir, detail, dsb.</p>
        </Drawer>
      </Row>

      <Row label="Toast — semua tone">
        <Button size="sm" variant="primary" onClick={() => toast.success('Tender berhasil disimpan')}>Success</Button>
        <Button size="sm" variant="danger" onClick={() => toast.error('Gagal menghubungi server')}>Error</Button>
        <Button size="sm" variant="secondary" onClick={() => toast.warning('Deadline tinggal 2 hari')}>Warning</Button>
        <Button size="sm" variant="ghost" onClick={() => toast.info('AI sedang menganalisa tender')}>Info</Button>
      </Row>

      <Row label="Confirmation dialog">
        <Button variant="danger" onClick={() => setConfirmOpen(true)}>Hapus Tender</Button>
        <ConfirmDialog
          open={confirmOpen}
          title="Hapus Tender?"
          description="Aksi ini tidak dapat dibatalkan. Data tender akan dihapus permanen."
          confirmLabel="Hapus"
          onConfirm={() => { setConfirmOpen(false); toast.success('Tender dihapus') }}
          onCancel={() => setConfirmOpen(false)}
        />
      </Row>
    </Section>
  )
}

function SkeletonSection() {
  return (
    <Section title="Skeleton + Empty state (TK-02.3.4)">
      <Row label="Skeleton — text / rect / circle">
        <div className="w-48">
          <Skeleton variant="text" />
        </div>
        <div className="w-48">
          <Skeleton variant="rect" />
        </div>
        <Skeleton variant="circle" className="w-10 h-10" />
      </Row>

      <Row label="SkeletonText — 3 baris / 1 baris">
        <div className="w-64">
          <SkeletonText lines={3} />
        </div>
        <div className="w-64">
          <SkeletonText lines={1} />
        </div>
      </Row>

      <Row label="Skeleton card">
        <div className="w-72 border border-line rounded-card p-4 space-y-3">
          <div className="flex items-center gap-3">
            <Skeleton variant="circle" className="w-10 h-10 shrink-0" />
            <div className="flex-1 space-y-2">
              <Skeleton variant="text" className="w-3/4" />
              <Skeleton variant="text" className="w-1/2 h-3" />
            </div>
          </div>
          <SkeletonText lines={3} />
          <Skeleton variant="rect" className="h-8" />
        </div>
      </Row>

      <Row label="Empty state — lengkap">
        <div className="border border-line rounded-card w-full">
          <EmptyState
            icon={<Search className="w-7 h-7" />}
            title="Belum ada tender ditemukan"
            description="AI belum menemukan tender yang cocok. Lengkapi Otak Agent agar hasil lebih relevan."
            action={<Button variant="primary">Lengkapi Otak Agent</Button>}
          />
        </div>
      </Row>

      <Row label="Empty state — sederhana (tanpa ikon & deskripsi)">
        <div className="border border-line rounded-card w-64">
          <EmptyState title="Tidak ada notifikasi" />
        </div>
      </Row>
    </Section>
  )
}


function ScoreRingSection() {
  return (
    <Section title="ScoreRing (TK-02.4.1)">
      <Row label="Varian skor — 48 (rose) / 56 (sky) / 72 (amber) / 86 (emerald)">
        <div className="flex items-center gap-6">
          <div className="flex flex-col items-center gap-1">
            <ScoreRing score={48} />
            <span className="text-caption text-fg-muted">48 — rose</span>
          </div>
          <div className="flex flex-col items-center gap-1">
            <ScoreRing score={56} />
            <span className="text-caption text-fg-muted">56 — sky</span>
          </div>
          <div className="flex flex-col items-center gap-1">
            <ScoreRing score={72} />
            <span className="text-caption text-fg-muted">72 — amber</span>
          </div>
          <div className="flex flex-col items-center gap-1">
            <ScoreRing score={86} />
            <span className="text-caption text-fg-muted">86 — emerald</span>
          </div>
        </div>
      </Row>
      <Row label="Varian ukuran — mini (32) / md (64) / lg (96)">
        <div className="flex items-end gap-4">
          <ScoreRing score={78} size={32} strokeWidth={4} />
          <ScoreRing score={78} size={64} />
          <ScoreRing score={78} size={96} strokeWidth={8} />
        </div>
      </Row>
      <Row label="Tanpa label">
        <ScoreRing score={65} showLabel={false} />
      </Row>
    </Section>
  )
}

function StatCardSection() {
  return (
    <Section title="StatCard + AiCallout (TK-02.4.2)">
      <Row label="StatCard — berbagai varian">
        <StatCard
          label="Total Pipeline"
          value="Rp 12,5 M"
          icon={<BarChart2 className="w-5 h-5" />}
          delta={{ value: '+18% MoM', trend: 'up' }}
          hint="vs bulan lalu"
        />
        <StatCard
          label="Prospek Aktif"
          value="24"
          icon={<Target className="w-5 h-5" />}
          delta={{ value: '-3 dari minggu lalu', trend: 'down' }}
        />
        <StatCard
          label="Penemuan Hari Ini"
          value="7 tender"
          hint="Diperbarui 10 menit lalu"
        />
        <StatCard label="Win Rate" value="68%" />
      </Row>

      <Row label="AiCallout — sederhana">
        <div className="w-full max-w-lg">
          <AiCallout
            title="Analisa AI"
            meta="Dibuat AI • Tinggi • 2 jam lalu"
          >
            Tender ini memiliki kesesuaian tinggi dengan kapabilitas IT Infrastruktur dan pengalaman proyek serupa di Kemenkominfo.
          </AiCallout>
        </div>
      </Row>

      <Row label="AiCallout — dengan 'Lihat alasan'">
        <div className="w-full max-w-lg">
          <AiCallout
            title="Fit Score: 86"
            meta="Dibuat AI • Sangat Tinggi • 5 menit lalu"
            reason={
              <ul className="list-disc list-inside space-y-1 text-body text-fg-muted">
                <li>Kapabilitas IT Infrastruktur cocok (skor: 9/10)</li>
                <li>Pengalaman proyek pemerintah serupa (skor: 8/10)</li>
                <li>Deadline masih 45 hari — cukup waktu (skor: 8/10)</li>
                <li>Nilai tender sesuai target Rp 1M–10M (skor: 9/10)</li>
              </ul>
            }
          >
            Tender pengadaan server memiliki profil risiko rendah dan kesesuaian kapabilitas sangat tinggi.
          </AiCallout>
        </div>
      </Row>
    </Section>
  )
}

function StreamingRiskSection() {
  const [streaming, setStreaming] = useState(false)
  const text = 'AI sedang menganalisa tender ini. Kesesuaian kapabilitas sangat tinggi. Perlu perhatian pada syarat pengalaman minimal 5 tahun.'

  return (
    <Section title="StreamingText + RiskFlag (TK-02.4.3)">
      <Row label="StreamingText — toggle streaming on/off">
        <div className="w-full max-w-lg">
          <div className="flex items-center gap-3 mb-3">
            <Toggle
              checked={streaming}
              onChange={setStreaming}
              label={streaming ? 'Streaming aktif' : 'Streaming nonaktif'}
              size="sm"
            />
          </div>
          <div className="p-3 border border-line rounded-card bg-surface text-body text-fg">
            <StreamingText text={text} streaming={streaming} />
          </div>
        </div>
      </Row>

      <Row label="RiskFlag — warning / danger">
        <RiskFlag label="Pengalaman sejenis diperlukan" severity="warning" />
        <RiskFlag label="Deadline mepet (12 hari)" severity="danger" />
        <RiskFlag label="Nilai estimasi di batas bawah" severity="warning" />
      </Row>

      <Row label="RiskFlagList — daftar">
        <RiskFlagList
          items={[
            { label: 'Syarat pengalaman min. 5 tahun', severity: 'warning' },
            { label: 'Persaingan tinggi (8 peserta)', severity: 'warning' },
            { label: 'Pembayaran via LC (risiko)', severity: 'danger' },
          ]}
        />
      </Row>
    </Section>
  )
}

const ONBOARDING_STEPS = [
  { id: 'profil', label: 'Profil Perusahaan' },
  { id: 'kapabilitas', label: 'Kapabilitas' },
  { id: 'target', label: 'Target & No-Go' },
  { id: 'aktivasi', label: 'Aktivasi Agent' },
]

function PopoverMenuSection() {
  const menuItems = [
    { label: 'Profil', onSelect: () => alert('Profil') },
    { label: 'Settings', onSelect: () => alert('Settings') },
    { label: 'Keluar', onSelect: () => alert('Keluar'), tone: 'danger' as const },
  ]

  return (
    <Section title="Popover + Menu (TK-02.5.3)">
      <Row label="Popover align=end (default)">
        <Popover
          align="end"
          trigger={
            <button className="px-3 py-1.5 rounded-btn border border-line text-body text-fg hover:bg-surface-subtle">
              Buka Popover
            </button>
          }
        >
          <div className="p-3 text-body text-fg">Konten popover bebas.</div>
        </Popover>
      </Row>

      <Row label="Popover align=start">
        <Popover
          align="start"
          trigger={
            <button className="px-3 py-1.5 rounded-btn border border-line text-body text-fg hover:bg-surface-subtle">
              Buka (kiri)
            </button>
          }
        >
          <div className="p-3 text-body text-fg">Konten popover rata kiri.</div>
        </Popover>
      </Row>

      <Row label="Menu — 3 item (keyboard nav ↑↓)">
        <Popover
          trigger={
            <button className="px-3 py-1.5 rounded-btn border border-line text-body text-fg hover:bg-surface-subtle">
              Menu ▾
            </button>
          }
        >
          <Menu items={menuItems} />
        </Popover>
      </Row>
    </Section>
  )
}

function DropzoneStepperSection() {
  const [step, setStep] = useState(1)
  const [files, setFiles] = useState<string[]>([])

  return (
    <Section title="FileDropzone + Stepper (TK-02.4.4)">
      <Row label="Stepper — 4 langkah">
        <div className="w-full max-w-lg">
          <Stepper steps={ONBOARDING_STEPS} current={step} />
          <div className="flex gap-2 mt-4">
            <Button size="sm" variant="secondary" disabled={step === 0} onClick={() => setStep((s) => Math.max(0, s - 1))}>Kembali</Button>
            <Button size="sm" disabled={step === ONBOARDING_STEPS.length} onClick={() => setStep((s) => Math.min(ONBOARDING_STEPS.length, s + 1))}>Lanjut</Button>
          </div>
        </div>
      </Row>

      <Row label="FileDropzone — PDF">
        <div className="w-full max-w-md">
          <FileDropzone
            onFiles={(f) => setFiles((prev) => [...prev, ...f.map((x) => x.name)])}
            onError={(msg) => alert(msg)}
          >
            {files.length > 0 && (
              <ul className="space-y-1">
                {files.map((name, i) => (
                  <li key={i} className="text-caption text-fg-muted flex items-center gap-1">
                    <span className="text-success">✓</span> {name}
                  </li>
                ))}
              </ul>
            )}
          </FileDropzone>
        </div>
      </Row>

      <Row label="FileDropzone — disabled">
        <div className="w-full max-w-md">
          <FileDropzone onFiles={() => {}} disabled />
        </div>
      </Row>
    </Section>
  )
}
