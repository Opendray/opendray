// Client for the /api/v1/backups, /backup-schedules, /backup-targets,
// /backup-status endpoints. Mirrors the Go shapes in internal/backup.
//
// Backup endpoints are admin-only; this module assumes the caller is
// authenticated. 404 on /backup-status means the feature is disabled
// at the server level (OPENDRAY_BACKUP_ENABLED + OPENDRAY_BACKUP_KEY
// are both required to turn it on).

import { api } from './api'

export type BackupStatus =
  | 'pending'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'deleted'

export type TriggeredBy = 'scheduler' | 'manual' | 'api'

export type TargetKind =
  | 'local'
  | 'smb'
  | 's3'
  | 'webdav'
  | 'sftp'
  | 'rclone'

// TARGET_KINDS is the canonical UI ordering shown in the picker.
// Local first (default), then off-host options in increasing
// "external dependency" weight.
export const TARGET_KINDS: {
  kind: TargetKind
  label: string
  description: string
  examples: string
}[] = [
  {
    kind: 'local',
    label: 'Local disk',
    description: 'Folder on the machine running opendray',
    examples: '~/.opendray/backups · /var/backups · external HDD mounted to a path',
  },
  {
    kind: 'smb',
    label: 'SMB / CIFS',
    description: 'Windows shares + most home NAS appliances',
    examples: 'Synology · QNAP · UNAS · Windows file shares',
  },
  {
    kind: 's3',
    label: 'S3-compatible',
    description: 'AWS S3 + every S3-compatible service',
    examples: 'AWS S3 · Cloudflare R2 · Backblaze B2 · MinIO · Alibaba Cloud OSS (阿里云 OSS) · Tencent Cloud COS (腾讯云 COS) · DigitalOcean Spaces · Wasabi',
  },
  {
    kind: 'webdav',
    label: 'WebDAV',
    description: 'Self-hosted clouds + file-sharing services',
    examples: 'Nextcloud · ownCloud · Synology DSM (群晖 DSM) · Box.com · Jianguoyun (坚果云)',
  },
  {
    kind: 'sftp',
    label: 'SFTP / SSH',
    description: 'Any SSH-accessible server (Linux VPS, etc.)',
    examples: 'Hetzner Storage Box · self-hosted VPS · home Linux server',
  },
  {
    kind: 'rclone',
    label: 'rclone passthrough',
    description: 'Tap into 70+ backends configured via the rclone CLI',
    examples: 'Google Drive · OneDrive · Dropbox · Mega · pCloud · iCloud-WebDAV · Baidu Pan (百度网盘) · Aliyun Drive (阿里云盘)',
  },
]

export interface Backup {
  id: string
  schedule_id?: string | null
  target_id: string
  status: BackupStatus
  triggered_by: TriggeredBy
  started_at: string
  finished_at?: string | null
  bytes: number
  sha256?: string
  encrypted: boolean
  key_fingerprint?: string
  target_path?: string
  pg_version?: string
  opendray_version?: string
  git_sha?: string
  error?: string
  metadata?: Record<string, unknown>
}

export interface Schedule {
  id: string
  target_id: string
  interval_sec: number
  retention: number
  enabled: boolean
  last_run_at?: string | null
  next_run_at: string
  created_at: string
  updated_at: string
}

export interface TargetSpec {
  id: string
  kind: TargetKind
  config: Record<string, unknown>
  enabled: boolean
  created_at: string
  updated_at: string
}

// Since PR #49 /backup-status always returns 200 — `enabled` tells
// you whether the feature is actively running, `configured` whether
// a passphrase is available from any source, and `requires_restart`
// whether the operator just wrote a key file via the UI and the
// gateway hasn't picked it up yet. The legacy `ok` field is still
// here for the live-feature pg_dump health check.
export interface BackupStatusReport {
  enabled: boolean
  configured: boolean
  configured_via: 'env' | 'file' | ''
  can_disable_via_ui: boolean
  requires_restart: boolean
  key_file_path: string
  // Populated only when enabled === true.
  ok?: boolean
  key_fingerprint?: string
  pg_dump_version?: string
  pg_dump_error?: string
  pg_restore_version?: string
}

export async function fetchBackupStatus(): Promise<BackupStatusReport> {
  return api<BackupStatusReport>('/api/v1/backup-status')
}

export interface BackupSetupResult {
  ok: boolean
  key_file_path: string
  requires_restart: boolean
  /** Server-generated passphrase; only present when mode==='generate'. */
  passphrase?: string
}

/** POST /backup-setup. Writes the key file; returns the passphrase
 * once when mode==='generate'. The operator MUST save it before
 * continuing — no recovery path exists. */
