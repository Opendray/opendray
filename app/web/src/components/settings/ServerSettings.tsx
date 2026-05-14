import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  Archive,
  Eye,
  EyeOff,
  Loader2,
  Plus,
  Power,
  RotateCcw,
  Save,
  Search,
  X,
} from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Dialog, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  emptyConfig,
  fetchServerSettings,
  restartServer,
  updateServerSettings,
  type ServerConfig,
} from '@/lib/settings'
import { cn } from '@/lib/utils'

import {
  fetchMemoryStatus,
  probeEmbeddingEndpoint,
  type ProbeResult,
} from '@/lib/memory'
import {
  type BackupStatusReport,
  type Schedule,
  type TargetSpec,
  deleteTarget,
  fetchBackupStatus,
  formatInterval,
  listSchedules,
  listTargets,
  testTarget,
} from '@/lib/backup'
import { TargetEditor, targetSummary } from '@/components/backup/TargetEditor'

import { LogViewer } from './LogViewer'
import { MemoryAmbientSection } from './MemoryAmbientSection'
import { PathInput } from './PathInput'

// SECTIONS describe every server-settings panel rendered to the right
// of the sidebar. ID is used as the URL hash + active key. Title/desc
// are looked up at render time via useServerSectionLabel().
export const SERVER_SECTIONS = [
  { id: 'general' },
  { id: 'logging' },
  { id: 'sessions' },
  { id: 'vault' },
  { id: 'mcp' },
  { id: 'memory' },
  { id: 'memory-ambient' },
  { id: 'backup' },
  { id: 'claude' },
  { id: 'codex' },
  { id: 'gemini' },
] as const

export type ServerSectionId = (typeof SERVER_SECTIONS)[number]['id']

// Map kebab-cased section id to the i18n key segment (camelCase).
function sectionI18nKey(id: ServerSectionId): string {
  return id === 'memory-ambient' ? 'memoryAmbient' : id
}

// useServerSectionLabel returns translated title + desc for a section id.
// eslint-disable-next-line react-refresh/only-export-components
export function useServerSectionLabel() {
  const { t } = useTranslation()
  return (id: ServerSectionId) => ({
    title: t(`web.serverSettings.sections.${sectionI18nKey(id)}.title`),
    desc: t(`web.serverSettings.sections.${sectionI18nKey(id)}.desc`),
  })
}

// Sections whose values affect bound resources (listen/log/admin)
// — flag them so we can show "Restart required" when changed.
const RESTART_REQUIRED_SECTIONS: Record<ServerSectionId, boolean> = {
  general: true,
  logging: true,
  sessions: true,
  vault: true,
  mcp: true,
  memory: true, // backend / store wiring is read once at app.New
  'memory-ambient': false, // CRUD via REST; no toml binding to restart
  backup: true, // pg_dump path + cipher are bound at NewService
  claude: false, // history paths are read on each request, no restart needed
  codex: false,
  gemini: false,
}

interface ServerSettingsProps {
  activeSection: ServerSectionId
  searchQuery: string
}

// ServerSettings renders ONE section at a time (the one named by
// `activeSection`). The sidebar in SettingsPage owns navigation.
// Saving/restarting are exposed to the parent through events (toast)
// so the sticky toolbar in the page can also trigger them.
export function ServerSettings({
  activeSection,
  searchQuery,
}: ServerSettingsProps) {
  const qc = useQueryClient()
  const { t } = useTranslation()
  const sectionLabel = useServerSectionLabel()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['server-settings'],
    queryFn: fetchServerSettings,
  })

  const [draft, setDraft] = useState<ServerConfig>(emptyConfig())
  const [showPassword, setShowPassword] = useState(false)

  useEffect(() => {
    if (data?.config) setDraft(data.config)
  }, [data])

  const dirty = useMemo(() => {
    if (!data?.config) return false
    return JSON.stringify(data.config) !== JSON.stringify(draft)
  }, [data, draft])

  const dangerousChanged = useMemo(() => {
    if (!data?.config) return false
    return (
      draft.listen !== data.config.listen ||
      draft.admin.user !== data.config.admin.user ||
      draft.admin.password.length > 0
    )
  }, [data, draft])

  const save = useMutation({
    mutationFn: (cfg: ServerConfig) => updateServerSettings(cfg),
    onSuccess: () => {
      toast.success(t('web.serverSettings.saveToastTitle'), {
        description: t('web.serverSettings.saveToastDesc'),
      })
      qc.invalidateQueries({ queryKey: ['server-settings'] })
    },
    onError: (err: Error) =>
      toast.error(t('web.serverSettings.saveErrorTitle'), { description: err.message }),
  })

  const onSave = () => {
    if (dangerousChanged) {
      const ok = window.confirm(t('web.serverSettings.dangerousConfirm'))
      if (!ok) return
    }
    save.mutate(draft)
  }

  const onResetSection = () => {
    if (!data?.config) return
    if (
      !window.confirm(
        t('web.serverSettings.resetConfirm', {
          section: sectionLabel(activeSection).title,
        }),
      )
    )
      return
    const fresh = JSON.parse(JSON.stringify(data.config)) as ServerConfig
    // Replace only the slice owned by this section.
    setDraft((cur) => mergeSection(cur, fresh, activeSection))
  }

  // Loading + error states
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-[12px] text-muted-foreground p-6">
        <Loader2 className="size-3 animate-spin" />
        {t('web.serverSettings.loading')}
      </div>
    )
  }
  if (isError) {
    return (
      <div className="flex items-center gap-2 text-[12px] text-destructive p-6">
        <AlertTriangle className="size-3" />
        {t('web.serverSettings.loadFailed', { message: (error as Error).message })}
      </div>
    )
  }
  if (!data) return null

  if (!data.config_path) {
    return (
      <div className="rounded-lg border border-yellow-700/40 bg-yellow-950/20 p-4 text-[12px] text-yellow-200/80">
        {t('web.serverSettings.noConfigFlag')}
      </div>
    )
  }

  const sectionMeta = sectionLabel(activeSection)
  const restartRequired = dirty && RESTART_REQUIRED_SECTIONS[activeSection]

  return (
    <div className="flex flex-col">
      {/* Section header + per-section actions */}
      <header className="flex items-start justify-between gap-3 mb-4">
        <div>
          <h2 className="text-[15px] font-semibold tracking-tight">
            {sectionMeta.title}
          </h2>
          <p className="text-[12px] text-muted-foreground mt-0.5">
            {sectionMeta.desc}
          </p>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 text-[11px] text-muted-foreground hover:text-foreground"
            onClick={onResetSection}
            title={t('web.serverSettings.resetButtonTitle')}
          >
            <RotateCcw className="size-3 mr-1" />
            {t('web.serverSettings.resetButton')}
          </Button>
        </div>
      </header>

      {/* Toolbar showing source path + dirty/restart badges */}
      <div className="flex items-center justify-between gap-3 mb-5 pb-3 border-b border-border">
        <p className="text-[10px] text-muted-foreground/70 font-mono truncate">
          {data.config_path}
        </p>
        <div className="flex items-center gap-2 shrink-0">
          {restartRequired && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/15 text-amber-300 border border-amber-500/30">
              {t('web.serverSettings.badgeRestartRequired')}
            </span>
          )}
          {dirty && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-300 border border-blue-500/30">
              {t('web.serverSettings.badgeUnsaved')}
            </span>
          )}
        </div>
      </div>

      {/* Active section */}
      <SectionForm
        active={activeSection}
        draft={draft}
        setDraft={setDraft}
        showPassword={showPassword}
        toggleShowPassword={() => setShowPassword((v) => !v)}
        searchQuery={searchQuery}
      />

      {/* Sticky bottom action bar */}
      <div className="sticky bottom-0 -mx-6 px-6 pt-4 mt-8 pb-4 bg-gradient-to-t from-background via-background to-transparent">
        <div className="flex items-center gap-2">
          <Button
            type="button"
            onClick={onSave}
            disabled={!dirty || save.isPending}
            className="h-9 px-4"
          >
            {save.isPending ? (
              <Loader2 className="size-3.5 mr-2 animate-spin" />
            ) : (
              <Save className="size-3.5 mr-2" />
            )}
            {t('web.serverSettings.saveButton')}
          </Button>
          <RestartButton dirty={dirty} />
          <div className="ml-auto text-[10px] text-muted-foreground">
            {dirty
              ? t('web.serverSettings.unsavedHint')
              : t('web.serverSettings.savedHint')}
          </div>
        </div>
      </div>
    </div>
  )
}

