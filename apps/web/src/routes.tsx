import { Routes, Route, Navigate } from 'react-router'
import AppShell from './layout/AppShell'
import RequireAuth from './components/RequireAuth'
import PagePlaceholder from './components/PagePlaceholder'
import OtakAgentBanner from './components/OtakAgentBanner'
import ComponentsGallery from './dev/ComponentsGallery'
import Login from './pages/Login'
import Chat from './pages/Chat'
import TenderList from './pages/tenders/TenderList'
import TenderDetail from './pages/tenders/TenderDetail'
import EventList from './pages/events/EventList'
import EventDetail from './pages/events/EventDetail'
import ProspectBoard from './pages/prospects/ProspectBoard'
import Onboarding from './pages/onboarding/Onboarding'
import OtakAgent from './pages/profile/OtakAgent'

export default function AppRoutes() {
  return (
    <Routes>
      {/* Dev */}
      <Route path="/dev/components" element={<ComponentsGallery />} />

      {/* Standalone (tanpa shell) */}
      <Route path="/login" element={<Login />} />
      <Route path="/onboarding" element={<Onboarding />} />

      {/* Halaman utama — dalam shell */}
      <Route element={<RequireAuth />}>
        <Route element={<AppShell />}>
          <Route
            index
            element={
              <div className="flex flex-col gap-4">
                <OtakAgentBanner />
                <PagePlaceholder title="Dashboard" />
              </div>
            }
          />
          <Route path="discovery" element={<PagePlaceholder title="Penemuan AI" />} />
          <Route path="tenders" element={<TenderList />} />
          <Route path="tenders/:id" element={<TenderDetail />} />
          <Route path="events" element={<EventList />} />
          <Route path="events/:id" element={<EventDetail />} />
          <Route path="prospects" element={<ProspectBoard />} />
          <Route path="playbooks" element={<PagePlaceholder title="Playbooks" />} />
          <Route path="reports" element={<PagePlaceholder title="Reports" />} />
          <Route path="chat" element={<Chat />} />
          <Route path="otak-agent" element={<OtakAgent />} />
          <Route path="settings" element={<PagePlaceholder title="Settings" />} />
        </Route>
      </Route>

      {/* Catch-all */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
