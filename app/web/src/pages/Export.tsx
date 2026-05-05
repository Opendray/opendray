import { useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { toast } from 'sonner'
import { Download, Package, ShieldAlert, Trash2, Upload } from 'lucide-react'

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
  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <header className="px-6 py-4 border-b border-border bg-card/30">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-base font-medium flex items-center gap-2">
              <Package className="size-4 text-accent" />
              Export data
            </h1>
            <p className="text-[12px] text-muted-foreground mt-0.5">
              Take a one-shot zip bundle of selected logical entities.
              Bundles are kept on the server for 24 hours, then
              automatically reaped.
            </p>
          </div>
          <Button asChild variant="outline" size="sm" className="h-8 text-[11px]">
            <Link to="/backups">← Backups</Link>
          </Button>
        </div>
      </header>
      <div className="flex-1 min-h-0 overflow-y-auto px-6 py-5">
        <div className="max-w-3xl flex flex-col gap-6">
          <SectionHeader>Export</SectionHeader>
          <ExportForm />
          <ExportHistory />
          <SectionHeader>Import</SectionHeader>
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
  const [memories, setMemories] = useState(true)
  const [customTasks, setCustomTasks] = useState(true)
  const [integrations, setIntegrations] =
    useState<IntegrationExportMode>('metadata')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)

  const wantsPlaintext = integrations === 'plaintext'
  const confirmReady =
    !wantsPlaintext || confirm.trim().toLowerCase() === 'i understand'

  async function submit() {
    setBusy(true)
    try {
      const e = await createExport({
        memories,
        integrations,
        customTasks,
      })
      toast.success('Export ready', {
        description: `${e.bytes.toLocaleString()} bytes`,
      })
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPI(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error('Export failed', { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="rounded-md border border-border p-5 flex flex-col gap-4 bg-card/20">
      <div className="flex flex-col gap-3">
        <div className="text-[12px] font-medium text-muted-foreground uppercase tracking-wider">
          Scope
        </div>

        <label className="flex items-start gap-3 text-[13px]">
          <Switch
            checked={memories}
            onCheckedChange={setMemories}
            className="mt-0.5"
          />
          <div>
            <div className="font-medium">Memories</div>
            <div className="text-muted-foreground text-[12px]">
              Cross-CLI persistent memory rows (text + scope + metadata).
              Embedding vectors are omitted; importer re-embeds.
            </div>
          </div>
        </label>

        <div className="flex flex-col gap-2 pl-1 border-l-2 border-border ml-2 pl-3">
          <div className="text-[13px] font-medium">Integrations</div>
          <div className="flex flex-col gap-1.5">
            <RadioRow
              checked={integrations === 'none'}
              onClick={() => setIntegrations('none')}
              label="None"
              hint="Skip the integrations table entirely."
            />
            <RadioRow
              checked={integrations === 'metadata'}
              onClick={() => setIntegrations('metadata')}
              label="Metadata only (recommended)"
              hint="ID, name, route prefix, scopes — no API key material."
            />
            <RadioRow
              checked={integrations === 'plaintext'}
              onClick={() => setIntegrations('plaintext')}
              label="Include plaintext API keys"
              hint="v1 bcrypt-only: no recoverable plaintext exists. Manifest documents this; nothing leaks."
              danger
            />
          </div>
          {wantsPlaintext && (
            <div className="rounded-md border border-state-failed/40 bg-state-failed/10 p-3 text-[12px] flex gap-2 items-start">
              <ShieldAlert className="size-4 text-state-failed shrink-0 mt-0.5" />
              <div className="flex-1 flex flex-col gap-2">
                <div>
                  Type{' '}
                  <code className="px-1 rounded bg-card text-foreground">
                    I understand
                  </code>{' '}
                  to confirm. opendray currently stores only bcrypt
                  hashes — selecting plaintext does NOT export any
                  plaintext (the feature is reserved for a future
                  release that keeps plaintext caches).
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
        </div>

        <label className="flex items-start gap-3 text-[13px]">
          <Switch
            checked={customTasks}
            onCheckedChange={setCustomTasks}
            className="mt-0.5"
          />
          <div>
            <div className="font-medium">Custom tasks</div>
            <div className="text-muted-foreground text-[12px]">
              Operator-defined tasks shown in the Inspector's Tasks tab.
            </div>
          </div>
        </label>
      </div>

      <div className="border-t border-border pt-3 flex items-center justify-between">
        <div className="text-[11px] text-muted-foreground">
          Audit logs and session transcripts are out of scope —
          covered by /backups (operator dump) instead.
        </div>
        <Button onClick={submit} disabled={busy || !confirmReady}>
          {busy ? 'Building…' : 'Create export'}
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
  const [rows, setRows] = useState<ExportRecord[] | null>(null)
  const [tokenCache, setTokenCache] = useState<Record<string, string>>({})

  async function refresh() {
    try {
      const list = await listExports()
      setRows(list)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Failed to list exports', { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    const t = window.setInterval(refresh, 5000)
    return () => window.clearInterval(t)
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
        toast.error('No download token (expired?)')
        return
      }
      setTokenCache((c) => ({ ...c, [id]: token }))
      window.location.href = exportDownloadURL(id, token)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Download failed', { description: msg })
    }
  }

  async function onDelete(id: string) {
    if (!window.confirm(`Delete export ${id}?`)) return
    try {
      await deleteExport(id)
      toast.success('Export deleted')
      await refresh()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Delete failed', { description: msg })
    }
  }

  if (rows === null) return <div className="text-muted-foreground text-sm">Loading…</div>
  if (rows.length === 0) {
    return (
      <div className="text-[12px] text-muted-foreground">
        No exports yet. Use the form above to create one.
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="text-[12px] font-medium text-muted-foreground uppercase tracking-wider">
        History
      </div>
      <div className="rounded-md border border-border overflow-hidden">
        <table className="w-full text-[12px]">
          <thead className="bg-card/50 text-muted-foreground">
            <tr className="text-left">
              <th className="px-3 py-2 font-medium">ID</th>
              <th className="px-3 py-2 font-medium">Status</th>
              <th className="px-3 py-2 font-medium">Scope</th>
              <th className="px-3 py-2 font-medium">Size</th>
              <th className="px-3 py-2 font-medium">Expires</th>
              <th className="px-3 py-2 font-medium text-right">Actions</th>
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
                  {scopeSummary(e.scope)}
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
                        Download
                      </Button>
                    )}
                    <Button
                      onClick={() => onDelete(e.id)}
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
  return parts.join(', ') || '(empty)'
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
  const [file, setFile] = useState<File | null>(null)
  const [memories, setMemories] = useState(true)
  const [integrations, setIntegrations] = useState(true)
  const [customTasks, setCustomTasks] = useState(true)
  const [busy, setBusy] = useState(false)
  const [last, setLast] = useState<ImportRecord | null>(null)

  async function submit() {
    if (!file) {
      toast.error('Pick a bundle file first')
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
        toast.success('Import done', {
          description: importSummary(imp),
        })
      } else {
        toast.warning('Import finished with errors', {
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
      toast.error('Import failed', { description: msg })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="rounded-md border border-border p-5 flex flex-col gap-4 bg-card/20">
      <p className="text-[12px] text-muted-foreground">
        Replay an export bundle (zip) into the live database.
        Conflicts (matching id, or unique route_prefix for
        integrations) are <strong>skipped</strong> by default.
        Memories are tagged{' '}
        <code className="text-foreground">embedder=imported_v1</code>{' '}
        and need a re-embed pass before search returns them; trigger
        re-embed under{' '}
        <Link to="/memory" className="underline">
          Memory → Maintenance
        </Link>
        . Integrations are imported with{' '}
        <code className="text-foreground">enabled=false</code> and a
        non-bcrypt placeholder key — operator must rotate before use.
      </p>

      <div className="flex flex-col gap-1.5">
        <Label className="text-[12px]">Bundle (.zip)</Label>
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
          Memories
        </label>
        <label className="flex items-center gap-2 text-[13px]">
          <Switch
            checked={integrations}
            onCheckedChange={setIntegrations}
            className="scale-75"
          />
          Integrations (metadata only — keys never imported)
        </label>
        <label className="flex items-center gap-2 text-[13px]">
          <Switch
            checked={customTasks}
            onCheckedChange={setCustomTasks}
            className="scale-75"
          />
          Custom tasks
        </label>
      </div>

      <div className="flex justify-end">
        <Button
          onClick={submit}
          disabled={busy || !file || (!memories && !integrations && !customTasks)}
        >
          <Upload className="size-3.5 mr-1.5" />
          {busy ? 'Importing…' : 'Import bundle'}
        </Button>
      </div>

      {last && <ImportSummaryCard imp={last} />}
    </div>
  )
}

function ImportSummaryCard({ imp }: { imp: ImportRecord }) {
  return (
    <div className="rounded-md border border-border bg-card/30 p-3 text-[12px] flex flex-col gap-1.5">
      <div className="flex items-center gap-2">
        <span className="font-mono text-[11px]">{imp.id}</span>
        <ImportStatusBadge status={imp.status} />
      </div>
      <CountsRow label="Memories" c={imp.counts.memories} />
      <CountsRow label="Integrations" c={imp.counts.integrations} />
      <CountsRow label="Custom tasks" c={imp.counts.custom_tasks} />
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
  if (c.created + c.skipped + c.failed === 0) {
    return null
  }
  return (
    <div className="flex items-center gap-3 text-muted-foreground">
      <span className="w-32">{label}</span>
      <span>
        <strong className="text-foreground">{c.created}</strong> created
      </span>
      <span>{c.skipped} skipped</span>
      {c.failed > 0 && (
        <span className="text-state-failed">{c.failed} failed</span>
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
  const [rows, setRows] = useState<ImportRecord[] | null>(null)

  async function refresh() {
    try {
      const list = await listImports(20)
      setRows(list)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Failed to list imports', { description: msg })
    }
  }

  useEffect(() => {
    refresh()
    const t = window.setInterval(refresh, 5000)
    return () => window.clearInterval(t)
  }, [])

  if (rows === null) return <div className="text-muted-foreground text-sm">Loading…</div>
  if (rows.length === 0) {
    return (
      <div className="text-[12px] text-muted-foreground">
        No imports yet.
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="text-[11px] font-medium text-muted-foreground tracking-wider uppercase">
        History
      </div>
      <div className="rounded-md border border-border overflow-hidden">
        <table className="w-full text-[12px]">
          <thead className="bg-card/50 text-muted-foreground">
            <tr className="text-left">
              <th className="px-3 py-2 font-medium">ID</th>
              <th className="px-3 py-2 font-medium">Status</th>
              <th className="px-3 py-2 font-medium">Source</th>
              <th className="px-3 py-2 font-medium">Counts</th>
              <th className="px-3 py-2 font-medium">When</th>
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
                  {importSummary(imp) || '(none)'}
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
