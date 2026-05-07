import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  type FullNote,
  type NoteSummary,
  getNote,
  listNotes,
  MobileAPIError,
} from '../lib/api'

interface Props {
  serverURL: string
  token: string
  onAuthExpired: () => void
}

type View =
  | { kind: 'list'; loading: boolean; notes: NoteSummary[]; error?: string }
  | { kind: 'detail'; loading: boolean; note?: FullNote; path: string; error?: string }

// Notes vault viewer (read-only on mobile). The desktop admin handles
// markdown editing — small-screen rich-text edit is a separate
// project. Mobile shows the file tree as a flat list and the body as
// monospace text; markdown rendering is a future polish task.
export function NotesScreen({ serverURL, token, onAuthExpired }: Props) {
  const [view, setView] = useState<View>({ kind: 'list', loading: true, notes: [] })

  const loadList = useCallback(async () => {
    setView({ kind: 'list', loading: true, notes: [] })
    try {
      const notes = await listNotes(serverURL, token)
      // Most-recent-modified-first reads better on mobile than the
      // raw filesystem order.
      notes.sort((a, b) => Date.parse(b.modified) - Date.parse(a.modified))
      setView({ kind: 'list', loading: false, notes })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setView({
        kind: 'list',
        loading: false,
        notes: [],
        error: err instanceof Error ? err.message : 'Failed to load notes',
      })
    }
  }, [serverURL, token, onAuthExpired])

  useEffect(() => {
    void loadList()
  }, [loadList])

  async function openNote(path: string) {
    setView({ kind: 'detail', loading: true, path })
    try {
      const note = await getNote(serverURL, token, path)
      setView({ kind: 'detail', loading: false, path, note })
    } catch (err) {
      if (err instanceof MobileAPIError && err.status === 401) {
        onAuthExpired()
        return
      }
      setView({
        kind: 'detail',
        loading: false,
        path,
        error: err instanceof Error ? err.message : 'Failed to load note',
      })
    }
  }

  if (view.kind === 'detail') {
    return (
      <div className="min-h-screen bg-background text-foreground flex flex-col">
        <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-3 py-2 flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={loadList}>
            ← Back
          </Button>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium truncate">
              {view.note?.title ?? view.path.split('/').pop()}
            </div>
            <div className="text-[10px] text-muted-foreground truncate">{view.path}</div>
          </div>
        </header>
        <main className="flex-1 px-4 py-3 overflow-auto">
          {view.loading && (
            <div className="text-sm text-muted-foreground">Loading…</div>
          )}
          {view.error && (
            <div className="rounded-md border border-destructive/30 bg-destructive/10 text-destructive text-sm px-3 py-2">
              {view.error}
            </div>
          )}
          {view.note && (
            <pre className="whitespace-pre-wrap break-words font-mono text-[12px] leading-relaxed text-foreground">
              {view.note.body}
            </pre>
          )}
        </main>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-3">
        <h1 className="text-lg font-semibold leading-tight">Notes</h1>
        <p className="text-xs text-muted-foreground">
          Read-only on mobile. Edit on desktop.
        </p>
      </header>
      <main className="flex-1 px-4 py-3">
        {view.loading && (
          <div className="text-sm text-muted-foreground">Loading…</div>
        )}
        {view.error && (
          <div className="space-y-3">
            <div className="rounded-md border border-destructive/30 bg-destructive/10 text-destructive text-sm px-3 py-2">
              {view.error}
            </div>
            <Button variant="outline" size="sm" onClick={loadList}>
              Retry
            </Button>
          </div>
        )}
        {!view.loading && !view.error && view.notes.length === 0 && (
          <div className="rounded-md border border-border bg-card p-6 text-center text-sm text-muted-foreground">
            Vault is empty. Create notes from the desktop admin.
          </div>
        )}
        {!view.loading && view.notes.length > 0 && (
          <ul className="space-y-2">
            {view.notes.map((n) => (
              <li key={n.path}>
                <button
                  type="button"
                  onClick={() => openNote(n.path)}
                  className="w-full text-left rounded-md border border-border bg-card p-3 active:bg-accent/10 transition-colors"
                >
                  <div className="text-sm font-medium truncate">{n.title || n.path}</div>
                  <div className="text-[11px] text-muted-foreground truncate">
                    {n.path}
                  </div>
                  <div className="text-[10px] text-muted-foreground">
                    {humanSize(n.size)} · {formatRelative(n.modified)}
                  </div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  )
}

function formatRelative(iso: string): string {
  const ts = Date.parse(iso)
  if (Number.isNaN(ts)) return iso
  const seconds = Math.round((Date.now() - ts) / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.round(hours / 24)
  return `${days}d ago`
}

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