export async function postBackupSetup(opts: {
  mode: 'generate' | 'paste'
  passphrase?: string
}): Promise<BackupSetupResult> {
  return api<BackupSetupResult>('/api/v1/backup-setup', {
    method: 'POST',
    body: JSON.stringify(opts),
    headers: { 'Content-Type': 'application/json' },
  })
}

/** POST /backup-setup/disable. Removes the key file (no-op for env-
 * configured deployments — server rejects with 409 in that case). */
export async function postBackupDisable(): Promise<{
  ok: boolean
  requires_restart: boolean
}> {
  return api<{ ok: boolean; requires_restart: boolean }>(
    '/api/v1/backup-setup/disable',
    { method: 'POST' },
  )
}

export async function listBackups(opts?: {
  status?: BackupStatus
  targetId?: string
  limit?: number
}): Promise<Backup[]> {
  const params = new URLSearchParams()
  if (opts?.status) params.set('status', opts.status)
  if (opts?.targetId) params.set('target_id', opts.targetId)
  if (opts?.limit) params.set('limit', String(opts.limit))
  const q = params.toString()
  const res = await api<{ backups: Backup[] }>(
    `/api/v1/backups${q ? `?${q}` : ''}`,
  )
  return res.backups
}

export async function getBackup(id: string): Promise<Backup> {
  return api<Backup>(`/api/v1/backups/${encodeURIComponent(id)}`)
}

export async function createBackup(opts: {
  targetId?: string
  includeConfig?: boolean
}): Promise<Backup> {
  return api<Backup>('/api/v1/backups', {
    method: 'POST',
    body: {
      target_id: opts.targetId ?? 'local',
      include_config: opts.includeConfig ?? false,
    },
  })
}

export async function deleteBackup(id: string): Promise<void> {
  await api<unknown>(`/api/v1/backups/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

/** Browser-friendly download URL — admin token rides via cookie/auth header
 *  on the underlying fetch the browser issues when the user clicks. */
export function backupDownloadURL(id: string): string {
  return `/api/v1/backups/${encodeURIComponent(id)}/download`
}

export async function listTargets(): Promise<TargetSpec[]> {
  const res = await api<{ targets: TargetSpec[] }>('/api/v1/backup-targets')
  return res.targets
}

export async function createTarget(opts: {
  id?: string
  kind: TargetKind
  config: Record<string, unknown>
  enabled: boolean
}): Promise<TargetSpec> {
  return api<TargetSpec>('/api/v1/backup-targets', {
    method: 'POST',
    body: opts,
  })
}

export async function updateTarget(
  id: string,
  patch: { config?: Record<string, unknown>; enabled?: boolean },
): Promise<TargetSpec> {
  return api<TargetSpec>(`/api/v1/backup-targets/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: patch,
  })
}

export async function deleteTarget(id: string): Promise<void> {
  await api<unknown>(`/api/v1/backup-targets/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export interface TestTargetResult {
  ok: boolean
  error?: string
}

export async function testTarget(id: string): Promise<TestTargetResult> {
  return api<TestTargetResult>(
    `/api/v1/backup-targets/${encodeURIComponent(id)}/test`,
    { method: 'POST' },
  )
}

// ── schedules ────────────────────────────────────────────────────

export async function listSchedules(): Promise<Schedule[]> {
  const res = await api<{ schedules: Schedule[] }>('/api/v1/backup-schedules')
  return res.schedules
}

export async function createSchedule(opts: {
  targetId: string
  intervalSec: number
  retention: number
  enabled: boolean
}): Promise<Schedule> {
  return api<Schedule>('/api/v1/backup-schedules', {
    method: 'POST',
    body: {
      target_id: opts.targetId,
      interval_sec: opts.intervalSec,
      retention: opts.retention,
      enabled: opts.enabled,
    },
  })
}

export async function updateSchedule(
  id: string,
  patch: { intervalSec?: number; retention?: number; enabled?: boolean },
): Promise<Schedule> {
  return api<Schedule>(`/api/v1/backup-schedules/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: {
      ...(patch.intervalSec !== undefined && {
        interval_sec: patch.intervalSec,
      }),
      ...(patch.retention !== undefined && { retention: patch.retention }),
      ...(patch.enabled !== undefined && { enabled: patch.enabled }),
    },
  })
}

