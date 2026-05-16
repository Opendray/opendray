import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { GitBranchPlus, GitMerge, Loader2, Upload } from 'lucide-react'
import { toast } from 'sonner'

import {
  checkoutGitBranch,
  createGitBranch,
  listGitBranches,
  pushGit,
} from '@/lib/git'
import { cn } from '@/lib/utils'

interface BranchControlsProps {
  cwd: string
  ahead: number
  upstream?: string
}

// BranchControls is the row of in-place actions that sit under the
// status header in GitPanel. Switching branches refuses on a dirty
// tree (server returns 409); creating a new branch is a single-
// input flow. Push uses set-upstream the first time a branch is
// pushed (no upstream tracked yet) and `--force-with-lease` when
// the operator explicitly opts in.
export function BranchControls({ cwd, ahead, upstream }: BranchControlsProps) {
  const qc = useQueryClient()
  const [creating, setCreating] = useState(false)
  const [newBranch, setNewBranch] = useState('')

  const branches = useQuery({
    queryKey: ['git', 'branches', cwd],
    queryFn: () => listGitBranches(cwd),
    refetchInterval: 30_000,
  })

  const checkout = useMutation({
    mutationFn: (name: string) => checkoutGitBranch(cwd, name),
    onSuccess: (_, name) => {
      toast.success(`Switched to ${name}`)
      qc.invalidateQueries({ queryKey: ['git', 'status', cwd] })
      qc.invalidateQueries({ queryKey: ['git', 'branches', cwd] })
      qc.invalidateQueries({ queryKey: ['git', 'log', cwd] })
    },
    onError: (err) => {
      toast.error('Checkout failed', { description: (err as Error).message })
    },
  })

  const create = useMutation({
    mutationFn: () =>
      createGitBranch({
        dir: cwd,
        name: newBranch.trim(),
        switch: true,
      }),
    onSuccess: () => {
      toast.success(`Created and checked out ${newBranch.trim()}`)
      setNewBranch('')
      setCreating(false)
      qc.invalidateQueries({ queryKey: ['git', 'status', cwd] })
      qc.invalidateQueries({ queryKey: ['git', 'branches', cwd] })
    },
    onError: (err) => {
      toast.error('Create branch failed', { description: (err as Error).message })
    },
  })

  // Push: no upstream → first-push (-u). Existing upstream + ahead
  // > 0 → regular push. ahead == 0 + upstream set → button disabled
  // (nothing to ship).
  const push = useMutation({
    mutationFn: () =>
      pushGit(cwd, {
        set_upstream: !upstream,
      }),
    onSuccess: (res) => {
      toast.success(`Pushed ${res.branch}`)
      qc.invalidateQueries({ queryKey: ['git', 'status', cwd] })
    },
    onError: (err) => {
      toast.error('Push failed', { description: (err as Error).message })
    },
  })

  const refs = branches.data?.branches ?? []
  const local = refs.filter((b) => !b.is_remote)
  const current = branches.data?.current ?? ''

  // Disable push when there's an upstream AND we're not ahead.
  // First push (no upstream) is always allowed even without ahead
  // commits — the operator typically wants to publish the branch
  // even if it has nothing new yet (then opens a PR from it).
  const pushDisabled = !!upstream && ahead === 0

  return (
    <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
      <select
        value={current}
        onChange={(e) => {
          const target = e.target.value
          if (target && target !== current) checkout.mutate(target)
        }}
        disabled={checkout.isPending || branches.isLoading}
        className="bg-card border border-border rounded px-1.5 py-0.5 font-mono text-[11px] outline-none focus:border-state-running"
        title="Switch branch (refuses on dirty tree)"
      >
        {local.length === 0 && <option value="">(no branches)</option>}
        {local.map((b) => (
          <option key={b.name} value={b.name}>
            {b.name}
            {b.upstream ? `  ↦ ${b.upstream}` : ''}
          </option>
        ))}
      </select>

      <button
        type="button"
        onClick={() => setCreating((v) => !v)}
        className="flex items-center gap-0.5 px-1.5 py-0.5 rounded text-muted-foreground hover:text-foreground hover:bg-card"
        title="Create a new branch from HEAD"
      >
        <GitBranchPlus className="size-3" />
        {creating ? 'Cancel' : 'New'}
      </button>

      <button
        type="button"
        onClick={() => push.mutate()}
        disabled={push.isPending || pushDisabled}
        className={cn(
          'ml-auto flex items-center gap-1 px-2 py-0.5 rounded transition-colors',
          push.isPending || pushDisabled
            ? 'text-muted-foreground/40'
            : 'text-state-running hover:bg-state-running/10',
        )}
        title={
          pushDisabled
            ? 'Nothing to push (upstream is up to date)'
            : upstream
              ? `Push to ${upstream}`
              : 'Push and set upstream'
        }
      >
        {push.isPending ? (
          <Loader2 className="size-3 animate-spin" />
        ) : (
          <Upload className="size-3" />
        )}
        {upstream ? `Push (${ahead})` : 'Push (-u)'}
      </button>

      {creating && (
        <div className="basis-full flex items-center gap-1.5 pt-1">
          <GitMerge className="size-3 text-muted-foreground" />
          <input
            type="text"
            value={newBranch}
            onChange={(e) => setNewBranch(e.target.value)}
            placeholder="branch name"
            autoFocus
            className="flex-1 bg-transparent border border-border rounded px-1.5 py-0.5 font-mono text-[11px] outline-none focus:border-state-running"
            onKeyDown={(e) => {
              if (e.key === 'Enter' && newBranch.trim()) create.mutate()
              if (e.key === 'Escape') setCreating(false)
            }}
          />
          <button
            type="button"
            onClick={() => create.mutate()}
            disabled={create.isPending || !newBranch.trim()}
            className={cn(
              'text-[11px] px-2 py-0.5 rounded',
              create.isPending || !newBranch.trim()
                ? 'bg-muted/30 text-muted-foreground/50'
                : 'bg-state-running text-background hover:opacity-90',
            )}
          >
            {create.isPending ? 'Creating…' : 'Create'}
          </button>
        </div>
      )}
    </div>
  )
}
