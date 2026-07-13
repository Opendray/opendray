import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  Play,
  Loader2,
  FileCode,
  Hammer,
  Wrench,
  Settings2,
  UserPlus,
  Search,
  Box,
  Coffee,
  Terminal,
  Pencil,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import { useQueryClient } from '@tanstack/react-query'

import { listDir, readFile } from '@/lib/fs'
import {
  listCustomTasks,
  deleteCustomTask,
  type CustomTask,
} from '@/lib/customTasks'
import { createSession } from '@/lib/sessions'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import type { Session } from '@/lib/types'
import { Input } from '@/components/ui/input'
import { CustomTaskDialog } from '@/components/tasks/CustomTaskDialog'
import { useSessionTabs } from '@/stores/sessionTabs'

interface TaskRunnerPanelProps {
  session: Session
}

interface DiscoveredTask {
  name: string
  command: string
  /** Origin manifest, e.g. "package.json" / "Custom" */
  source: string
  description?: string
  /**
   * The backing custom task, present only for user-defined tasks (the
   * "Custom" group). Manifest/script tasks are read-only and leave this
   * undefined — that's what gates the inline edit/delete affordances.
   */
  custom?: CustomTask
}

interface TaskGroup {
  source: string
  icon: typeof FileCode
  tasks: DiscoveredTask[]
}

// PackageRunner picks the JS package manager from the lockfile present
// in cwd. Falls back to npm when none is found.
type PackageRunner = 'pnpm' | 'yarn' | 'bun' | 'npm'

function pickPackageRunner(entries: string[]): PackageRunner {
  if (entries.includes('pnpm-lock.yaml')) return 'pnpm'
  if (entries.includes('bun.lockb') || entries.includes('bun.lock')) return 'bun'
  if (entries.includes('yarn.lock')) return 'yarn'
  return 'npm'
}

// Single fs/list of cwd → drives every subsequent decision (which
// manifests to read, which lockfile, which language toolchains).
function useCwdEntries(cwd: string) {
  return useQuery({
    queryKey: ['fs', cwd],
    queryFn: () => listDir(cwd),
    staleTime: 10_000,
  })
}

function useManifests(cwd: string, present: Set<string>) {
  return useQuery({
    queryKey: ['task-manifests', cwd, [...present].sort().join(',')],
    queryFn: async () => {
      const want = (name: string) =>
        present.has(name) ? readFile(`${cwd}/${name}`) : Promise.resolve(null)
      const [pkg, make, taskfile, justfile, pyproject] = await Promise.all([
        want('package.json'),
        want('Makefile'),
        want('Taskfile.yml').catch(() => null),
        want('justfile').catch(() => null),
        want('pyproject.toml').catch(() => null),
      ])
      return { pkg, make, taskfile, justfile, pyproject }
    },
    enabled: present.size > 0,
    staleTime: 30_000,
  })
}

function parsePackageJson(
  text: string | null,
  runner: PackageRunner,
): DiscoveredTask[] {
  if (!text) return []
  try {
    const obj = JSON.parse(text) as { scripts?: Record<string, string> }
    if (!obj.scripts) return []
    return Object.entries(obj.scripts).map(([name, body]) => ({
      name,
      command: `${runner} run ${name}`,
      source: `package.json (${runner})`,
      description: body,
    }))
  } catch {
    return []
  }
}

function parseMakefile(text: string | null): DiscoveredTask[] {
  if (!text) return []
  const out: DiscoveredTask[] = []
  const seen = new Set<string>()
  for (const raw of text.split('\n')) {
    if (raw.startsWith('\t') || raw.startsWith(' ')) continue
    const m = raw.match(/^([A-Za-z0-9_.-]+)\s*:(?!=)/)
    if (!m) continue
    const name = m[1]
    if (name.startsWith('.') || name.includes('%') || seen.has(name)) continue
    seen.add(name)
    out.push({ name, command: `make ${name}`, source: 'Makefile' })
  }
  return out
}

