import { useState } from 'react'
import Tabs, { TabPanel } from '../../components/ui/Tabs'
import { useAuthStore } from '../../store/auth'
import { can } from '../../lib/rbac'
import ProfileTab from './ProfileTab'
import WorkspaceTab from './WorkspaceTab'
import UsersTab from './UsersTab'
import AiHermesTab from './AiHermesTab'
import AiProviderTab from './AiProviderTab'

type SettingsTabId = 'profile' | 'workspace' | 'users' | 'ai-hermes' | 'ai-provider'

export default function SettingsPage() {
  const role = useAuthStore((s) => s.user?.role)
  const isAdmin = can(role, 'ManageUsers')

  const tabs = [
    { id: 'profile', label: 'Profil' },
    { id: 'workspace', label: 'Workspace' },
    ...(isAdmin ? [{ id: 'users', label: 'Users' }] : []),
    { id: 'ai-hermes', label: 'AI/Hermes' },
    // AI Provider config is sensitive (provider/model/API key) — ADMIN only,
    // same gate the backend enforces (CapManageUsers on /settings/ai).
    ...(isAdmin ? [{ id: 'ai-provider', label: 'AI Provider' }] : []),
  ]

  const [active, setActive] = useState<SettingsTabId>('profile')

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-h2 font-semibold text-fg">Settings</h1>

      <Tabs tabs={tabs} value={active} onChange={(id) => setActive(id as SettingsTabId)} />

      <TabPanel id="profile" className={active === 'profile' ? '' : 'hidden'}>
        <ProfileTab />
      </TabPanel>
      <TabPanel id="workspace" className={active === 'workspace' ? '' : 'hidden'}>
        <WorkspaceTab />
      </TabPanel>
      {isAdmin && (
        <TabPanel id="users" className={active === 'users' ? '' : 'hidden'}>
          <UsersTab />
        </TabPanel>
      )}
      <TabPanel id="ai-hermes" className={active === 'ai-hermes' ? '' : 'hidden'}>
        <AiHermesTab />
      </TabPanel>
      {isAdmin && (
        <TabPanel id="ai-provider" className={active === 'ai-provider' ? '' : 'hidden'}>
          <AiProviderTab />
        </TabPanel>
      )}
    </div>
  )
}
