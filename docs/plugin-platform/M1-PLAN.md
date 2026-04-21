# Implementation Plan: OpenDray Plugin Platform M1 — Foundations + Declarative

> Output file: `/home/linivek/workspace/opendray/docs/plugin-platform/M1-PLAN.md`
> Design contract: `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md` §M1
> North star: a stranger can build, publish-via-local-path, install, and ship a working plugin end-to-end. Demo polish is NOT the priority.

---

## 1. Scope boundary

**IN (M1 contract):**
- `plugin/install/` package: consent-token install flow for local-filesystem sources (`local:<abs-path>`), sha256 verify of extracted bundles, extraction into `${OPENDRAY_DATA_DIR}/plugins/.installed/<name>/<version>/`.
- Manifest v1 parser as a strict superset of the existing `plugin.Provider` struct. All 6 agent manifests + 11 panel manifests under `plugins/agents/*` and `plugins/panels/*` load unchanged through compat-mode (synthesised v1 manifest in memory).
- DB migrations for `plugin_consents`, `plugin_kv`, `plugin_secret`, `plugin_audit`. Only `plugin_consents` + `plugin_audit` have live write paths in M1; `plugin_kv` and `plugin_secret` are DDL-only scaffolds awaiting M2.
- Bridge gateway skeleton: HTTP-only `/api/plugins/*` endpoints, a capability-gate middleware that consults `plugin_consents` and writes to `plugin_audit` on every invocation. No WebSocket. No `plugin://` asset scheme.
- Four contribution points end-to-end: `contributes.commands`, `contributes.statusBar`, `contributes.keybindings`, `contributes.menus`.
- `run` action dispatcher for manifest-declared commands with action kinds limited to `notify`, `openUrl`, `exec` (capability-gated), and `runTask` (maps to the existing `gateway/tasks` runner). `kind: host`, `openView` → return `EUNAVAIL` in M1.
- SDK scaffold CLI: `opendray plugin scaffold --form declarative <name>` as a subcommand of the existing `cmd/opendray` binary.
- Flutter shell: new `features/workbench/` module rendering status-bar strip, command palette (`Cmd/Ctrl+Shift+P`), global keybinding dispatcher, menu slot in session/dashboard app bar. Fetches contributions from new `GET /api/workbench/contributions`.
- Local install CLI path: `opendray plugin install <abs-path-or-zip>` subcommand; `marketplace://` URL parser that validates scheme and returns `"not yet implemented (M4)"`.
- Reference plugin `plugins/examples/time-ninja/` exercising all four contribution points + one declared capability.

**OUT — enumerated DEFERRED so no scope creep:**

