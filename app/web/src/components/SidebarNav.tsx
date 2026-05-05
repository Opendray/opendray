import { Fragment } from 'react'
import { Link, useRouterState } from '@tanstack/react-router'
import {
  Layers,
  Cpu,
  MessageSquare,
  Plug,
  Activity,
  Settings,
  Boxes,
  NotebookPen,
  BookOpen,
  Brain,
  Archive,
  type LucideIcon,
} from 'lucide-react'

import { cn } from '@/lib/utils'
import { useLayout } from '@/stores/layout'
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip'

interface NavItem {
  to: string
  icon: LucideIcon
  label: string
  shortcut: string
}

// Three semantic groups separated by a thin divider:
//   1. Outputs   — what the operator produces / observes
//   2. Plumbing  — upstream agents, downstream channels, sideways integrations
//   3. Platform  — extensions, config, help
const groups: NavItem[][] = [
  [
    { to: '/sessions', icon: Layers, label: 'Sessions', shortcut: 'g s' },
    { to: '/notes', icon: NotebookPen, label: 'Notes', shortcut: 'g n' },
    { to: '/memory', icon: Brain, label: 'Memory', shortcut: 'g m' },
    { to: '/activity', icon: Activity, label: 'Activity', shortcut: 'g a' },
  ],
  [
    { to: '/providers', icon: Cpu, label: 'Providers', shortcut: 'g p' },
    { to: '/channels', icon: MessageSquare, label: 'Channels', shortcut: 'g c' },
    { to: '/integrations', icon: Plug, label: 'Integrations', shortcut: 'g i' },
  ],
  [
    { to: '/plugins', icon: Boxes, label: 'Plugins', shortcut: 'g l' },
    { to: '/backups', icon: Archive, label: 'Backups', shortcut: 'g b' },
    { to: '/settings', icon: Settings, label: 'Settings', shortcut: 'g ,' },
    { to: '/tutorial', icon: BookOpen, label: 'Tutorial', shortcut: 'g h' },
  ],
]

export function SidebarNav() {
  const { location } = useRouterState()
  const collapsed = useLayout((s) => s.sidebarCollapsed)
  return (
    <nav
      className={cn(
        'shrink-0 border-r border-border bg-card/40 flex flex-col py-3 gap-0.5 transition-[width] duration-150',
        collapsed ? 'w-12 px-1.5' : 'w-56 px-2',
      )}
    >
      {!collapsed && (
        <div className="px-2 pb-2 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
          Workspace
        </div>
      )}
      {groups.map((group, gi) => (
        <Fragment key={gi}>
          {gi > 0 && (
            <div
              className={cn(
                'my-1.5 border-t border-border/60',
                collapsed ? 'mx-1.5' : 'mx-2',
              )}
              aria-hidden
            />
          )}
          {group.map(({ to, icon: Icon, label, shortcut }) => {
            const active =
              to === '/sessions'
                ? location.pathname.startsWith('/sessions') ||
                  location.pathname === '/'
                : location.pathname.startsWith(to)
            const link = (
              <Link
                key={to}
                to={to}
                aria-label={label}
                className={cn(
                  'flex items-center h-7 rounded-md text-[13px] transition-all duration-100',
                  'text-muted-foreground hover:text-foreground hover:bg-card',
                  active && 'bg-card text-foreground',
                  collapsed ? 'justify-center px-0' : 'gap-2.5 px-2.5',
                )}
              >
                <Icon className="size-3.5 shrink-0" />
                {!collapsed && (
                  <>
                    <span className="flex-1">{label}</span>
                    <kbd className="opacity-0 group-hover:opacity-100">
                      {shortcut}
                    </kbd>
                  </>
                )}
              </Link>
            )
            if (!collapsed) return link
            return (
              <Tooltip key={to}>
                <TooltipTrigger asChild>{link}</TooltipTrigger>
                <TooltipContent side="right">
                  {label}
                  <span className="ml-2 text-muted-foreground">{shortcut}</span>
                </TooltipContent>
              </Tooltip>
            )
          })}
        </Fragment>
      ))}
    </nav>
  )
}
