# Implementation Plan: OpenDray Plugin Platform M3 — Host sidecar runtime

> Output file: `/home/linivek/workspace/opendray/docs/plugin-platform/M3-PLAN.md`
> Design contract: `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md` §M3
> Predecessor: M2 shipped on branch `kevlab` — 23/26 tasks landed (see `docs/plugin-platform/M2-RELEASE.md`); T16b desktop WebView, T23 E2E kanban, T25 CSP test rolled forward into M3 as carry-ons.
> North star: `rust-analyzer-od` (or the minimal `fs-readme` reference plugin) provides LSP completion for a Rust file served by `opendray.fs.read`; killing the sidecar mid-request returns a clean `EUNAVAIL` and the supervisor restarts it within its backoff window. Every privileged call survives the capability gate with a declared path/command/URL/namespace match — no path traversal, no SSRF, no fork-bomb.

---

## 1. Scope boundary

**IN (M3 contract):**

- **`opendray.fs.*`** — `readFile`, `writeFile`, `exists`, `stat`, `readDir`, `mkdir`, `remove`, `watch`. All I/O is performed host-side; the plugin never holds a file descriptor. Paths are matched against declared glob allowlists in `permissions.fs.{read|write}` after canonicalisation, with `${workspace}`, `${home}`, `${dataDir}`, `${tmp}` base-variable expansion (consistent with 05-capabilities.md §Path patterns). Soft caps: 10 MiB per read, 10 MiB per write, 4096 entries per `readDir`. `watch` returns a subscription stream id via the same chunk/end envelopes T11 uses for `events.subscribe`.
- **`opendray.exec.*`** — `run` (one-shot), `spawn` (streamed), `kill`, `wait`. Command-line allowlist matched via existing `bridge.MatchExecGlobs`. Default 10 s timeout per spawn (override via `opts.timeoutMs` up to 5 minutes). Combined stdout+stderr stream-chunk envelopes, 16 KiB per-chunk cap (mirrors the existing `commands.Dispatcher.truncate` constant but streams incrementally rather than truncating). Linux: per-process cgroup (CPU + mem soft limits) when `cgroup v2` is mounted — documented gap on macOS and Windows. All platforms: `Setpgid=true` so the supervisor can kill the entire process group on revoke / disconnect.
- **`opendray.http.*`** — `request`, `stream`. URL-pattern allowlist via the existing `bridge.MatchHTTPURL` (RFC1918 / loopback / link-local deny stays). Request body cap 4 MiB, response body cap 16 MiB, response streaming uses `{stream:"chunk"}` envelopes. Redirect limit 5, follows only to URLs that still match the plugin's allowlist. TLS: Go's default `crypto/tls` verification.
- **`opendray.secret.*`** — `get`, `set`, `delete`. Values are AES-GCM encrypted at rest in the existing `plugin_secret` table. The Key Encryption Key (KEK) is derived via HKDF-SHA256 from the host's credential-store master secret (same material that protects admin bcrypt rows today — see `kernel/auth/credentials.go`); the data-encryption key (DEK) is 32 random bytes wrapped with the KEK and stored in a new `plugin_secret_kek` row per plugin. Never log plaintext. Never return another plugin's secret.
- **Host sidecar supervisor** — new `plugin/host/supervisor.go`: spawns a sidecar subprocess per `form:"host"` plugin on demand, pipes stdio, frames JSON-RPC 2.0 with LSP `Content-Length: N\r\n\r\n<json>` headers, restarts on crash with exponential backoff (200 ms → 5 s, capped), shuts the sidecar down after `IdleShutdownMinutes` (default 10) of no requests. One sidecar per plugin. Supervisor is feature-flagged off on iOS — `form:"host"` plugins refuse to install on iOS builds (per 10-security.md iOS policy).
- **`form:"host"` manifest wiring** — activate the already-reserved `HostV1` struct (`entry`, `runtime ∈ {binary,node,deno,python3,bun,custom}`, `platforms`, `protocol ∈ {jsonrpc-stdio}`, `restart`, `env`, `cwd`) on `Provider`. Validator rejects host plugins on iOS; installer rejects bundles with missing or non-executable `entry`.
- **`contributes.languageServers`** — new contribution point. The gateway exposes `GET /api/plugins/lsp/{plugin}/{language}/proxy` (WebSocket) that tunnels the editor's LSP traffic to the sidecar's JSON-RPC stream. Server-side dedupe: one LSP per language, plugins tried in install order (first to reply to `initialize` wins).
- **Capability gate additions** — already-implemented matchers (`MatchExecGlobs`, `MatchHTTPURL`, `MatchFSPath`) gain a thin wrapper that expands base variables before match. New matcher `MatchSecretNamespace(plugin, key string)` enforces that `key` never contains `/` or `..` and is scoped to the plugin implicitly by SQL row ownership.
- **Consent UI additions** — per-capability granular toggles in the Flutter Settings → Plugins page (already live from M2 T21). New toggles: per-fs-path-glob (on/off per declared entry), per-exec-pattern, per-http-URL-glob. Toggling flips `perms_json` through a new endpoint `PATCH /api/plugins/{name}/consents` that accepts a partial `PermissionsV1` patch; `bridgeMgr.InvalidateConsent` fires per touched cap so active streams terminate under the 200 ms SLO established in M2.
- **DB migrations 014–017** — see §7.
- **Reference plugin** — `plugins/examples/fs-readme/` (minimal host-form plugin: reads `${workspace}/README.md`, summarises via a tiny sidecar Node script, returns a preview via `opendray.workbench.showMessage`). Exercises fs.read + exec (sidecar spawn) + host-form manifest. `rust-analyzer-od` is a stretch goal — the acceptance bar is the smaller `fs-readme` plugin.

**OUT — enumerated DEFERRED to keep M3 on-budget:**

