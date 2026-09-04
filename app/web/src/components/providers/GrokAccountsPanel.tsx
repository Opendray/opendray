import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CircleDot, Download, KeyRound, Loader2, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  deleteGrokAccount,
  importLocalGrokAccounts,
  listGrokAccounts,
  toggleGrokAccount,
} from '@/lib/grokAccounts'
import type { GrokAccount } from '@/lib/types'

// GrokAccountsPanel renders the multi-account list for the Grok Build
// (grok) provider. grok keys its credential + config state off GROK_HOME,
// so an account is a dedicated GROK_HOME dir holding its own auth.json
// login token. Account creation is guided-login driven: the operator runs
// `GROK_HOME=~/.grok-accounts/<name> grok login` on the gateway host,
// completes the xAI sign-in, then clicks Import local to surface the row.
//
// There is intentionally no Add-account form: the login token can only be
// minted by grok's interactive sign-in flow, so forcing the host-shell
// login keeps the affordance honest. Mirrors AntigravityAccountsPanel.

function relativeAgo(iso: string): string {
  const t = new Date(iso).getTime()
  if (!Number.isFinite(t)) return ''
  const dsec = Math.max(0, Math.floor((Date.now() - t) / 1000))
  if (dsec < 60) return `${dsec}s ago`
  if (dsec < 3600) return `${Math.floor(dsec / 60)}m ago`
  if (dsec < 86400) return `${Math.floor(dsec / 3600)}h ago`
  return `${Math.floor(dsec / 86400)}d ago`
}