export async function deleteSchedule(id: string): Promise<void> {
  await api<unknown>(`/api/v1/backup-schedules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

// ── helpers ──────────────────────────────────────────────────────

// ── exports (Plan C) ─────────────────────────────────────────────

export type IntegrationExportMode = 'none' | 'metadata' | 'plaintext'

export type ExportStatus =
  | 'pending'
  | 'running'
  | 'ready'
  | 'failed'
  | 'expired'

export interface ExportScope {
  memories: boolean
  integrations: IntegrationExportMode
  custom_tasks: boolean
}

export interface ExportRecord {
  id: string
  status: ExportStatus
  requested_by: string
  scope: ExportScope
  started_at: string
  finished_at?: string | null
  expires_at: string
  bytes: number
  sha256?: string
  download_token?: string
  error?: string
}

export async function listExports(): Promise<ExportRecord[]> {
  const res = await api<{ exports: ExportRecord[] }>('/api/v1/exports')
  return res.exports
}

export async function getExport(id: string): Promise<ExportRecord> {
  return api<ExportRecord>(`/api/v1/exports/${encodeURIComponent(id)}`)
}

export async function createExport(opts: {
  memories: boolean
  integrations: IntegrationExportMode
  customTasks: boolean
}): Promise<ExportRecord> {
  return api<ExportRecord>('/api/v1/exports', {
    method: 'POST',
    body: {
      memories: opts.memories,
      integrations: opts.integrations,
      custom_tasks: opts.customTasks,
    },
  })
}

export async function deleteExport(id: string): Promise<void> {
  await api<unknown>(`/api/v1/exports/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function exportDownloadURL(id: string, token: string): string {
  return `/api/v1/exports/${encodeURIComponent(id)}/download?token=${encodeURIComponent(token)}`
}

// ── inventory (what's in a backup) ──────────────────────────────

export interface InventoryTable {
  name: string
  count: number
}

export interface InventoryGroup {
  id: string
  label: string
  description: string
  tables: InventoryTable[]
}

export async function fetchBackupInventory(): Promise<InventoryGroup[]> {
  const res = await api<{ groups: InventoryGroup[] }>(
    '/api/v1/backup-inventory',
  )
  return res.groups
}

// ── restore (A) ──────────────────────────────────────────────────

export interface RestoreResult {
  manifest: {
    version: string
    backup_id: string
    created_at: string
    opendray_version?: string
    pg_version?: string
    encryption: { algo: string; fingerprint: string }
  }
  bytes_read: number
  target_dsn_used: string
  fingerprint_ok: boolean
  pg_restore_output: string
  started_at: string
  finished_at: string
}

export async function restoreBackup(opts: {
  bundle: File
  targetDsn?: string
  clean: boolean
  confirm?: string
  note?: string
}): Promise<RestoreResult> {
  const fd = new FormData()
  fd.set('bundle', opts.bundle)
  if (opts.targetDsn) fd.set('target_dsn', opts.targetDsn)
  fd.set('clean', String(opts.clean))
  if (opts.confirm) fd.set('confirm', opts.confirm)
  if (opts.note) fd.set('note', opts.note)
  return api<RestoreResult>('/api/v1/backups/restore', {
    method: 'POST',
    body: fd,
  })
}

// ── imports (C reverse) ──────────────────────────────────────────

export type ImportStatus = 'pending' | 'running' | 'succeeded' | 'failed'

export interface EntityCounts {
  created: number
  skipped: number
  failed: number
}

export interface ImportRecord {
  id: string
  status: ImportStatus
  requested_by: string
  started_at: string
  finished_at?: string | null
  source_filename?: string
  source_bytes: number
  counts: {
    memories: EntityCounts
    integrations: EntityCounts
    custom_tasks: EntityCounts
  }
  error?: string
}

export async function listImports(limit = 20): Promise<ImportRecord[]> {
  const res = await api<{ imports: ImportRecord[] }>(
    `/api/v1/imports?limit=${limit}`,
  )
  return res.imports
}

export async function getImport(id: string): Promise<ImportRecord> {
  return api<ImportRecord>(`/api/v1/imports/${encodeURIComponent(id)}`)
}

export async function createImport(opts: {
  bundle: File
  memories: boolean
  integrations: boolean
  customTasks: boolean
}): Promise<ImportRecord> {
  const fd = new FormData()
  fd.set('bundle', opts.bundle)
  fd.set('memories', String(opts.memories))
  fd.set('integrations', String(opts.integrations))
  fd.set('custom_tasks', String(opts.customTasks))
  return api<ImportRecord>('/api/v1/imports', {
    method: 'POST',
    body: fd,
  })
}

// ── helpers ──────────────────────────────────────────────────────

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MiB`
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GiB`
}

export function formatInterval(sec: number): string {
  if (sec < 60) return `${sec}s`
  if (sec < 3600) return `${Math.round(sec / 60)} min`
  if (sec < 86400) return `${Math.round(sec / 3600)} h`
  return `${Math.round(sec / 86400)} d`
}