| Tempting creep | Deferred to |
|---|---|
| Marketplace client, `plugin/market/`, `opendray plugin publish`, signature verification, revocation list polling | **M4** |
| Declarative widget tree renderer for `render:"declarative"` views | **M5** |
| Node.js / Deno / Python runtime installers — M3 assumes the runtime binary is already on PATH and fails clean (`EUNAVAIL` with human message) if not | **M6 / post-v1** |
| Hot reload for plugin authors (`opendray plugin dev`), bridge trace tooling, portable `opendray-dev` | **M6** |
| Multi-process plugin (more than one sidecar per plugin) | **post-v1** |
| Full seccomp filter set — M3 ships minimum viable: Linux `unshare(CLONE_NEWNET)` network-namespace isolation where `CAP_SYS_ADMIN` is available (usually not — document gap) and `Setpgid=true` for clean termination. Anything fancier is post-v1. | **post-v1** |
| `opendray.session.*`, `opendray.commands.execute` cross-plugin, `opendray.tasks.*`, `opendray.clipboard.*`, `opendray.llm.*`, `opendray.git.*`, `opendray.telegram.*`, `opendray.logger.*` | **M5** (gated by existing HTTP APIs; wire-up is wiring-only) |
| Sidecar-initiated bridge calls (sidecar → host `workbench.showMessage`) | **M3 sub-scope** — covered by bidirectional JSON-RPC in T12, but explicit out-of-scope for `fs.*`/`exec.*`/`http.*`/`secret.*` calls from sidecars (sidecars already have host-level access to those resources; they don't need the bridge gate). |
| `WKProcessPool` / Android data-dir suffix hardening for the bridge WS origin (already shipped M2; M3 keeps the same policy) | — |
| Wire-format bump — M3 stays on `V=1` | — |
| CSP golden-file test (M2 T25 carry-on) — ships inside T27 | — |
| Desktop inline WebView (M2 T16b carry-on) | **M6** |

---

## 2. Task graph

> **Convention:** every task has id T#, depends-on list, files to create / modify (absolute paths), core types / signatures, acceptance criteria, tests, complexity S/M/L, risks. File paths under `/home/linivek/workspace/opendray/` unless otherwise stated.

### T1 — Activate `HostV1` on `Provider` and validator
- **Depends on:** none
- **Modify:** `/home/linivek/workspace/opendray/plugin/manifest.go` — add `Host *HostV1 \`json:"host,omitempty"\`` to `Provider`; add `HostV1` struct mirroring the `02-manifest.md` §host schema: `Entry string`, `Runtime string` (enum `binary|node|deno|python3|bun|custom`, default `binary`), `Platforms map[string]string` (keys `^(linux|darwin|windows)-(x64|arm64)$`), `Protocol string` (enum `jsonrpc-stdio`), `Restart string` (enum `on-failure|always|never`, default `on-failure`), `Env map[string]string`, `Cwd string`, `IdleShutdownMinutes int` (default 10). `/home/linivek/workspace/opendray/plugin/manifest_validate.go` — add `validateHostV1(p Provider)` called from `validateContributes` when `p.EffectiveForm() == FormHost`: entry required + no `..`, runtime in enum, protocol must be `jsonrpc-stdio`, restart in enum, platform keys match the regex, env keys regex `^[A-Z_][A-Z0-9_]*$`. iOS build-tag: `validateHostV1` returns an error unconditionally on iOS (see T2 for the build-tag plumbing).
- **Acceptance:** `go build ./...` clean. Every existing manifest continues to parse. A hand-crafted host-form manifest with `runtime:"node"`, `entry:"sidecar/index.js"` parses + validates on desktop. The same manifest fails validation on an `//go:build ios` build with error `contributes.host: host-form plugins are not supported on iOS`.
- **Tests:** extend `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go` → `TestLoadManifest_V1Host` golden file; extend `/home/linivek/workspace/opendray/plugin/manifest_validate_test.go` → 8 invalid cases + 2 valid.
- **Complexity:** S
- **Risk/Mitigation:** Low. Additive + validator. Risk: forgetting iOS build tag — mitigation: explicit `ios_test.go` build-tag-gated test.

### T2 — iOS host-form refusal build-tag
- **Depends on:** T1
- **Create:** `/home/linivek/workspace/opendray/plugin/host_os_ios.go` (`//go:build ios`) — exports const `HostFormAllowed = false`; `/home/linivek/workspace/opendray/plugin/host_os_other.go` (`//go:build !ios`) — `HostFormAllowed = true`.
- **Modify:** `/home/linivek/workspace/opendray/plugin/manifest_validate.go` — `validateHostV1` short-circuits to error when `!HostFormAllowed`. `/home/linivek/workspace/opendray/plugin/install/install.go` — `Installer.Stage` rejects bundles with `form:"host"` when `!HostFormAllowed`, returning a new sentinel `ErrHostFormNotSupported`.
- **Acceptance:** building with `GOOS=ios` (or the existing iOS build path) makes any host-form install fail with the sentinel. Desktop/Android keep working.
- **Tests:** `/home/linivek/workspace/opendray/plugin/install/install_ios_test.go` (build tag `//go:build ios`) asserting Stage rejects; `/home/linivek/workspace/opendray/plugin/install/install_desktop_test.go` (build tag `//go:build !ios`) asserting Stage accepts.
- **Complexity:** S
- **Risk/Mitigation:** Low. Isolated to manifest + installer paths.

### T3 — Capability gate base-variable expansion
- **Depends on:** none (parallel with T1/T2)
- **Modify:** `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go` — add a new pure function `ExpandPathVars(pattern string, ctx PathVarCtx) string` where `PathVarCtx = struct{ Workspace, Home, DataDir, Tmp string }`. Substitutes `${workspace}`, `${home}`, `${dataDir}`, `${tmp}` in a declared pattern before `MatchFSPath` runs. `Gate.Check` gains `WithPathVars(ctx PathVarCtx) *Gate` functional option (or a new `CheckExpanded(ctx, plugin, need, vars PathVarCtx)` method — pick the latter to avoid mutating the immutable Gate per preferences.md).
- **Acceptance:** `Gate.CheckExpanded(ctx, "fs-readme", Need{Cap:"fs.read", Target:"/home/kev/proj/README.md"}, PathVarCtx{Workspace:"/home/kev/proj", Home:"/home/kev"})` returns `nil` when `perms.fs.read = ["${workspace}/**"]`. Same call with `Target:"/etc/passwd"` returns `PermError`. Path-traversal tricks (e.g. `${workspace}/../../etc/passwd`) fail: `MatchFSPath` already canonicalises via `filepath.Clean`; T3 adds a `TestExpandPathVars_TraversalBlocked` regression.
- **Tests:** `/home/linivek/workspace/opendray/plugin/bridge/capabilities_expand_test.go` — 12 cases (clean, traversal, unknown var, empty var, nested `${}`).
- **Complexity:** S
- **Risk/Mitigation:** Medium — unknown `${var}` is currently ambiguous. Decision: `ExpandPathVars` leaves unknown vars literal (`${unknown}` stays as-is, `MatchFSPath` then fails to match — safe default). Cite 05-capabilities.md §Path patterns.

### T4 — DB migration 014: plugin_secret_kek
- **Depends on:** none
- **Create:** `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek.sql`, `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek_down.sql`
- **Schema (014 up):**
  ```sql
  CREATE TABLE IF NOT EXISTS plugin_secret_kek (
      plugin_name  TEXT PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
      wrapped_dek  BYTEA NOT NULL,           -- 32-byte DEK encrypted under the host KEK
      kek_kid      TEXT NOT NULL,            -- KEK key-id for rotation
      created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  ```
- **Acceptance:** `embedded-postgres` test harness applies the migration cleanly. Down migration drops the table.
- **Tests:** `/home/linivek/workspace/opendray/kernel/store/migrations_test.go` gets an additional case in the existing migration round-trip test.
- **Complexity:** S
- **Risk/Mitigation:** Low.

### T5 — DB migration 015: plugin_host_state
- **Depends on:** none (parallel with T4)
- **Create:** `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state.sql`, `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state_down.sql`
- **Schema:**
  ```sql
  CREATE TABLE IF NOT EXISTS plugin_host_state (
      plugin_name     TEXT PRIMARY KEY REFERENCES plugins(name) ON DELETE CASCADE,
      last_started_at TIMESTAMPTZ,
      last_exit_code  INTEGER,
      restart_count   INTEGER NOT NULL DEFAULT 0,
      last_error      TEXT
  );
  ```
- **Acceptance:** migration round-trip. Supervisor persists lifecycle stats here so Settings UI can show "restarted 3× in last hour".
- **Complexity:** S

### T6 — DB migration 016: plugin_secret encryption upgrade
- **Depends on:** T4
- **Modify:** none (the existing `012_plugin_secret.sql` already has `ciphertext BYTEA`; M3 just starts using it correctly). `/home/linivek/workspace/opendray/kernel/store/migrations/016_plugin_secret_nonce.sql` — ALTER TABLE adds `nonce BYTEA NOT NULL DEFAULT ''::bytea` column (AES-GCM nonce, 12 bytes). Down migration drops the column.
- **Acceptance:** old secret rows (none today — M1 scaffolded, M2 unused) re-key on first read via a one-shot migrator in `kernel/store/plugin_secret.go` (T15). Empty-default `nonce` identifies pre-M3 rows.
- **Complexity:** S

### T7 — Secret crypto primitive: KEK / DEK helpers
- **Depends on:** T4
- **Create:** `/home/linivek/workspace/opendray/kernel/auth/secret_kek.go`, `/home/linivek/workspace/opendray/kernel/auth/secret_kek_test.go`
- **Core types/functions:**
  ```go
  // KEKProvider derives the host KEK on every call. Keeping it derivation-based
  // means the KEK is never persisted — only the bcrypt admin hash it's derived
  // from, which already lives in admin_auth.
  type KEKProvider interface {
      DeriveKEK(ctx context.Context, kid string) ([]byte, error) // 32 bytes
  }

  // NewKEKProviderFromAdminAuth wires a provider that reads the admin_auth row
  // and derives a 32-byte KEK via HKDF-SHA256 with info="opendray-plugin-kek/<kid>".
  func NewKEKProviderFromAdminAuth(store *CredentialStore) KEKProvider

  // WrapDEK encrypts the 32-byte DEK under kek using AES-256-GCM. Returns the
  // concatenation nonce||ciphertext (nonce is 12 bytes). Panics if dek is not
  // 32 bytes or kek is not 32 bytes.
  func WrapDEK(kek, dek []byte) (wrapped []byte, err error)

  // UnwrapDEK is the inverse. Returns the 32-byte DEK or an error.
  func UnwrapDEK(kek, wrapped []byte) (dek []byte, err error)
  ```
- **Acceptance:** round-trip `Wrap`/`Unwrap` for 1000 random DEKs, all succeed. Tampering with any ciphertext byte fails the GCM tag. KEK rotation: unwrap with old `kid`, rewrap with new `kid` — a dedicated `TestKEKRotation_RewrapSucceeds` asserts.
- **Tests:** table-driven, includes a negative test for wrong-sized keys + fuzz corpus for the wrap format.
- **Complexity:** M
- **Risk/Mitigation:** **High** — key leak is an irrecoverable event. Mitigations: (a) KEK material never leaves `crypto/subtle`-comparable byte slices; (b) `secret_kek.go` has no log lines that include any key material; (c) `gosec ./...` runs in CI for this package with zero HIGH findings as a merge gate; (d) document in 10-security.md that "rotating the admin password rotates the KEK — host walks every `plugin_secret_kek` row on login and rewraps with the new KEK".

### T8 — Bridge Namespace signature additions for streams
- **Depends on:** none (parallel)
- **Modify:** `/home/linivek/workspace/opendray/gateway/plugins_bridge.go` — no changes to the `Namespace` interface itself (it already takes `envID` + `*bridge.Conn` since T20b). Document stream-frame contract: namespaces emit chunks/ends via `conn.WriteEnvelope(bridge.NewStreamChunk(envID, data))` and `conn.WriteEnvelope(bridge.NewStreamEnd(envID))`. Add a helper `bridge.NewStreamChunkErr(id string, we *WireError)` to `/home/linivek/workspace/opendray/plugin/bridge/protocol.go` for terminal-error-in-stream cases (exec non-zero exit mid-stream, http stream truncated, fs.watch error).
- **Acceptance:** existing tests continue to pass; new helper round-trips via JSON.
- **Tests:** extend `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go` with `TestNewStreamChunkErr_EnvelopeShape`.
- **Complexity:** S

### T9 — `opendray.fs.*` namespace (read-path)
- **Depends on:** T3, T8
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/api_fs.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_fs_test.go`
- **Core types/functions:**
  ```go
  // FSConfig wires the dependencies an FS namespace needs: a Gate for
  // capability checks, a PathVarResolver for per-call session context
  // (workspace root etc.), and a Clock for test-injectable watch debouncing.
  type FSConfig struct {
      Gate     *Gate
      Resolver PathVarResolver
      Log      *slog.Logger
  }

  // PathVarResolver returns the current {workspace, home, dataDir, tmp} for a
  // given plugin at call time. Implemented by gateway — resolves workspace
  // from the active session, dataDir from ${PluginsDataDir}/<name>/<version>/data/.
  type PathVarResolver interface {
      Resolve(ctx context.Context, plugin string) (PathVarCtx, error)
  }

  type FSAPI struct { /* unexported */ }
  func NewFSAPI(cfg FSConfig) *FSAPI
  // Dispatch implements gateway.Namespace.
  func (a *FSAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *Conn) (any, error)
  // Methods implemented here (read-path only):
  //   readFile(path, opts?{encoding}) → string (utf8) | base64 string
  //   exists(path) → bool
  //   stat(path) → {size, mtime, isDir}
  //   readDir(path) → [{name, isDir}]
  ```
- **Capability enforcement:** every method resolves path vars, cleans the path, then calls `Gate.Check(ctx, plugin, Need{Cap:"fs.read", Target: cleanedAbsolute})`. Read cap = 10 MiB hard cap; `readDir` cap = 4096 entries.
- **Acceptance:** plugin with `permissions.fs.read=["${workspace}/**"]` can `readFile("${workspace}/README.md")` (server expands var before match). Reading `/etc/passwd` → `EPERM`. Reading a file > 10 MiB → `EINVAL {message:"fs.readFile: file exceeds 10 MiB cap"}`.
- **Tests:** table-driven, includes traversal attempts from M2 T8's attack list plus unicode normalisation ones (`README\u202e.md`). Use `t.TempDir()` as synthetic workspace.
- **Complexity:** M
- **Risk/Mitigation:** **High — TOCTOU**. Mitigation: every call re-canonicalises the path through `filepath.EvalSymlinks` before the final `Open`; the resulting resolved path is re-checked against the grant globs. Symlinks escaping the workspace fail the second check.

### T10 — `opendray.fs.*` namespace (write-path + watch)
- **Depends on:** T9
- **Modify:** `/home/linivek/workspace/opendray/plugin/bridge/api_fs.go` — add methods `writeFile`, `mkdir`, `remove`, `watch`. `watch` is stream-capable: spawns an `fsnotify.Watcher`, registers a subscription via `conn.Subscribe(envID, "fs.watch")`, pushes `{create|modify|delete, path}` chunks. Unsubscribe via `fs.unwatch{subId}` or revoke.
- **Dependency:** add `github.com/fsnotify/fsnotify v1.8.0` to `go.mod` (pinned; already in module cache from task-runner plugin if present — verify).
- **Capability:** `fs.write`; `watch` requires `fs.read` (subscription sees events for files it could have read).
- **Acceptance:** writeFile creates the file with 0644 mode (override via `opts.mode`); `remove{recursive:true}` refuses paths outside the grant; `watch` delivers 3 events within 500 ms of external file changes in a `t.TempDir`.
- **Tests:** plus `TestFSWatch_DisposesOnRevoke` (M2 T11 pattern).
- **Complexity:** L
- **Risk/Mitigation:** High — fsnotify has platform quirks (Linux inotify limits). Mitigation: per-plugin max 16 active watches; exceed → `EINVAL`. Fail fast on inotify-limit-reached with a clear message pointing at `sysctl fs.inotify.max_user_watches`.

### T11 — `opendray.exec.*` namespace
- **Depends on:** T3, T8
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/api_exec.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_exec_test.go`
- **Core types/functions:**
  ```go
  type ExecConfig struct {
      Gate      *Gate
      Resolver  PathVarResolver
      Log       *slog.Logger
      MaxTimeout time.Duration // default 5 min; per-call opts.timeoutMs clamped
  }
  type ExecAPI struct { /* unexported */ }
  func NewExecAPI(cfg ExecConfig) *ExecAPI
  // Methods: run(cmd, args, opts?) → {exitCode, stdout, stderr, timedOut}
  //          spawn(cmd, args, opts?) → {pid, subId}  + stream chunks
  //          write(subId, input)     → null
  //          kill(subId, signal?)    → null
  //          wait(subId)             → {exitCode}
  ```
- **Capability enforcement:** on every `run`/`spawn`, compute `cmdline = cmd + " " + strings.Join(args, " ")`, call `Gate.Check(ctx, plugin, Need{Cap:"exec", Target: cmdline})`. Reject `opts.cwd` if outside the plugin's declared fs.read/fs.write grants (configurable via `ExecConfig.AllowCwdOutsideFS`, default false).
- **Process attributes:** `Setpgid=true`. Linux additionally calls `syscall.Unshare(syscall.CLONE_NEWNET)` **only when** `ExecConfig.IsolateNetNS=true` AND `cap_sys_admin` is held by opendray — otherwise logs a one-time warning at supervisor start and skips. Document gap in 10-security.md.
- **Streaming cap:** each chunk ≤16 KiB; output truncation follows existing `commands.Dispatcher` semantics but surfaces mid-stream `{error:{code:"EINVAL",message:"output truncated"}}` envelopes as a soft terminator instead of silent truncation.
- **Acceptance:** `run("git",["status","--short"])` from a plugin with `permissions.exec=["git *"]` returns a result with exit code and captured stdout. `run("rm",["-rf","/"])` with the same grant returns `EPERM`. `spawn` with 1 MiB of stdout is streamed in ≤64 chunks, no buffering explosion.
- **Tests:** 14 cases incl. zero-exit success, non-zero exit success (bubbles through `Result.Exit`), timeout kill, kill(SIGTERM) → SIGKILL escalation after 5 s, cwd outside grant.
- **Complexity:** L
- **Risk/Mitigation:** **High — fork bomb / resource exhaustion.** Mitigation: per-plugin hard cap of 4 concurrent spawns (queue beyond that → `ETIMEOUT` with retryAfter); Linux cgroup v2 `memory.max`=512 MiB, `cpu.max`=50% if writable (log once per host boot if not). Document that macOS/Windows have no cgroup equivalent and rely on OS ulimit.

### T12 — `opendray.http.*` namespace
- **Depends on:** T3, T8
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/api_http.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_http_test.go`
- **Core types/functions:**
  ```go
  type HTTPConfig struct {
      Gate            *Gate
      Log             *slog.Logger
      MaxRequestBody  int64 // default 4 MiB
      MaxResponseBody int64 // default 16 MiB
      MaxRedirects    int   // default 5
      DialTimeout     time.Duration // default 10 s
      TotalTimeout    time.Duration // default 60 s
      TLSMinVersion   uint16 // default tls.VersionTLS12
  }
  type HTTPAPI struct { /* unexported */ }
  func NewHTTPAPI(cfg HTTPConfig) *HTTPAPI
  // Methods: request(req) → {status, headers, body(base64)}
  //          stream(req)  → chunk envelopes of base64 body slices; end when closed
  ```
- **Capability enforcement:** `Gate.Check(ctx, plugin, Need{Cap:"http", Target: req.URL})`. Existing `MatchHTTPURL` already denies RFC1918 / loopback / link-local. Redirect hop: re-check every hop (not just the initial URL) — new helper `followRedirectsWithGate` in the file; a hop that fails `Gate.Check` breaks the chain with `EPERM` + message naming the hop's URL.
- **Body caps:** request body rejected if `len(body) > MaxRequestBody`. Response body truncated at `MaxResponseBody` with a trailing `{error:{code:"EINVAL",message:"response body truncated"}}` envelope on the stream path (non-stream path returns the truncated body + the same code in `result.truncated=true`).
- **Acceptance:** a plugin with `permissions.http=["https://api.github.com/*"]` can `request({url:"https://api.github.com/repos/opendray/opendray"})`. A request to `https://169.254.169.254` (AWS IMDS) is denied even if somehow pattern-matched (matcher already blocks). A redirect from `api.github.com` to `malicious.com` is blocked with EPERM at the hop.
- **Tests:** includes a `httptest.Server` chain that returns redirects and asserts the gate blocks mid-chain.
- **Complexity:** L
- **Risk/Mitigation:** **High — SSRF via DNS rebinding.** Mitigation: dial uses a custom `net.Dialer.Control` that re-checks the resolved IP against `isPrivateHost` just before `connect()` — blocks both DNS rebind AND bypass by specifying an IP literal that resolves to private. Dedicated test `TestHTTP_SSRF_DNSRebind` uses a custom resolver that returns `10.0.0.1` on the second lookup.

### T13 — `opendray.secret.*` namespace
- **Depends on:** T6, T7
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/api_secret.go`, `/home/linivek/workspace/opendray/plugin/bridge/api_secret_test.go`; `/home/linivek/workspace/opendray/kernel/store/plugin_secret.go`, `/home/linivek/workspace/opendray/kernel/store/plugin_secret_test.go`
- **Core types/functions:**
  ```go
  // Store layer.
  func (d *DB) SecretGet(ctx, plugin, key string) (string, bool, error)
  func (d *DB) SecretSet(ctx, plugin, key, value string, wrap func([]byte) ([]byte, []byte, error)) error
  // wrap returns (ciphertext, nonce, err); DB layer stays crypto-agnostic.
  func (d *DB) SecretDelete(ctx, plugin, key string) error
  func (d *DB) SecretList(ctx, plugin string) ([]string, error) // keys only — never values

  // DEK management.
  func (d *DB) EnsureKEKRow(ctx, plugin string, wrappedDEK []byte, kid string) error
  func (d *DB) GetWrappedDEK(ctx, plugin string) (wrapped []byte, kid string, err error)

  // Bridge layer.
  type SecretAPI struct{ /* unexported */ }
  type SecretConfig struct {
      DB  *store.DB
      Gate *Gate
      KEK  auth.KEKProvider
  }
  func NewSecretAPI(cfg SecretConfig) *SecretAPI
  ```
- **Lifecycle:** first `SecretSet` for a plugin generates a random 32-byte DEK, wraps via `KEKProvider.DeriveKEK` → `WrapDEK`, `EnsureKEKRow`. Subsequent sets reuse the same DEK. Uninstall cascades via FK → both `plugin_secret_kek` and `plugin_secret` rows drop.
- **Capability:** `secret:true`. Key validation: `MatchSecretNamespace(key)` — regex `^[a-zA-Z0-9._-]{1,128}$`, no `/`, no `..`.
- **Acceptance:** round-trip of 1 KiB secret. `get` of missing key returns `null` (result), not an error. Another plugin's `get` of the same key returns `null` (secrets are namespaced by plugin row — one plugin cannot see another's).
- **Tests:** the log-redaction rule from 10-security.md is enforced with a `TestSecret_NeverLogged` that captures slog output during set/get and asserts the secret value is absent.
- **Complexity:** L
- **Risk/Mitigation:** **Critical.** See §6 threat table. All crypto-touching code runs through `gosec ./...` with zero HIGH findings as a merge gate.

### T14 — Host sidecar supervisor skeleton
- **Depends on:** T1, T2, T5
- **Create:** `/home/linivek/workspace/opendray/plugin/host/supervisor.go`, `/home/linivek/workspace/opendray/plugin/host/supervisor_test.go`, `/home/linivek/workspace/opendray/plugin/host/process.go`, `/home/linivek/workspace/opendray/plugin/host/process_unix.go` (`//go:build unix`), `/home/linivek/workspace/opendray/plugin/host/process_windows.go` (`//go:build windows`)
- **Core types/functions:**
  ```go
  // Supervisor owns the lifecycle of every host-form plugin's sidecar.
  type Supervisor struct { /* unexported */ }
  type Config struct {
      DataDir           string              // ${PluginsDataDir}
      Runtime           *plugin.Runtime     // to resolve Host spec
      State             *store.DB           // plugin_host_state writer
      Log               *slog.Logger
      IdleShutdown      time.Duration       // default 10 min
      MaxRestartBackoff time.Duration       // default 5 s
      InitialBackoff    time.Duration       // default 200 ms
  }
  func NewSupervisor(cfg Config) *Supervisor
  // Ensure starts the sidecar if not running, otherwise touches lastUsedAt.
  // Returns a *Sidecar handle. Concurrent callers share one sidecar per plugin.
  func (s *Supervisor) Ensure(ctx context.Context, plugin string) (*Sidecar, error)
  // Kill terminates the sidecar + cancels any in-flight requests.
  func (s *Supervisor) Kill(plugin string, reason string) error
  // Stop halts all sidecars (graceful). Called on host shutdown.
  func (s *Supervisor) Stop(ctx context.Context) error

  // Sidecar exposes the JSON-RPC write + read channels.
  type Sidecar struct { /* unexported */ }
  func (s *Sidecar) Call(ctx context.Context, method string, params any) (json.RawMessage, error)
  func (s *Sidecar) Notify(method string, params any) error
  // Subscribe returns a channel of server-pushed notifications (LSP $/progress etc.)
  func (s *Sidecar) Subscribe() <-chan Notification
  ```
- **Process spawning:** looks up `Provider.Host` (T1); for `runtime="node"` execs `node <entry>`; for `binary` uses `Host.Platforms[runtime.GOOS+"-"+runtime.GOARCH]`; `Setpgid=true`; pipes stdin/stdout/stderr. Stderr drains to a ring buffer (last 64 KiB) surfaced via a future settings → logs UI.
- **Backoff:** on crash, wait `min(initial*2^crashes, max)`; reset counter after 5 minutes of stable uptime. `restart:"never"` disables restart; `restart:"always"` ignores exit code.
- **Idle shutdown:** ticker checks `time.Since(lastUsedAt)`; if exceeds config, sends a `shutdown` JSON-RPC notification, waits 2 s, then `SIGTERM`, then 5 s later `SIGKILL`.
- **Acceptance:** `Ensure` on a fixture node script starts the process, responds to `ping` JSON-RPC with `pong` in < 200 ms, idles after the configured timeout, respawns after a simulated crash within backoff window.
- **Tests:** the fixture is a 40-line `sidecar.js` that loops on stdin, replies to `{method:"ping"}`. Windows path uses `node.exe` detection via `exec.LookPath`.
- **Complexity:** L
- **Risk/Mitigation:** **High — zombie processes, deadlocks on stdin/stdout close.** Mitigation: every goroutine is tied to a single `context.Context`; sidecar shutdown fans out through that context with a `sync.WaitGroup` join and a 10 s hard deadline. `-race` test `TestSupervisor_KillUnderLoad` spawns 50 concurrent `Ensure` calls + `Kill` in a loop, 100 iterations.

### T15 — JSON-RPC LSP framing codec
- **Depends on:** none (parallel with T14)
- **Create:** `/home/linivek/workspace/opendray/plugin/host/jsonrpc.go`, `/home/linivek/workspace/opendray/plugin/host/jsonrpc_test.go`
- **Core types/functions:**
  ```go
  // FramedReader reads one LSP-framed JSON-RPC message per call. Framing is:
  //   Content-Length: N\r\n
  //   (optional other headers, ignored)
  //   \r\n
  //   <N bytes of JSON>
  type FramedReader struct { /* unexported */ }
  func NewFramedReader(r io.Reader) *FramedReader
  func (r *FramedReader) Read() (json.RawMessage, error)

  type FramedWriter struct { /* unexported */ }
  func NewFramedWriter(w io.Writer) *FramedWriter
  // Write encodes msg as JSON, writes Content-Length header, then payload.
  // Safe for concurrent callers via an internal sync.Mutex.
  func (w *FramedWriter) Write(msg any) error

  // RPC is the minimal JSON-RPC 2.0 wire shape.
  type RPC struct {
      JSONRPC string          `json:"jsonrpc"` // always "2.0"
      ID      json.RawMessage `json:"id,omitempty"`
      Method  string          `json:"method,omitempty"`
      Params  json.RawMessage `json:"params,omitempty"`
      Result  json.RawMessage `json:"result,omitempty"`
      Error   *RPCError       `json:"error,omitempty"`
  }
  type RPCError struct {
      Code    int             `json:"code"`
      Message string          `json:"message"`
      Data    json.RawMessage `json:"data,omitempty"`
  }
  ```
- **Acceptance:** round-trip encode / decode for every example in the LSP 3.17 spec's §Base Protocol. Malformed Content-Length header (negative, > 16 MiB, missing \r\n\r\n terminator) returns a named error; reader recovers by skipping to the next valid header.
- **Tests:** fuzz `FramedReader` with 10 k random byte sequences; no panics, no infinite loops (bounded at 8 KiB headers + 16 MiB body).
- **Complexity:** M
- **Risk/Mitigation:** Medium — malformed frames from a crashing sidecar must not DOS the host. Mitigation: hard cap body size at 16 MiB, return EINVAL on overflow.

### T16 — Sidecar bidirectional call multiplexer
- **Depends on:** T14, T15
- **Create:** `/home/linivek/workspace/opendray/plugin/host/mux.go`, `/home/linivek/workspace/opendray/plugin/host/mux_test.go`
- **Core types/functions:**
  ```go
  // Mux owns the inbound/outbound demultiplexing on a single sidecar's
  // JSON-RPC stream. Outbound requests get an id, inbound responses resolve
  // the matching pending call. Inbound requests (sidecar → host) are delivered
  // to a handler. Inbound notifications go on Subscribe().
  type Mux struct { /* unexported */ }
  func NewMux(r io.Reader, w io.Writer, handler RPCHandler, log *slog.Logger) *Mux
  func (m *Mux) Start(ctx context.Context)
  func (m *Mux) Call(ctx context.Context, method string, params any) (json.RawMessage, error)
  func (m *Mux) Notify(method string, params any) error
  func (m *Mux) Notifications() <-chan Notification

  // RPCHandler resolves sidecar → host RPC calls. For M3, this is limited to
  // a few well-known methods (workbench/showMessage, fs/readFile); anything
  // else returns MethodNotFound.
  type RPCHandler interface {
      Handle(ctx context.Context, method string, params json.RawMessage) (any, error)
  }
  ```
- **Id generation:** monotonic int64 starting at 1; wraps at math.MaxInt32 after 5 years of 1 req/s — practically never.
- **Acceptance:** 1000 concurrent `Call` on a sidecar loopback that echoes `params` as `result` all resolve correctly under `-race`. A sidecar → host `workbench.showMessage` call routes through `RPCHandler.Handle` and the response returns to the sidecar.
- **Tests:** Paired with a stub sidecar that runs inside the test process (`io.Pipe`).
- **Complexity:** L
- **Risk/Mitigation:** High — id reuse after wraparound corrupts call routing. Mitigation: detect map collision (id already in pending) and fail the older call with `EINTERNAL`.

### T17 — Supervisor ↔ namespaces wiring (sidecar-backed capabilities)
- **Depends on:** T14, T16; decoupled from T9/T11/T12 by design
- **Create:** `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler.go`, `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler_test.go`
- **Core types/functions:**
  ```go
  // HostRPCHandler is the RPCHandler wired into every Mux. It routes sidecar
  // → host calls through the same bridge.Namespace surface that webview
  // plugins use, so capability enforcement is a single code path.
  type HostRPCHandler struct { /* unexported */ }
  type HostRPCConfig struct {
      Namespaces map[string]bridge.Dispatcher // "fs", "exec", "http", "secret", "workbench", "storage", "events"
      Gate       *bridge.Gate
      Log        *slog.Logger
      Plugin     string // the owning plugin name; immutable per Mux
  }
  func NewHostRPCHandler(cfg HostRPCConfig) *HostRPCHandler
  // Handle implements RPCHandler; maps "<ns>/<method>" strings to the right
  // dispatcher, threading Plugin from the constructor so sidecars cannot
  // impersonate other plugins.
  func (h *HostRPCHandler) Handle(ctx context.Context, method string, params json.RawMessage) (any, error)
  ```
- **Contract:** sidecar calls take the form `{"method":"fs/readFile","params":["/abs/path"]}`. The `/` in `method` separates namespace from method; both pass through the same `Dispatcher.Dispatch` webview plugins use. Capability checks run identically.
- **Acceptance:** a host-form fixture sidecar that sends `fs/readFile` of a granted path gets the bytes back; sending `fs/readFile` of an ungranted path gets a JSON-RPC error with `code=-32001` and message "EPERM: ...".
- **Tests:** table-driven across every namespace stub.
- **Complexity:** M
- **Risk/Mitigation:** Medium — method-name injection. Mitigation: reject `method` containing more than one `/` or any of `..`, `\x00`. Test `TestHostRPC_MethodInjectionRejected`.

### T18 — `contributes.languageServers` contribution + Flatten
- **Depends on:** T1
- **Modify:** `/home/linivek/workspace/opendray/plugin/manifest.go` — add `LanguageServers []LanguageServerV1 \`json:"languageServers,omitempty"\`` to `ContributesV1`; add `LanguageServerV1 struct { ID string; Languages []string; Transport string; InitializationOptions json.RawMessage }`. `/home/linivek/workspace/opendray/plugin/manifest_validate.go` — validate `transport="stdio"` (enum), languages non-empty, id regex. `/home/linivek/workspace/opendray/plugin/contributions/registry.go` — extend `FlatContributions` with `LanguageServers []OwnedLanguageServer`; add `func (r *Registry) LookupLanguageServer(language string) (OwnedLanguageServer, bool)` (first-match by install order).
- **Acceptance:** registering a plugin with `contributes.languageServers=[{id:"rust",languages:["rust"],transport:"stdio"}]` makes `LookupLanguageServer("rust")` return it. A second plugin declaring `languages:["rust"]` is skipped by Lookup.
- **Tests:** extend registry tests with install-order determinism.
- **Complexity:** S

### T19 — LSP proxy gateway route
- **Depends on:** T14, T16, T17, T18
- **Create:** `/home/linivek/workspace/opendray/gateway/plugins_lsp.go`, `/home/linivek/workspace/opendray/gateway/plugins_lsp_test.go`
- **Modify:** `/home/linivek/workspace/opendray/gateway/server.go` — register `r.Get("/api/plugins/lsp/{language}/proxy", s.pluginsLSPProxy)` in the protected group.
- **Core types/functions:**
  ```go
  // pluginsLSPProxy handles WebSocket upgrades where the editor tunnels raw
  // LSP JSON-RPC traffic. The proxy:
  //   1. Looks up which plugin owns the language via
  //      contributions.Registry.LookupLanguageServer.
  //   2. Asks Supervisor.Ensure to start the sidecar if not running.
  //   3. Copies frames in both directions until either side closes.
  // Capability check: this route requires a new built-in cap "lsp" granted
  // implicitly to any plugin with contributes.languageServers. No user consent
  // beyond the install consent screen (matches contributes.commands' story).
  func (s *Server) pluginsLSPProxy(w http.ResponseWriter, r *http.Request)
  ```
- **Framing translation:** WS → LSP: gateway reads WS binary messages, expects them pre-framed (editor sends Content-Length headers as-is). LSP → WS: reads FramedReader output and wraps each message as one WS message. Maximum message size matches bridge WS: 1 MiB per frame; oversize triggers graceful close with code 1009.
- **Acceptance:** `wscat` connects, sends an `initialize` request, gets `initialize` response from the sidecar, then a `textDocument/didOpen` notification survives round-trip.
- **Tests:** integration harness boots a stub sidecar that implements `initialize` → empty-capabilities response.
- **Complexity:** L
- **Risk/Mitigation:** Medium — proxy must not buffer entire LSP sessions. Mitigation: true streaming via two goroutines + `io.Copy`; bounded chunk reads; explicit close propagation.

### T20 — Consent patch endpoint + granular UI plumbing
- **Depends on:** M2 T12 (`UpdateConsentPerms` already exists)
- **Create:** `/home/linivek/workspace/opendray/gateway/plugins_consents_patch.go`, `/home/linivek/workspace/opendray/gateway/plugins_consents_patch_test.go`
- **Modify:** `/home/linivek/workspace/opendray/gateway/plugins_consents.go` — add route `PATCH /api/plugins/{name}/consents` that accepts a partial `PermissionsV1`-shaped body; merges into the stored `perms_json`; calls `bridgeMgr.InvalidateConsent` per **diff'd** capability (not per top-level key — a toggle removing one glob from `fs.read` must invalidate `fs.read` subscriptions even though the top-level `fs` key stays).
- **Core types/functions:**
  ```go
  // ConsentPatch is the partial permissions merge payload. Nil fields are
  // left untouched; zero-length slices are explicit clears.
  type ConsentPatch struct {
      Fs     *FSPermsPatch   `json:"fs,omitempty"`
      Exec   *[]string       `json:"exec,omitempty"`  // pointer-to-slice: nil = unchanged, &[] = clear
      HTTP   *[]string       `json:"http,omitempty"`
      Secret *bool           `json:"secret,omitempty"`
      // Storage, Events, Session, Clipboard, Git, Telegram, LLM patched similarly.
  }
  type FSPermsPatch struct {
      Read  *[]string `json:"read,omitempty"`
      Write *[]string `json:"write,omitempty"`
  }
  ```
- **Acceptance:** `PATCH {fs:{read:["${workspace}/**"]}}` updates the stored perms; subsequent `fs.readFile` calls with a matching path succeed; calls outside the new glob are denied within 200 ms (M2 SLO still holds).
- **Tests:** SLO regression test mirrors M2's `TestRevoke_StorageWithin200ms` but targets a fs-glob drop.
- **Complexity:** M

### T21 — Flutter Settings UI: granular caps
- **Depends on:** T20
- **Modify:** `/home/linivek/workspace/opendray/app/lib/features/settings/plugin_consents_page.dart` — add expandable sections under each cap toggle: fs shows one row per declared glob (toggleable), exec shows one row per pattern, http shows one row per URL glob. `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart` — add `patchPluginConsents(name, ConsentPatch)`.
- **Acceptance:** toggling a single fs.read glob off (e.g. `${home}/.ssh/**`) updates perms and the next plugin call against that glob fails with EPERM shown via SnackBar. UI does not crash; toggle can be flipped back on.
- **Tests:** widget test `test/features/settings/plugin_consents_granular_test.dart` asserts toggle → API call mapping for each cap type.
- **Complexity:** M

### T22 — Flutter bridge SDK: fs/exec/http/secret TS types
- **Depends on:** T9–T13
- **Modify:** `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart` — add Dart-side fallthrough routing for the new namespaces (no new code needed beyond envelope routing; the channel already forwards unknown ns strings). Ensure the preload shim string bundled by the WebView host includes `fs`, `exec`, `http`, `secret` proxy subsets.
- **Modify:** the embedded Go asset at `gateway.OpenDrayShimJS` (constant inside `/home/linivek/workspace/opendray/gateway/plugins_assets.go` — verify location; otherwise create `plugins_shim.go`) — extend `nsProxy` calls to cover the new namespaces + `fs.watch` / `exec.spawn` stream-callback pattern.
- **Acceptance:** a webview plugin can `await opendray.fs.readFile("${workspace}/README.md")` and get the content. The shim still fits the 60-line budget (M2 was 40; M3 allows up to 60 with streaming helpers).
- **Tests:** extend `/home/linivek/workspace/opendray/app/test/features/workbench/plugin_bridge_channel_test.dart` with two mocked round-trips through each new namespace.
- **Complexity:** M

### T23 — Main wiring
- **Depends on:** T7, T9, T10, T11, T12, T13, T14, T15, T16, T17, T19, T20
- **Modify:** `/home/linivek/workspace/opendray/cmd/opendray/main.go` — after existing bridge wiring around line 365:
  1. Construct `hostSupervisor := host.NewSupervisor(host.Config{DataDir: cfg.PluginsDataDir, Runtime: providerRuntime, State: db, Log: logger})`.
  2. Register `hostSupervisor.Stop` in the shutdown hook chain (same lifecycle as `installer.Stop`).
  3. Build namespaces: `fsAPI := bridge.NewFSAPI(...)`, `execAPI := bridge.NewExecAPI(...)`, `httpAPI := bridge.NewHTTPAPI(...)`, `secretAPI := bridge.NewSecretAPI(...)`.
  4. `gw.RegisterNamespace("fs", fsAPI)`, same for exec/http/secret.
  5. Wire `hostSupervisor.SetRPCHandlerFactory(func(pluginName string) host.RPCHandler { return host.NewHostRPCHandler(host.HostRPCConfig{Namespaces: {...}, Plugin: pluginName, ...}) })`.
  6. Construct the KEK provider via `auth.NewKEKProviderFromAdminAuth(adminCreds)`.
- **Acceptance:** `./opendray` boots. `GET /api/plugins/kanban/bridge/ws` sends `{v:1,id:"1",ns:"fs",method:"readFile",args:["..."]}` → EPERM (kanban has no fs grants). `GET /api/plugins/lsp/rust/proxy` 404s when no plugin claims rust; installing a test host plugin makes it 101 upgrade.
- **Tests:** extend `/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go` with `TestBridge_M3NamespacesRegistered`.
- **Complexity:** M

### T24 — PathVarResolver implementation (gateway)
- **Depends on:** T9 (defines the interface)
- **Create:** `/home/linivek/workspace/opendray/gateway/path_vars.go`, `/home/linivek/workspace/opendray/gateway/path_vars_test.go`
- **Core types/functions:**
  ```go
  // pathVarResolver implements bridge.PathVarResolver for the live gateway.
  type pathVarResolver struct {
      plugins *plugin.Runtime
      dataDir string
      sessions sessionLookup     // minimal interface over sessionHub for workspace root
  }
  // sessionLookup returns the user's active session's cwd, or "" if no session.
  type sessionLookup interface {
      ActiveWorkspace(userID string) string
  }
  func (r *pathVarResolver) Resolve(ctx context.Context, plugin string) (bridge.PathVarCtx, error)
  ```
- **Semantics:** `workspace = sessions.ActiveWorkspace(defaultUserID)` (may be empty — plugin gets empty expansion, which fails the match as a safe default). `home = os.UserHomeDir()`. `dataDir = filepath.Join(cfg.PluginsDataDir, plugin, "<version>", "data")` and the dir is created on first resolve with mode 0700. `tmp = os.TempDir()`.
- **Acceptance:** plugin context resolution is consistent across bridge calls within one session. Workspace changes take effect on the next bridge call.
- **Tests:** table-driven.
- **Complexity:** S

### T25 — Reference plugin `plugins/examples/fs-readme/`
- **Depends on:** T1, T2, T9, T11, T14, T22
- **Create:**
  - `/home/linivek/workspace/opendray/plugins/examples/fs-readme/manifest.json` — host-form, runtime `node`, entry `sidecar.js`, contributes `commands` (one command `fs-readme.summarise`), permissions `fs.read:["${workspace}/**"]` + `exec:["node *"]`.
  - `/home/linivek/workspace/opendray/plugins/examples/fs-readme/sidecar.js` — 80-line Node script using `readline` on stdin, replying to JSON-RPC with `{result:<first 200 bytes of README>}` for method `summarise`. Demonstrates sidecar → host `fs/readFile` call to fetch the README via the capability gate (host does the I/O).
  - `/home/linivek/workspace/opendray/plugins/examples/fs-readme/README.md` — usage + what-this-proves.
- **Acceptance:** `opendray plugin install ./plugins/examples/fs-readme --yes` on a dev host with Node on PATH. Invoking the command returns the first 200 bytes of the workspace README via the command invoke HTTP endpoint. Killing the sidecar while idle respawns it within 500 ms; killing it during a request returns `EUNAVAIL` to the caller.
- **Tests:** covered by T26 E2E.
- **Complexity:** M

### T26 — E2E test: fs-readme full lifecycle
- **Depends on:** T25, T23
- **Create:** extend `/home/linivek/workspace/opendray/plugin/e2e_test.go` with `TestE2E_FSReadmeFullLifecycle` (build tag `//go:build e2e`).
- **Scenario:**
  1. Install fs-readme via local source.
  2. Assert `contributes.commands` lists the summarise command.
  3. Invoke `fs-readme.summarise` → returns first 200 bytes of the test fixture README.
  4. `DELETE /api/plugins/fs-readme/consents/fs.read` — next invoke returns EPERM within 200 ms.
  5. Re-grant via PATCH; next invoke works.
  6. Kill the sidecar via `SIGTERM` to its PID — `Supervisor` restarts it within 2 s; next invoke works.
  7. Try to install on a forced-iOS build (via env var flipping `HostFormAllowed`) — install refused.
  8. Uninstall — plugin_secret_kek row cascades, plugin_host_state row cascades.
- **Acceptance:** `go test -race -tags=e2e -timeout=10m ./plugin/...` green. Time-budget for SLO step hard-deadlined.
- **Complexity:** L
- **Risk/Mitigation:** Flaky sidecar spawn on CI (Node not on PATH). Mitigation: skip test with `t.Skip("node not on PATH")` when `exec.LookPath("node")` errors — document in sign-off checklist that CI machine must provide Node 20.

### T27 — Carry-on: CSP test + desktop webview stub + kanban E2E
- **Depends on:** —
- **Create:** `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go` (the M2 T25 deferred test — golden-file exact header match), extend `/home/linivek/workspace/opendray/plugin/e2e_test.go` with `TestE2E_KanbanFullLifecycle` (the M2 T23 deferred test — from M2-PLAN §T23 spec).
- **Modify:** `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host_desktop.dart` — land the M2 T16b fallback widget (open plugin view in a modal desktop window via `desktop_webview_window` 0.2.3). Document soft-isolation limitation in 10-security.md §Network policy.
- **Acceptance:** M2 sign-off checklist items T16b/T23/T25 flip from 🟡 to ✅.
- **Complexity:** M
- **Risk/Mitigation:** Low — these are catch-up tasks, scope is already frozen.

### T28 — Documentation
- **Depends on:** all
- **Modify:** `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md` — add threat/mitigation rows from §6 below (TOCTOU fs, fork-bomb exec, SSRF http, KEK leak); clarify KEK derivation + rotation policy; document Linux-only cgroup limits with macOS/Windows gap. `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md` — add "Host-form plugin authoring: the 15-minute tutorial" using fs-readme as the walkthrough. `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md` — mark M3 rows green, update the TOC with the new PathVarResolver and Supervisor interfaces. `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md` — no signature changes (M3 fills in the methods already declared). `/home/linivek/workspace/opendray/docs/plugin-platform/M3-RELEASE.md` — new file mirroring M2-RELEASE.md style: task table, what ships, carve-outs, manual smoke test, sign-off checklist.
- **Acceptance:** new plugin author can follow the host-form tutorial and ship fs-readme-lite in ≤60 minutes.
- **Complexity:** S

### T29 — First-PR seam (optional)
- **Depends on:** none
- **Bundle:** T1 + T3 + T4 + T5 + T8 + T15 into one PR. Zero functional change on the gateway; supervisor/namespaces land per-branch afterwards. Net: new types + migrations + framing codec, all independently testable.
- **Acceptance:** `go test -race ./...` green; behaviour-wise the binary is identical to current `kevlab` HEAD.
- **Complexity:** S

---

## 3. Task dependency graph

```
T1 ──────┬──▶ T2 ────────────────────────────────┐
         ├──▶ T18 ──▶ T19 ──────────────▶ T23 ───┤
         └──▶ T14 ──▶ T16 ──▶ T17 ─────┐         │
                │                       │         │
T3 ──────┬──────│──▶ T9 ──▶ T10 ───────┤         │
         ├──────│──▶ T11 ──────────────┤         │
         └──────│──▶ T12 ──────────────┤         │
                │                       │         │
T4 ──▶ T7 ──▶ T13 ────────────────────┤         │
T6 ──────────┘                          │         │
                                        │         │
T5 ──▶ T14 (dep counted above)          │         │
T15 ──▶ T16 ──▶ T17 ─────────────────── ┤         │
T8  ──▶ T9/T10/T11/T12 (stream helpers) │         │
                                        ▼         │
                                      T23 ────────┘
                                        │
                                        ▼
T24 ──▶ (resolver used by T9/T11/T12 at call time, wired in T23)
T20 ──▶ T21                             │
                                        ▼
                                      T22 ──▶ T25 ──▶ T26 ──▶ T28
                                                    T27 (parallel)
```

Critical path: `T1 → T14 → T16 → T17 → T23 → T25 → T26` (7 hops). Everything else can run in parallel after the first-PR seam (§5).

---

## 4. Test matrix

### Unit tests (`go test -race`, target ≥80% on touched packages)
- `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go` extended (T1)
- `/home/linivek/workspace/opendray/plugin/manifest_validate_test.go` extended (T1, T18)
- `/home/linivek/workspace/opendray/plugin/install/install_ios_test.go` + `install_desktop_test.go` (T2)
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities_expand_test.go` (T3)
- `/home/linivek/workspace/opendray/kernel/store/migrations_test.go` extended (T4, T5, T6)
- `/home/linivek/workspace/opendray/kernel/auth/secret_kek_test.go` (T7)
- `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go` extended (T8)
- `/home/linivek/workspace/opendray/plugin/bridge/api_fs_test.go` (T9, T10)
- `/home/linivek/workspace/opendray/plugin/bridge/api_exec_test.go` (T11)
- `/home/linivek/workspace/opendray/plugin/bridge/api_http_test.go` (T12)
- `/home/linivek/workspace/opendray/plugin/bridge/api_secret_test.go` + `kernel/store/plugin_secret_test.go` (T13)
- `/home/linivek/workspace/opendray/plugin/host/supervisor_test.go` (T14)
- `/home/linivek/workspace/opendray/plugin/host/jsonrpc_test.go` (T15; includes fuzz)
- `/home/linivek/workspace/opendray/plugin/host/mux_test.go` (T16)
- `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler_test.go` (T17)
- `/home/linivek/workspace/opendray/plugin/contributions/registry_test.go` extended (T18)
- `/home/linivek/workspace/opendray/gateway/path_vars_test.go` (T24)

### Integration tests
- `/home/linivek/workspace/opendray/gateway/plugins_lsp_test.go` (T19) — `httptest` + stub sidecar speaks `initialize`.
- `/home/linivek/workspace/opendray/gateway/plugins_consents_patch_test.go` (T20) — includes SLO regression.
- `/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go` extended (T23) — asserts fs/exec/http/secret namespaces registered.
- `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go` (T27).

### End-to-end tests (`//go:build e2e`)
- `/home/linivek/workspace/opendray/plugin/e2e_test.go` extended with `TestE2E_FSReadmeFullLifecycle` (T26) and `TestE2E_KanbanFullLifecycle` (T27).

### Flutter widget tests
- `/home/linivek/workspace/opendray/app/test/features/workbench/plugin_bridge_channel_test.dart` extended (T22).
- `/home/linivek/workspace/opendray/app/test/features/settings/plugin_consents_granular_test.dart` (T21).

### Coverage gate
CI runs `go test -race -cover ./plugin/... ./gateway/... ./kernel/...` with 80% line coverage on every touched package. `gosec ./plugin/bridge/... ./plugin/host/... ./kernel/auth/...` runs on every PR; any new HIGH finding blocks merge. Carry-forward target from M2 (80%+) holds.

---

## 5. Rollout order

### Recommended linear order (single engineer, sequential)

```
T29 (seam) → T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T15 → T18 → T24 → T14 → T16 → T17 → T9 → T10 → T11 → T12 → T13 → T19 → T20 → T21 → T22 → T23 → T25 → T26 → T27 → T28
```

### Parallel lanes (after T29 seam merges)

Three agents can work in parallel:

- **Lane A — Supervisor track:** T14 → T15 (interchangeable order) → T16 → T17 → T19. Owns `/home/linivek/workspace/opendray/plugin/host/*` entirely; no file overlap with the other lanes.
- **Lane B — Capability namespaces:** T3 → T7 → T8 (parallel) → T9 → T10 (sequential — share `api_fs.go`) → T11 → T12 → T13. Owns `/home/linivek/workspace/opendray/plugin/bridge/api_*.go` and `/home/linivek/workspace/opendray/kernel/store/plugin_secret.go` + `plugin_secret_kek.go` + `secret_kek.go`.
- **Lane C — Manifest + UI + docs:** T1 → T2 → T18 → T20 → T21 → T22 → T27 → T28. Owns manifest.go, contributions/registry.go (registry extension only), settings/flutter UI, docs.

**Merge points (no conflicts expected):**
- T23 (main wiring) must be last — consumes everything.
- T25 (fs-readme plugin) authored alongside Lane B; tested by T26 after Lane A + Lane B + T23 merge.
- T24 (path var resolver) authored in Lane C but wired from T23.

**File-isolation analysis:**
- Lane A: 5 new files under `plugin/host/`, plus `cmd/opendray/main.go` additions in T23 only.
- Lane B: 8 new files under `plugin/bridge/` + `kernel/store/` + `kernel/auth/`, plus `cmd/opendray/main.go` additions in T23 only.
- Lane C: 2 files in `plugin/`, 2 in `plugin/contributions/`, 3 Dart files under `app/lib/features/settings/` and `app/lib/features/workbench/`.

Only `cmd/opendray/main.go` sees multi-lane edits; stage the three main.go additions (hostSupervisor, new namespaces, path-var resolver) under a single ordering via T23 once lanes A and B report green.

---

## 6. Security model addendum

References `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md` §Threat model + §Mitigations matrix. M3 adds the following privileged-cap threat rows; these extend, not replace, the existing matrix.

| Threat | Affected cap | Mitigation | Verified by |
|---|---|---|---|
| **TOCTOU** on `fs.readFile` — plugin supplies a path that becomes a symlink to `/etc/passwd` between `stat` and `open`. | `fs.read`, `fs.write` | `filepath.EvalSymlinks` run immediately before `os.Open`, then re-match the resolved path against the grant globs. Any change between the two checks fails closed. | T9 `TestFS_TOCTOUSymlinkEscape`, T10 `TestFS_WriteTOCTOU`. |
| **Fork bomb** via `exec.spawn` — plugin loops spawning itself, exhausting host resources. | `exec` | Per-plugin hard cap of 4 concurrent spawns; queue-and-reject past that with `ETIMEOUT`. Linux cgroup v2 `pids.max=32` per-plugin when writable (warn-once otherwise). Supervisor kills the entire process group on consent revoke. | T11 `TestExec_ForkBombCapped` spawns 100 short-lived `sh` processes; assert ≤4 simultaneously alive. |
| **SSRF bypass** on `http.request` via DNS rebind — plugin-allowed host resolves to `169.254.169.254` on the second lookup. | `http` | Custom `net.Dialer.Control` callback runs immediately before `connect(2)` and re-checks the **resolved IP** via `isPrivateHost`. Blocks both rebind and IP-literal attacks independent of DNS. | T12 `TestHTTP_SSRF_DNSRebind` with a malicious resolver stub. |
| **KEK leak** — host memory dump or `/proc/<pid>/mem` exposes the derived KEK. | `secret` | KEK is derived on every use (never persisted); lives in `crypto/subtle`-safe byte slice; zeroed via `explicit_bzero`-equivalent (`runtime.KeepAlive` + manual fill) in a deferred block after every wrap/unwrap. `plugin_secret_kek.wrapped_dek` is useless without the KEK. Password rotation rotates the KEK (walk + rewrap). `gosec` merge gate on `kernel/auth/secret_kek.go`. | T7 `TestKEK_ZeroedAfterUse`, manual `strings /proc/<pid>/mem` smoke on dev host (documented in M3-RELEASE.md). |
| **Secret cross-plugin leak** via a crafted key name. | `secret` | `MatchSecretNamespace` regex `^[a-zA-Z0-9._-]{1,128}$`; row-level DB scoping (PK is `(plugin_name, key)`). The API never accepts a `plugin_name` parameter — it's implicit from the bridge Conn's plugin. | T13 `TestSecret_KeyInjectionRejected`. |
| **Path-variable injection** — plugin declares `permissions.fs.read=["${workspace}/**"]` but the workspace is a symlink to `/`. | `fs.read`, `fs.write` | `PathVarResolver.Resolve` calls `filepath.EvalSymlinks` on the workspace before returning; if the symlink points outside `$HOME`, the resolver returns an error and the gate fails closed. | T24 `TestPathVar_WorkspaceSymlinkEscape`. |
| **Sidecar impersonation** — sidecar sends a JSON-RPC call claiming to be another plugin. | All sidecar-reachable caps | `HostRPCHandler` is constructed once per sidecar with `Plugin` bound from the Supervisor's plugin-name field; sidecar cannot override it. Methods receive `plugin` from the handler, never from the RPC payload. | T17 `TestHostRPC_MethodInjectionRejected`. |
| **LSP traffic tampering** by an MITM on the loopback bridge. | `contributes.languageServers` | LSP proxy accepts only authenticated WS upgrades from the Flutter host (same JWT middleware as `/api/plugins/{name}/bridge/ws`). Loopback-only in mobile builds per M2 origin policy. | T19 `TestLSP_UnauthenticatedRejected`. |

**Hard guarantees extended (appended to 10-security.md §Hard guarantees):**
- A sidecar cannot read or write another plugin's `plugin_secret` rows regardless of capability grants — enforced by `HostRPCConfig.Plugin` being constructor-injected.
- A plugin without `exec` capability cannot spawn a sidecar child process through `exec.run`; the sidecar itself is governed by `form:"host"` and launches only via Supervisor (which the plugin cannot invoke through the bridge).

---

## 7. Migration from M2

### DB migrations

Next free id is 014 (confirmed: `kernel/store/migrations/013_plugin_audit.sql` is the highest M1 migration; M2 added none). M3 introduces three:

| Migration | File | Purpose |
|---|---|---|
| 014 | `kernel/store/migrations/014_plugin_secret_kek.sql` | Per-plugin wrapped DEK row (T4) |
| 015 | `kernel/store/migrations/015_plugin_host_state.sql` | Supervisor persistence (T5) |
| 016 | `kernel/store/migrations/016_plugin_secret_nonce.sql` | Add AES-GCM nonce column (T6) |

(017 is reserved for potential post-T26 fixup; do not allocate up front.)

Each ships with a matching `*_down.sql`. All migrations are additive — no backfill of existing rows (secret table is empty in M2).

### Wire-format compatibility

M3 stays on `V=1` envelopes. Every M3 method fits inside the existing envelope shape (request/response + `stream:"chunk"|"end"`). No new frame types. Flutter M2 builds continue to work — they just don't expose the new namespaces through the shim (the shim is version-gated by the gateway's served `opendray-shim.js` content; updating the server updates the shim automatically, no Flutter rebuild needed).

