import { useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { toast } from 'sonner'
import { Download, Package, ShieldAlert, Trash2, Upload } from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'

import {
  type ExportRecord,
  type ImportRecord,
  type IntegrationExportMode,
  createExport,
  createImport,
  deleteExport,
  exportDownloadURL,
  formatBytes,
  getExport,
  listExports,
  listImports,
} from '@/lib/backup'
import { APIError } from '@/lib/api'

// ExportPage is opendray's user-level data export — separate from
// /backups (the operator-facing disaster-recovery view). Outputs a
// zip bundle the operator can download once via a single-use token.
//
// Scope is opt-in per logical entity. Sensitive fields (plaintext
// API keys) require an explicit "I understand" confirmation; v1
// has no recoverable plaintext keys (all bcrypt hashes), so the
// option exists mostly to surface that fact in the manifest.
export function ExportPage() {
  const { t } = useTranslation()
  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <header className="px-6 py-4 border-b border-border bg-card/30">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-base font-medium flex items-center gap-2">
              <Package className="size-4 text-accent" />
              {t('web.export.title')}
            </h1>
            <p className="text-[12px] text-muted-foreground mt-0.5">
              {t('web.export.subtitle')}
            </p>
          </div>
          <Button asChild variant="outline" size="sm" className="h-8 text-[11px]">
            <Link to="/backups">{t('web.export.backToBackups')}</Link>
          </Button>
        </div>
      </header>
      <div className="flex-1 min-h-0 overflow-y-auto px-6 py-5">
        <div className="max-w-3xl flex flex-col gap-6">
          <SectionHeader>{t('web.export.sections.export')}</SectionHeader>
          <ExportForm />
          <ExportHistory />
          <SectionHeader>{t('web.export.sections.import')}</SectionHeader>
          <ImportForm />
          <ImportHistory />
        </div>
      </div>
    </div>
  )
}

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-[11px] font-semibold tracking-wider uppercase text-muted-foreground border-b border-border pb-1.5">
      {children}
    </div>
  )
}

