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
  Brain,
  Archive,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import { useLayout } from '@/stores/layout'
import { useIsMobile } from '../lib/useIsMobile'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

interface NavItem {
  to: string
  icon: LucideIcon
  labelKey: string
  shortcut: string
}

// Three semantic groups separated by a thin divider:
//   1. Outputs   — what the operator produces / observes
//   2. Plumbing  — upstream agents, downstream channels, sideways integrations
//   3. Platform  — extensions, config, help
const groups: NavItem[][] = [
  [
    { to: '/sessions', icon: Layers, labelKey: 'nav.sessions', shortcut: 'g s' },
    { to: '/notes', icon: NotebookPen, labelKey: 'nav.notes', shortcut: 'g n' },
    { to: '/memory', icon: Brain, labelKey: 'nav.memory', shortcut: 'g m' },
    { to: '/activity', icon: Activity, labelKey: 'nav.activity', shortcut: 'g a' },
  ],
  [
    { to: '/providers', icon: Cpu, labelKey: 'nav.providers', shortcut: 'g p' },
    { to: '/channels', icon: MessageSquare, labelKey: 'nav.channels', shortcut: 'g c' },
    { to: '/integrations', icon: Plug, labelKey: 'nav.integrations', shortcut: 'g i' },
  ],
  [
    { to: '/plugins', icon: Boxes, labelKey: 'nav.plugins', shortcut: 'g l' },
    { to: '/backups', icon: Archive, labelKey: 'nav.backups', shortcut: 'g b' },
    { to: '/settings', icon: Settings, labelKey: 'nav.settings', shortcut: 'g ,' },
  ],
]

export function SidebarNav() {
  const { t } = useTranslation()
  const { location } = useRouterState()
  const collapsedState = useLayout((s) => s.sidebarCollapsed)
  const isMobile = useIsMobile()
  // On mobile the nav is a full-width slide-over (positioned by AppShell),
  // so never collapse it to the icon rail there.
  const collapsed = collapsedState && !isMobile
  return (
    <nav
      className={cn(
        'shrink-0 border-r border-border bg-card/40 flex flex-col py-3 gap-0.5 transition-[width] duration-150',
        collapsed ? 'w-12 px-1.5' : 'w-56 px-2',
      )}
    >
      {!collapsed && (
        <div className="px-2 pb-2 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
          {t('nav.workspace')}
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
          {group.map(({ to, icon: Icon, labelKey, shortcut }) => {
            const label = t(labelKey)
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
