import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CircleDot,
  Download,
  HelpCircle,
  KeyRound,
  Loader2,
  Plus,
  Trash2,
} from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import {
  createClaudeAccount,
  deleteClaudeAccount,
  importLocalClaudeAccounts,
  listClaudeAccounts,
  toggleClaudeAccount,
} from '@/lib/claudeAccounts'

// ClaudeAccountsPanel renders the v1-style multi-account UI: each
// account is a (name, displayName, configDir, tokenPath) tuple. The
// OAuth token is uploaded separately via the Set-token form because
// it's a long string and operators usually paste it from the host
// tool `claude-acc`'s output.
export function ClaudeAccountsPanel() {
  const qc = useQueryClient()
  const { data: accounts, isLoading } = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
  })

  const [showAdd, setShowAdd] = useState(false)
  const [name, setName] = useState('')
  const [displayName, setDisplayName] = useState('')

  const add = useMutation({
    mutationFn: () =>
      createClaudeAccount({
        name: name.trim(),
        display_name: displayName.trim() || undefined,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['claude-accounts'] })
      toast.success('Account row created — populate the credentials with `claude login` on the gateway host.')
      setName('')
      setDisplayName('')
      setShowAdd(false)
    },
    onError: (e: Error) => toast.error('Add failed', { description: e.message }),
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

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      toggleClaudeAccount(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['claude-accounts'] }),
    onError: (e: Error) =>
      toast.error('Toggle failed', { description: e.message }),
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

  const submitAdd = (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      toast.error('Name is required')
      return
    }
    add.mutate()
  }

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
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => importLocal.mutate()}
            disabled={importLocal.isPending}
            className="text-[11px] gap-1"
            title="Scan ~/.claude-accounts/ on the gateway host and register any new accounts. Only useful when the gateway has filesystem access to your home dir."
          >
            {importLocal.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Download className="size-3.5" />
            )}
            Import local
          </Button>
          {!showAdd && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowAdd(true)}
              className="text-[11px] gap-1"
            >
              <Plus className="size-3.5" />
              Add account
            </Button>
          )}
        </div>
      </div>

      {isLoading && (
        <div className="text-[12px] text-muted-foreground italic">
          Loading…
        </div>
      )}

      {!isLoading && (accounts?.length ?? 0) === 0 && !showAdd && (
        <p className="text-[12px] text-muted-foreground italic">
          No Claude accounts yet. Use{' '}
          <span className="font-mono">Import local</span> if you've already
          run <span className="font-mono">claude-acc init</span> on the
          gateway host, or click{' '}
          <span className="font-mono">Add account</span> to register one.
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
              <Switch
                checked={a.enabled}
                onCheckedChange={(v) =>
                  toggle.mutate({ id: a.id, enabled: v })
                }
                aria-label={`Toggle ${a.name}`}
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

      {showAdd && (
        <>
          <Separator />
          <div className="rounded-md border border-border bg-muted/20 px-3 py-2.5 text-[11px] text-muted-foreground leading-relaxed">
            <span className="font-medium text-foreground">
              How to populate the credentials.
            </span>{' '}
            "Add account" only reserves the row; the OAuth login itself
            is run on the gateway host:
            <pre className="mt-1.5 mb-1 px-2 py-1.5 rounded bg-background/60 text-[10.5px] overflow-x-auto">
{`mkdir -p ~/.claude-accounts/<name>
CLAUDE_CONFIG_DIR=~/.claude-accounts/<name> claude login`}
            </pre>
            opendray's filesystem watcher picks up the new dir on
            its own — you can also click <span className="font-mono">Import local</span> to
            scan immediately.{' '}
            <Link
              to="/tutorial"
              hash="providers-claude-accounts"
              className="underline hover:text-foreground"
            >
              Full guide →
            </Link>
          </div>
          <form onSubmit={submitAdd} className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="acc-name">Name (slug)</Label>
              <Input
                id="acc-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="personal"
                required
              />
              <p className="text-[10px] text-muted-foreground/70">
                Used as a slug. config_dir defaults to{' '}
                <span className="font-mono">
                  ~/.claude-accounts/&lt;name&gt;
                </span>
                .
              </p>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="acc-display">Display name (optional)</Label>
              <Input
                id="acc-display"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="Personal subscription"
              />
            </div>
            <div className="flex justify-end gap-2 pt-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setShowAdd(false)}
                disabled={add.isPending}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="accent"
                size="sm"
                disabled={add.isPending}
              >
                {add.isPending && <Loader2 className="size-3.5 animate-spin" />}
                {add.isPending ? 'Adding…' : 'Add account'}
              </Button>
            </div>
          </form>
        </>
      )}
    </div>
  )
}
