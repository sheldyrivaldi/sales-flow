import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
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

export interface UserCreateRequest {
  email: string
  name: string
  role: UserRole
  password: string
}

export interface UserUpdateRequest {
  name?: string
  role?: UserRole
  active?: boolean
}

export interface ResetPasswordResponse {
  password?: string
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

// ── Mutation Hooks (Settings → Users tab, ADMIN only — EP-18 TK-18.2.1) ──────

export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: UserCreateRequest) =>
      apiFetch<User>('/api/users', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}

export function useUpdateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: UserUpdateRequest }) =>
      apiFetch<User>(`/api/users/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}

export function useResetPassword() {
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<ResetPasswordResponse>(`/api/users/${id}/reset-password`, { method: 'POST' }),
  })
}
