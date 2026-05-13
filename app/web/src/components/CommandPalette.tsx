import { useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
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

interface CommandPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const navigate = useNavigate()
  const setMode = useTheme((s) => s.setMode)
  const clear = useAuth((s) => s.clear)

  const go = (path: string) => () => {
    onOpenChange(false)
    navigate({ to: path })
  }

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
      <CommandInput placeholder="Search…" autoFocus />
      <CommandList>
        <CommandEmpty>No results.</CommandEmpty>

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
          <CommandItem onSelect={() => navigate({ to: '/memory/project', search: { cwd: '' } })}>
            <Brain /> Project memory
          </CommandItem>
          <CommandItem onSelect={go('/memory/cleanup')}>
            <Brain /> Cleanup inbox
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
