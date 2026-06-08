import { useEffect, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  Layers,
  Cpu,
  MessageSquare,
  Plug,
  Activity,
  Settings,
  Sun,
  Moon,
  Monitor,
  LogOut,
  Brain,
} from 'lucide-react'

import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandShortcut,
} from '@/components/ui/command'
import { useTheme, type ThemeMode } from '@/stores/theme'
import { useAuth } from '@/stores/auth'
import { useSessionTabs } from '@/stores/sessionTabs'
import { listSessions } from '@/lib/sessions'
import { isTerminalSessionState } from '@/lib/types'
import { cwdTail } from '@/lib/providers'

interface CommandPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const navigate = useNavigate()
  const setMode = useTheme((s) => s.setMode)
  const clear = useAuth((s) => s.clear)
  const openTab = useSessionTabs((s) => s.open)
  const recents = useSessionTabs((s) => s.recents)
  const [query, setQuery] = useState('')

  // Only fetch the session list while the palette is open. Shares the
  // ['sessions'] cache with the session list when both are mounted.
  const { data: sessions } = useQuery({
    queryKey: ['sessions'],
    queryFn: listSessions,
    enabled: open,
  })

  // Most-recently-opened first, so a quick Cmd+K → Enter reopens the
  // session you were just in.
  const sessionItems = (sessions ?? []).slice().sort((a, b) => {
    const r = (recents[b.id] ?? 0) - (recents[a.id] ?? 0)
    if (r !== 0) return r
    return new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
  })

  const openSession = (id: string, name?: string) => {
    openTab({ id, name })
    onOpenChange(false)
    navigate({ to: '/sessions' })
  }

  const go = (path: string) => () => {
    onOpenChange(false)
    navigate({ to: path })
  }

  // Reset the query each time the palette closes so it reopens clean.
  useEffect(() => {
    if (!open) setQuery('')
  }, [open])

  const setTheme = (m: ThemeMode) => () => {
    setMode(m)
    onOpenChange(false)
  }

  const logout = () => {
    clear()
    onOpenChange(false)
    navigate({ to: '/login', search: { next: undefined } })
  }

  return (
    <CommandDialog open={open} onOpenChange={onOpenChange}>
      <CommandInput
        placeholder="Search sessions or jump to…"
        autoFocus
        value={query}
        onValueChange={setQuery}
      />
      <CommandList>
        <CommandEmpty>No results.</CommandEmpty>

        {query.trim() !== '' && sessionItems.length > 0 && (
          <CommandGroup heading="Sessions">
            {sessionItems.map((s) => (
              <CommandItem
                key={s.id}
                // value drives cmdk's fuzzy match — name + cwd so either
                // matches; id appended to keep values unique.
                value={`${s.name ?? ''} ${s.cwd} ${s.id}`}
                onSelect={() => openSession(s.id, s.name)}
              >
                <Layers />
                <span className="truncate">{s.name || cwdTail(s.cwd)}</span>
                <CommandShortcut className="opacity-60">
                  {isTerminalSessionState(s.state) ? 'ended' : s.state}
                </CommandShortcut>
              </CommandItem>
            ))}
          </CommandGroup>
        )}

        <CommandGroup heading="Navigate">
          <CommandItem onSelect={go('/sessions')}>
            <Layers /> Sessions
            <CommandShortcut>g s</CommandShortcut>
          </CommandItem>
          <CommandItem onSelect={go('/providers')}>
            <Cpu /> Providers
            <CommandShortcut>g p</CommandShortcut>
          </CommandItem>
          <CommandItem onSelect={go('/channels')}>
            <MessageSquare /> Channels
            <CommandShortcut>g c</CommandShortcut>
          </CommandItem>
          <CommandItem onSelect={go('/integrations')}>
            <Plug /> Integrations
            <CommandShortcut>g i</CommandShortcut>
          </CommandItem>
          <CommandItem onSelect={go('/memory')}>
            <Brain /> Memory
            <CommandShortcut>g m</CommandShortcut>
          </CommandItem>
          <CommandItem onSelect={() => navigate({ to: '/notes', search: { mode: 'project', cwd: '' } })}>
            <Brain /> Project notes
          </CommandItem>
          <CommandItem onSelect={go('/memory/archived')}>
            <Brain /> Archived memories
          </CommandItem>
          <CommandItem onSelect={go('/activity')}>
            <Activity /> Activity
            <CommandShortcut>g a</CommandShortcut>
          </CommandItem>
          <CommandItem onSelect={go('/settings')}>
            <Settings /> Settings
            <CommandShortcut>g ,</CommandShortcut>
          </CommandItem>
        </CommandGroup>

        <CommandGroup heading="Theme">
          <CommandItem onSelect={setTheme('light')}>
            <Sun /> Light
          </CommandItem>
          <CommandItem onSelect={setTheme('dark')}>
            <Moon /> Dark
          </CommandItem>
          <CommandItem onSelect={setTheme('system')}>
            <Monitor /> System
          </CommandItem>
        </CommandGroup>

        <CommandGroup heading="Account">
          <CommandItem onSelect={logout}>
            <LogOut /> Sign out
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  )
}

/**
 * Hook that wires ⌘K / Ctrl K to a CommandPalette open-state setter.
 * Call from AppShell (so palette only mounts inside protected area).
 */
export function useCommandPaletteHotkey(setOpen: (next: (v: boolean) => boolean) => void) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
        e.preventDefault()
        setOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [setOpen])
}
