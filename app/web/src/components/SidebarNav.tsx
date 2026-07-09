import { Fragment, useCallback, useState } from 'react'
import { Link, useRouterState } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  Layers,
  Cpu,
  MessageSquare,
  Plug,
  Activity,
  Settings,
  Boxes,
  BookText,
  Brain,
  Archive,
  Sparkles,
  BookOpen,
  Send,
  Heart,
  ExternalLink,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import { useLayout } from '@/stores/layout'
import { useIsMobile } from '../lib/useIsMobile'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { getVersionInfo, type VersionInfo } from '@/lib/version'
import {
  isReleaseUnread,
  normalizeReleaseVersion,
} from '@/lib/releases'
import { UpdatesDrawer, useLatestReleaseQuery } from './UpdatesDrawer'

// 6 hours between background version checks — releases are weekly at
// most, so anything more frequent burns server cycles for a state
// that rarely changes. The badge also picks up any refetch the
// Settings/About panel triggers, since both observers share the
// `['version']` query cache.
const UPDATE_POLL_INTERVAL_MS = 6 * 60 * 60 * 1000

/** External community / docs links shown under Settings. */
const DOCS_URL = 'https://opendray.dev/docs/'
const COMMUNITY_URL = 'https://t.me/opendraycommunity'
const SPONSOR_URL = 'https://opendray.dev/sponsors/'

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
    // Cortex — the unified Memory → Notes → Knowledge module. One
    // entry; the three rungs are layered inside (they are one loop,
    // not three silos).
    { to: '/cortex', icon: Brain, labelKey: 'nav.cortex', shortcut: 'g x' },
    { to: '/activity', icon: Activity, labelKey: 'nav.activity', shortcut: 'g a' },
  ],
  [
    { to: '/providers', icon: Cpu, labelKey: 'nav.providers', shortcut: 'g p' },
    { to: '/channels', icon: MessageSquare, labelKey: 'nav.channels', shortcut: 'g c' },
    { to: '/integrations', icon: Plug, labelKey: 'nav.integrations', shortcut: 'g i' },
  ],
  [
    { to: '/vault', icon: BookText, labelKey: 'nav.vault', shortcut: 'g v' },
    { to: '/plugins', icon: Boxes, labelKey: 'nav.plugins', shortcut: 'g l' },
    { to: '/backups', icon: Archive, labelKey: 'nav.backups', shortcut: 'g b' },
    { to: '/settings', icon: Settings, labelKey: 'nav.settings', shortcut: 'g ,' },
  ],
]

interface ExternalNavItem {
  id: string
  href: string
  icon: LucideIcon
  labelKey: string
}

const externalLinks: ExternalNavItem[] = [
  {
    id: 'docs',
    href: DOCS_URL,
    icon: BookOpen,
    labelKey: 'nav.docs',
  },
  {
    id: 'community',
    href: COMMUNITY_URL,
    icon: Send,
    labelKey: 'nav.community',
  },
  {
    id: 'sponsor',
    href: SPONSOR_URL,
    icon: Heart,
    labelKey: 'nav.sponsor',
  },
]

