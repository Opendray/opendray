import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'
import { copyText } from '@/lib/clipboard'
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
  ShieldCheck,
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
  type BackupHealth,
  type BackupKind,
  type BackupSetupResult,
  type BackupStatusReport,
  type InventoryGroup,
  type RestorePlan,
  type Schedule,
  type TargetSpec,
  type TriggeredBy,
  backupDownloadURL,
  createBackup,
  createSchedule,
  deleteBackup,
  deleteSchedule,
  deleteTarget,
  fetchBackupHealth,
  fetchBackupInventory,
  fetchBackupStatus,
  fetchRecoveryKit,
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
  const { t } = useTranslation()
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
      toast.error(t('web.backups.loadStatusFailedToast'), { description: msg })
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  if (status === null) {
    return <div className="text-muted-foreground text-sm">{t('web.backups.loading')}</div>
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
      <HealthStrip />
      <StatusBanner status={status} />
      <InventoryCard />
      <Tabs defaultValue="backups" className="w-full">
        <TabsList>
          <TabsTrigger value="backups">{t('web.backups.tabs.backups')}</TabsTrigger>
          <TabsTrigger value="schedules">{t('web.backups.tabs.schedules')}</TabsTrigger>
          <TabsTrigger value="targets">{t('web.backups.tabs.targets')}</TabsTrigger>
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
  const { t } = useTranslation()
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
      toast.error(t('web.backups.inventory.loadFailedToast'), { description: msg })
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
        <span className="text-[13px] font-medium">{t('web.backups.inventory.title')}</span>
        {groups && (
          <span className="text-[11px] text-muted-foreground ml-1">
            {t('web.backups.inventory.summary', {
              rows: totalRows.toLocaleString(),
              tables: groups.reduce((a, g) => a + g.tables.length, 0),
            })}
          </span>
        )}
      </button>
      {open && (
        <div className="px-4 pb-4 pt-1 border-t border-border/50 flex flex-col gap-3">
          <p className="text-[12px] text-muted-foreground">
            <Trans
              i18nKey="web.backups.inventory.description"
              components={{ 1: <code />, 3: <code />, 5: <code /> }}
            />
          </p>
          {loading && (
            <div className="text-muted-foreground text-[12px]">{t('web.backups.loading')}</div>
          )}
          {groups?.map((g) => (
            <div key={g.id} className="flex flex-col gap-1.5">
              <div className="flex items-baseline gap-2">
                <h4 className="text-[12px] font-semibold">{g.label}</h4>
                <span className="text-[11px] text-muted-foreground">
                  {g.tables
                    .reduce((a, tbl) => a + tbl.count, 0)
                    .toLocaleString()}{' '}
                  {t('web.backups.inventory.rowsLabel')}
                </span>
              </div>
              <p className="text-[11px] text-muted-foreground">
                {g.description}
              </p>
              <div className="flex flex-wrap gap-1.5 mt-0.5">
                {g.tables.map((tbl) => (
                  <span
                    key={tbl.name}
                    className="inline-flex items-baseline gap-1.5 px-2 py-0.5 rounded border border-border bg-card text-[11px]"
                  >
                    <code className="text-foreground">{tbl.name}</code>
                    <span className="text-muted-foreground">
                      {tbl.count.toLocaleString()}
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
  const { t } = useTranslation()
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
          <div className="font-medium">{t('web.backups.restart.title')}</div>
          <div className="text-muted-foreground text-[13px] mt-1">
            {t('web.backups.restart.description')}
          </div>
          {status.configured_via === 'file' && status.key_file_path && (
            <div className="mt-3 text-[12px]">
              <span className="text-muted-foreground">{t('web.backups.restart.keyFile')}</span>{' '}
              <code className="text-foreground">{status.key_file_path}</code>
            </div>
          )}
          {status.configured_via === 'env' && (
            <div className="mt-3 text-[12px]">
              <span className="text-muted-foreground">{t('web.backups.restart.configuredVia')}</span>{' '}
              <code className="text-foreground">{t('web.backups.restart.envVar')}</code>
            </div>
          )}
          <div className="mt-4 flex gap-2">
            <Button size="sm" onClick={() => void recheck()} disabled={busy}>
              <RefreshCw className={cn('size-3.5', busy && 'animate-spin')} />
              {t('web.backups.restart.checkAgain')}
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
  const { t } = useTranslation()
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
        <div className="font-medium">{t('web.backups.setup.title')}</div>
      </div>
      <div className="text-muted-foreground text-[13px] mt-2">
        <Trans
          i18nKey="web.backups.setup.description"
          components={{ 1: <strong className="text-foreground" /> }}
        />
      </div>

      <div className="mt-4 flex gap-2">
        <Button
          variant={mode === 'generate' ? 'default' : 'outline'}
          size="sm"
          onClick={() => setMode('generate')}
        >
          <Dice5 className="size-3.5" />
          {t('web.backups.setup.generate')}
        </Button>
        <Button
          variant={mode === 'paste' ? 'default' : 'outline'}
          size="sm"
          onClick={() => setMode('paste')}
        >
          <KeyRound className="size-3.5" />
          {t('web.backups.setup.pasteOwn')}
        </Button>
      </div>

      {mode === 'generate' ? (
        <div className="mt-4 rounded-md border border-border bg-input/20 p-3 text-[12px]">
          <div className="font-medium">{t('web.backups.setup.generateTitle')}</div>
          <div className="text-muted-foreground mt-1">
            {t('web.backups.setup.generateHint')}
          </div>
        </div>
      ) : (
        <div className="mt-4">
          <Label htmlFor="paste" className="text-[12px]">
            {t('web.backups.setup.pasteLabel')}
          </Label>
          <Input
            id="paste"
            value={pasted}
            onChange={(e) => setPasted(e.target.value)}
            placeholder={t('web.backups.setup.pastePlaceholder')}
            className="mt-1 font-mono text-[13px]"
            autoFocus
          />
          <div className="text-muted-foreground text-[11px] mt-1">
            {t('web.backups.setup.pasteHint')}
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
            {t('web.backups.setup.savesTo')} <code>{status.key_file_path}</code>
          </div>
        )}
        <Button
          size="sm"
          onClick={() => void submit()}
          disabled={busy || (mode === 'paste' && pasted.trim().length < 20)}
        >
          {busy
            ? t('web.backups.setup.saving')
            : mode === 'generate'
              ? t('web.backups.setup.generateAndSave')
              : t('web.backups.setup.save')}
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
  const { t } = useTranslation()
  const pass = result.passphrase ?? ''
  async function copy() {
    try {
      if (!(await copyText(pass))) throw new Error('clipboard unavailable')
      toast.success(t('web.backups.generated.copiedToast'))
    } catch {
      toast.error(t('web.backups.generated.copyFailedToast'))
    }
  }
  return (
    <div className="rounded-md border border-amber-500/40 bg-amber-500/5 p-5">
      <div className="flex items-center gap-2">
        <ShieldAlert className="size-5 text-amber-500" />
        <div className="font-medium">{t('web.backups.generated.title')}</div>
      </div>
      <div className="text-muted-foreground text-[13px] mt-2">
        <Trans
          i18nKey="web.backups.generated.description"
          components={{ 1: <strong className="text-foreground" /> }}
        />
      </div>
      <div className="mt-4 rounded-md border border-accent/40 bg-input/30 p-3 font-mono text-[13px] break-all select-all">
        {pass}
      </div>
      <div className="mt-2 flex gap-2">
        <Button variant="outline" size="sm" onClick={() => void copy()}>
          <Copy className="size-3.5" />
          {t('web.backups.generated.copy')}
        </Button>
      </div>
      {result.key_file_path && (
        <div className="text-muted-foreground text-[11px] mt-3">
          {t('web.backups.generated.savedTo')} <code>{result.key_file_path}</code>
        </div>
      )}
      <label className="mt-4 flex items-start gap-2 text-[13px] cursor-pointer">
        <input
          type="checkbox"
          checked={ackSaved}
          onChange={(e) => setAckSaved(e.target.checked)}
          className="mt-0.5"
        />
        <span>{t('web.backups.generated.ack')}</span>
      </label>
      <div className="mt-4">
        <Button size="sm" onClick={onContinue} disabled={!ackSaved}>
          {t('web.backups.generated.continue')}
        </Button>
      </div>
    </div>
  )
}

// ── Health strip (at-a-glance: last good backup + what needs attention) ──

// HealthStrip is the dashboard's first line: when the last good backup
// landed and a count of anything currently demanding attention (recent
// failures, failed restore-verifications, overdue schedules). Green and
// quiet when all is well, red when not, neutral before the first backup.
function HealthStrip() {
  const { t } = useTranslation()
  const [health, setHealth] = useState<BackupHealth | null>(null)

  async function refresh() {
    try {
      setHealth(await fetchBackupHealth())
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.health.loadFailedToast'), { description: msg })
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  if (health === null) return null

  const issues =
    health.recent_failures +
    health.verify_failures +
    health.overdue_schedules
  const neverBackedUp = !health.last_success_at
  const allClear = issues === 0 && !neverBackedUp

  return (
    <div
      className={cn(
        'rounded-md border p-3 text-[12px]',
        allClear
          ? 'border-state-running/30 bg-state-running/10'
          : neverBackedUp
            ? 'border-border bg-card/30'
            : 'border-state-failed/30 bg-state-failed/10',
      )}
    >
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
        <div className="flex items-center gap-2">
          {allClear ? (
            <ShieldCheck className="size-3.5 text-state-running" />
          ) : (
            <ShieldAlert
              className={cn(
                'size-3.5',
                neverBackedUp ? 'text-muted-foreground' : 'text-state-failed',
              )}
            />
          )}
          <span className="text-muted-foreground">
            {t('web.backups.health.lastSuccess')}
          </span>
          {health.last_success_at ? (
            <code
              className="text-foreground"
              title={new Date(health.last_success_at).toLocaleString()}
            >
              {formatRelative(health.last_success_at)}
            </code>
          ) : (
            <span className="text-muted-foreground">
              {t('web.backups.health.never')}
            </span>
          )}
        </div>

        {health.recent_failures > 0 && (
          <span className="text-state-failed">
            {t('web.backups.health.recentFailures', {
              count: health.recent_failures,
            })}
          </span>
        )}
        {health.verify_failures > 0 && (
          <span className="text-state-failed">
            {t('web.backups.health.verifyFailures', {
              count: health.verify_failures,
            })}
          </span>
        )}
        {health.overdue_schedules > 0 && (
          <span className="text-state-failed">
            {t('web.backups.health.overdueSchedules', {
              count: health.overdue_schedules,
            })}
          </span>
        )}

        {health.schedules > 0 && (
          <div className="flex items-center gap-2 text-muted-foreground">
            <RotateCw className="size-3.5" />
            <span>
              {t('web.backups.health.scheduleSummary', {
                enabled: health.enabled_schedules,
                total: health.schedules,
              })}
            </span>
          </div>
        )}

        {allClear && (
          <span className="text-state-running">
            {t('web.backups.health.allClear')}
          </span>
        )}
      </div>
    </div>
  )
}

function StatusBanner({ status }: { status: BackupStatusReport }) {
  const { t } = useTranslation()
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
          <span className="text-muted-foreground">{t('web.backups.status.keyFingerprint')}</span>
          <code className="text-foreground">{status.key_fingerprint}</code>
        </div>
        <div className="flex items-center gap-2">
          <HardDrive className="size-3.5 text-accent" />
          <span className="text-muted-foreground">{t('web.backups.status.pgDump')}</span>
          {status.ok ? (
            <code className="text-foreground">{status.pg_dump_version}</code>
          ) : (
            <span className="text-state-failed">
              {status.pg_dump_error || t('web.backups.status.pgDumpUnavailable')}
            </span>
          )}
        </div>
      </div>
      {!status.ok && (
        <div className="mt-2 text-state-failed">
          <Trans
            i18nKey="web.backups.status.pgDumpHint"
            components={{ 1: <code />, 3: <code /> }}
          />
        </div>
      )}
    </div>
  )
}

// ── Backups tab ──────────────────────────────────────────────────

function BackupsTab() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<Backup[] | null>(null)
  const [busy, setBusy] = useState(false)
  const [includeConfig, setIncludeConfig] = useState(true)
  const [fullInstance, setFullInstance] = useState(false)
  const [restoreOpen, setRestoreOpen] = useState(false)
  const [kitOpen, setKitOpen] = useState(false)

  async function refresh() {
    try {
      const list = await listBackups({ limit: 50 })
      setRows(list)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.backupsTab.listFailedToast'), { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    const id = window.setInterval(refresh, 5000)
    return () => window.clearInterval(id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function trigger() {
    setBusy(true)
    try {
      await createBackup({
        kind: fullInstance ? 'full_instance' : 'db_only',
        includeConfig,
      })
      toast.success(t('web.backups.backupsTab.queuedToast'))
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.backupsTab.triggerFailedToast'), { description: msg })
    } finally {
      setBusy(false)
    }
  }

  async function onDelete(id: string) {
    if (
      !window.confirm(t('web.backups.backupsTab.deleteConfirm', { id }))
    ) {
      return
    }
    try {
      await deleteBackup(id)
      toast.success(t('web.backups.backupsTab.deletedToast'))
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.backupsTab.deleteFailedToast'), { description: msg })
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3 flex-wrap">
        <Button onClick={trigger} disabled={busy} size="sm" className="h-8">
          <Play className="size-3.5 mr-1.5" />
          {busy
            ? t('web.backups.backupsTab.triggering')
            : t('web.backups.backupsTab.backupNow')}
        </Button>
        <label
          className="flex items-center gap-2 text-[12px] text-muted-foreground"
          title={t('web.backups.backupsTab.fullInstanceHint')}
        >
          <Switch
            checked={fullInstance}
            onCheckedChange={setFullInstance}
            className="scale-75"
          />
          {t('web.backups.backupsTab.fullInstance')}
        </label>
        <label className="flex items-center gap-2 text-[12px] text-muted-foreground">
          <Switch
            checked={includeConfig || fullInstance}
            onCheckedChange={setIncludeConfig}
            disabled={fullInstance}
            className="scale-75"
          />
          {t('web.backups.backupsTab.includeConfig')}
        </label>
        <Dialog open={kitOpen} onOpenChange={setKitOpen}>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm" className="h-8">
              <KeyRound className="size-3.5 mr-1.5" />
              {t('web.backups.recoveryKit.button')}
            </Button>
          </DialogTrigger>
          <RecoveryKitDialog />
        </Dialog>
        <Dialog open={restoreOpen} onOpenChange={setRestoreOpen}>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm" className="h-8 ml-auto">
              <Upload className="size-3.5 mr-1.5" />
              {t('web.backups.backupsTab.restoreFromFile')}
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
          {t('web.backups.backupsTab.refresh')}
        </Button>
      </div>
      <BackupTable rows={rows} onDelete={onDelete} />
    </div>
  )
}

function RestoreDialog({ onDone }: { onDone: () => void | Promise<void> }) {
  const { t } = useTranslation()
  const [file, setFile] = useState<File | null>(null)
  const [targetDsn, setTargetDsn] = useState('')
  const [clean, setClean] = useState(true)
  const [confirm, setConfirm] = useState('')
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [output, setOutput] = useState<string | null>(null)
  const [plan, setPlan] = useState<RestorePlan | null>(null)

  const restoringOwn = targetDsn === ''
  const confirmReady =
    !restoringOwn ||
    confirm.trim() === t('web.backups.restore.confirmSentinel')

  function restoreErr(err: unknown): string {
    return err instanceof APIError
      ? msgFromAPI(err)
      : err instanceof Error
        ? err.message
        : 'Unknown error'
  }

  // Step 1: dry run — validates the bundle and reports a plan; changes
  // nothing on disk or in the database.
  async function preview() {
    if (!file) {
      toast.error(t('web.backups.restore.pickFileToast'))
      return
    }
    setBusy(true)
    setOutput(null)
    setPlan(null)
    try {
      const res = await restoreBackup({
        bundle: file,
        targetDsn: targetDsn || undefined,
        clean,
        apply: false,
      })
      setPlan(res.plan)
      toast.success(t('web.backups.restore.dryRunToast'))
    } catch (err) {
      toast.error(t('web.backups.restore.failedToast'), {
        description: restoreErr(err),
      })
    } finally {
      setBusy(false)
    }
  }

  // Step 2: apply — commits the restore (safety snapshot + write + pg_restore).
  async function apply() {
    if (!file) return
    setBusy(true)
    try {
      const res = await restoreBackup({
        bundle: file,
        targetDsn: targetDsn || undefined,
        clean,
        apply: true,
        confirm: restoringOwn ? confirm : undefined,
        note,
      })
      setOutput(res.pg_restore_output || t('web.backups.restore.noPgRestoreOutput'))
      toast.success(t('web.backups.restore.succeededToast'), {
        description: t('web.backups.restore.replayedDescription', {
          bytes: formatBytes(res.bytes_read),
          id: res.manifest.backup_id,
        }),
      })
      await onDone()
    } catch (err) {
      toast.error(t('web.backups.restore.failedToast'), {
        description: restoreErr(err),
      })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{t('web.backups.restore.title')}</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">{t('web.backups.restore.bundleLabel')}</Label>
          <input
            type="file"
            accept=".enc,.tar.gz.enc,application/octet-stream"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            className="text-[12px]"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">
            {t('web.backups.restore.targetDsnLabel')}
            <span className="text-muted-foreground ml-1 text-[11px]">
              {t('web.backups.restore.targetDsnHint')}
            </span>
          </Label>
          <Input
            value={targetDsn}
            onChange={(e) => setTargetDsn(e.target.value)}
            placeholder={t('web.backups.restore.targetDsnPlaceholder')}
            className="h-8 font-mono text-[11px]"
          />
        </div>
        <label className="flex items-center gap-2 text-[12px]">
          <Switch
            checked={clean}
            onCheckedChange={setClean}
            className="scale-75"
          />
          {t('web.backups.restore.cleanLabel')}
        </label>
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">{t('web.backups.restore.auditNoteLabel')}</Label>
          <Input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={t('web.backups.restore.auditNotePlaceholder')}
            className="h-8"
          />
        </div>

        {restoringOwn && (
          <div className="rounded-md border border-state-failed/40 bg-state-failed/10 p-3 text-[12px] flex gap-2 items-start">
            <ShieldAlert className="size-4 text-state-failed shrink-0 mt-0.5" />
            <div className="flex-1 flex flex-col gap-2">
              <div>
                <Trans
                  i18nKey="web.backups.restore.ownDbWarning"
                  components={{
                    1: <strong />,
                    3: <code className="px-1 rounded bg-card text-foreground" />,
                  }}
                />
              </div>
              <Input
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                placeholder={t('web.backups.restore.confirmPlaceholder')}
                className="h-7 text-[12px]"
              />
            </div>
          </div>
        )}

        {plan && (
          <div className="rounded-md border border-border bg-card/30 p-3 text-[12px] flex flex-col gap-1.5">
            <div className="font-medium text-foreground">
              {t('web.backups.restore.planTitle')}
            </div>
            <ul className="flex flex-col gap-1 text-muted-foreground">
              <li>
                {t('web.backups.restore.planDump', {
                  size: formatBytes(plan.dump_bytes),
                })}
              </li>
              {plan.config_path && (
                <li>
                  {t('web.backups.restore.planConfig', { path: plan.config_path })}
                </li>
              )}
              {plan.secrets_path && (
                <li>
                  {t('web.backups.restore.planSecrets', {
                    path: plan.secrets_path,
                  })}
                </li>
              )}
              {plan.vault_files > 0 && (
                <li>
                  {t('web.backups.restore.planVault', {
                    files: plan.vault_files,
                    roots: (plan.vault_roots ?? []).join(', '),
                  })}
                </li>
              )}
            </ul>
            <div className="text-[11px] text-state-warning mt-1">
              {t('web.backups.restore.planApplyHint')}
            </div>
          </div>
        )}

        {output && (
          <details className="rounded-md border border-border bg-card/30 p-2 text-[11px]">
            <summary className="cursor-pointer text-muted-foreground">
              {t('web.backups.restore.pgRestoreOutput')}
            </summary>
            <pre className="mt-2 whitespace-pre-wrap font-mono">{output}</pre>
          </details>
        )}
      </div>
      <DialogFooter className="gap-2">
        <Button
          variant="outline"
          onClick={preview}
          disabled={busy || !file}
        >
          {busy && !plan
            ? t('web.backups.restore.previewing')
            : t('web.backups.restore.preview')}
        </Button>
        <Button
          onClick={apply}
          disabled={busy || !file || !plan || !confirmReady}
          title={!plan ? t('web.backups.restore.previewFirstHint') : undefined}
        >
          {busy && plan
            ? t('web.backups.restore.restoring')
            : t('web.backups.restore.applyRestore')}
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
  const { t } = useTranslation()
  if (rows === null) {
    return <div className="text-muted-foreground text-sm">{t('web.backups.loading')}</div>
  }
  if (rows.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-border p-6 text-center text-muted-foreground text-[13px]">
        {t('web.backups.backupsTab.empty')}
      </div>
    )
  }
  return (
    <div className="rounded-md border border-border overflow-hidden">
      <table className="w-full text-[12px]">
        <thead className="bg-card/50 text-muted-foreground">
          <tr className="text-left">
            <th className="px-3 py-2 font-medium">{t('web.backups.backupsTab.columns.id')}</th>
            <th className="px-3 py-2 font-medium">{t('web.backups.backupsTab.columns.type')}</th>
            <th className="px-3 py-2 font-medium">{t('web.backups.backupsTab.columns.target')}</th>
            <th className="px-3 py-2 font-medium">{t('web.backups.backupsTab.columns.status')}</th>
            <th className="px-3 py-2 font-medium">{t('web.backups.backupsTab.columns.started')}</th>
            <th className="px-3 py-2 font-medium">{t('web.backups.backupsTab.columns.size')}</th>
            <th className="px-3 py-2 font-medium text-right">{t('web.backups.backupsTab.columns.actions')}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((b) => (
            <tr key={b.id} className="border-t border-border/60">
              <td className="px-3 py-2 font-mono text-[11px]">{b.id}</td>
              <td className="px-3 py-2">
                <div className="flex items-center gap-1.5">
                  <KindBadge kind={b.kind} />
                  <TriggerBadge triggeredBy={b.triggered_by} />
                </div>
              </td>
              <td className="px-3 py-2">{b.target_id}</td>
              <td className="px-3 py-2">
                <div className="flex items-center gap-1.5">
                  <StatusBadge status={b.status} />
                  {b.status === 'succeeded' && <VerifiedBadge backup={b} />}
                </div>
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
                      title={t('web.backups.backupsTab.downloadTooltip')}
                    >
                      <Download className="size-3.5" />
                    </a>
                  )}
                  <Button
                    onClick={() => onDelete(b.id)}
                    variant="outline"
                    size="sm"
                    className="h-7 w-7 p-0"
                    title={t('web.backups.backupsTab.deleteTooltip')}
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

// VerifiedBadge reflects post-backup verification: green when the blob
// was decrypted + pg_restore --list succeeded, red when it failed, and
// nothing while verification hasn't run (older rows / no pg_restore).
function VerifiedBadge({ backup }: { backup: Backup }) {
  const { t } = useTranslation()
  if (backup.verify_error) {
    return (
      <Badge variant="danger" title={backup.verify_error}>
        {t('web.backups.verify.failed')}
      </Badge>
    )
  }
  if (backup.verified_at) {
    return (
      <Badge variant="success" title={t('web.backups.verify.okHint')}>
        {t('web.backups.verify.ok')}
      </Badge>
    )
  }
  return null
}

function KindBadge({ kind }: { kind: BackupKind }) {
  const { t } = useTranslation()
  if (kind === 'full_instance') {
    return (
      <Badge variant="default" title={t('web.backups.kind.fullInstanceHint')}>
        {t('web.backups.kind.fullInstance')}
      </Badge>
    )
  }
  return <Badge variant="muted">{t('web.backups.kind.dbOnly')}</Badge>
}

// TriggerBadge calls out automatic snapshots (pre-migrate / pre-restore)
// so an operator can tell them apart from manual/scheduled backups.
function TriggerBadge({ triggeredBy }: { triggeredBy: TriggeredBy }) {
  const { t } = useTranslation()
  if (triggeredBy === 'pre_migrate') {
    return (
      <Badge variant="warning" title={t('web.backups.trigger.preMigrateHint')}>
        {t('web.backups.trigger.preMigrate')}
      </Badge>
    )
  }
  if (triggeredBy === 'pre_restore') {
    return (
      <Badge variant="warning" title={t('web.backups.trigger.preRestoreHint')}>
        {t('web.backups.trigger.preRestore')}
      </Badge>
    )
  }
  return null
}

// RecoveryKitDialog asks for a recovery passphrase, fetches the wrapped
// kit, and downloads it client-side — with a mandatory "store this
// somewhere safe" warning, since it's the only thing that recovers the
// backup passphrase if the host is lost.
function RecoveryKitDialog() {
  const { t } = useTranslation()
  const [recoveryPass, setRecoveryPass] = useState('')
  const [confirmPass, setConfirmPass] = useState('')
  const [busy, setBusy] = useState(false)
  const mismatch = confirmPass.length > 0 && recoveryPass !== confirmPass
  const ready = recoveryPass.length >= 8 && recoveryPass === confirmPass

  async function download() {
    setBusy(true)
    try {
      const kit = await fetchRecoveryKit(recoveryPass)
      const blob = new Blob([JSON.stringify(kit, null, 2)], {
        type: 'application/json',
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'opendray-recovery-kit.json'
      a.click()
      URL.revokeObjectURL(url)
      toast.success(t('web.backups.recoveryKit.downloadedToast'))
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error(t('web.backups.recoveryKit.failedToast'), { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{t('web.backups.recoveryKit.title')}</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3">
        <div className="rounded-md border border-state-warning/40 bg-state-warning/10 p-3 text-[12px] flex gap-2 items-start">
          <ShieldAlert className="size-4 text-state-warning shrink-0 mt-0.5" />
          <div>{t('web.backups.recoveryKit.warning')}</div>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">
            {t('web.backups.recoveryKit.passphraseLabel')}
          </Label>
          <Input
            type="password"
            value={recoveryPass}
            onChange={(e) => setRecoveryPass(e.target.value)}
            placeholder={t('web.backups.recoveryKit.passphrasePlaceholder')}
            className="h-8"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">
            {t('web.backups.recoveryKit.confirmLabel')}
          </Label>
          <Input
            type="password"
            value={confirmPass}
            onChange={(e) => setConfirmPass(e.target.value)}
            className="h-8"
          />
          {mismatch && (
            <span className="text-[11px] text-state-failed">
              {t('web.backups.recoveryKit.mismatch')}
            </span>
          )}
        </div>
      </div>
      <DialogFooter>
        <Button onClick={download} disabled={busy || !ready}>
          <Download className="size-3.5 mr-1.5" />
          {busy
            ? t('web.backups.recoveryKit.generating')
            : t('web.backups.recoveryKit.download')}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── Schedules tab ────────────────────────────────────────────────

function SchedulesTab() {
  const { t } = useTranslation()
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
      toast.error(t('web.backups.schedulesTab.loadFailedToast'), { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function onDelete(id: string) {
    if (!window.confirm(t('web.backups.schedulesTab.deleteConfirm', { id }))) return
    try {
      await deleteSchedule(id)
      toast.success(t('web.backups.schedulesTab.deletedToast'))
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.schedulesTab.deleteFailedToast'), { description: msg })
    }
  }

  async function toggle(id: string, enabled: boolean) {
    try {
      await updateSchedule(id, { enabled })
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.schedulesTab.toggleFailedToast'), { description: msg })
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <p className="text-[12px] text-muted-foreground">
          {t('web.backups.schedulesTab.description')}
        </p>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm" className="h-8">
              <Plus className="size-3.5 mr-1.5" />
              {t('web.backups.schedulesTab.newSchedule')}
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
        <div className="text-muted-foreground text-sm">{t('web.backups.loading')}</div>
      ) : rows.length === 0 ? (
        <div className="rounded-md border border-dashed border-border p-6 text-center text-muted-foreground text-[13px]">
          {t('web.backups.schedulesTab.empty')}
        </div>
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-[12px]">
            <thead className="bg-card/50 text-muted-foreground">
              <tr className="text-left">
                <th className="px-3 py-2 font-medium">{t('web.backups.schedulesTab.columns.id')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.schedulesTab.columns.target')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.schedulesTab.columns.interval')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.schedulesTab.columns.keep')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.schedulesTab.columns.nextRun')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.schedulesTab.columns.enabled')}</th>
                <th className="px-3 py-2 font-medium text-right">{t('web.backups.schedulesTab.columns.actions')}</th>
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
                    {t('web.backups.schedulesTab.keepCount', { count: s.retention })}
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
                      title={t('web.backups.schedulesTab.deleteTooltip')}
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
  const { t } = useTranslation()
  const enabled = targets.filter((target) => target.enabled)
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
      toast.success(t('web.backups.newSchedule.createdToast'))
      await onCreated()
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error(t('web.backups.newSchedule.createFailedToast'), { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{t('web.backups.newSchedule.title')}</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">{t('web.backups.newSchedule.targetLabel')}</Label>
          <select
            value={targetId}
            onChange={(e) => setTargetId(e.target.value)}
            className="h-8 px-2 rounded-md border border-border bg-card text-[12px]"
          >
            {enabled.map((target) => (
              <option key={target.id} value={target.id}>
                {target.id} ({target.kind})
              </option>
            ))}
          </select>
        </div>
        <div className="flex gap-3">
          <div className="flex-1 flex flex-col gap-1.5">
            <Label className="text-[12px]">{t('web.backups.newSchedule.everyHoursLabel')}</Label>
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
            <Label className="text-[12px]">{t('web.backups.newSchedule.keepLastNLabel')}</Label>
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
          {t('web.backups.newSchedule.enableImmediately')}
        </label>
      </div>
      <DialogFooter>
        <Button onClick={submit} disabled={busy || !targetId}>
          {busy
            ? t('web.backups.newSchedule.creating')
            : t('web.backups.newSchedule.create')}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// ── Targets tab ──────────────────────────────────────────────────

function TargetsTab() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<TargetSpec[] | null>(null)
  const [open, setOpen] = useState(false)
  const [testing, setTesting] = useState<string | null>(null)

  async function refresh() {
    try {
      setRows(await listTargets())
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.targetsTab.listFailedToast'), { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function onDelete(id: string) {
    if (!window.confirm(t('web.backups.targetsTab.deleteConfirm', { id }))) {
      return
    }
    try {
      await deleteTarget(id)
      toast.success(t('web.backups.targetsTab.deletedToast'))
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.targetsTab.deleteFailedToast'), { description: msg })
    }
  }

  async function onTest(id: string) {
    setTesting(id)
    try {
      const res = await testTarget(id)
      if (res.ok) {
        toast.success(t('web.backups.targetsTab.connectionOkToast'), { description: id })
      } else {
        toast.error(t('web.backups.targetsTab.connectionFailedToast'), { description: res.error })
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.backups.targetsTab.testFailedToast'), { description: msg })
    } finally {
      setTesting(null)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <p className="text-[12px] text-muted-foreground">
          <Trans
            i18nKey="web.backups.targetsTab.description"
            components={{ 1: <code />, 3: <code /> }}
          />
        </p>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button size="sm" className="h-8">
              <Plus className="size-3.5 mr-1.5" />
              {t('web.backups.targetsTab.newTarget')}
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
        <div className="text-muted-foreground text-sm">{t('web.backups.loading')}</div>
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-[12px]">
            <thead className="bg-card/50 text-muted-foreground">
              <tr className="text-left">
                <th className="px-3 py-2 font-medium">{t('web.backups.targetsTab.columns.id')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.targetsTab.columns.kind')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.targetsTab.columns.config')}</th>
                <th className="px-3 py-2 font-medium">{t('web.backups.targetsTab.columns.enabled')}</th>
                <th className="px-3 py-2 font-medium text-right">{t('web.backups.targetsTab.columns.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((target) => (
                <tr key={target.id} className="border-t border-border/60">
                  <td className="px-3 py-2 font-mono text-[11px]">{target.id}</td>
                  <td className="px-3 py-2">
                    <Badge variant="outline">{target.kind}</Badge>
                  </td>
                  <td className="px-3 py-2 text-muted-foreground font-mono text-[11px]">
                    {targetSummary(target)}
                  </td>
                  <td className="px-3 py-2">
                    {target.enabled ? (
                      <Badge variant="success">{t('web.backups.targetsTab.on')}</Badge>
                    ) : (
                      <Badge variant="muted">{t('web.backups.targetsTab.off')}</Badge>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="inline-flex gap-1">
                      <Button
                        onClick={() => onTest(target.id)}
                        variant="outline"
                        size="sm"
                        className="h-7 text-[11px]"
                        disabled={testing === target.id}
                      >
                        {testing === target.id
                          ? t('web.backups.targetsTab.testing')
                          : t('web.backups.targetsTab.test')}
                      </Button>
                      <Button
                        onClick={() => onDelete(target.id)}
                        variant="outline"
                        size="sm"
                        className="h-7 w-7 p-0"
                        title={t('web.backups.targetsTab.deleteTooltip')}
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
