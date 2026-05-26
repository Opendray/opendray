import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Trash2,
  Loader2,
  KeyRound,
  GitBranch,
  Pencil,
  Play,
  Sparkles,
  Lock,
  RotateCcw,
  Plug,
  ShieldCheck,
  ChevronDown,
} from 'lucide-react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  listGitHosts,
  createGitHost,
  updateGitHost,
  deleteGitHost,
  type GitHost,
  type GitHostKind,
} from '@/lib/githost'
import {
  listCustomTasks,
  createCustomTask,
  updateCustomTask,
  deleteCustomTask,
  type CustomTask,
} from '@/lib/customTasks'
import {
  listSkills,
  getSkill,
  createSkill,
  updateSkill,
  deleteSkill,
} from '@/lib/skills'
import {
  listMcps,
  getMcp,
  createMcp,
  updateMcp,
  deleteMcp,
  testMcp,
  getMcpSecrets,
  setMcpSecret,
  deleteMcpSecret,
  defaultMcpServer,
  type McpServer,
  type McpTransport,
} from '@/lib/mcps'
import { cn } from '@/lib/utils'

// PluginsPage is the Workspace nav entry that hosts inspector-tab
// configuration. v1 ships the Git-host token manager; future panels
// (Logs filters, Tasks aliases, …) will land as additional sections
// here rather than spawning more nav items.
export function PluginsPage() {
  const { t } = useTranslation()
  return (
    <div className="h-full flex flex-col p-6 gap-4 overflow-y-auto">
      <header className="flex flex-col gap-1">
        <h1 className="text-[18px] font-semibold tracking-tight">
          {t('web.plugins.title')}
        </h1>
        <p className="text-[12px] text-muted-foreground max-w-[640px]">
          {t('web.plugins.subtitle')}
        </p>
      </header>

      <GitHostsSection />
      <CustomTasksSection />
      <SkillsSection />
      <McpSection />
      <McpSecretsSection />
    </div>
  )
}

// CollapsibleSection wraps each plugin block with a clickable header
// (chevron + icon + title). Open/closed state persists per-id in
// localStorage so reloads keep the user's preferred layout.
//
// Body and the action button are unmounted when collapsed so the
// section's own queries pause too — keeps the page light when the
// user has narrowed focus to one plugin.
function CollapsibleSection({
  id,
  icon,
  title,
  description,
  badge,
  action,
  defaultOpen = true,
  children,
}: {
  id: string
  icon: ReactNode
  title: string
  description?: ReactNode
  badge?: ReactNode
  action?: ReactNode
  defaultOpen?: boolean
  children: ReactNode
}) {
  const lsKey = `opendray.plugins.collapsed.${id}`
  const [open, setOpen] = useState<boolean>(() => {
    if (typeof window === 'undefined') return defaultOpen
    const stored = localStorage.getItem(lsKey)
    return stored == null ? defaultOpen : stored === '0'
  })
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(lsKey, open ? '0' : '1')
    }
  }, [lsKey, open])

  return (
    <section className="flex flex-col gap-3 max-w-[840px]">
      <div className="flex items-start justify-between gap-2">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className={cn(
            'flex items-start gap-2 text-left flex-1 min-w-0 rounded-md',
            'hover:bg-card/40 -mx-2 px-2 py-1 transition-colors',
          )}
          aria-expanded={open}
        >
          <ChevronDown
            className={cn(
              'size-3.5 mt-1 text-muted-foreground transition-transform shrink-0',
              !open && '-rotate-90',
            )}
          />
          <div className="flex flex-col gap-0.5 min-w-0">
            <h2 className="text-[14px] font-semibold flex items-center gap-2">
              {icon}
              {title}
              {badge}
            </h2>
            {description && open && (
              <p className="text-[11px] text-muted-foreground">{description}</p>
            )}
          </div>
        </button>
        {open && action}
      </div>
      {open && children}
    </section>
  )
}