### M2 contracts preserved

- Every M2 HTTP endpoint + WS endpoint works unchanged.
- `ContributesV1` JSON is additive (new `languageServers` field is `omitempty`).
- `PermissionsV1` JSON is additive — no field renames; existing fields keep their semantics.
- `plugin_kv` + `plugin_consents` + `plugin_audit` schemas untouched.
- M2 `TestE2E_KanbanFullLifecycle` (landed in T27 as a carry-on) continues to pass unchanged.

### Rollback

Forward-compatible: setting `OPENDRAY_DISABLE_HOST_PLUGINS=1` makes `hostSupervisor.Ensure` return `EUNAVAIL` without killing other features (webview plugins, fs/exec/http/secret from webview plugins, LSP proxy returns 503). Schema rollback: run 016 → 015 → 014 `_down.sql` in reverse; no data loss because `plugin_secret` rows are empty under M2.

---

## 8. Acceptance criteria for "M3 done"

- [ ] `fs-readme` reference plugin installs via `opendray plugin install ./plugins/examples/fs-readme --yes` on Linux + macOS desktop builds (iOS/Android excluded by `!HostFormAllowed`).
- [ ] Invoking `fs-readme.summarise` returns the first 200 bytes of the workspace `README.md` within 500 ms.
- [ ] Killing the sidecar mid-request returns `EUNAVAIL` to the caller (T14 restart + T16 mux integration); a subsequent invoke succeeds within 2 s (supervisor respawned).
- [ ] `rust-analyzer-od` (stretch goal) provides completion on a Rust file opened through the LSP proxy — if Node/rust-analyzer not on CI PATH, skipped with `t.Skip` and noted in M3-RELEASE.md; desktop dev walkthrough still required.
- [ ] All four privileged namespaces respond correctly to their M3 methods under capability enforcement; unauthorized calls return EPERM; SLO: revoke → next call EPERM ≤ 200 ms (matches M2 SLO for the new cap diff path landed in T20).
- [ ] AES-GCM secret round-trip survives host restart; password rotation rotates the KEK via a one-shot rewrap walk on next login (T7 + T13).
- [ ] `go test -race -cover ./...` ≥ 80% on every package touched by M3.
- [ ] `go vet ./...` clean; `staticcheck ./...` clean on touched packages; `gosec ./plugin/bridge/... ./plugin/host/... ./kernel/auth/...` introduces no new HIGH findings.
- [ ] `flutter test` passes: every M1 + M2 widget test still green + new T21/T22 widget tests green.
- [ ] All 17 bundled legacy manifests + kanban + time-ninja + fs-readme load byte-for-byte unchanged (golden file assertion extended).
- [ ] M2's `TestE2E_KanbanFullLifecycle` (now landed via T27) continues to pass unchanged.
- [ ] iOS archive builds successfully. Any attempt to install a `form:"host"` plugin on iOS fails with `ErrHostFormNotSupported` at Stage — verified in `TestE2E_FSReadmeFullLifecycle` step 7.
- [ ] LSP proxy accepts an `initialize` request from a `wscat` client and gets a response from the test fixture sidecar.
- [ ] SSRF test `TestHTTP_SSRF_DNSRebind` + TOCTOU test `TestFS_TOCTOUSymlinkEscape` + fork-bomb test `TestExec_ForkBombCapped` all green.
- [ ] Docs: `11-developer-experience.md` has a host-form plugin authoring tutorial; `10-security.md` carries the §6 threat/mitigation rows verbatim; `M3-RELEASE.md` published with task table + smoke-test walkthrough + sign-off checklist.

