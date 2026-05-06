import { Button } from '@/components/ui/button'

interface Props {
  serverURL: string
  username: string
  expiresAt: string | null
  onLogout: () => void
}

// B3 placeholder home screen. Replaced by the real Sessions list in B5.
// Confirms end-to-end auth flow works: token persisted in Preferences,
// re-launch picks it up, this screen renders without re-prompting.
export function HomeScreen({ serverURL, username, expiresAt, onLogout }: Props) {
  return (
    <div className="min-h-screen bg-background text-foreground flex items-center justify-center p-6">
      <div className="max-w-md w-full space-y-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">OpenDray</h1>
          <p className="text-sm text-muted-foreground">Signed in as {username}</p>
        </div>
        <div className="rounded-md border border-border bg-card text-card-foreground p-4 text-sm space-y-2">
          <div className="space-y-0.5">
            <div className="text-muted-foreground text-xs">Server</div>
            <div className="break-all">{serverURL}</div>
          </div>
          {expiresAt && (
            <div className="space-y-0.5">
              <div className="text-muted-foreground text-xs">Token expires</div>
              <div>{new Date(expiresAt).toLocaleString()}</div>
            </div>
          )}
        </div>
        <p className="text-xs text-muted-foreground">
          Sessions list, terminal view and the rest of the admin
          surface land in subsequent phases (B5+). This placeholder
          confirms end-to-end auth works.
        </p>
        <Button variant="default" onClick={onLogout} className="w-full">
          Sign out
        </Button>
      </div>
    </div>
  )
}
