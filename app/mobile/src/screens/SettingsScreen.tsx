import { Button } from '@/components/ui/button'
import { useTheme, type ThemeMode } from '@/stores/theme'

interface Props {
  username: string
  serverURL: string
  expiresAt: string | null
  onBack: () => void
}

// Minimal mobile Settings screen. The big-picture rule is mobile is
// for monitoring, not configuring — all editing-heavy admin
// (channels, integrations, providers, backup targets) has its own
// dedicated screen elsewhere or is intentionally desktop-only.
//
// What lives here today:
//   - Theme toggle (light / dark / system) via the shared zustand
//     store; persists across launches via the same Preferences
//     storage as auth state
//   - Read-only "About" card: who am I, where am I connected,
//     when does my token expire
//
// Future polish: app-version display, push-notification toggle
// (after C ships), language switcher, etc.
export function SettingsScreen({
  username,
  serverURL,
  expiresAt,
  onBack,
}: Props) {
  const mode = useTheme((s) => s.mode)
  const setMode = useTheme((s) => s.setMode)

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-3 py-2 flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={onBack}>
          ← Back
        </Button>
        <h1 className="flex-1 text-base font-semibold">Settings</h1>
      </header>
      <main className="flex-1 px-4 py-3 space-y-4">
        <Section title="Appearance">
          <div className="grid grid-cols-3 gap-2">
            {(['light', 'dark', 'system'] as ThemeMode[]).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setMode(m)}
                className={`rounded-md border p-3 text-sm capitalize transition-colors ${
                  mode === m
                    ? 'border-accent bg-accent/10 text-accent'
                    : 'border-border bg-card hover:bg-accent/5'
                }`}
              >
                {m}
              </button>
            ))}
          </div>
        </Section>

        <Section title="About this device">
          <Field label="Signed in as" value={username} />
          <Field label="Server URL" value={serverURL} mono />
          {expiresAt && (
            <Field label="Token expires" value={new Date(expiresAt).toLocaleString()} />
          )}
        </Section>

        <Section title="Notes">
          <p className="text-xs text-muted-foreground leading-relaxed">
            Mobile is a monitoring surface. Configuration of
            channels, integrations, providers, and backups happens
            on the desktop admin where copy-paste of API keys and
            multi-tab forms is reasonable. Tap any of those entries
            in the &quot;More&quot; tab to view their current state.
          </p>
        </Section>
      </main>
    </div>
  )
}

function Section({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="space-y-2">
      <h2 className="text-xs uppercase tracking-wide text-muted-foreground">
        {title}
      </h2>
      <div className="rounded-md border border-border bg-card p-3 space-y-2">
        {children}
      </div>
    </section>
  )
}

function Field({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="space-y-0.5">
      <div className="text-[11px] text-muted-foreground">{label}</div>
      <div className={`text-sm break-all ${mono ? 'font-mono' : ''}`}>
        {value}
      </div>
    </div>
  )
}
