import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Sun,
  Moon,
  Monitor,
  Check,
  Type,
  User as UserIcon,
  Server,
  Settings2,
  Info,
  Activity,
  ChevronRight,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import { api } from '@/lib/api'
import { useTheme, type ThemeMode } from '@/stores/theme'
import { useAuth } from '@/stores/auth'
import { useLayout } from '@/stores/layout'
import { cn } from '@/lib/utils'
import {
  ServerSettings,
  SettingsSearchInput,
  SERVER_SECTIONS,
  type ServerSectionId,
} from '@/components/settings/ServerSettings'

interface HealthResponse {
  status: string
  version: string
  commit: string
  uptime_s: number
  db_ok: boolean
}

// Top-level sections shown in the left sidebar. Server sub-sections
// expand inline below the "Server" group.
type TopSection =
  | 'appearance'
  | 'font'
  | 'account'
  | `server.${ServerSectionId}`
  | 'system'
  | 'about'

const TOP_GROUPS: {
  id: string
  title: string
  items: { key: TopSection; label: string; icon: LucideIcon }[]
}[] = [
  {
    id: 'workspace',
    title: 'Workspace',
    items: [
      { key: 'appearance', label: 'Appearance', icon: Monitor },
      { key: 'font', label: 'Font size', icon: Type },
      { key: 'account', label: 'Account', icon: UserIcon },
    ],
  },
]

export function SettingsPage() {
  const [active, setActive] = useState<TopSection>('appearance')
  const [search, setSearch] = useState('')

  const username = useAuth((s) => s.username)
  const expiresAt = useAuth((s) => s.expiresAt)
  const mode = useTheme((s) => s.mode)
  const setMode = useTheme((s) => s.setMode)
  const fontScale = useLayout((s) => s.fontScale)
  const setFontScale = useLayout((s) => s.setFontScale)

  const { data: health } = useQuery<HealthResponse>({
    queryKey: ['health'],
    queryFn: () => api<HealthResponse>('/api/v1/health'),
    refetchInterval: 15_000,
  })

  return (
    <div className="flex h-full min-h-0">
      {/* Sidebar */}
      <aside className="w-60 shrink-0 border-r border-border bg-background flex flex-col">
        <div className="px-5 pt-6 pb-3">
          <h1 className="text-[15px] font-semibold tracking-tight">Settings</h1>
          <p className="text-[11px] text-muted-foreground mt-0.5">
            Workspace, account, and gateway config.
          </p>
        </div>

        <nav className="flex-1 overflow-y-auto px-2 pb-6">
          {TOP_GROUPS.map((g) => (
            <SidebarGroup key={g.id} title={g.title}>
              {g.items.map((item) => (
                <SidebarItem
                  key={item.key}
                  icon={item.icon}
                  label={item.label}
                  active={active === item.key}
                  onClick={() => setActive(item.key)}
                />
              ))}
            </SidebarGroup>
          ))}

          <SidebarGroup title="Server">
            {SERVER_SECTIONS.map((s) => {
              const key: TopSection = `server.${s.id}`
              return (
                <SidebarItem
                  key={s.id}
                  icon={Settings2}
                  label={s.title}
                  active={active === key}
                  onClick={() => setActive(key)}
                />
              )
            })}
          </SidebarGroup>

          <SidebarGroup title="System">
            <SidebarItem
              icon={Activity}
              label="Status"
              active={active === 'system'}
              onClick={() => setActive('system')}
            />
            <SidebarItem
              icon={Info}
              label="About"
              active={active === 'about'}
              onClick={() => setActive('about')}
            />
          </SidebarGroup>
        </nav>

        {/* Mini health badge at the bottom */}
        <div className="border-t border-border px-4 py-3 flex items-center gap-2 text-[10.5px]">
          <span
            className={cn(
              'size-1.5 rounded-full shrink-0',
              health?.db_ok ? 'bg-emerald-400' : 'bg-rose-400',
              !health && 'bg-muted-foreground/40 animate-pulse',
            )}
          />
          <span className="text-muted-foreground truncate">
            {health
              ? `${health.version} · ${health.db_ok ? 'db ok' : 'db down'}`
              : 'connecting…'}
          </span>
        </div>
      </aside>

      {/* Content */}
      <div className="flex-1 min-w-0 overflow-y-auto">
        <div className="max-w-[860px] mx-auto px-8 py-8">
          {/* Sticky search row, only shown when a server section is active */}
          {active.startsWith('server.') && (
            <div className="flex items-center gap-3 mb-6">
              <Server className="size-3.5 text-muted-foreground/60" />
              <span className="text-[11px] text-muted-foreground">Server</span>
              <ChevronRight className="size-3 text-muted-foreground/40" />
              <span className="text-[11px] text-foreground font-medium">
                {SERVER_SECTIONS.find((s) => `server.${s.id}` === active)?.title}
              </span>
              <div className="ml-auto">
                <SettingsSearchInput value={search} onChange={setSearch} />
              </div>
            </div>
          )}

          <ContentRouter
            active={active}
            mode={mode}
            setMode={setMode}
            fontScale={fontScale}
            setFontScale={setFontScale}
            username={username}
            expiresAt={expiresAt}
            health={health}
            search={search}
          />
        </div>
      </div>
    </div>
  )
}

