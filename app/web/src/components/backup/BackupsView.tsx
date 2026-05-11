import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  ChevronDown,
  ChevronRight,
  Copy,
  Dice5,
  Download,
  HardDrive,
  KeyRound,
  Lock,
  Package,
  Play,
  Plus,
  RefreshCw,
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
  type BackupSetupResult,
  type BackupStatusReport,
  type InventoryGroup,
  type Schedule,
  type TargetSpec,
  backupDownloadURL,
  createBackup,
  createSchedule,
  deleteBackup,
  deleteSchedule,
  deleteTarget,
  fetchBackupInventory,
  fetchBackupStatus,
  formatBytes,
  formatInterval,
  listBackups,
  listSchedules,
  listTargets,
  postBackupSetup,
  restoreBackup,
  testTarget,
  updateSchedule,
} from '@/lib/backup'
import { TargetEditor, targetSummary } from './TargetEditor'
import { APIError } from '@/lib/api'
import { cn } from '@/lib/utils'

export function BackupsView() {
  const [status, setStatus] = useState<BackupStatusReport | null>(null)

  // refresh is exposed to the Setup/Restart child views so they can
  // trigger a re-fetch after writing the key file or restarting the
  // gateway, without parent/child plumbing.
  async function refresh() {
    try {
      const next = await fetchBackupStatus()
      setStatus(next)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Failed to load backup status', { description: msg })
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  if (status === null) {
    return <div className="text-muted-foreground text-sm">Loading…</div>
  }

  // Three-state machine based on the always-200 status payload:
  //
  //   enabled=true                          → live dashboard
  //   enabled=false, requires_restart=true  → restart prompt (key file written, awaiting bounce)
  //   enabled=false, requires_restart=false → first-time setup wizard
  //
  // The boolean fields are computed server-side so we don't have
  // to repeat the env-vs-file decision tree here.
  if (!status.enabled) {
    if (status.requires_restart) {
      return <RestartRequiredCard status={status} onRecheck={refresh} />
    }
    return <SetupWizardCard status={status} onComplete={refresh} />
  }

  return (
    <div className="flex flex-col gap-5">
      <StatusBanner status={status} />
      <InventoryCard />
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

// ── Inventory card (what does a backup contain right now?) ──────

function InventoryCard() {
  const [open, setOpen] = useState(false)
  const [groups, setGroups] = useState<InventoryGroup[] | null>(null)
  const [loading, setLoading] = useState(false)

  async function load() {
    if (groups !== null || loading) return
    setLoading(true)
    try {
      setGroups(await fetchBackupInventory())
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Failed to load inventory', { description: msg })
    } finally {
      setLoading(false)
    }
  }

  function toggle() {
    setOpen((v) => !v)
    void load()
  }

  const totalRows = groups
    ? groups.reduce(
        (acc, g) => acc + g.tables.reduce((a, t) => a + t.count, 0),
        0,
      )
    : 0

  return (
    <div className="rounded-md border border-border bg-card/30">
      <button
        type="button"
        onClick={toggle}
        className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-card/60 transition-colors"
      >
        {open ? (
          <ChevronDown className="size-3.5 text-muted-foreground" />
        ) : (
          <ChevronRight className="size-3.5 text-muted-foreground" />
        )}
        <Package className="size-3.5 text-accent" />
        <span className="text-[13px] font-medium">What's in a backup?</span>
        {groups && (
          <span className="text-[11px] text-muted-foreground ml-1">
            {totalRows.toLocaleString()} rows across{' '}
            {groups.reduce((a, g) => a + g.tables.length, 0)} tables
          </span>
        )}
      </button>
      {open && (
        <div className="px-4 pb-4 pt-1 border-t border-border/50 flex flex-col gap-3">
          <p className="text-[12px] text-muted-foreground">
            Each backup is a <code>pg_dump --format=custom</code> of every
            table below, plus <code>manifest.json</code> and (optionally){' '}
            <code>config.toml</code>. Counts are live; the bundle captures
            whatever's there at backup time.
          </p>
          {loading && (
            <div className="text-muted-foreground text-[12px]">Loading…</div>
          )}
          {groups?.map((g) => (
            <div key={g.id} className="flex flex-col gap-1.5">
              <div className="flex items-baseline gap-2">
                <h4 className="text-[12px] font-semibold">{g.label}</h4>
                <span className="text-[11px] text-muted-foreground">
                  {g.tables.reduce((a, t) => a + t.count, 0).toLocaleString()}{' '}
                  rows
                </span>
              </div>
              <p className="text-[11px] text-muted-foreground">
                {g.description}
              </p>
              <div className="flex flex-wrap gap-1.5 mt-0.5">
                {g.tables.map((t) => (
                  <span
                    key={t.name}
                    className="inline-flex items-baseline gap-1.5 px-2 py-0.5 rounded border border-border bg-card text-[11px]"
                  >
                    <code className="text-foreground">{t.name}</code>
                    <span className="text-muted-foreground">
                      {t.count.toLocaleString()}
                    </span>
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ── Setup wizard + restart prompt ────────────────────────────────

// Rendered when the operator wrote a key file (via this UI or a
// prior install) but the gateway hasn't loaded it yet — i.e. the
// service refuses to fail-and-restart the running process, so the
// only path to live-backup is a manual bounce. After the operator
// restarts opendray, the Check-again button picks up the new state.
function RestartRequiredCard({
  status,
  onRecheck,
}: {
  status: BackupStatusReport
  onRecheck: () => void | Promise<void>
}) {
  const [busy, setBusy] = useState(false)
  async function recheck() {
    setBusy(true)
    try {
      await onRecheck()
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="rounded-md border border-accent/30 bg-accent/5 p-5">
      <div className="flex items-start gap-3">
        <RefreshCw className="size-5 mt-0.5 text-accent" />
        <div className="flex-1">
          <div className="font-medium">
            Restart opendray to activate backups
          </div>
          <div className="text-muted-foreground text-[13px] mt-1">
            Your passphrase is saved. The gateway only loads it at startup,
            so the feature stays off until you bounce the process.
          </div>
          {status.configured_via === 'file' && status.key_file_path && (
            <div className="mt-3 text-[12px]">
              <span className="text-muted-foreground">Key file:</span>{' '}
              <code className="text-foreground">{status.key_file_path}</code>
            </div>
          )}
          {status.configured_via === 'env' && (
            <div className="mt-3 text-[12px]">
              <span className="text-muted-foreground">Configured via:</span>{' '}
              <code className="text-foreground">OPENDRAY_BACKUP_KEY</code> env var
            </div>
          )}
          <div className="mt-4 flex gap-2">
            <Button size="sm" onClick={() => void recheck()} disabled={busy}>
              <RefreshCw className={cn('size-3.5', busy && 'animate-spin')} />
              Check again
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

// First-time setup wizard. Mirrors the mobile flow — Generate
// (server picks random key, returns once) or Paste (operator
// supplies). On submit the server writes ~/.opendray/secrets/
// backup.key (0600); the operator restarts the gateway to pick it
// up; the parent re-fetches status and transitions to either
// RestartRequiredCard (the natural next step) or directly to the
// live dashboard if the operator was fast enough to restart
// concurrently.
function SetupWizardCard({
  status,
  onComplete,
}: {
  status: BackupStatusReport
  onComplete: () => void | Promise<void>
}) {
  const [mode, setMode] = useState<'generate' | 'paste'>('generate')
  const [pasted, setPasted] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Result of a successful generate call — must be shown once and
  // the operator must acknowledge they saved it before we
  // transition to the next step.
  const [generated, setGenerated] = useState<BackupSetupResult | null>(null)
  const [ackSaved, setAckSaved] = useState(false)

  async function submit() {
    setError(null)
    setBusy(true)
    try {
      const result = await postBackupSetup(
        mode === 'generate'
          ? { mode: 'generate' }
          : { mode: 'paste', passphrase: pasted.trim() },
      )
      if (result.passphrase) {
        // Generate path — show the passphrase for save confirm
        // before triggering parent refresh.
        setGenerated(result)
      } else {
        // Paste path — caller already knows their passphrase,
        // skip the confirm step.
        await onComplete()
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      setError(msg)
    } finally {
      setBusy(false)
    }
  }

  if (generated) {
    return (
      <GeneratedPassphrasePanel
        result={generated}
        ackSaved={ackSaved}
        setAckSaved={setAckSaved}
        onContinue={() => void onComplete()}
      />
    )
  }

  return (
    <div className="rounded-md border border-border bg-card p-5">
      <div className="flex items-center gap-2">
        <Lock className="size-5 text-accent" />
        <div className="font-medium">Set up backups</div>
      </div>
      <div className="text-muted-foreground text-[13px] mt-2">
        Choose a master passphrase. opendray uses it to encrypt every backup
        blob. <strong className="text-foreground">Lose it and your backups
        become unrecoverable</strong>, so save it in a password manager
        (Vaultwarden, 1Password, …) before continuing.
      </div>

      <div className="mt-4 flex gap-2">
        <Button
          variant={mode === 'generate' ? 'default' : 'outline'}
          size="sm"
          onClick={() => setMode('generate')}
        >
          <Dice5 className="size-3.5" />
          Generate
        </Button>
        <Button
          variant={mode === 'paste' ? 'default' : 'outline'}
          size="sm"
          onClick={() => setMode('paste')}
        >
          <KeyRound className="size-3.5" />
          Paste my own
        </Button>
      </div>

      {mode === 'generate' ? (
        <div className="mt-4 rounded-md border border-border bg-input/20 p-3 text-[12px]">
          <div className="font-medium">256-bit random key</div>
          <div className="text-muted-foreground mt-1">
            Server generates a cryptographically random passphrase and shows
            it once. You must copy it before continuing — there is no
            recovery path.
          </div>
        </div>
      ) : (
        <div className="mt-4">
          <Label htmlFor="paste" className="text-[12px]">
            Your passphrase
          </Label>
          <Input
            id="paste"
            value={pasted}
            onChange={(e) => setPasted(e.target.value)}
            placeholder="At least 20 characters"
            className="mt-1 font-mono text-[13px]"
            autoFocus
          />
          <div className="text-muted-foreground text-[11px] mt-1">
            Recommended: 40+ characters from a password manager.
          </div>
        </div>
      )}

      {error && (
        <div className="mt-3 rounded-md border border-state-failed/40 bg-state-failed/10 p-2 text-[12px] text-state-failed">
          {error}
        </div>
      )}

      <div className="mt-4 flex items-center justify-between gap-3">
        {status.key_file_path && (
          <div className="text-muted-foreground text-[11px] truncate">
            Saves to: <code>{status.key_file_path}</code>
          </div>
        )}
        <Button
          size="sm"
          onClick={() => void submit()}
          disabled={busy || (mode === 'paste' && pasted.trim().length < 20)}
        >
          {busy ? 'Saving…' : mode === 'generate' ? 'Generate and save' : 'Save'}
        </Button>
      </div>
    </div>
  )
}

function GeneratedPassphrasePanel({
  result,
  ackSaved,
  setAckSaved,
  onContinue,
}: {
  result: BackupSetupResult
  ackSaved: boolean
  setAckSaved: (v: boolean) => void
  onContinue: () => void
}) {
  const pass = result.passphrase ?? ''
  async function copy() {
    try {
      await navigator.clipboard.writeText(pass)
      toast.success('Passphrase copied to clipboard')
    } catch {
      toast.error('Copy failed — select and copy manually')
    }
  }
  return (
    <div className="rounded-md border border-amber-500/40 bg-amber-500/5 p-5">
      <div className="flex items-center gap-2">
        <ShieldAlert className="size-5 text-amber-500" />
        <div className="font-medium">Save this passphrase NOW</div>
      </div>
      <div className="text-muted-foreground text-[13px] mt-2">
        This is shown <strong className="text-foreground">once</strong>. It
        will not be retrievable from opendray or anywhere else. Copy it into
        a password manager before continuing.
      </div>
      <div className="mt-4 rounded-md border border-accent/40 bg-input/30 p-3 font-mono text-[13px] break-all select-all">
        {pass}
      </div>
      <div className="mt-2 flex gap-2">
        <Button variant="outline" size="sm" onClick={() => void copy()}>
          <Copy className="size-3.5" />
          Copy
        </Button>
      </div>
      {result.key_file_path && (
        <div className="text-muted-foreground text-[11px] mt-3">
          Saved to: <code>{result.key_file_path}</code>
        </div>
      )}
      <label className="mt-4 flex items-start gap-2 text-[13px] cursor-pointer">
        <input
          type="checkbox"
          checked={ackSaved}
          onChange={(e) => setAckSaved(e.target.checked)}
          className="mt-0.5"
        />
        <span>I have saved this passphrase to my password manager</span>
      </label>
      <div className="mt-4">
        <Button size="sm" onClick={onContinue} disabled={!ackSaved}>
          Continue
        </Button>
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
          <TargetEditor
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
