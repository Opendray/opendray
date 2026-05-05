import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  Archive,
  Download,
  HardDrive,
  KeyRound,
  Play,
  Plus,
  RotateCw,
  ShieldAlert,
  Trash2,
  Upload,
} from 'lucide-react'

import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogTrigger,
} from '@/components/ui/dialog'

import {
  type Backup,
  type BackupStatusReport,
  type Schedule,
  type TargetKind,
  type TargetSpec,
  backupDownloadURL,
  createBackup,
  createSchedule,
  createTarget,
  deleteBackup,
  deleteSchedule,
  deleteTarget,
  fetchBackupStatus,
  formatBytes,
  formatInterval,
  listBackups,
  listSchedules,
  listTargets,
  restoreBackup,
  testTarget,
  updateSchedule,
} from '@/lib/backup'
import { APIError } from '@/lib/api'
import { cn } from '@/lib/utils'

export function BackupsView() {
  const [status, setStatus] = useState<BackupStatusReport | null | undefined>(
    undefined,
  )

  useEffect(() => {
    fetchBackupStatus()
      .then(setStatus)
      .catch((err: unknown) => {
        const msg = err instanceof Error ? err.message : 'Unknown error'
        toast.error('Failed to load backup status', { description: msg })
        setStatus(null)
      })
  }, [])

  if (status === undefined) {
    return <div className="text-muted-foreground text-sm">Loading…</div>
  }
  if (status === null) {
    return <FeatureDisabledBanner />
  }
  return (
    <div className="flex flex-col gap-5">
      <StatusBanner status={status} />
      <Tabs defaultValue="backups" className="w-full">
        <TabsList>
          <TabsTrigger value="backups">Backups</TabsTrigger>
          <TabsTrigger value="schedules">Schedules</TabsTrigger>
          <TabsTrigger value="targets">Targets</TabsTrigger>
        </TabsList>
        <TabsContent value="backups" className="mt-4">
          <BackupsTab />
        </TabsContent>
        <TabsContent value="schedules" className="mt-4">
          <SchedulesTab />
        </TabsContent>
        <TabsContent value="targets" className="mt-4">
          <TargetsTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// ── Status banner ────────────────────────────────────────────────

function FeatureDisabledBanner() {
  return (
    <div className="rounded-md border border-state-idle/40 bg-state-idle/10 p-4">
      <div className="flex items-start gap-3">
        <Archive className="size-4 mt-0.5 text-state-idle" />
        <div className="text-[13px]">
          <div className="font-medium">Backup feature is disabled</div>
          <div className="text-muted-foreground mt-1">
            Set <code className="text-foreground">OPENDRAY_BACKUP_ENABLED=1</code>{' '}
            and <code className="text-foreground">OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</code>{' '}
            in opendray's environment, then restart. Without a master
            passphrase the server refuses to encrypt or decrypt any
            backup blob.
          </div>
        </div>
      </div>
    </div>
  )
}

function StatusBanner({ status }: { status: BackupStatusReport }) {
  return (
    <div
      className={cn(
        'rounded-md border p-3 text-[12px]',
        status.ok
          ? 'border-state-running/30 bg-state-running/10'
          : 'border-state-failed/30 bg-state-failed/10',
      )}
    >
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
        <div className="flex items-center gap-2">
          <KeyRound className="size-3.5 text-accent" />
          <span className="text-muted-foreground">Key fingerprint:</span>
          <code className="text-foreground">{status.key_fingerprint}</code>
        </div>
        <div className="flex items-center gap-2">
          <HardDrive className="size-3.5 text-accent" />
          <span className="text-muted-foreground">pg_dump:</span>
          {status.ok ? (
            <code className="text-foreground">{status.pg_dump_version}</code>
          ) : (
            <span className="text-state-failed">
              {status.pg_dump_error || 'unavailable'}
            </span>
          )}
        </div>
      </div>
      {!status.ok && (
        <div className="mt-2 text-state-failed">
          Backups can't run until pg_dump is on PATH (or its absolute
          path is set in <code>backup.pg_dump_path</code>). Install{' '}
          <code>postgresql-client</code> matching your server's major
          version and restart.
        </div>
      )}
    </div>
  )
}

// ── Backups tab ──────────────────────────────────────────────────

function BackupsTab() {
  const [rows, setRows] = useState<Backup[] | null>(null)
  const [busy, setBusy] = useState(false)
  const [includeConfig, setIncludeConfig] = useState(true)
  const [restoreOpen, setRestoreOpen] = useState(false)

  async function refresh() {
    try {
      const list = await listBackups({ limit: 50 })
      setRows(list)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Failed to list backups', { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    const t = window.setInterval(refresh, 5000)
    return () => window.clearInterval(t)
  }, [])

  async function trigger() {
    setBusy(true)
    try {
      await createBackup({ includeConfig })
      toast.success('Backup queued')
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Trigger failed', { description: msg })
    } finally {
      setBusy(false)
    }
  }

  async function onDelete(id: string) {
    if (!window.confirm(`Delete backup ${id}? The blob is removed from its target.`)) {
      return
    }
    try {
      await deleteBackup(id)
      toast.success('Backup deleted')
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Delete failed', { description: msg })
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3 flex-wrap">
        <Button onClick={trigger} disabled={busy} size="sm" className="h-8">
          <Play className="size-3.5 mr-1.5" />
          {busy ? 'Triggering…' : 'Backup now'}
        </Button>
        <label className="flex items-center gap-2 text-[12px] text-muted-foreground">
          <Switch
            checked={includeConfig}
            onCheckedChange={setIncludeConfig}
            className="scale-75"
          />
          include config.toml
        </label>
        <Dialog open={restoreOpen} onOpenChange={setRestoreOpen}>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm" className="h-8 ml-auto">
              <Upload className="size-3.5 mr-1.5" />
              Restore from file
            </Button>
          </DialogTrigger>
          <RestoreDialog
            onDone={async () => {
              setRestoreOpen(false)
              await refresh()
            }}
          />
        </Dialog>
        <Button
          onClick={refresh}
          variant="outline"
          size="sm"
          className="h-8"
        >
          <RotateCw className="size-3.5 mr-1.5" />
          Refresh
        </Button>
      </div>
      <BackupTable rows={rows} onDelete={onDelete} />
    </div>
  )
}

function RestoreDialog({ onDone }: { onDone: () => void | Promise<void> }) {
  const [file, setFile] = useState<File | null>(null)
  const [targetDsn, setTargetDsn] = useState('')
  const [clean, setClean] = useState(true)
  const [confirm, setConfirm] = useState('')
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [output, setOutput] = useState<string | null>(null)

  const restoringOwn = targetDsn === ''
  const confirmReady = !restoringOwn || confirm.trim() === 'I understand'

  async function submit() {
    if (!file) {
      toast.error('Pick a bundle file first')
      return
    }
    setBusy(true)
    setOutput(null)
    try {
      const res = await restoreBackup({
        bundle: file,
        targetDsn: targetDsn || undefined,
        clean,
        confirm: restoringOwn ? confirm : undefined,
        note,
      })
      setOutput(res.pg_restore_output || '(no pg_restore output)')
      toast.success('Restore succeeded', {
        description: `${formatBytes(res.bytes_read)} replayed from manifest ${res.manifest.backup_id}`,
      })
      await onDone()
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error('Restore failed', { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Restore from backup bundle</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">Encrypted bundle (.tar.gz.enc)</Label>
          <input
            type="file"
            accept=".enc,.tar.gz.enc,application/octet-stream"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            className="text-[12px]"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">
            Target database DSN
            <span className="text-muted-foreground ml-1 text-[11px]">
              (blank = opendray's own DB — DANGEROUS)
            </span>
          </Label>
          <Input
            value={targetDsn}
            onChange={(e) => setTargetDsn(e.target.value)}
            placeholder="postgres://user:pass@host:5432/dbname"
            className="h-8 font-mono text-[11px]"
          />
        </div>
        <label className="flex items-center gap-2 text-[12px]">
          <Switch
            checked={clean}
            onCheckedChange={setClean}
            className="scale-75"
          />
          --clean --if-exists (drop existing schema first; required when
          restoring over a populated DB)
        </label>
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">Audit note (optional)</Label>
          <Input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Reason for restore — appears in slog"
            className="h-8"
          />
        </div>

        {restoringOwn && (
          <div className="rounded-md border border-state-failed/40 bg-state-failed/10 p-3 text-[12px] flex gap-2 items-start">
            <ShieldAlert className="size-4 text-state-failed shrink-0 mt-0.5" />
            <div className="flex-1 flex flex-col gap-2">
              <div>
                You're restoring into <strong>opendray's own database</strong>.
                With "--clean" enabled this drops every table and replays the
                backup verbatim — irreversible. Type{' '}
                <code className="px-1 rounded bg-card text-foreground">
                  I understand
                </code>{' '}
                to proceed.
              </div>
              <Input
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                placeholder="I understand"
                className="h-7 text-[12px]"
              />
            </div>
          </div>
        )}

        {output && (
          <details className="rounded-md border border-border bg-card/30 p-2 text-[11px]">
            <summary className="cursor-pointer text-muted-foreground">
              pg_restore output (last 8 KiB)
            </summary>
            <pre className="mt-2 whitespace-pre-wrap font-mono">{output}</pre>
          </details>
        )}
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy || !file || !confirmReady}>
          {busy ? 'Restoring…' : 'Restore'}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

function BackupTable({
  rows,
  onDelete,
}: {
  rows: Backup[] | null
  onDelete: (id: string) => void | Promise<void>
}) {
  if (rows === null) {
    return <div className="text-muted-foreground text-sm">Loading…</div>
  }
  if (rows.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-border p-6 text-center text-muted-foreground text-[13px]">
        No backups yet. Click "Backup now" above to take the first one.
      </div>
    )
  }
  return (
    <div className="rounded-md border border-border overflow-hidden">
      <table className="w-full text-[12px]">
        <thead className="bg-card/50 text-muted-foreground">
          <tr className="text-left">
            <th className="px-3 py-2 font-medium">ID</th>
            <th className="px-3 py-2 font-medium">Target</th>
            <th className="px-3 py-2 font-medium">Status</th>
            <th className="px-3 py-2 font-medium">Started</th>
            <th className="px-3 py-2 font-medium">Size</th>
            <th className="px-3 py-2 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((b) => (
            <tr key={b.id} className="border-t border-border/60">
              <td className="px-3 py-2 font-mono text-[11px]">{b.id}</td>
              <td className="px-3 py-2">{b.target_id}</td>
              <td className="px-3 py-2">
                <StatusBadge status={b.status} />
                {b.error && (
                  <span
                    className="ml-2 text-state-failed text-[11px]"
                    title={b.error}
                  >
                    {b.error.length > 40 ? b.error.slice(0, 40) + '…' : b.error}
                  </span>
                )}
              </td>
              <td className="px-3 py-2 text-muted-foreground">
                {formatRelative(b.started_at)}
              </td>
              <td className="px-3 py-2 text-muted-foreground">
                {b.bytes > 0 ? formatBytes(b.bytes) : '—'}
              </td>
              <td className="px-3 py-2">
                <div className="flex justify-end gap-1">
                  {b.status === 'succeeded' && (
                    <a
                      href={backupDownloadURL(b.id)}
                      className="inline-flex items-center justify-center h-7 w-7 rounded-md border border-border hover:bg-card"
                      title="Download"
                    >
                      <Download className="size-3.5" />
                    </a>
                  )}
                  <Button
                    onClick={() => onDelete(b.id)}
                    variant="outline"
                    size="sm"
                    className="h-7 w-7 p-0"
                    title="Delete"
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function StatusBadge({ status }: { status: Backup['status'] }) {
  const map: Record<Backup['status'], 'default' | 'success' | 'warning' | 'danger' | 'muted'> = {
    pending: 'warning',
    running: 'warning',
    succeeded: 'success',
    failed: 'danger',
    deleted: 'muted',
  }
  return <Badge variant={map[status]}>{status}</Badge>
}

// ── Schedules tab ────────────────────────────────────────────────

function SchedulesTab() {
  const [rows, setRows] = useState<Schedule[] | null>(null)
  const [targets, setTargets] = useState<TargetSpec[]>([])
  const [open, setOpen] = useState(false)

  async function refresh() {
    try {
      const [scheds, tgts] = await Promise.all([listSchedules(), listTargets()])
      setRows(scheds)
      setTargets(tgts)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Failed to load schedules', { description: msg })
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  async function onDelete(id: string) {
    if (!window.confirm(`Delete schedule ${id}?`)) return
    try {
      await deleteSchedule(id)
      toast.success('Schedule deleted')
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Delete failed', { description: msg })
    }
  }

  async function toggle(id: string, enabled: boolean) {
    try {
      await updateSchedule(id, { enabled })
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Toggle failed', { description: msg })
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <p className="text-[12px] text-muted-foreground">
          Recurring backups. The scheduler polls every 30s and runs the oldest due schedule.
        </p>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm" className="h-8">
              <Plus className="size-3.5 mr-1.5" />
              New schedule
            </Button>
          </DialogTrigger>
          <NewScheduleDialog
            targets={targets}
            onCreated={async () => {
              setOpen(false)
              await refresh()
            }}
          />
        </Dialog>
      </div>
      {rows === null ? (
        <div className="text-muted-foreground text-sm">Loading…</div>
      ) : rows.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-6 text-center text-muted-foreground text-[13px]">
          No schedules. Add one to take automatic recurring backups.
        </div>
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-[12px]">
            <thead className="bg-card/50 text-muted-foreground">
              <tr className="text-left">
                <th className="px-3 py-2 font-medium">ID</th>
                <th className="px-3 py-2 font-medium">Target</th>
                <th className="px-3 py-2 font-medium">Interval</th>
                <th className="px-3 py-2 font-medium">Keep</th>
                <th className="px-3 py-2 font-medium">Next run</th>
                <th className="px-3 py-2 font-medium">Enabled</th>
                <th className="px-3 py-2 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((s) => (
                <tr key={s.id} className="border-t border-border/60">
                  <td className="px-3 py-2 font-mono text-[11px]">{s.id}</td>
                  <td className="px-3 py-2">{s.target_id}</td>
                  <td className="px-3 py-2 text-muted-foreground">
                    {formatInterval(s.interval_sec)}
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">
                    {s.retention} backups
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">
                    {formatRelative(s.next_run_at)}
                  </td>
                  <td className="px-3 py-2">
                    <Switch
                      checked={s.enabled}
                      onCheckedChange={(v) => toggle(s.id, v)}
                      className="scale-75"
                    />
                  </td>
                  <td className="px-3 py-2 text-right">
                    <Button
                      onClick={() => onDelete(s.id)}
                      variant="outline"
                      size="sm"
                      className="h-7 w-7 p-0"
                      title="Delete"
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function NewScheduleDialog({
  targets,
  onCreated,
}: {
  targets: TargetSpec[]
  onCreated: () => void | Promise<void>
}) {
  const enabled = targets.filter((t) => t.enabled)
  const [targetId, setTargetId] = useState(enabled[0]?.id ?? '')
  const [hours, setHours] = useState('24')
  const [retention, setRetention] = useState('7')
  const [enabledFlag, setEnabledFlag] = useState(true)
  const [busy, setBusy] = useState(false)

  async function submit() {
    setBusy(true)
    try {
      const intervalSec = Math.max(60, Math.round(Number(hours) * 3600))
      await createSchedule({
        targetId,
        intervalSec,
        retention: Math.max(0, Number(retention)),
        enabled: enabledFlag,
      })
      toast.success('Schedule created')
      await onCreated()
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error('Create failed', { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>New backup schedule</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">Target</Label>
          <select
            value={targetId}
            onChange={(e) => setTargetId(e.target.value)}
            className="h-8 px-2 rounded-md border border-border bg-card text-[12px]"
          >
            {enabled.map((t) => (
              <option key={t.id} value={t.id}>
                {t.id} ({t.kind})
              </option>
            ))}
          </select>
        </div>
        <div className="flex gap-3">
          <div className="flex-1 flex flex-col gap-1.5">
            <Label className="text-[12px]">Every (hours)</Label>
            <Input
              type="number"
              min="0.1"
              step="0.1"
              value={hours}
              onChange={(e) => setHours(e.target.value)}
              className="h-8"
            />
          </div>
          <div className="flex-1 flex flex-col gap-1.5">
            <Label className="text-[12px]">Keep last N</Label>
            <Input
              type="number"
              min="0"
              step="1"
              value={retention}
              onChange={(e) => setRetention(e.target.value)}
              className="h-8"
            />
          </div>
        </div>
        <label className="flex items-center gap-2 text-[12px]">
          <Switch
            checked={enabledFlag}
            onCheckedChange={setEnabledFlag}
            className="scale-75"
          />
          Enable immediately
        </label>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy || !targetId}>
          {busy ? 'Creating…' : 'Create'}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── Targets tab ──────────────────────────────────────────────────

function TargetsTab() {
  const [rows, setRows] = useState<TargetSpec[] | null>(null)
  const [open, setOpen] = useState(false)
  const [testing, setTesting] = useState<string | null>(null)

  async function refresh() {
    try {
      setRows(await listTargets())
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Failed to list targets', { description: msg })
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  async function onDelete(id: string) {
    if (!window.confirm(`Delete target ${id}? Schedules referencing it will block the delete.`)) {
      return
    }
    try {
      await deleteTarget(id)
      toast.success('Target deleted')
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Delete failed', { description: msg })
    }
  }

  async function onTest(id: string) {
    setTesting(id)
    try {
      const res = await testTarget(id)
      if (res.ok) {
        toast.success('Connection OK', { description: id })
      } else {
        toast.error('Connection failed', { description: res.error })
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Test failed', { description: msg })
    } finally {
      setTesting(null)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <p className="text-[12px] text-muted-foreground">
          Storage destinations. v1 supports <code>local</code> (disk
          on the opendray host) and <code>smb</code> (any SMB / CIFS
          share, e.g. UNAS or Synology).
        </p>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm" className="h-8">
              <Plus className="size-3.5 mr-1.5" />
              New target
            </Button>
          </DialogTrigger>
          <NewTargetDialog
            onCreated={async () => {
              setOpen(false)
              await refresh()
            }}
          />
        </Dialog>
      </div>
      {rows === null ? (
        <div className="text-muted-foreground text-sm">Loading…</div>
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-[12px]">
            <thead className="bg-card/50 text-muted-foreground">
              <tr className="text-left">
                <th className="px-3 py-2 font-medium">ID</th>
                <th className="px-3 py-2 font-medium">Kind</th>
                <th className="px-3 py-2 font-medium">Config</th>
                <th className="px-3 py-2 font-medium">Enabled</th>
                <th className="px-3 py-2 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((t) => (
                <tr key={t.id} className="border-t border-border/60">
                  <td className="px-3 py-2 font-mono text-[11px]">{t.id}</td>
                  <td className="px-3 py-2">
                    <Badge variant="outline">{t.kind}</Badge>
                  </td>
                  <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">
                    {targetSummary(t)}
                  </td>
                  <td className="px-3 py-2">
                    {t.enabled ? (
                      <Badge variant="success">on</Badge>
                    ) : (
                      <Badge variant="muted">off</Badge>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="inline-flex gap-1">
                      <Button
                        onClick={() => onTest(t.id)}
                        variant="outline"
                        size="sm"
                        className="h-7 text-[11px]"
                        disabled={testing === t.id}
                      >
                        {testing === t.id ? 'Testing…' : 'Test'}
                      </Button>
                      <Button
                        onClick={() => onDelete(t.id)}
                        variant="outline"
                        size="sm"
                        className="h-7 w-7 p-0"
                        title="Delete"
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function targetSummary(t: TargetSpec): string {
  if (t.kind === 'local') {
    return String(t.config?.root ?? '(default local dir)')
  }
  if (t.kind === 'smb') {
    const host = t.config?.host
    const share = t.config?.share
    const user = t.config?.user
    const prefix = t.config?.path_prefix
    return `//${host ?? '?'}/${share ?? '?'} as ${user ?? '?'}${prefix ? ` → ${prefix}/` : ''}`
  }
  return JSON.stringify(t.config)
}

function NewTargetDialog({
  onCreated,
}: {
  onCreated: () => void | Promise<void>
}) {
  const [kind, setKind] = useState<TargetKind>('smb')
  const [id, setId] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [busy, setBusy] = useState(false)
  // local-specific
  const [localRoot, setLocalRoot] = useState('')
  // smb-specific
  const [host, setHost] = useState('')
  const [port, setPort] = useState('445')
  const [share, setShare] = useState('')
  const [user, setUser] = useState('')
  const [password, setPassword] = useState('')
  const [pathPrefix, setPathPrefix] = useState('')

  async function submit() {
    setBusy(true)
    try {
      const config: Record<string, unknown> =
        kind === 'local'
          ? localRoot
            ? { root: localRoot }
            : {}
          : {
              host,
              port: Number(port) || 445,
              share,
              user,
              password,
              path_prefix: pathPrefix || undefined,
            }
      await createTarget({
        id: id || undefined,
        kind,
        config,
        enabled,
      })
      toast.success('Target created')
      await onCreated()
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error('Create failed', { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>New backup target</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">Kind</Label>
          <select
            value={kind}
            onChange={(e) => setKind(e.target.value as TargetKind)}
            className="h-8 px-2 rounded-md border border-border bg-card text-[12px]"
          >
            <option value="local">local (disk on opendray host)</option>
            <option value="smb">smb (CIFS share)</option>
          </select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">ID (optional)</Label>
          <Input
            value={id}
            onChange={(e) => setId(e.target.value)}
            placeholder="auto-generated if blank"
            className="h-8"
          />
        </div>

        {kind === 'local' && (
          <div className="flex flex-col gap-1.5">
            <Label className="text-[12px]">Root directory</Label>
            <Input
              value={localRoot}
              onChange={(e) => setLocalRoot(e.target.value)}
              placeholder="leave blank to use cfg.backup.local_dir"
              className="h-8"
            />
          </div>
        )}

        {kind === 'smb' && (
          <>
            <div className="flex gap-2">
              <div className="flex-1 flex flex-col gap-1.5">
                <Label className="text-[12px]">Host</Label>
                <Input
                  value={host}
                  onChange={(e) => setHost(e.target.value)}
                  placeholder="192.168.9.8"
                  className="h-8"
                />
              </div>
              <div className="w-24 flex flex-col gap-1.5">
                <Label className="text-[12px]">Port</Label>
                <Input
                  value={port}
                  onChange={(e) => setPort(e.target.value)}
                  className="h-8"
                />
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-[12px]">Share</Label>
              <Input
                value={share}
                onChange={(e) => setShare(e.target.value)}
                placeholder="Claude_Workspace"
                className="h-8"
              />
            </div>
            <div className="flex gap-2">
              <div className="flex-1 flex flex-col gap-1.5">
                <Label className="text-[12px]">User</Label>
                <Input
                  value={user}
                  onChange={(e) => setUser(e.target.value)}
                  className="h-8"
                />
              </div>
              <div className="flex-1 flex flex-col gap-1.5">
                <Label className="text-[12px]">Password</Label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="h-8"
                />
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-[12px]">Path prefix (optional)</Label>
              <Input
                value={pathPrefix}
                onChange={(e) => setPathPrefix(e.target.value)}
                placeholder="opendray/backups"
                className="h-8"
              />
            </div>
          </>
        )}

        <label className="flex items-center gap-2 text-[12px]">
          <Switch
            checked={enabled}
            onCheckedChange={setEnabled}
            className="scale-75"
          />
          Enable immediately
        </label>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          {busy ? 'Creating…' : 'Create'}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── helpers ──────────────────────────────────────────────────────

function formatRelative(iso: string): string {
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return iso
  const diff = Date.now() - t
  if (diff < 0) {
    const inSec = Math.round(-diff / 1000)
    if (inSec < 60) return `in ${inSec}s`
    if (inSec < 3600) return `in ${Math.round(inSec / 60)}m`
    if (inSec < 86400) return `in ${Math.round(inSec / 3600)}h`
    return `in ${Math.round(inSec / 86400)}d`
  }
  const sec = Math.round(diff / 1000)
  if (sec < 60) return `${sec}s ago`
  if (sec < 3600) return `${Math.round(sec / 60)}m ago`
  if (sec < 86400) return `${Math.round(sec / 3600)}h ago`
  return `${Math.round(sec / 86400)}d ago`
}

function msgFromAPI(err: APIError): string {
  if (
    err.body &&
    typeof err.body === 'object' &&
    'error' in err.body &&
    typeof (err.body as { error: unknown }).error === 'string'
  ) {
    return (err.body as { error: string }).error
  }
  return err.message
}
