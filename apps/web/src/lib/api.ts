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

// parseJsonBody handles 204/empty-body responses (every DELETE handler in
// this API returns 204 No Content) — Response.json() on an empty body throws
// SyntaxError, which would otherwise make every successful delete look like
// a failed mutation to callers.
async function parseJsonBody<T>(res: Response): Promise<T> {
  if (res.status === 204) return undefined as T
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const isAuthPath = AUTH_PATHS.some((p) => path.startsWith(p))

  try {
    const res = await rawFetch(path, init, !isAuthPath)
    return parseJsonBody<T>(res)
  } catch (err) {
    if (err instanceof ApiError && err.status === 401 && !isAuthPath) {
      await ensureRefreshed()
      try {
        // retry once with fresh token
        const res = await rawFetch(path, init, true)
        return parseJsonBody<T>(res)
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

// ---- Query helpers --------------------------------------------------------

// buildQueryString encodes a filter object into a `?a=1&b=2` suffix, skipping
// undefined/null/empty-string values. Shared by all api/* modules so query
// encoding lives in one place. Booleans (incl. false) are kept.
export function buildQueryString(
  params: Record<string, string | number | boolean | undefined | null>,
): string {
  const sp = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    sp.set(key, String(value))
  }
  const qs = sp.toString()
  return qs ? `?${qs}` : ''
}
