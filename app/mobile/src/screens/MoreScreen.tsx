// "More" tab — overflow list of secondary surfaces. Lower-frequency
// admin pages (Channels / Integrations / Providers / Settings /
// Backups) live here so the bottom tab bar stays at 5 entries
// instead of 9+.
//
// Each row navigates to a full-screen sub-page via the App.tsx
// state machine, mirroring how SessionsScreen → SessionDetailScreen
// works (full-screen, no bottom tab visible during sub-page).
//
// Icons use lucide-react (SVG) — same reason as BottomTabBar:
// iOS 26.3 doesn't render the symbol-block Unicode glyphs we'd
// otherwise want.

import {
  ChevronRight,
  HardDrive,
  type LucideIcon,
  MessageSquare,
  Plug,
  Settings as SettingsIcon,
  Terminal,
} from 'lucide-react'

interface Props {
  username: string
  serverURL: string
  expiresAt: string | null
  onOpen: (page: SubPage) => void
  onLogout: () => void
}

export type SubPage =
  | 'channels'
  | 'integrations'
  | 'providers'
  | 'backups'
  | 'settings'

const ITEMS: {
  id: SubPage
  label: string
  description: string
  icon: LucideIcon
}[] = [
  {
    id: 'channels',
    label: 'Channels',
    description: 'Telegram / Discord / Slack adapters',
    icon: MessageSquare,
  },
  {
    id: 'integrations',
    label: 'Integrations',
    description: 'External apps registered against the gateway',
    icon: Plug,
  },
  {
    id: 'providers',
    label: 'CLI Providers',
    description: 'Claude Code / Codex / Gemini configurations',
    icon: Terminal,
  },
  {
    id: 'backups',
    label: 'Backups',
    description: 'Recent encrypted snapshot history',
    icon: HardDrive,
  },
  {
    id: 'settings',
    label: 'Settings',
    description: 'Theme + device info',
    icon: SettingsIcon,
  },
]

export function MoreScreen({
  username,
  serverURL,
  expiresAt,
  onOpen,
  onLogout,
}: Props) {
  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-3">
        <h1 className="text-lg font-semibold leading-tight">More</h1>
      </header>
      <main className="flex-1 px-4 py-3 space-y-4">
        <ul className="space-y-2">
          {ITEMS.map((it) => {
            const Icon = it.icon
            return (
              <li key={it.id}>
                <button
                  type="button"
                  onClick={() => onOpen(it.id)}
                  className="w-full text-left rounded-md border border-border bg-card text-card-foreground p-3 active:bg-accent/10 transition-colors flex items-center gap-3"
                >
                  <Icon className="w-5 h-5 text-accent shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium">{it.label}</div>
                    <div className="text-xs text-muted-foreground truncate">
                      {it.description}
                    </div>
                  </div>
                  <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0" />
                </button>
              </li>
            )
          })}
        </ul>

        <div className="rounded-md border border-border bg-card p-3 text-xs space-y-1">
          <div className="text-muted-foreground">Signed in as</div>
          <div className="font-medium">{username}</div>
          <div className="text-muted-foreground pt-2">Server</div>
          <div className="break-all">{serverURL}</div>
          {expiresAt && (
            <>
              <div className="text-muted-foreground pt-2">Token expires</div>
              <div>{new Date(expiresAt).toLocaleString()}</div>
            </>
          )}
        </div>

        <button
          type="button"
          onClick={onLogout}
          className="w-full rounded-md border border-destructive/30 bg-destructive/10 text-destructive text-sm px-3 py-2.5 active:bg-destructive/20 transition-colors"
        >
          Sign out
        </button>
      </main>
    </div>
  )
}
