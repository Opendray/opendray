import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, ChevronDown, Loader2, UserRound } from 'lucide-react'
import { toast } from 'sonner'

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
import { switchClaudeAccount } from '@/lib/sessions'
import { cn } from '@/lib/utils'
import type { Session } from '@/lib/types'

interface AccountSwitcherProps {
  session: Session
}

// AccountSwitcher renders a header dropdown that lets the user rebind
// a *running* Claude session to a different account. The backend
// terminates the current child process and respawns it under the new
// credential — the conversation context maintained inside the CLI is
// lost (the underlying process is replaced), so the dropdown shows a
// confirm prompt before firing.
export function AccountSwitcher({ session }: AccountSwitcherProps) {
  const qc = useQueryClient()
  const { data: accounts } = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    staleTime: 30_000,
  })
  const enabled = (accounts ?? []).filter((a) => a.enabled)
  const current = (accounts ?? []).find((a) => a.id === session.claude_account_id)
  const currentLabel = session.claude_account_id
    ? current?.display_name || current?.name || session.claude_account_id
    : 'default'

  const mutation = useMutation({
    mutationFn: (accountId: string) =>
      switchClaudeAccount(session.id, accountId),
    onSuccess: (next) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      toast.success('Account switched', {
        description: `Now using @${
          next.claude_account_id
            ? enabled.find((a) => a.id === next.claude_account_id)?.display_name
              || next.claude_account_id
            : 'default'
        } · pid ${next.pid ?? '—'}`,
      })
    },
    onError: (err: Error) =>
      toast.error('Switch failed', { description: err.message }),
  })

  const pick = (accountId: string) => {
    if (accountId === (session.claude_account_id ?? '')) return
    if (
      !confirm(
        'Switching account will restart the Claude CLI process. ' +
          'In-progress conversation state inside the CLI will be lost. Continue?',
      )
    ) {
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
          title="Switch Claude account (restarts the CLI process)"
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
          Switch Claude account
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
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
              session.claude_account_id ? 'opacity-0' : 'opacity-100',
            )}
          />
          <div className="flex flex-col flex-1 min-w-0">
            <span className="text-[12px]">Default</span>
            <span className="text-[10px] text-muted-foreground">
              CLI's system keychain / env
            </span>
          </div>
        </DropdownMenuItem>
        {enabled.length > 0 && <DropdownMenuSeparator />}
        {enabled.map((a) => {
          const active = session.claude_account_id === a.id
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
                    <span className="ml-1 text-amber-500/90">·empty</span>
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
