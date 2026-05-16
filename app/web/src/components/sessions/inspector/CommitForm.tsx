import { useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { GitCommit, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { commitGit, stageGitFiles, type GitStatusFile } from '@/lib/git'
import { cn } from '@/lib/utils'

interface CommitFormProps {
  cwd: string
  files: GitStatusFile[]
}

// CommitForm sits under the working-tree file list. Auto-detects
// whether anything is staged; when nothing is, the operator picks
// "Stage all" first. Commit message is required (server also
// validates) and submission happens via Cmd/Ctrl+Enter or button.
export function CommitForm({ cwd, files }: CommitFormProps) {
  const [message, setMessage] = useState('')
  const qc = useQueryClient()

  // Porcelain xy: first char is index (staged), second is worktree.
  // " " = no change in that bucket; "?" = untracked. Anything else
  // in the first column means there's something staged already.
  const staged = useMemo(
    () => files.filter((f) => f.xy[0] !== ' ' && f.xy[0] !== '?'),
    [files],
  )

  const stageAll = useMutation({
    mutationFn: () => stageGitFiles(cwd, []),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['git', 'status', cwd] })
    },
    onError: (err) => {
      toast.error('Stage failed', { description: (err as Error).message })
    },
  })

  const commit = useMutation({
    mutationFn: () => commitGit(cwd, message.trim()),
    onSuccess: (res) => {
      toast.success('Committed', {
        description: res.hash.slice(0, 12),
      })
      setMessage('')
      qc.invalidateQueries({ queryKey: ['git', 'status', cwd] })
      qc.invalidateQueries({ queryKey: ['git', 'log', cwd] })
    },
    onError: (err) => {
      toast.error('Commit failed', { description: (err as Error).message })
    },
  })

  const hasMessage = message.trim() !== ''
  const canCommit = hasMessage && staged.length > 0 && !commit.isPending

  return (
    <div className="flex flex-col gap-1.5 px-1 pt-1">
      <div className="flex items-center justify-between text-[10px] text-muted-foreground/70">
        <span>
          {staged.length === 0
            ? 'Nothing staged'
            : `${staged.length} staged · ${files.length - staged.length} unstaged`}
        </span>
        {staged.length === 0 && files.length > 0 && (
          <button
            type="button"
            onClick={() => stageAll.mutate()}
            disabled={stageAll.isPending}
            className="text-state-running hover:underline disabled:opacity-50"
          >
            {stageAll.isPending ? 'Staging…' : 'Stage all'}
          </button>
        )}
      </div>
      <textarea
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        onKeyDown={(e) => {
          if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && canCommit) {
            e.preventDefault()
            commit.mutate()
          }
        }}
        placeholder={
          staged.length === 0
            ? 'Stage files first, then write a commit message'
            : 'Commit message (Cmd/Ctrl+Enter to commit)'
        }
        rows={2}
        className="text-[11px] font-mono bg-card/30 border border-border rounded px-2 py-1 outline-none focus:border-state-running resize-y"
      />
      <div className="flex justify-end">
        <button
          type="button"
          onClick={() => commit.mutate()}
          disabled={!canCommit}
          className={cn(
            'text-[11px] px-2 py-0.5 rounded flex items-center gap-1',
            canCommit
              ? 'bg-state-running text-background hover:opacity-90'
              : 'bg-muted/30 text-muted-foreground/50 cursor-not-allowed',
          )}
        >
          {commit.isPending ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <GitCommit className="size-3" />
          )}
          Commit
        </button>
      </div>
    </div>
  )
}
