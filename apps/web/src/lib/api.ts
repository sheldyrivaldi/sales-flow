export interface ApiErrorShape {
  code: string
  message: string
}

export class ApiError extends Error {
  code: string
  status: number

  constructor(message: string, code: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

// ---- Types ----------------------------------------------------------------

export interface UserDTO {
  id: string
  email: string
  name: string
  role: 'SALES' | 'OPS' | 'MANAGER' | 'ADMIN'
  active: boolean
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  user: UserDTO
}

export interface RefreshResponse {
  access_token: string
  refresh_token: string
}

// ---- Injection callbacks ---------------------------------------------------

let getAccessToken: () => string | null = () => null
let getRefreshToken: () => string | null = () => null
let onTokens: (access: string, refresh: string) => void = () => {}
let onUnauthorized: () => void = () => {}

export function configureApi(opts: {
  getAccessToken?: () => string | null
  getRefreshToken?: () => string | null
  onTokens?: (access: string, refresh: string) => void
  onUnauthorized?: () => void
}) {
  if (opts.getAccessToken) getAccessToken = opts.getAccessToken
  if (opts.getRefreshToken) getRefreshToken = opts.getRefreshToken
  if (opts.onTokens) onTokens = opts.onTokens
  if (opts.onUnauthorized) onUnauthorized = opts.onUnauthorized
}

// ---- Single-flight refresh ------------------------------------------------

let refreshPromise: Promise<void> | null = null

export async function ensureRefreshed(): Promise<void> {
  if (refreshPromise) return refreshPromise

  refreshPromise = (async () => {
    const rt = getRefreshToken()
    if (!rt) throw new ApiError('Sesi berakhir, silakan login ulang', 'UNAUTHORIZED', 401)

    const res = await fetch('/api/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: rt }),
    })

    if (!res.ok) {
      throw new ApiError('Sesi berakhir, silakan login ulang', 'UNAUTHORIZED', 401)
    }

    const data: RefreshResponse = await res.json()
    onTokens(data.access_token, data.refresh_token)
  })()
    .catch((err) => {
      onUnauthorized()
      throw err
    })
    .finally(() => {
      refreshPromise = null
    })

  return refreshPromise
}

// ---- Core fetch -----------------------------------------------------------

const AUTH_PATHS = ['/api/auth/login', '/api/auth/refresh']

async function rawFetch(path: string, init: RequestInit = {}, withAuth = true): Promise<Response> {
  const headers = new Headers(init.headers)

  if (!(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  if (withAuth) {
    const token = getAccessToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(path, { ...init, headers })

  if (!res.ok) {
    let code = 'REQUEST_ERROR'
    let message = `HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body?.error?.code) code = body.error.code
      if (body?.error?.message) message = body.error.message
    } catch {
      // ignore JSON parse error — use defaults above
    }
    throw new ApiError(message, code, res.status)
  }

  return res
}

// ---- Public fetch wrapper -------------------------------------------------

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const isAuthPath = AUTH_PATHS.some((p) => path.startsWith(p))

  try {
    const res = await rawFetch(path, init, !isAuthPath)
    return res.json() as Promise<T>
  } catch (err) {
    if (err instanceof ApiError && err.status === 401 && !isAuthPath) {
      await ensureRefreshed()
      try {
        // retry once with fresh token
        const res = await rawFetch(path, init, true)
        return res.json() as Promise<T>
      } catch (retryErr) {
        if (retryErr instanceof ApiError && retryErr.status === 401) {
          onUnauthorized()
        }
        throw retryErr
      }
    }
    throw err
  }
}

// ---- Auth helpers ---------------------------------------------------------

export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await rawFetch('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
  return res.json()
}

export async function refreshTokens(refreshToken: string): Promise<RefreshResponse> {
  const res = await rawFetch('/api/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
  return res.json()
}

export async function getMe(): Promise<UserDTO> {
  return apiFetch<UserDTO>('/api/me')
}

// Exposes the current access token for raw fetch calls (e.g. SSE streaming)
// that cannot go through apiFetch because it parses JSON.
export function currentAccessToken(): string | null {
  return getAccessToken()
}