function McpSection() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: servers, isLoading } = useQuery({
    queryKey: ['mcps'],
    queryFn: listMcps,
  })
  const [editingId, setEditingId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const remove = useMutation({
    mutationFn: deleteMcp,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcps'] })
      toast.success(t('web.plugins.mcp.removedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.mcp.deleteFailedToast'), {
        description: err.message,
      }),
  })

  const test = useMutation({
    mutationFn: (id: string) => testMcp(id),
    onSuccess: (res, id) => {
      if (res.ok) {
        const summary =
          res.transport === 'stdio'
            ? t('web.plugins.mcp.test.connected', { count: res.toolCount ?? 0 })
            : t('web.plugins.mcp.test.reachable')
        toast.success(`${id}: ${summary}`, {
          description:
            (res.tools?.length ? res.tools.join(', ') : res.note) || undefined,
        })
      } else {
        const failed = res.checks.find((c) => !c.ok)
        toast.error(`${id}: ${t('web.plugins.mcp.test.failed')}`, {
          description: failed
            ? `${failed.name}: ${failed.detail ?? ''}`
            : res.note,
        })
      }
    },
    onError: (err: Error, id) =>
      toast.error(`${id}: ${t('web.plugins.mcp.test.failed')}`, {
        description: err.message,
      }),
  })

  const toggle = useMutation({
    mutationFn: async (s: McpServer) =>
      updateMcp(s.id, { ...s, enabled: !s.enabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcps'] })
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.mcp.toggleFailedToast'), {
        description: err.message,
      }),
  })

  return (
    <CollapsibleSection
      id="mcp-servers"
      icon={<Plug className="size-4 text-muted-foreground" />}
      title={t('web.plugins.mcp.title')}
      description={
        <Trans
          i18nKey="web.plugins.mcp.description"
          components={{ 1: <code />, 3: <code />, 5: <strong /> }}
        />
      }
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          {t('web.plugins.mcp.newServer')}
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            {t('web.plugins.common.loading')}
          </div>
        ) : (servers ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            {t('web.plugins.mcp.empty')}
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.mcp.columns.name')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.mcp.columns.transport')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.mcp.columns.spec')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.mcp.columns.enabled')}
                </th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {servers!.map((s) => (
                <tr
                  key={s.id}
                  className="border-t border-border hover:bg-card/40 align-top"
                >
                  <td className="px-3 py-2">
                    <div className="font-medium font-mono">{s.name}</div>
                    {s.description && (
                      <div className="text-[10px] text-muted-foreground/70 italic">
                        {s.description}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2 font-mono text-[10.5px]">
                    {s.transport ?? 'stdio'}
                    {(s.transport === 'sse' || s.transport === 'http') && (
                      <div
                        className="mt-1 inline-flex items-center text-[9.5px] px-1.5 py-px rounded bg-amber-500/15 text-amber-300 border border-amber-500/30 font-sans"
                        title={t('web.plugins.mcp.codexUnsupportedTooltip')}
                      >
                        {t('web.plugins.mcp.codexUnsupportedBadge')}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2 font-mono text-[10.5px] break-all max-w-[260px]">
                    {s.transport === 'sse' || s.transport === 'http'
                      ? s.url || (
                          <span className="opacity-50">
                            {t('web.plugins.mcp.noUrl')}
                          </span>
                        )
                      : s.command
                        ? `${s.command}${s.args?.length ? ' ' + s.args.join(' ') : ''}`
                        : (
                          <span className="opacity-50">
                            {t('web.plugins.mcp.noCommand')}
                          </span>
                        )}
                  </td>
                  <td className="px-3 py-2">
                    <Switch
                      checked={s.enabled}
                      onCheckedChange={() => toggle.mutate(s)}
                      disabled={toggle.isPending}
                    />
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => test.mutate(s.id)}
                        disabled={test.isPending && test.variables === s.id}
                        className="h-7 px-2 text-[11px] gap-1"
                        title={t('web.plugins.mcp.test.title')}
                      >
                        {test.isPending && test.variables === s.id ? (
                          <Loader2 className="size-3 animate-spin" />
                        ) : (
                          <Plug className="size-3" />
                        )}
                        {t('web.plugins.mcp.test.button')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditingId(s.id)}
                        className="h-7 px-2 text-[11px] gap-1"
                      >
                        <Pencil className="size-3" />
                        {t('web.plugins.common.edit')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (
                            confirm(
                              t('web.plugins.mcp.deleteConfirm', { id: s.id }),
                            )
                          ) {
                            remove.mutate(s.id)
                          }
                        }}
                        className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="size-3" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <McpEditor
        open={creating}
        onOpenChange={setCreating}
        mode="create"
      />
      <McpEditor
        open={editingId != null}
        onOpenChange={(v) => !v && setEditingId(null)}
        mode="edit"
        editingId={editingId}
      />
    </CollapsibleSection>
  )
}

interface McpEditorProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  mode: 'create' | 'edit'
  editingId?: string | null
}

function McpEditor({ open, onOpenChange, mode, editingId }: McpEditorProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [id, setId] = useState('')
  const [body, setBody] = useState('')
  const [parseError, setParseError] = useState<string | null>(null)
  const [transport, setTransport] = useState<McpTransport>('stdio')

  const { data: existing, isLoading } = useQuery({
    queryKey: ['mcp', editingId],
    queryFn: () => getMcp(editingId!),
    enabled: open && mode === 'edit' && !!editingId,
  })

  useEffect(() => {
    if (mode === 'create' && open) {
      setId('')
      setTransport('stdio')
      setBody(prettyMcp(defaultMcpServer('stdio')))
      setParseError(null)
    } else if (mode === 'edit' && existing) {
      setId(existing.id)
      setTransport((existing.transport ?? 'stdio') as McpTransport)
      setBody(prettyMcp(existing))
      setParseError(null)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, existing?.id, open])

  // Switching transport in create mode swaps the JSON template so the
  // user sees fields appropriate for the chosen transport — sse/http
  // need url/headers, stdio needs command/args/env. Edit mode keeps
  // the user's hand-tuned body intact; changing transport there is a
  // raw-JSON edit by design.
  const handleTransportChange = (next: McpTransport) => {
    setTransport(next)
    if (mode === 'create') {
      setBody(prettyMcp(defaultMcpServer(next)))
      setParseError(null)
    }
  }

  const create = useMutation({
    mutationFn: async () => {
      const parsed = parseMcp(body, id)
      return createMcp(id, parsed)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcps'] })
      toast.success(t('web.plugins.mcp.editor.createdToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.mcp.editor.createFailedToast'), {
        description: err.message,
      }),
  })

  const update = useMutation({
    mutationFn: async () => {
      const parsed = parseMcp(body, editingId!)
      return updateMcp(editingId!, parsed)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcps'] })
      qc.invalidateQueries({ queryKey: ['mcp', editingId] })
      toast.success(t('web.plugins.mcp.editor.savedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.mcp.editor.saveFailedToast'), {
        description: err.message,
      }),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    try {
      JSON.parse(body)
      setParseError(null)
    } catch (err) {
      setParseError((err as Error).message)
      return
    }
    if (mode === 'create') create.mutate()
    else update.mutate()
  }

  const busy = create.isPending || update.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[min(92vw,860px)] w-[min(92vw,860px)]">
        <DialogHeader>
          <DialogTitle>
            {mode === 'create'
              ? t('web.plugins.mcp.editor.createTitle')
              : t('web.plugins.mcp.editor.editTitle', { id: editingId })}
          </DialogTitle>
          <DialogDescription>
            <Trans
              i18nKey="web.plugins.mcp.editor.description"
              components={{
                1: <code />,
                3: <code />,
                5: <code />,
                7: <code />,
                9: <code />,
                11: <code />,
                13: <code />,
              }}
            />
          </DialogDescription>
        </DialogHeader>
        {isLoading && mode === 'edit' ? (
          <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-6">
            <Loader2 className="size-3 animate-spin" />
            {t('web.plugins.common.loading')}
          </div>
        ) : (
          <form onSubmit={submit} className="flex flex-col gap-3">
            {mode === 'create' && (
              <div className="space-y-1.5">
                <Label htmlFor="mcp-id">
                  {t('web.plugins.mcp.editor.idLabel')}
                </Label>
                <Input
                  id="mcp-id"
                  value={id}
                  onChange={(e) => setId(e.target.value)}
                  placeholder={t('web.plugins.mcp.editor.idPlaceholder')}
                  required
                  className="font-mono"
                />
                <p className="text-[10.5px] text-muted-foreground/80">
                  <Trans
                    i18nKey="web.plugins.mcp.editor.idHint"
                    components={{ 1: <code /> }}
                  />
                </p>
              </div>
            )}
            {mode === 'create' && (
              <div className="space-y-1.5">
                <Label htmlFor="mcp-transport">
                  {t('web.plugins.mcp.editor.transportLabel')}
                </Label>
                <Select
                  value={transport}
                  onValueChange={(v) =>
                    handleTransportChange(v as McpTransport)
                  }
                >
                  <SelectTrigger id="mcp-transport">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="stdio">
                      {t('web.plugins.mcp.editor.transportStdio')}
                    </SelectItem>
                    <SelectItem value="sse">
                      {t('web.plugins.mcp.editor.transportSse')}
                    </SelectItem>
                    <SelectItem value="http">
                      {t('web.plugins.mcp.editor.transportHttp')}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-[10.5px] text-muted-foreground/80">
                  {t('web.plugins.mcp.editor.transportHint')}
                </p>
              </div>
            )}
            <div className="space-y-1.5">
              <Label htmlFor="mcp-body">
                {t('web.plugins.mcp.editor.bodyLabel')}
              </Label>
              <textarea
                id="mcp-body"
                value={body}
                onChange={(e) => {
                  setBody(e.target.value)
                  if (parseError) setParseError(null)
                }}
                rows={20}
                className={cn(
                  'w-full font-mono text-[12px] rounded-md border',
                  'bg-input/40 px-3 py-2 text-foreground transition-colors',
                  'placeholder:text-muted-foreground/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-y',
                  parseError ? 'border-destructive' : 'border-border',
                )}
                spellCheck={false}
              />
              {parseError && (
                <p className="text-[11px] text-destructive">
                  {t('web.plugins.mcp.editor.invalidJson', { error: parseError })}
                </p>
              )}
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => onOpenChange(false)}
                disabled={busy}
              >
                {t('web.plugins.common.cancel')}
              </Button>
              <Button type="submit" variant="accent" size="sm" disabled={busy}>
                {busy && <Loader2 className="size-3.5 animate-spin" />}
                {mode === 'create'
                  ? t('web.plugins.common.create')
                  : t('web.plugins.common.save')}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}

// prettyMcp renders a server as the canonical pretty-printed JSON
// shown in the editor. ID is excluded — it's a directory-name field
// the user controls separately on create, immutable on edit.
function prettyMcp(s: McpServer): string {
  const { id: _id, ...rest } = s
  return JSON.stringify(rest, null, 2)
}

// parseMcp turns the textarea body back into an McpServer, defaulting
// `name` to the id when the user omitted it.
function parseMcp(body: string, id: string): McpServer {
  const parsed = JSON.parse(body)
  return {
    ...parsed,
    id,
    name: parsed.name || id,
    enabled: parsed.enabled ?? true,
  }
}

function McpSecretsSection() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({
    queryKey: ['mcp-secrets'],
    queryFn: getMcpSecrets,
  })
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [adding, setAdding] = useState(false)

  const remove = useMutation({
    mutationFn: deleteMcpSecret,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcp-secrets'] })
      toast.success(t('web.plugins.mcpSecrets.removedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.mcpSecrets.deleteFailedToast'), {
        description: err.message,
      }),
  })

  const keyCount = data?.keys.length ?? 0

  return (
    <CollapsibleSection
      id="mcp-secrets"
      icon={<ShieldCheck className="size-4 text-muted-foreground" />}
      title={t('web.plugins.mcpSecrets.title')}
      badge={
        data && (
          <span
            className={cn(
              'inline-flex items-center gap-1 text-[10px] px-1.5 py-px rounded font-mono',
              data.encrypted
                ? 'bg-state-running/15 text-state-running border border-state-running/30'
                : 'bg-amber-500/15 text-amber-300 border border-amber-500/30',
            )}
            title={
              data.encrypted
                ? t('web.plugins.mcpSecrets.encryptedTooltip')
                : t('web.plugins.mcpSecrets.plaintextTooltip')
            }
          >
            {data.encrypted
              ? t('web.plugins.mcpSecrets.encryptedBadge')
              : t('web.plugins.mcpSecrets.plaintextBadge')}
          </span>
        )
      }
      description={
        <>
          <Trans
            i18nKey="web.plugins.mcpSecrets.description"
            components={{ 1: <code />, 3: <code />, 5: <strong /> }}
          />
          {data?.path && (
            <Trans
              i18nKey="web.plugins.mcpSecrets.descriptionStored"
              values={{ path: data.path }}
              components={{ 1: <code /> }}
            />
          )}
        </>
      }
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setAdding(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          {t('web.plugins.mcpSecrets.addSecret')}
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            {t('web.plugins.common.loading')}
          </div>
        ) : keyCount === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            <Trans
              i18nKey="web.plugins.mcpSecrets.empty"
              components={{ 1: <code /> }}
            />
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.mcpSecrets.columns.key')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.mcpSecrets.columns.value')}
                </th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {data!.keys.map((k) => (
                <tr
                  key={k}
                  className="border-t border-border hover:bg-card/40"
                >
                  <td className="px-3 py-2 font-mono font-medium">{k}</td>
                  <td className="px-3 py-2 text-muted-foreground/60 font-mono tracking-widest select-none">
                    ••••••••••••
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditingKey(k)}
                        className="h-7 px-2 text-[11px] gap-1"
                        title={t('web.plugins.mcpSecrets.editTooltip')}
                      >
                        <Pencil className="size-3" />
                        {t('web.plugins.common.edit')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (
                            confirm(
                              t('web.plugins.mcpSecrets.deleteConfirm', { key: k }),
                            )
                          ) {
                            remove.mutate(k)
                          }
                        }}
                        className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="size-3" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <SecretEditor
        open={adding}
        onOpenChange={setAdding}
        mode="create"
        existingKeys={data?.keys ?? []}
      />
      <SecretEditor
        open={editingKey != null}
        onOpenChange={(v) => !v && setEditingKey(null)}
        mode="edit"
        keyName={editingKey ?? ''}
        existingKeys={data?.keys ?? []}
      />
    </CollapsibleSection>
  )
}

interface SecretEditorProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  mode: 'create' | 'edit'
  keyName?: string
  existingKeys: string[]
}

function SecretEditor({
  open,
  onOpenChange,
  mode,
  keyName,
  existingKeys,
}: SecretEditorProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [value, setValue] = useState('')

  // Reset fields each time the dialog opens. In edit mode the name is
  // locked (server-side route is keyed on it); in create mode the
  // user fills it in.
  useEffect(() => {
    if (!open) return
    setName(mode === 'edit' ? keyName ?? '' : '')
    setValue('')
  }, [open, mode, keyName])

  const save = useMutation({
    mutationFn: () => setMcpSecret(name, value),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcp-secrets'] })
      toast.success(
        mode === 'create'
          ? t('web.plugins.mcpSecrets.editor.addedToast')
          : t('web.plugins.mcpSecrets.editor.updatedToast'),
      )
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.mcpSecrets.editor.saveFailedToast'), {
        description: err.message,
      }),
  })

  const validKey = /^[A-Za-z_][A-Za-z0-9_]*$/.test(name)
  const collision =
    mode === 'create' && validKey && existingKeys.includes(name)
  const canSubmit = validKey && !collision && value.length > 0 && !save.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {mode === 'create'
              ? t('web.plugins.mcpSecrets.editor.addTitle')
              : t('web.plugins.mcpSecrets.editor.updateTitle', { key: keyName })}
          </DialogTitle>
          <DialogDescription>
            {mode === 'create'
              ? t('web.plugins.mcpSecrets.editor.addDescription')
              : t('web.plugins.mcpSecrets.editor.editDescription')}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (canSubmit) save.mutate()
          }}
          className="flex flex-col gap-3 mt-2"
        >
          <div className="space-y-1.5">
            <Label htmlFor="secret-name">
              {t('web.plugins.mcpSecrets.editor.keyLabel')}
            </Label>
            <Input
              id="secret-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('web.plugins.mcpSecrets.editor.keyPlaceholder')}
              required
              disabled={mode === 'edit'}
              className="font-mono"
              autoFocus={mode === 'create'}
            />
            {name && !validKey && (
              <p className="text-[10.5px] text-destructive">
                <Trans
                  i18nKey="web.plugins.mcpSecrets.editor.keyPattern"
                  components={{ 1: <code /> }}
                />
              </p>
            )}
            {collision && (
              <p className="text-[10.5px] text-amber-300">
                {t('web.plugins.mcpSecrets.editor.keyCollision')}
              </p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="secret-value">
              {t('web.plugins.mcpSecrets.editor.valueLabel')}
            </Label>
            <Input
              id="secret-value"
              type="password"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="••••••••••••"
              required
              autoComplete="new-password"
              spellCheck={false}
              className="font-mono"
              autoFocus={mode === 'edit'}
            />
            <p className="text-[10.5px] text-muted-foreground/80">
              {t('web.plugins.mcpSecrets.editor.valueHint')}
            </p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={save.isPending}
            >
              {t('web.plugins.common.cancel')}
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={!canSubmit}
            >
              {save.isPending && <Loader2 className="size-3.5 animate-spin" />}
              {mode === 'create'
                ? t('web.plugins.common.add')
                : t('web.plugins.common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function SkillsSection() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: skills, isLoading } = useQuery({
    queryKey: ['skills'],
    queryFn: listSkills,
  })
  const [editingId, setEditingId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const remove = useMutation({
    mutationFn: deleteSkill,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['skills'] })
      toast.success(t('web.plugins.skills.removedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.skills.deleteFailedToast'), {
        description: err.message,
      }),
  })

  return (
    <CollapsibleSection
      id="agent-skills"
      icon={<Sparkles className="size-4 text-muted-foreground" />}
      title={t('web.plugins.skills.title')}
      description={
        <Trans
          i18nKey="web.plugins.skills.description"
          components={{ 1: <code />, 3: <strong />, 5: <code /> }}
        />
      }
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          {t('web.plugins.skills.newSkill')}
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            {t('web.plugins.common.loading')}
          </div>
        ) : (skills ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            {t('web.plugins.skills.empty')}
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.skills.columns.id')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.skills.columns.description')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.skills.columns.source')}
                </th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {skills!.map((s) => (
                <tr
                  key={s.id}
                  className="border-t border-border hover:bg-card/40 align-top"
                >
                  <td className="px-3 py-2 font-mono text-[11.5px] font-medium">
                    {s.id}
                  </td>
                  <td className="px-3 py-2 text-muted-foreground/90">
                    {s.description || (
                      <span className="italic opacity-60">
                        {t('web.plugins.skills.noDescription')}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    {s.source === 'builtin' ? (
                      <span
                        className="inline-flex items-center gap-1 text-[10px] text-muted-foreground/70"
                        title={t('web.plugins.skills.builtinTooltip')}
                      >
                        <Lock className="size-3" />
                        {t('web.plugins.skills.builtinBadge')}
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 text-[10px]">
                        <span className="text-state-running">
                          {t('web.plugins.skills.vaultBadge')}
                        </span>
                        {s.overrides_builtin && (
                          <span
                            className="text-[9px] text-amber-400 px-1 py-px rounded bg-amber-500/10 border border-amber-500/30"
                            title={t('web.plugins.skills.overridesBuiltinTooltip')}
                          >
                            {t('web.plugins.skills.overridesBuiltin')}
                          </span>
                        )}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditingId(s.id)}
                        className="h-7 px-2 text-[11px] gap-1"
                        title={
                          s.source === 'builtin'
                            ? t('web.plugins.skills.customizeTooltip')
                            : t('web.plugins.skills.editTooltip')
                        }
                      >
                        <Pencil className="size-3" />
                        {s.source === 'builtin'
                          ? t('web.plugins.skills.customize')
                          : t('web.plugins.common.edit')}
                      </Button>
                      {s.source === 'vault' && s.overrides_builtin && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            if (
                              confirm(
                                t('web.plugins.skills.resetConfirm', { id: s.id }),
                              )
                            ) {
                              remove.mutate(s.id)
                            }
                          }}
                          className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-foreground"
                          title={t('web.plugins.skills.resetTooltip')}
                        >
                          <RotateCcw className="size-3" />
                          {t('web.plugins.skills.reset')}
                        </Button>
                      )}
                      {s.source === 'vault' && !s.overrides_builtin && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            if (
                              confirm(
                                t('web.plugins.skills.deleteConfirm', { id: s.id }),
                              )
                            ) {
                              remove.mutate(s.id)
                            }
                          }}
                          className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-destructive"
                        >
                          <Trash2 className="size-3" />
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <SkillEditor
        open={creating}
        onOpenChange={setCreating}
        mode="create"
      />
      <SkillEditor
        open={editingId != null}
        onOpenChange={(v) => !v && setEditingId(null)}
        mode="edit"
        editingId={editingId}
      />
    </CollapsibleSection>
  )
}

interface SkillEditorProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  mode: 'create' | 'edit'
  editingId?: string | null
}

function SkillEditor({ open, onOpenChange, mode, editingId }: SkillEditorProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [id, setId] = useState('')
  const [body, setBody] = useState('')

  const { data: existing, isLoading } = useQuery({
    queryKey: ['skill', editingId],
    queryFn: () => getSkill(editingId!),
    enabled: open && mode === 'edit' && !!editingId,
  })

  useEffect(() => {
    if (mode === 'create') {
      setId('')
      setBody('')
    } else if (existing) {
      setId(existing.id)
      setBody(existing.body ?? '')
    }
  }, [mode, existing?.id])

  // "Customize" flow: editing a built-in. The textarea is editable
  // and Save writes to the vault (PUT, which upserts), creating an
  // override at the same id. The user can revert later with the
  // "Reset" action on the vault row.
  const isCustomizingBuiltin = mode === 'edit' && existing?.source === 'builtin'

  const create = useMutation({
    mutationFn: () => createSkill(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['skills'] })
      toast.success(t('web.plugins.skills.editor.createdToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.skills.editor.createFailedToast'), {
        description: err.message,
      }),
  })

  const update = useMutation({
    mutationFn: () => updateSkill(editingId!, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['skills'] })
      qc.invalidateQueries({ queryKey: ['skill', editingId] })
      toast.success(
        isCustomizingBuiltin
          ? t('web.plugins.skills.editor.savedOverrideToast')
          : t('web.plugins.skills.editor.savedToast'),
      )
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.skills.editor.saveFailedToast'), {
        description: err.message,
      }),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (mode === 'create') create.mutate()
    else update.mutate()
  }

  const busy = create.isPending || update.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[min(92vw,860px)] w-[min(92vw,860px)]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {isCustomizingBuiltin && (
              <Lock className="size-3.5 text-muted-foreground" />
            )}
            {mode === 'create'
              ? t('web.plugins.skills.editor.createTitle')
              : isCustomizingBuiltin
                ? t('web.plugins.skills.editor.customizeTitle', { id: editingId })
                : t('web.plugins.skills.editor.editTitle', { id: editingId })}
          </DialogTitle>
          <DialogDescription>
            {isCustomizingBuiltin
              ? t('web.plugins.skills.editor.customizeDescription')
              : t('web.plugins.skills.editor.editDescription')}
          </DialogDescription>
        </DialogHeader>
        {isLoading && mode === 'edit' ? (
          <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-6">
            <Loader2 className="size-3 animate-spin" />
            {t('web.plugins.common.loading')}
          </div>
        ) : (
          <form onSubmit={submit} className="flex flex-col gap-3">
            {mode === 'create' && (
              <div className="space-y-1.5">
                <Label htmlFor="skill-id">
                  {t('web.plugins.skills.editor.idLabel')}
                </Label>
                <Input
                  id="skill-id"
                  value={id}
                  onChange={(e) => setId(e.target.value)}
                  placeholder={t('web.plugins.skills.editor.idPlaceholder')}
                  required
                  className="font-mono"
                />
                <p className="text-[10.5px] text-muted-foreground/80">
                  <Trans
                    i18nKey="web.plugins.skills.editor.idHint"
                    components={{ 1: <code /> }}
                  />
                </p>
              </div>
            )}
            <div className="space-y-1.5">
              <Label htmlFor="skill-body">
                {t('web.plugins.skills.editor.bodyLabel')}
              </Label>
              <textarea
                id="skill-body"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                rows={20}
                className={cn(
                  'w-full font-mono text-[12px] rounded-md border border-border',
                  'bg-input/40 px-3 py-2 text-foreground transition-colors',
                  'placeholder:text-muted-foreground/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-y',
                )}
                placeholder="---\nname: my-helper\ndescription: One-line trigger description.\n---\n\n# my-helper\n\n..."
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => onOpenChange(false)}
                disabled={busy}
              >
                {t('web.plugins.common.cancel')}
              </Button>
              <Button type="submit" variant="accent" size="sm" disabled={busy}>
                {busy && <Loader2 className="size-3.5 animate-spin" />}
                {mode === 'create'
                  ? t('web.plugins.common.create')
                  : isCustomizingBuiltin
                    ? t('web.plugins.skills.editor.saveAsOverride')
                    : t('web.plugins.common.save')}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}

