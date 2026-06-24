import { useAuthStore } from '../store/auth'
import { can } from './rbac'
import type { Capability } from './rbac'

export function useCan(cap: Capability): boolean {
  const role = useAuthStore((s) => s.user?.role)
  return can(role, cap)
}
