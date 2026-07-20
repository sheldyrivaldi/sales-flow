// Preset chip catalogs for the Otak Agent profile form (Design §4.13).
// Procurement types mirror the backend default (profile_service.go
// defaultProcurementTypes) so FE/BE taxonomy stays aligned.

export const CAPABILITY_PRESETS = [
  'Web App',
  'System Integration',
  'AI/Automation',
  'Data/BI',
  'Cloud/DevOps',
  'Maintenance',
  'QA',
]

export const COUNTRY_PRESETS = ['Indonesia']

export const INDUSTRY_PRESETS = [
  'Government',
  'Finance',
  'Manufacturing',
  'Retail',
  'Healthcare',
  'Education',
]

export const PROCUREMENT_TYPE_PRESETS = [
  'Barang',
  'Jasa Konsultansi',
  'Jasa Lainnya',
  'Pekerjaan Konstruksi',
]

export const NOGO_PRESET_FLAGS = [
  'Hardware murni',
  'Onsite penuh luar kota',
  'Embedded/IoT',
  'Kontrak < 3 bulan',
  'Cybersecurity/pentest besar tanpa partner',
  'Desktop-only legacy tanpa modernisasi',
  'Sertifikasi wajib tidak dimiliki',
  'Unpaid PoC besar',
  'Payment 100% setelah delivery',
]

export const DEFAULT_VALUE_MIN = 1_000_000_000
export const DEFAULT_DEADLINE_MIN_DAYS = 7

export const DOCUMENT_LANGUAGE_PRESETS = ['Indonesia', 'Inggris']

export const WORK_MODEL_OPTIONS = ['Remote', 'Hybrid', 'Onsite Terbatas']

export const DECISION_MAKER_PRESETS = [
  'IT',
  'Digital Transformation',
  'Procurement',
  'Operations',
  'Finance',
  'Business Unit',
]

/** Dokumen pendukung tender yang umum diminta panitia pengadaan di
 * Indonesia — preset untuk kartu "Dokumen Pendukung Tender". */
export const SUPPORT_DOCUMENT_PRESETS = [
  'NIB',
  'NPWP',
  'Akta Pendirian',
  'SIUP/OSS',
  'ISO 9001',
  'ISO 27001',
  'Laporan Keuangan Audited',
  'Surat Referensi Kerja',
  'SKK/SBU',
  'KTA KADIN',
  'BPJS Ketenagakerjaan',
  'SPT Tahunan',
]
