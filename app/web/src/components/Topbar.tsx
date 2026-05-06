import { useNavigate } from '@tanstack/react-router'
import {
  Sun,
  Moon,
  Monitor,
  Terminal as TerminalIcon,
  LogOut,
  Search,
  Check,
  PanelLeftClose,
  PanelLeftOpen,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuShortcut,
} from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useTheme, type ThemeMode } from '@/stores/theme'
import { useAuth } from '@/stores/auth'
import { useLayout } from '@/stores/layout'

interface TopbarProps {
  onOpenPalette?: () => void
}

const themeOptions: { mode: ThemeMode; label: string; icon: typeof Sun }[] = [
  { mode: 'light', label: 'Light', icon: Sun },
  { mode: 'dark', label: 'Dark', icon: Moon },
  { mode: 'system', label: 'System', icon: Monitor },
]

export function Topbar({ onOpenPalette }: TopbarProps) {
  const mode = useTheme((s) => s.mode)
  const setMode = useTheme((s) => s.setMode)
  const username = useAuth((s) => s.username)
  const expiresAt = useAuth((s) => s.expiresAt)
  const clear = useAuth((s) => s.clear)
  const navigate = useNavigate()
  const sidebarCollapsed = useLayout((s) => s.sidebarCollapsed)
  const toggleSidebar = useLayout((s) => s.toggleSidebar)

  const ThemeIcon =
    mode === 'dark' ? Moon : mode === 'light' ? Sun : Monitor

  return (
    <div className="h-11 border-b border-border bg-background flex items-center px-3 gap-1.5 shrink-0">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleSidebar}
            aria-label={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            className="size-7"
          >
            {sidebarCollapsed ? (
              <PanelLeftOpen className="size-3.5" />
            ) : (
              <PanelLeftClose className="size-3.5" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          {sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        </TooltipContent>
      </Tooltip>
      <div className="flex items-center gap-1.5 pl-1">
        <TerminalIcon
          className="size-3.5 text-accent"
          strokeWidth={2.5}
        />
        <span className="text-[12px] font-semibold tracking-tight">
          opendray
        </span>
      </div>
      <div className="flex-1" />

      {onOpenPalette && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              onClick={onOpenPalette}
              className="h-7 gap-2 text-muted-foreground bg-card/50 hover:bg-card font-normal text-[12px]"
            >
              <Search className="size-3" />
              <span>Search</span>
              <kbd className="ml-1">⌘K</kbd>
            </Button>
          </TooltipTrigger>
          <TooltipContent>Open command palette</TooltipContent>
        </Tooltip>
      )}

      <DropdownMenu>
        <Tooltip>
          <TooltipTrigger asChild>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label={`Theme: ${mode}`}
              >
                <ThemeIcon className="size-3.5" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>Theme</TooltipContent>
        </Tooltip>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>Appearance</DropdownMenuLabel>
          {themeOptions.map(({ mode: m, label, icon: Icon }) => (
            <DropdownMenuItem key={m} onSelect={() => setMode(m)}>
              <Icon />
              <span>{label}</span>
              {mode === m && <Check className="ml-auto size-3" />}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {username && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-[12px] font-normal text-muted-foreground gap-1.5"
            >
              <span className="size-1.5 rounded-full bg-state-running" />
              {username}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="min-w-[220px]">
            <DropdownMenuLabel>Signed in as</DropdownMenuLabel>
            <div className="px-2 pb-1.5 text-[12px]">{username}</div>
            {expiresAt && (
              <>
                <DropdownMenuLabel>Token expires</DropdownMenuLabel>
                <div className="px-2 pb-1.5 text-[11px] text-muted-foreground font-mono">
                  {new Date(expiresAt).toLocaleString()}
                </div>
              </>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={() => {
                clear()
                navigate({ to: '/login', search: { next: undefined } })
              }}
            >
              <LogOut /> Sign out
              <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  )
}
