import { Outlet } from 'react-router'

// Sub-navigation (Profile / AI Agent) lives in the sidebar itself
// (layout/Sidebar.tsx, via navItems.ts children) — this layout only supplies
// the shared route frame for /settings/*.
export default function SettingsLayout() {
  return <Outlet />
}
