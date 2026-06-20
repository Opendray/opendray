import { api } from './api'

// ServerConfig mirrors internal/config/config.go's Config struct.
// Sensitive fields (database.url, admin.password) come back as ""
// from GET — backend strips them so the browser never holds the
// real secret. PUT preserves them when sent as "".
export interface ServerConfig {
  listen: string
  database: { url: string; max_conns: number }
  admin: {
    user: string
    password: string
    token_ttl: string
    mobile_token_ttl: string
  }
  log: {
    level: string
    format: string
    file: string
  }
  session: {
    idle_threshold: string
    idle_interval: string
  }
  vault: {
    root: string
    notes: string
    skills: string
    git_root: string
    personal_prefix: string
    projects_prefix: string
  }
  mcp: {
    root: string
    secrets_file: string
  }
  providers: {
    claude: {
      history_roots: string[] | null
      accounts_dir: string
      /** nil = default (enabled): fsnotify watcher auto-registers new
       * accounts under accounts_dir. */
      watcher_enabled: boolean | null
      /** nil = default (disabled): auto-switch a rate-limited live
       * session to another Claude account. Opt-in — changes billing
       * attribution without a click. */
      auto_failover_enabled: boolean | null
    }
    codex: {
      sessions_root: string
    }
    antigravity: {
      conversations_root: string
    }
  }
  memory: {
    backend: string
    store: string
    default_top_k: number
    similarity_threshold: number
    /** Write-time fold threshold. 0 = embedder-relative default;
     * negative disables folding entirely. */
    dedup_threshold: number
    gatekeeper: {
      enabled: boolean
      summarizer_id: string
      max_latency_ms: number
    }
    cleaner: {
      enabled: boolean
      interval_seconds: number
      initial_delay_seconds: number
      batch_size: number
      min_age_hours: number
      skip_if_decided_within_hours: number
      call_timeout_ms: number
      include_global_scope: boolean
      lifecycle_dormant_days: number
      grace_days: number
    }
    local: {
      model: string
      library_path: string
      model_path: string
      tokenizer_path: string
      max_seq_len: number
    }
    http: {
      base_url: string
      model: string
      api_key: string
      dimensions: number
    }
    scope: {
      default: string
    }
  }
  backup: {
    enabled: boolean
    local_dir: string
    export_dir: string
    pg_dump_path: string
    pg_restore_path: string
  }
  knowledge: {
    enabled: boolean
  }
}

export interface SettingsResponse {
  config: ServerConfig
  config_path: string
}

export interface TestPathResponse {
  path: string
  exists: boolean
  is_dir: boolean
  child_count?: number
  note?: string
}

export async function fetchServerSettings(): Promise<SettingsResponse> {
  return api<SettingsResponse>('/api/v1/admin/settings')
}

export async function updateServerSettings(cfg: ServerConfig): Promise<void> {
  await api('/api/v1/admin/settings', { method: 'PUT', body: cfg })
}

export async function testServerPath(path: string): Promise<TestPathResponse> {
  const q = new URLSearchParams({ path })
  return api<TestPathResponse>(`/api/v1/admin/settings/test-path?${q}`)
}

export async function restartServer(): Promise<void> {
  // Returns 202 + json body; server execs itself ~500ms later.
  await api('/api/v1/admin/restart', { method: 'POST' })
}

// emptyConfig is used as initial form state before the GET resolves.
export function emptyConfig(): ServerConfig {
  return {
    listen: '',
    database: { url: '', max_conns: 0 },
    admin: { user: '', password: '', token_ttl: '', mobile_token_ttl: '' },
    log: { level: '', format: '', file: '' },
    session: { idle_threshold: '', idle_interval: '' },
    vault: {
      root: '',
      notes: '',
      skills: '',
      git_root: '',
      personal_prefix: '',
      projects_prefix: '',
    },
    mcp: { root: '', secrets_file: '' },
    providers: {
      claude: {
        history_roots: [],
        accounts_dir: '',
        watcher_enabled: null,
        auto_failover_enabled: null,
      },
      codex: { sessions_root: '' },
      antigravity: { conversations_root: '' },
    },
    memory: {
      backend: '',
      store: '',
      default_top_k: 0,
      similarity_threshold: 0,
      dedup_threshold: 0,
      gatekeeper: { enabled: false, summarizer_id: '', max_latency_ms: 0 },
      cleaner: {
        enabled: false,
        interval_seconds: 0,
        initial_delay_seconds: 0,
        batch_size: 0,
        min_age_hours: 0,
        skip_if_decided_within_hours: 0,
        call_timeout_ms: 0,
        include_global_scope: false,
        lifecycle_dormant_days: 0,
        grace_days: 0,
      },
      local: {
        model: '',
        library_path: '',
        model_path: '',
        tokenizer_path: '',
        max_seq_len: 0,
      },
      http: { base_url: '', model: '', api_key: '', dimensions: 0 },
      scope: { default: '' },
    },
    backup: {
      enabled: false,
      local_dir: '',
      export_dir: '',
      pg_dump_path: '',
      pg_restore_path: '',
    },
    knowledge: { enabled: false },
  }
}