function ExportForm() {
  const { t } = useTranslation()
  const [memories, setMemories] = useState(true)
  const [customTasks, setCustomTasks] = useState(true)
  const [integrations, setIntegrations] =
    useState<IntegrationExportMode>('metadata')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)

  const wantsPlaintext = integrations === 'plaintext'
  const confirmReady =
    !wantsPlaintext ||
    confirm.trim().toLowerCase() === t('web.export.form.confirmSentinel')

  async function submit() {
    setBusy(true)
    try {
      const e = await createExport({
        memories,
        integrations,
        customTasks,
      })
      toast.success(t('web.export.form.readyToast'), {
        description: t('web.export.form.readyDescription', {
          bytes: e.bytes.toLocaleString(),
        }),
      })
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error(t('web.export.form.failedToast'), { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="rounded-md border border-border p-5 flex flex-col gap-4 bg-card/20">
      <div className="flex flex-col gap-3">
        <div className="text-[12px] font-medium text-muted-foreground uppercase tracking-wider">
          {t('web.export.form.scope')}
        </div>

        <label className="flex items-start gap-3 text-[13px]">
          <Switch
            checked={memories}
            onCheckedChange={setMemories}
            className="mt-0.5"
          />
          <div>
            <div className="font-medium">{t('web.export.form.memories')}</div>
            <div className="text-muted-foreground text-[12px]">
              {t('web.export.form.memoriesHint')}
            </div>
          </div>
        </label>

        <div className="flex flex-col gap-2 pl-1 border-l-2 border-border ml-2 pl-3">
          <div className="text-[13px] font-medium">{t('web.export.form.integrations')}</div>
          <div className="flex flex-col gap-1.5">
            <RadioRow
              checked={integrations === 'none'}
              onClick={() => setIntegrations('none')}
              label={t('web.export.form.integrationOptions.none')}
              hint={t('web.export.form.integrationOptions.noneHint')}
            />
            <RadioRow
              checked={integrations === 'metadata'}
              onClick={() => setIntegrations('metadata')}
              label={t('web.export.form.integrationOptions.metadata')}
              hint={t('web.export.form.integrationOptions.metadataHint')}
            />
            <RadioRow
              checked={integrations === 'plaintext'}
              onClick={() => setIntegrations('plaintext')}
              label={t('web.export.form.integrationOptions.plaintext')}
              hint={t('web.export.form.integrationOptions.plaintextHint')}
              danger
            />
          </div>
          {wantsPlaintext && (
            <div className="rounded-md border border-state-failed/40 bg-state-failed/10 p-3 text-[12px] flex gap-2 items-start">
              <ShieldAlert className="size-4 text-state-failed shrink-0 mt-0.5" />
              <div className="flex-1 flex flex-col gap-2">
                <div>
                  <Trans
                    i18nKey="web.export.form.confirmWarning"
                    components={{
                      1: <code className="px-1 rounded bg-card text-foreground" />,
                    }}
                  />
                </div>
                <Input
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  placeholder={t('web.export.form.confirmPlaceholder')}
                  className="h-7 text-[12px]"
                />
              </div>
            </div>
          )}
        </div>

        <label className="flex items-start gap-3 text-[13px]">
          <Switch
            checked={customTasks}
            onCheckedChange={setCustomTasks}
            className="mt-0.5"
          />
          <div>
            <div className="font-medium">{t('web.export.form.customTasks')}</div>
            <div className="text-muted-foreground text-[12px]">
              {t('web.export.form.customTasksHint')}
            </div>
          </div>
        </label>
      </div>

      <div className="border-t border-border pt-3 flex items-center justify-between">
        <div className="text-[11px] text-muted-foreground">
          {t('web.export.form.footnote')}
        </div>
        <Button onClick={submit} disabled={busy || !confirmReady}>
          {busy ? t('web.export.form.building') : t('web.export.form.create')}
        </Button>
      </div>
    </div>
  )
}

function RadioRow({
  checked,
  onClick,
  label,
  hint,
  danger,
}: {
  checked: boolean
  onClick: () => void
  label: string
  hint: string
  danger?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        'flex items-start gap-2.5 text-left p-2 rounded-md border transition-colors ' +
        (checked
          ? danger
            ? 'border-state-failed/40 bg-state-failed/5'
            : 'border-accent/40 bg-accent/5'
          : 'border-border hover:bg-card/40')
      }
    >
      <span
        className={
          'mt-0.5 size-3.5 rounded-full border flex items-center justify-center ' +
          (checked ? 'border-accent' : 'border-border')
        }
      >
        {checked && <span className="size-1.5 rounded-full bg-accent" />}
      </span>
      <div>
        <div className="text-[12px] font-medium">{label}</div>
        <div className="text-[11px] text-muted-foreground">{hint}</div>
      </div>
    </button>
  )
}

function ExportHistory() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<ExportRecord[] | null>(null)
  const [tokenCache, setTokenCache] = useState<Record<string, string>>({})

  async function refresh() {
    try {
      const list = await listExports()
      setRows(list)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.export.history.listFailedToast'), { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    const intervalId = window.setInterval(refresh, 5000)
    return () => window.clearInterval(intervalId)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function onDownload(id: string) {
    try {
      // List endpoint redacts the token, so re-fetch detail to get it.
      let token = tokenCache[id]
      if (!token) {
        const detail = await getExport(id)
        token = detail.download_token ?? ''
      }
      if (!token) {
        toast.error(t('web.export.history.noTokenToast'))
        return
      }
      setTokenCache((c) => ({ ...c, [id]: token }))
      window.location.href = exportDownloadURL(id, token)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.export.history.downloadFailedToast'), { description: msg })
    }
  }

  async function onDelete(id: string) {
    if (!window.confirm(t('web.export.history.deleteConfirm', { id }))) return
    try {
      await deleteExport(id)
      toast.success(t('web.export.history.deletedToast'))
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.export.history.deleteFailedToast'), { description: msg })
    }
  }

  if (rows === null) return <div className="text-muted-foreground text-sm">{t('web.export.history.loading')}</div>
  if (rows.length === 0) {
    return (
      <div className="text-[12px] text-muted-foreground">
        {t('web.export.history.empty')}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="text-[12px] font-medium text-muted-foreground uppercase tracking-wider">
        {t('web.export.history.title')}
      </div>
      <div className="rounded-md border border-border overflow-hidden">
        <table className="w-full text-[12px]">
          <thead className="bg-card/50 text-muted-foreground">
            <tr className="text-left">
              <th className="px-3 py-2 font-medium">{t('web.export.history.columns.id')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.history.columns.status')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.history.columns.scope')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.history.columns.size')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.history.columns.expires')}</th>
              <th className="px-3 py-2 font-medium text-right">{t('web.export.history.columns.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((e) => (
              <tr key={e.id} className="border-t border-border/60">
                <td className="px-3 py-2 font-mono text-[11px]">{e.id}</td>
                <td className="px-3 py-2">
                  <ExportStatusBadge status={e.status} />
                </td>
                <td className="px-3 py-2 text-muted-foreground">
                  {scopeSummary(e.scope) || t('web.export.history.scopeEmpty')}
                </td>
                <td className="px-3 py-2 text-muted-foreground">
                  {e.bytes > 0 ? formatBytes(e.bytes) : '—'}
                </td>
                <td className="px-3 py-2 text-muted-foreground">
                  {formatRelative(e.expires_at)}
                </td>
                <td className="px-3 py-2 text-right">
                  <div className="inline-flex gap-1">
                    {e.status === 'ready' && (
                      <Button
                        onClick={() => onDownload(e.id)}
                        variant="outline"
                        size="sm"
                        className="h-7 text-[11px]"
                      >
                        <Download className="size-3 mr-1" />
                        {t('web.export.history.download')}
                      </Button>
                    )}
                    <Button
                      onClick={() => onDelete(e.id)}
                      variant="outline"
                      size="sm"
                      className="h-7 w-7 p-0"
                      title={t('web.export.history.deleteTooltip')}
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
    </div>
  )
}

function ExportStatusBadge({ status }: { status: ExportRecord['status'] }) {
  const m: Record<
    ExportRecord['status'],
    'success' | 'warning' | 'danger' | 'muted'
  > = {
    pending: 'warning',
    running: 'warning',
    ready: 'success',
    failed: 'danger',
    expired: 'muted',
  }
  return <Badge variant={m[status]}>{status}</Badge>
}

function scopeSummary(s: ExportRecord['scope']): string {
  const parts: string[] = []
  if (s.memories) parts.push('memories')
  if (s.integrations !== 'none') parts.push(`integrations(${s.integrations})`)
  if (s.custom_tasks) parts.push('custom_tasks')
  return parts.join(', ')
}

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
  return `${Math.round(sec / 3600)}h ago`
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

// ── Import (C reverse) ──────────────────────────────────────────

function ImportForm() {
  const { t } = useTranslation()
  const [file, setFile] = useState<File | null>(null)
  const [memories, setMemories] = useState(true)
  const [integrations, setIntegrations] = useState(true)
  const [customTasks, setCustomTasks] = useState(true)
  const [busy, setBusy] = useState(false)
  const [last, setLast] = useState<ImportRecord | null>(null)

  async function submit() {
    if (!file) {
      toast.error(t('web.export.import.pickFileToast'))
      return
    }
    setBusy(true)
    try {
      const imp = await createImport({
        bundle: file,
        memories,
        integrations,
        customTasks,
      })
      setLast(imp)
      if (imp.status === 'succeeded') {
        toast.success(t('web.export.import.doneToast'), {
          description: importSummary(imp),
        })
      } else {
        toast.warning(t('web.export.import.finishedWithErrors'), {
          description: imp.error || importSummary(imp),
        })
      }
      setFile(null)
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error(t('web.export.import.failedToast'), { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="rounded-md border border-border p-5 flex flex-col gap-4 bg-card/20">
      <p className="text-[12px] text-muted-foreground">
        <Trans
          i18nKey="web.export.import.intro"
          components={{
            1: <strong />,
            3: <code className="text-foreground" />,
            5: <Link to="/memory" className="underline" />,
            7: <code className="text-foreground" />,
          }}
        />
      </p>

      <div className="flex flex-col gap-1.5">
        <Label className="text-[12px]">{t('web.export.import.bundleLabel')}</Label>
        <input
          type="file"
          accept=".zip,application/zip"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          className="text-[12px]"
        />
      </div>

      <div className="flex flex-col gap-2">
        <label className="flex items-center gap-2 text-[13px]">
          <Switch
            checked={memories}
            onCheckedChange={setMemories}
            className="scale-75"
          />
          {t('web.export.import.memoriesLabel')}
        </label>
        <label className="flex items-center gap-2 text-[13px]">
          <Switch
            checked={integrations}
            onCheckedChange={setIntegrations}
            className="scale-75"
          />
          {t('web.export.import.integrationsLabel')}
        </label>
        <label className="flex items-center gap-2 text-[13px]">
          <Switch
            checked={customTasks}
            onCheckedChange={setCustomTasks}
            className="scale-75"
          />
          {t('web.export.import.customTasksLabel')}
        </label>
      </div>

      <div className="flex justify-end">
        <Button
          onClick={submit}
          disabled={busy || !file || (!memories && !integrations && !customTasks)}
        >
          <Upload className="size-3.5 mr-1.5" />
          {busy ? t('web.export.import.importing') : t('web.export.import.importBundle')}
        </Button>
      </div>

      {last && <ImportSummaryCard imp={last} />}
    </div>
  )
}

function ImportSummaryCard({ imp }: { imp: ImportRecord }) {
  const { t } = useTranslation()
  return (
    <div className="rounded-md border border-border bg-card/30 p-3 text-[12px] flex flex-col gap-1.5">
      <div className="flex items-center gap-2">
        <span className="font-mono text-[11px]">{imp.id}</span>
        <ImportStatusBadge status={imp.status} />
      </div>
      <CountsRow label={t('web.export.import.summaryCard.memories')} c={imp.counts.memories} />
      <CountsRow label={t('web.export.import.summaryCard.integrations')} c={imp.counts.integrations} />
      <CountsRow label={t('web.export.import.summaryCard.customTasks')} c={imp.counts.custom_tasks} />
      {imp.error && (
        <div className="mt-1 text-state-failed">{imp.error}</div>
      )}
    </div>
  )
}

function CountsRow({
  label,
  c,
}: {
  label: string
  c: { created: number; skipped: number; failed: number }
}) {
  const { t } = useTranslation()
  if (c.created + c.skipped + c.failed === 0) {
    return null
  }
  return (
    <div className="flex items-center gap-3 text-muted-foreground">
      <span className="w-32">{label}</span>
      <span>
        <strong className="text-foreground">{c.created}</strong>{' '}
        {t('web.export.import.summaryCard.created')}
      </span>
      <span>
        {c.skipped} {t('web.export.import.summaryCard.skipped')}
      </span>
      {c.failed > 0 && (
        <span className="text-state-failed">
          {c.failed} {t('web.export.import.summaryCard.failed')}
        </span>
      )}
    </div>
  )
}

function ImportStatusBadge({ status }: { status: ImportRecord['status'] }) {
  const m: Record<
    ImportRecord['status'],
    'success' | 'warning' | 'danger' | 'muted'
  > = {
    pending: 'warning',
    running: 'warning',
    succeeded: 'success',
    failed: 'danger',
  }
  return <Badge variant={m[status]}>{status}</Badge>
}

function importSummary(imp: ImportRecord): string {
  const parts: string[] = []
  const m = imp.counts.memories
  const i = imp.counts.integrations
  const t = imp.counts.custom_tasks
  if (m.created || m.skipped) parts.push(`memories: ${m.created}/${m.created + m.skipped}`)
  if (i.created || i.skipped) parts.push(`integrations: ${i.created}/${i.created + i.skipped}`)
  if (t.created || t.skipped) parts.push(`custom_tasks: ${t.created}/${t.created + t.skipped}`)
  return parts.join(' • ')
}

function ImportHistory() {
  const { t } = useTranslation()
  const [rows, setRows] = useState<ImportRecord[] | null>(null)

  async function refresh() {
    try {
      const list = await listImports(20)
      setRows(list)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error(t('web.export.imports.listFailedToast'), { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    const intervalId = window.setInterval(refresh, 5000)
    return () => window.clearInterval(intervalId)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (rows === null) return <div className="text-muted-foreground text-sm">{t('web.export.imports.loading')}</div>
  if (rows.length === 0) {
    return (
      <div className="text-[12px] text-muted-foreground">
        {t('web.export.imports.empty')}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="text-[11px] font-medium text-muted-foreground tracking-wider uppercase">
        {t('web.export.imports.title')}
      </div>
      <div className="rounded-md border border-border overflow-hidden">
        <table className="w-full text-[12px]">
          <thead className="bg-card/50 text-muted-foreground">
            <tr className="text-left">
              <th className="px-3 py-2 font-medium">{t('web.export.imports.columns.id')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.imports.columns.status')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.imports.columns.source')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.imports.columns.counts')}</th>
              <th className="px-3 py-2 font-medium">{t('web.export.imports.columns.when')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((imp) => (
              <tr key={imp.id} className="border-t border-border/60">
                <td className="px-3 py-2 font-mono text-[11px]">{imp.id}</td>
                <td className="px-3 py-2">
                  <ImportStatusBadge status={imp.status} />
                </td>
                <td className="px-3 py-2 text-muted-foreground">
                  {imp.source_filename || '—'}
                  {imp.source_bytes > 0 && (
                    <span className="ml-1 text-[10px]">
                      ({formatBytes(imp.source_bytes)})
                    </span>
                  )}
                </td>
                <td className="px-3 py-2 text-muted-foreground text-[11px]">
                  {importSummary(imp) || t('web.export.imports.noneCounts')}
                </td>
                <td className="px-3 py-2 text-muted-foreground">
                  {formatRelative(imp.started_at)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// Suppress unused Label import (kept for future field labels).
void Label
