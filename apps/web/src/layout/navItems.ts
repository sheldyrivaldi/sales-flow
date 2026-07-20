import type { LucideIcon } from 'lucide-react'
import {
  LayoutDashboard,
  Compass,
  Briefcase,
  ClipboardCheck,
  UserCog,
  MessageSquare,
  Settings,
} from 'lucide-react'
import type { Capability } from '../lib/rbac'

export interface NavSubItem {
  path: string
  label: string
  capability?: Capability
}

export interface NavItem {
  path: string
  label: string
  icon: LucideIcon
  badge?: 'ai' | 'count'
  dividerBefore?: boolean
  capability?: Capability
  /** Sub-items rendered nested under this item in the sidebar (expand/collapse). */
  children?: NavSubItem[]
}

/** Struktur nav mengikuti siklus hidup penjualan proyek:
 *  Pra-Proyek (berburu peluang) → Proyek Berjalan (delivery yang dipantau
 *  sales) → Pasca-Proyek (feedback & analisa client). */
export const navItems: NavItem[] = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  {
    path: '/presales', label: 'Pra-Proyek', icon: Compass,
    children: [
      { path: '/discovery', label: 'Radar Tender', capability: 'RunDiscovery' },
      { path: '/tenders',   label: 'Tenders' },
      { path: '/events',    label: 'Events' },
      { path: '/prospects', label: 'Pipeline' },
      { path: '/playbooks', label: 'Playbooks' },
    ],
  },
  {
    path: '/ongoing', label: 'Proyek Berjalan', icon: Briefcase,
    children: [
      { path: '/ongoing/summary',  label: 'Ringkasan' },
      { path: '/ongoing/projects', label: 'Daftar Proyek' },
    ],
  },
  {
    path: '/postproject', label: 'Pasca-Proyek', icon: ClipboardCheck,
    children: [
      { path: '/postproject/feedback',  label: 'Feedback Client' },
      { path: '/postproject/analytics', label: 'Analisa Feedback' },
    ],
  },
  { path: '/chat', label: 'Chat', icon: MessageSquare, badge: 'ai' },
  { path: '/users', label: 'User Management', icon: UserCog, dividerBefore: true, capability: 'ManageUsers' },
  {
    path: '/settings', label: 'Settings', icon: Settings,
    children: [
      { path: '/settings/profile', label: 'Profile' },
      { path: '/settings/ai-agent', label: 'AI Agent' },
    ],
  },
]
