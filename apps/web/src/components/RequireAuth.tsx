import { Navigate, Outlet, useLocation } from 'react-router'
import { useIsAuthenticated } from '../store/auth'

export default function RequireAuth() {
  const authed = useIsAuthenticated()
  const location = useLocation()

  if (!authed) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />
  }

  return <Outlet />
}
