import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CloudUpload,
  CloudDownload,
  GitCommit,
  GitBranch,
  ArrowUp,
  ArrowDown,
  Loader2,
  Settings2,
  CheckCircle2,
  AlertCircle,
  Clock,
  Play,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { toast } from 'sonner'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { Link } from '@tanstack/react-router'

import {
  vaultStatus,
  vaultInit,
  vaultCommit,
  vaultPull,
  vaultPush,
  vaultLog,
  vaultGetRemotes,
  vaultSetRemote,
  vaultAuthInfo,
  vaultAbort,
  vaultResetToRemote,
  vaultSyncConfig,
  setVaultSyncConfig,
  vaultSyncRunNow,
} from '@/lib/vaultgit'
import type {
  VaultStatus,
  VaultStatusFile,
  VaultAuthInfo,
  VaultGitState,
  VaultSyncConfig,
  VaultSyncConfigUpdate,
} from '@/lib/vaultgit'

interface VaultSyncDialogProps {
  open: boolean
  onOpenChange: (v: boolean) => void
}

// VaultSyncDialog is the one place users go to manage the vault's git
// state. Three tiers visible at once:
//   - top: branch / ahead-behind chips + primary actions (commit, pull, push)
//   - middle: file list with XY status codes
//   - bottom: remote URL config + recent commit history
//
// When the vault isn't a repo yet, we render the init prompt instead.
export function VaultSyncDialog({ open, onOpenChange }: VaultSyncDialogProps) {
  const qc = useQueryClient()

  const status = useQuery({
    queryKey: ['vault-status'],
    queryFn: vaultStatus,
    enabled: open,
    refetchInterval: open ? 4_000 : false,
  })

  const remotes = useQuery({
    queryKey: ['vault-remotes'],
    queryFn: vaultGetRemotes,
    enabled: open && status.data?.is_repo === true,
  })

  const log = useQuery({
    queryKey: ['vault-log'],
    queryFn: () => vaultLog(10),
    enabled: open && status.data?.is_repo === true,
  })

  const auth = useQuery({
    queryKey: ['vault-auth'],
    queryFn: vaultAuthInfo,
    enabled: open && status.data?.is_repo === true,
    refetchInterval: open ? 8_000 : false,
  })

  const [message, setMessage] = useState('')
  const [showRemoteForm, setShowRemoteForm] = useState(false)
  const [remoteUrl, setRemoteUrl] = useState('')

  // Reset transient form state when dialog reopens.
  useEffect(() => {
    if (open) {
      setMessage('')
      setShowRemoteForm(false)
    }
  }, [open])

  // Pre-fill remote URL field when editing an existing remote.
  useEffect(() => {
    const origin = remotes.data?.find((r) => r.name === 'origin')
    if (origin && remoteUrl === '') setRemoteUrl(origin.url)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [remotes.data])

  const init = useMutation({
    mutationFn: vaultInit,
    onSuccess: () => {
      toast.success('Vault initialised as git repo')
      qc.invalidateQueries({ queryKey: ['vault-status'] })
      qc.invalidateQueries({ queryKey: ['vault-remotes'] })
      qc.invalidateQueries({ queryKey: ['vault-log'] })
    },
    onError: (e: Error) => toast.error('Init failed', { description: e.message }),
  })

  const commit = useMutation({
    mutationFn: () => vaultCommit({ message }),
    onSuccess: (res) => {
      toast.success(`Committed ${res.hash}`, { description: res.message })
      setMessage('')
      qc.invalidateQueries({ queryKey: ['vault-status'] })
      qc.invalidateQueries({ queryKey: ['vault-log'] })
    },
    onError: (e: Error) => toast.error('Commit failed', { description: e.message }),
  })

  const pull = useMutation({
    mutationFn: vaultPull,
    onSuccess: () => {
      toast.success('Pulled')
      qc.invalidateQueries({ queryKey: ['vault-status'] })
      qc.invalidateQueries({ queryKey: ['vault-log'] })
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      qc.invalidateQueries({ queryKey: ['notes-list-all'] })
    },
    onError: (e: Error) => toast.error('Pull failed', { description: e.message }),
  })

  const push = useMutation({
    mutationFn: vaultPush,
    onSuccess: () => {
      toast.success('Pushed')
      qc.invalidateQueries({ queryKey: ['vault-status'] })
    },
    onError: (e: Error) => toast.error('Push failed', { description: e.message }),
  })

  const setRemote = useMutation({
    mutationFn: () => vaultSetRemote('origin', remoteUrl.trim()),
    onSuccess: () => {
      toast.success('Remote saved')
      setShowRemoteForm(false)
      qc.invalidateQueries({ queryKey: ['vault-remotes'] })
      qc.invalidateQueries({ queryKey: ['vault-status'] })
    },
    onError: (e: Error) =>
      toast.error('Set remote failed', { description: e.message }),
  })

  const abort = useMutation({
    mutationFn: () => vaultAbort('auto'),
    onSuccess: (res) => {
      toast.success(`Aborted ${res.kind}`, {
        description: 'Working tree restored to pre-operation state.',
      })
      qc.invalidateQueries({ queryKey: ['vault-status'] })
      qc.invalidateQueries({ queryKey: ['vault-log'] })
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      qc.invalidateQueries({ queryKey: ['notes-list-all'] })
    },
    onError: (e: Error) =>
      toast.error('Abort failed', { description: e.message }),
  })

  const resetToRemote = useMutation({
    mutationFn: () => vaultResetToRemote(),
    onSuccess: (res) => {
      toast.success(`Reset to ${res.remote_branch}`, {
        description: 'Local changes discarded; vault matches remote.',
      })
      qc.invalidateQueries({ queryKey: ['vault-status'] })
      qc.invalidateQueries({ queryKey: ['vault-log'] })
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      qc.invalidateQueries({ queryKey: ['notes-list-all'] })
    },
    onError: (e: Error) =>
      toast.error('Reset failed', { description: e.message }),
  })

  const summary = useMemo(() => {
    const files = status.data?.files ?? []
    let staged = 0
    let modified = 0
    let untracked = 0
    for (const f of files) {
      const x = f.xy[0]
      const y = f.xy[1]
      if (x === '?' && y === '?') untracked++
      else {
        if (x !== ' ' && x !== '?') staged++
        if (y !== ' ' && y !== '?') modified++
      }
    }
    return { staged, modified, untracked, total: files.length }
  }, [status.data])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[min(92vw,820px)] w-[min(92vw,820px)] h-[min(85vh,720px)] gap-2 flex flex-col">
        <DialogHeader className="shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <GitBranch className="size-4 text-muted-foreground" />
            Vault sync
          </DialogTitle>
          <DialogDescription>
            Commit, pull, and push the notes vault as a git repository.
            Authentication uses your gateway host's git credentials
            (SSH agent / credential helper).
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 min-h-0 overflow-y-auto flex flex-col gap-4">
          {status.isLoading ? (
            <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-3">
              <Loader2 className="size-3 animate-spin" />
              Reading vault state…
            </div>
          ) : !status.data?.is_repo ? (
            <InitPrompt
              root={status.data?.root}
              busy={init.isPending}
              onInit={() => init.mutate()}
            />
          ) : (
            <>
              {status.data.state && (
                <ConflictBanner
                  state={status.data.state}
                  aborting={abort.isPending}
                  resetting={resetToRemote.isPending}
                  onAbort={() => abort.mutate()}
                  onReset={() =>
                    resetToRemote.mutate(undefined as never)
                  }
                />
              )}
              <BranchBar
                branch={status.data.branch}
                upstream={status.data.upstream}
                ahead={status.data.ahead}
                behind={status.data.behind}
                summary={summary}
              />

              <ActionRow
                hasChanges={summary.total > 0}
                hasRemote={(remotes.data?.length ?? 0) > 0}
                hasUpstream={!!status.data.upstream}
                ahead={status.data.ahead}
                behind={status.data.behind}
                pulling={pull.isPending}
                pushing={push.isPending}
                onPull={() => pull.mutate()}
                onPush={() => push.mutate()}
              />

              <CommitForm
                message={message}
                setMessage={setMessage}
                summary={summary}
                committing={commit.isPending}
                onCommit={() => commit.mutate()}
              />

              <FileList files={status.data.files} />

              <RemoteSection
                remotes={remotes.data ?? []}
                editing={showRemoteForm}
                url={remoteUrl}
                setUrl={setRemoteUrl}
                onToggle={() => setShowRemoteForm((v) => !v)}
                onSave={() => setRemote.mutate()}
                saving={setRemote.isPending}
              />

              <AuthSection
                info={auth.data}
                onClose={() => onOpenChange(false)}
              />

              <AutoSyncSection
                hasRemote={(remotes.data?.length ?? 0) > 0}
                open={open}
              />

              <CommitHistory commits={log.data ?? []} loading={log.isLoading} />
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function InitPrompt({
  root,
  busy,
  onInit,
}: {
  root?: string
  busy: boolean
  onInit: () => void
}) {
  return (
    <div className="flex flex-col items-center text-center gap-3 py-6">
      <GitBranch className="size-8 text-muted-foreground/40" strokeWidth={1.5} />
      <div className="space-y-1">
        <h3 className="text-[14px] font-semibold">Vault is not a git repo yet</h3>
        <p className="text-[12px] text-muted-foreground max-w-[480px]">
          Initialising will run <code>git init -b main</code> in your vault
          root and add a sane <code>.gitignore</code>. After that you can
          commit your notes and configure a remote (GitHub / Gitea / GitLab)
          for cross-machine sync.
        </p>
        {root && (
          <div className="text-[11px] font-mono text-muted-foreground/70 mt-2">
            {root}
          </div>
        )}
      </div>
      <Button onClick={onInit} disabled={busy} variant="accent" size="sm">
        {busy && <Loader2 className="size-3.5 animate-spin" />}
        Initialise vault as git repo
      </Button>
    </div>
  )
}

function BranchBar({
  branch,
  upstream,
  ahead,
  behind,
  summary,
}: {
  branch?: string
  upstream?: string
  ahead: number
  behind: number
  summary: { staged: number; modified: number; untracked: number; total: number }
}) {
  return (
    <section className="rounded-md border border-border/60 bg-card/40 px-3 py-2 flex flex-wrap items-center gap-x-3 gap-y-1.5">
      <div className="flex items-center gap-1.5 text-[12px]">
        <GitBranch className="size-3.5 text-muted-foreground" />
        <span className="font-mono font-medium">{branch ?? '—'}</span>
        {upstream && (
          <span className="text-[10px] text-muted-foreground/70 font-mono">
            ↦ {upstream}
          </span>
        )}
      </div>
      <div className="flex items-center gap-2 text-[10.5px] font-mono">
        {ahead > 0 && (
          <span className="text-state-running flex items-center gap-0.5">
            <ArrowUp className="size-3" />
            {ahead}
          </span>
        )}
        {behind > 0 && (
          <span className="text-state-idle flex items-center gap-0.5">
            <ArrowDown className="size-3" />
            {behind}
          </span>
        )}
      </div>
      <div className="flex-1" />
      <div className="flex items-center gap-2 text-[10.5px] font-mono">
        {summary.total === 0 ? (
          <span className="text-state-running flex items-center gap-1">
            <CheckCircle2 className="size-3" />
            clean
          </span>
        ) : (
          <>
            {summary.staged > 0 && (
              <span className="text-state-running">{summary.staged} staged</span>
            )}
            {summary.modified > 0 && (
              <span className="text-state-idle">{summary.modified} modified</span>
            )}
            {summary.untracked > 0 && (
              <span className="text-muted-foreground">
                {summary.untracked} untracked
              </span>
            )}
          </>
        )}
      </div>
    </section>
  )
}

function ActionRow({
  hasChanges,
  hasRemote,
  hasUpstream,
  ahead,
  pulling,
  pushing,
  onPull,
  onPush,
}: {
  hasChanges: boolean
  hasRemote: boolean
  hasUpstream: boolean
  ahead: number
  behind: number
  pulling: boolean
  pushing: boolean
  onPull: () => void
  onPush: () => void
}) {
  // Push is enabled whenever we have a remote — `git push -u origin
  // HEAD` will create the upstream tracking on its first call, so we
  // must not block on hasUpstream (chicken-and-egg).
  const pushTitle = !hasRemote
    ? 'Configure a remote first'
    : hasUpstream
      ? 'git push -u origin HEAD'
      : 'First push — will set upstream to origin/HEAD'
  const pullTitle = !hasRemote
    ? 'Configure a remote first'
    : hasUpstream
      ? 'git pull --rebase --autostash'
      : 'Pulls origin\'s HEAD; sets up tracking implicitly'
  return (
    <div className="flex items-center gap-2 flex-wrap">
      <Button
        type="button"
        size="sm"
        variant="outline"
        onClick={onPull}
        disabled={!hasRemote || pulling}
        className="gap-1.5"
        title={pullTitle}
      >
        {pulling ? (
          <Loader2 className="size-3 animate-spin" />
        ) : (
          <CloudDownload className="size-3" />
        )}
        Pull
      </Button>
      <Button
        type="button"
        size="sm"
        variant="outline"
        onClick={onPush}
        disabled={!hasRemote || pushing}
        className={cn('gap-1.5', (ahead > 0 || hasChanges) && 'border-state-running/40')}
        title={pushTitle}
      >
        {pushing ? (
          <Loader2 className="size-3 animate-spin" />
        ) : (
          <CloudUpload className="size-3" />
        )}
        Push
      </Button>
      {!hasRemote ? (
        <span className="text-[10.5px] text-muted-foreground/70 inline-flex items-center gap-1">
          <AlertCircle className="size-3" />
          No remote configured — pull/push disabled
        </span>
      ) : !hasUpstream ? (
        <span className="text-[10.5px] text-muted-foreground/70 inline-flex items-center gap-1">
          <AlertCircle className="size-3" />
          No upstream tracking yet — first push will set it.
        </span>
      ) : null}
    </div>
  )
}

function CommitForm({
  message,
  setMessage,
  summary,
  committing,
  onCommit,
}: {
  message: string
  setMessage: (s: string) => void
  summary: { staged: number; modified: number; untracked: number; total: number }
  committing: boolean
  onCommit: () => void
}) {
  return (
    <section className="flex flex-col gap-2 rounded-md border border-border/60 bg-card/40 px-3 py-2.5">
      <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
        <GitCommit className="size-3" />
        Commit
      </div>
      <div className="flex gap-2">
        <Input
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder={`Notes: ${new Date().toISOString().slice(0, 10)} (default)`}
          className="flex-1 text-[12px]"
          onKeyDown={(e) => {
            if (e.key === 'Enter' && summary.total > 0) {
              e.preventDefault()
              onCommit()
            }
          }}
        />
        <Button
          type="button"
          size="sm"
          variant="accent"
          disabled={summary.total === 0 || committing}
          onClick={onCommit}
          className="gap-1.5"
        >
          {committing ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <GitCommit className="size-3" />
          )}
          Commit all
        </Button>
      </div>
      <p className="text-[10.5px] text-muted-foreground/70">
        Stages every change (<code>git add .</code>) then commits with this
        message. Empty message defaults to a timestamped subject.
      </p>
    </section>
  )
}

function FileList({ files }: { files: VaultStatusFile[] }) {
  if (files.length === 0) {
    return null
  }
  return (
    <section className="flex flex-col gap-1">
      <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium px-1">
        Working tree · {files.length}
      </div>
      <div className="rounded-md border border-border/60 bg-card/30 max-h-[180px] overflow-y-auto">
        {files.slice(0, 200).map((f) => {
          const x = f.xy[0]
          const y = f.xy[1]
          const untracked = x === '?' && y === '?'
          const colorIdx = untracked
            ? 'text-muted-foreground'
            : x !== ' '
              ? 'text-state-running'
              : 'text-state-idle'
          return (
            <div
              key={f.path}
              className="flex items-center gap-2 px-2 py-0.5 hover:bg-card/60"
              title={`${f.xy} ${f.path}`}
            >
              <span
                className={cn(
                  'shrink-0 inline-block w-5 text-center text-[10px] font-mono font-semibold',
                  colorIdx,
                )}
              >
                {f.xy.replace(/ /g, '·')}
              </span>
              <span className="text-[11px] font-mono truncate flex-1">{f.path}</span>
            </div>
          )
        })}
        {files.length > 200 && (
          <div className="text-[10px] text-muted-foreground/60 px-2 py-1">
            +{files.length - 200} more
          </div>
        )}
      </div>
    </section>
  )
}

function RemoteSection({
  remotes,
  editing,
  url,
  setUrl,
  onToggle,
  onSave,
  saving,
}: {
  remotes: { name: string; url: string }[]
  editing: boolean
  url: string
  setUrl: (s: string) => void
  onToggle: () => void
  onSave: () => void
  saving: boolean
}) {
  const origin = remotes.find((r) => r.name === 'origin')
  return (
    <section className="flex flex-col gap-2 rounded-md border border-border/60 bg-card/40 px-3 py-2.5">
      <div className="flex items-center gap-1.5">
        <Settings2 className="size-3 text-muted-foreground" />
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
          Remote (origin)
        </span>
        <div className="flex-1" />
        <button
          type="button"
          onClick={onToggle}
          className="text-[11px] text-muted-foreground hover:text-foreground"
        >
          {editing ? 'Cancel' : origin ? 'Change' : 'Configure'}
        </button>
      </div>
      {!editing && origin ? (
        <div
          className="text-[11px] font-mono text-muted-foreground/80 truncate"
          title={origin.url}
        >
          {origin.url}
        </div>
      ) : !editing && !origin ? (
        <p className="text-[11px] text-muted-foreground/70">
          No remote set. Add an HTTPS or SSH URL (e.g.{' '}
          <code>git@github.com:you/notes.git</code> or{' '}
          <code>https://tea.linivek.online/you/notes.git</code>) to enable
          push / pull.
        </p>
      ) : (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="remote-url" className="text-[10.5px] text-muted-foreground/80">
            URL (HTTPS or SSH)
          </Label>
          <div className="flex gap-2">
            <Input
              id="remote-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="git@host:owner/notes.git"
              className="flex-1 text-[12px] font-mono"
            />
            <Button
              type="button"
              size="sm"
              variant="accent"
              disabled={saving || !url.trim()}
              onClick={onSave}
            >
              {saving ? <Loader2 className="size-3 animate-spin" /> : 'Save'}
            </Button>
          </div>
        </div>
      )}
    </section>
  )
}

function CommitHistory({
  commits,
  loading,
}: {
  commits: { hash: string; short_hash: string; author: string; when: string; subject: string }[]
  loading: boolean
}) {
  return (
    <section className="flex flex-col gap-1">
      <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium px-1">
        Recent commits
      </div>
      {loading ? (
        <div className="flex items-center gap-2 text-[11px] text-muted-foreground px-1">
          <Loader2 className="size-3 animate-spin" />
          Loading…
        </div>
      ) : commits.length === 0 ? (
        <div className="text-[11px] text-muted-foreground/60 px-1">
          No commits yet.
        </div>
      ) : (
        <div className="rounded-md border border-border/60 bg-card/30 divide-y divide-border/40">
          {commits.map((c) => (
            <div key={c.hash} className="px-2 py-1.5">
              <div className="text-[12px] truncate">{c.subject}</div>
              <div className="text-[10px] text-muted-foreground/60 font-mono">
                {c.short_hash} · {c.author} ·{' '}
                {(() => {
                  try {
                    return c.when || ''
                  } catch {
                    return c.when
                  }
                })()}
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  )
}

function ConflictBanner({
  state,
  aborting,
  resetting,
  onAbort,
  onReset,
}: {
  state: VaultGitState
  aborting: boolean
  resetting: boolean
  onAbort: () => void
  onReset: () => void
}) {
  const kind = state.rebase_in_progress
    ? 'rebase'
    : state.merge_in_progress
      ? 'merge'
      : state.cherry_pick_in_progress
        ? 'cherry-pick'
        : 'operation'
  const conflicts = state.conflicted_files ?? []
  return (
    <section className="rounded-md border border-state-failed/40 bg-state-failed/5 px-3 py-3 flex flex-col gap-2.5">
      <div className="flex items-center gap-1.5">
        <AlertCircle className="size-4 text-state-failed" />
        <span className="text-[12px] font-semibold text-state-failed">
          Vault has a paused {kind} with unresolved conflicts
        </span>
      </div>
      <p className="text-[11.5px] text-foreground/85 leading-relaxed">
        Pull, push and commit are blocked until the {kind} finishes. You can
        either <strong>abort</strong> (restore the working tree to its state
        before the {kind} — keeps your local commits, drops the remote ones)
        or <strong>force reset to remote</strong> (discard ALL local
        commits + uncommitted changes; vault becomes an exact mirror of
        origin).
      </p>
      {conflicts.length > 0 && (
        <div className="text-[10.5px] font-mono text-muted-foreground/80 border-l-2 border-state-failed/40 pl-2 max-h-32 overflow-y-auto">
          <div className="text-[10px] uppercase tracking-wider text-muted-foreground/60 mb-1">
            Conflicted files · {conflicts.length}
          </div>
          {conflicts.map((p) => (
            <div key={p} className="truncate" title={p}>
              {p}
            </div>
          ))}
        </div>
      )}
      <div className="flex items-center gap-2 flex-wrap">
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={onAbort}
          disabled={aborting || resetting}
          className="gap-1.5"
          title={`git ${kind} --abort`}
        >
          {aborting ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <AlertCircle className="size-3" />
          )}
          Abort {kind}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => {
            if (
              confirm(
                'DESTRUCTIVE: this will\n  • abort the in-progress ' +
                  kind +
                  '\n  • run git fetch origin\n  • reset --hard to origin/<branch>\n  • clean -fd (drop untracked files)\n\n' +
                  'Any local commits not pushed to origin AND any uncommitted edits will be PERMANENTLY LOST.\n\n' +
                  'Continue?',
              )
            ) {
              onReset()
            }
          }}
          disabled={aborting || resetting}
          className="gap-1.5 text-destructive hover:text-destructive border-destructive/40 hover:border-destructive/60"
          title="git fetch && git reset --hard origin/<branch> && git clean -fd"
        >
          {resetting ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <CloudDownload className="size-3" />
          )}
          Force reset to remote
        </Button>
      </div>
    </section>
  )
}

function AuthSection({
  info,
  onClose,
}: {
  info: VaultAuthInfo | undefined
  onClose: () => void
}) {
  if (!info) return null
  if (!info.has_remote) return null

  const isHTTPS = info.scheme === 'https' || info.scheme === 'http'
  const tone =
    info.using_token
      ? 'ok'
      : info.token_missing
        ? 'warn'
        : 'info'

  const cls =
    tone === 'ok'
      ? 'border-state-running/40 bg-state-running/5'
      : tone === 'warn'
        ? 'border-state-idle/40 bg-state-idle/5'
        : 'border-border/60 bg-card/40'

  return (
    <section
      className={cn(
        'flex flex-col gap-2 rounded-md border px-3 py-2.5',
        cls,
      )}
    >
      <div className="flex items-center gap-1.5">
        {info.using_token ? (
          <CheckCircle2 className="size-3 text-state-running" />
        ) : info.token_missing ? (
          <AlertCircle className="size-3 text-state-idle" />
        ) : (
          <Settings2 className="size-3 text-muted-foreground" />
        )}
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
          Authentication
        </span>
      </div>

      <div className="text-[11.5px] text-foreground/85 leading-relaxed">
        {isHTTPS ? (
          info.using_token ? (
            <>
              Will use the token stored for{' '}
              <code className="font-mono text-[11px]">{info.host}</code> in
              Plugins → Git hosts. ✓
            </>
          ) : (
            <>
              HTTPS remote on{' '}
              <code className="font-mono text-[11px]">{info.host}</code> with
              no opendray token configured. Push / pull will likely fail for
              private repos until you add one.
            </>
          )
        ) : (
          <>
            SSH remote on{' '}
            <code className="font-mono text-[11px]">{info.host}</code>. Auth
            uses the gateway host's <code className="font-mono">~/.ssh/</code>{' '}
            (ssh-agent, identity file, host config). Verify with{' '}
            <code className="font-mono text-[11px]">
              ssh -T git@{info.host}
            </code>{' '}
            from the host shell.
          </>
        )}
      </div>

      {info.helpful_hint && tone !== 'ok' && (
        <div className="text-[10.5px] text-muted-foreground/80 italic">
          {info.helpful_hint}
        </div>
      )}

      {isHTTPS && info.token_missing && (
        <Link
          to="/plugins"
          onClick={onClose}
          className="text-[11px] text-state-running hover:underline self-start inline-flex items-center gap-0.5"
        >
          → Configure git host token
        </Link>
      )}
    </section>
  )
}

// AutoSyncSection wraps the persistent /sync/config: enable toggle,
// commit/pull intervals (parsed as Go-style "10m"/"1h" strings),
// per-channel toggles, last-run readout and a manual "Run now" button.
// State only writes to the server on Save — keeps things calm if the
// user is mid-edit when the 4s status poll fires.
function AutoSyncSection({
  hasRemote,
  open,
}: {
  hasRemote: boolean
  open: boolean
}) {
  const qc = useQueryClient()
  const cfg = useQuery({
    queryKey: ['vault-sync-config'],
    queryFn: vaultSyncConfig,
    enabled: open,
    refetchInterval: open ? 8_000 : false,
  })

  const [draft, setDraft] = useState<VaultSyncConfigUpdate>({})
  const [collapsed, setCollapsed] = useState(true)

  // Whenever the server config changes (and the user isn't actively
  // editing) clear the local draft so the inputs reflect the truth.
  useEffect(() => {
    if (cfg.data) setDraft({})
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cfg.dataUpdatedAt])

  const save = useMutation({
    mutationFn: () => setVaultSyncConfig(draft),
    onSuccess: (next) => {
      qc.setQueryData(['vault-sync-config'], next)
      setDraft({})
      toast.success('Auto-sync settings saved')
    },
    onError: (e: Error) =>
      toast.error('Save failed', { description: e.message }),
  })

  const runNow = useMutation({
    mutationFn: vaultSyncRunNow,
    onSuccess: () => {
      toast.success('Auto-sync triggered')
      qc.invalidateQueries({ queryKey: ['vault-sync-config'] })
      qc.invalidateQueries({ queryKey: ['vault-status'] })
      qc.invalidateQueries({ queryKey: ['vault-log'] })
    },
    onError: (e: Error) =>
      toast.error('Run failed', { description: e.message }),
  })

  const c = cfg.data
  // Effective view = server values overlaid with draft edits.
  const v = useMemo<VaultSyncConfig | null>(() => {
    if (!c) return null
    return {
      ...c,
      enabled: draft.enabled ?? c.enabled,
      commit_interval: draft.commit_interval ?? prettyDuration(c.commit_interval),
      pull_interval: draft.pull_interval ?? prettyDuration(c.pull_interval),
      push_enabled: draft.push_enabled ?? c.push_enabled,
      pull_enabled: draft.pull_enabled ?? c.pull_enabled,
      commit_message: draft.commit_message ?? (c.commit_message ?? ''),
    }
  }, [c, draft])

  const dirty = Object.keys(draft).length > 0

  if (cfg.isLoading || !v || !c) {
    return (
      <section className="flex items-center gap-2 rounded-md border border-border/60 bg-card/40 px-3 py-2.5 text-[11px] text-muted-foreground">
        <Loader2 className="size-3 animate-spin" />
        Loading auto-sync settings…
      </section>
    )
  }

  return (
    <section className="flex flex-col gap-2 rounded-md border border-border/60 bg-card/40 px-3 py-2.5">
      <div className="flex items-center gap-1.5">
        <Clock className="size-3 text-muted-foreground" />
        <span className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
          Auto-sync
        </span>
        {v.enabled && (
          <span className="text-[10px] font-mono text-state-running inline-flex items-center gap-0.5">
            <span className="size-1.5 rounded-full bg-state-running animate-pulse" />
            on
          </span>
        )}
        <div className="flex-1" />
        <button
          type="button"
          onClick={() => runNow.mutate()}
          disabled={runNow.isPending}
          className="text-[11px] inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-muted-foreground hover:text-foreground hover:bg-card disabled:opacity-50"
          title="Wake the sync loop now (skips the wait, then runs whichever steps are due)"
        >
          {runNow.isPending ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <Play className="size-3" />
          )}
          Run now
        </button>
        <button
          type="button"
          onClick={() => setCollapsed((p) => !p)}
          className="text-[11px] text-muted-foreground hover:text-foreground"
        >
          {collapsed ? 'Configure' : 'Hide'}
        </button>
      </div>

      <div className="flex items-center gap-2 text-[11px]">
        <label className="inline-flex items-center gap-1.5 cursor-pointer">
          <input
            type="checkbox"
            checked={v.enabled}
            onChange={(e) =>
              setDraft((p) => ({ ...p, enabled: e.target.checked }))
            }
            className="size-3.5 accent-state-running"
            disabled={!hasRemote && !v.enabled}
            title={
              !hasRemote
                ? 'Configure a remote first to enable auto-sync'
                : undefined
            }
          />
          <span>Enabled</span>
        </label>
        {!hasRemote && (
          <span className="text-[10.5px] text-muted-foreground/70 inline-flex items-center gap-1">
            <AlertCircle className="size-3" />
            No remote — push/pull will be skipped.
          </span>
        )}
      </div>

      {!collapsed && (
        <div className="flex flex-col gap-2.5 pt-1">
          <div className="grid grid-cols-2 gap-2">
            <div className="flex flex-col gap-1">
              <Label
                htmlFor="commit-interval"
                className="text-[10.5px] text-muted-foreground/80"
              >
                Commit every
              </Label>
              <Input
                id="commit-interval"
                value={v.commit_interval}
                onChange={(e) =>
                  setDraft((p) => ({ ...p, commit_interval: e.target.value }))
                }
                placeholder="10m"
                className="text-[12px] font-mono h-8"
              />
              <span className="text-[10px] text-muted-foreground/60">
                Examples: <code>30s</code>, <code>10m</code>, <code>2h</code>.
                Min 30s.
              </span>
            </div>

            <div className="flex flex-col gap-1">
              <Label
                htmlFor="pull-interval"
                className="text-[10.5px] text-muted-foreground/80"
              >
                Pull every
              </Label>
              <Input
                id="pull-interval"
                value={v.pull_interval}
                onChange={(e) =>
                  setDraft((p) => ({ ...p, pull_interval: e.target.value }))
                }
                placeholder="1h"
                className="text-[12px] font-mono h-8"
                disabled={!v.pull_enabled}
              />
              <span className="text-[10px] text-muted-foreground/60">
                Only used when Pull is enabled.
              </span>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-[11px]">
            <label className="inline-flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={v.push_enabled}
                onChange={(e) =>
                  setDraft((p) => ({ ...p, push_enabled: e.target.checked }))
                }
                className="size-3.5 accent-state-running"
              />
              <span>Push after commit</span>
            </label>
            <label className="inline-flex items-center gap-1.5 cursor-pointer">
              <input
                type="checkbox"
                checked={v.pull_enabled}
                onChange={(e) =>
                  setDraft((p) => ({ ...p, pull_enabled: e.target.checked }))
                }
                className="size-3.5 accent-state-running"
              />
              <span>Pull periodically</span>
            </label>
          </div>

          <div className="flex flex-col gap-1">
            <Label
              htmlFor="commit-template"
              className="text-[10.5px] text-muted-foreground/80"
            >
              Commit message template
            </Label>
            <Input
              id="commit-template"
              value={v.commit_message ?? ''}
              onChange={(e) =>
                setDraft((p) => ({ ...p, commit_message: e.target.value }))
              }
              placeholder="Auto-sync: {date}  (default if empty)"
              className="text-[12px] h-8"
            />
          </div>

          <div className="flex items-center gap-2">
            <Button
              type="button"
              size="sm"
              variant="accent"
              onClick={() => save.mutate()}
              disabled={!dirty || save.isPending}
            >
              {save.isPending && <Loader2 className="size-3 animate-spin" />}
              Save settings
            </Button>
            {dirty && (
              <button
                type="button"
                onClick={() => setDraft({})}
                className="text-[11px] text-muted-foreground hover:text-foreground"
              >
                Discard
              </button>
            )}
          </div>
        </div>
      )}

      <div className="flex flex-col gap-0.5 text-[10.5px] text-muted-foreground/70 font-mono pt-1 border-t border-border/40">
        <SyncTimestampRow
          label="last commit"
          ts={c.last_commit_at}
          extra={c.last_commit_hash ? `· ${c.last_commit_hash}` : ''}
        />
        <SyncTimestampRow label="last push" ts={c.last_push_at} />
        <SyncTimestampRow label="last pull" ts={c.last_pull_at} />
        {c.last_error && (
          <div className="text-state-failed mt-1 text-[10.5px] font-sans whitespace-pre-wrap break-all">
            <AlertCircle className="size-3 inline-block mr-1 align-text-bottom" />
            {c.last_error}
            {c.last_error_at && (
              <span className="text-muted-foreground/60 ml-1">
                ({relativeTime(c.last_error_at)})
              </span>
            )}
          </div>
        )}
      </div>
    </section>
  )
}

function SyncTimestampRow({
  label,
  ts,
  extra,
}: {
  label: string
  ts?: string
  extra?: string
}) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-muted-foreground/50 w-20">{label}</span>
      <span>{ts ? relativeTime(ts) : 'never'}</span>
      {extra && <span className="text-muted-foreground/50">{extra}</span>}
    </div>
  )
}

// prettyDuration trims Go's verbose duration form (e.g. "10m0s", "1h0m0s")
// down to something nice in an input field. Falls back to original on
// anything we don't recognise.
function prettyDuration(s: string): string {
  if (!s) return ''
  return s
    .replace(/(\d+h)0m0s$/, '$1')
    .replace(/0h(\d+m)0s$/, '$1')
    .replace(/(\d+m)0s$/, '$1')
    .replace(/^0h/, '')
}

function relativeTime(iso: string): string {
  try {
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return iso
    return formatDistanceToNow(d, { addSuffix: true })
  } catch {
    return iso
  }
}

// Sync status indicator suitable for the Notes page header. Compact —
// one chip showing branch + change count, click to open the dialog.
// When auto-sync is enabled we add a small clock dot so the user can
// see the loop is active without opening the dialog.
export function VaultSyncBadge({
  status,
  onClick,
}: {
  status: VaultStatus | undefined
  onClick: () => void
}) {
  // Cheap independent fetch — same query key as the dialog so we share
  // the cache. enabled=is_repo so we don't poll on a non-repo vault.
  const autoCfg = useQuery({
    queryKey: ['vault-sync-config'],
    queryFn: vaultSyncConfig,
    enabled: !!status?.is_repo,
    refetchInterval: status?.is_repo ? 30_000 : false,
  })
  const autoOn = !!autoCfg.data?.enabled
  const autoErr = autoCfg.data?.last_error
  if (!status) {
    return (
      <button
        type="button"
        onClick={onClick}
        className="text-[11px] inline-flex items-center gap-1 px-2 py-1 rounded-md text-muted-foreground hover:bg-card hover:text-foreground"
        title="Loading…"
      >
        <Loader2 className="size-3 animate-spin" />
        Sync
      </button>
    )
  }
  if (!status.is_repo) {
    return (
      <button
        type="button"
        onClick={onClick}
        className="text-[11px] inline-flex items-center gap-1 px-2 py-1 rounded-md text-muted-foreground hover:bg-card hover:text-foreground border border-dashed border-border"
        title="Vault is not a git repo yet"
      >
        <GitBranch className="size-3" />
        Init
      </button>
    )
  }
  // Conflict state takes priority — push/pull are blocked until it's
  // resolved, so the badge screams red until the user opens the dialog.
  const conflicted = !!status.state
  if (conflicted) {
    const n = status.state?.conflicted_files?.length ?? 0
    return (
      <button
        type="button"
        onClick={onClick}
        className="text-[11px] inline-flex items-center gap-1.5 px-2 py-1 rounded-md border border-state-failed/40 bg-state-failed/10 text-state-failed hover:bg-state-failed/15"
        title="Vault has unresolved conflicts — click to recover"
      >
        <AlertCircle className="size-3" />
        Conflict{n > 0 ? ` · ${n}` : ''}
      </button>
    )
  }

  const dirty = status.files.length > 0
  const ahead = status.ahead
  const behind = status.behind
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'text-[11px] inline-flex items-center gap-1.5 px-2 py-1 rounded-md',
        'hover:bg-card border border-border',
        dirty
          ? 'border-state-idle/40 text-state-idle'
          : ahead > 0
            ? 'border-state-running/40 text-state-running'
            : 'text-muted-foreground hover:text-foreground',
      )}
      title={`branch ${status.branch ?? '—'} · ${status.files.length} changes · ${ahead} ahead · ${behind} behind${
        autoOn ? ' · auto-sync on' : ''
      }${autoErr ? ` · last error: ${autoErr}` : ''}`}
    >
      <GitBranch className="size-3" />
      <span className="font-mono">{status.branch ?? 'sync'}</span>
      {dirty && (
        <span className="text-[10px] font-mono">
          ·{status.files.length}
        </span>
      )}
      {ahead > 0 && (
        <span className="text-[10px] font-mono inline-flex items-center">
          <ArrowUp className="size-2.5" />
          {ahead}
        </span>
      )}
      {behind > 0 && (
        <span className="text-[10px] font-mono inline-flex items-center text-state-idle">
          <ArrowDown className="size-2.5" />
          {behind}
        </span>
      )}
      {autoOn && (
        <Clock
          className={cn(
            'size-2.5',
            autoErr ? 'text-state-failed' : 'text-state-running',
          )}
        />
      )}
    </button>
  )
}

