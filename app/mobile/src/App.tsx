import { Button } from '@/components/ui/button'
import { Capacitor } from '@capacitor/core'

// B1 placeholder screen. Confirms three things:
//   1. Capacitor's Vite + WebView pipeline is wired correctly
//   2. shared-ui primitives import + render under the mobile entry
//   3. Tailwind v4 + design tokens (copied from web for now; A5 will
//      consolidate into shared-ui/styles) are applied
//
// Real screens land in B5 (Sessions list) onwards.
export function App() {
  const platform = Capacitor.getPlatform()
  const isNative = Capacitor.isNativePlatform()

  return (
    <div className="min-h-screen bg-background text-foreground flex items-center justify-center p-6">
      <div className="max-w-md w-full space-y-4 text-center">
        <h1 className="text-2xl font-semibold">OpenDray</h1>
        <p className="text-sm text-muted-foreground">
          Mobile shell — phase B1
        </p>
        <div className="rounded-md border border-border bg-card text-card-foreground p-4 text-left text-sm space-y-1">
          <div>
            Platform: <span className="text-accent">{platform}</span>
          </div>
          <div>
            Native: <span className="text-accent">{String(isNative)}</span>
          </div>
        </div>
        <Button
          variant="default"
          onClick={() => alert(`Hello from ${platform}`)}
        >
          Test shared-ui Button
        </Button>
      </div>
    </div>
  )
}
