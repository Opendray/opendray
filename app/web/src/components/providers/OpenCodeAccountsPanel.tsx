import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CircleDot, Download, KeyRound, Loader2, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  createOpenCodeAccount,
  deleteOpenCodeAccount,
  importLocalOpenCodeAccount,
  listOpenCodeAccounts,
  toggleOpenCodeAccount,
} from '@/lib/opencodeAccounts'
import type { OpenCodeAccount } from '@/lib/types'

// OpenCodeAccountsPanel renders the multi-account list for the opencode
// provider. Unlike grok/antigravity there is no per-account home dir: an
// account is a named credential bundle (auth.json-shaped) that the gateway
// injects per spawn via OPENCODE_AUTH_CONTENT, while every account shares
// opencode's one session DB — which is why a live session can switch
// accounts and keep its conversation.
//
// Credentials are encrypted at rest and never come back to the browser:
// rows show only provider ids and a masked key hint. Accounts are added
// either from the host's current opencode login (Import local) or by
// entering a provider id + API key.

function relativeAgo(iso: string): string {
  const t = new Date(iso).getTime()
  if (!Number.isFinite(t)) return ''
  const dsec = Math.max(0, Math.floor((Date.now() - t) / 1000))
  if (dsec < 60) return `${dsec}s ago`
  if (dsec < 3600) return `${Math.floor(dsec / 60)}m ago`
  if (dsec < 86400) return `${Math.floor(dsec / 3600)}h ago`
  return `${Math.floor(dsec / 86400)}d ago`
}

const QK = ['opencode-accounts']

export function OpenCodeAccountsPanel() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: accounts, isLoading } = useQuery({
    queryKey: QK,
    queryFn: listOpenCodeAccounts,
    refetchInterval: 5000,
  })

  const [adding, setAdding] = useState(false)
  const [name, setName] = useState('')
  const [providerId, setProviderId] = useState('')
  const [apiKey, setApiKey] = useState('')

  const resetForm = () => {
    setName('')
    setProviderId('')
    setApiKey('')
    setAdding(false)
  }

  const create = useMutation({
    mutationFn: () =>
      createOpenCodeAccount({
        name: name.trim(),
        provider_id: providerId.trim(),
        api_key: apiKey.trim(),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK })
      toast.success(t('web.providers.opencodeAccounts.addedToast'))
      resetForm()
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.opencodeAccounts.addFailedToast'), {
        description: e.message,
      }),
  })

  const importLocal = useMutation({
    mutationFn: () => importLocalOpenCodeAccount(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK })
      toast.success(t('web.providers.opencodeAccounts.importedToast'))
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.opencodeAccounts.importFailedToast'), {
        description: e.message,
      }),
  })

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      toggleOpenCodeAccount(id, enabled),
    onMutate: async ({ id, enabled }) => {
      await qc.cancelQueries({ queryKey: QK })
      const prev = qc.getQueryData<OpenCodeAccount[]>(QK)
      if (prev) {
        qc.setQueryData<OpenCodeAccount[]>(
          QK,
          prev.map((a) => (a.id === id ? { ...a, enabled } : a)),
        )
      }
      return { prev }
    },
    onError: (e: Error, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(QK, ctx.prev)
      toast.error(t('web.providers.opencodeAccounts.toggleFailedToast'), {
        description: e.message,
      })
    },
    onSettled: () => qc.invalidateQueries({ queryKey: QK }),
  })

  const remove = useMutation({
    mutationFn: deleteOpenCodeAccount,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK })
      toast.success(t('web.providers.opencodeAccounts.removedToast'))
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.opencodeAccounts.removeFailedToast'), {
        description: e.message,
      }),
  })

  const canSubmit =
    name.trim() !== '' &&
    providerId.trim() !== '' &&
    apiKey.trim() !== '' &&
    !create.isPending

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-[12px] font-semibold uppercase tracking-wider text-muted-foreground/80">
            {t('web.providers.opencodeAccounts.title')}
          </h2>
          <span className="text-[10px] text-muted-foreground/60 font-mono">
            {accounts?.length ?? 0}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => importLocal.mutate()}
            disabled={importLocal.isPending}
            className="text-[11px] gap-1"
            title={t('web.providers.opencodeAccounts.importLocalTooltip')}
          >
            {importLocal.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Download className="size-3.5" />
            )}
            {t('web.providers.opencodeAccounts.importLocal')}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setAdding((v) => !v)}
            className="text-[11px] gap-1"
          >
            <Plus className="size-3.5" />
            {t('web.providers.opencodeAccounts.add')}
          </Button>
        </div>
      </div>

      <div className="rounded-md border border-border bg-muted/20 px-3 py-2.5 text-[11px] text-muted-foreground leading-relaxed">
        {t('web.providers.opencodeAccounts.explainer')}
      </div>

      {adding && (
        <form
          className="rounded-md border border-border px-3 py-2.5 space-y-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (canSubmit) create.mutate()
          }}
        >
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('web.providers.opencodeAccounts.namePlaceholder')}
            className="h-8 text-[12px]"
            autoComplete="off"
          />
          <Input
            value={providerId}
            onChange={(e) => setProviderId(e.target.value)}
            placeholder={t('web.providers.opencodeAccounts.providerPlaceholder')}
            className="h-8 text-[12px] font-mono"
            autoComplete="off"
          />
          <Input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={t('web.providers.opencodeAccounts.apiKeyPlaceholder')}
            className="h-8 text-[12px] font-mono"
            autoComplete="new-password"
          />
          <div className="flex justify-end gap-1">
            <Button type="button" variant="ghost" size="sm" onClick={resetForm}>
              {t('web.providers.opencodeAccounts.cancel')}
            </Button>
            <Button type="submit" size="sm" disabled={!canSubmit}>
              {create.isPending && <Loader2 className="size-3.5 animate-spin" />}
              {t('web.providers.opencodeAccounts.save')}
            </Button>
          </div>
        </form>
      )}

      {isLoading && (
        <div className="text-[12px] text-muted-foreground italic">
          {t('web.providers.opencodeAccounts.loading')}
        </div>
      )}

      {!isLoading && (accounts?.length ?? 0) === 0 && (
        <p className="text-[12px] text-muted-foreground italic">
          {t('web.providers.opencodeAccounts.empty')}
        </p>
      )}

      <div className="space-y-1.5">
        {(accounts ?? []).map((a) => (
          <div key={a.id} className="rounded-md border border-border px-3 py-2.5">
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
                      {t('web.providers.opencodeAccounts.unusable')}
                    </span>
                  )}
                  <span className="text-[10px] rounded px-1.5 py-0.5 bg-foreground/5 text-muted-foreground/80">
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
                  {(a.credentials ?? []).length > 0
                    ? a.credentials
                        .map((c) =>
                          c.hint ? `${c.provider} ${c.hint}` : `${c.provider} (${c.type})`,
                        )
                        .join(' · ')
                    : (a.providers ?? []).join(' · ')}
                </div>
              </div>
              <ToggleButton
                enabled={a.enabled}
                pending={toggle.isPending}
                onToggle={(v) => toggle.mutate({ id: a.id, enabled: v })}
                ariaLabel={t('web.providers.opencodeAccounts.toggleAria', {
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
                      t('web.providers.opencodeAccounts.removeConfirm', {
                        name: a.name,
                      }),
                    )
                  ) {
                    remove.mutate(a.id)
                  }
                }}
                disabled={remove.isPending}
                aria-label={t('web.providers.opencodeAccounts.removeAria', {
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