// SectionForm is the body of the right column — exactly one of the
// 9 sections, filtered by `searchQuery` (matching label or hint).
function SectionForm({
  active,
  draft,
  setDraft,
  showPassword,
  toggleShowPassword,
  searchQuery,
}: {
  active: ServerSectionId
  draft: ServerConfig
  setDraft: React.Dispatch<React.SetStateAction<ServerConfig>>
  showPassword: boolean
  toggleShowPassword: () => void
  searchQuery: string
}) {
  const c = draft
  // Filter rows by search; case-insensitive match against label or hint.
  const filter = searchQuery.trim().toLowerCase()
  const visible = (label: string, hint?: string) => {
    if (!filter) return true
    return (
      label.toLowerCase().includes(filter) ||
      (hint?.toLowerCase().includes(filter) ?? false)
    )
  }

  switch (active) {
    case 'general':
      return (
        <div className="flex flex-col gap-8">
          <FormGroup heading="Network">
            {visible('Listen address', 'IP and port the gateway binds to') && (
              <FieldRow
                label="Listen address"
                hint="The host:port the HTTP server binds to. Example: 0.0.0.0:8770."
                tomlKey="listen"
              >
                <Input
                  value={c.listen}
                  onChange={(e) =>
                    setDraft({ ...draft, listen: e.target.value })
                  }
                  placeholder="0.0.0.0:8770"
                  className="h-9 font-mono"
                />
              </FieldRow>
            )}
          </FormGroup>

          <FormGroup heading="Operator account">
            {visible('Username', 'Login name for the admin') && (
              <FieldRow
                label="Username"
                hint="Login name used in the sign-in form. Changing this forces a re-login on the next request."
                tomlKey="admin.user"
              >
                <Input
                  value={c.admin.user}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      admin: { ...c.admin, user: e.target.value },
                    })
                  }
                  className="h-9 font-mono"
                  autoComplete="username"
                />
              </FieldRow>
            )}
            {visible('Password', 'Login password') && (
              <FieldRow
                label="Password"
                hint="Leave blank to keep the current password. Sending a value overwrites it."
                tomlKey="admin.password"
              >
                <div className="flex gap-1.5">
                  <Input
                    type={showPassword ? 'text' : 'password'}
                    value={c.admin.password}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        admin: { ...c.admin, password: e.target.value },
                      })
                    }
                    placeholder="••••••••"
                    className="h-9 font-mono flex-1"
                    autoComplete="new-password"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-9 w-9 p-0"
                    onClick={toggleShowPassword}
                    title={showPassword ? 'Hide' : 'Reveal'}
                  >
                    {showPassword ? (
                      <EyeOff className="size-3.5" />
                    ) : (
                      <Eye className="size-3.5" />
                    )}
                  </Button>
                </div>
              </FieldRow>
            )}
            {visible('Token TTL', 'Bearer token lifetime') && (
              <FieldRow
                label="Token TTL"
                hint='Bearer-token lifetime as a Go duration, e.g. "24h", "30m". Empty = never expire.'
                tomlKey="admin.token_ttl"
              >
                <Input
                  value={c.admin.token_ttl}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      admin: { ...c.admin, token_ttl: e.target.value },
                    })
                  }
                  placeholder="24h"
                  className="h-9 font-mono w-32"
                />
              </FieldRow>
            )}
          </FormGroup>
        </div>
      )

    case 'logging':
      return (
        <div className="flex flex-col gap-8">
          <FormGrid>
            {visible('Log level', 'Verbosity threshold') && (
              <FieldRow
                label="Log level"
                hint="Lines below this level are dropped."
                tomlKey="log.level"
              >
                <SegmentedSelect
                  value={c.log.level || 'info'}
                  options={[
                    { value: 'debug', label: 'debug' },
                    { value: 'info', label: 'info' },
                    { value: 'warn', label: 'warn' },
                    { value: 'error', label: 'error' },
                  ]}
                  onChange={(v) =>
                    setDraft({ ...draft, log: { ...c.log, level: v } })
                  }
                />
              </FieldRow>
            )}
            {visible('Format', 'Log encoding') && (
              <FieldRow
                label="Format"
                hint='"text" is human-readable; "json" is machine-parsable.'
                tomlKey="log.format"
              >
                <SegmentedSelect
                  value={c.log.format || 'text'}
                  options={[
                    { value: 'text', label: 'text' },
                    { value: 'json', label: 'json' },
                  ]}
                  onChange={(v) =>
                    setDraft({ ...draft, log: { ...c.log, format: v } })
                  }
                />
              </FieldRow>
            )}
            {visible('Log file', 'Persist log to disk') && (
              <FieldRow
                label="Log file"
                hint="Optional file path. Auto-rotates at 10 MB, keeps 5 backups. Empty = stderr only."
                tomlKey="log.file"
              >
                <PathInput
                  value={c.log.file}
                  onChange={(v) =>
                    setDraft({ ...draft, log: { ...c.log, file: v } })
                  }
                  placeholder="~/.opendray/opendray.log"
                />
              </FieldRow>
            )}
          </FormGrid>

          <div>
            <div className="flex items-baseline justify-between mb-2">
              <h3 className="text-[12.5px] font-medium">Live tail</h3>
              <p className="text-[10px] text-muted-foreground/70">
                In-memory ring buffer (last ~2,000 records). Resets on restart.
              </p>
            </div>
            <LogViewer />
          </div>
        </div>
      )

    case 'sessions':
      return (
        <FormGrid>
          {visible('Idle threshold', 'How long before idle fires') && (
            <FieldRow
              label="Idle threshold"
              hint="A session is silent for this long before session.idle fires. Empty = 30s."
              tomlKey="session.idle_threshold"
            >
              <Input
                value={c.session.idle_threshold}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    session: { ...c.session, idle_threshold: e.target.value },
                  })
                }
                placeholder="30s"
                className="h-9 font-mono w-32"
              />
            </FieldRow>
          )}
          {visible('Idle poll interval', 'Idle detector cadence') && (
            <FieldRow
              label="Idle poll interval"
              hint="How often the idle detector wakes up. Lower = lower latency, more wakeups. Empty = 5s."
              tomlKey="session.idle_interval"
            >
              <Input
                value={c.session.idle_interval}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    session: { ...c.session, idle_interval: e.target.value },
                  })
                }
                placeholder="5s"
                className="h-9 font-mono w-32"
              />
            </FieldRow>
          )}
        </FormGrid>
      )

    case 'vault':
      return (
        <FormGrid>
          {visible('Vault root', 'Top-level directory') && (
            <FieldRow
              label="Vault root"
              hint="Top-level directory for notes, skills, and MCP registry."
              tomlKey="vault.root"
            >
              <PathInput
                value={c.vault.root}
                onChange={(v) =>
                  setDraft({ ...draft, vault: { ...c.vault, root: v } })
                }
                placeholder="~/.opendray/vault"
                expectDir
              />
            </FieldRow>
          )}
          {visible('Notes directory', 'Where notes are written') && (
            <FieldRow
              label="Notes directory"
              hint="Override notes location. Defaults to <vault root>/notes."
              tomlKey="vault.notes"
            >
              <PathInput
                value={c.vault.notes}
                onChange={(v) =>
                  setDraft({ ...draft, vault: { ...c.vault, notes: v } })
                }
                placeholder="<vault>/notes"
                expectDir
              />
            </FieldRow>
          )}
          {visible('Skills directory', 'Where skill files live') && (
            <FieldRow
              label="Skills directory"
              hint="Override skills location. Defaults to <vault root>/skills."
              tomlKey="vault.skills"
            >
              <PathInput
                value={c.vault.skills}
                onChange={(v) =>
                  setDraft({ ...draft, vault: { ...c.vault, skills: v } })
                }
                placeholder="<vault>/skills"
                expectDir
              />
            </FieldRow>
          )}
          {visible('Git root', 'Repo for vault sync') && (
            <FieldRow
              label="Git root"
              hint="Working tree the Vault Sync feature commits to."
              tomlKey="vault.git_root"
            >
              <PathInput
                value={c.vault.git_root}
                onChange={(v) =>
                  setDraft({ ...draft, vault: { ...c.vault, git_root: v } })
                }
                placeholder="<vault root>"
                expectDir
              />
            </FieldRow>
          )}
          {visible('Personal prefix', 'Default folder for personal notes') && (
            <FieldRow
              label="Personal prefix"
              hint='Folder name used for personal notes when auto-deriving paths. Default "personal".'
              tomlKey="vault.personal_prefix"
            >
              <Input
                value={c.vault.personal_prefix}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    vault: { ...c.vault, personal_prefix: e.target.value },
                  })
                }
                placeholder="personal"
                className="h-9 font-mono w-48"
              />
            </FieldRow>
          )}
          {visible('Projects prefix', 'Default folder for project notes') && (
            <FieldRow
              label="Projects prefix"
              hint='Folder name used for project notes. Default "projects".'
              tomlKey="vault.projects_prefix"
            >
              <Input
                value={c.vault.projects_prefix}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    vault: { ...c.vault, projects_prefix: e.target.value },
                  })
                }
                placeholder="projects"
                className="h-9 font-mono w-48"
              />
            </FieldRow>
          )}
        </FormGrid>
      )

    case 'mcp':
      return (
        <FormGrid>
          {visible('Registry root', 'Where MCP server defs live') && (
            <FieldRow
              label="Registry root"
              hint="Directory holding MCP server JSON definitions. Defaults to <vault>/mcp."
              tomlKey="mcp.root"
            >
              <PathInput
                value={c.mcp.root}
                onChange={(v) =>
                  setDraft({ ...draft, mcp: { ...c.mcp, root: v } })
                }
                placeholder="<vault>/mcp"
                expectDir
              />
            </FieldRow>
          )}
          {visible('Secrets file', 'Substitution source for ${KEY}') && (
            <FieldRow
              label="Secrets file"
              hint="key=value file substituted into MCP server commands at spawn time."
              tomlKey="mcp.secrets_file"
            >
              <PathInput
                value={c.mcp.secrets_file}
                onChange={(v) =>
                  setDraft({
                    ...draft,
                    mcp: { ...c.mcp, secrets_file: v },
                  })
                }
                placeholder="~/.opendray/secrets.env"
              />
            </FieldRow>
          )}
        </FormGrid>
      )

    case 'memory':
      return (
        <div className="flex flex-col gap-8">
          <FormGroup heading="Configuration">
            {visible('Backend', 'Embedder choice') && (
              <FieldRow
                label="Embedder backend"
                hint='"auto" / "bm25" use the cgo-free pure-Go keyword path. "http" calls any OpenAI-compatible /v1/embeddings (ollama / OpenAI / LocalAI). "local" runs an ONNX sentence-transformer in-process — requires a binary built with `-tags local_onnx`.'
                tomlKey="memory.backend"
              >
                <SegmentedSelect
                  value={c.memory.backend || 'auto'}
                  options={[
                    { value: 'auto', label: 'auto' },
                    { value: 'bm25', label: 'bm25' },
                    { value: 'http', label: 'http' },
                    { value: 'local', label: 'local (onnx)' },
                  ]}
                  onChange={(v) =>
                    setDraft({ ...draft, memory: { ...c.memory, backend: v } })
                  }
                />
              </FieldRow>
            )}
            {visible('Store', 'Vector store backend') && (
              <FieldRow
                label="Store"
                hint={`"pgvector" reuses opendray's existing PG with the vector extension; only option in v1.`}
                tomlKey="memory.store"
              >
                <SegmentedSelect
                  value={c.memory.store || 'pgvector'}
                  options={[{ value: 'pgvector', label: 'pgvector' }]}
                  onChange={(v) =>
                    setDraft({ ...draft, memory: { ...c.memory, store: v } })
                  }
                />
              </FieldRow>
            )}
            {visible('Top K', 'Default search result count') && (
              <FieldRow
                label="Default top-K"
                hint="How many hits memory_search returns when the agent doesn't specify. Empty = 5."
                tomlKey="memory.default_top_k"
              >
                <Input
                  type="number"
                  value={c.memory.default_top_k || ''}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        default_top_k: parseInt(e.target.value || '0', 10),
                      },
                    })
                  }
                  placeholder="5"
                  className="h-9 font-mono w-24"
                />
              </FieldRow>
            )}
            {visible('Threshold', 'Cosine similarity floor') && (
              <FieldRow
                label="Similarity threshold"
                hint="Hits below this score are dropped. Empty = 0.1 (permissive — BM25 sparse vectors rarely break 0.5)."
                tomlKey="memory.similarity_threshold"
              >
                <Input
                  type="number"
                  step="0.01"
                  min="-1"
                  max="1"
                  value={c.memory.similarity_threshold || ''}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        similarity_threshold: parseFloat(e.target.value || '0'),
                      },
                    })
                  }
                  placeholder="0.1"
                  className="h-9 font-mono w-28"
                />
              </FieldRow>
            )}
            {visible('Default scope', 'Where new memories land') && (
              <FieldRow
                label="Default scope"
                hint={`What memory_store uses when the agent doesn't specify. "project" (recommended) groups by cwd; "session" isolates per session; "global" shares across cwds.`}
                tomlKey="memory.scope.default"
              >
                <SegmentedSelect
                  value={c.memory.scope.default || 'project'}
                  options={[
                    { value: 'project', label: 'project' },
                    { value: 'session', label: 'session' },
                    { value: 'global', label: 'global' },
                  ]}
                  onChange={(v) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        scope: { ...c.memory.scope, default: v },
                      },
                    })
                  }
                />
              </FieldRow>
            )}
          </FormGroup>

          <FormGroup heading="HTTP backend (used when backend=http)">
            <HttpBackendHelpers
              draft={c}
              onApply={(patch) =>
                setDraft({
                  ...draft,
                  memory: {
                    ...c.memory,
                    http: { ...c.memory.http, ...patch },
                  },
                })
              }
            />
            {visible('Base URL', 'OpenAI-compatible /v1/embeddings') && (
              <FieldRow
                label="Base URL"
                hint='e.g. "http://localhost:11434/v1" for ollama, "https://api.openai.com/v1" for OpenAI.'
                tomlKey="memory.http.base_url"
              >
                <Input
                  value={c.memory.http.base_url}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        http: { ...c.memory.http, base_url: e.target.value },
                      },
                    })
                  }
                  placeholder="http://localhost:11434/v1"
                  className="h-9 font-mono"
                />
              </FieldRow>
            )}
            {visible('Model', 'Embedding model name') && (
              <FieldRow
                label="Model"
                hint='e.g. "nomic-embed-text" for ollama, "text-embedding-3-small" for OpenAI.'
                tomlKey="memory.http.model"
              >
                <Input
                  value={c.memory.http.model}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        http: { ...c.memory.http, model: e.target.value },
                      },
                    })
                  }
                  placeholder="nomic-embed-text"
                  className="h-9 font-mono"
                />
              </FieldRow>
            )}
            {visible('API key', 'Bearer for the HTTP backend') && (
              <FieldRow
                label="API key"
                hint="Empty for ollama / local servers. Required for OpenAI / Voyage / hosted services."
                tomlKey="memory.http.api_key"
              >
                <Input
                  type="password"
                  value={c.memory.http.api_key}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        http: { ...c.memory.http, api_key: e.target.value },
                      },
                    })
                  }
                  placeholder="sk-…"
                  className="h-9 font-mono"
                />
              </FieldRow>
            )}
          </FormGroup>

          <FormGroup heading="Local ONNX (used when backend=local)">
            <div className="rounded-md border border-amber-500/30 bg-amber-500/5 p-3 text-[11px] text-amber-200/80 leading-snug">
              Requires the binary to be compiled with{' '}
              <code className="font-mono">-tags local_onnx</code>. The
              standard build returns a clear stub error when this backend is
              selected. See <strong>Memory → Local ONNX</strong> tutorial for
              setup steps.
            </div>
            {visible('Model name', 'Cosmetic label') && (
              <FieldRow
                label="Model name"
                hint='Cosmetic — appears in logs / Inspector. e.g. "bge-m3", "bge-small-en-v1.5".'
                tomlKey="memory.local.model"
              >
                <Input
                  value={c.memory.local.model}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        local: { ...c.memory.local, model: e.target.value },
                      },
                    })
                  }
                  placeholder="bge-m3"
                  className="h-9 font-mono w-48"
                />
              </FieldRow>
            )}
            {visible('Library path', 'Where libonnxruntime lives') && (
              <FieldRow
                label="Library path"
                hint="Directory holding libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux). After `brew install onnxruntime`, that's /opt/homebrew/opt/onnxruntime/lib."
                tomlKey="memory.local.library_path"
              >
                <PathInput
                  value={c.memory.local.library_path}
                  onChange={(v) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        local: { ...c.memory.local, library_path: v },
                      },
                    })
                  }
                  placeholder="/opt/homebrew/opt/onnxruntime/lib"
                  expectDir
                />
              </FieldRow>
            )}
            {visible('Model path', 'Path to model.onnx') && (
              <FieldRow
                label="Model path"
                hint="Absolute path to the .onnx weights. Download from HuggingFace, e.g. Xenova/bge-m3 or Xenova/bge-small-en-v1.5."
                tomlKey="memory.local.model_path"
              >
                <PathInput
                  value={c.memory.local.model_path}
                  onChange={(v) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        local: { ...c.memory.local, model_path: v },
                      },
                    })
                  }
                  placeholder="~/.opendray/models/bge-m3/model.onnx"
                />
              </FieldRow>
            )}
            {visible('Tokenizer path', 'Path to tokenizer.json') && (
              <FieldRow
                label="Tokenizer path"
                hint="Absolute path to tokenizer.json (HuggingFace standard format) — usually right next to the model."
                tomlKey="memory.local.tokenizer_path"
              >
                <PathInput
                  value={c.memory.local.tokenizer_path}
                  onChange={(v) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        local: { ...c.memory.local, tokenizer_path: v },
                      },
                    })
                  }
                  placeholder="~/.opendray/models/bge-m3/tokenizer.json"
                />
              </FieldRow>
            )}
            {visible('Max sequence length', 'Token cap per text') && (
              <FieldRow
                label="Max sequence length"
                hint="Tokens beyond this are truncated. bge-m3 default is 512. Empty = 512."
                tomlKey="memory.local.max_seq_len"
              >
                <Input
                  type="number"
                  min="32"
                  max="8192"
                  value={c.memory.local.max_seq_len || ''}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      memory: {
                        ...c.memory,
                        local: {
                          ...c.memory.local,
                          max_seq_len: parseInt(e.target.value || '0', 10),
                        },
                      },
                    })
                  }
                  placeholder="512"
                  className="h-9 font-mono w-24"
                />
              </FieldRow>
            )}
          </FormGroup>

          <div className="rounded-md border border-border bg-card/30 px-3 py-2.5 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-[12.5px] font-medium">Inspector</h3>
              <p className="text-[11px] text-muted-foreground/70 mt-0.5">
                Browse, search and edit stored memories on the dedicated page.
              </p>
            </div>
            <Button asChild variant="outline" size="sm" className="h-8 text-[11px]">
              <Link to="/memory">Open Memory →</Link>
            </Button>
          </div>
        </div>
      )

    case 'memory-ambient':
      return <MemoryAmbientSection />

    case 'backup':
      return <BackupSection draft={draft} setDraft={setDraft} visible={visible} />

    case 'claude':
      return (
        <FormGrid>
          {visible('History roots', 'Where Claude transcripts live') && (
            <FieldRow
              label="History roots"
              hint="Directories scanned for Claude per-project JSONL transcripts. Empty = scan ~/.claude/projects + every ~/.claude-accounts/*/projects."
              tomlKey="providers.claude.history_roots"
            >
              <StringList
                value={c.providers.claude.history_roots ?? []}
                onChange={(arr) =>
                  setDraft({
                    ...draft,
                    providers: {
                      ...c.providers,
                      claude: { ...c.providers.claude, history_roots: arr },
                    },
                  })
                }
                placeholder="~/.claude/projects"
              />
            </FieldRow>
          )}
          {visible('Accounts directory', 'Multi-account root') && (
            <FieldRow
              label="Accounts directory"
              hint="Root used for opendray-managed Claude account ConfigDirs. Default ~/.claude-accounts."
              tomlKey="providers.claude.accounts_dir"
            >
              <PathInput
                value={c.providers.claude.accounts_dir}
                onChange={(v) =>
                  setDraft({
                    ...draft,
                    providers: {
                      ...c.providers,
                      claude: { ...c.providers.claude, accounts_dir: v },
                    },
                  })
                }
                placeholder="~/.claude-accounts"
                expectDir
              />
            </FieldRow>
          )}
        </FormGrid>
      )

    case 'codex':
      return (
        <FormGrid>
          {visible('Sessions root', 'Codex rollouts directory') && (
            <FieldRow
              label="Sessions root"
              hint="Directory walked for Codex rollout JSONL files. Default ~/.codex/sessions."
              tomlKey="providers.codex.sessions_root"
            >
              <PathInput
                value={c.providers.codex.sessions_root}
                onChange={(v) =>
                  setDraft({
                    ...draft,
                    providers: {
                      ...c.providers,
                      codex: { sessions_root: v },
                    },
                  })
                }
                placeholder="~/.codex/sessions"
                expectDir
              />
            </FieldRow>
          )}
        </FormGrid>
      )

    case 'gemini':
      return (
        <FormGrid>
          {visible('tmp directory', 'Per-project Gemini logs') && (
            <FieldRow
              label="Tmp directory"
              hint="Root holding Gemini per-project tmp folders. Default ~/.gemini/tmp."
              tomlKey="providers.gemini.tmp_root"
            >
              <PathInput
                value={c.providers.gemini.tmp_root}
                onChange={(v) =>
                  setDraft({
                    ...draft,
                    providers: {
                      ...c.providers,
                      gemini: { ...c.providers.gemini, tmp_root: v },
                    },
                  })
                }
                placeholder="~/.gemini/tmp"
                expectDir
              />
            </FieldRow>
          )}
          {visible('projects.json', 'cwd → short-name map') && (
            <FieldRow
              label="projects.json"
              hint="Path to Gemini's cwd→short-name mapping file. Default ~/.gemini/projects.json."
              tomlKey="providers.gemini.projects_file"
            >
              <PathInput
                value={c.providers.gemini.projects_file}
                onChange={(v) =>
                  setDraft({
                    ...draft,
                    providers: {
                      ...c.providers,
                      gemini: { ...c.providers.gemini, projects_file: v },
                    },
                  })
                }
                placeholder="~/.gemini/projects.json"
              />
            </FieldRow>
          )}
        </FormGrid>
      )
  }
}

