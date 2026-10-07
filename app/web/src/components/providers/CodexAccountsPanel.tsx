import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CircleDot,
  Download,
  KeyRound,
  Loader2,
  Trash2,
  UserRound,
} from 'lucide-react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  deleteCodexAccount,
  createCodexAccount,
  importLocalCodexAccounts,
  listCodexAccounts,
  setCodexAccountAPIKey,
  toggleCodexAccount,
} from '@/lib/codexAccounts'
import type { CodexAccount } from '@/lib/types'

// CodexAccountsPanel renders the multi-account list for the OpenAI Codex
// (codex) provider. codex keys its credential + config state off
// CODEX_HOME, so an account is a dedicated CODEX_HOME dir holding its own
// auth.json login. Two ways in:
//
//   - ChatGPT sign-in: the operator runs
//     `CODEX_HOME=~/.codex-accounts/<name> codex login --device-auth` on the
//     gateway host, then clicks Import local to surface the row.
//   - API key: the inline form creates the row and hands the key to
//     `codex login --with-api-key` on the gateway (never stored by opendray).
//
// Mirrors GrokAccountsPanel. Never copy one account's auth.json into
// another's home — codex rotates refresh tokens, so they'd log each other out.

function relativeAgo(iso: string): string {
  const t = new Date(iso).getTime()
  if (!Number.isFinite(t)) return ''
  const dsec = Math.max(0, Math.floor((Date.now() - t) / 1000))
  if (dsec < 60) return `${dsec}s ago`
  if (dsec < 3600) return `${Math.floor(dsec / 60)}m ago`
  if (dsec < 86400) return `${Math.floor(dsec / 3600)}h ago`
  return `${Math.floor(dsec / 86400)}d ago`
}