export function GrokAccountsPanel() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: accounts, isLoading } = useQuery({
    queryKey: ['grok-accounts'],
    queryFn: listGrokAccounts,
    refetchInterval: 5000,
  })

  const importLocal = useMutation({
    mutationFn: importLocalGrokAccounts,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['grok-accounts'] })
      if (res.count === 0) {
        toast.success(t('web.providers.grokAccounts.importedNothingToast'))
      } else {
        toast.success(
          t('web.providers.grokAccounts.importedToast', { count: res.count }),
        )
      }
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.grokAccounts.importFailedToast'), {
        description: e.message,
      }),
  })

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      toggleGrokAccount(id, enabled),
    onMutate: async ({ id, enabled }) => {
      await qc.cancelQueries({ queryKey: ['grok-accounts'] })
      const prev = qc.getQueryData<GrokAccount[]>(['grok-accounts'])
      if (prev) {
        qc.setQueryData<GrokAccount[]>(
          ['grok-accounts'],
          prev.map((a) => (a.id === id ? { ...a, enabled } : a)),
        )
      }
      return { prev }
    },
    onError: (e: Error, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(['grok-accounts'], ctx.prev)
      toast.error(t('web.providers.grokAccounts.toggleFailedToast'), {
        description: e.message,
      })
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ['grok-accounts'] }),
  })

  const remove = useMutation({
    mutationFn: deleteGrokAccount,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['grok-accounts'] })
      toast.success(t('web.providers.grokAccounts.removedToast'))
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.grokAccounts.removeFailedToast'), {
        description: e.message,
      }),
  })

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-[12px] font-semibold uppercase tracking-wider text-muted-foreground/80">
            {t('web.providers.grokAccounts.title')}
          </h2>
          <span className="text-[10px] text-muted-foreground/60 font-mono">
            {accounts?.length ?? 0}
          </span>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => importLocal.mutate()}
          disabled={importLocal.isPending}
          className="text-[11px] gap-1"
          title={t('web.providers.grokAccounts.importLocalTooltip')}
        >
          {importLocal.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Download className="size-3.5" />
          )}
          {t('web.providers.grokAccounts.importLocal')}
        </Button>
      </div>

      <div className="rounded-md border border-border bg-muted/20 px-3 py-2.5 text-[11px] text-muted-foreground leading-relaxed">
        <span className="font-medium text-foreground">
          {t('web.providers.grokAccounts.addingTitle')}
        </span>{' '}
        {t('web.providers.grokAccounts.addingBodyPrefix')}
        <pre className="mt-1.5 mb-1.5 px-2 py-1.5 rounded bg-background/60 text-[10.5px] overflow-x-auto">
{`mkdir -p ~/.grok-accounts/<name>
GROK_HOME=~/.grok-accounts/<name> grok login   # complete xAI sign-in, then exit`}
        </pre>
        <Trans
          i18nKey="web.providers.grokAccounts.addingBodySuffix"
          components={{ 1: <span className="font-mono" /> }}
        />
      </div>

      {isLoading && (
        <div className="text-[12px] text-muted-foreground italic">
          {t('web.providers.grokAccounts.loading')}
        </div>
      )}

      {!isLoading && (accounts?.length ?? 0) === 0 && (
        <p className="text-[12px] text-muted-foreground italic">
          <Trans
            i18nKey="web.providers.grokAccounts.empty"
            shouldUnescape
            components={{ 1: <span className="font-mono" /> }}
          />
        </p>
      )}

      <div className="space-y-1.5">
        {(accounts ?? []).map((a) => (
          <div
            key={a.id}
            className="rounded-md border border-border px-3 py-2.5"
          >
            <div className="flex items-center gap-3">
              <KeyRound
                className={
                  a.token_filled
                    ? 'size-4 text-foreground/80 shrink-0'
                    : 'size-4 text-muted-foreground/50 shrink-0'
                }
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-[12px] font-medium">
                    {a.display_name || a.name}
                  </span>
                  <span className="text-[10px] text-muted-foreground/60 font-mono">
                    {a.name}
                  </span>
                  {!a.token_filled && (
                    <span className="text-[10px] uppercase tracking-wide text-amber-500/90 inline-flex items-center gap-1">
                      <CircleDot className="size-2.5" />
                      {t('web.providers.grokAccounts.noTokenYet')}
                    </span>
                  )}
                  <span
                    className="text-[10px] rounded px-1.5 py-0.5 bg-foreground/5 text-muted-foreground/80"
                    title="sessions currently pinned to this account"
                  >
                    {a.active_sessions ?? 0} active
                  </span>
                  {a.last_used_at && (
                    <span
                      className="text-[10px] text-muted-foreground/60"
                      title={`last session: ${a.last_used_at}`}
                    >
                      used {relativeAgo(a.last_used_at)}
                    </span>
                  )}
                </div>
                <div className="text-[10px] font-mono text-muted-foreground/70 truncate">
                  {t('web.providers.grokAccounts.homeDir')}{' '}
                  {a.config_dir || 'default'}
                </div>
              </div>
              <ToggleButton
                enabled={a.enabled}
                pending={toggle.isPending}
                onToggle={(v) => toggle.mutate({ id: a.id, enabled: v })}
                ariaLabel={t('web.providers.grokAccounts.toggleAria', {
                  name: a.name,
                })}
              />
              <Button
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground hover:text-destructive"
                onClick={() => {
                  if (
                    confirm(
                      t('web.providers.grokAccounts.removeConfirm', {
                        name: a.name,
                      }),
                    )
                  ) {
                    remove.mutate(a.id)
                  }
                }}
                disabled={remove.isPending}
                aria-label={t('web.providers.grokAccounts.removeAria', {
                  name: a.name,
                })}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function ToggleButton({
  enabled,
  pending,
  onToggle,
  ariaLabel,
}: {
  enabled: boolean
  pending: boolean
  onToggle: (next: boolean) => void
  ariaLabel: string
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={enabled}
      aria-label={ariaLabel}
      disabled={pending}
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        onToggle(!enabled)
      }}
      className={cn(
        'inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border border-border transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        'disabled:cursor-wait disabled:opacity-60',
        enabled ? 'bg-accent' : 'bg-muted',
      )}
    >
      <span
        className={cn(
          'pointer-events-none block size-4 rounded-full bg-background shadow-sm transition-transform',
          enabled ? 'translate-x-4' : 'translate-x-0.5',
        )}
      />
    </button>
  )
}
