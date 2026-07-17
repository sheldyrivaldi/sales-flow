import { Routes, Route, Navigate } from 'react-router'
import AppShell from './layout/AppShell'
import RequireAuth from './components/RequireAuth'
import ComponentsGallery from './dev/ComponentsGallery'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Chat from './pages/Chat'
import TenderList from './pages/tenders/TenderList'
import TenderDetail from './pages/tenders/TenderDetail'
import EventList from './pages/events/EventList'
import EventDetail from './pages/events/EventDetail'
import ProspectBoard from './pages/prospects/ProspectBoard'
import Onboarding from './pages/onboarding/Onboarding'
import DiscoveryInbox from './pages/discovery/DiscoveryInbox'
import PlaybooksIndex from './pages/playbooks/PlaybooksIndex'
import ReportsPage from './pages/reports/ReportsPage'
import UserManagement from './pages/users/UserManagement'
import SettingsLayout from './pages/settings/SettingsLayout'
import SettingsProfile from './pages/settings/SettingsProfile'
import SettingsAiHermes from './pages/settings/SettingsAiHermes'

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
          <Route index element={<Dashboard />} />
          <Route path="discovery" element={<DiscoveryInbox />} />
          <Route path="tenders" element={<TenderList />} />
          <Route path="tenders/:id" element={<TenderDetail />} />
          <Route path="events" element={<EventList />} />
          <Route path="events/:id" element={<EventDetail />} />
          <Route path="prospects" element={<ProspectBoard />} />
          <Route path="playbooks" element={<PlaybooksIndex />} />
          <Route path="reports" element={<ReportsPage />} />
          <Route path="chat" element={<Chat />} />
          <Route path="users" element={<UserManagement />} />
          <Route path="settings" element={<SettingsLayout />}>
            <Route index element={<Navigate to="profile" replace />} />
            <Route path="profile" element={<SettingsProfile />} />
            <Route path="ai-agent" element={<SettingsAiHermes />} />
          </Route>
        </Route>
      </Route>

      {/* Catch-all */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