### Smoke test walkthrough (manual)

Run on Linux desktop with Node 20 and PostgreSQL (via embedded-postgres) on PATH.

```bash
# 1. Fresh data dir
export OPENDRAY_DATA_DIR="${HOME}/.opendray-test-m3"
rm -rf "$OPENDRAY_DATA_DIR"; mkdir -p "$OPENDRAY_DATA_DIR/plugins/.installed"

# 2. Build + launch
cd /home/linivek/workspace/opendray
go build -o opendray ./cmd/opendray
OPENDRAY_ALLOW_LOCAL_PLUGINS=1 OPENDRAY_DATA_DIR="$OPENDRAY_DATA_DIR" ./opendray &
GATEWAY_PID=$!; sleep 3

# 3. Install fs-readme
./opendray plugin install ./plugins/examples/fs-readme --yes
# Expected: "Installing fs-readme@1.0.0 with capabilities: fs.read, exec"

# 4. Invoke the command
TOKEN="<device-code flow token>"
curl -X POST "http://localhost:8080/api/plugins/fs-readme/commands/fs-readme.summarise/invoke" \
     -H "Authorization: Bearer $TOKEN"
# Expected: first 200 bytes of workspace README.md

# 5. Revoke fs.read; retry; expect EPERM
curl -X PATCH "http://localhost:8080/api/plugins/fs-readme/consents" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"fs":{"read":[]}}'
curl -X POST "http://localhost:8080/api/plugins/fs-readme/commands/fs-readme.summarise/invoke" \
     -H "Authorization: Bearer $TOKEN"
# Expected: {error:{code:"EPERM"}}

# 6. Kill sidecar; supervisor restarts
pkill -f "node.*fs-readme/sidecar.js"
sleep 2
curl -X POST "http://localhost:8080/api/plugins/fs-readme/commands/fs-readme.summarise/invoke" \
     -H "Authorization: Bearer $TOKEN"
# Expected: success (after re-grant); sidecar PID is different from before

# 7. Uninstall; verify cascade
curl -X DELETE "http://localhost:8080/api/plugins/fs-readme" -H "Authorization: Bearer $TOKEN"
psql $OPENDRAY_DATABASE_URL -c "SELECT count(*) FROM plugin_secret_kek WHERE plugin_name='fs-readme';"
# Expected: 0

kill $GATEWAY_PID
```

