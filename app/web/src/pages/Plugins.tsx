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
  getMcpSecrets,
  setMcpSecret,
  deleteMcpSecret,
  defaultMcpServer,
  type McpServer,
} from '@/lib/mcps'
import { cn } from '@/lib/utils'

// PluginsPage is the Workspace nav entry that hosts inspector-tab
// configuration. v1 ships the Git-host token manager; future panels
// (Logs filters, Tasks aliases, …) will land as additional sections
// here rather than spawning more nav items.
export function PluginsPage() {
  return (
    <div className="h-full flex flex-col p-6 gap-4 overflow-y-auto">
      <header className="flex flex-col gap-1">
        <h1 className="text-[18px] font-semibold tracking-tight">
          Inspector plugins
        </h1>
        <p className="text-[12px] text-muted-foreground max-w-[640px]">
          Configure data sources surfaced in the right-hand Inspector panel
          when a session is open. Each plugin is admin-only and shared
          across all sessions. Click a section header to collapse it.
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
      toast.success('MCP server removed')
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  const toggle = useMutation({
    mutationFn: async (s: McpServer) =>
      updateMcp(s.id, { ...s, enabled: !s.enabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcps'] })
    },
    onError: (err: Error) =>
      toast.error('Toggle failed', { description: err.message }),
  })

  return (
    <CollapsibleSection
      id="mcp-servers"
      icon={<Plug className="size-4 text-muted-foreground" />}
      title="MCP servers"
      description={
        <>
          Model Context Protocol servers injected into every spawn (claude /
          codex). Vault entries live at{' '}
          <code>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</code>; secrets
          (referenced as <code>${'{KEY}'}</code> in env / headers) come from
          the <strong>MCP secrets</strong> section below.
        </>
      }
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          New server
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : (servers ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            No MCP servers yet. Add one to expose extra tools to your agent
            sessions.
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">Name</th>
                <th className="text-left px-3 py-2 font-medium">Transport</th>
                <th className="text-left px-3 py-2 font-medium">Spec</th>
                <th className="text-left px-3 py-2 font-medium">Enabled</th>
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
                  </td>
                  <td className="px-3 py-2 font-mono text-[10.5px] break-all max-w-[260px]">
                    {s.transport === 'sse' || s.transport === 'http'
                      ? s.url || <span className="opacity-50">no url</span>
                      : s.command
                        ? `${s.command}${s.args?.length ? ' ' + s.args.join(' ') : ''}`
                        : <span className="opacity-50">no command</span>}
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
                        onClick={() => setEditingId(s.id)}
                        className="h-7 px-2 text-[11px] gap-1"
                      >
                        <Pencil className="size-3" />
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (confirm(`Delete MCP server "${s.id}"?`)) {
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
  const qc = useQueryClient()
  const [id, setId] = useState('')
  const [body, setBody] = useState('')
  const [parseError, setParseError] = useState<string | null>(null)

  const { data: existing, isLoading } = useQuery({
    queryKey: ['mcp', editingId],
    queryFn: () => getMcp(editingId!),
    enabled: open && mode === 'edit' && !!editingId,
  })

  useEffect(() => {
    if (mode === 'create' && open) {
      setId('')
      setBody(prettyMcp(defaultMcpServer()))
      setParseError(null)
    } else if (mode === 'edit' && existing) {
      setId(existing.id)
      setBody(prettyMcp(existing))
      setParseError(null)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, existing?.id, open])

  const create = useMutation({
    mutationFn: async () => {
      const parsed = parseMcp(body, id)
      return createMcp(id, parsed)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcps'] })
      toast.success('MCP server created')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Create failed', { description: err.message }),
  })

  const update = useMutation({
    mutationFn: async () => {
      const parsed = parseMcp(body, editingId!)
      return updateMcp(editingId!, parsed)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['mcps'] })
      qc.invalidateQueries({ queryKey: ['mcp', editingId] })
      toast.success('MCP server saved')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Save failed', { description: err.message }),
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
            {mode === 'create' ? 'New MCP server' : `Edit MCP: ${editingId}`}
          </DialogTitle>
          <DialogDescription>
            JSON shape: <code>command</code>+<code>args</code>+<code>env</code>{' '}
            for stdio (default), or <code>transport</code> +<code> url</code>+
            <code>headers</code> for sse / http. Reference secrets as{' '}
            <code>${'{API_KEY}'}</code> — they get substituted at spawn time
            from the secrets file.
          </DialogDescription>
        </DialogHeader>
        {isLoading && mode === 'edit' ? (
          <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-6">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : (
          <form onSubmit={submit} className="flex flex-col gap-3">
            {mode === 'create' && (
              <div className="space-y-1.5">
                <Label htmlFor="mcp-id">ID</Label>
                <Input
                  id="mcp-id"
                  value={id}
                  onChange={(e) => setId(e.target.value)}
                  placeholder="filesystem"
                  required
                  className="font-mono"
                />
                <p className="text-[10.5px] text-muted-foreground/80">
                  Lowercase / digits / dash / underscore. Becomes both the
                  directory name and the default <code>name</code>.
                </p>
              </div>
            )}
            <div className="space-y-1.5">
              <Label htmlFor="mcp-body">mcp.json</Label>
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
                  Invalid JSON: {parseError}
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
                Cancel
              </Button>
              <Button type="submit" variant="accent" size="sm" disabled={busy}>
                {busy && <Loader2 className="size-3.5 animate-spin" />}
                {mode === 'create' ? 'Create' : 'Save'}
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
      toast.success('Secret removed')
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  const keyCount = data?.keys.length ?? 0

  return (
    <CollapsibleSection
      id="mcp-secrets"
      icon={<ShieldCheck className="size-4 text-muted-foreground" />}
      title="MCP secrets"
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
                ? 'AES-GCM encrypted on disk; key stored in OS keychain'
                : 'OS keychain unavailable — file is plaintext on disk. Check the gateway log.'
            }
          >
            {data.encrypted ? 'encrypted' : 'plaintext'}
          </span>
        )
      }
      description={
        <>
          Values referenced from <code>${'{KEY}'}</code> placeholders in any{' '}
          <code>mcp.json</code> get substituted at spawn time.{' '}
          <strong>Saved values are never returned over the API</strong> — you
          can overwrite or delete them but not read them back.
          {data?.path && (
            <>
              {' '}
              Stored at <code>{data.path}</code>.
            </>
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
          Add secret
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : keyCount === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            No secrets stored. Add one to start referencing it as{' '}
            <code>${'{KEY}'}</code> in your MCP server configs.
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">Key</th>
                <th className="text-left px-3 py-2 font-medium">Value</th>
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
                        title="Overwrite the stored value"
                      >
                        <Pencil className="size-3" />
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (
                            confirm(
                              `Delete secret "${k}"? Any mcp.json that references \${${k}} will fall back to the literal placeholder until you set a new value.`,
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
      toast.success(mode === 'create' ? 'Secret added' : 'Secret updated')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Save failed', { description: err.message }),
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
            {mode === 'create' ? 'Add secret' : `Update ${keyName}`}
          </DialogTitle>
          <DialogDescription>
            {mode === 'create'
              ? 'Stored encrypted on disk if the OS keychain is available. Reference it from any mcp.json env / headers / args / url with ${KEY}.'
              : 'Enter the new value to overwrite. The previous value cannot be recovered.'}
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
            <Label htmlFor="secret-name">Key</Label>
            <Input
              id="secret-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="BRAVE_API_KEY"
              required
              disabled={mode === 'edit'}
              className="font-mono"
              autoFocus={mode === 'create'}
            />
            {name && !validKey && (
              <p className="text-[10.5px] text-destructive">
                Must match <code>[A-Za-z_][A-Za-z0-9_]*</code>
              </p>
            )}
            {collision && (
              <p className="text-[10.5px] text-amber-300">
                Already exists — use Edit instead, or pick a different name.
              </p>
            )}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="secret-value">Value</Label>
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
              Hidden as you type. Saved value is never returned over the API.
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
              Cancel
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={!canSubmit}
            >
              {save.isPending && <Loader2 className="size-3.5 animate-spin" />}
              {mode === 'create' ? 'Add' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function SkillsSection() {
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
      toast.success('Skill removed')
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  return (
    <CollapsibleSection
      id="agent-skills"
      icon={<Sparkles className="size-4 text-muted-foreground" />}
      title="Agent skills"
      description={
        <>
          Reusable capabilities injected into Claude sessions as a Tier 1
          index — the agent loads full SKILL.md on demand via{' '}
          <code>opendray skill describe &lt;id&gt;</code>. Built-ins ship in
          the binary but can be <strong>customized</strong> — your edits land
          at <code>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</code> and
          override the embedded version. Use Reset to revert.
        </>
      }
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          New skill
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : (skills ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            No skills found.
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">ID</th>
                <th className="text-left px-3 py-2 font-medium">Description</th>
                <th className="text-left px-3 py-2 font-medium">Source</th>
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
                      <span className="italic opacity-60">no description</span>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    {s.source === 'builtin' ? (
                      <span
                        className="inline-flex items-center gap-1 text-[10px] text-muted-foreground/70"
                        title="Embedded in the opendray binary — click Customize to override in your vault"
                      >
                        <Lock className="size-3" />
                        builtin
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 text-[10px]">
                        <span className="text-state-running">vault</span>
                        {s.overrides_builtin && (
                          <span
                            className="text-[9px] text-amber-400 px-1 py-px rounded bg-amber-500/10 border border-amber-500/30"
                            title="This vault skill overrides the built-in version of the same id"
                          >
                            overrides builtin
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
                            ? 'Open the SKILL.md and save changes as a vault override'
                            : 'Edit this vault skill'
                        }
                      >
                        <Pencil className="size-3" />
                        {s.source === 'builtin' ? 'Customize' : 'Edit'}
                      </Button>
                      {s.source === 'vault' && s.overrides_builtin && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            if (
                              confirm(
                                `Reset "${s.id}" to the built-in version? This deletes your vault SKILL.md and falls back to the embedded copy.`,
                              )
                            ) {
                              remove.mutate(s.id)
                            }
                          }}
                          className="h-7 px-2 text-[11px] gap-1 text-muted-foreground hover:text-foreground"
                          title="Delete vault override and fall back to the built-in version"
                        >
                          <RotateCcw className="size-3" />
                          Reset
                        </Button>
                      )}
                      {s.source === 'vault' && !s.overrides_builtin && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            if (
                              confirm(
                                `Delete skill "${s.id}" from your vault? This removes the SKILL.md file.`,
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
      toast.success('Skill created')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Create failed', { description: err.message }),
  })

  const update = useMutation({
    mutationFn: () => updateSkill(editingId!, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['skills'] })
      qc.invalidateQueries({ queryKey: ['skill', editingId] })
      toast.success(
        isCustomizingBuiltin ? 'Saved as vault override' : 'Skill saved',
      )
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Save failed', { description: err.message }),
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
              ? 'New skill'
              : isCustomizingBuiltin
                ? `Customize built-in: ${editingId}`
                : `Edit skill: ${editingId}`}
          </DialogTitle>
          <DialogDescription>
            {isCustomizingBuiltin
              ? 'You\'re viewing a built-in skill embedded in opendray. Saving will create a vault override at the same id — your edits live under ~/.opendray/vault/skills/<id>/SKILL.md and shadow the built-in until you Reset.'
              : 'SKILL.md format — frontmatter with name + description, then markdown instructions. The description appears in the agent\'s Tier 1 index.'}
          </DialogDescription>
        </DialogHeader>
        {isLoading && mode === 'edit' ? (
          <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-6">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : (
          <form onSubmit={submit} className="flex flex-col gap-3">
            {mode === 'create' && (
              <div className="space-y-1.5">
                <Label htmlFor="skill-id">ID</Label>
                <Input
                  id="skill-id"
                  value={id}
                  onChange={(e) => setId(e.target.value)}
                  placeholder="my-helper"
                  required
                  className="font-mono"
                />
                <p className="text-[10.5px] text-muted-foreground/80">
                  Lowercase / digits / dash / underscore. Becomes the directory
                  name under <code>~/.opendray/vault/skills/&lt;id&gt;/</code>.
                </p>
              </div>
            )}
            <div className="space-y-1.5">
              <Label htmlFor="skill-body">SKILL.md</Label>
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
                Cancel
              </Button>
              <Button type="submit" variant="accent" size="sm" disabled={busy}>
                {busy && <Loader2 className="size-3.5 animate-spin" />}
                {mode === 'create'
                  ? 'Create'
                  : isCustomizingBuiltin
                    ? 'Save as vault override'
                    : 'Save'}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}

function CustomTasksSection() {
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
      toast.success('Task removed')
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  return (
    <CollapsibleSection
      id="custom-tasks"
      icon={<Play className="size-4 text-muted-foreground" />}
      title="Custom tasks"
      description="Click-to-run shortcuts surfaced in the Tasks tab. Leave cwd blank for global tasks visible in every session, or pin to an absolute path to scope."
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          Add task
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : (tasks ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            No custom tasks yet.
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">Name</th>
                <th className="text-left px-3 py-2 font-medium">Command</th>
                <th className="text-left px-3 py-2 font-medium">Scope</th>
                <th className="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {tasks!.map((t) => (
                <tr
                  key={t.id}
                  className="border-t border-border hover:bg-card/40 align-top"
                >
                  <td className="px-3 py-2">
                    <div className="font-medium">{t.name}</div>
                    {t.description && (
                      <div className="text-[10px] text-muted-foreground/70 italic">
                        {t.description}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2 font-mono text-[11px] break-all">
                    {t.command}
                  </td>
                  <td className="px-3 py-2 font-mono text-[10px]">
                    {t.cwd ? (
                      <span title={t.cwd}>{trimPath(t.cwd)}</span>
                    ) : (
                      <span className="text-muted-foreground/70">global</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditing(t)}
                        className="h-7 px-2 text-[11px] gap-1"
                      >
                        <Pencil className="size-3" />
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (confirm(`Delete custom task "${t.name}"?`)) {
                            remove.mutate(t.id)
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
      toast.success('Task added')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Add failed', { description: err.message }),
  })

  const update = useMutation({
    mutationFn: () =>
      updateCustomTask(task!.id, { name, command, description, cwd }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['custom-tasks-all'] })
      qc.invalidateQueries({ queryKey: ['custom-tasks'] })
      toast.success('Task updated')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Update failed', { description: err.message }),
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
            {mode === 'create' ? 'Add custom task' : `Edit ${task?.name}`}
          </DialogTitle>
          <DialogDescription>
            The command is sent verbatim into the session's terminal. Same as
            typing it at the prompt and pressing Enter.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-3 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="task-name">Name</Label>
            <Input
              id="task-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="dev"
              required
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-cmd">Command</Label>
            <Textarea
              id="task-cmd"
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="docker compose up --build"
              rows={2}
              required
              className="font-mono text-[12px]"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-desc">Description (optional)</Label>
            <Input
              id="task-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Boots dev infra and tails logs"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="task-cwd">Cwd scope (optional)</Label>
            <Input
              id="task-cwd"
              value={cwd}
              onChange={(e) => setCwd(e.target.value)}
              placeholder="/Users/me/projects/foo  (blank = global)"
              className="font-mono text-[12px]"
            />
            <p className="text-[10.5px] text-muted-foreground/80">
              Blank = visible in every session. Otherwise the task only shows
              when the session's cwd matches this absolute path.
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
              Cancel
            </Button>
            <Button type="submit" variant="accent" size="sm" disabled={busy}>
              {busy && <Loader2 className="size-3.5 animate-spin" />}
              {mode === 'create' ? 'Add' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function GitHostsSection() {
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
      toast.success('Git host removed')
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  return (
    <CollapsibleSection
      id="git-hosts"
      icon={<GitBranch className="size-4 text-muted-foreground" />}
      title="Git hosts"
      description={
        <>
          One token per host — used by the Git tab to fetch pull requests
          <strong> and by the Notes vault sync</strong> when its remote uses
          HTTPS to a private repo on the same host. GitHub.com, self-hosted
          GitHub Enterprise, Gitea, and GitLab are supported.
        </>
      }
      action={
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreating(true)}
          className="gap-1"
        >
          <Plus className="size-3.5" />
          Add host
        </Button>
      }
    >
      <div className="rounded-md border border-border overflow-hidden">
        {isLoading ? (
          <div className="px-4 py-6 flex items-center gap-2 text-[12px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            Loading…
          </div>
        ) : (hosts ?? []).length === 0 ? (
          <div className="px-4 py-8 text-center text-[12px] text-muted-foreground">
            No git hosts configured.
            <br />
            Add one to enable the PR list in the inspector's Git tab.
          </div>
        ) : (
          <table className="w-full text-[12px]">
            <thead className="bg-card/40 text-[10px] uppercase tracking-wider text-muted-foreground/70">
              <tr>
                <th className="text-left px-3 py-2 font-medium">Host</th>
                <th className="text-left px-3 py-2 font-medium">Kind</th>
                <th className="text-left px-3 py-2 font-medium">Token</th>
                <th className="text-left px-3 py-2 font-medium">Enabled</th>
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
                      {h.enabled ? 'enabled' : 'disabled'}
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
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (
                            confirm(
                              `Remove git host ${h.host}? PR queries against this host will stop working.`,
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
      toast.success('Git host added')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Add failed', { description: err.message }),
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
      toast.success('Git host updated')
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Update failed', { description: err.message }),
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
            {mode === 'create' ? 'Add git host' : `Edit ${host?.host}`}
          </DialogTitle>
          <DialogDescription>
            Token is stored on the gateway. Used only for read-only API
            calls (list PRs, etc.).
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-3 mt-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="kind">Kind</Label>
              <Select
                value={kind}
                onValueChange={(v) => setKind(v as GitHostKind)}
              >
                <SelectTrigger id="kind">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="github">GitHub</SelectItem>
                  <SelectItem value="gitea">Gitea</SelectItem>
                  <SelectItem value="gitlab">GitLab</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="host">Host</Label>
              <Input
                id="host"
                value={hostName}
                onChange={(e) => setHostName(e.target.value)}
                placeholder="github.com"
                required
                autoFocus
              />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="name">Display name (optional)</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Personal"
            />
          </div>
          <div className="space-y-1.5">
            <Label
              htmlFor="token"
              className="flex items-center gap-1.5 text-foreground"
            >
              <KeyRound className="size-3 text-muted-foreground" />
              {mode === 'create' ? 'Token' : 'New token (leave blank to keep)'}
            </Label>
            <Input
              id="token"
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder={
                mode === 'create' ? 'ghp_… / gho_… / glpat-…' : '…'
              }
              required={mode === 'create'}
              className="font-mono"
            />
            <p className="text-[10.5px] text-muted-foreground/80">
              GitHub: PAT with <code>repo</code> scope. Gitea: token with{' '}
              <code>read:repository</code>. GitLab: PAT with <code>read_api</code>.
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
                Enabled
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
              Cancel
            </Button>
            <Button type="submit" variant="accent" size="sm" disabled={busy}>
              {busy && <Loader2 className="size-3.5 animate-spin" />}
              {mode === 'create' ? 'Add' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

