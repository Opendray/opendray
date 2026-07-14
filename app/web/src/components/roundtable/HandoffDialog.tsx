import { useEffect, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { FolderOpen } from 'lucide-react'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ProviderIcon } from '@/components/ProviderIcon'
import { FileBrowserDialog } from '@/components/sessions/FileBrowserDialog'
import { listProviders } from '@/lib/catalog'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import { cn } from '@/lib/utils'
import { handoffRoundTable, type RoundTable } from '@/lib/roundtable'

// Per-provider bypass / autonomy flag — mirrors SpawnDialog so the handoff
// session can start in skip-permissions / YOLO mode just like a normal
// hand-created session.
const BYPASS_FLAGS: Record<string, string[]> = {
  claude: ['--dangerously-skip-permissions'],
  codex: ['--dangerously-bypass-approvals-and-sandbox'],
  antigravity: ['--dangerously-skip-permissions'],
  opencode: ['--dangerously-skip-permissions'],
}

interface Props {
  rt: RoundTable
  open: boolean
  onClose: () => void
  onDone: (sessionId: string) => void
}

export function HandoffDialog({ rt, open, onClose, onDone }: Props) {
  const { t } = useTranslation()
  const hasPrior = !!rt.resulting_session_id

  // Any available agent — the executor need not be a discussion member
  // (handoff spawns a normal session, like session creation).
  const { data: providers } = useQuery({
    queryKey: ['providers'],
    queryFn: listProviders,
    enabled: open,
  })

  // Continue in the prior execution session by default when one exists.
  const [continueExisting, setContinueExisting] = useState(hasPrior)
  const [providerId, setProviderId] = useState('')
  const [accountId, setAccountId] = useState('')
  const [cwd, setCwd] = useState(rt.cwd ?? '')
  const [bypassEnabled, setBypassEnabled] = useState(false)
  const [browserOpen, setBrowserOpen] = useState(false)

  useEffect(() => {
    setContinueExisting(hasPrior)
  }, [hasPrior, open])

  // Default to the first enabled provider once the list loads.
  useEffect(() => {
    if (open && providers && !providerId) {
      const first = providers.find((p) => p.enabled) ?? providers[0]
      if (first) setProviderId(first.manifest.id)
    }
  }, [open, providers, providerId])

  const isClaude = providerId === 'claude'
  const { data: claudeAccounts } = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    enabled: open && isClaude && !continueExisting,
  })
  const accounts = (claudeAccounts ?? []).filter((a) => a.enabled)

  // Each provider's bypass flag differs, so reset the toggle when it changes.
  useEffect(() => {
    setAccountId('')
    setBypassEnabled(false)
  }, [providerId])

  const bypassFlags =
    !continueExisting && bypassEnabled ? BYPASS_FLAGS[providerId] : undefined

  const run = useMutation({
    mutationFn: () =>
      handoffRoundTable(rt.id, {
        provider: providerId,
        cwd: cwd.trim(),
        account_id: isClaude && accountId ? accountId : undefined,
        force_new: !continueExisting,
        args: bypassFlags,
      }),
    onSuccess: ({ session_id }) => {
      toast.success(t('web.roundTable.handoff.started'))
      onDone(session_id)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  // Continuing needs nothing else; a new session needs a provider + cwd.
  const canRun = continueExisting || (providerId !== '' && cwd.trim() !== '')

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogTitle>{t('web.roundTable.handoff.title')}</DialogTitle>
        <p className="text-xs text-muted-foreground -mt-1">
          {t('web.roundTable.handoff.description')}
        </p>

        <div className="mt-2 flex flex-col gap-4">
          {hasPrior && (
            <label className="flex items-start gap-2 rounded-md border border-border bg-card/40 px-3 py-2 text-[12px] cursor-pointer select-none">
              <input
                type="checkbox"
                checked={continueExisting}
                onChange={(e) => setContinueExisting(e.target.checked)}
                className="mt-0.5"
              />
              <span className="flex-1">
                <span className="font-medium">
                  {t('web.roundTable.handoff.continueLabel')}
                </span>
                <span className="mt-0.5 block text-[11px] text-muted-foreground">
                  {t('web.roundTable.handoff.continueHint')}
                </span>
              </span>
            </label>
          )}

          {!continueExisting && (
            <>
              <div className="flex flex-col gap-1.5">
                <Label>{t('web.roundTable.handoff.executor')}</Label>
                <div className="grid grid-cols-2 gap-2">
                  {(providers ?? []).map((p) => {
                    const active = providerId === p.manifest.id
                    return (
                      <button
                        key={p.manifest.id}
                        type="button"
                        onClick={() => setProviderId(p.manifest.id)}
                        disabled={!p.enabled}
                        className={cn(
                          'flex items-center gap-2 rounded-md border px-2 py-2 text-left transition-colors disabled:opacity-50',
                          active
                            ? 'border-foreground/30 bg-card'
                            : 'border-border hover:border-foreground/20 hover:bg-card',
                        )}
                      >
                        <ProviderIcon
                          providerId={p.manifest.id}
                          fallbackLetter={p.manifest.displayName?.charAt(0) ?? '?'}
                          size={28}
                          title={p.manifest.displayName}
                        />
                        <span className="truncate text-[12px] font-medium">
                          {p.manifest.displayName}
                        </span>
                      </button>
                    )
                  })}
                </div>
                <span className="text-[11px] text-muted-foreground">
                  {t('web.roundTable.handoff.executorHint')}
                </span>
              </div>

              {isClaude && accounts.length > 0 && (
                <div className="flex flex-col gap-1.5">
                  <Label>{t('web.roundTable.handoff.claudeAccount')}</Label>
                  <div className="flex flex-wrap gap-1.5">
                    <button
                      type="button"
                      onClick={() => setAccountId('')}
                      className={cn(
                        'rounded-md border px-2 py-1 text-[11px] transition-colors',
                        accountId === ''
                          ? 'border-foreground/30 bg-card'
                          : 'border-border hover:bg-card',
                      )}
                    >
                      {t('web.roundTable.handoff.accountDefault')}
                    </button>
                    {accounts.map((a) => (
                      <button
                        key={a.id}
                        type="button"
                        onClick={() => setAccountId(a.id)}
                        disabled={!a.token_filled}
                        className={cn(
                          'rounded-md border px-2 py-1 text-[11px] transition-colors disabled:opacity-50',
                          accountId === a.id
                            ? 'border-foreground/30 bg-card'
                            : 'border-border hover:bg-card',
                        )}
                      >
                        {a.display_name || a.name}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="handoff-cwd">
                  {t('web.roundTable.handoff.project')}
                </Label>
                <div className="flex gap-1.5">
                  <Input
                    id="handoff-cwd"
                    value={cwd}
                    onChange={(e) => setCwd(e.target.value)}
                    placeholder={t('web.roundTable.handoff.projectPlaceholder')}
                    className="flex-1 font-mono text-xs"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setBrowserOpen(true)}
                    className="shrink-0 gap-1"
                  >
                    <FolderOpen className="size-3.5" />
                    {t('web.roundTable.dialog.browse')}
                  </Button>
                </div>
                <span className="text-[11px] text-muted-foreground">
                  {t('web.roundTable.handoff.projectHint')}
                </span>
              </div>

              {BYPASS_FLAGS[providerId] && (
                <label className="flex items-start gap-2 rounded-md border border-border bg-card/40 px-3 py-2 text-[12px] cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={bypassEnabled}
                    onChange={(e) => setBypassEnabled(e.target.checked)}
                    className="mt-0.5"
                  />
                  <span className="flex-1">
                    <span className="font-medium">
                      {t('web.roundTable.handoff.bypassLabel')}
                    </span>
                    <span className="mt-0.5 block text-[11px] text-muted-foreground">
                      {t('web.roundTable.handoff.bypassHint')}
                    </span>
                  </span>
                </label>
              )}
            </>
          )}
        </div>

        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            size="sm"
            disabled={!canRun || run.isPending}
            onClick={() => run.mutate()}
          >
            {continueExisting
              ? t('web.roundTable.handoff.runContinue')
              : t('web.roundTable.handoff.run')}
          </Button>
        </div>

        <FileBrowserDialog
          open={browserOpen}
          onOpenChange={setBrowserOpen}
          initialPath={cwd.trim() || undefined}
          onSelect={(p) => setCwd(p)}
        />
      </DialogContent>
    </Dialog>
  )
}
