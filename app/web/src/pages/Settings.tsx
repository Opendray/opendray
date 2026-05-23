import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearch } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
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
  ExternalLink,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import { toast } from 'sonner'

import { api } from '@/lib/api'
import { getVersionInfo, requestSelfUpdate, type VersionInfo } from '@/lib/version'
import { useTheme, type ThemeMode } from '@/stores/theme'
import { useAuth } from '@/stores/auth'
import { useLayout } from '@/stores/layout'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  ServerSettings,
  SettingsSearchInput,
  SERVER_SECTIONS,
  useServerSectionLabel,
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

// Map item keys → i18n key suffixes; labels resolved at render time
// inside the sidebar.
const TOP_GROUPS: {
  id: string
  titleKey: string
  items: { key: TopSection; labelKey: string; icon: LucideIcon }[]
}[] = [
  {
    id: 'workspace',
    titleKey: 'web.settings.groups.workspace',
    items: [
      { key: 'appearance', labelKey: 'web.settings.items.appearance', icon: Monitor },
      { key: 'font', labelKey: 'web.settings.items.font', icon: Type },
      { key: 'account', labelKey: 'web.settings.items.account', icon: UserIcon },
    ],
  },
]

const TOP_SECTION_KEYS = new Set<string>([
  'appearance',
  'font',
  'account',
  'system',
  'about',
])

function isValidTopSection(s: string | undefined): s is TopSection {
  if (!s) return false
  if (TOP_SECTION_KEYS.has(s)) return true
  if (s.startsWith('server.')) {
    return SERVER_SECTIONS.some((x) => `server.${x.id}` === s)
  }
  return false
}