### SLO targets

| Metric | Target | Measured in |
|---|---|---|
| `fs.readFile` (p95, 100 KiB file, warm cache) | ≤ 30 ms | T9 benchmark |
| `exec.run` (`git --version`, cold sidecar) | ≤ 150 ms | T11 benchmark |
| `http.request` (GET api.github.com, LAN) | ≤ 300 ms | T12 benchmark |
| `secret.get` + `secret.set` round-trip | ≤ 20 ms | T13 benchmark |
| Sidecar cold start (Node, fs-readme fixture) | ≤ 800 ms | T14 benchmark |
| Consent revoke → next EPERM (carry-forward from M2) | ≤ 200 ms | T20 SLO test |
| LSP `initialize` round-trip through proxy | ≤ 500 ms | T19 integration test |

---

## Open questions

1. **KEK source.** Proposal: derive via HKDF from the bcrypt admin password hash (not the password itself — the stored hash). Con: rotating the admin password requires walking every `plugin_secret` row and rewrapping. Alternative: introduce a dedicated host KEK row in a new `host_kek` table, seeded on first boot from `crypto/rand` and wrapped by an OS keychain entry (macOS Keychain, Linux `libsecret`). Pro: rotation is local. Con: cross-platform keychain integration is an M6 rabbit hole. **Recommend option A (HKDF from admin hash) for M3; flag option B as an M6 follow-up.** Kev to sign off.