| Tempting creep | Deferred to |
|---|---|
| Webview runtime, `plugin://` asset handler, WebView preload, bridge WebSocket `/api/plugins/{name}/bridge/ws` | **M2** |
| `contributes.activityBar`, `contributes.views`, `contributes.panels` | **M2** |
| `opendray.workbench.*`, `opendray.storage.*`, `opendray.secret.*`, `opendray.events.*`, `opendray.ui.*` bridge methods | **M2** |
| Host sidecar supervisor, `plugin/host/supervisor.go`, JSON-RPC 2.0 stdio, LSP framing | **M3** |
| `opendray.fs.*`, `opendray.exec.*`, `opendray.http.*` full implementations (capability *types* locked in M1; runtime enforcement is skeletal for M1's `exec` action only) | **M3** |
| `contributes.languageServers`, `contributes.debuggers`, `contributes.taskRunners` native pluggable runner | **M3 / M7** |
| Marketplace client, `plugin/market/`, `index.json` resolver, sha256 of remote artifacts, ed25519 signature verify, revocation polling, `opendray plugin publish` | **M4** |
| Hot reload `opendray plugin dev`, bridge trace, `opendray-dev` portable host, localization pipeline | **M6** |
| Cross-plugin `commands.execute` with `exported: true` gate | **post-v1** |
| Per-plugin WebSocket rate limiting, runtime consent toggles in Settings UI | **M2+** |
| Flutter Android/iOS-specific keybinding layer beyond desktop + browser builds | **M2** |

---

## 2. Task graph

### T1 — Extend manifest struct (v1 superset)
- **Depends on:** none
- **Create:** —
- **Modify:** `plugin/manifest.go` — add new optional fields (`Publisher`, `Engines`, `Form`, `Activation`, `Contributes`, `Permissions`, `V2Reserved`) to `Provider`, plus new types `ContributesV1`, `PermissionsV1`, `CommandV1`, `StatusBarItemV1`, `KeybindingV1`, `MenuEntryV1`, `CommandRunV1`, `EnginesV1`, all with JSON tags preserving zero-values as omitempty. Do not break existing `Type`, `CLI`, `Capabilities`, `ConfigSchema`, `Icon`, `Version` fields.
- **Core types/functions:**
  ```go
  type ContributesV1 struct {
      Commands    []CommandV1       `json:"commands,omitempty"`
      StatusBar   []StatusBarItemV1 `json:"statusBar,omitempty"`
      Keybindings []KeybindingV1    `json:"keybindings,omitempty"`
      Menus       map[string][]MenuEntryV1 `json:"menus,omitempty"`
  }
  func (p Provider) EffectiveForm() string // returns p.Form or derives from p.Type (compat)
  func (p Provider) IsV1() bool            // true iff Publisher != "" && Engines.Opendray != ""
  ```
- **Acceptance:** `go build ./...` succeeds. `json.Unmarshal` on every existing `plugins/*/manifest.json` produces a struct with `IsV1() == false` and the legacy fields populated identically to today.
- **Tests:** `plugin/manifest_v1_test.go` — table-driven: `TestLoadManifest_LegacyCompat` loads all 17 bundled manifests and asserts each has `Name`, `Version`, `Type` unchanged and `Publisher` empty. `TestLoadManifest_V1Superset` loads a hand-crafted v1 manifest with `form`, `publisher`, `engines`, `contributes.commands`, `permissions.exec`.
- **Complexity:** S
- **Risk/Mitigation:** Low. Risk: accidentally changing JSON-tag casing on existing fields. Mitigation: keep additions purely additive; run `go test -race ./plugin/...` which includes the existing `manifest_test.go` compat coverage.

### T2 — Manifest v1 validator
- **Depends on:** T1
- **Create:** `plugin/manifest_validate.go`
- **Modify:** —
- **Core types/functions:**
  ```go
  type ValidationError struct{ Path, Msg string }
  func (v ValidationError) Error() string
  func ValidateV1(p Provider) []ValidationError       // returns nil on legacy/compat manifests (IsV1 == false)
  func validateName(name string) error                // regex ^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$
  func validateSemver(v string) error
  func validateCommandID(id string) error             // ^[a-z0-9._-]+$
  func validatePermissions(p PermissionsV1) []ValidationError
  ```
- **Acceptance:** `ValidateV1` returns empty slice for `plugins/examples/time-ninja/manifest.json`. Fails with a named path (`contributes.commands[0].id`) for any violation. Legacy manifests pass by short-circuit.
- **Tests:** `plugin/manifest_validate_test.go` — table-driven with 20+ invalid cases covering every regex in the JSON schema in `02-manifest.md` §JSON Schema.
- **Complexity:** M
- **Risk/Mitigation:** Medium — schema drift vs. 02-manifest.md. Mitigation: every regex is copy-pasted verbatim from the doc; add a comment citing doc line numbers. Surface any inconsistency between doc and implementation in the PR description, do not silently resolve.

### T3 — DB migrations: consents, kv, secret, audit
- **Depends on:** none (parallel with T1)
- **Create:** `kernel/store/migrations/010_plugin_consents.sql`, `011_plugin_kv.sql`, `012_plugin_secret.sql`, `013_plugin_audit.sql`, plus matching `*_down.sql` variants checked in alongside but not wired to `Migrate()` (we follow the existing forward-only pattern in `db.go`; downs exist as docs-as-SQL).
- **Modify:** `kernel/store/db.go` — append the four new file names to the `files` slice inside `Migrate`.
- **Schema (up):**
  ```sql
  -- 010_plugin_consents.sql
  CREATE TABLE IF NOT EXISTS plugin_consents (
      plugin_name   TEXT PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
      manifest_hash TEXT NOT NULL,      -- sha256 hex of canonicalised manifest
      perms_json    JSONB NOT NULL,     -- PermissionsV1 exactly as granted
      granted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
  );

  -- 011_plugin_kv.sql  (scaffold — no live writers in M1)
  CREATE TABLE IF NOT EXISTS plugin_kv (
      plugin_name TEXT NOT NULL REFERENCES plugins(name) ON DELETE CASCADE,
      key         TEXT NOT NULL,
      value       JSONB NOT NULL,
      size_bytes  INT  NOT NULL DEFAULT 0,
      updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
      PRIMARY KEY (plugin_name, key)
  );
  CREATE INDEX IF NOT EXISTS idx_plugin_kv_plugin ON plugin_kv(plugin_name);

  -- 012_plugin_secret.sql  (scaffold — no live writers in M1)
  CREATE TABLE IF NOT EXISTS plugin_secret (
      plugin_name TEXT NOT NULL REFERENCES plugins(name) ON DELETE CASCADE,
      key         TEXT NOT NULL,
      ciphertext  BYTEA NOT NULL,
      updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
      PRIMARY KEY (plugin_name, key)
  );

  -- 013_plugin_audit.sql
  CREATE TABLE IF NOT EXISTS plugin_audit (
      id           BIGSERIAL PRIMARY KEY,
      ts           TIMESTAMPTZ NOT NULL DEFAULT now(),
      plugin_name  TEXT NOT NULL,
      ns           TEXT NOT NULL,           -- "install" | "exec" | "command" | ...
      method       TEXT NOT NULL,
      caps         TEXT[] NOT NULL DEFAULT '{}',
      result       TEXT NOT NULL,           -- "ok" | "denied" | "error"
      duration_ms  INT  NOT NULL DEFAULT 0,
      args_hash    TEXT NOT NULL DEFAULT '',
      message      TEXT
  );
  CREATE INDEX IF NOT EXISTS idx_plugin_audit_plugin_ts ON plugin_audit(plugin_name, ts DESC);
  ```
- **Acceptance:** Fresh DB migrates cleanly; existing DB migrates idempotently (re-run safe). `SELECT relname FROM pg_class WHERE relname LIKE 'plugin_%'` returns all four.
- **Tests:** `kernel/store/plugin_tables_test.go` — uses `fergusstrange/embedded-postgres` (already in `go.sum`) to boot a clean Postgres, runs `Migrate`, asserts the four tables exist with expected columns via `information_schema.columns`.
- **Complexity:** S
- **Risk/Mitigation:** Low. Risk: `plugin_consents.plugin_name FK` could block install ordering. Mitigation: install flow writes to `plugins` row first (existing `UpsertPlugin`), then `plugin_consents`.

### T4 — Consent + audit queries
- **Depends on:** T3
- **Create:** `kernel/store/plugin_consents.go`
- **Modify:** —
- **Core types/functions:**
  ```go
  type PluginConsent struct {
      PluginName   string
      ManifestHash string
      PermsJSON    json.RawMessage
      GrantedAt    time.Time
      UpdatedAt    time.Time
  }
  func (d *DB) UpsertConsent(ctx, PluginConsent) error
  func (d *DB) GetConsent(ctx, name string) (PluginConsent, error)
  func (d *DB) DeleteConsent(ctx, name string) error

  type AuditEntry struct {
      PluginName, Ns, Method, Result, ArgsHash, Message string
      Caps       []string
      DurationMs int
  }
  func (d *DB) AppendAudit(ctx, AuditEntry) error
  func (d *DB) TailAudit(ctx, name string, limit int) ([]AuditEntry, error)
  ```
- **Acceptance:** Round-trip insert + read is exact. Delete of parent `plugins` row cascades to `plugin_consents` (FK `ON DELETE CASCADE`) and audit rows remain (audit is historical, not FK'd).
- **Tests:** `kernel/store/plugin_consents_test.go` — table-driven insert/read/delete; cascade test that creates a plugins row, a consent row, deletes the plugin row, asserts consent row gone and audit rows preserved.
- **Complexity:** S
- **Risk/Mitigation:** Low.

### T5 — Capability gate (no runtime I/O yet)
- **Depends on:** T1
- **Create:** `plugin/bridge/capabilities.go`
- **Core types/functions:**
  ```go
  type Gate struct { db *store.DB; log *slog.Logger }
  func NewGate(db *store.DB, log *slog.Logger) *Gate
  // Check returns nil if action is permitted given stored consent; otherwise *PermError.
  func (g *Gate) Check(ctx context.Context, plugin string, need Need) error
  type Need struct {
      Cap     string // "exec" | "fs.read" | "http" | ...
      Target  string // command line, URL, path — matcher input
  }
  type PermError struct{ Code, Msg string } // Code == "EPERM"
  func (e *PermError) Error() string

  // Matchers (pure, testable).
  func MatchExecGlobs(granted []string, cmdline string) bool
  func MatchHTTPURL(granted []string, rawURL string) bool
  func MatchFSPath(granted []string, absPath string) bool
  ```
- **Acceptance:** Given a consent row granting `exec: ["git *","npm *"]`, `Check("exec","git status")` returns nil and writes an `ok` audit row; `Check("exec","rm -rf /")` returns `*PermError{Code:"EPERM"}` and writes a `denied` audit row.
- **Tests:** `plugin/bridge/capabilities_test.go` — table-driven glob matcher tests, including the RFC1918 / link-local denial from `05-capabilities.md` §URL patterns. Audit-write assertion via mock `AppendAudit` recorder.
- **Complexity:** M
- **Risk/Mitigation:** Medium. Risk: glob matcher semantics drift from spec. Mitigation: use `filepath.Match` for fs, `path.Match` with scheme validation for http, `strings.HasPrefix` on normalised `cmd+" "+args` for exec — all three documented inline with examples from `05-capabilities.md`.

### T6 — Install package: local source + extract + consent token
- **Depends on:** T2, T4, T5
- **Create:** `plugin/install/install.go`, `plugin/install/source.go`, `plugin/install/consent.go`, `plugin/install/hash.go`
- **Core types/functions:**
  ```go
  type Source interface{ Fetch(ctx) (bundlePath string, cleanup func(), err error) }
  func ParseSource(raw string) (Source, error)   // dispatches on scheme
  type localSource struct{ path string }         // "local:/abs/path" or just "/abs/path"
  type httpsSource struct{ url string }          // returns ENotImplemented in M1
  type marketplaceSource struct{ raw string }    // returns ENotImplemented in M1

  type Installer struct {
      DataDir string            // e.g. ${OPENDRAY_DATA_DIR}/plugins/.installed
      DB      *store.DB
      Runtime *plugin.Runtime
      Gate    *bridge.Gate
      Log     *slog.Logger
  }

  type PendingInstall struct {
      Token        string          // random 32-byte hex
      Name         string
      Version      string
      ManifestHash string
      Perms        plugin.PermissionsV1
      StagedPath   string          // path in a temp dir; moved on confirm
      ExpiresAt    time.Time
  }

  func (i *Installer) Stage(ctx context.Context, src Source) (*PendingInstall, error)
  func (i *Installer) Confirm(ctx context.Context, token string) error
  func (i *Installer) Uninstall(ctx context.Context, name string) error

  // hash.go
  func SHA256File(path string) (string, error)
  func SHA256CanonicalManifest(p plugin.Provider) (string, error) // stable JSON with sorted keys
  ```
- **Installer.Stage flow:** fetch via `Source`, extract zip/tar or `cp -a` (local dir) to a temp dir, read `manifest.json`, `ValidateV1`, compute canonicalised manifest hash, generate consent token, store in an in-memory `map[token]*PendingInstall` protected by a mutex (TTL 10 min).
- **Installer.Confirm flow:** look up token, move staged dir to `${DataDir}/<name>/<version>/`, `UpsertPlugin`, `UpsertConsent`, `Runtime.Register`, audit `install/ok`. Pending entries past TTL are dropped by a background janitor goroutine started by `NewInstaller`.
- **Installer.Uninstall flow:** `Runtime.Remove`, `DeleteConsent`, `DeletePlugin` (cascades audit? no — audit is historical), remove extracted dir, emit audit `uninstall/ok`.
- **Acceptance:** Given `plugins/examples/time-ninja/`, `Stage("local:/abs/path/time-ninja")` returns a `PendingInstall` with `Name="time-ninja"`. `Confirm(token)` creates the extracted dir, writes `plugin_consents` row, makes `Runtime.Get("time-ninja")` return the provider.
- **Tests:** `plugin/install/install_test.go` — integration-style using embedded Postgres + `t.TempDir()` for `DataDir`. Scenarios: stage-then-confirm happy path, stage with invalid manifest rejects, expired token rejects, uninstall removes all traces (assert DB rows gone, dir gone). Name: `TestInstaller_HappyPath`, `TestInstaller_InvalidManifestRejected`, `TestInstaller_ExpiredTokenRejected`, `TestInstaller_UninstallRemovesAllTraces`.
- **Complexity:** L
- **Risk/Mitigation:** High — filesystem + DB + in-memory token state with concurrency. Mitigation: janitor uses a `sync.Mutex`; install atomicity via staged-dir + `os.Rename`; `t.Cleanup` ensures temp dirs removed; run with `-race` in CI.

### T7 — HTTP install endpoints
- **Depends on:** T6
- **Create:** `gateway/plugins_install.go`
- **Modify:** `gateway/server.go` — in the protected route group, add:
  ```go
  r.Post("/api/plugins/install",            s.pluginsInstall)
  r.Post("/api/plugins/install/confirm",    s.pluginsInstallConfirm)
  r.Delete("/api/plugins/{name}",           s.pluginsUninstall)
  r.Get("/api/plugins/{name}/audit",        s.pluginsAudit)
  ```
  Add new field `installer *plugininstall.Installer` to `Server` + `Config`; wire in `New`.
- **Core types/functions:**
  ```go
  func (s *Server) pluginsInstall(w, r)         // body: {"src":"local:/..."} → 202 {token, name, version, perms}
  func (s *Server) pluginsInstallConfirm(w, r)  // body: {"token":"..."} → 200 {installed: true}
  func (s *Server) pluginsUninstall(w, r)       // → 200 {status:"uninstalled"}
  func (s *Server) pluginsAudit(w, r)           // query: ?limit=100 → 200 [...entries]
  ```
- **Note:** The existing `DELETE /api/providers/{name}` route stays for legacy compat; new flows use `DELETE /api/plugins/{name}`. Both call through to the same `Runtime.Remove` (the new one additionally calls `Installer.Uninstall` for full teardown).
- **Acceptance:** `curl -X POST /api/plugins/install -d '{"src":"local:/abs/time-ninja"}'` returns 202 with a token. `curl -X POST /api/plugins/install/confirm -d '{"token":"..."}'` returns 200. `curl -X DELETE /api/plugins/time-ninja` returns 200 and the bundle dir is gone.
- **Tests:** `gateway/plugins_install_test.go` — `httptest.NewRecorder` + embedded-pg, assert full install→confirm→uninstall flow via HTTP. Name: `TestPluginsInstall_EndToEnd`.
- **Complexity:** M
- **Risk/Mitigation:** Medium — auth middleware coverage. Mitigation: register under the existing auth Group so the JWT middleware is automatic.

### T8 — Contribution registry
- **Depends on:** T1
- **Create:** `plugin/contributions/registry.go`
- **Core types/functions:**
  ```go
  type Registry struct { mu sync.RWMutex; byPlugin map[string]plugin.ContributesV1 }
  func NewRegistry() *Registry
  func (r *Registry) Set(pluginName string, c plugin.ContributesV1)
  func (r *Registry) Remove(pluginName string)

  type FlatContributions struct {
      Commands    []OwnedCommand       `json:"commands"`
      StatusBar   []OwnedStatusBarItem `json:"statusBar"`
      Keybindings []OwnedKeybinding    `json:"keybindings"`
      Menus       map[string][]OwnedMenuEntry `json:"menus"`
  }
  // "Owned" wrappers add PluginName field so the shell knows who contributed what.
  func (r *Registry) Flatten() FlatContributions
  ```
- **Registration hook:** `plugin.Runtime.Register` and `.Remove` call into `Registry.Set/Remove`. Modify `plugin/runtime.go` to take an optional `*contributions.Registry` via a functional option, default `nil` for backward compat.
- **Acceptance:** After registering `time-ninja`, `Flatten().StatusBar` contains one entry with `PluginName=="time-ninja"`. After `Remove`, the entry is gone.
- **Tests:** `plugin/contributions/registry_test.go` — concurrent Set/Remove with `-race`.
- **Complexity:** S
- **Risk/Mitigation:** Low.

### T9 — Workbench contributions endpoint
- **Depends on:** T8
- **Create:** `gateway/workbench.go`
- **Modify:** `gateway/server.go` — register `r.Get("/api/workbench/contributions", s.workbenchContributions)` inside the protected group.
- **Core types/functions:**
  ```go
  func (s *Server) workbenchContributions(w, r) // returns registry.Flatten()
  ```
- **Acceptance:** After installing `time-ninja`, `GET /api/workbench/contributions` returns a payload with all four slots populated.
- **Tests:** `gateway/workbench_test.go` — install a fixture plugin and assert JSON response shape.
- **Complexity:** S

### T10 — Command dispatcher
- **Depends on:** T5, T8
- **Create:** `plugin/commands/dispatcher.go`
- **Core types/functions:**
  ```go
  type Dispatcher struct {
      registry *contributions.Registry
      gate     *bridge.Gate
      tasks    *tasks.Runner
      hub      *hub.Hub
      log      *slog.Logger
  }
  func NewDispatcher(...) *Dispatcher
  // Invoke looks up the command by id, resolves the run spec, capability-checks, and executes.
  func (d *Dispatcher) Invoke(ctx context.Context, pluginName, commandID string, args map[string]any) (any, error)
  ```
- **Run-kind handlers (M1):**
  - `notify`: returns `{ok:true}` + client renders toast via the API response → Flutter shows a SnackBar.
  - `openUrl`: validated URL returned to client; Flutter calls `url_launcher`.
  - `exec`: spawns via `os/exec` with capability check against `permissions.exec` globs, 10-second hard timeout, output truncated to 16 KiB.
  - `runTask`: calls `tasks.Runner.Run` with the named task id; requires `permissions.exec` since tasks run shell.
  - `host`, `openView`: return `EUNAVAIL` with message "requires M2/M3".
- **Acceptance:** `Invoke("time-ninja","time.start",nil)` with `notify` run returns `{ok:true, kind:"notify", message:"Pomodoro started"}`. `Invoke("badplugin","cmd.bad")` where `exec` not granted returns `EPERM` and an audit row is written.
- **Tests:** `plugin/commands/dispatcher_test.go` — one test per run-kind including the EUNAVAIL paths. Name: `TestDispatcher_NotifyKind`, `TestDispatcher_ExecDeniedWithoutCapability`, `TestDispatcher_ExecAllowedWithCapability`, `TestDispatcher_HostKindReturnsEUNAVAIL`.
- **Complexity:** M
- **Risk/Mitigation:** Medium — `exec.CommandContext` leak. Mitigation: `context.WithTimeout` + `SetpgidOnFork`-style process-group cleanup on Unix.

### T11 — Command invoke HTTP endpoint
- **Depends on:** T10
- **Create:** `gateway/plugins_command.go`
- **Modify:** `gateway/server.go` — add `r.Post("/api/plugins/{name}/commands/{id}/invoke", s.commandInvoke)`.
- **Core types/functions:**
  ```go
  func (s *Server) commandInvoke(w, r) // body: {"args": {...}}
  ```
- **Acceptance:** `curl -X POST /api/plugins/time-ninja/commands/time.start/invoke -d '{}'` returns 200 `{kind:"notify", message:"Pomodoro started"}`.
- **Tests:** `gateway/plugins_command_test.go` — happy path + EPERM + EUNAVAIL paths via `httptest`.
- **Complexity:** S

### T12 — Compat-mode synthesis for legacy manifests
- **Depends on:** T1, T8
- **Create:** `plugin/compat/synthesize.go`
- **Modify:** `plugin/runtime.go` — in `loadIntoMemory`, if `p.IsV1() == false` call `compat.Synthesize(p)` to obtain a v1 overlay; push that overlay's `Contributes` into the registry. The on-disk manifest file is **never** rewritten.
- **Core types/functions:**
  ```go
  // Synthesize returns a v1-shaped overlay for a legacy Provider.
  // The overlay is held in memory only and never persisted to disk.
  func Synthesize(p plugin.Provider) plugin.Provider
  // Rules (from 07-lifecycle.md §Compat mode):
  //   form      = "host" if Type in {cli,local,shell} else "declarative"
  //   publisher = "opendray-builtin"
  //   engines   = {opendray: ">=0"}
  //   contributes.{agentProviders|views} populated from legacy fields
  //   permissions = {} (builtins are trusted)
  ```
- **Acceptance:** All 6 existing agent manifests continue to resolve CLI commands via `Runtime.ResolveCLI`. All 11 existing panel manifests stay listed in `GET /api/providers`. No on-disk `manifest.json` byte changes.
- **Tests:** `plugin/compat/synthesize_test.go` — load every `plugins/*/manifest.json` from `plugins.FS`, synthesize, assert `publisher=="opendray-builtin"` and that legacy fields are preserved. `TestCompat_NoDiskRewrite` asserts the underlying bundled manifest bytes are unchanged after boot.
- **Complexity:** M
- **Risk/Mitigation:** Medium — any field alias mismatch silently breaks legacy plugins. Mitigation: golden-file test comparing `Runtime.ListInfo()` output before/after T12 merges.

### T13 — `cmd/opendray plugin` subcommand skeleton
- **Depends on:** none (parallel-safe; no code wiring yet)
- **Create:** `cmd/opendray/plugin_cli.go`
- **Modify:** `cmd/opendray/main.go` — in the subcommand switch, add `case "plugin":` that delegates to `runPluginCLI(os.Args[2:])`.
- **Core types/functions:**
  ```go
  func runPluginCLI(args []string) int  // dispatches scaffold | install | validate
  func pluginCmdScaffold(args []string) int
  func pluginCmdInstall(args []string) int
  func pluginCmdValidate(args []string) int
  ```
- **Acceptance:** `opendray plugin --help` prints usage. Unknown subcommands exit 2 with error.
- **Tests:** `cmd/opendray/plugin_cli_test.go` — test help text + unknown command exit code via `exec.Command` invoking the built binary from a temp dir, skipped on `-short`.
- **Complexity:** S

### T14 — `opendray plugin scaffold --form declarative`
- **Depends on:** T13
- **Create:** `cmd/opendray/plugin_scaffold.go`, `cmd/opendray/templates/declarative/manifest.json.tmpl`, `cmd/opendray/templates/declarative/README.md.tmpl`, `cmd/opendray/templates/embed.go` (`//go:embed all:declarative`).
- **Core types/functions:**
  ```go
  type scaffoldOpts struct { form, name, publisher, outDir string }
  func pluginCmdScaffold(args []string) int  // parses flags, writes files via text/template
  ```
- **Template output:** generates directory `<name>/` containing `manifest.json` with one command, one status-bar item, one keybinding, one menu entry, all pointing to each other. Includes `README.md` with three-step test recipe. No webview, no host.
- **Acceptance:** `opendray plugin scaffold --form declarative my-plugin` creates `./my-plugin/` such that `opendray plugin validate ./my-plugin` passes, and `opendray plugin install ./my-plugin` (T15) succeeds end-to-end against a running server.
- **Tests:** `cmd/opendray/plugin_scaffold_test.go` — scaffold into `t.TempDir()`, read the generated manifest, run it through `plugin.ValidateV1`, assert no errors.
- **Complexity:** M
- **Risk/Mitigation:** Medium — template/publisher name collisions. Mitigation: reject names that don't match the manifest name regex before writing any files.

### T15 — `opendray plugin install <path>` CLI
- **Depends on:** T7, T13
- **Create:** `cmd/opendray/plugin_install_cli.go`
- **Core types/functions:**
  ```go
  func pluginCmdInstall(args []string) int
  // Reads OPENDRAY_SERVER_URL + OPENDRAY_TOKEN from env or ~/.opendray/cli.toml,
  // POSTs to /api/plugins/install, prints perms, prompts y/N, POSTs /confirm.
  ```
- **Acceptance:** Against a running server with `OPENDRAY_ALLOW_LOCAL_PLUGINS=1`, `opendray plugin install ./time-ninja` prompts for consent, prints the capability list, installs on confirm. `--yes` flag skips prompt.
- **Tests:** `cmd/opendray/plugin_install_cli_test.go` — use `httptest.NewServer` as fake backend, assert the two requests in order.
- **Complexity:** M
- **Risk/Mitigation:** Low — CLI is a thin HTTP client.

### T16 — `opendray plugin validate [dir]` CLI
- **Depends on:** T2, T13
- **Create:** `cmd/opendray/plugin_validate_cli.go`
- **Core types/functions:**
  ```go
  func pluginCmdValidate(args []string) int
  // Reads manifest.json from the given dir (default "."), runs ValidateV1, prints errors with path: msg; exit 1 if any.
  ```
- **Acceptance:** Valid plugin prints `ok`, exit 0. Invalid prints `error: contributes.commands[0].id: invalid format`, exit 1.
- **Tests:** `cmd/opendray/plugin_validate_cli_test.go` — fixture dir with known-bad manifest, assert exit code + stderr contains the path.
- **Complexity:** S

### T17 — Reference plugin `time-ninja`
- **Depends on:** T1
- **Create:** `plugins/examples/time-ninja/manifest.json`, `plugins/examples/time-ninja/README.md`
- **Modify:** — (not embedded; lives as an on-disk example that the acceptance-test harness installs from a local path)
- **Acceptance:** See §10 for the exact manifest. `plugin.ValidateV1` returns empty slice. After install, all four contribution points are populated in `GET /api/workbench/contributions`.
- **Tests:** covered by T18 E2E.
- **Complexity:** S

### T18 — E2E install + invoke test harness
- **Depends on:** T6, T7, T10, T11, T17
- **Create:** `plugin/e2e_test.go` (build tag `//go:build e2e`)
- **Core types/functions:** spin up `gateway.Server` with embedded Postgres, call the full flow via `httptest.Server`:
  1. `POST /api/plugins/install {src:"local:plugins/examples/time-ninja"}` → assert 202 + token + perms list contains no dangerous items.
  2. `POST /api/plugins/install/confirm {token}` → assert 200.
  3. `GET /api/workbench/contributions` → assert `commands[].id` contains `time.start`, `statusBar[]` has `time.bar`, `keybindings[]` has `ctrl+alt+p`, `menus["statusBar/right"]` has the entry.
  4. `POST /api/plugins/time-ninja/commands/time.start/invoke` → assert 200 with `{kind:"notify"}`.
  5. Restart harness (close + re-create `gateway.Server` with same DB) → repeat step 3 → contributions still present (survives restart).
  6. `DELETE /api/plugins/time-ninja` → assert 200, extracted dir gone, `plugin_consents` row gone, in-memory registry empty.
- **Acceptance:** Run with `go test -race -tags=e2e ./plugin/...` — passes.
- **Tests:** self-contained.
- **Complexity:** L
- **Risk/Mitigation:** High — embedded-pg boot flakiness. Mitigation: `t.Cleanup` + 30-second boot timeout + retry-on-port-in-use.

### T19 — Flutter command palette widget
- **Depends on:** T9
- **Create:** `app/lib/features/workbench/command_palette.dart`, `app/lib/features/workbench/workbench_service.dart`, `app/lib/features/workbench/workbench_models.dart`
- **Modify:** `app/lib/app.dart` — wrap `MaterialApp.router`'s `builder` with a `Shortcuts` + `Actions` widget that listens for `Cmd/Ctrl+Shift+P` and shows the palette via an `Overlay`.
- **Core widgets/services:**
  - `WorkbenchService extends ChangeNotifier` — fetches `/api/workbench/contributions` on app start + after plugin install events. Holds `List<WorkbenchCommand>`, `List<WorkbenchStatusBarItem>`, etc.
  - `CommandPalette` — shows a searchable list (fuzzy match on title + plugin name), invokes `ApiClient.invokePluginCommand(pluginName, id)` on selection, handles the returned `kind` (`notify`→SnackBar, `openUrl`→`url_launcher`, `exec`→shows output in a bottom sheet).
- **Acceptance:** Cmd/Ctrl+Shift+P (desktop/web) opens the palette; typing `pomodoro` filters to `time.start`; pressing Enter invokes the command and shows a SnackBar "Pomodoro started".
- **Tests:** `app/test/features/workbench/command_palette_test.dart` — widget test mounting `CommandPalette` with a fake `WorkbenchService` and a fake `ApiClient`, asserting filter + tap behaviour.
- **Complexity:** M
- **Risk/Mitigation:** Medium — keybinding registration on mobile (Android soft-keyboard + iOS) is non-trivial. Mitigation: M1 ships desktop + web keybindings only; mobile access is via a new "Command" entry in the app-bar actions. Mobile keybindings tracked as an M2 polish item.

### T20 — Flutter status-bar strip
- **Depends on:** T9, T19
- **Create:** `app/lib/features/workbench/status_bar_strip.dart`
- **Modify:** `app/lib/features/dashboard/dashboard_page.dart` — inject `StatusBarStrip` as `bottomNavigationBar` (or as a thin footer `Row`). `app/lib/features/session/session_page.dart` — same.
- **Core widgets:**
  ```dart
  class StatusBarStrip extends StatelessWidget {
    // Reads WorkbenchService.statusBarItems, renders left-group + right-group
    // with priority sort; tapping an item invokes its bound command via
    // CommandPaletteService.invoke.
  }
  ```
- **Acceptance:** Installing `time-ninja` causes the "🍅 25:00" chip to appear at the bottom right of both the Dashboard and Session pages. Tapping fires `time.start` (shows notify SnackBar).
- **Tests:** `app/test/features/workbench/status_bar_strip_test.dart`.
- **Complexity:** S

### T21 — Flutter keybinding dispatcher
- **Depends on:** T19
- **Create:** `app/lib/features/workbench/keybindings.dart`
- **Modify:** `app/lib/app.dart` — mount a `CallbackShortcuts` at the root whose map is rebuilt whenever `WorkbenchService.keybindings` changes.
- **Core widgets:**
  ```dart
  class WorkbenchKeybindings extends StatefulWidget { final Widget child; ... }
  // Parses "ctrl+alt+p" into a LogicalKeySet, binds to a callback that invokes the command.
  ```
- **Acceptance:** With `time-ninja` installed, pressing `Ctrl+Alt+P` on web/desktop fires `time.start`.
- **Tests:** `app/test/features/workbench/keybindings_test.dart` — sendKeyEvent harness.
- **Complexity:** M
- **Risk/Mitigation:** Medium — key-set parser robustness. Mitigation: table-driven tests for 20+ key combos including `mac` overrides.

### T22 — Flutter menu slot
- **Depends on:** T9
- **Create:** `app/lib/features/workbench/menu_slot.dart`
- **Modify:** `app/lib/features/dashboard/dashboard_page.dart` — add `MenuSlot(id: 'appBar/right')` next to the existing `FilledButton.icon('New')`.
- **Core widgets:**
  ```dart
  class MenuSlot extends StatelessWidget {
    final String id;
    // Reads WorkbenchService.menus[id], renders as a popup menu of contributed entries.
  }
  ```
- **Acceptance:** A menu entry declared for slot `appBar/right` in `time-ninja` shows up in the dashboard app-bar popup.
- **Tests:** `app/test/features/workbench/menu_slot_test.dart`.
- **Complexity:** S

### T23 — `ApiClient.invokePluginCommand` + `ApiClient.getContributions`
- **Depends on:** T11, T9
- **Modify:** `app/lib/core/api/api_client.dart` — add two methods + DTOs in `app/lib/features/workbench/workbench_models.dart`.
- **Acceptance:** Methods serialise/deserialise cleanly; errors mapped to typed exceptions (`PluginPermissionDeniedException`, `PluginCommandUnavailableException`).
- **Tests:** `app/test/core/api/workbench_api_test.dart`.
- **Complexity:** S

### T24 — Uninstall-leaves-no-trace verification
- **Depends on:** T6, T8, T18
- **Add to:** T18's E2E test.
- **Checks:** after `DELETE /api/plugins/time-ninja`:
  - `SELECT count(*) FROM plugins WHERE name='time-ninja'` → 0
  - `SELECT count(*) FROM plugin_consents WHERE plugin_name='time-ninja'` → 0
  - `os.Stat("${DataDir}/time-ninja/1.0.0")` → `ErrNotExist`
  - `registry.Flatten().Commands` contains no entries with `PluginName=="time-ninja"`
- **Acceptance:** assertion helpers above all pass.
- **Complexity:** S

### T25 — `OPENDRAY_ALLOW_LOCAL_PLUGINS` gate + data dir config
- **Depends on:** T6, T7
- **Modify:** `kernel/config/config.go` (find by searching) — add `PluginsDataDir` (default `${user_home}/.opendray/plugins`) + `AllowLocalPlugins` (default false, env-overridable via `OPENDRAY_ALLOW_LOCAL_PLUGINS`). Wire into `gateway.New` → `Installer`.
- **Enforcement:** `Installer.ParseSource` refuses `local:` scheme when `AllowLocalPlugins` is false, returning a structured error surfaced as HTTP 403.
- **Acceptance:** Without the env var, local install returns 403 "local plugin installs disabled; set OPENDRAY_ALLOW_LOCAL_PLUGINS=1".
- **Tests:** `plugin/install/install_test.go` — add `TestInstaller_LocalSourceGatedByEnv`.
- **Complexity:** S

---

## 3. Suggested linear ordering

Critical path (single-thread, 17 sequential hops):

```
T3 → T4 → T1 → T2 → T5 → T6 → T7 → T8 → T12 → T9 → T10 → T11 → T17 → T18 → T19 → T20 → T21
```

### Fork points (safe parallelism)

**After T1 lands** (first PR seam — see §9), three branches can run concurrently:

- **Branch A — Install backbone:** T3 → T4 → T5 → T6 → T7 → T25 → T18 (E2E)
- **Branch B — CLI:** T13 → T14, T15, T16 (three tasks parallel-safe)
- **Branch C — Flutter:** T23 first (blocks nothing server-side), then T19 → T20, T21, T22 parallel

**After T8 lands:** T9, T10 can run in parallel.
**After T12 lands:** compat test rerun validates nothing regressed; no new work unblocked beyond confidence.
**T17 (reference plugin) has no dependencies beyond T1** — it's just a JSON file — so it can be authored extremely early and used as a fixture for every test.

---

## 4. Interfaces locked in M1

### 4.1 Go interfaces + core types

```go
// plugin/manifest.go (extended)
type Provider struct {
    // existing fields kept
    Name, DisplayName, Description, Icon, Version, Type, Category string
    CLI          *CLISpec
    Capabilities Capabilities
    ConfigSchema []ConfigField

    // v1 additions (all optional)
    Publisher   string          `json:"publisher,omitempty"`
    Engines     EnginesV1       `json:"engines,omitempty"`
    Form        string          `json:"form,omitempty"`        // "declarative" | "webview" | "host"
    Activation  []string        `json:"activation,omitempty"`
    Contributes ContributesV1   `json:"contributes,omitempty"`
    Permissions PermissionsV1   `json:"permissions,omitempty"`
    V2Reserved  json.RawMessage `json:"v2Reserved,omitempty"`
}

type EnginesV1 struct {
    Opendray string `json:"opendray"`
    Node     string `json:"node,omitempty"`
    Deno     string `json:"deno,omitempty"`
}

type ContributesV1 struct {
    Commands    []CommandV1                `json:"commands,omitempty"`
    StatusBar   []StatusBarItemV1          `json:"statusBar,omitempty"`
    Keybindings []KeybindingV1             `json:"keybindings,omitempty"`
    Menus       map[string][]MenuEntryV1   `json:"menus,omitempty"`
    // NOTE: views / activityBar / panels / settings / languageServers / etc.
    // are accepted as json.RawMessage and round-tripped but NOT acted upon in M1.
    Raw json.RawMessage `json:"-"` // populated by custom UnmarshalJSON for M2+ fields
}

type CommandV1 struct {
    ID       string        `json:"id"`
    Title    string        `json:"title"`
    Icon     string        `json:"icon,omitempty"`
    Category string        `json:"category,omitempty"`
    When     string        `json:"when,omitempty"`
    Run      CommandRunV1  `json:"run"`
}

type CommandRunV1 struct {
    Kind    string   `json:"kind"` // "notify" | "openUrl" | "exec" | "runTask" | "host" | "openView"
    Message string   `json:"message,omitempty"`
    URL     string   `json:"url,omitempty"`
    Command string   `json:"command,omitempty"` // for exec
    Args    []string `json:"args,omitempty"`
    TaskID  string   `json:"taskId,omitempty"`
    ViewID  string   `json:"viewId,omitempty"`
    Method  string   `json:"method,omitempty"` // for host (EUNAVAIL in M1)
}

type StatusBarItemV1 struct {
    ID        string `json:"id"`
    Text      string `json:"text"`
    Tooltip   string `json:"tooltip,omitempty"`
    Command   string `json:"command,omitempty"`
    Alignment string `json:"alignment,omitempty"` // "left" | "right"
    Priority  int    `json:"priority,omitempty"`
}

type KeybindingV1 struct {
    Command string `json:"command"`
    Key     string `json:"key"`
    Mac     string `json:"mac,omitempty"`
    When    string `json:"when,omitempty"`
}

type MenuEntryV1 struct {
    Command string `json:"command"`
    Group   string `json:"group,omitempty"`
    When    string `json:"when,omitempty"`
}

type PermissionsV1 struct {
    FS        FSPerm       `json:"fs,omitempty"`
    Exec      ExecPerm     `json:"exec,omitempty"`
    HTTP      HTTPPerm     `json:"http,omitempty"`
    Session   string       `json:"session,omitempty"`   // "" | "read" | "write"
    Storage   bool         `json:"storage,omitempty"`
    Secret    bool         `json:"secret,omitempty"`
    Clipboard string       `json:"clipboard,omitempty"`
    Telegram  bool         `json:"telegram,omitempty"`
    Git       string       `json:"git,omitempty"`
    LLM       bool         `json:"llm,omitempty"`
    Events    []string     `json:"events,omitempty"`
}

// FSPerm, ExecPerm, HTTPPerm each use json.RawMessage + custom UnmarshalJSON
// to accept both bool and the richer object/array forms from the v1 schema.
type FSPerm struct{ All bool; Read, Write []string }
type ExecPerm struct{ All bool; Globs []string }
type HTTPPerm struct{ All bool; Patterns []string }
```

### 4.2 HTTP endpoints introduced

| Method + Path | Auth | Request body | Response |
|---|---|---|---|
| `POST /api/plugins/install` | JWT | `{"src":"local:/abs/path" \| "marketplace://..." \| "https://..."}` | `202 {token, name, version, perms, manifestHash}` or `403/501` |
| `POST /api/plugins/install/confirm` | JWT | `{"token":"hex"}` | `200 {installed:true, enabled:true}` |
| `DELETE /api/plugins/{name}` | JWT | — | `200 {status:"uninstalled"}` |
| `GET /api/plugins/{name}/audit?limit=N` | JWT | — | `200 [{ts, ns, method, result, caps, durationMs, argsHash, message}...]` |
| `GET /api/workbench/contributions` | JWT | — | `200 FlatContributions` (see T8) |
| `POST /api/plugins/{name}/commands/{id}/invoke` | JWT | `{"args":{...}}` | `200 {kind:"notify"\|"openUrl"\|"exec", ...}` or `403 EPERM` or `501 EUNAVAIL` |

Existing `/api/providers/*` routes stay untouched for compat. The terminal/session WebSocket is unrelated.

### 4.3 DB tables introduced

See §2 T3 for full DDL. Column types + constraints are the contract.

### 4.4 JSON schema for action responses

```json
{ "kind": "notify",  "message": "string" }
{ "kind": "openUrl", "url": "https://..." }
{ "kind": "exec",    "exitCode": 0, "stdout": "...", "stderr": "...", "timedOut": false }
```

Errors:
```json
{ "error": { "code": "EPERM",    "message": "exec not granted for: rm -rf /" } }
{ "error": { "code": "EUNAVAIL", "message": "kind=host requires M2/M3" } }
{ "error": { "code": "EINVAL",   "message": "bad command id" } }
```

### 4.5 CLI contract (subcommands)

```
opendray plugin scaffold --form declarative <name> [--publisher <id>] [--out <dir>]
opendray plugin validate [dir]
opendray plugin install <path-or-url> [--yes]
```

Env vars read by CLI: `OPENDRAY_SERVER_URL`, `OPENDRAY_TOKEN`.

---

## 5. Test strategy

### Unit tests (go test, `-race`, target ≥80% on touched packages)
- `plugin/manifest_test.go` (extend) + `plugin/manifest_v1_test.go` + `plugin/manifest_validate_test.go`
- `plugin/bridge/capabilities_test.go` — glob matchers, audit-write assertion via mock recorder
- `plugin/compat/synthesize_test.go` — every bundled manifest goes through compat, output asserted field-by-field
- `plugin/contributions/registry_test.go` — concurrent set/remove under `-race`
- `plugin/commands/dispatcher_test.go` — one test per run-kind
- `plugin/install/install_test.go` — uses `t.TempDir()`, `embedded-postgres`, covers stage/confirm/uninstall/expired-token/env-gate
- `kernel/store/plugin_consents_test.go`, `kernel/store/plugin_tables_test.go`
- `gateway/plugins_install_test.go`, `gateway/workbench_test.go`, `gateway/plugins_command_test.go`
- `cmd/opendray/plugin_scaffold_test.go`, `plugin_validate_cli_test.go`, `plugin_install_cli_test.go`

### Integration tests (build tag `//go:build e2e`)
- `plugin/e2e_test.go` — full install → contributions visible → invoke → restart → contributions still visible → uninstall → no traces. This test IS the M1 acceptance criterion mechanised.

### Flutter widget tests
- `app/test/features/workbench/command_palette_test.dart`
- `app/test/features/workbench/status_bar_strip_test.dart`
- `app/test/features/workbench/keybindings_test.dart`
- `app/test/features/workbench/menu_slot_test.dart`
- `app/test/core/api/workbench_api_test.dart`

### Coverage gate
- CI runs `go test -race -cover ./...`; PR fails if any touched package drops below 80% line coverage. The untouched `cmd/opendray/setup_cli.go`, `kernel/hub/*`, `gateway/git/*` etc. are exempt (not touched).

### End-to-end fixture
- `plugins/examples/time-ninja/` is the canonical fixture. Tests install from its absolute path via `os.Getwd() + "/../examples/time-ninja"`.

---

## 6. Migration & compat

### Migration sequence
Append to the existing `files` slice in `kernel/store/db.go` `Migrate`:
```go
"migrations/010_plugin_consents.sql",
"migrations/011_plugin_kv.sql",
"migrations/012_plugin_secret.sql",
"migrations/013_plugin_audit.sql",
```
Each uses `CREATE TABLE IF NOT EXISTS` so re-runs are idempotent, matching the existing pattern (see `001_init.sql`).

### Rollback
Forward-only by convention. If M1 needs emergency rollback, drop in order `013, 012, 011, 010` — documented in `kernel/store/migrations/README.md` (new file).

### Compat invariants
1. Every file under `plugins/agents/*/manifest.json` and `plugins/panels/*/manifest.json` is read byte-for-byte unchanged. Git diff across M1 merge must show zero changes under these directories (enforced by T12 golden-file test).
2. `plugin.Provider` fields used by the existing `ResolveCLI`, `DetectModels`, `HealthCheck`, `ListInfo` paths retain identical semantics.
3. The `plugins` DB table schema is untouched. New metadata lives in the four new tables.
4. Existing `/api/providers/*` routes continue to work identically; new `/api/plugins/*` routes are additive.

### Post-M1 DB shape
```
plugins              (existing — unchanged)
plugin_consents      (new)
plugin_kv            (new — scaffold only)
plugin_secret        (new — scaffold only)
plugin_audit         (new)
sessions, mcp_*, claude_accounts, llm_providers, admin_auth (unchanged)
```

---

## 7. Definition of Done

- [ ] `plugins/examples/time-ninja/` installs via `opendray plugin install ./plugins/examples/time-ninja` (local path) and works end-to-end.
- [ ] All 6 existing `plugins/agents/*` launch unchanged; `Runtime.ResolveCLI` output byte-identical to pre-M1 (golden file).
- [ ] All 11 existing `plugins/panels/*` load unchanged (compat path); `GET /api/providers` returns the same set as today.
- [ ] Uninstall removes every trace: verified by SQL query (`plugins`, `plugin_consents` rows = 0) + filesystem check (`${DataDir}/time-ninja` gone) + in-memory registry check.
- [ ] `time-ninja` plugin survives backend restart (installed state persists in `plugins` + `plugin_consents` tables; contributions re-populate from DB on boot).
- [ ] `opendray plugin scaffold --form declarative new-plugin` produces a working plugin in one command; the output passes `opendray plugin validate`.
- [ ] Capability gate blocks an un-consented `exec` call with `EPERM`, and a `plugin_audit` row is written with `result="denied"`.
- [ ] `go test -race -cover ./...` passes with ≥80% line coverage on every package touched by M1.
- [ ] `go vet ./...` clean. `staticcheck ./...` clean.
- [ ] `gosec ./...` reports no new HIGH findings introduced by M1 code.
- [ ] Compat-mode smoke test: one legacy manifest (`plugins/agents/claude`) + one v1 manifest (`time-ninja`) coexist in the same running runtime; both are visible in `ListInfo()`.
- [ ] `marketplace://` scheme parses without panic but returns `501 ENotImplemented` with message "marketplace ships in M4".
- [ ] Flutter command palette opens on `Cmd/Ctrl+Shift+P` (desktop + web builds).
- [ ] Flutter status-bar strip renders `time-ninja`'s chip on dashboard + session pages.
- [ ] Flutter keybinding `Ctrl+Alt+P` fires `time.start` on desktop/web.
- [ ] M1-PLAN.md and all referenced design docs remain consistent: no in-code comment contradicts a `> **Locked:**` line from `/docs/plugin-platform/*.md`.

---

## 8. Out-of-scope escape valves

Top 5 places M1 will feel the pull of M2+ territory, with sanctioned workarounds:

1. **"We need `opendray.workbench.showMessage` for the `notify` command to feel native."**
   Workaround: the `notify` kind returns `{kind:"notify", message:"..."}` in the HTTP response body; the Flutter caller renders the SnackBar client-side. No bridge method needed. This is a deliberate M1 simplification — a webview plugin can't call `notify` because there is no webview in M1.

2. **"Status-bar text needs to update dynamically (e.g. countdown timer)."**
   Workaround: M1 ships static status-bar text only. Dynamic updates require `opendray.workbench.updateStatusBar`, which is M2. The `time-ninja` manifest uses the static placeholder `"🍅 25:00"`; acceptance doesn't depend on live updates.

3. **"Tests need real `storage.get/set` to validate state survives restart."**
   Workaround: plugin-registered state IS persisted via the existing `plugins.config` JSONB column + `plugin_consents` row. `plugin_kv` is DDL-scaffolded in M1 but has no reader/writer — verified by a grep-guard test asserting zero references to `plugin_kv` from `.go` source except the migration.

4. **"I want to test that revoking `exec` at runtime causes the next call to fail."**
   Workaround: M1 has no runtime-toggle UI. The manifest-declared permissions are the only input. To simulate revocation in tests, delete the `plugin_consents` row directly — acceptable because the runtime consent UI is explicitly M2+.

5. **"A plugin needs to declare a view in the activity bar."**
   Workaround: declare nothing. `contributes.views` + `contributes.activityBar` are parsed (held as `json.RawMessage` round-trip) but not rendered in M1. Document this in the SDK scaffold README as "[coming in M2]". The plugin will still install; its views just won't appear.

---

## 9. First-PR seam

**Smallest mergeable commit that unlocks parallel work:** T1 + T3.

Contents:
- `plugin/manifest.go` — add all v1 optional fields (T1)
- `kernel/store/migrations/010_plugin_consents.sql` through `013_plugin_audit.sql` (T3)
- `kernel/store/db.go` — append four migration paths
- `plugin/manifest_v1_test.go` — compat tests covering all 17 bundled manifests

Net behaviour change: **zero**. No new endpoints, no new runtime code paths fire. The four new DB tables exist but nothing writes to them.

Why first: unblocks three parallel branches (see §3 fork points). Anyone can start building against the new types without stepping on each other.

Size target: ≤400 lines changed. Reviewable in one sitting.

---

## 10. Reference plugin spec — `plugins/examples/time-ninja/`

Full file contents:

**`plugins/examples/time-ninja/manifest.json`**
```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "time-ninja",
  "version": "1.0.0",
  "publisher": "opendray-examples",
  "displayName": "Time Ninja",
  "description": "Pomodoro reminder that lives in the status bar. The M1 reference plugin.",
  "icon": "🍅",
  "engines": { "opendray": "^1.0.0" },
  "form": "declarative",
  "activation": ["onStartup"],
  "contributes": {
    "commands": [
      {
        "id": "time.start",
        "title": "Start Pomodoro",
        "category": "Time Ninja",
        "run": { "kind": "notify", "message": "Pomodoro started — 25 minutes" }
      }
    ],
    "statusBar": [
      {
        "id": "time.bar",
        "text": "🍅 25:00",
        "tooltip": "Start a pomodoro",
        "command": "time.start",
        "alignment": "right",
        "priority": 50
      }
    ],
    "keybindings": [
      { "command": "time.start", "key": "ctrl+alt+p", "mac": "cmd+alt+p" }
    ],
    "menus": {
      "appBar/right": [
        { "command": "time.start", "group": "timer@1" }
      ]
    }
  },
  "permissions": {}
}
```

**`plugins/examples/time-ninja/README.md`**
```markdown
# time-ninja

The OpenDray Plugin Platform M1 reference plugin. Exercises all four declarative
contribution points (commands, statusBar, keybindings, menus) and one capability
posture (empty permissions — no risky grants).

## Try it

1. Start OpenDray with `OPENDRAY_ALLOW_LOCAL_PLUGINS=1 opendray`.
2. `opendray plugin install ./plugins/examples/time-ninja`.
3. Confirm the empty-permissions consent screen.
4. Press `Ctrl+Alt+P` (or `Cmd+Alt+P` on Mac), or tap the 🍅 chip in the status bar.
5. You should see a "Pomodoro started — 25 minutes" toast.

## What this plugin proves

- Install flow works with zero dangerous caps (consent screen shows a harmless list).
- Contribution points round-trip through the manifest parser + registry + HTTP API.
- The command dispatcher's `notify` run-kind is functional.
- Keybinding, status-bar, menu all fire the same command id.
- Uninstall leaves no trace.
```

Notes:
- Zero declared permissions → the consent screen in T19 shows "no special permissions" — the happy unit-testable baseline.
- `menus["appBar/right"]` hits the menu slot registered by T22 on the dashboard.
- Localization (`%keys%`) is deliberately not used in M1 — deferred to M6.

---

## Executive summary (≤200 words)

**Total tasks:** 25. **Critical path:** 17 sequential tasks (T3→T4→T1→T2→T5→T6→T7→T8→T12→T9→T10→T11→T17→T18→T19→T20→T21). **Target calendar:** ~4 working weeks with one backend engineer + one Flutter engineer working in parallel after T1+T3 merge (first-PR seam, §9). **Biggest unknown:** whether the capability-glob matcher semantics in T5 will exactly match the spec in `05-capabilities.md` across edge cases (RFC1918 denial, path-variable expansion of `${workspace}`); mitigated by table-driven tests derived verbatim from the doc plus a PR reviewer checklist flagging any implementation that "simplifies" the spec.

**Suggested first task:** **T1 — Extend manifest struct.** Additive, zero runtime behaviour change, unblocks every downstream task (validator, install, dispatcher, compat, CLI, reference plugin). Ship it with T3's four SQL migrations in the same PR per §9. Do not start any other task until this lands.

**Non-negotiable boundary:** no webview, no bridge WebSocket, no `plugin://` asset scheme, no sidecar supervisor, no marketplace client, no hot reload. Those are M2/M3/M4/M6. If a task in this plan starts drifting toward any of them, stop and file a DEFERRED entry in §1.

---

## Relevant file paths (all absolute)

Design contract:
- `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/01-architecture.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/02-manifest.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/05-capabilities.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/07-lifecycle.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md`

Existing code anchors:
- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/hooks.go`
- `/home/linivek/workspace/opendray/plugin/manifest_test.go`
- `/home/linivek/workspace/opendray/kernel/store/db.go`
- `/home/linivek/workspace/opendray/kernel/store/queries.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/001_init.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/003_plugin_config.sql`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/api.go`
- `/home/linivek/workspace/opendray/plugins/embed.go`
- `/home/linivek/workspace/opendray/plugins/agents/claude/manifest.json`
- `/home/linivek/workspace/opendray/plugins/panels/git/manifest.json`
- `/home/linivek/workspace/opendray/cmd/opendray/main.go`
- `/home/linivek/workspace/opendray/app/lib/app.dart`
- `/home/linivek/workspace/opendray/app/lib/features/dashboard/dashboard_page.dart`
- `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart`

Files to be created (M1):
- `/home/linivek/workspace/opendray/plugin/manifest_validate.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go`
- `/home/linivek/workspace/opendray/plugin/install/{install,source,consent,hash}.go`
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go`
- `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go`
- `/home/linivek/workspace/opendray/plugin/compat/synthesize.go`
- `/home/linivek/workspace/opendray/gateway/plugins_install.go`
- `/home/linivek/workspace/opendray/gateway/plugins_command.go`
- `/home/linivek/workspace/opendray/gateway/workbench.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/{010_plugin_consents,011_plugin_kv,012_plugin_secret,013_plugin_audit}.sql`
- `/home/linivek/workspace/opendray/cmd/opendray/{plugin_cli,plugin_scaffold,plugin_install_cli,plugin_validate_cli}.go`
- `/home/linivek/workspace/opendray/cmd/opendray/templates/declarative/{manifest.json.tmpl,README.md.tmpl}`
- `/home/linivek/workspace/opendray/plugins/examples/time-ninja/{manifest.json,README.md}`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/{command_palette,status_bar_strip,keybindings,menu_slot,workbench_service,workbench_models}.dart`
