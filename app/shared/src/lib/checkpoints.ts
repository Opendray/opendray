import { api } from './api'

// Checkpoint is one captured context snapshot of a session's working dir
// (uncommitted git diff + untracked files + input history). Mirrors the Go
// checkpoint.Checkpoint JSON. The heavy payload lives on the gateway's disk;
// this is the manifest.
export interface Checkpoint {
  id: string
  session_id: string
  created_at: string
  trigger: 'interrupted' | 'manual'
  cwd: string
  is_git: boolean
  git_head?: string
  git_dirty: boolean
  diff_bytes: number
  untracked_files: number
  untracked_bytes: number
  input_bytes: number
  truncated: boolean
  note?: string
}

// RestoreResult reports what a restore actually did (mirrors Go).
export interface RestoreResult {
  checkpoint_id: string
  diff_applied: boolean
  untracked_restored: number
  untracked_skipped?: string[]
}

export async function listCheckpoints(
  sessionId: string,
): Promise<Checkpoint[]> {
  return (
    (await api<Checkpoint[]>(`/api/v1/sessions/${sessionId}/checkpoints`)) ?? []
  )
}

export async function captureCheckpoint(
  sessionId: string,
  note?: string,
): Promise<Checkpoint> {
  return api<Checkpoint>(`/api/v1/sessions/${sessionId}/checkpoints`, {
    method: 'POST',
    body: note ? { note } : {},
  })
}

// readCheckpointDiff returns the raw uncommitted diff (text/plain; empty when
// the working tree had no tracked changes at capture time).
export async function readCheckpointDiff(id: string): Promise<string> {
  return api<string>(`/api/v1/checkpoints/${id}/diff`)
}

// restoreCheckpoint re-applies a checkpoint onto its cwd. The gateway
// enforces strict guards (HEAD match, clean tracked tree, dry-run) and
// returns 409 on any guard failure — surfaced here as an APIError whose
// message is git's/the guard's explanation.
export async function restoreCheckpoint(id: string): Promise<RestoreResult> {
  return api<RestoreResult>(`/api/v1/checkpoints/${id}/restore`, {
    method: 'POST',
  })
}

export async function deleteCheckpoint(id: string): Promise<void> {
  await api(`/api/v1/checkpoints/${id}`, { method: 'DELETE' })
}