2. **gRPC vs JSON-RPC for sidecars.** M3 proposes JSON-RPC 2.0 over stdio with LSP framing because (a) LSP sidecars already speak it, (b) Node/Python/Deno implementations are trivial, (c) matches `02-manifest.md` §host which locks `protocol: "jsonrpc-stdio"`. gRPC is nicer for structured RPC but forces a code-gen toolchain on plugin authors. **Recommend JSON-RPC stays. No change needed.** Flagged for completeness.

3. **iOS host-form.** M3 hard-refuses `form:"host"` on iOS builds (App Store §2.5.2 risk). This means rust-analyzer-od never ships on iOS, which is fine, but it also means **desktop Linux, desktop macOS, and Android** are the supported host-form platforms. Android in particular may need extra scrutiny (Google Play §4.4 — external code). **Recommend: M3 ships host-form on Linux + macOS desktop only; Android is build-flag-gated off until a Google Play review is done (M4).** Kev to sign off.

4. **cgroup v2 on Linux.** Without `CAP_SYS_ADMIN`, opendray cannot enforce cgroup limits. User-mode opendray running under a login session typically does have its own cgroup (`user.slice/user-<uid>.slice/user@<uid>.service`), writable for non-privileged operations, but the set of writable controllers is distro-dependent. **Recommend: attempt to write `memory.max` + `pids.max` on sidecar start; log a single warning per host boot if rejected; do not fail the sidecar start.** Document the gap in 10-security.md.