export function SidebarNav() {
  const { t } = useTranslation()
  const { location } = useRouterState()
  const collapsedState = useLayout((s) => s.sidebarCollapsed)
  const isMobile = useIsMobile()
  // On mobile the nav is a full-width slide-over (positioned by AppShell),
  // so never collapse it to the icon rail there.
  const collapsed = collapsedState && !isMobile

  const [updatesOpen, setUpdatesOpen] = useState(false)
  // Bumps when the operator marks a release read so the badge re-renders
  // without waiting for a query refetch.
  const [readEpoch, setReadEpoch] = useState(0)

  // Background poll for available binary updates (About / self-update).
  const { data: version } = useQuery<VersionInfo>({
    queryKey: ['version'],
    queryFn: getVersionInfo,
    refetchInterval: UPDATE_POLL_INTERVAL_MS,
    staleTime: UPDATE_POLL_INTERVAL_MS,
  })
  // Separate query: release notes body for the Updates drawer + unread
  // badge. Shares cache with UpdatesDrawer via useLatestReleaseQuery.
  const { data: release } = useLatestReleaseQuery()

  const updateAvailable = !!version?.updateAvailable && !version?.pending

  // Unread = operator hasn't marked this release as read. Prefer the
  // GitHub release version; fall back to the gateway's `latest` so a
  // failed notes fetch still badges when a binary update exists.
  const latestForUnread =
    release?.version || normalizeReleaseVersion(version?.latest)
  // readEpoch is intentionally read so mark-read re-evaluates isReleaseUnread.
  void readEpoch
  const notesUnread = isReleaseUnread(latestForUnread)
  const showUpdatesBadge = notesUnread || updateAvailable
  const highlightCount = release?.highlights.length ?? 0
  // "Updates · 3" when we know highlight count and notes are unread;
  // otherwise a plain dot is enough.
  const showCountChip = notesUnread && highlightCount > 0 && !collapsed

  const onMarkedRead = useCallback(() => {
    setReadEpoch((n) => n + 1)
  }, [])

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
            // Keep a quiet secondary cue on Settings when a binary
            // update is waiting; the primary Updates row carries the
            // "what's new" unread state.
            const showSettingsDot = to === '/settings' && updateAvailable
            const updateLabel = t('nav.updateAvailable')
            const link = (
              <Link
                key={to}
                to={to}
                aria-label={
                  showSettingsDot ? `${label} — ${updateLabel}` : label
                }
                className={cn(
                  'flex items-center h-7 rounded-md text-[13px] transition-all duration-100',
                  'text-muted-foreground hover:text-foreground hover:bg-card',
                  active && 'bg-card text-foreground',
                  collapsed ? 'justify-center px-0' : 'gap-2.5 px-2.5',
                )}
              >
                <span className="relative shrink-0">
                  <Icon className="size-3.5 shrink-0" />
                  {showSettingsDot && (
                    <span
                      className="absolute -right-0.5 -top-0.5 size-1.5 rounded-full bg-accent ring-2 ring-card"
                      aria-hidden
                    />
                  )}
                </span>
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
                  {showSettingsDot && (
                    <span className="ml-2 text-accent">{updateLabel}</span>
                  )}
                </TooltipContent>
              </Tooltip>
            )
          })}
        </Fragment>
      ))}

      {/* Community / docs / updates — under Settings, above the fold
          bottom of the rail. mt-auto pins the block to the bottom when
          the nav is tall enough. */}
      <div
        className={cn(
          'mt-auto pt-2 border-t border-border/60',
          collapsed ? 'mx-1.5' : 'mx-0',
        )}
      >
        {!collapsed && (
          <div className="px-2 pb-1.5 pt-1 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
            {t('nav.resources')}
          </div>
        )}

        {/* Updates — native drawer, not an external link */}
        {(() => {
          const label = t('nav.updates.title')
          const badgeLabel = showUpdatesBadge
            ? showCountChip
              ? t('nav.updates.badgeCount', { count: highlightCount })
              : t('nav.updates.unread')
            : label
          const btn = (
            <button
              type="button"
              onClick={() => setUpdatesOpen(true)}
              aria-label={badgeLabel}
              className={cn(
                'w-full flex items-center h-7 rounded-md text-[13px] transition-all duration-100',
                'text-muted-foreground hover:text-foreground hover:bg-card',
                updatesOpen && 'bg-card text-foreground',
                collapsed ? 'justify-center px-0' : 'gap-2.5 px-2.5',
              )}
            >
              <span className="relative shrink-0">
                <Sparkles className="size-3.5 shrink-0" />
                {showUpdatesBadge && (
                  // Accent (orange) when binary update is available;
                  // primary (blue-ish) when only release notes are unread.
                  <span
                    className={cn(
                      'absolute -right-0.5 -top-0.5 size-1.5 rounded-full ring-2 ring-card',
                      updateAvailable ? 'bg-accent' : 'bg-primary',
                    )}
                    aria-hidden
                  />
                )}
              </span>
              {!collapsed && (
                <>
                  <span className="flex-1 text-left">{label}</span>
                  {showCountChip && (
                    <span className="text-[10px] tabular-nums text-muted-foreground">
                      · {highlightCount}
                    </span>
                  )}
                </>
              )}
            </button>
          )
          if (!collapsed) return btn
          return (
            <Tooltip>
              <TooltipTrigger asChild>{btn}</TooltipTrigger>
              <TooltipContent side="right">
                {label}
                {showUpdatesBadge && (
                  <span className="ml-2 text-accent">
                    {updateAvailable
                      ? t('nav.updateAvailable')
                      : t('nav.updates.unread')}
                  </span>
                )}
              </TooltipContent>
            </Tooltip>
          )
        })()}

        {externalLinks.map(({ id, href, icon: Icon, labelKey }) => {
          const label = t(labelKey)
          const link = (
            <a
              key={id}
              href={href}
              target="_blank"
              rel="noreferrer"
              aria-label={label}
              className={cn(
                'flex items-center h-7 rounded-md text-[13px] transition-all duration-100',
                'text-muted-foreground hover:text-foreground hover:bg-card',
                collapsed ? 'justify-center px-0' : 'gap-2.5 px-2.5',
              )}
            >
              <Icon className="size-3.5 shrink-0" />
              {!collapsed && (
                <>
                  <span className="flex-1">{label}</span>
                  <ExternalLink className="size-3 opacity-40" aria-hidden />
                </>
              )}
            </a>
          )
          if (!collapsed) return link
          return (
            <Tooltip key={id}>
              <TooltipTrigger asChild>{link}</TooltipTrigger>
              <TooltipContent side="right">{label}</TooltipContent>
            </Tooltip>
          )
        })}
      </div>

      <UpdatesDrawer
        open={updatesOpen}
        onClose={() => setUpdatesOpen(false)}
        onMarkedRead={onMarkedRead}
      />
    </nav>
  )
}