export function SettingsPage() {
  const { t } = useTranslation()
  const serverSectionLabel = useServerSectionLabel()
  // Deep-link: /settings?section=server.memory selects that section on
  // mount. The Memory page "Configuration →" button uses this so users
  // land on the memory config instead of the default Appearance.
  const sp = useSearch({ strict: false }) as { section?: string }
  const initialSection = isValidTopSection(sp.section) ? sp.section : 'appearance'
  const [active, setActive] = useState<TopSection>(initialSection)
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (isValidTopSection(sp.section) && sp.section !== active) {
      setActive(sp.section)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sp.section])

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
    <div className="flex h-full min-h-0 overflow-hidden">
      {/* Sidebar */}
      <aside className="w-60 shrink-0 border-r border-border bg-background flex flex-col">
        <div className="px-5 pt-6 pb-3">
          <h1 className="text-[15px] font-semibold tracking-tight">
            {t('web.settings.title')}
          </h1>
          <p className="text-[11px] text-muted-foreground mt-0.5">
            {t('web.settings.subtitle')}
          </p>
        </div>

        <nav className="flex-1 overflow-y-auto px-2 pb-6">
          {TOP_GROUPS.map((g) => (
            <SidebarGroup key={g.id} title={t(g.titleKey)}>
              {g.items.map((item) => (
                <SidebarItem
                  key={item.key}
                  icon={item.icon}
                  label={t(item.labelKey)}
                  active={active === item.key}
                  onClick={() => setActive(item.key)}
                />
              ))}
            </SidebarGroup>
          ))}

          <SidebarGroup title={t('web.settings.groups.server')}>
            {SERVER_SECTIONS.map((s) => {
              const key: TopSection = `server.${s.id}`
              return (
                <SidebarItem
                  key={s.id}
                  icon={Settings2}
                  label={serverSectionLabel(s.id).title}
                  active={active === key}
                  onClick={() => setActive(key)}
                />
              )
            })}
          </SidebarGroup>

          <SidebarGroup title={t('web.settings.groups.system')}>
            <SidebarItem
              icon={Activity}
              label={t('web.settings.items.status')}
              active={active === 'system'}
              onClick={() => setActive('system')}
            />
            <SidebarItem
              icon={Info}
              label={t('web.settings.items.about')}
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
              ? `${health.version} · ${
                  health.db_ok
                    ? t('web.settings.health.dbOk')
                    : t('web.settings.health.dbDown')
                }`
              : t('web.settings.health.connecting')}
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
              <span className="text-[11px] text-muted-foreground">
                {t('web.settings.breadcrumb.server')}
              </span>
              <ChevronRight className="size-3 text-muted-foreground/40" />
              <span className="text-[11px] text-foreground font-medium">
                {(() => {
                  const id = SERVER_SECTIONS.find(
                    (s) => `server.${s.id}` === active,
                  )?.id
                  return id ? serverSectionLabel(id).title : ''
                })()}
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
  const { t } = useTranslation()
  const themeOptions: {
    mode: ThemeMode
    labelKey: string
    descKey: string
    icon: LucideIcon
  }[] = [
    {
      mode: 'light',
      labelKey: 'web.settings.appearance.options.light',
      descKey: 'web.settings.appearance.options.lightDesc',
      icon: Sun,
    },
    {
      mode: 'dark',
      labelKey: 'web.settings.appearance.options.dark',
      descKey: 'web.settings.appearance.options.darkDesc',
      icon: Moon,
    },
    {
      mode: 'system',
      labelKey: 'web.settings.appearance.options.system',
      descKey: 'web.settings.appearance.options.systemDesc',
      icon: Monitor,
    },
  ]
  return (
    <div>
      <SectionHeader
        title={t('web.settings.appearance.title')}
        description={t('web.settings.appearance.description')}
      />
      <div className="grid grid-cols-3 gap-2">
        {themeOptions.map(({ mode: m, labelKey, descKey, icon: Icon }) => {
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
                <span className="text-[13px] font-medium">{t(labelKey)}</span>
                <span className="text-[11px] text-muted-foreground leading-snug">
                  {t(descKey)}
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
  const { t } = useTranslation()
  const opts: { scale: number; labelKey: string }[] = [
    { scale: 0.85, labelKey: 'web.settings.font.options.compact' },
    { scale: 1, labelKey: 'web.settings.font.options.default' },
    { scale: 1.15, labelKey: 'web.settings.font.options.comfy' },
    { scale: 1.3, labelKey: 'web.settings.font.options.large' },
  ]
  return (
    <div>
      <SectionHeader
        title={t('web.settings.font.title')}
        description={t('web.settings.font.description')}
      />
      <div className="grid grid-cols-4 gap-2">
        {opts.map(({ scale, labelKey }) => {
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
                <span className="text-[13px] font-medium">{t(labelKey)}</span>
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
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <div>
      <SectionHeader
        title={t('web.settings.account.title')}
        description={t('web.settings.account.description')}
      />
      <div className="flex flex-col gap-1.5">
        <Field label={t('web.settings.account.username')} value={username ?? '—'} />
        <Field
          label={t('web.settings.account.tokenExpires')}
          value={expiresAt ? new Date(expiresAt).toLocaleString() : '—'}
          monospace
        />
      </div>
      <div className="mt-3">
        <button
          type="button"
          className="text-[12px] px-3 py-1.5 rounded-md border border-border hover:bg-card transition-colors"
          onClick={() => setOpen(true)}
        >
          {t('web.settings.account.changeCredentials')}
        </button>
      </div>
      {open && (
        <ChangeCredentialsDialog
          currentUsername={username ?? ''}
          onClose={() => setOpen(false)}
        />
      )}
    </div>
  )
}

// ChangeCredentialsDialog mirrors the mobile flow: verify current
// password, pick new username + password, server hot-swaps the
// hashed-cred keyfile and returns a fresh token under the new
// credentials. The new token replaces the existing one in zustand
// so the operator stays signed in.
function ChangeCredentialsDialog({
  currentUsername,
  onClose,
}: {
  currentUsername: string
  onClose: () => void
}) {
  const { t } = useTranslation()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newUser, setNewUser] = useState(currentUsername)
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const setSession = useAuth((s) => s.setSession)

  async function submit() {
    setError(null)
    if (newPassword.length < 8) {
      setError(t('web.settings.changeCredentials.errorTooShort'))
      return
    }
    if (newPassword !== confirm) {
      setError(t('web.settings.changeCredentials.errorMismatch'))
      return
    }
    setBusy(true)
    try {
      const res = await api<{
        token: string
        username: string
        issued_at: string
        expires_at: string
      }>('/api/v1/auth/change-credentials', {
        method: 'POST',
        body: JSON.stringify({
          current_password: currentPassword,
          new_user: newUser.trim() || undefined,
          new_password: newPassword,
        }),
        headers: { 'Content-Type': 'application/json' },
      })
      // Replace the existing zustand session with the fresh token
      // returned by the server. Without this, the very next
      // request would 401 because the old token was revoked
      // server-side.
      setSession(res.token, res.username, res.expires_at)
      onClose()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      // Distinguish wrong-current-password (401) from validation
      // failures so the operator knows where to look.
      if (msg.includes('401') || msg.toLowerCase().includes('invalid')) {
        setError(t('web.settings.changeCredentials.errorWrongPassword'))
      } else {
        setError(msg)
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={onClose}
    >
      <div
        className="w-[min(440px,calc(100vw-2rem))] max-h-[calc(100vh-2rem)] overflow-y-auto rounded-md border border-border bg-background p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="font-medium text-[15px]">{t('web.settings.changeCredentials.title')}</div>
        <div className="text-muted-foreground text-[12px] mt-1">
          {t('web.settings.changeCredentials.description')}
        </div>

        <div className="mt-4 flex flex-col gap-3 text-[13px]">
          <label className="flex flex-col gap-1">
            <span className="text-[11px] text-muted-foreground">
              {t('web.settings.changeCredentials.currentPassword')}
            </span>
            <input
              type="password"
              autoComplete="current-password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              className="h-8 px-2 rounded-md border border-border bg-input/40"
              autoFocus
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-[11px] text-muted-foreground">
              {t('web.settings.changeCredentials.newUsername')}
            </span>
            <input
              type="text"
              autoComplete="username"
              value={newUser}
              onChange={(e) => setNewUser(e.target.value)}
              className="h-8 px-2 rounded-md border border-border bg-input/40 font-mono"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-[11px] text-muted-foreground">
              {t('web.settings.changeCredentials.newPassword')}
            </span>
            <input
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="h-8 px-2 rounded-md border border-border bg-input/40"
            />
            <span className="text-[11px] text-muted-foreground">
              {t('web.settings.changeCredentials.newPasswordHint')}
            </span>
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-[11px] text-muted-foreground">
              {t('web.settings.changeCredentials.confirm')}
            </span>
            <input
              type="password"
              autoComplete="new-password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className="h-8 px-2 rounded-md border border-border bg-input/40"
            />
          </label>
        </div>

        {error && (
          <div className="mt-3 rounded-md border border-destructive/40 bg-destructive/10 p-2 text-[12px] text-destructive">
            {error}
          </div>
        )}

        <div className="mt-4 flex justify-end gap-2">
          <button
            type="button"
            className="text-[12px] px-3 py-1.5 rounded-md border border-border hover:bg-card transition-colors"
            onClick={onClose}
            disabled={busy}
          >
            {t('web.settings.changeCredentials.cancel')}
          </button>
          <button
            type="button"
            className="text-[12px] px-3 py-1.5 rounded-md border border-accent bg-accent text-accent-foreground hover:bg-accent/90 transition-colors disabled:opacity-50"
            onClick={() => void submit()}
            disabled={busy}
          >
            {busy
              ? t('web.settings.changeCredentials.saving')
              : t('web.settings.changeCredentials.update')}
          </button>
        </div>
      </div>
    </div>
  )
}

function SystemSection({ health }: { health: HealthResponse | undefined }) {
  const { t } = useTranslation()
  return (
    <div>
      <SectionHeader
        title={t('web.settings.system.title')}
        description={t('web.settings.system.description')}
      />
      <div className="flex flex-col gap-1.5">
        <Field label={t('web.settings.system.status')} value={health?.status ?? '…'} />
        <Field
          label={t('web.settings.system.version')}
          value={
            health ? `${health.version} (${health.commit.slice(0, 7)})` : '…'
          }
          monospace
        />
        <Field
          label={t('web.settings.system.uptime')}
          value={health ? formatUptime(health.uptime_s) : '…'}
        />
        <Field
          label={t('web.settings.system.database')}
          value={
            health
              ? health.db_ok
                ? t('web.settings.system.reachable')
                : t('web.settings.system.unreachable')
              : '…'
          }
          tone={health?.db_ok === false ? 'fail' : 'ok'}
        />
      </div>
    </div>
  )
}

function AboutSection() {
  const { t } = useTranslation()
  const { data, refetch } = useQuery<VersionInfo>({
    queryKey: ['version'],
    queryFn: getVersionInfo,
  })
  // phase: idle → confirming (inline confirm) → upgrading (polling for the
  // restarted daemon to come back on the new version).
  const [phase, setPhase] = useState<'idle' | 'confirming' | 'upgrading'>('idle')

  // While upgrading, the daemon downloads + swaps + restarts, so requests
  // fail mid-flight. Poll until `current` reaches `latest` (or give up).
  useEffect(() => {
    if (phase !== 'upgrading') return
    const target = data?.latest
    let tries = 0
    const id = setInterval(async () => {
      tries++
      try {
        const v = await getVersionInfo()
        if (!v.updateAvailable || (target && v.current === target)) {
          clearInterval(id)
          setPhase('idle')
          toast.success(t('web.settings.about.upgraded', { version: v.current }))
          void refetch()
          return
        }
      } catch {
        /* expected during the restart window — keep polling */
      }
      if (tries > 30) {
        clearInterval(id)
        setPhase('idle')
        toast.message(t('web.settings.about.upgradeSlow'))
        void refetch()
      }
    }, 4000)
    return () => clearInterval(id)
  }, [phase, data?.latest, refetch, t])

  const [checking, setChecking] = useState(false)
  // confirmForce distinguishes a normal upgrade from a force re-install
  // (the "Re-install" action shown when already on the latest).
  const [confirmForce, setConfirmForce] = useState(false)

  async function checkNow() {
    setChecking(true)
    try {
      const r = await refetch()
      const info = r.data
      if (info?.checkError) toast.error(t('web.settings.about.checkFailed'))
      else if (info?.updateAvailable)
        toast.success(t('web.settings.about.updateAvailable', { version: info.latest }))
      else toast.message(t('web.settings.about.upToDate'))
    } finally {
      setChecking(false)
    }
  }

  async function startUpgrade(force = false) {
    try {
      const res = await requestSelfUpdate(force)
      if (res.error) {
        toast.error(res.error)
        setPhase('idle')
        return
      }
      toast.success(t('web.settings.about.upgrading', { version: res.to ?? '' }), {
        description: res.note,
      })
      setPhase('upgrading')
    } catch (e) {
      toast.error(String(e))
      setPhase('idle')
    }
  }

  const busy = phase === 'upgrading' || !!data?.pending
  const canSelfUpdate = !!data?.selfUpdate
  const buildDate = formatBuildDate(data?.date)
  const channel = deriveChannel(data?.current)
  return (
    <div className="space-y-5">
      <SectionHeader
        title={t('web.settings.about.title')}
        description={t('web.settings.about.description')}
      />

      {/* Product identity — flat, matching the System/Server section rhythm */}
      <div>
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-baseline gap-2">
            <span className="text-[15px] font-semibold tracking-tight text-foreground">
              OpenDray
            </span>
            <span className="font-mono text-[12px] text-muted-foreground">
              {data?.current ?? '…'}
            </span>
          </div>
          <AboutStatusBadge data={data} />
        </div>
        <p className="mt-0.5 text-[12px] text-muted-foreground">
          {t('web.settings.about.tagline')}
        </p>
      </div>

      {/* Build & legal details */}
      <div className="flex flex-col gap-1.5">
        <Field
          label={t('web.settings.about.version')}
          value={data?.current ?? '…'}
          monospace
        />
        {data?.commit && (
          <Field label={t('web.settings.about.commit')} value={data.commit} monospace />
        )}
        {buildDate && (
          <Field label={t('web.settings.about.buildDate')} value={buildDate} monospace />
        )}
        {data?.platform && (
          <Field label={t('web.settings.about.platform')} value={data.platform} monospace />
        )}
        <Field label={t('web.settings.about.channel')} value={channel} monospace />
        <Field label={t('web.settings.about.license')} value="Apache-2.0" monospace />
      </div>

      {/* Resources */}
      <div>
        <div className="mb-3 flex items-center gap-3">
          <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
            {t('web.settings.about.resourcesHeading')}
          </span>
          <span className="h-px flex-1 bg-border" />
        </div>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
          {RESOURCE_LINKS.map((l) => (
            <a
              key={l.key}
              href={l.href}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-[12px] text-accent underline-offset-4 hover:underline"
            >
              {t(l.label)}
              <ExternalLink className="h-3 w-3" />
            </a>
          ))}
        </div>
      </div>

      {/* Software updates */}
      <div className="pt-1">
        <div className="mb-3 flex items-center gap-3">
          <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
            {t('web.settings.about.updatesHeading')}
          </span>
          <span className="h-px flex-1 bg-border" />
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-[12px] text-muted-foreground">
            {data?.updateAvailable ? (
              <>
                <span className="font-medium text-foreground">
                  {t('web.settings.about.updateAvailable', { version: data.latest })}
                </span>
                {data.notesUrl && (
                  <a
                    href={data.notesUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="ml-2 inline-flex items-center gap-1 text-accent underline-offset-4 hover:underline"
                  >
                    {t('web.settings.about.releaseNotes')}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                )}
              </>
            ) : data?.checkError ? (
              t('web.settings.about.checkFailed')
            ) : (
              t('web.settings.about.upToDate')
            )}
          </p>

          {phase === 'confirming' ? (
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-[11px] text-muted-foreground">
                {t('web.settings.about.confirmRestart')}
              </span>
              <Button size="sm" onClick={() => startUpgrade(confirmForce)}>
                {t('web.settings.about.confirmUpgrade')}
              </Button>
              <Button size="sm" variant="outline" onClick={() => setPhase('idle')}>
                {t('common.cancel')}
              </Button>
            </div>
          ) : (
            <div className="flex flex-wrap items-center gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={checkNow}
                disabled={checking || busy}
              >
                {checking
                  ? t('web.settings.about.checking')
                  : t('web.settings.about.checkUpdates')}
              </Button>

              {data?.updateAvailable && canSelfUpdate && (
                <Button
                  size="sm"
                  onClick={() => {
                    setConfirmForce(false)
                    setPhase('confirming')
                  }}
                  disabled={busy}
                >
                  {busy
                    ? t('web.settings.about.upgradingShort')
                    : t('web.settings.about.updateNow')}
                </Button>
              )}

              {!data?.updateAvailable && !data?.checkError && canSelfUpdate && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setConfirmForce(true)
                    setPhase('confirming')
                  }}
                  disabled={busy}
                >
                  {busy
                    ? t('web.settings.about.upgradingShort')
                    : t('web.settings.about.reinstall')}
                </Button>
              )}
            </div>
          )}
        </div>

        {data?.updateAvailable && !canSelfUpdate && (
          <p className="mt-2 text-[11px] text-muted-foreground">
            {t('web.settings.about.guidedHint')}
            <code className="ml-1 font-mono text-foreground">opendray update</code>
          </p>
        )}
      </div>
    </div>
  )
}

// Compact status pill for the About identity card. Mirrors the version
// feed: update-available → warning, check failed → muted, else success.
function AboutStatusBadge({ data }: { data?: VersionInfo }) {
  const { t } = useTranslation()
  const dot = <span className="h-1.5 w-1.5 rounded-full bg-current" />
  if (!data)
    return (
      <Badge variant="muted">
        <span className="h-1.5 w-1.5 rounded-full bg-current opacity-60" />…
      </Badge>
    )
  if (data.updateAvailable)
    return (
      <Badge variant="warning">
        {dot}
        {t('web.settings.about.statusUpdate')}
      </Badge>
    )
  if (data.checkError)
    return (
      <Badge variant="muted">
        {dot}
        {t('web.settings.about.statusOffline')}
      </Badge>
    )
  return (
    <Badge variant="success">
      {dot}
      {t('web.settings.about.statusCurrent')}
    </Badge>
  )
}

// External resources shown in the About section. Labels are i18n keys.
const RESOURCE_LINKS = [
  { key: 'docs', label: 'web.settings.about.linkDocs', href: 'https://opendray.dev' },
  {
    key: 'releases',
    label: 'web.settings.about.releaseNotes',
    href: 'https://github.com/Opendray/opendray/releases',
  },
  {
    key: 'source',
    label: 'web.settings.about.linkSource',
    href: 'https://github.com/Opendray/opendray',
  },
  {
    key: 'security',
    label: 'web.settings.about.linkSecurity',
    href: 'https://github.com/Opendray/opendray/security/policy',
  },
] as const

// formatBuildDate renders the release build timestamp as a short local date,
// or "" when absent / a dev build (so the row is hidden).
function formatBuildDate(date?: string): string {
  if (!date || date === 'unknown') return ''
  const d = new Date(date)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

// deriveChannel infers the release channel from the version string:
// a "+build" suffix (e.g. "2.2.0+ct128") is a custom build, "-dev"/"snapshot"
// is a pre-release, everything else is a tagged stable release.
function deriveChannel(current?: string): string {
  if (!current) return '…'
  if (/dev|snapshot/i.test(current)) return 'dev'
  if (current.includes('+')) return 'custom'
  return 'stable'
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
