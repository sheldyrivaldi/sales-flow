import type { LucideIcon } from 'lucide-react'
import {
  LayoutDashboard,
  Sparkles,
  FileText,
  Calendar,
  Users,
  BookOpen,
  BarChart3,
  MessageSquare,
  Brain,
  Settings,
} from 'lucide-react'
import type { Capability } from '../lib/rbac'

export interface NavItem {
  path: string
  label: string
  icon: LucideIcon
  badge?: 'ai' | 'count'
  dividerBefore?: boolean
  capability?: Capability
}

export const navItems: NavItem[] = [
  { path: '/',            label: 'Dashboard',   icon: LayoutDashboard },
  { path: '/discovery',  label: 'Penemuan AI', icon: Sparkles,      badge: 'count', capability: 'RunDiscovery' },
  { path: '/tenders',    label: 'Tenders',      icon: FileText },
  { path: '/events',     label: 'Events',       icon: Calendar },
  { path: '/prospects',  label: 'Prospects',    icon: Users },
  { path: '/playbooks',  label: 'Playbooks',    icon: BookOpen },
  { path: '/reports',    label: 'Reports',      icon: BarChart3 },
  { path: '/chat',       label: 'Chat',         icon: MessageSquare, badge: 'ai' },
  { path: '/otak-agent', label: 'Otak Agent',   icon: Brain,         dividerBefore: true, capability: 'EditProfile' },
  { path: '/settings',   label: 'Settings',     icon: Settings },
]