// parseTaskfile picks up `desc:` for each task to surface as the
// task's description. Indent-tolerant scanner — no YAML dep.
function parseTaskfile(text: string | null): DiscoveredTask[] {
  if (!text) return []
  const lines = text.split('\n')
  let inTasks = false
  let baseIndent = -1
  const out: DiscoveredTask[] = []
  let current: DiscoveredTask | null = null
  for (const raw of lines) {
    if (!inTasks) {
      if (/^tasks:\s*$/.test(raw)) inTasks = true
      continue
    }
    if (raw.trim() === '' || raw.trimStart().startsWith('#')) continue
    const indent = raw.match(/^( *)/)?.[1].length ?? 0
    if (baseIndent === -1) baseIndent = indent
    if (indent < baseIndent) {
      inTasks = false
      if (current) out.push(current)
      current = null
      continue
    }
    if (indent === baseIndent) {
      const m = raw.match(/^ *([A-Za-z0-9_:.-]+):\s*$/)
      if (m) {
        if (current) out.push(current)
        current = {
          name: m[1],
          command: `task ${m[1]}`,
          source: 'Taskfile.yml',
        }
      }
      continue
    }
    // Deeper indent → properties of the current task.
    const desc = raw.match(/^ *desc:\s*['"]?(.+?)['"]?\s*$/)
    if (desc && current) current.description = desc[1]
  }
  if (current) out.push(current)
  return out
}

function parseJustfile(text: string | null): DiscoveredTask[] {
  if (!text) return []
  const out: DiscoveredTask[] = []
  for (const raw of text.split('\n')) {
    if (raw.startsWith(' ') || raw.startsWith('\t') || raw.startsWith('#')) {
      continue
    }
    const m = raw.match(/^([A-Za-z0-9_-]+)(?:\s+[^:=]*)?:\s*(?:[#].*)?$/)
    if (!m) continue
    if (raw.includes(':=')) continue
    out.push({ name: m[1], command: `just ${m[1]}`, source: 'justfile' })
  }
  return out
}

// Cargo / Go: no script discovery — just surface the canonical set so
// users can run them with one click. Same affordance most IDEs offer.
function cargoTasks(): DiscoveredTask[] {
  return [
    { name: 'check', command: 'cargo check', source: 'Cargo.toml' },
    { name: 'build', command: 'cargo build', source: 'Cargo.toml' },
    { name: 'test', command: 'cargo test', source: 'Cargo.toml' },
    { name: 'run', command: 'cargo run', source: 'Cargo.toml' },
    { name: 'fmt', command: 'cargo fmt', source: 'Cargo.toml' },
    { name: 'clippy', command: 'cargo clippy', source: 'Cargo.toml' },
  ]
}

function goTasks(): DiscoveredTask[] {
  return [
    { name: 'build', command: 'go build ./...', source: 'go.mod' },
    { name: 'test', command: 'go test ./...', source: 'go.mod' },
    { name: 'test:race', command: 'go test -race ./...', source: 'go.mod' },
    { name: 'vet', command: 'go vet ./...', source: 'go.mod' },
    { name: 'tidy', command: 'go mod tidy', source: 'go.mod' },
  ]
}

// pyproject parser is intentionally tiny — pulls poetry/PEP-621
// scripts plus the canonical pytest/ruff/mypy commands when the
// project file exists.
function pyprojectTasks(text: string | null): DiscoveredTask[] {
  const out: DiscoveredTask[] = [
    { name: 'pytest', command: 'pytest', source: 'pyproject.toml' },
    { name: 'ruff', command: 'ruff check', source: 'pyproject.toml' },
    { name: 'mypy', command: 'mypy .', source: 'pyproject.toml' },
  ]
  if (!text) return out
  // [tool.poetry.scripts] / [project.scripts] sections — extract the
  // script names so users can poetry run them.
  const sections = ['tool.poetry.scripts', 'project.scripts']
  let inSection = false
  for (const line of text.split('\n')) {
    const trimmed = line.trim()
    if (trimmed.startsWith('[')) {
      inSection = sections.some((s) => trimmed === `[${s}]`)
      continue
    }
    if (!inSection) continue
    const m = trimmed.match(/^([A-Za-z0-9_-]+)\s*=/)
    if (!m) continue
    out.push({
      name: m[1],
      command: `poetry run ${m[1]}`,
      source: 'pyproject.toml',
    })
  }
  return out
}

const sourceIcon: Record<string, typeof FileCode> = {
  'package.json': FileCode,
  Makefile: Hammer,
  'Taskfile.yml': Wrench,
  justfile: Wrench,
  'Cargo.toml': Box,
  'go.mod': Box,
  'pyproject.toml': Coffee,
  Custom: UserPlus,
  Scripts: Terminal,
}

// Shell-script extensions surfaced as runnable Tasks. .py / .rb etc.
// are intentionally left out — they need a runtime prefix and we'd
// have to guess; users can add those as Custom tasks.
const SHELL_EXTS = ['.sh', '.bash', '.zsh']

function isShellScript(name: string): boolean {
  const lower = name.toLowerCase()
  return SHELL_EXTS.some((ext) => lower.endsWith(ext))
}

// Common locations to scan for shell scripts beyond the cwd root.
// Capped per directory so a `scripts/` with hundreds of files
// doesn't drown the panel.
const SCRIPT_DIRS = ['scripts', 'bin', 'tools']
const MAX_SCRIPTS_PER_DIR = 25

function iconFor(source: string): typeof FileCode {
  // package.json (pnpm) -> match prefix
  for (const k of Object.keys(sourceIcon)) {
    if (source.startsWith(k)) return sourceIcon[k]
  }
  return Settings2
}

export function TaskRunnerPanel({ session }: TaskRunnerPanelProps) {
  const [filter, setFilter] = useState('')
  // Create a custom task inline, pre-scoped to this session's project,
  // instead of bouncing to the Plugins page to type the cwd by hand.
  const [creatingTask, setCreatingTask] = useState(false)
  // Edit a custom task inline (same reason — no trip to the Plugins page
  // just to change a command). null = dialog closed.
  const [editingTask, setEditingTask] = useState<CustomTask | null>(null)

  const dirEntries = useCwdEntries(session.cwd)
  const presentSet = useMemo(() => {
    const names = new Set<string>()
    for (const e of dirEntries.data?.entries ?? []) {
      if (!e.is_dir) names.add(e.name)
    }
    return names
  }, [dirEntries.data])

  const runner = useMemo<PackageRunner>(
    () => pickPackageRunner([...presentSet]),
    [presentSet],
  )

  const manifests = useManifests(session.cwd, presentSet)
  const customQ = useQuery({
    queryKey: ['custom-tasks', session.cwd],
    queryFn: () => listCustomTasks({ cwd: session.cwd }),
    staleTime: 30_000,
  })

  // Folder names from the cwd root that we know hold scripts. Listed
  // separately because we only fetch their contents if they exist —
  // saves a wasteful request on projects without a `scripts/` dir.
  const scriptDirsPresent = useMemo(() => {
    const names = new Set<string>()
    for (const e of dirEntries.data?.entries ?? []) {
      if (e.is_dir && SCRIPT_DIRS.includes(e.name)) names.add(e.name)
    }
    return [...names]
  }, [dirEntries.data])

  // One fetch per existing script dir. react-query dedupes on key so
  // re-renders don't re-fire.
  const scriptsQ = useQuery({
    queryKey: ['script-dirs', session.cwd, scriptDirsPresent.join(',')],
    queryFn: async () => {
      const out: Record<string, string[]> = {}
      await Promise.all(
        scriptDirsPresent.map(async (d) => {
          try {
            const list = await listDir(`${session.cwd}/${d}`)
            out[d] = list.entries
              .filter((e) => !e.is_dir && isShellScript(e.name))
              .map((e) => e.name)
              .slice(0, MAX_SCRIPTS_PER_DIR)
          } catch {
            out[d] = []
          }
        }),
      )
      return out
    },
    enabled: scriptDirsPresent.length > 0,
    staleTime: 30_000,
  })

  const groups = useMemo<TaskGroup[]>(() => {
    const out: TaskGroup[] = []
    if (!manifests.data) return out

    const pkg = parsePackageJson(manifests.data.pkg, runner)
    if (pkg.length)
      out.push({
        source: `package.json · ${runner}`,
        icon: sourceIcon['package.json'],
        tasks: pkg,
      })

    const mk = parseMakefile(manifests.data.make)
    if (mk.length)
      out.push({ source: 'Makefile', icon: sourceIcon['Makefile'], tasks: mk })

    const tf = parseTaskfile(manifests.data.taskfile)
    if (tf.length)
      out.push({
        source: 'Taskfile.yml',
        icon: sourceIcon['Taskfile.yml'],
        tasks: tf,
      })

    const just = parseJustfile(manifests.data.justfile)
    if (just.length)
      out.push({
        source: 'justfile',
        icon: sourceIcon['justfile'],
        tasks: just,
      })

    if (presentSet.has('Cargo.toml')) {
      out.push({
        source: 'Cargo.toml',
        icon: sourceIcon['Cargo.toml'],
        tasks: cargoTasks(),
      })
    }
    if (presentSet.has('go.mod')) {
      out.push({
        source: 'go.mod',
        icon: sourceIcon['go.mod'],
        tasks: goTasks(),
      })
    }
    if (presentSet.has('pyproject.toml')) {
      out.push({
        source: 'pyproject.toml',
        icon: sourceIcon['pyproject.toml'],
        tasks: pyprojectTasks(manifests.data.pyproject),
      })
    }

    // Shell scripts at the cwd root.
    const rootScripts = [...presentSet]
      .filter(isShellScript)
      .sort()
      .map((name) => ({
        name,
        command: `./${name}`,
        source: 'Scripts · ./',
      }))
    if (rootScripts.length > 0) {
      out.push({
        source: 'Scripts · ./',
        icon: sourceIcon['Scripts'],
        tasks: rootScripts,
      })
    }

    // Shell scripts under common script directories (scripts/, bin/,
    // tools/) — one group per directory so the user can see scope.
    if (scriptsQ.data) {
      for (const d of scriptDirsPresent) {
        const files = scriptsQ.data[d] ?? []
        if (files.length === 0) continue
        out.push({
          source: `Scripts · ${d}/`,
          icon: sourceIcon['Scripts'],
          tasks: files.map((name) => ({
            name,
            command: `./${d}/${name}`,
            source: `Scripts · ${d}/`,
          })),
        })
      }
    }
    return out
  }, [manifests.data, runner, presentSet, scriptsQ.data, scriptDirsPresent])

  const customGroups = useMemo<TaskGroup[]>(() => {
    const tasks = customQ.data ?? []
    if (tasks.length === 0) return []
    const mapped: DiscoveredTask[] = tasks.map((t) => ({
      name: t.name,
      command: t.command,
      source: t.cwd ? 'Custom · scoped' : 'Custom · global',
      description: t.description || undefined,
      custom: t,
    }))
    return [{ source: 'Custom', icon: sourceIcon['Custom'], tasks: mapped }]
  }, [customQ.data])

  const allGroups = useMemo(
    () => [...customGroups, ...groups],
    [customGroups, groups],
  )

  const visible = useMemo(() => {
    const q = filter.trim().toLowerCase()
    if (!q) return allGroups
    const scored: TaskGroup[] = []
    for (const g of allGroups) {
      const matches = g.tasks.filter(
        (t) =>
          t.name.toLowerCase().includes(q) ||
          t.command.toLowerCase().includes(q) ||
          (t.description ?? '').toLowerCase().includes(q),
      )
      if (matches.length > 0) scored.push({ ...g, tasks: matches })
    }
    return scored
  }, [allGroups, filter])

  const taskCount = useMemo(
    () => allGroups.reduce((n, g) => n + g.tasks.length, 0),
    [allGroups],
  )

  const qc = useQueryClient()
  const openTab = useSessionTabs((s) => s.open)

  // Tasks always run in a *new* shell session so the command is
  // executed by an actual shell (zsh/bash) and not typed into the
  // current session's prompt — which would just stuff text into
  // claude/codex/antigravity's input box. The new session inherits the
  // current cwd; the user is auto-switched to its tab.
  const runner_ = useMutation({
    mutationFn: async (task: DiscoveredTask) => {
      const newSess = await createSession({
        provider_id: 'shell',
        cwd: session.cwd,
        name: `task: ${task.name}`,
        // Nest under the originating session so the sidebar groups
        // its task runs under the project they belong to.
        parent_session_id: session.id,
      })
      // Small delay isn't needed — the manager has already started
      // the PTY before returning. Send the command as if the user
      // typed it; the trailing \n commits the line.
      await api(`/api/v1/sessions/${newSess.id}/input`, {
        method: 'POST',
        body: { data: task.command + '\n' },
      })
      return { newSess, task }
    },
    onSuccess: ({ newSess, task }) => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
      openTab({ id: newSess.id, name: newSess.name || `task: ${task.name}` })
      toast.success('Task running', {
        description: `${task.command} · new shell session`,
      })
    },
    onError: (err: Error) =>
      toast.error('Run failed', { description: err.message }),
  })

  const remove = useMutation({
    mutationFn: deleteCustomTask,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['custom-tasks', session.cwd] })
      qc.invalidateQueries({ queryKey: ['custom-tasks-all'] })
      toast.success('Task removed')
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  const isLoading = dirEntries.isLoading || (presentSet.size > 0 && manifests.isLoading)
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-3">
        <Loader2 className="size-3 animate-spin" />
        Scanning for tasks…
      </div>
    )
  }

  if (allGroups.length === 0) {
    return (
      <div className="flex flex-col gap-3">
        <div className="flex flex-col items-center text-center gap-2 py-6 text-muted-foreground">
          <Play className="size-5 opacity-40" strokeWidth={1.5} />
          <div className="text-[12px]">No tasks discovered.</div>
          <div className="text-[10px] opacity-70 max-w-[230px]">
            Drop a <code>package.json</code>, <code>Makefile</code>,{' '}
            <code>Cargo.toml</code>, <code>go.mod</code>, or{' '}
            <code>pyproject.toml</code> in the cwd, or define one below.
          </div>
        </div>
        <button
          type="button"
          onClick={() => setCreatingTask(true)}
          className="text-[11px] text-state-running hover:underline self-center"
        >
          Add a custom task →
        </button>
        <CustomTaskDialog
          open={creatingTask}
          onOpenChange={setCreatingTask}
          mode="create"
          initialCwd={session.cwd}
        />
      </div>
    )
  }

  // We spawn a new shell session for every task, so the current
  // session's lifecycle state is irrelevant. The only gate is the
  // in-flight mutation.
  const disabled = runner_.isPending

  return (
    <div className="flex flex-col gap-3">
      <div className="text-[10.5px] text-muted-foreground/70 px-1">
        Tasks run in a new shell session nested under this one in the
        sidebar — cwd is inherited.
      </div>

      <div className="relative">
        <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground/60" />
        <Input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder={`Filter ${taskCount} tasks…`}
          className="pl-7 h-7 text-[12px]"
        />
      </div>

      {visible.length === 0 && (
        <div className="text-[11px] text-muted-foreground/60 px-1 py-2">
          No tasks match "{filter}".
        </div>
      )}

      {visible.map((g) => {
        const Icon = iconFor(g.source)
        return (
          <section key={g.source} className="flex flex-col gap-1">
            <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium px-1">
              <Icon className="size-3" />
              {g.source}
              <span className="text-muted-foreground/50 normal-case tracking-normal">
                · {g.tasks.length}
              </span>
            </div>
            <div className="flex flex-col">
              {g.tasks.map((t) => (
                <div
                  key={`${g.source}/${t.name}`}
                  className={cn(
                    'group flex items-stretch rounded-md',
                    'hover:bg-card border border-transparent hover:border-border/60',
                  )}
                >
                  <button
                    type="button"
                    disabled={disabled}
                    onClick={() => runner_.mutate(t)}
                    className="flex items-start gap-2 px-2 py-1.5 text-left flex-1 min-w-0 disabled:opacity-50"
                    title={t.description || t.command}
                  >
                    <Play className="size-3 mt-0.5 text-muted-foreground/60 shrink-0 group-hover:text-foreground" />
                    <div className="flex flex-col min-w-0 flex-1">
                      <span className="text-[12px] font-medium truncate">
                        {t.name}
                      </span>
                      <span className="text-[10px] text-muted-foreground/70 font-mono truncate">
                        $ {t.command}
                      </span>
                      {t.description && (
                        <span className="text-[10px] text-muted-foreground/60 truncate italic">
                          {t.description}
                        </span>
                      )}
                    </div>
                  </button>
                  {/* Custom tasks get inline edit/delete; discovered
                      manifest/script tasks are read-only. */}
                  {t.custom && (
                    <div className="flex items-center gap-0.5 pr-1 shrink-0 opacity-0 group-hover:opacity-100 focus-within:opacity-100">
                      <button
                        type="button"
                        onClick={() => setEditingTask(t.custom!)}
                        className="p-1 rounded text-muted-foreground/70 hover:text-foreground hover:bg-muted"
                        title="Edit task"
                      >
                        <Pencil className="size-3" />
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          const task = t.custom!
                          if (
                            confirm(`Delete custom task "${task.name}"?`)
                          )
                            remove.mutate(task.id)
                        }}
                        className="p-1 rounded text-muted-foreground/70 hover:text-destructive hover:bg-muted"
                        title="Delete task"
                      >
                        <Trash2 className="size-3" />
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </section>
        )
      })}

      <div className="pt-1">
        <button
          type="button"
          onClick={() => setCreatingTask(true)}
          className="text-[11px] text-muted-foreground/70 hover:text-foreground hover:underline"
        >
          + Add custom task
        </button>
      </div>

      <CustomTaskDialog
        open={creatingTask}
        onOpenChange={setCreatingTask}
        mode="create"
        initialCwd={session.cwd}
      />
      <CustomTaskDialog
        open={editingTask != null}
        onOpenChange={(v) => !v && setEditingTask(null)}
        mode="edit"
        task={editingTask ?? undefined}
      />
    </div>
  )
}
