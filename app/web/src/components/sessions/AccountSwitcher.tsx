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
import { switchClaudeAccount, switchAntigravityAccount } from '@/lib/sessions'
import { cn } from '@/lib/utils'
import type { Session } from '@/lib/types'

interface AccountSwitcherProps {
  session: Session
}

// Minimal shape shared by ClaudeAccount and AntigravityAccount — the
// only fields this dropdown renders. Lets one component drive both
// providers' multi-account switching.
interface SwitcherAccount {
  id: string
  name: string
  display_name: string
  config_dir: string
  enabled: boolean
  token_filled: boolean
}

// AccountSwitcher renders a header dropdown that lets the user rebind a
// *running* multi-account session (claude or antigravity) to a different
// account. The backend terminates the current child process and respawns
// it under the new credential — the in-CLI conversation is lost (the
// process is replaced), so the dropdown confirms before firing.
//
// Claude isolates accounts via CLAUDE_CONFIG_DIR and supports carrying a
// recap across the switch (the carry toggle). Antigravity isolates via
// HOME and has no cross-account recap builder yet, so its switch is
// always clean-slate and the carry toggle is hidden.
export function AccountSwitcher({ session }: AccountSwitcherProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const isAgy = session.provider_id === 'antigravity'

  const { data: accounts } = useQuery<SwitcherAccount[]>({
    queryKey: isAgy ? ['antigravity-accounts'] : ['claude-accounts'],
    queryFn: isAgy ? listAntigravityAccounts : listClaudeAccounts,
    staleTime: 30_000,
  })
  const currentId = isAgy
    ? session.antigravity_account_id
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
      isAgy
        ? switchAntigravityAccount(session.id, accountId)
        : switchClaudeAccount(session.id, accountId, carryContext),
    onSuccess: (next) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      const nextId = isAgy
        ? next.antigravity_account_id
        : next.claude_account_id
      const account = nextId
        ? enabled.find((a) => a.id === nextId)?.display_name || nextId
        : t('web.sessions.accountSwitcher.switchedDefault')
      toast.success(t('web.sessions.accountSwitcher.switchedToast'), {
        description: t('web.sessions.accountSwitcher.switchedDescription', {
          account,
          pid: next.pid ?? '—',
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
    const msg = isAgy
      ? t('web.sessions.accountSwitcher.confirmSwitchAgy')
      : carryContext
        ? t('web.sessions.accountSwitcher.confirmSwitchCarry')
        : t('web.sessions.accountSwitcher.confirmSwitch')
    if (!confirm(msg)) {
      return
    }
    mutation.mutate(accountId)
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          disabled={mutation.isPending}
          className="text-[11px] gap-1 hover:text-foreground"
          title={t(
            isAgy
              ? 'web.sessions.accountSwitcher.tooltipAgy'
              : 'web.sessions.accountSwitcher.tooltip',
          )}
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
          {t(
            isAgy
              ? 'web.sessions.accountSwitcher.menuTitleAgy'
              : 'web.sessions.accountSwitcher.menuTitle',
          )}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {/* Carry-over toggle (claude only). Stays open on click
            (preventDefault) so the operator sets it before picking a
            destination. The subtitle is the consent surface for the
            cross-account data flow. */}
        {!isAgy && (
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