function CustomTasksSection() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: tasks, isLoading } = useQuery({
    queryKey: ['custom-tasks-all'],
    queryFn: () => listCustomTasks({ all: true }),
  })
  const [editing, setEditing] = useState<CustomTask | null>(null)
  const [creating, setCreating] = useState(false)

  const remove = useMutation({
    mutationFn: deleteCustomTask,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['custom-tasks-all'] })
      qc.invalidateQueries({ queryKey: ['custom-tasks'] })
      toast.success(t('web.plugins.customTasks.removedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.customTasks.deleteFailedToast'), {
        description: err.message,
      }),
  })

  return (
    <CollapsibleSection
      id="custom-tasks"
      icon={<Play className="size-4 text-muted-foreground" />}
      title={t('web.plugins.customTasks.title')}
      description={t('web.plugins.customTasks.description')}
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          {t('web.plugins.customTasks.addTask')}
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            {t('web.plugins.common.loading')}
          </div>
        ) : (tasks ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            {t('web.plugins.customTasks.empty')}
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.customTasks.columns.name')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.customTasks.columns.command')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.customTasks.columns.scope')}
                </th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {tasks!.map((task) => (
                <tr
                  key={task.id}
                  className="border-t border-border hover:bg-card/40 align-top"
                >
                  <td className="px-3 py-2">
                    <div className="font-medium">{task.name}</div>
                    {task.description && (
                      <div className="text-[10px] text-muted-foreground/70 italic">
                        {task.description}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2 font-mono text-[11px] break-all">
                    {task.command}
                  </td>
                  <td className="px-3 py-2 font-mono text-[10px]">
                    {task.cwd ? (
                      <span title={task.cwd}>{trimPath(task.cwd)}</span>
                    ) : (
                      <span className="text-muted-foreground/70">
                        {t('web.plugins.customTasks.globalScope')}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditing(task)}
                        className="h-7 px-2 text-[11px] gap-1"
                      >
                        <Pencil className="size-3" />
                        {t('web.plugins.common.edit')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (
                            confirm(
                              t('web.plugins.customTasks.deleteConfirm', {
                                name: task.name,
                              }),
                            )
                          ) {
                            remove.mutate(task.id)
                          }
                        }}
                        className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="size-3" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <CustomTaskDialog
        open={creating}
        onOpenChange={setCreating}
        mode="create"
      />
      <CustomTaskDialog
        open={editing != null}
        onOpenChange={(v) => !v && setEditing(null)}
        mode="edit"
        task={editing ?? undefined}
      />
    </CollapsibleSection>
  )
}

function trimPath(p: string): string {
  const parts = p.split('/').filter(Boolean)
  if (parts.length <= 2) return p
  return '…/' + parts.slice(-2).join('/')
}

interface CustomTaskDialogProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  mode: 'create' | 'edit'
  task?: CustomTask
}

function CustomTaskDialog({ open, onOpenChange, mode, task }: CustomTaskDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [name, setName] = useState(task?.name ?? '')
  const [command, setCommand] = useState(task?.command ?? '')
  const [description, setDescription] = useState(task?.description ?? '')
  const [cwd, setCwd] = useState(task?.cwd ?? '')

  useEffect(() => {
    setName(task?.name ?? '')
    setCommand(task?.command ?? '')
    setDescription(task?.description ?? '')
    setCwd(task?.cwd ?? '')
  }, [task?.id])

  const create = useMutation({
    mutationFn: () =>
      createCustomTask({ name, command, description, cwd: cwd || undefined }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['custom-tasks-all'] })
      qc.invalidateQueries({ queryKey: ['custom-tasks'] })
      toast.success(t('web.plugins.customTasks.dialog.addedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.customTasks.dialog.addFailedToast'), {
        description: err.message,
      }),
  })

  const update = useMutation({
    mutationFn: () =>
      updateCustomTask(task!.id, { name, command, description, cwd }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['custom-tasks-all'] })
      qc.invalidateQueries({ queryKey: ['custom-tasks'] })
      toast.success(t('web.plugins.customTasks.dialog.updatedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.customTasks.dialog.updateFailedToast'), {
        description: err.message,
      }),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (mode === 'create') create.mutate()
    else update.mutate()
  }

  const busy = create.isPending || update.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {mode === 'create'
              ? t('web.plugins.customTasks.dialog.addTitle')
              : t('web.plugins.customTasks.dialog.editTitle', { name: task?.name })}
          </DialogTitle>
          <DialogDescription>
            {t('web.plugins.customTasks.dialog.description')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-3 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="task-name">
              {t('web.plugins.customTasks.dialog.nameLabel')}
            </Label>
            <Input
              id="task-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.namePlaceholder')}
              required
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-cmd">
              {t('web.plugins.customTasks.dialog.commandLabel')}
            </Label>
            <Textarea
              id="task-cmd"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.commandPlaceholder')}
              rows={2}
              required
              className="font-mono text-[12px]"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-desc">
              {t('web.plugins.customTasks.dialog.descLabel')}
            </Label>
            <Input
              id="task-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.descPlaceholder')}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-cwd">
              {t('web.plugins.customTasks.dialog.cwdLabel')}
            </Label>
            <Input
              id="task-cwd"
              value={cwd}
              onChange={(e) => setCwd(e.target.value)}
              placeholder={t('web.plugins.customTasks.dialog.cwdPlaceholder')}
              className="font-mono text-[12px]"
            />
            <p className="text-[10.5px] text-muted-foreground/80">
              {t('web.plugins.customTasks.dialog.cwdHint')}
            </p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={busy}
            >
              {t('web.plugins.common.cancel')}
            </Button>
            <Button type="submit" variant="accent" size="sm" disabled={busy}>
              {busy && <Loader2 className="size-3.5 animate-spin" />}
              {mode === 'create'
                ? t('web.plugins.common.add')
                : t('web.plugins.common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function GitHostsSection() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: hosts, isLoading } = useQuery({
    queryKey: ['git-hosts'],
    queryFn: listGitHosts,
  })
  const [editing, setEditing] = useState<GitHost | null>(null)
  const [creating, setCreating] = useState(false)

  const remove = useMutation({
    mutationFn: deleteGitHost,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['git-hosts'] })
      toast.success(t('web.plugins.gitHosts.removedToast'))
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.gitHosts.deleteFailedToast'), {
        description: err.message,
      }),
  })

  return (
    <CollapsibleSection
      id="git-hosts"
      icon={<GitBranch className="size-4 text-muted-foreground" />}
      title={t('web.plugins.gitHosts.title')}
      description={
        <Trans
          i18nKey="web.plugins.gitHosts.description"
          components={{ 1: <strong /> }}
        />
      }
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          {t('web.plugins.gitHosts.addHost')}
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            {t('web.plugins.common.loading')}
          </div>
        ) : (hosts ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground whitespace-pre-line">
            {t('web.plugins.gitHosts.empty')}
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.gitHosts.columns.host')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.gitHosts.columns.kind')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.gitHosts.columns.token')}
                </th>
                <th className="text-left px-3 py-2 font-medium">
                  {t('web.plugins.gitHosts.columns.enabled')}
                </th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {hosts!.map((h) => (
                <tr
                  key={h.id}
                  className="border-t border-border hover:bg-card/40"
                >
                  <td className="px-3 py-2">
                    <div className="font-medium font-mono">{h.host}</div>
                    {h.name && (
                      <div className="text-[10px] text-muted-foreground/70">
                        {h.name}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2 font-mono">{h.kind}</td>
                  <td className="px-3 py-2 font-mono text-muted-foreground">
                    {h.token_mask || '—'}
                  </td>
                  <td className="px-3 py-2">
                    <span
                      className={cn(
                        'inline-flex items-center gap-1 text-[10px]',
                        h.enabled
                          ? 'text-state-running'
                          : 'text-muted-foreground/60',
                      )}
                    >
                      <span
                        className={cn(
                          'size-1.5 rounded-full',
                          h.enabled
                            ? 'bg-state-running'
                            : 'bg-muted-foreground/40',
                        )}
                      />
                      {h.enabled
                        ? t('web.plugins.gitHosts.statusEnabled')
                        : t('web.plugins.gitHosts.statusDisabled')}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditing(h)}
                        className="h-7 px-2 text-[11px] gap-1"
                      >
                        <Pencil className="size-3" />
                        {t('web.plugins.common.edit')}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (
                            confirm(
                              t('web.plugins.gitHosts.deleteConfirm', {
                                host: h.host,
                              }),
                            )
                          ) {
                            remove.mutate(h.id)
                          }
                        }}
                        className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="size-3" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <GitHostDialog
        open={creating}
        onOpenChange={setCreating}
        mode="create"
      />
      <GitHostDialog
        open={editing != null}
        onOpenChange={(v) => !v && setEditing(null)}
        mode="edit"
        host={editing ?? undefined}
      />
    </CollapsibleSection>
  )
}

