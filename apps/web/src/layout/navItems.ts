import type { LucideIcon } from 'lucide-react'
import {
  LayoutDashboard,
  Sparkles,
  FileText,
  Calendar,
  Users,
  UserCog,
  BookOpen,
  BarChart3,
  MessageSquare,
  Settings,
} from 'lucide-react'
import type { Capability } from '../lib/rbac'

export interface NavSubItem {
  path: string
  label: string
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

export const navItems: NavItem[] = [
  { path: '/',            label: 'Dashboard',   icon: LayoutDashboard },
  { path: '/discovery',  label: 'Radar Tender', icon: Sparkles,      badge: 'count', capability: 'RunDiscovery' },
  { path: '/tenders',    label: 'Tenders',      icon: FileText },
  { path: '/events',     label: 'Events',       icon: Calendar },
  { path: '/prospects',  label: 'Pipeline',     icon: Users },
  { path: '/playbooks',  label: 'Playbooks',    icon: BookOpen },
  { path: '/reports',    label: 'Reports',      icon: BarChart3 },
  { path: '/chat',       label: 'Chat',         icon: MessageSquare, badge: 'ai' },
  { path: '/users',      label: 'User Management', icon: UserCog,    dividerBefore: true, capability: 'ManageUsers' },
  {
    path: '/settings', label: 'Settings', icon: Settings,
    children: [
      { path: '/settings/profile', label: 'Profile' },
      { path: '/settings/ai-agent', label: 'AI Agent' },
    ],
  },
]
