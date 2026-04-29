import { Link, useRouterState } from '@tanstack/react-router'
import {
  Layers,
  Cpu,
  MessageSquare,
  Plug,
  Activity,
  Settings,
  type LucideIcon,
} from 'lucide-react'

import { cn } from '@/lib/utils'

interface NavItem {
  to: string
  icon: LucideIcon
  label: string
  shortcut: string
}

const items: NavItem[] = [
  { to: '/sessions', icon: Layers, label: 'Sessions', shortcut: 'g s' },
  { to: '/providers', icon: Cpu, label: 'Providers', shortcut: 'g p' },
  { to: '/channels', icon: MessageSquare, label: 'Channels', shortcut: 'g c' },
  { to: '/integrations', icon: Plug, label: 'Integrations', shortcut: 'g i' },
  { to: '/activity', icon: Activity, label: 'Activity', shortcut: 'g a' },
  { to: '/settings', icon: Settings, label: 'Settings', shortcut: 'g ,' },
]

export function SidebarNav() {
  const { location } = useRouterState()
  return (
    <nav className="w-56 shrink-0 border-r border-border bg-card/40 flex flex-col py-3 px-2 gap-0.5">
      <div className="px-2 pb-2 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
        Workspace
      </div>
      {items.map(({ to, icon: Icon, label, shortcut }) => {
        const active =
          to === '/sessions'
            ? location.pathname.startsWith('/sessions') ||
              location.pathname === '/'
            : location.pathname.startsWith(to)
        return (
          <Link
            key={to}
            to={to}
            className={cn(
              'flex items-center gap-2.5 h-7 px-2.5 rounded-md text-[13px] transition-all duration-100',
              'text-muted-foreground hover:text-foreground hover:bg-card',
              active && 'bg-card text-foreground',
            )}
          >
            <Icon className="size-3.5 shrink-0" />
            <span className="flex-1">{label}</span>
            <kbd className="opacity-0 group-hover:opacity-100">
              {shortcut}
            </kbd>
          </Link>
        )
      })}
    </nav>
  )
}