interface GitHostDialogProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  mode: 'create' | 'edit'
  host?: GitHost
}

function GitHostDialog({ open, onOpenChange, mode, host }: GitHostDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [kind, setKind] = useState<GitHostKind>(host?.kind ?? 'github')
  const [hostName, setHostName] = useState(host?.host ?? '')
  const [name, setName] = useState(host?.name ?? '')
  const [token, setToken] = useState('')
  const [enabled, setEnabled] = useState(host?.enabled ?? true)

  // Sync form fields when the editing target changes (dialog re-uses
  // mounted state across opens).
  useEffect(() => {
    setKind(host?.kind ?? 'github')
    setHostName(host?.host ?? '')
    setName(host?.name ?? '')
    setToken('')
    setEnabled(host?.enabled ?? true)
  }, [host?.id])

  const create = useMutation({
    mutationFn: () => createGitHost({ kind, host: hostName, name, token }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['git-hosts'] })
      toast.success(t('web.plugins.gitHosts.dialog.addedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.gitHosts.dialog.addFailedToast'), {
        description: err.message,
      }),
  })

  const update = useMutation({
    mutationFn: () =>
      updateGitHost(host!.id, {
        kind,
        host: hostName,
        name,
        enabled,
        token: token || undefined,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['git-hosts'] })
      toast.success(t('web.plugins.gitHosts.dialog.updatedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error(t('web.plugins.gitHosts.dialog.updateFailedToast'), {
        description: err.message,
      }),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (mode === 'create') create.mutate()
    else update.mutate()
  }

  const busy = create.isPending || update.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {mode === 'create'
              ? t('web.plugins.gitHosts.dialog.addTitle')
              : t('web.plugins.gitHosts.dialog.editTitle', { host: host?.host })}
          </DialogTitle>
          <DialogDescription>
            {t('web.plugins.gitHosts.dialog.description')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-3 mt-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="kind">
                {t('web.plugins.gitHosts.dialog.kindLabel')}
              </Label>
              <Select
                value={kind}
                onValueChange={(v) => setKind(v as GitHostKind)}
              >
                <SelectTrigger id="kind">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="github">
                    {t('web.plugins.gitHosts.dialog.kindGitHub')}
                  </SelectItem>
                  <SelectItem value="gitea">
                    {t('web.plugins.gitHosts.dialog.kindGitea')}
                  </SelectItem>
                  <SelectItem value="gitlab">
                    {t('web.plugins.gitHosts.dialog.kindGitLab')}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="host">
                {t('web.plugins.gitHosts.dialog.hostLabel')}
              </Label>
              <Input
                id="host"
                value={hostName}
                onChange={(e) => setHostName(e.target.value)}
                placeholder={t('web.plugins.gitHosts.dialog.hostPlaceholder')}
                required
                autoFocus
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="name">
              {t('web.plugins.gitHosts.dialog.displayNameLabel')}
            </Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('web.plugins.gitHosts.dialog.displayNamePlaceholder')}
            />
          </div>
          <div className="space-y-1.5">
            <Label
              htmlFor="token"
              className="flex items-center gap-1.5 text-foreground"
            >
              <KeyRound className="size-3 text-muted-foreground" />
              {mode === 'create'
                ? t('web.plugins.gitHosts.dialog.tokenLabel')
                : t('web.plugins.gitHosts.dialog.newTokenLabel')}
            </Label>
            <Input
              id="token"
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder={
                mode === 'create'
                  ? t('web.plugins.gitHosts.dialog.tokenPlaceholder')
                  : t('web.plugins.gitHosts.dialog.tokenPlaceholderEdit')
              }
              required={mode === 'create'}
              className="font-mono"
            />
            <p className="text-[10.5px] text-muted-foreground/80">
              <Trans
                i18nKey="web.plugins.gitHosts.dialog.tokenHint"
                components={{ 1: <code />, 3: <code />, 5: <code /> }}
              />
            </p>
          </div>
          {mode === 'edit' && (
            <div className="flex items-center gap-2">
              <Switch
                id="enabled"
                checked={enabled}
                onCheckedChange={setEnabled}
              />
              <Label htmlFor="enabled" className="text-[12px]">
                {t('web.plugins.gitHosts.dialog.enabledLabel')}
              </Label>
            </div>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={busy}
            >
              {t('web.plugins.common.cancel')}
            </Button>
            <Button type="submit" variant="accent" size="sm" disabled={busy}>
              {busy && <Loader2 className="size-3.5 animate-spin" />}
              {mode === 'create'
                ? t('web.plugins.common.add')
                : t('web.plugins.common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

