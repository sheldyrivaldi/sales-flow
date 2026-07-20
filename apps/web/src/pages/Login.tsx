import { useState, type FormEvent } from 'react'
import { Navigate, useNavigate, useLocation } from 'react-router'
import { Eye, EyeOff } from 'lucide-react'
import Button from '../components/ui/Button'
import Input from '../components/ui/Input'
import Field from '../components/ui/Field'
import { login, ApiError } from '../lib/api'
import { useAuthStore, useIsAuthenticated } from '../store/auth'
import { LogoBadge, LogoWordmark } from '../components/Logo'

export default function Login() {
  const navigate = useNavigate()
  const location = useLocation()
  const authed = useIsAuthenticated()
  const setSession = useAuthStore((s) => s.setSession)

  const from = (location.state as { from?: string } | null)?.from ?? '/'

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (authed) {
    return <Navigate to={from} replace />
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const res = await login(email, password)
      setSession({
        accessToken: res.access_token,
        refreshToken: res.refresh_token,
        user: res.user,
      })
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Gagal masuk, coba lagi')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary to-accent p-4">
      <div className="w-full max-w-sm bg-surface rounded-card shadow-lg p-8 flex flex-col gap-6">
        {/* Logo + tagline */}
        <div className="flex flex-col items-center gap-2">
          <LogoBadge size={48} />
          <LogoWordmark className="text-h2" />
          <p className="text-caption text-fg-muted text-center">
            Platform AI untuk tim sales B2B
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
          <Field label="Email" htmlFor="email" required error={undefined}>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="nama@perusahaan.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={submitting}
              required
            />
          </Field>

          <Field label="Password" htmlFor="password" required error={error ?? undefined}>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={submitting}
                invalid={!!error}
                required
                className="pr-10"
              />
              <button
                type="button"
                aria-label={showPassword ? 'Sembunyikan password' : 'Tampilkan password'}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-fg-muted hover:text-fg transition-colors"
                onClick={() => setShowPassword((v) => !v)}
                tabIndex={-1}
              >
                {showPassword ? (
                  <EyeOff className="w-4 h-4" aria-hidden="true" />
                ) : (
                  <Eye className="w-4 h-4" aria-hidden="true" />
                )}
              </button>
            </div>
          </Field>

          <Button type="submit" loading={submitting} className="w-full mt-1">
            Masuk
          </Button>
        </form>

        {/* Lupa password placeholder */}
        <div className="flex flex-col items-center gap-3">
          <button
            type="button"
            className="text-caption text-primary hover:underline"
            onClick={() => {}}
          >
            Lupa password?
          </button>
          <p className="text-caption text-fg-muted text-center">
            Akun dikelola Admin internal
          </p>
        </div>
      </div>
    </div>
  )
}