function ContentRouter({
  active,
  mode,
  setMode,
  fontScale,
  setFontScale,
  username,
  expiresAt,
  health,
  search,
}: {
  active: TopSection
  mode: ThemeMode
  setMode: (m: ThemeMode) => void
  fontScale: number
  setFontScale: (s: number) => void
  username: string | null
  expiresAt: string | null
  health: HealthResponse | undefined
  search: string
}) {
  if (active.startsWith('server.')) {
    const sectionId = active.slice('server.'.length) as ServerSectionId
    return <ServerSettings activeSection={sectionId} searchQuery={search} />
  }

  switch (active) {
    case 'appearance':
      return <AppearanceSection mode={mode} setMode={setMode} />
    case 'font':
      return (
        <FontSection fontScale={fontScale} setFontScale={setFontScale} />
      )
    case 'account':
      return <AccountSection username={username} expiresAt={expiresAt} />
    case 'system':
      return <SystemSection health={health} />
    case 'about':
      return <AboutSection />
  }
}

function SidebarGroup({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="mt-3 first:mt-0">
      <p className="px-3 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/50">
        {title}
      </p>
      <div className="flex flex-col gap-0.5">{children}</div>
    </div>
  )
}

function SidebarItem({
  icon: Icon,
  label,
  active,
  onClick,
}: {
  icon: LucideIcon
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 px-3 py-1.5 rounded text-[12px] text-left transition-colors',
        active
          ? 'bg-card text-foreground font-medium'
          : 'text-muted-foreground hover:text-foreground hover:bg-card/50',
      )}
    >
      <Icon className="size-3.5 shrink-0 opacity-70" />
      <span className="truncate">{label}</span>
    </button>
  )
}

function SectionHeader({
  title,
  description,
}: {
  title: string
  description?: string
}) {
  return (
    <header className="mb-5 pb-3 border-b border-border">
      <h2 className="text-[15px] font-semibold tracking-tight">{title}</h2>
      {description && (
        <p className="text-[12px] text-muted-foreground mt-0.5">{description}</p>
      )}
    </header>
  )
}

