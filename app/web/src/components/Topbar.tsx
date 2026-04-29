import { Sun, Moon, Monitor, Terminal as TerminalIcon, LogOut } from 'lucide-react'
import { useNavigate } from '@tanstack/react-router'

import { Button } from './ui/button'
import { useTheme } from '@/stores/theme'
import { useAuth } from '@/stores/auth'

export function Topbar() {
  const mode = useTheme((s) => s.mode)
  const setMode = useTheme((s) => s.setMode)
  const username = useAuth((s) => s.username)
  const clear = useAuth((s) => s.clear)
  const navigate = useNavigate()

  const cycle = () => {
    const next: 'light' | 'dark' | 'system' =
      mode === 'dark' ? 'light' : mode === 'light' ? 'system' : 'dark'
    setMode(next)
  }

  const ThemeIcon = mode === 'dark' ? Moon : mode === 'light' ? Sun : Monitor

  return (
    <div className="h-11 border-b border-border bg-background flex items-center px-3 gap-2 shrink-0">
      <div className="flex items-center gap-1.5 px-1">
        <TerminalIcon className="size-3.5 text-accent" strokeWidth={2.5} />
        <span className="text-[12px] font-semibold tracking-tight">
          opendray
        </span>
      </div>
      <div className="flex-1" />
      <Button
        variant="ghost"
        size="icon"
        onClick={cycle}
        title={`Theme: ${mode}`}
        aria-label={`Theme: ${mode}`}
      >
        <ThemeIcon className="size-3.5" />
      </Button>
      {username && (
        <>
          <span className="text-[11px] text-muted-foreground px-1.5">
            {username}
          </span>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => {
              clear()
              navigate({ to: '/login', search: { next: undefined } })
            }}
            title="Logout"
            aria-label="Logout"
          >
            <LogOut className="size-3.5" />
          </Button>
        </>
      )}
    </div>
  )
}
