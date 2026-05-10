import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CircleDot,
  Download,
  HelpCircle,
  KeyRound,
  Loader2,
  Trash2,
} from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import {
  deleteClaudeAccount,
  importLocalClaudeAccounts,
  listClaudeAccounts,
  toggleClaudeAccount,
} from '@/lib/claudeAccounts'
import type { ClaudeAccount } from '@/lib/types'

// ClaudeAccountsPanel renders the multi-account list for the Claude
// provider. Account creation is filesystem-driven: operators run
// `CLAUDE_CONFIG_DIR=~/.claude-accounts/<name> claude login` on the
// gateway host and opendray's filesystem watcher picks the directory
// up automatically. The Import local button forces a synchronous
// scan for the case where the operator wants the row to appear
// immediately.
//
// There is intentionally no Add-account form: pasting an OAuth token
// into a web form produces an account that can't refresh and dies in
// ~1 hour, while the canonical claude-login flow produces a
// self-managed credentials file. Forcing the host-shell flow keeps
// the affordance honest.
export function ClaudeAccountsPanel() {
  const qc = useQueryClient()
  const { data: accounts, isLoading } = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
  })

  const importLocal = useMutation({
    mutationFn: importLocalClaudeAccounts,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['claude-accounts'] })
      if (res.count === 0) {
        toast.success('Nothing to import — accounts already in sync.')
      } else {
        toast.success(`Imported ${res.count} account(s) from ~/.claude-accounts`)
      }
    },
    onError: (e: Error) =>
      toast.error('Import failed', { description: e.message }),
  })

  // Optimistic toggle: Radix Switch is fully controlled via
  // `checked={a.enabled}`. Without an optimistic update the thumb
  // doesn't budge between click and refetch, which on a slow round-
  // trip looks indistinguishable from a broken button. We seed the
  // cache with the new value on click and reconcile on settle.
  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      toggleClaudeAccount(id, enabled),
    onMutate: async ({ id, enabled }) => {
      await qc.cancelQueries({ queryKey: ['claude-accounts'] })
      const prev = qc.getQueryData<ClaudeAccount[]>(['claude-accounts'])
      if (prev) {
        qc.setQueryData<ClaudeAccount[]>(
          ['claude-accounts'],
          prev.map((a) => (a.id === id ? { ...a, enabled } : a)),
        )
      }
      return { prev }
    },
    onError: (e: Error, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(['claude-accounts'], ctx.prev)
      toast.error('Toggle failed', { description: e.message })
    },
    onSettled: () =>
      qc.invalidateQueries({ queryKey: ['claude-accounts'] }),
  })

  const remove = useMutation({
    mutationFn: deleteClaudeAccount,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['claude-accounts'] })
      toast.success('Account removed')
    },
    onError: (e: Error) =>
      toast.error('Remove failed', { description: e.message }),
  })

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-[12px] font-semibold uppercase tracking-wider text-muted-foreground/80">
            Claude accounts
          </h2>
          <span className="text-[10px] text-muted-foreground/60 font-mono">
            {accounts?.length ?? 0}
          </span>
          <Link
            to="/tutorial"
            hash="providers-claude-accounts"
            className="text-muted-foreground/70 hover:text-foreground inline-flex items-center"
            title="Open the multi-account tutorial section"
          >
            <HelpCircle className="size-3.5" />
          </Link>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => importLocal.mutate()}
          disabled={importLocal.isPending}
          className="text-[11px] gap-1"
          title="Scan ~/.claude-accounts/ on the gateway host and register any new directories. The button is gateway-host only — see the tutorial."
        >
          {importLocal.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Download className="size-3.5" />
          )}
          Import local
        </Button>
      </div>

      <div className="rounded-md border border-border bg-muted/20 px-3 py-2.5 text-[11px] text-muted-foreground leading-relaxed">
        <span className="font-medium text-foreground">
          Adding a new account.
        </span>{' '}
        Run on the gateway host:
        <pre className="mt-1.5 mb-1.5 px-2 py-1.5 rounded bg-background/60 text-[10.5px] overflow-x-auto">
{`mkdir -p ~/.claude-accounts/<name>
CLAUDE_CONFIG_DIR=~/.claude-accounts/<name> claude login`}
        </pre>
        opendray's filesystem watcher will register the new directory
        automatically, or click <span className="font-mono">Import local</span> to
        scan immediately.{' '}
        <Link
          to="/tutorial"
          hash="providers-claude-accounts"
          className="underline hover:text-foreground"
        >
          Architecture &amp; full guide →
        </Link>
      </div>

      {isLoading && (
        <div className="text-[12px] text-muted-foreground italic">
          Loading…
        </div>
      )}

      {!isLoading && (accounts?.length ?? 0) === 0 && (
        <p className="text-[12px] text-muted-foreground italic">
          No Claude accounts yet. Run the shell command above on the
          gateway host, then click{' '}
          <span className="font-mono">Import local</span> to scan.
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
                      no token yet
                    </span>
                  )}
                </div>
                <div className="text-[10px] font-mono text-muted-foreground/70 truncate">
                  config_dir: {a.config_dir || '—'}
                </div>
                <div className="text-[10px] font-mono text-muted-foreground/70 truncate">
                  token_path: {a.token_path || '—'}
                </div>
              </div>
              <ToggleButton
                enabled={a.enabled}
                pending={toggle.isPending}
                onToggle={(v) => toggle.mutate({ id: a.id, enabled: v })}
                ariaLabel={`Toggle ${a.name}`}
              />
              <Button
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground hover:text-destructive"
                onClick={() => {
                  if (confirm(`Remove account "${a.name}"?`)) {
                    remove.mutate(a.id)
                  }
                }}
                disabled={remove.isPending}
                aria-label={`Remove ${a.name}`}
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

// ToggleButton is a hand-rolled enable/disable control replacing the
// Radix Switch primitive on this panel. Earlier versions used Switch
// but the click was somehow not reaching the primitive in production
// (no onCheckedChange fired, no network call). A plain <button> with
// explicit onClick sidesteps the issue and gives us full control over
// pending and disabled visual states.
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
