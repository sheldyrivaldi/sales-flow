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
]

export const DEFAULT_VALUE_MIN = 1_000_000_000
export const DEFAULT_DEADLINE_MIN_DAYS = 7
