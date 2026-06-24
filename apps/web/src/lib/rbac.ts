import type { Role } from '../store/auth'

export type Capability =
  | 'ViewData'
  | 'CRUDData'
  | 'EditProfile'
  | 'RunDiscovery'
  | 'UseAI'
  | 'MakeDecision'
  | 'ManageUsers'

// Mirror of internal/auth/rbac.go capabilityRoles — keep in sync.
const capabilityRoles: Record<Capability, Role[]> = {
  ViewData: ['SALES', 'OPS', 'MANAGER', 'ADMIN'],
  CRUDData: ['SALES', 'OPS', 'MANAGER', 'ADMIN'],
  EditProfile: ['OPS', 'MANAGER', 'ADMIN'],
  RunDiscovery: ['OPS', 'MANAGER', 'ADMIN'],
  UseAI: ['SALES', 'OPS', 'MANAGER', 'ADMIN'],
  MakeDecision: ['SALES', 'OPS', 'MANAGER', 'ADMIN'],
  ManageUsers: ['ADMIN'],
}

export function can(role: Role | null | undefined, cap: Capability): boolean {
  if (!role) return false
  return capabilityRoles[cap].includes(role)
}
