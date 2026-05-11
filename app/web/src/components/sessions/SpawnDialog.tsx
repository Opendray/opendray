import { useState, useEffect, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FolderOpen, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

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

export function SpawnDialog({
  open,
  onOpenChange,
  onSpawned,
  defaultCwd,
}: SpawnDialogProps) {
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
  const { data: claudeAccounts } = useQuery({
    queryKey: ['claude-accounts'],
    queryFn: listClaudeAccounts,
    enabled: open && isClaude,
  })
  const accounts = (claudeAccounts ?? []).filter((a) => a.enabled)

  // When provider changes, clear account selection so we don't keep
  // a stale id from a different provider.
  useEffect(() => {
    setAccountId('')
  }, [providerId])

  const mutation = useMutation({
    mutationFn: createSession,
    onSuccess: (session) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      toast.success('Session spawned', {
        description: `${session.provider_id} · pid ${session.pid ?? '—'}`,
      })
      onSpawned(session)
      onOpenChange(false)
      // Reset for next spawn.
      setName('')
      setArgsText('')
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
      setError('Pick a provider.')
      return
    }
    if (!cwd.trim()) {
      setError('cwd is required.')
      return
    }
    const args = argsText
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s.length > 0)
    mutation.mutate({
      provider_id: providerId,
      cwd: cwd.trim(),
      name: name.trim() || undefined,
      args: args.length > 0 ? args : undefined,
      claude_account_id: isClaude && accountId ? accountId : undefined,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Spawn session</DialogTitle>
          <DialogDescription>
            Start a CLI session under a registered provider.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-4 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="provider">Provider</Label>
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

          {isClaude && accounts.length > 0 && (
            <div className="space-y-1.5">
              <Label>Claude account</Label>
              <div className="flex flex-wrap gap-1.5">
                <button
                  type="button"
                  onClick={() => setAccountId('')}
                  className={`px-2 py-1 rounded-md border text-[11px] transition-colors ${
                    accountId === ''
                      ? 'border-foreground/30 bg-card'
                      : 'border-border hover:bg-card hover:border-foreground/20'
                  }`}
                  title="Use system keychain / env"
                >
                  Default
                </button>
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
                          : 'No token set — set token in Providers panel first'
                      }
                    >
                      {a.display_name || a.name}
                      {!a.token_filled && (
                        <span className="ml-1 text-amber-500/90">·empty</span>
                      )}
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="cwd">Working directory</Label>
            <div className="flex gap-1.5">
              <Input
                id="cwd"
                value={cwd}
                onChange={(e) => setCwd(e.target.value)}
                placeholder="/Users/you/projects/foo"
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
                Browse
              </Button>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="name">Name (optional)</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="claude in pet-tracker"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="args">CLI args (one per line)</Label>
            <textarea
              id="args"
              rows={3}
              value={argsText}
              onChange={(e) => setArgsText(e.target.value)}
              placeholder={`-c\necho hello`}
              className="w-full font-mono text-[12px] rounded-md border border-border bg-input/40 px-3 py-2 text-foreground transition-colors placeholder:text-muted-foreground/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-none"
            />
          </div>

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
              Cancel
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
              {mutation.isPending ? 'Spawning…' : 'Spawn'}
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
