import { useQuery } from '@tanstack/react-query'
import { Sun, Moon, Monitor, Check, Type } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import { api } from '@/lib/api'
import { useTheme, type ThemeMode } from '@/stores/theme'
import { useAuth } from '@/stores/auth'
import { useLayout } from '@/stores/layout'
import { cn } from '@/lib/utils'

interface HealthResponse {
  status: string
  version: string
  commit: string
  uptime_s: number
  db_ok: boolean
}

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
    description: 'Follow the operating system setting',
    icon: Monitor,
  },
]

const fontScaleOptions: { scale: number; label: string }[] = [
  { scale: 0.85, label: 'Compact' },
  { scale: 1, label: 'Default' },
  { scale: 1.15, label: 'Comfy' },
  { scale: 1.3, label: 'Large' },
]

export function SettingsPage() {
  const mode = useTheme((s) => s.mode)
  const setMode = useTheme((s) => s.setMode)
  const fontScale = useLayout((s) => s.fontScale)
  const setFontScale = useLayout((s) => s.setFontScale)
  const username = useAuth((s) => s.username)
  const expiresAt = useAuth((s) => s.expiresAt)

  const { data: health } = useQuery<HealthResponse>({
    queryKey: ['health'],
    queryFn: () => api<HealthResponse>('/api/v1/health'),
    refetchInterval: 15_000,
  })

  return (
    <div className="max-w-[640px] mx-auto p-6 flex flex-col gap-8">
      <div>
        <h1 className="text-[18px] font-semibold tracking-tight">Settings</h1>
        <p className="text-[12px] text-muted-foreground">
          Workspace preferences and operator account.
        </p>
      </div>

      <Section title="Appearance" description="Choose how opendray looks.">
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
      </Section>

      <Section
        title="Font size"
        description="Scales the entire interface (titles, body, icons, terminal). Persisted per browser."
      >
        <div className="grid grid-cols-4 gap-2">
          {fontScaleOptions.map(({ scale, label }) => {
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
      </Section>

      <Section title="Account" description="Operator and current bearer token.">
        <Field label="Username" value={username ?? '—'} />
        <Field
          label="Token expires"
          value={expiresAt ? new Date(expiresAt).toLocaleString() : '—'}
          monospace
        />
      </Section>

      <Section
        title="System"
        description="Live status from the gateway's /health endpoint."
      >
        <Field label="Status" value={health?.status ?? '…'} />
        <Field
          label="Version"
          value={
            health
              ? `${health.version} (${health.commit.slice(0, 7)})`
              : '…'
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
      </Section>

      <Section title="About">
        <p className="text-[12px] text-muted-foreground leading-relaxed">
          opendray v2 — the multiplexer + integration gateway for AI agent CLIs.
          Source under Apache 2.0.
        </p>
      </Section>
    </div>
  )
}

function Section({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: React.ReactNode
}) {
  return (
    <section className="flex flex-col gap-3">
      <div>
        <h2 className="text-[13px] font-semibold">{title}</h2>
        {description && (
          <p className="text-[11px] text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="flex flex-col gap-1.5">{children}</div>
    </section>
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
