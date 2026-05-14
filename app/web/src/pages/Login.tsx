import { useState, type FormEvent } from 'react'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { Terminal as TerminalIcon, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, APIError } from '@/lib/api'
import { useAuth } from '@/stores/auth'

interface LoginResponse {
  token: string
  username: string
  issued_at: string
  expires_at: string
}

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const search = useSearch({ strict: false }) as { next?: string }
  const setSession = useAuth((s) => s.setSession)
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      const res = await api<LoginResponse>('/api/v1/auth/login', {
        method: 'POST',
        body: { username, password },
        skipAuthRedirect: true,
      })
      setSession(res.token, res.username, res.expires_at)
      navigate({ to: search.next || '/sessions' })
    } catch (err) {
      const message =
        err instanceof APIError
          ? err.message
          : err instanceof Error
            ? err.message
            : null
      setError(
        message ? t('auth.errorGeneric', { error: message }) : t('auth.errorFallback'),
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="h-svh flex items-center justify-center bg-background p-6">
      <div className="w-[360px] flex flex-col gap-6">
        <div className="flex items-center gap-2 mb-2">
          <TerminalIcon className="size-5 text-accent" strokeWidth={2.5} />
          <span className="text-[15px] font-semibold tracking-tight">
            {t('web.brand')}
          </span>
        </div>
        <div className="space-y-1">
          <h1 className="text-[20px] font-semibold tracking-tight">
            {t('auth.signInTitle')}
          </h1>
          <p className="text-[13px] text-muted-foreground">
            {t('auth.subtitle')}
          </p>
        </div>
        <form onSubmit={submit} className="flex flex-col gap-4">
          <div className="space-y-1.5">
            <Label htmlFor="username">{t('auth.username')}</Label>
            <Input
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              required
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="password">{t('auth.password')}</Label>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              required
            />
          </div>
          {error && (
            <div className="text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2">
              {error}
            </div>
          )}
          <Button
            type="submit"
            variant="accent"
            disabled={submitting}
            className="mt-2"
          >
            {submitting ? <Loader2 className="size-3.5 animate-spin" /> : null}
            {submitting ? t('auth.signingIn') : t('auth.signIn')}
          </Button>
        </form>
      </div>
    </div>
  )
}