export function CodexAccountsPanel() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: accounts, isLoading } = useQuery({
    queryKey: ['codex-accounts'],
    queryFn: listCodexAccounts,
    refetchInterval: 5000,
  })

  const importLocal = useMutation({
    mutationFn: importLocalCodexAccounts,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['codex-accounts'] })
      if (res.count === 0) {
        toast.success(t('web.providers.codexAccounts.importedNothingToast'))
      } else {
        toast.success(
          t('web.providers.codexAccounts.importedToast', { count: res.count }),
        )
      }
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.codexAccounts.importFailedToast'), {
        description: e.message,
      }),
  })

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      toggleCodexAccount(id, enabled),
    onMutate: async ({ id, enabled }) => {
      await qc.cancelQueries({ queryKey: ['codex-accounts'] })
      const prev = qc.getQueryData<CodexAccount[]>(['codex-accounts'])
      if (prev) {
        qc.setQueryData<CodexAccount[]>(
          ['codex-accounts'],
          prev.map((a) => (a.id === id ? { ...a, enabled } : a)),
        )
      }
      return { prev }
    },
    onError: (e: Error, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(['codex-accounts'], ctx.prev)
      toast.error(t('web.providers.codexAccounts.toggleFailedToast'), {
        description: e.message,
      })
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ['codex-accounts'] }),
  })

  // API-key sign-in: create the row, then let the gateway run
  // `codex login --with-api-key` into the account's CODEX_HOME.
  const [newName, setNewName] = useState('')
  const [newKey, setNewKey] = useState('')
  const addWithKey = useMutation({
    mutationFn: async ({ name, key }: { name: string; key: string }) => {
      const acc = await createCodexAccount({ name })
      return setCodexAccountAPIKey(acc.id, key)
    },
    onSuccess: () => {
      setNewName('')
      setNewKey('')
      qc.invalidateQueries({ queryKey: ['codex-accounts'] })
      toast.success(t('web.providers.codexAccounts.apiKeyAddedToast'))
    },
    onError: (e: Error) => {
      qc.invalidateQueries({ queryKey: ['codex-accounts'] })
      toast.error(t('web.providers.codexAccounts.apiKeyFailedToast'), {
        description: e.message,
      })
    },
  })

  const remove = useMutation({
    mutationFn: deleteCodexAccount,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['codex-accounts'] })
      toast.success(t('web.providers.codexAccounts.removedToast'))
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.codexAccounts.removeFailedToast'), {
        description: e.message,
      }),
  })

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-[12px] font-semibold uppercase tracking-wider text-muted-foreground/80">
            {t('web.providers.codexAccounts.title')}
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
          title={t('web.providers.codexAccounts.importLocalTooltip')}
        >
          {importLocal.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Download className="size-3.5" />
          )}
          {t('web.providers.codexAccounts.importLocal')}
        </Button>
      </div>

      <div className="rounded-md border border-border bg-muted/20 px-3 py-2.5 text-[11px] text-muted-foreground leading-relaxed">
        <span className="font-medium text-foreground">
          {t('web.providers.codexAccounts.addingTitle')}
        </span>{' '}
        {t('web.providers.codexAccounts.addingBodyPrefix')}
        <pre className="mt-1.5 mb-1.5 px-2 py-1.5 rounded bg-background/60 text-[10.5px] overflow-x-auto">
{`mkdir -p ~/.codex-accounts/<name>
CODEX_HOME=~/.codex-accounts/<name> codex login --device-auth   # complete ChatGPT sign-in`}
        </pre>
        <Trans
          i18nKey="web.providers.codexAccounts.addingBodySuffix"
          components={{ 1: <span className="font-mono" /> }}
        />
      </div>

      <form
        className="flex flex-wrap items-center gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          const name = newName.trim()
          const key = newKey.trim()
          if (!name || !key) return
          addWithKey.mutate({ name, key })
        }}
      >
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder={t('web.providers.codexAccounts.apiKeyNamePlaceholder')}
          className="h-7 w-36 text-[12px]"
          aria-label={t('web.providers.codexAccounts.apiKeyNamePlaceholder')}
        />
        <Input
          type="password"
          autoComplete="off"
          value={newKey}
          onChange={(e) => setNewKey(e.target.value)}
          placeholder={t('web.providers.codexAccounts.apiKeyPlaceholder')}
          className="h-7 flex-1 min-w-[12rem] text-[12px] font-mono"
          aria-label={t('web.providers.codexAccounts.apiKeyPlaceholder')}
        />
        <Button
          type="submit"
          variant="outline"
          size="sm"
          className="h-7 text-[11px] gap-1"
          disabled={addWithKey.isPending || !newName.trim() || !newKey.trim()}
        >
          {addWithKey.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <KeyRound className="size-3.5" />
          )}
          {t('web.providers.codexAccounts.apiKeyAdd')}
        </Button>
      </form>

      {isLoading && (
        <div className="text-[12px] text-muted-foreground italic">
          {t('web.providers.codexAccounts.loading')}
        </div>
      )}

      {!isLoading && (accounts?.length ?? 0) === 0 && (
        <p className="text-[12px] text-muted-foreground italic">
          <Trans
            i18nKey="web.providers.codexAccounts.empty"
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
                      {t('web.providers.codexAccounts.noTokenYet')}
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
                {a.oauth_email && (
                  <div className="mt-0.5 flex items-center gap-1 text-[10px] text-muted-foreground/80 truncate">
                    <UserRound className="size-2.5 shrink-0" />
                    <span className="truncate">
                      {a.oauth_name
                        ? `${a.oauth_name} · ${a.oauth_email}`
                        : a.oauth_email}
                    </span>
                  </div>
                )}
                <div className="text-[10px] font-mono text-muted-foreground/70 truncate">
                  {t('web.providers.codexAccounts.homeDir')}{' '}
                  {a.config_dir || 'default'}
                </div>
              </div>
              <ToggleButton
                enabled={a.enabled}
                pending={toggle.isPending}
                onToggle={(v) => toggle.mutate({ id: a.id, enabled: v })}
                ariaLabel={t('web.providers.codexAccounts.toggleAria', {
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
                      t('web.providers.codexAccounts.removeConfirm', {
                        name: a.name,
                      }),
                    )
                  ) {
                    remove.mutate(a.id)
                  }
                }}
                disabled={remove.isPending}
                aria-label={t('web.providers.codexAccounts.removeAria', {
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
