import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { postMobileLogin } from '../lib/api'
import { setAuth } from '../lib/storage'

interface Props {
  serverURL: string
  onAuthed: () => void
  onChangeServer: () => void
}

// Second-launch screen (and every subsequent re-auth). Calls the
// mobile-login endpoint added in B2 (#22) so the issued token has
// the longer mobile TTL — see ADR 0015 §5.
export function LoginScreen({ serverURL, onAuthed, onChangeServer }: Props) {
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      const res = await postMobileLogin(serverURL, username, password)
      await setAuth(res.token, res.expires_at, res.username)
      onAuthed()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex items-center justify-center p-6">
      <form onSubmit={onSubmit} className="max-w-md w-full space-y-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Sign in</h1>
          <p className="text-sm text-muted-foreground break-all">{serverURL}</p>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="username">Username</Label>
          <Input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            autoComplete="username"
            required
            disabled={busy}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
            disabled={busy}
          />
        </div>
        {error && (
          <div className="text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2">
            {error}
          </div>
        )}
        <Button type="submit" disabled={busy || !password} className="w-full">
          {busy ? 'Signing in…' : 'Sign in'}
        </Button>
        <button
          type="button"
          onClick={onChangeServer}
          className="block w-full text-center text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          Use a different server
        </button>
      </form>
    </div>
  )
}