5. **fs.watch inotify limits.** Linux inotify has per-user limits (`/proc/sys/fs/inotify/max_user_watches`, default 8192 on most distros, 128 on some containers). Plugin authors hitting the limit see confusing errors. **Recommend: per-plugin cap of 16 active watches surfaced as a clear EINVAL. Document the sysctl knob in the authoring tutorial.**

6. **Redirect matching on http.request.** Should a redirect chain hop that matches a **different** plugin-declared URL pattern still be allowed, or must all hops stay inside the single initial pattern? The conservative choice is "re-match every hop against the full grant list" (any-match). Permissive choice is "only the final URL must match". **Recommend: any-match across the grant list (all hops re-checked against the entire allowlist, not only the initial pattern). Aligns with how browsers treat `connect-src`.** Kev to sign off.

7. **LSP proxy auth model.** M3 uses the existing JWT middleware for the proxy route, same as bridge WS. But plugins whose `contributes.languageServers` are declared implicitly get LSP access — there's no separate `permissions.lsp` cap. Is that acceptable, or should LSP traffic be gated by a new `permissions.languageServer` key? **Recommend: implicit grant via install consent + the existing `contributes.languageServers` contribution — matches VS Code's model. No new cap.** Kev to sign off.

8. **Python runtime support.** `02-manifest.md` §host locks `runtime: {binary, node, deno}`. The prompt adds `python3, bun, custom` — this is a manifest schema change. **Recommend: extend the enum in T1 and patch `02-manifest.md` in T28. Raise explicitly in the PR description so schema drift is visible.** Kev to confirm the enum extension is OK before T1 lands.

