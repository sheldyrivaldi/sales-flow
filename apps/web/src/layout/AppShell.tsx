import { useState, useEffect } from 'react'
import { Outlet } from 'react-router'
import Sidebar from './Sidebar'
import Topbar from './Topbar'
import AskAIButton from '../components/AskAIButton'
import AskAIDrawer from '../components/AskAIDrawer'

export default function AppShell() {
  const [collapsed, setCollapsed] = useState(() => window.innerWidth < 1024)

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 1023px)')
    const handler = (e: MediaQueryListEvent) => setCollapsed(e.matches)
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])

  const toggleCollapsed = () => setCollapsed((prev) => !prev)

  return (
    <div className="flex h-screen overflow-hidden bg-surface-muted">
      <Sidebar collapsed={collapsed} onToggle={toggleCollapsed} />
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        <Topbar collapsed={collapsed} onToggleSidebar={toggleCollapsed} />
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
          {/* Floating "Tanya AI" — EP-04 ST-04.6 */}
          <AskAIButton />
          <AskAIDrawer />
        </main>
      </div>
    </div>
  )
}
