import { Routes, Route, Navigate } from 'react-router'
import AppShell from './layout/AppShell'
import RequireAuth from './components/RequireAuth'
import ComponentsGallery from './dev/ComponentsGallery'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Chat from './pages/Chat'
import TendersPage from './pages/tenders/TendersPage'
import TenderDetail from './pages/tenders/TenderDetail'
import EventList from './pages/events/EventList'
import EventDetail from './pages/events/EventDetail'
import ProspectBoard from './pages/prospects/ProspectBoard'
import Onboarding from './pages/onboarding/Onboarding'
import PlaybooksIndex from './pages/playbooks/PlaybooksIndex'
import ReportsPage from './pages/reports/ReportsPage'
import UserManagement from './pages/users/UserManagement'
import SettingsLayout from './pages/settings/SettingsLayout'
import SettingsProfile from './pages/settings/SettingsProfile'
import SettingsAiHermes from './pages/settings/SettingsAiHermes'
import OngoingSummary from './pages/ongoing/OngoingSummary'
import OngoingProjects from './pages/ongoing/OngoingProjects'
import FeedbackFormsList from './pages/postproject/feedback/FeedbackFormsList'
import FeedbackFormBuilder from './pages/postproject/feedback/FeedbackFormBuilder'
import FeedbackFormDetail from './pages/postproject/feedback/FeedbackFormDetail'
import PublicFeedback from './pages/publicfeedback/PublicFeedback'
import PublicFeedbackForm from './pages/publicfeedback/PublicFeedbackForm'

export default function AppRoutes() {
  return (
    <Routes>
      {/* Dev */}
      <Route path="/dev/components" element={<ComponentsGallery />} />

      {/* Standalone (tanpa shell) */}
      <Route path="/login" element={<Login />} />
      <Route path="/onboarding" element={<Onboarding />} />
      {/* Form feedback publik untuk client — tanpa login */}
      <Route path="/f/:token" element={<PublicFeedback />} />
      <Route path="/form/:slug" element={<PublicFeedbackForm />} />

      {/* Halaman utama — dalam shell */}
      <Route element={<RequireAuth />}>
        <Route element={<AppShell />}>
          <Route index element={<Dashboard />} />
          {/* /discovery lama (menu "Radar Tender" terpisah) kini menyatu ke
              /tenders — redirect agar bookmark/link lama tak 404. */}
          <Route path="discovery" element={<Navigate to="/tenders" replace />} />
          <Route path="tenders" element={<TendersPage />} />
          <Route path="tenders/:id" element={<TenderDetail />} />
          <Route path="events" element={<EventList />} />
          <Route path="events/:id" element={<EventDetail />} />
          <Route path="prospects" element={<ProspectBoard />} />
          <Route path="playbooks" element={<PlaybooksIndex />} />
          <Route path="reports" element={<ReportsPage />} />
          <Route path="ongoing">
            <Route index element={<Navigate to="summary" replace />} />
            <Route path="summary" element={<OngoingSummary />} />
            <Route path="projects" element={<OngoingProjects />} />
          </Route>
          <Route path="postproject">
            <Route index element={<Navigate to="feedback" replace />} />
            <Route path="feedback" element={<FeedbackFormsList />} />
            <Route path="feedback/new" element={<FeedbackFormBuilder />} />
            <Route path="feedback/:id" element={<FeedbackFormDetail />} />
            <Route path="feedback/:id/edit" element={<FeedbackFormBuilder />} />
            {/* Link lama: hasil kini menyatu di halaman detail. */}
            <Route path="feedback/:id/results" element={<Navigate to=".." replace relative="path" />} />
            {/* Analisa Feedback kini bagian dari detail form → redirect ke daftar. */}
            <Route path="analytics" element={<Navigate to="/postproject/feedback" replace />} />
          </Route>
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
