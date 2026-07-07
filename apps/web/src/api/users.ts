import { useQuery } from '@tanstack/react-query'
import { apiFetch } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────

export type UserRole = 'SALES' | 'OPS' | 'MANAGER' | 'ADMIN'

export interface User {
  id: string
  email: string
  name: string
  role: UserRole
  active: boolean
}

export interface UserListResponse {
  items: User[]
  total: number
  page: number
  page_size: number
}

// ── Query Hooks ───────────────────────────────────────────────────────────────

/** Direktori user (read-only, semua role). Dipakai untuk resolve nama owner
 * di UI (mis. kartu/detail prospect) — jarang berubah, cache lebih lama. */
export function useUsers() {
  return useQuery({
    queryKey: ['users'],
    queryFn: () => apiFetch<UserListResponse>('/api/users?page_size=100'),
    staleTime: 5 * 60 * 1000,
  })
}
