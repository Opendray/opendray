import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Loader2, Play } from 'lucide-react'

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ProviderIcon } from '@/components/ProviderIcon'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import { listAntigravityAccounts } from '@/lib/antigravityAccounts'
import { runPlanStep, type PlanStep, type RoundTable } from '@/lib/roundtable'

// Pre-launch options for running one plan step — mirrors the handoff/spawn
// dialogs so a step spawns with the right account and (optionally) its bypass
// flag, instead of firing blind. The assignee is fixed (it's the step's).
const BYPASS_FLAGS: Record<string, string[]> = {
  claude: ['--dangerously-skip-permissions'],
  codex: ['--dangerously-bypass-approvals-and-sandbox'],
  antigravity: ['--dangerously-skip-permissions'],
  opencode: ['--dangerously-skip-permissions'],
  grok: ['--always-approve'],
}

const DEFAULT_ACCOUNT = '__default__'

interface Props {
  rt: RoundTable
  index: number
  step: PlanStep
  open: boolean
  onClose: () => void
  onLaunched: (sessionId: string) => void
}

export function RunStepDialog({
  rt,
  index,
  step,
  open,
  onClose,
  onLaunched,
}: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [accountId, setAccountId] = useState('')
  const [bypass, setBypass] = useState(false)

  const isClaude = step.assignee === 'claude'
  const isAgy = step.assignee === 'antigravity'
  const claudeAccounts = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    enabled: open && isClaude,
  })
  const agyAccounts = useQuery({
    queryKey: ['antigravity-accounts'],
    queryFn: listAntigravityAccounts,
    enabled: open && isAgy,
  })
  const accounts = (
    isClaude ? claudeAccounts.data : isAgy ? agyAccounts.data : []
  )
    ?.filter((a) => a.enabled)
    .map((a) => ({ id: a.id, label: a.display_name || a.name, usable: a.token_filled }))
  const showAccount = (isClaude || isAgy) && (accounts?.length ?? 0) > 0

  // Reset when the target step changes.
  useEffect(() => {
    setAccountId('')
    setBypass(false)
  }, [index, open])

  const run = useMutation({
    mutationFn: () =>
      runPlanStep(rt.id, index, {
        account_id: accountId || undefined,
        args: bypass ? BYPASS_FLAGS[step.assignee] : undefined,
      }),
    onSuccess: ({ session_id }) => {
      qc.invalidateQueries({ queryKey: ['round-table', rt.id] })
      onLaunched(session_id)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose()
      }}
    >
      <DialogContent className="max-w-sm">
        <DialogTitle className="flex items-center gap-2">
          <ProviderIcon providerId={step.assignee} size={18} />
          {t('web.roundTable.plan.runTitle')} · {index + 1}
        </DialogTitle>
        <p className="-mt-1 line-clamp-2 text-xs text-muted-foreground">
          {step.task}
        </p>

        <div className="mt-2 flex flex-col gap-4">
          {showAccount && (
            <div className="flex flex-col gap-1.5">
              <Label>{t('web.roundTable.plan.account')}</Label>
              <Select
                value={accountId === '' ? DEFAULT_ACCOUNT : accountId}
                onValueChange={(v) =>
                  setAccountId(v === DEFAULT_ACCOUNT ? '' : v)
                }
              >
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={DEFAULT_ACCOUNT}>
                    {t('web.roundTable.plan.accountDefault')}
                  </SelectItem>
                  {accounts?.map((a) => (
                    <SelectItem key={a.id} value={a.id} disabled={!a.usable}>
                      {a.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <label className="flex items-start gap-2">
            <Switch checked={bypass} onCheckedChange={setBypass} />
            <span className="flex flex-col">
              <span className="text-sm">{t('web.roundTable.plan.bypass')}</span>
              <span className="text-[11px] text-muted-foreground">
                {t('web.roundTable.plan.bypassHint')}
              </span>
            </span>
          </label>
        </div>

        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="accent"
            size="sm"
            disabled={run.isPending}
            onClick={() => run.mutate()}
          >
            {run.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Play className="size-3.5" />
            )}
            {t('web.roundTable.plan.runStep')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
