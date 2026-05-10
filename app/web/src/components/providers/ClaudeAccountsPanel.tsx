import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Check,
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
  setClaudeAccountToken,
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
  const [token, setToken] = useState('')
  const [tokenAccountId, setTokenAccountId] = useState<string | null>(null)
  const [pendingToken, setPendingToken] = useState('')

  const add = useMutation({
    mutationFn: () =>
      createClaudeAccount({
        name: name.trim(),
        display_name: displayName.trim() || undefined,
        token: token.trim() || undefined,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['claude-accounts'] })
      toast.success('Account added')
      setName('')
      setDisplayName('')
      setToken('')
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

  const setToken_ = useMutation({
    mutationFn: ({ id, t }: { id: string; t: string }) =>
      setClaudeAccountToken(id, t),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['claude-accounts'] })
      toast.success('Token saved')
      setTokenAccountId(null)
      setPendingToken('')
    },
    onError: (e: Error) =>
      toast.error('Save token failed', { description: e.message }),
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
                size="sm"
                onClick={() => {
                  setTokenAccountId(tokenAccountId === a.id ? null : a.id)
                  setPendingToken('')
                }}
                className="text-[11px]"
              >
                {a.token_filled ? 'Replace token' : 'Set token'}
              </Button>
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
            {tokenAccountId === a.id && (
              <div className="mt-2.5 pt-2.5 border-t border-border/60 flex items-center gap-1.5">
                <Input
                  autoFocus
                  type="password"
                  value={pendingToken}
                  onChange={(e) => setPendingToken(e.target.value)}
                  placeholder="paste OAuth token (sk-ant-…)"
                  className="flex-1 text-[12px]"
                />
                <Button
                  variant="accent"
                  size="sm"
                  disabled={!pendingToken.trim() || setToken_.isPending}
                  onClick={() =>
                    setToken_.mutate({ id: a.id, t: pendingToken.trim() })
                  }
                >
                  {setToken_.isPending ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <Check className="size-3.5" />
                  )}
                  Save
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setTokenAccountId(null)}
                  disabled={setToken_.isPending}
                >
                  Cancel
                </Button>
              </div>
            )}
          </div>
        ))}
      </div>

      {showAdd && (
        <>
          <Separator />
          <div className="rounded-md border border-border bg-muted/20 px-3 py-2.5 text-[11px] text-muted-foreground leading-relaxed">
            <span className="font-medium text-foreground">
              Where does the OAuth token come from?
            </span>{' '}
            On any machine where you've already run{' '}
            <span className="font-mono">claude login</span>, copy the JSON
            blob at{' '}
            <span className="font-mono">~/.claude/.credentials.json</span>{' '}
            (or under{' '}
            <span className="font-mono">
              ~/.claude-accounts/&lt;name&gt;/.claude/.credentials.json
            </span>{' '}
            if you used a per-account dir) and paste it below. Leave the
            field blank to add the row first and set the token later.{' '}
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
            <div className="space-y-1.5">
              <Label htmlFor="acc-token">OAuth token (optional)</Label>
              <Input
                id="acc-token"
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="paste now or set later"
                autoComplete="off"
              />
              <p className="text-[10px] text-muted-foreground/70">
                Leave empty to provision the row only. Token is written
                chmod 600 to <span className="font-mono">token_path</span>.
              </p>
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
