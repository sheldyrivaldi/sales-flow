// Kamus label Bahasa Indonesia untuk seluruh aplikasi.
// Konsumsi via `id.key` atau helper `t(key)` di bawah.

export const id = {
  // ── Aksi umum ─────────────────────────────────
  simpan: 'Simpan',
  batal: 'Batal',
  hapus: 'Hapus',
  edit: 'Edit',
  tutup: 'Tutup',
  tambah: 'Tambah',
  konfirmasi: 'Konfirmasi',
  lanjut: 'Lanjut',
  kembali: 'Kembali',
  cari: 'Cari',
  filter: 'Filter',
  reset: 'Reset',
  ekspor: 'Ekspor',
  salin: 'Salin',
  kirim: 'Kirim',
  jalankan: 'Jalankan',
  analisa: 'Analisa',
  baru: 'Baru',

  // ── Status / label state ───────────────────────
  loading: 'Memuat…',
  menyimpan: 'Menyimpan…',
  berhasil: 'Berhasil',
  gagal: 'Gagal',
  tidakAdaData: 'Tidak ada data',
  belumAdaNotifikasi: 'Belum ada notifikasi',
  halamanBelumTersedia: 'Halaman ini akan dibangun pada epik berikutnya.',
  errorGeneral: 'Terjadi kesalahan. Silakan coba lagi.',
  agentTidakTersedia: 'Agent tidak tersedia saat ini.',

  // ── Navigasi (selaras navItems.ts) ────────────
  nav: {
    dashboard: 'Dashboard',
    penemuanAi: 'Penemuan AI',
    tenders: 'Tenders',
    events: 'Events',
    prospects: 'Prospects',
    playbooks: 'Playbooks',
    reports: 'Reports',
    chat: 'Chat',
    otakAgent: 'Otak Agent',
    settings: 'Settings',
  },

  // ── Tender ────────────────────────────────────
  tender: {
    judul: 'Judul Tender',
    pembeli: 'Pembeli / Instansi',
    nilai: 'Nilai Estimasi',
    deadline: 'Deadline Pengajuan',
    status: 'Status',
    origin: 'Asal',
    fitScore: 'Fit Score',
    rekomendasiAksi: 'Rekomendasi',
  },

  // ── Prospek ───────────────────────────────────
  prospek: {
    nama: 'Nama Prospek',
    perusahaan: 'Perusahaan',
    stage: 'Tahap',
    owner: 'Owner',
    nilaiEstimasi: 'Nilai Estimasi',
    sumber: 'Sumber',
  },

  // ── Auth ──────────────────────────────────────
  auth: {
    masuk: 'Masuk',
    email: 'Email',
    kataSandi: 'Kata Sandi',
    lupaKataSandi: 'Lupa kata sandi?',
    akunDikelolaAdmin: 'Akun dikelola oleh Admin internal.',
    loginGagal: 'Email atau kata sandi salah.',
  },

  // ── Topbar ────────────────────────────────────
  topbar: {
    cariPlaceholder: 'Cari…',
    notifikasi: 'Notifikasi',
    profil: 'Profil',
    keluar: 'Keluar',
    toggleSidebar: 'Toggle sidebar',
  },

  // ── AI / Agent ────────────────────────────────
  ai: {
    dibuatAi: 'Dibuat AI',
    lihatAlasan: 'Lihat alasan',
    analisaUlang: 'Analisa ulang',
    agentBelajar: 'Asisten belajar dari aktivitas & hasil kamu',
    aiBelajarDariIni: 'AI akan belajar dari ini',
    sedangMencari: 'AI sedang mencari di {n} sumber…',
  },
} as const

export type I18nKey = keyof typeof id

export function t(key: string): string {
  const keys = key.split('.')
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let node: any = id
  for (const k of keys) {
    if (node && typeof node === 'object' && k in node) {
      node = node[k]
    } else {
      return key
    }
  }
  return typeof node === 'string' ? node : key
}