// FormGrid stacks rows with consistent spacing. Each row is a
// 220px-label + 1fr-control grid for crisp vertical alignment.
function FormGrid({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-5">{children}</div>
}

// FormGroup wraps a sub-section of a section with a small uppercase
// heading. Use when one section spans several semantic clusters
// (e.g. General has both Network and Operator account groups).
function FormGroup({
  heading,
  children,
}: {
  heading: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-3">
      <h3 className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        {heading}
      </h3>
      <FormGrid>{children}</FormGrid>
    </div>
  )
}

// FieldRow is the canonical row layout: label + description on the
// left, control on the right. The toml key is shown in tiny text
// under the label so power users can correlate to config.toml.
function FieldRow({
  label,
  hint,
  tomlKey,
  children,
}: {
  label: string
  hint?: string
  tomlKey?: string
  children: React.ReactNode
}) {
  return (
    <div className="grid grid-cols-[220px_1fr] gap-6 items-start">
      <div className="pt-1.5 min-w-0">
        <div className="text-[12.5px] font-medium text-foreground">
          {label}
        </div>
        {hint && (
          <p className="text-[11px] text-muted-foreground/80 leading-snug mt-1">
            {hint}
          </p>
        )}
        {tomlKey && (
          <p className="text-[10px] font-mono text-muted-foreground/50 mt-1.5">
            {tomlKey}
          </p>
        )}
      </div>
      <div className="min-w-0">{children}</div>
    </div>
  )
}

function SegmentedSelect({
  value,
  options,
  onChange,
}: {
  value: string
  options: { value: string; label: string }[]
  onChange: (v: string) => void
}) {
  return (
    <div className="inline-flex rounded-md border border-border overflow-hidden bg-background">
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          onClick={() => onChange(opt.value)}
          className={cn(
            'px-3 py-1.5 text-[11px] font-medium transition-colors',
            value === opt.value
              ? 'bg-card text-foreground'
              : 'text-muted-foreground hover:text-foreground hover:bg-card/40',
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

// StringList renders a list-of-strings field as one numbered row per
// entry plus an "Add" affordance at the bottom.
function StringList({
  value,
  onChange,
  placeholder,
}: {
  value: string[]
  onChange: (next: string[]) => void
  placeholder?: string
}) {
  const list = value ?? []
  return (
    <div className="flex flex-col gap-2">
      {list.length === 0 && (
        <p className="text-[11px] text-muted-foreground/70 italic">
          (none — using built-in defaults)
        </p>
      )}
      {list.map((v, i) => (
        <div key={i} className="flex gap-1.5 items-center">
          <span className="text-[10px] font-mono text-muted-foreground/50 w-5 text-right">
            {i + 1}
          </span>
          <Input
            value={v}
            onChange={(e) => {
              const next = [...list]
              next[i] = e.target.value
              onChange(next)
            }}
            placeholder={placeholder}
            className="h-9 font-mono flex-1"
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-9 w-9 p-0 text-muted-foreground hover:text-destructive"
            onClick={() => onChange(list.filter((_, j) => j !== i))}
            title="Remove"
          >
            <X className="size-3.5" />
          </Button>
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="h-8 self-start text-[11px]"
        onClick={() => onChange([...list, ''])}
      >
        <Plus className="size-3 mr-1" />
        Add path
      </Button>
    </div>
  )
}

// RestartButton triggers POST /admin/restart, then polls /health
// every 1s until it returns 200 again. While waiting, blocks the UI
// behind a fullscreen overlay so the operator doesn't try to use a
// dead server.
function RestartButton({ dirty }: { dirty: boolean }) {
  const { t } = useTranslation()
  const [waiting, setWaiting] = useState(false)
  const [tick, setTick] = useState(0)

  const restart = async () => {
    if (dirty) {
      const ok = window.confirm(t('web.serverSettings.restart.dirtyConfirm'))
      if (!ok) return
    }
    if (!window.confirm(t('web.serverSettings.restart.confirm'))) return

    setWaiting(true)
    setTick(0)
    try {
      await restartServer()
    } catch (err) {
      console.warn('restart request error (expected):', err)
    }

    const start = Date.now()
    const timer = setInterval(async () => {
      setTick((n) => n + 1)
      if (Date.now() - start > 30_000) {
        clearInterval(timer)
        setWaiting(false)
        toast.error(t('web.serverSettings.restart.timedOutTitle'), {
          description: t('web.serverSettings.restart.timedOutDesc'),
        })
        return
      }
      try {
        const res = await fetch('/api/v1/health', { cache: 'no-store' })
        if (res.ok) {
          clearInterval(timer)
          toast.success(t('web.serverSettings.restart.successToast'))
          setTimeout(() => window.location.reload(), 400)
        }
      } catch {
        // Still down; keep polling.
      }
    }, 1000)
  }

  return (
    <>
      <Button
        type="button"
        variant="outline"
        onClick={restart}
        disabled={waiting}
        className="h-9 px-4"
        title={t('web.serverSettings.restart.buttonTitle')}
      >
        <Power className="size-3.5 mr-2" />
        {t('web.serverSettings.restart.button')}
      </Button>
      {waiting && (
        <div className="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm flex flex-col items-center justify-center gap-3">
          <Loader2 className="size-8 animate-spin text-accent" />
          <p className="text-[14px] font-semibold">
            {t('web.serverSettings.restart.overlay')}
          </p>
          <p className="text-[12px] text-muted-foreground">
            {t('web.serverSettings.restart.waiting', { tick })}
          </p>
        </div>
      )}
    </>
  )
}

// mergeSection replaces ONE slice of `dst` with the same slice from
// `src`. Used by the Reset-section button so the operator can roll
// back changes scoped to just the panel they're looking at.
function mergeSection(
  dst: ServerConfig,
  src: ServerConfig,
  section: ServerSectionId,
): ServerConfig {
  const out = { ...dst }
  switch (section) {
    case 'general':
      out.listen = src.listen
      // password always reset to empty so the masked input doesn't
      // re-introduce the previous draft value.
      out.admin = { ...src.admin, password: '' }
      break
    case 'logging':
      out.log = src.log
      break
    case 'sessions':
      out.session = src.session
      break
    case 'vault':
      out.vault = src.vault
      break
    case 'mcp':
      out.mcp = src.mcp
      break
    case 'memory':
      out.memory = src.memory
      break
    case 'memory-ambient':
      // ambient memory has no toml-mergeable state — all CRUD goes
      // through dedicated REST endpoints (/memory-summarizer-providers etc.)
      break
    case 'backup':
      out.backup = src.backup
      break
    case 'claude':
      out.providers = {
        ...out.providers,
        claude: src.providers.claude,
      }
      break
    case 'codex':
      out.providers = {
        ...out.providers,
        codex: src.providers.codex,
      }
      break
    case 'gemini':
      out.providers = {
        ...out.providers,
        gemini: src.providers.gemini,
      }
      break
  }
  return out
}

// HttpBackendHelpers groups the "auto-detected services" banner +
// preset URL buttons + Test-connection button into one reusable
// row that lives at the top of the HTTP backend FormGroup. It
// reads /memory/status to pull whatever opendray noticed at
// startup and lets the operator one-click apply that endpoint.
function HttpBackendHelpers({
  draft,
  onApply,
}: {
  draft: ServerConfig
  onApply: (patch: { base_url?: string; model?: string; api_key?: string }) => void
}) {
  const { data: status } = useQuery({
    queryKey: ['memory-status-presets'],
    queryFn: fetchMemoryStatus,
    refetchInterval: 30_000,
  })
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<ProbeResult | null>(null)
  // Whatever base_url + api_key produced the current `result` —
  // when the operator edits either field we drop the result so
  // the green ✓ doesn't outlive the URL it was actually tested
  // against (caused real confusion in dogfooding).
  const [resultFor, setResultFor] = useState<{ url: string; key: string } | null>(null)
  if (
    result &&
    resultFor &&
    (resultFor.url !== draft.memory.http.base_url ||
      resultFor.key !== draft.memory.http.api_key)
  ) {
    setResult(null)
    setResultFor(null)
  }

  const presets = [
    { label: 'ollama', url: 'http://localhost:11434/v1', model: 'nomic-embed-text', tip: 'Local ollama daemon' },
    { label: 'LM Studio', url: 'http://localhost:1234/v1', model: '', tip: 'LM Studio local server' },
    { label: 'OpenAI', url: 'https://api.openai.com/v1', model: 'text-embedding-3-small', tip: 'OpenAI cloud (needs API key)' },
  ]

  const test = async () => {
    if (!draft.memory.http.base_url) return
    setBusy(true)
    try {
      const res = await probeEmbeddingEndpoint(
        draft.memory.http.base_url,
        draft.memory.http.api_key,
      )
      setResult(res)
      setResultFor({
        url: draft.memory.http.base_url,
        key: draft.memory.http.api_key,
      })
    } catch (err) {
      setResult({
        base_url: draft.memory.http.base_url,
        reachable: false,
        error: (err as Error).message,
      })
      setResultFor({
        url: draft.memory.http.base_url,
        key: draft.memory.http.api_key,
      })
    } finally {
      setBusy(false)
    }
  }

  const detected = (status?.auto_detected ?? []).filter(
    (h, i, all) => all.findIndex((x) => x.detected === h.detected) === i,
  )

  return (
    <div className="flex flex-col gap-2">
      {detected.length > 0 && (
        <div className="rounded-md border border-emerald-500/30 bg-emerald-500/5 p-2.5 text-[11px]">
          <p className="font-medium text-emerald-300/90 mb-1">
            Auto-detected at startup
          </p>
          <div className="flex flex-wrap gap-1.5">
            {detected.map((d) => (
              <button
                key={d.base_url}
                type="button"
                onClick={() => {
                  // Pick the first embedding-looking model when applying.
                  const embedModel =
                    d.models?.find((m) => /embed/i.test(m)) ?? d.models?.[0] ?? ''
                  onApply({ base_url: d.base_url, model: embedModel })
                }}
                className="px-2 py-0.5 rounded border border-emerald-500/40 bg-emerald-500/10 text-emerald-200 hover:bg-emerald-500/20 transition-colors font-mono text-[10.5px]"
                title={`${d.models?.length ?? 0} model(s) — click to use`}
              >
                {d.detected} · {d.base_url} ({d.models?.length ?? 0} models)
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-[10px] text-muted-foreground/70 mr-1">Presets:</span>
        {presets.map((p) => (
          <Button
            key={p.label}
            type="button"
            variant="outline"
            size="sm"
            className="h-7 text-[11px]"
            onClick={() => onApply({ base_url: p.url, model: p.model })}
            title={p.tip}
          >
            {p.label}
          </Button>
        ))}
        <Button
          type="button"
          size="sm"
          className="h-7 text-[11px] ml-auto"
          disabled={!draft.memory.http.base_url || busy}
          onClick={test}
        >
          {busy ? <Loader2 className="size-3 animate-spin" /> : 'Test connection'}
        </Button>
      </div>

      {result && (
        <ProbeResultLine
          res={result}
          configuredModel={draft.memory.http.model}
          onApplyModel={(m) => onApply({ model: m })}
        />
      )}
    </div>
  )
}

function ProbeResultLine({
  res,
  configuredModel,
  onApplyModel,
}: {
  res: ProbeResult
  configuredModel: string
  onApplyModel: (m: string) => void
}) {
  if (!res.reachable) {
    return (
      <p className="text-[10.5px] text-destructive bg-destructive/10 border border-destructive/30 rounded px-2 py-1">
        ✗ unreachable: {res.error ?? 'connection failed'}
      </p>
    )
  }
  const allModels = res.models ?? []
  // Heuristic: model id contains "embed" → likely an embedding
  // model (works for ollama's bge-m3 / nomic-embed-text / mxbai
  // and LM Studio's text-embedding-* prefix).
  const embedModels = allModels.filter((m) => /embed/i.test(m))
  const hasConfigured = configuredModel && allModels.includes(configuredModel)
  return (
    <div className="text-[10.5px] bg-emerald-500/10 border border-emerald-500/30 rounded px-2 py-1.5 flex flex-col gap-1">
      <p className="text-emerald-300">
        ✓ reachable {res.detected ? `(${res.detected})` : ''} ·{' '}
        {allModels.length} model(s) total · {embedModels.length} embedding
      </p>
      {configuredModel && !hasConfigured && (
        <p className="text-amber-300">
          ⚠ Configured model{' '}
          <code className="font-mono">{configuredModel}</code> isn't in the
          list. Pick one of the embedding models below or fix the name.
        </p>
      )}
      {embedModels.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-0.5">
          <span className="text-emerald-300/70">embedding models:</span>
          {embedModels.slice(0, 6).map((m) => {
            const active = m === configuredModel
            return (
              <button
                key={m}
                type="button"
                onClick={() => onApplyModel(m)}
                className={`px-1.5 py-0.5 rounded font-mono text-[10px] border transition-colors ${
                  active
                    ? 'border-emerald-400 bg-emerald-500/30 text-emerald-100'
                    : 'border-emerald-500/30 hover:bg-emerald-500/20 text-emerald-200'
                }`}
                title={active ? 'Currently configured' : 'Click to apply'}
              >
                {active ? '✓ ' : ''}
                {m}
              </button>
            )
          })}
          {embedModels.length > 6 && (
            <span className="text-emerald-300/60">
              +{embedModels.length - 6} more
            </span>
          )}
        </div>
      )}
      {embedModels.length === 0 && allModels.length > 0 && (
        <p className="text-amber-300">
          ⚠ No model name contains "embed". The endpoint might not have
          an embedding model loaded — check your local server.
        </p>
      )}
    </div>
  )
}

// BackupSection is the operator's one-stop control center for the
// backup feature: status, where backups go (targets), what's
// scheduled, plus advanced (path overrides). The act-on-a-backup
// pages (/backups, /export) handle the operational verbs.
function BackupSection({
  draft,
  setDraft,
  visible,
}: {
  draft: ServerConfig
  setDraft: React.Dispatch<React.SetStateAction<ServerConfig>>
  visible: (label: string, hint?: string) => boolean
}) {
  const c = draft
  const qc = useQueryClient()
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [newTargetOpen, setNewTargetOpen] = useState(false)

  const { data: status } = useQuery<BackupStatusReport | null>({
    queryKey: ['backup-status'],
    queryFn: fetchBackupStatus,
    retry: false,
  })

  const { data: targets } = useQuery<TargetSpec[]>({
    queryKey: ['backup-targets'],
    queryFn: listTargets,
    enabled: status !== null && status !== undefined,
  })

  const { data: schedules } = useQuery<Schedule[]>({
    queryKey: ['backup-schedules'],
    queryFn: listSchedules,
    enabled: status !== null && status !== undefined,
  })

  const featureOff = status === null
  const showFilter = !visible || true // search filter not applied to control-center widgets

  return (
    <div className="flex flex-col gap-8">
      {/* ── Status ─────────────────────────────────────────────── */}
      <FormGroup heading="Status">
        <div
          className={cn(
            'rounded-md border p-3 text-[12px] flex flex-col gap-2',
            status?.ok
              ? 'border-state-running/30 bg-state-running/5'
              : 'border-state-idle/30 bg-state-idle/5',
          )}
        >
          {featureOff && (
            <div className="flex items-start gap-2">
              <Archive className="size-3.5 mt-0.5 text-state-idle" />
              <div>
                <div className="font-medium">Feature disabled</div>
                <div className="text-muted-foreground mt-0.5">
                  Set{' '}
                  <code className="text-foreground">OPENDRAY_BACKUP_ENABLED=1</code>{' '}
                  +{' '}
                  <code className="text-foreground">
                    OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;
                  </code>{' '}
                  in opendray's environment, then restart. The master
                  passphrase is env-only — it never touches config.toml.
                </div>
              </div>
            </div>
          )}
          {status && (
            <>
              <Row label="Status">
                <span className={status.ok ? 'text-state-running' : 'text-state-failed'}>
                  {status.ok ? 'enabled · healthy' : 'enabled · degraded'}
                </span>
              </Row>
              <Row label="Key fingerprint">
                <code className="text-foreground">{status.key_fingerprint}</code>
                <span className="ml-2 text-[10.5px] text-muted-foreground">
                  record in Vaultwarden — losing it locks all prior backups
                </span>
              </Row>
              <Row label="pg_dump">
                {status.ok ? (
                  <code className="text-foreground">{status.pg_dump_version}</code>
                ) : (
                  <span className="text-state-failed">
                    {status.pg_dump_error || 'unavailable'}
                  </span>
                )}
              </Row>
              <Row label="pg_restore">
                <code className="text-foreground">
                  {status.pg_restore_version || '(not resolved)'}
                </code>
              </Row>
              <div className="pt-1 border-t border-border/50 flex items-center justify-between">
                <Link to="/backups" className="text-[11.5px] underline text-accent">
                  Open Backups page →
                </Link>
                <Link to="/export" className="text-[11.5px] underline text-accent">
                  Open Export / Import →
                </Link>
              </div>
            </>
          )}
        </div>
      </FormGroup>

      {/* ── Where backups go (targets) ─────────────────────────── */}
      {!featureOff && showFilter && (
        <FormGroup heading="Where backups go">
          <p className="text-[12px] text-muted-foreground mb-1">
            Each target is one place a backup blob can be written.
            opendray supports <strong>local disk</strong>,{' '}
            <strong>SMB/CIFS</strong> (Windows / NAS),{' '}
            <strong>S3-compatible</strong> (AWS, R2, B2, MinIO,
            Alibaba Cloud OSS, Tencent Cloud COS, ...),{' '}
            <strong>WebDAV</strong> (Nextcloud, Synology, Jianguoyun),{' '}
            <strong>SFTP</strong>, plus an <strong>rclone</strong>{' '}
            passthrough that taps into 70+ extra backends (Google
            Drive, OneDrive, Dropbox, Baidu Pan, Aliyun Drive, ...).
          </p>

          <div className="flex flex-col gap-1.5">
            {targets === undefined ? (
              <div className="text-muted-foreground text-[12px]">Loading…</div>
            ) : targets.length === 0 ? (
              <div className="rounded-md border border-dashed border-border p-4 text-center text-[12px] text-muted-foreground">
                No targets yet. Add one to start backing up.
              </div>
            ) : (
              targets.map((t) => (
                <TargetRow
                  key={t.id}
                  target={t}
                  onChanged={() => qc.invalidateQueries({ queryKey: ['backup-targets'] })}
                />
              ))
            )}
          </div>

          <Dialog open={newTargetOpen} onOpenChange={setNewTargetOpen}>
            <DialogTrigger asChild>
              <Button size="sm" className="mt-2 self-start">
                <Plus className="size-3.5 mr-1.5" />
                Add target
              </Button>
            </DialogTrigger>
            <TargetEditor
              onCreated={async () => {
                setNewTargetOpen(false)
                qc.invalidateQueries({ queryKey: ['backup-targets'] })
              }}
            />
          </Dialog>
        </FormGroup>
      )}

      {/* ── Schedules ──────────────────────────────────────────── */}
      {!featureOff && (
        <FormGroup heading="Schedules">
          {schedules === undefined ? (
            <div className="text-muted-foreground text-[12px]">Loading…</div>
          ) : schedules.length === 0 ? (
            <div className="text-[12px] text-muted-foreground">
              No recurring schedules. Add one on{' '}
              <Link to="/backups" className="underline text-accent">
                /backups → Schedules
              </Link>{' '}
              to take backups automatically.
            </div>
          ) : (
            <div className="rounded-md border border-border bg-card/30 overflow-hidden">
              <table className="w-full text-[12px]">
                <thead className="bg-card/50 text-muted-foreground">
                  <tr className="text-left">
                    <th className="px-3 py-2 font-medium">Schedule</th>
                    <th className="px-3 py-2 font-medium">Target</th>
                    <th className="px-3 py-2 font-medium">Cadence</th>
                    <th className="px-3 py-2 font-medium">Keep</th>
                    <th className="px-3 py-2 font-medium">State</th>
                  </tr>
                </thead>
                <tbody>
                  {schedules.map((s) => (
                    <tr key={s.id} className="border-t border-border/60">
                      <td className="px-3 py-2 font-mono text-[11px]">{s.id}</td>
                      <td className="px-3 py-2 font-mono text-[11px]">
                        {s.target_id}
                      </td>
                      <td className="px-3 py-2 text-muted-foreground">
                        every {formatInterval(s.interval_sec)}
                      </td>
                      <td className="px-3 py-2 text-muted-foreground">
                        {s.retention} backups
                      </td>
                      <td className="px-3 py-2">
                        {s.enabled ? (
                          <span className="text-state-running">enabled</span>
                        ) : (
                          <span className="text-muted-foreground">paused</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="px-3 py-2 border-t border-border/60 text-right">
                <Link to="/backups" className="text-[11px] underline text-accent">
                  Manage on /backups → Schedules →
                </Link>
              </div>
            </div>
          )}
        </FormGroup>
      )}

      {/* ── What's in a backup? ───────────────────────────────── */}
      {!featureOff && (
        <FormGroup heading="What's in a backup?">
          <div className="text-[12px] text-muted-foreground">
            Each backup is a <code>pg_dump --format=custom</code> of every
            opendray table (sessions, integrations, memories,
            audit_log, etc.) plus a <code>manifest.json</code> and
            (optionally) the live <code>config.toml</code>. Open the
            "What's in a backup?" panel on the{' '}
            <Link to="/backups" className="underline text-accent">
              Backups page
            </Link>{' '}
            to see the live inventory with row counts.
          </div>
        </FormGroup>
      )}

      {/* ── Advanced (paths, collapsible) ─────────────────────── */}
      <div className="rounded-md border border-border bg-card/20">
        <button
          type="button"
          onClick={() => setAdvancedOpen((v) => !v)}
          className="w-full flex items-center gap-2 px-4 py-2.5 hover:bg-card/40 text-left text-[12px] font-medium"
        >
          {advancedOpen ? '▾' : '▸'} Advanced (paths & client binaries) — restart required
        </button>
        {advancedOpen && (
          <div className="px-4 pb-4 pt-2 border-t border-border/50 flex flex-col gap-5">
            {visible('Local backup directory', 'Where local-target bundles are written') && (
              <FieldRow
                label="Local backup directory"
                hint="Default root for the auto-created `local` target. Empty = ~/.opendray/backups. Restart required."
                tomlKey="backup.local_dir"
              >
                <PathInput
                  value={c.backup.local_dir}
                  onChange={(v) =>
                    setDraft({ ...draft, backup: { ...c.backup, local_dir: v } })
                  }
                  placeholder="~/.opendray/backups"
                />
              </FieldRow>
            )}
            {visible('Export directory', 'Where /export bundles are staged') && (
              <FieldRow
                label="Export directory"
                hint="Where one-shot export zips are staged on disk. Empty = ~/.opendray/exports. Bundles auto-expire after 24h. Restart required."
                tomlKey="backup.export_dir"
              >
                <PathInput
                  value={c.backup.export_dir}
                  onChange={(v) =>
                    setDraft({ ...draft, backup: { ...c.backup, export_dir: v } })
                  }
                  placeholder="~/.opendray/exports"
                />
              </FieldRow>
            )}
            {visible('pg_dump path', 'Override the resolved pg_dump binary') && (
              <FieldRow
                label="pg_dump path"
                hint="Absolute path to pg_dump. Major version must be ≥ the server's. Empty = first pg_dump on PATH."
                tomlKey="backup.pg_dump_path"
              >
                <PathInput
                  value={c.backup.pg_dump_path}
                  onChange={(v) =>
                    setDraft({ ...draft, backup: { ...c.backup, pg_dump_path: v } })
                  }
                  placeholder="/opt/homebrew/opt/postgresql@17/bin/pg_dump"
                />
              </FieldRow>
            )}
            {visible('pg_restore path', 'Override the resolved pg_restore binary') && (
              <FieldRow
                label="pg_restore path"
                hint="Absolute path to pg_restore for the /backups/restore flow. Same major-version rule."
                tomlKey="backup.pg_restore_path"
              >
                <PathInput
                  value={c.backup.pg_restore_path}
                  onChange={(v) =>
                    setDraft({ ...draft, backup: { ...c.backup, pg_restore_path: v } })
                  }
                  placeholder="/opt/homebrew/opt/postgresql@17/bin/pg_restore"
                />
              </FieldRow>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function TargetRow({
  target,
  onChanged,
}: {
  target: TargetSpec
  onChanged: () => void
}) {
  const [testing, setTesting] = useState(false)

  async function onTest() {
    setTesting(true)
    try {
      const res = await testTarget(target.id)
      if (res.ok) toast.success(`${target.id}: connection OK`)
      else toast.error('Connection failed', { description: res.error })
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Test failed', { description: msg })
    } finally {
      setTesting(false)
    }
  }

  async function onDelete() {
    if (!window.confirm(`Delete target "${target.id}"? Schedules referencing it will block the delete.`)) return
    try {
      await deleteTarget(target.id)
      toast.success('Target deleted')
      onChanged()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error'
      toast.error('Delete failed', { description: msg })
    }
  }

  return (
    <div className="flex items-center gap-3 p-2.5 rounded-md border border-border bg-card/30">
      <span className="px-2 py-0.5 rounded border border-border text-[10px] uppercase tracking-wide bg-card font-mono">
        {target.kind}
      </span>
      <div className="flex-1 min-w-0">
        <div className="font-mono text-[11.5px] truncate">{target.id}</div>
        <div className="text-[11px] text-muted-foreground truncate">
          {targetSummary(target)}
        </div>
      </div>
      <span
        className={cn(
          'text-[10.5px] uppercase tracking-wide',
          target.enabled ? 'text-state-running' : 'text-muted-foreground',
        )}
      >
        {target.enabled ? 'on' : 'off'}
      </span>
      <Button
        onClick={onTest}
        variant="outline"
        size="sm"
        className="h-7 text-[11px]"
        disabled={testing}
      >
        {testing ? 'Testing…' : 'Test'}
      </Button>
      <Button
        onClick={onDelete}
        variant="outline"
        size="sm"
        className="h-7 px-2 text-[11px]"
      >
        Delete
      </Button>
    </div>
  )
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-baseline gap-3">
      <span className="w-32 text-muted-foreground">{label}</span>
      <span className="flex-1">{children}</span>
    </div>
  )
}

// SettingsSearch is exported so the parent SettingsPage can render
// it in the sticky header (above the section title) without
// duplicating layout state. State lives in the page; we only ship
// the reusable input.
export function SettingsSearchInput({
  value,
  onChange,
  placeholder,
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
}) {
  const { t } = useTranslation()
  return (
    <div className="relative">
      <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground/60" />
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder ?? t('web.serverSettings.searchPlaceholder')}
        className="h-8 pl-8 text-xs w-56"
      />
    </div>
  )
}
