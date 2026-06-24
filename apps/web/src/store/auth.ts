import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { configureApi } from '../lib/api'
import type { UserDTO } from '../lib/api'

export type Role = 'SALES' | 'OPS' | 'MANAGER' | 'ADMIN'

export interface AuthUser {
  id: string
  email: string
  name: string
  role: Role
  active: boolean
}

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  user: AuthUser | null
  setSession: (session: { accessToken: string; refreshToken: string; user: UserDTO }) => void
  setTokens: (accessToken: string, refreshToken: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      user: null,

      setSession: ({ accessToken, refreshToken, user }) =>
        set({ accessToken, refreshToken, user: user as AuthUser }),

      setTokens: (accessToken, refreshToken) => set({ accessToken, refreshToken }),

      logout: () => set({ accessToken: null, refreshToken: null, user: null }),
    }),
    {
      name: 'salespilot-auth',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        user: state.user,
      }),
    }
  )
)

// Wire api.ts to use auth store — runs once when module is evaluated.
configureApi({
  getAccessToken: () => useAuthStore.getState().accessToken,
  getRefreshToken: () => useAuthStore.getState().refreshToken,
  onTokens: (access, refresh) => useAuthStore.getState().setTokens(access, refresh),
  onUnauthorized: () => {
    useAuthStore.getState().logout()
    window.location.assign('/login')
  },
})

export function useIsAuthenticated(): boolean {
  return useAuthStore((s) => !!s.accessToken)
}