function AppearanceSection({
  mode,
  setMode,
}: {
  mode: ThemeMode
  setMode: (m: ThemeMode) => void
}) {
  const themeOptions: {
    mode: ThemeMode
    label: string
    description: string
    icon: LucideIcon
  }[] = [
    { mode: 'light', label: 'Light', description: 'Always light', icon: Sun },
    { mode: 'dark', label: 'Dark', description: 'Always dark', icon: Moon },
    {
      mode: 'system',
      label: 'System',
      description: 'Follow the OS setting',
      icon: Monitor,
    },
  ]
  return (
    <div>
      <SectionHeader
        title="Appearance"
        description="Choose how opendray looks."
      />
      <div className="grid grid-cols-3 gap-2">
        {themeOptions.map(({ mode: m, label, description, icon: Icon }) => {
          const active = mode === m
          return (
            <button
              key={m}
              type="button"
              onClick={() => setMode(m)}
              className={cn(
                'relative flex flex-col gap-2 items-start text-left p-3 rounded-md border transition-colors',
                active
                  ? 'border-foreground/30 bg-card'
                  : 'border-border hover:bg-card hover:border-foreground/20',
              )}
            >
              <Icon className="size-4 text-muted-foreground" />
              <div className="flex flex-col gap-0.5">
                <span className="text-[13px] font-medium">{label}</span>
                <span className="text-[11px] text-muted-foreground leading-snug">
                  {description}
                </span>
              </div>
              {active && (
                <Check className="absolute right-2 top-2 size-3 text-accent" />
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}

function FontSection({
  fontScale,
  setFontScale,
}: {
  fontScale: number
  setFontScale: (s: number) => void
}) {
  const opts: { scale: number; label: string }[] = [
    { scale: 0.85, label: 'Compact' },
    { scale: 1, label: 'Default' },
    { scale: 1.15, label: 'Comfy' },
    { scale: 1.3, label: 'Large' },
  ]
  return (
    <div>
      <SectionHeader
        title="Font size"
        description="Scales the entire interface. Persisted per browser."
      />
      <div className="grid grid-cols-4 gap-2">
        {opts.map(({ scale, label }) => {
          const active = Math.abs(fontScale - scale) < 0.001
          return (
            <button
              key={scale}
              type="button"
              onClick={() => setFontScale(scale)}
              className={cn(
                'relative flex flex-col gap-1 items-start text-left p-3 rounded-md border transition-colors',
                active
                  ? 'border-foreground/30 bg-card'
                  : 'border-border hover:bg-card hover:border-foreground/20',
              )}
            >
              <Type className="size-4 text-muted-foreground" />
              <div className="flex flex-col gap-0.5">
                <span className="text-[13px] font-medium">{label}</span>
                <span className="text-[11px] text-muted-foreground">
                  {Math.round(scale * 100)}%
                </span>
              </div>
              {active && (
                <Check className="absolute right-2 top-2 size-3 text-accent" />
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}

function AccountSection({
  username,
  expiresAt,
}: {
  username: string | null
  expiresAt: string | null
}) {
  return (
    <div>
      <SectionHeader
        title="Account"
        description="Operator and current bearer token."
      />
      <div className="flex flex-col gap-1.5">
        <Field label="Username" value={username ?? '—'} />
        <Field
          label="Token expires"
          value={expiresAt ? new Date(expiresAt).toLocaleString() : '—'}
          monospace
        />
      </div>
    </div>
  )
}

function SystemSection({ health }: { health: HealthResponse | undefined }) {
  return (
    <div>
      <SectionHeader
        title="System status"
        description="Live status from the gateway's /health endpoint."
      />
      <div className="flex flex-col gap-1.5">
        <Field label="Status" value={health?.status ?? '…'} />
        <Field
          label="Version"
          value={
            health ? `${health.version} (${health.commit.slice(0, 7)})` : '…'
          }
          monospace
        />
        <Field
          label="Uptime"
          value={health ? formatUptime(health.uptime_s) : '…'}
        />
        <Field
          label="Database"
          value={health ? (health.db_ok ? 'reachable' : 'unreachable') : '…'}
          tone={health?.db_ok === false ? 'fail' : 'ok'}
        />
      </div>
    </div>
  )
}

function AboutSection() {
  return (
    <div>
      <SectionHeader title="About" />
      <p className="text-[12px] text-muted-foreground leading-relaxed">
        opendray v2 — the multiplexer + integration gateway for AI agent CLIs.
        Source under Apache 2.0.
      </p>
    </div>
  )
}

function Field({
  label,
  value,
  monospace,
  tone,
}: {
  label: string
  value: string
  monospace?: boolean
  tone?: 'ok' | 'fail'
}) {
  return (
    <div className="flex items-baseline justify-between border-b border-border/60 py-1.5">
      <span className="text-[11px] text-muted-foreground">{label}</span>
      <span
        className={cn(
          'text-[12px]',
          monospace && 'font-mono',
          tone === 'fail' && 'text-destructive',
          tone === 'ok' && 'text-foreground',
        )}
      >
        {value}
      </span>
    </div>
  )
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const m = Math.floor(seconds / 60)
  if (m < 60) return `${m}m ${seconds % 60}s`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ${m % 60}m`
  return `${Math.floor(h / 24)}d ${h % 24}h`
}
