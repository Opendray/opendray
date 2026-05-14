import { useState, useEffect, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FolderOpen, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ProviderIcon } from '@/components/ProviderIcon'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { createSession } from '@/lib/sessions'
import { listProviders } from '@/lib/catalog'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import type { Session } from '@/lib/types'

interface SpawnDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSpawned: (session: Session) => void
  defaultCwd?: string
}

const HOME_HINT = '/Users/' // macOS-friendly default; user can edit.

// Per-provider "bypass" / autonomy flag set. The provider config
// (Settings → Providers) may also bake these in; the per-session
// toggle below is purely additive. Keeping the list here as a const
// table avoids importing config logic from the gateway side.
const BYPASS_FLAGS: Record<string, string[]> = {
  claude: ['--dangerously-skip-permissions'],
  codex: ['--ask-for-approval', 'never', '-c', 'approval_policy="never"'],
  gemini: ['--yolo'],
}

// i18n key suffix per provider for the bypass toggle label. Different
// CLIs use different vocabulary; pretending they're all "Auto-approve"
// would confuse operators who only know their tool's term.
const BYPASS_LABEL_KEY: Record<string, string> = {
  claude: 'web.sessions.spawn.bypassClaude',
  codex: 'web.sessions.spawn.bypassCodex',
  gemini: 'web.sessions.spawn.bypassGemini',
}

