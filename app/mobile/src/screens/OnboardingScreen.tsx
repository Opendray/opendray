import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getHealth } from '../lib/api'
import { setServerURL } from '../lib/storage'

interface Props {
  onConnected: (url: string) => void
}

// First-launch screen. The user types their gateway URL (e.g.
// `https://opendray.example.com`); we hit `/api/v1/health` to verify
// it's reachable and actually opendray, then persist it for future
// launches.
export function OnboardingScreen({ onConnected }: Props) {
  const [url, setUrl] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    const trimmed = url.trim().replace(/\/+$/, '')
    if (!trimmed.startsWith('http://') && !trimmed.startsWith('https://')) {
      setError('URL must start with http:// or https://')
      return
    }
    setBusy(true)
    try {
      const health = await getHealth(trimmed)
      if (!health.status) {
        throw new Error(`Endpoint reachable but didn't return an opendray health payload`)
      }
      await setServerURL(trimmed)
      onConnected(trimmed)
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Could not reach the server. Check the URL and your network.',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex items-center justify-center p-6">
      <form onSubmit={onSubmit} className="max-w-md w-full space-y-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Connect to OpenDray</h1>
          <p className="text-sm text-muted-foreground">
            Enter the URL of your opendray gateway. We&rsquo;ll verify it before continuing.
          </p>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="url">Server URL</Label>
          <Input
            id="url"
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://opendray.example.com"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            inputMode="url"
            required
            disabled={busy}
          />
        </div>
        {error && (
          <div className="text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2">
            {error}
          </div>
        )}
        <Button type="submit" disabled={busy || !url} className="w-full">
          {busy ? 'Checking…' : 'Continue'}
        </Button>
      </form>
    </div>
  )
}
