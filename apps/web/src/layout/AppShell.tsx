import { useState, useEffect } from 'react'
import { Outlet, useLocation } from 'react-router'
import Sidebar from './Sidebar'
import Topbar from './Topbar'
import Breadcrumbs from './Breadcrumbs'
import AskAIButton from '../components/AskAIButton'
import AskAIDrawer from '../components/AskAIDrawer'
import DiscoveryRunWatcher from '../components/DiscoveryRunWatcher'

export default function AppShell() {
  const [collapsed, setCollapsed] = useState(() => window.innerWidth < 1024)
  const { pathname } = useLocation()

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 1023px)')
    const handler = (e: MediaQueryListEvent) => setCollapsed(e.matches)
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])

  const toggleCollapsed = () => setCollapsed((prev) => !prev)

  return (
    <div className="flex h-screen overflow-hidden bg-canvas">
      <Sidebar collapsed={collapsed} onToggle={toggleCollapsed} />
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        <Topbar collapsed={collapsed} onToggleSidebar={toggleCollapsed} />
        <Breadcrumbs />
        <main className="flex-1 overflow-y-auto p-6">
          {/* Route transition: key per pathname me-remount wrapper sehingga
              animasi fade+slide 8px jalan tiap pindah halaman. AskAI berada
              DI LUAR wrapper ber-key agar drawer tidak ikut ter-reset. */}
          <div key={pathname} className="animate-page-enter h-full">
            <Outlet />
          </div>
          {/* Floating "Tanya AI" — EP-04 ST-04.6 */}
          <AskAIButton />
          <AskAIDrawer />
          {/* Pengawas crawling Radar Tender lintas halaman */}
          <DiscoveryRunWatcher />
        </main>
      </div>
    </div>
  )
}