export function SpawnDialog({
  open,
  onOpenChange,
  onSpawned,
  defaultCwd,
}: SpawnDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: providers } = useQuery({
    queryKey: ['providers'],
    queryFn: listProviders,
    enabled: open,
  })

  const [providerId, setProviderId] = useState<string>('')
  const [accountId, setAccountId] = useState<string>('')
  const [name, setName] = useState('')
  const [cwd, setCwd] = useState(defaultCwd ?? HOME_HINT)
  const [argsText, setArgsText] = useState('')
  // Per-session bypass toggle (Claude --dangerously-skip-permissions,
  // Codex --ask-for-approval never, Gemini --yolo). Defaults OFF;
  // operators opt in per spawn. Independent of the provider's own
  // bypass config — additive only.
  const [bypassEnabled, setBypassEnabled] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [browserOpen, setBrowserOpen] = useState(false)

  // Default to first enabled provider when list loads.
  useEffect(() => {
    if (open && providers && !providerId) {
      const first = providers.find((p) => p.enabled) ?? providers[0]
      if (first) setProviderId(first.manifest.id)
    }
  }, [open, providers, providerId])

  const isClaude = providerId === 'claude'
  const { data: claudeAccounts, isLoading: claudeAccountsLoading } = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    enabled: open && isClaude,
  })
  const accounts = (claudeAccounts ?? []).filter((a) => a.enabled)
  // Multi-account mode (≥2 enabled): no "Default" button — operator
  // must pick one. Single-account mode keeps Default for parity
  // with the pre-PR-54 behaviour.
  const multiAccount = accounts.length >= 2

  // When provider changes, clear account selection so we don't keep
  // a stale id from a different provider. Also reset the bypass
  // toggle — each provider's flag is different, so a stale "ON"
  // would carry semantics that don't apply.
  useEffect(() => {
    setAccountId('')
    setBypassEnabled(false)
  }, [providerId])

  // Multi-account auto-pick: when 2+ accounts are configured and
  // nothing is selected yet, force-select the first one. Avoids
  // the dialog ever submitting with an empty (== "Default")
  // account id when Default isn't shown to the operator.
  useEffect(() => {
    if (multiAccount && !accountId && accounts.length > 0) {
      setAccountId(accounts[0].id)
    }
  }, [multiAccount, accountId, accounts])

  const mutation = useMutation({
    mutationFn: createSession,
    onSuccess: (session) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      toast.success(t('web.sessions.spawn.spawnedToast'), {
        description: t('web.sessions.spawn.spawnedDescription', {
          provider: session.provider_id,
          pid: session.pid ?? t('web.sessions.spawn.pidFallback'),
        }),
      })
      onSpawned(session)
      onOpenChange(false)
      // Reset for next spawn. Bypass intentionally resets too —
      // each new session should be a deliberate opt-in.
      setName('')
      setArgsText('')
      setBypassEnabled(false)
      setError(null)
    },
    onError: (err: Error) => {
      setError(err.message)
    },
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (!providerId) {
      setError(t('web.sessions.spawn.errorPickProvider'))
      return
    }
    if (!cwd.trim()) {
      setError(t('web.sessions.spawn.errorCwdRequired'))
      return
    }
    const userArgs = argsText
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s.length > 0)
    // Bypass flags come first so the operator's explicit Extra args
    // can still override (codex's --ask-for-approval is last-wins
    // in the upstream parser).
    const bypassFlags = bypassEnabled
      ? BYPASS_FLAGS[providerId] ?? []
      : []
    const args = [...bypassFlags, ...userArgs]
    mutation.mutate({
      provider_id: providerId,
      cwd: cwd.trim(),
      name: name.trim() || undefined,
      args: args.length > 0 ? args : undefined,
      claude_account_id: isClaude && accountId ? accountId : undefined,
    })
  }

  const bypassLabelKey = BYPASS_LABEL_KEY[providerId]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[480px]">
        <DialogHeader>
          <DialogTitle>{t('web.sessions.spawn.title')}</DialogTitle>
          <DialogDescription>
            {t('web.sessions.spawn.description')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-4 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="provider">{t('web.sessions.spawn.provider')}</Label>
            <div className="grid grid-cols-2 gap-2">
              {(providers ?? []).map((p) => {
                const active = providerId === p.manifest.id
                return (
                  <button
                    key={p.manifest.id}
                    type="button"
                    onClick={() => setProviderId(p.manifest.id)}
                    disabled={!p.enabled}
                    className={`flex items-center gap-2 px-2 py-2 rounded-md border text-left transition-colors disabled:opacity-50 ${
                      active
                        ? 'border-foreground/30 bg-card'
                        : 'border-border hover:bg-card hover:border-foreground/20'
                    }`}
                  >
                    <ProviderIcon
                      providerId={p.manifest.id}
                      fallbackLetter={p.manifest.displayName?.charAt(0) ?? '?'}
                      size={32}
                      title={p.manifest.displayName}
                    />
                    <div className="flex flex-col min-w-0">
                      <span className="text-[12px] font-medium truncate">
                        {p.manifest.displayName}
                      </span>
                      <span className="text-[10px] text-muted-foreground font-mono truncate">
                        {p.manifest.executable}
                      </span>
                    </div>
                  </button>
                )
              })}
            </div>
          </div>

          {isClaude && (
            <div className="space-y-1.5">
              <Label>{t('web.sessions.spawn.claudeAccount')}</Label>
              {claudeAccountsLoading && accounts.length === 0 ? (
                <div className="text-[11px] text-muted-foreground">
                  {t('web.sessions.spawn.loadingAccounts')}
                </div>
              ) : accounts.length === 0 ? (
                <div className="text-[11px] text-muted-foreground">
                  {t('web.sessions.spawn.noAccounts')}
                </div>
              ) : (
                <div className="flex flex-wrap gap-1.5">
                  {!multiAccount && (
                    <button
                      type="button"
                      onClick={() => setAccountId('')}
                      className={`px-2 py-1 rounded-md border text-[11px] transition-colors ${
                        accountId === ''
                          ? 'border-foreground/30 bg-card'
                          : 'border-border hover:bg-card hover:border-foreground/20'
                      }`}
                      title={t('web.sessions.spawn.defaultTooltip')}
                    >
                      {t('web.sessions.spawn.default')}
                    </button>
                  )}
                  {accounts.map((a) => {
                    const active = accountId === a.id
                    return (
                      <button
                        key={a.id}
                        type="button"
                        onClick={() => setAccountId(a.id)}
                        disabled={!a.token_filled}
                        className={`px-2 py-1 rounded-md border text-[11px] transition-colors disabled:opacity-50 ${
                          active
                            ? 'border-foreground/30 bg-card'
                            : 'border-border hover:bg-card hover:border-foreground/20'
                        }`}
                        title={
                          a.token_filled
                            ? `${a.config_dir || a.name}`
                            : t('web.sessions.spawn.tokenMissingTooltip')
                        }
                      >
                        {a.display_name || a.name}
                        {!a.token_filled && (
                          <span className="ml-1 text-amber-500/90">
                            {t('web.sessions.spawn.tokenEmptyBadge')}
                          </span>
                        )}
                      </button>
                    )
                  })}
                </div>
              )}
              {multiAccount && (
                <div className="text-[11px] text-muted-foreground">
                  {t('web.sessions.spawn.multiAccountHint')}
                </div>
              )}
            </div>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="cwd">{t('web.sessions.spawn.cwd')}</Label>
            <div className="flex gap-1.5">
              <Input
                id="cwd"
                value={cwd}
                onChange={(e) => setCwd(e.target.value)}
                placeholder={t('web.sessions.spawn.cwdPlaceholder')}
                required
                autoFocus
                className="flex-1"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setBrowserOpen(true)}
                className="shrink-0 gap-1"
              >
                <FolderOpen className="size-3.5" />
                {t('web.sessions.spawn.browse')}
              </Button>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="name">{t('web.sessions.spawn.nameLabel')}</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('web.sessions.spawn.namePlaceholder')}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="args">{t('web.sessions.spawn.argsLabel')}</Label>
            <textarea
              id="args"
              rows={3}
              value={argsText}
              onChange={(e) => setArgsText(e.target.value)}
              placeholder={`-c\necho hello`}
              className="w-full font-mono text-[12px] rounded-md border border-border bg-input/40 px-3 py-2 text-foreground transition-colors placeholder:text-muted-foreground/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
            />
          </div>

          {bypassLabelKey && (
            <label className="flex items-start gap-2 text-[12px] cursor-pointer select-none">
              <input
                type="checkbox"
                checked={bypassEnabled}
                onChange={(e) => setBypassEnabled(e.target.checked)}
                className="mt-0.5"
              />
              <span className="flex-1">
                <span className="font-medium">{t(bypassLabelKey)}</span>
                <span className="block text-muted-foreground mt-0.5 text-[11px]">
                  {bypassEnabled
                    ? t('web.sessions.spawn.bypassOnHint')
                    : t('web.sessions.spawn.bypassOffHint')}
                </span>
              </span>
            </label>
          )}

          {error && (
            <div className="text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2">
              {error}
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={mutation.isPending}
            >
              {t('web.sessions.spawn.cancel')}
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={mutation.isPending}
            >
              {mutation.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              {mutation.isPending
                ? t('web.sessions.spawn.submitting')
                : t('web.sessions.spawn.submit')}
            </Button>
          </DialogFooter>
        </form>
        <FileBrowserDialog
          open={browserOpen}
          onOpenChange={setBrowserOpen}
          initialPath={cwd && cwd !== HOME_HINT ? cwd : undefined}
          onSelect={(p) => setCwd(p)}
        />
      </DialogContent>
    </Dialog>
  )
}
