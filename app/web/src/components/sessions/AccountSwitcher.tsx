import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, ChevronDown, Loader2, UserRound } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from '@/components/ui/dropdown-menu'
import { listClaudeAccounts } from '@/lib/claudeAccounts'
import { listAntigravityAccounts } from '@/lib/antigravityAccounts'
import { listGrokAccounts } from '@/lib/grokAccounts'
import {
  switchClaudeAccount,
  switchAntigravityAccount,
  switchGrokAccount,
} from '@/lib/sessions'
import { cn } from '@/lib/utils'
import type { Session } from '@/lib/types'

interface AccountSwitcherProps {
  session: Session
}

// Minimal shape shared by Claude, Antigravity, and Grok accounts, the
// only fields this dropdown renders. Lets one component drive every
// provider's multi-account switching.
interface SwitcherAccount {
  id: string
  name: string
  display_name: string
  config_dir: string
  enabled: boolean
  token_filled: boolean
}

// AccountSwitcher renders a header dropdown that lets the user rebind a
// *running* multi-account session (claude, antigravity, or grok) to a
// different account. The backend terminates the current child process and
// respawns it under the new credential, so the in-CLI conversation is lost
// (the process is replaced) and the dropdown confirms before firing.
//
// Claude isolates accounts via CLAUDE_CONFIG_DIR and supports carrying a
// recap across the switch (the carry toggle). Antigravity (HOME) and Grok
// (GROK_HOME) have no cross-account recap builder yet, so their switch is
// always clean-slate and the carry toggle is hidden.
export function AccountSwitcher({ session }: AccountSwitcherProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const kind: 'claude' | 'antigravity' | 'grok' =
    session.provider_id === 'antigravity'
      ? 'antigravity'
      : session.provider_id === 'grok'
        ? 'grok'
        : 'claude'
  // Claude and Grok both carry a recap across the switch (Claude via
  // --append-system-prompt, Grok via --rules); Antigravity carries the
  // whole conversation with no toggle. So the carry toggle shows for the
  // two recap providers.
  const supportsCarry = kind === 'claude' || kind === 'grok'

  const queryKey =
    kind === 'antigravity'
      ? ['antigravity-accounts']
      : kind === 'grok'
        ? ['grok-accounts']
        : ['claude-accounts']
  const queryFn =
    kind === 'antigravity'
      ? listAntigravityAccounts
      : kind === 'grok'
        ? listGrokAccounts
        : listClaudeAccounts

  const { data: accounts } = useQuery<SwitcherAccount[]>({
    queryKey,
    queryFn,
    staleTime: 30_000,
  })
  const currentId =
    kind === 'antigravity'
      ? session.antigravity_account_id
      : kind === 'grok'
        ? session.grok_account_id
        : session.claude_account_id
  const enabled = (accounts ?? []).filter((a) => a.enabled)
  const current = (accounts ?? []).find((a) => a.id === currentId)
  const currentLabel = currentId
    ? current?.display_name || current?.name || currentId
    : t('web.sessions.accountSwitcher.currentDefault')

  // Carry-over toggle (claude only). When on, the switch seeds the new
  // account's fresh session with a recap of the prior conversation.
  const [carryContext, setCarryContext] = useState(true)

  const mutation = useMutation({
    mutationFn: (accountId: string) =>
      kind === 'antigravity'
        ? switchAntigravityAccount(session.id, accountId)
        : kind === 'grok'
          ? switchGrokAccount(session.id, accountId, carryContext)
          : switchClaudeAccount(session.id, accountId, carryContext),
    onSuccess: (next) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      const nextId =
        kind === 'antigravity'
          ? next.antigravity_account_id
          : kind === 'grok'
            ? next.grok_account_id
            : next.claude_account_id
      const account = nextId
        ? enabled.find((a) => a.id === nextId)?.display_name || nextId
        : t('web.sessions.accountSwitcher.switchedDefault')
      toast.success(t('web.sessions.accountSwitcher.switchedToast'), {
        description: t('web.sessions.accountSwitcher.switchedDescription', {
          account,
          pid: next.pid ?? 'unknown',
        }),
      })
    },
    onError: (err: Error) =>
      toast.error(t('web.sessions.accountSwitcher.switchFailedToast'), {
        description: err.message,
      }),
  })

  const pick = (accountId: string) => {
    if (accountId === (currentId ?? '')) return
    const msg =
      kind === 'antigravity'
        ? t('web.sessions.accountSwitcher.confirmSwitchAgy')
        : kind === 'grok'
          ? carryContext
            ? t('web.sessions.accountSwitcher.confirmSwitchGrokCarry')
            : t('web.sessions.accountSwitcher.confirmSwitchGrok')
          : carryContext
            ? t('web.sessions.accountSwitcher.confirmSwitchCarry')
            : t('web.sessions.accountSwitcher.confirmSwitch')
    if (!confirm(msg)) {
      return
    }
    mutation.mutate(accountId)
  }

  const tooltipKey =
    kind === 'antigravity'
      ? 'web.sessions.accountSwitcher.tooltipAgy'
      : kind === 'grok'
        ? 'web.sessions.accountSwitcher.tooltipGrok'
        : 'web.sessions.accountSwitcher.tooltip'
  const menuTitleKey =
    kind === 'antigravity'
      ? 'web.sessions.accountSwitcher.menuTitleAgy'
      : kind === 'grok'
        ? 'web.sessions.accountSwitcher.menuTitleGrok'
        : 'web.sessions.accountSwitcher.menuTitle'

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          disabled={mutation.isPending}
          className="text-[11px] gap-1 hover:text-foreground"
          title={t(tooltipKey)}
        >
          {mutation.isPending ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <UserRound className="size-3" />
          )}
          <span className="font-mono">@{currentLabel}</span>
          <ChevronDown className="size-3 opacity-60" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-[220px]">
        <DropdownMenuLabel className="text-[10px] uppercase tracking-wider text-muted-foreground/70">
          {t(menuTitleKey)}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {/* Carry-over toggle (claude only). Stays open on click
            (preventDefault) so the operator sets it before picking a
            destination. The subtitle is the consent surface for the
            cross-account data flow. */}
        {supportsCarry && (
          <>
            <DropdownMenuItem
              onSelect={(e) => {
                e.preventDefault()
                setCarryContext((v) => !v)
              }}
              className="gap-2"
            >
              <Check
                className={cn(
                  'size-3 shrink-0',
                  carryContext ? 'opacity-100' : 'opacity-0',
                )}
              />
              <div className="flex flex-col flex-1 min-w-0">
                <span className="text-[12px]">
                  {t('web.sessions.accountSwitcher.carryContext')}
                </span>
                <span className="text-[10px] text-muted-foreground whitespace-normal">
                  {t('web.sessions.accountSwitcher.carryContextHelp')}
                </span>
              </div>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
          </>
        )}
        <DropdownMenuItem
          onSelect={(e) => {
            e.preventDefault()
            pick('')
          }}
          className="gap-2"
        >
          <Check
            className={cn(
              'size-3 shrink-0',
              currentId ? 'opacity-0' : 'opacity-100',
            )}
          />
          <div className="flex flex-col flex-1 min-w-0">
            <span className="text-[12px]">
              {t('web.sessions.accountSwitcher.defaultName')}
            </span>
            <span className="text-[10px] text-muted-foreground">
              {t('web.sessions.accountSwitcher.defaultSubtitle')}
            </span>
          </div>
        </DropdownMenuItem>
        {enabled.length > 0 && <DropdownMenuSeparator />}
        {enabled.map((a) => {
          const active = currentId === a.id
          return (
            <DropdownMenuItem
              key={a.id}
              disabled={!a.token_filled}
              onSelect={(e) => {
                e.preventDefault()
                pick(a.id)
              }}
              className="gap-2"
            >
              <Check
                className={cn(
                  'size-3 shrink-0',
                  active ? 'opacity-100' : 'opacity-0',
                )}
              />
              <div className="flex flex-col flex-1 min-w-0">
                <span className="text-[12px] truncate">
                  {a.display_name || a.name}
                </span>
                <span className="text-[10px] text-muted-foreground truncate">
                  {a.config_dir || a.name}
                  {!a.token_filled && (
                    <span className="ml-1 text-amber-500/90">
                      {t('web.sessions.accountSwitcher.tokenEmpty')}
                    </span>
                  )}
                </span>
              </div>
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
