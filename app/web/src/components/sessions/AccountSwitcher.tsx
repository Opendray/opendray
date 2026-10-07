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
import { listCodexAccounts } from '@/lib/codexAccounts'
import { listOpenCodeAccounts } from '@/lib/opencodeAccounts'
import {
  switchClaudeAccount,
  switchAntigravityAccount,
  switchGrokAccount,
  switchCodexAccount,
  switchOpenCodeAccount,
} from '@/lib/sessions'
import { cn } from '@/lib/utils'
import type { Session } from '@/lib/types'

interface AccountSwitcherProps {
  session: Session
}

// Minimal shape shared by every provider's accounts — the only fields this
// dropdown renders. Lets one component drive all multi-account switching.
interface SwitcherAccount {
  id: string
  name: string
  display_name: string
  config_dir: string
  enabled: boolean
  token_filled: boolean
  // opencode accounts are credential bundles (no dir); show their
  // provider ids as the subtitle instead.
  providers?: string[]
}

type Kind = 'claude' | 'antigravity' | 'grok' | 'codex' | 'opencode'

// Per-provider wiring. carry=true shows the "carry over" toggle, which is
// the operator's consent to send the prior conversation to the provider
// under the new account (claude, grok, codex and opencode resume it in
// full); antigravity always carries its conversation, so no toggle.
const KINDS: Record<
  Kind,
  {
    queryKey: string
    list: () => Promise<SwitcherAccount[]>
    accountOf: (s: Session) => string | undefined
    switchTo: (id: string, accountId: string, carry: boolean) => Promise<Session>
    carry: boolean
    confirmKey: string
    confirmCarryKey: string
    tooltipKey: string
    menuTitleKey: string
  }
> = {
  claude: {
    queryKey: 'claude-accounts',
    list: listClaudeAccounts,
    accountOf: (s) => s.claude_account_id,
    switchTo: switchClaudeAccount,
    carry: true,
    confirmKey: 'web.sessions.accountSwitcher.confirmSwitch',
    confirmCarryKey: 'web.sessions.accountSwitcher.confirmSwitchCarry',
    tooltipKey: 'web.sessions.accountSwitcher.tooltip',
    menuTitleKey: 'web.sessions.accountSwitcher.menuTitle',
  },
  antigravity: {
    queryKey: 'antigravity-accounts',
    list: listAntigravityAccounts,
    accountOf: (s) => s.antigravity_account_id,
    switchTo: (id, accountId) => switchAntigravityAccount(id, accountId),
    carry: false,
    confirmKey: 'web.sessions.accountSwitcher.confirmSwitchAgy',
    confirmCarryKey: 'web.sessions.accountSwitcher.confirmSwitchAgy',
    tooltipKey: 'web.sessions.accountSwitcher.tooltipAgy',
    menuTitleKey: 'web.sessions.accountSwitcher.menuTitleAgy',
  },
  grok: {
    queryKey: 'grok-accounts',
    list: listGrokAccounts,
    accountOf: (s) => s.grok_account_id,
    switchTo: switchGrokAccount,
    carry: true,
    confirmKey: 'web.sessions.accountSwitcher.confirmSwitchGrok',
    confirmCarryKey: 'web.sessions.accountSwitcher.confirmSwitchGrokCarry',
    tooltipKey: 'web.sessions.accountSwitcher.tooltipGrok',
    menuTitleKey: 'web.sessions.accountSwitcher.menuTitleGrok',
  },
  codex: {
    queryKey: 'codex-accounts',
    list: listCodexAccounts,
    accountOf: (s) => s.codex_account_id,
    switchTo: switchCodexAccount,
    carry: true,
    confirmKey: 'web.sessions.accountSwitcher.confirmSwitchCodex',
    confirmCarryKey: 'web.sessions.accountSwitcher.confirmSwitchCodexCarry',
    tooltipKey: 'web.sessions.accountSwitcher.tooltipCodex',
    menuTitleKey: 'web.sessions.accountSwitcher.menuTitleCodex',
  },
  opencode: {
    queryKey: 'opencode-accounts',
    list: listOpenCodeAccounts,
    accountOf: (s) => s.opencode_account_id,
    switchTo: switchOpenCodeAccount,
    carry: true,
    confirmKey: 'web.sessions.accountSwitcher.confirmSwitchOpenCode',
    confirmCarryKey: 'web.sessions.accountSwitcher.confirmSwitchOpenCodeCarry',
    tooltipKey: 'web.sessions.accountSwitcher.tooltipOpenCode',
    menuTitleKey: 'web.sessions.accountSwitcher.menuTitleOpenCode',
  },
}

function kindOf(providerId: string): Kind {
  return providerId === 'antigravity' ||
    providerId === 'grok' ||
    providerId === 'codex' ||
    providerId === 'opencode'
    ? providerId
    : 'claude'
}

// AccountSwitcher renders a header dropdown that lets the user rebind a
// *running* multi-account session (claude, antigravity, grok, codex or
// opencode)
// to a different account. The backend terminates the current child process
// and respawns it under the new credential; with the carry toggle on the
// conversation follows it, and the dropdown confirms before firing.
export function AccountSwitcher({ session }: AccountSwitcherProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const kind = kindOf(session.provider_id)
  const cfg = KINDS[kind]

  const { data: accounts } = useQuery<SwitcherAccount[]>({
    queryKey: [cfg.queryKey],
    queryFn: cfg.list,
    staleTime: 30_000,
  })
  const currentId = cfg.accountOf(session)
  const enabled = (accounts ?? []).filter((a) => a.enabled)
  const current = (accounts ?? []).find((a) => a.id === currentId)
  const currentLabel = currentId
    ? current?.display_name || current?.name || currentId
    : t('web.sessions.accountSwitcher.currentDefault')

  // Carry-over toggle. When on, the conversation follows the switch.
  const [carryContext, setCarryContext] = useState(true)

  const mutation = useMutation({
    mutationFn: (accountId: string) =>
      cfg.switchTo(session.id, accountId, carryContext),
    onSuccess: (next) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      const nextId = cfg.accountOf(next)
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
    const msg = t(
      cfg.carry && carryContext ? cfg.confirmCarryKey : cfg.confirmKey,
    )
    if (!confirm(msg)) {
      return
    }
    mutation.mutate(accountId)
  }

  const supportsCarry = cfg.carry
  const tooltipKey = cfg.tooltipKey
  const menuTitleKey = cfg.menuTitleKey

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
        {/* Carry-over toggle (every provider but antigravity). Stays open on click
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
                  {a.config_dir || a.providers?.join(', ') || a.name}
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