---

## Relevant file paths (all absolute)

### Design contract
- `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/02-manifest.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/03-contribution-points.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/05-capabilities.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/07-lifecycle.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M1-PLAN.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M2-PLAN.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M2-RELEASE.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md`

### Existing code anchors
- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/manifest_validate.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go`
- `/home/linivek/workspace/opendray/plugin/bridge/manager.go`
- `/home/linivek/workspace/opendray/plugin/bridge/protocol.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_workbench.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_storage.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_events.go`
- `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go`
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go`
- `/home/linivek/workspace/opendray/plugin/compat/synthesize.go`
- `/home/linivek/workspace/opendray/plugin/install/install.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/e2e_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/012_plugin_secret.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/013_plugin_audit.sql`
- `/home/linivek/workspace/opendray/kernel/auth/credentials.go`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/plugins_bridge.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents.go`
- `/home/linivek/workspace/opendray/gateway/workbench_stream.go`
- `/home/linivek/workspace/opendray/cmd/opendray/main.go`
- `/home/linivek/workspace/opendray/app/lib/features/settings/plugin_consents_page.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart`
- `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart`

### Files to be created (M3)
- `/home/linivek/workspace/opendray/plugin/host_os_ios.go`
- `/home/linivek/workspace/opendray/plugin/host_os_other.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_fs.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_fs_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_exec.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_exec_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_http.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_http_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_secret.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_secret_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities_expand_test.go`
- `/home/linivek/workspace/opendray/plugin/host/supervisor.go`
- `/home/linivek/workspace/opendray/plugin/host/supervisor_test.go`
- `/home/linivek/workspace/opendray/plugin/host/process.go`
- `/home/linivek/workspace/opendray/plugin/host/process_unix.go`
- `/home/linivek/workspace/opendray/plugin/host/process_windows.go`
- `/home/linivek/workspace/opendray/plugin/host/jsonrpc.go`
- `/home/linivek/workspace/opendray/plugin/host/jsonrpc_test.go`
- `/home/linivek/workspace/opendray/plugin/host/mux.go`
- `/home/linivek/workspace/opendray/plugin/host/mux_test.go`
- `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler.go`
- `/home/linivek/workspace/opendray/plugin/host/host_rpc_handler_test.go`
- `/home/linivek/workspace/opendray/plugin/install/install_ios_test.go`
- `/home/linivek/workspace/opendray/plugin/install/install_desktop_test.go`
- `/home/linivek/workspace/opendray/kernel/auth/secret_kek.go`
- `/home/linivek/workspace/opendray/kernel/auth/secret_kek_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_secret.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_secret_test.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/014_plugin_secret_kek_down.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/015_plugin_host_state_down.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/016_plugin_secret_nonce.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/016_plugin_secret_nonce_down.sql`
- `/home/linivek/workspace/opendray/gateway/plugins_lsp.go`
- `/home/linivek/workspace/opendray/gateway/plugins_lsp_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents_patch.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents_patch_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go`
- `/home/linivek/workspace/opendray/gateway/path_vars.go`
- `/home/linivek/workspace/opendray/gateway/path_vars_test.go`
- `/home/linivek/workspace/opendray/plugins/examples/fs-readme/manifest.json`
- `/home/linivek/workspace/opendray/plugins/examples/fs-readme/sidecar.js`
- `/home/linivek/workspace/opendray/plugins/examples/fs-readme/README.md`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host_desktop.dart` (M2 T16b carry-on)
- `/home/linivek/workspace/opendray/app/test/features/settings/plugin_consents_granular_test.dart`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M3-RELEASE.md`
