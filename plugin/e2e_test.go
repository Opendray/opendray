//go:build e2e

// Package plugin_test — T18 + T24: end-to-end install → invoke → uninstall
// integration harness for the M1 plugin platform.
//
// Boots an embedded Postgres, constructs every live component
// (contributions registry, runtime, bridge gate, installer, dispatcher,
// gateway server), and exercises the full HTTP surface via httptest.Server
// against the time-ninja reference fixture.
//
// Run with:
//
//	go test -race -tags=e2e -timeout=10m ./plugin/...
//
// Without the `e2e` build tag this file is excluded entirely, so the
// default `go test ./...` run stays fast and offline.
package plugin_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/opendray/opendray/gateway"
	"github.com/opendray/opendray/kernel/hub"
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/commands"
	"github.com/opendray/opendray/plugin/contributions"
	"github.com/opendray/opendray/plugin/install"
)

// ─── Consent / audit adapters ────────────────────────────────────────────────
//
// bridge.Gate speaks two tiny local interfaces (ConsentReader, AuditSink) so
// that the bridge package stays decoupled from kernel/store. These adapters
// satisfy those interfaces by forwarding to *store.DB.

type storeConsentReader struct{ db *store.DB }

func (s *storeConsentReader) Load(ctx context.Context, pluginName string) ([]byte, bool, error) {
	c, err := s.db.GetConsent(ctx, pluginName)
	if errors.Is(err, store.ErrConsentNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte(c.PermsJSON), true, nil
}

type storeAuditSink struct{ db *store.DB }

func (s *storeAuditSink) Append(ctx context.Context, ev bridge.AuditEvent) error {
	return s.db.AppendAudit(ctx, store.AuditEntry{
		PluginName: ev.PluginName,
		Ns:         ev.Ns,
		Method:     ev.Method,
		Caps:       ev.Caps,
		Result:     ev.Result,
		DurationMs: ev.DurationMs,
		ArgsHash:   ev.ArgsHash,
		Message:    ev.Message,
	})
}

// dispatcherInvoker wraps commands.Dispatcher.Invoke so it satisfies the
// gateway package's unexported commandInvoker interface:
//
//	Invoke(ctx, plugin, id, args) (any, error)
//
// *commands.Dispatcher returns (*commands.Result, error); the signature
// difference means the concrete type does not itself satisfy the interface.
// This thin adapter is the same shape cmd/main.go will eventually ship.
type dispatcherInvoker struct{ d *commands.Dispatcher }

func (a dispatcherInvoker) Invoke(ctx context.Context, pluginName, commandID string, args map[string]any) (any, error) {
	return a.d.Invoke(ctx, pluginName, commandID, args)
}

// ─── testHarness ─────────────────────────────────────────────────────────────

// testHarness bundles every live component so subtests can reach into them
// without re-wiring. restartServer() tears down the HTTP layer + in-memory
// state and rebuilds it against the SAME DB + same DataDir — this is the
// exact restart scenario the T12 loadIntoMemory contract promises to survive.
type testHarness struct {
	t *testing.T

	pgPort int
	pg     *embeddedpostgres.EmbeddedPostgres

	db      *store.DB
	dataDir string

	// Per-boot components (destroyed and recreated by restartServer).
	registry   *contributions.Registry
	rt         *plugin.Runtime
	hookBus    *plugin.HookBus
	gate       *bridge.Gate
	installer  *install.Installer
	dispatcher *commands.Dispatcher
	h          *hub.Hub
	gwServer   *gateway.Server
	httpServer *httptest.Server
	client     *http.Client
	baseURL    string
}

// newHarness brings up embedded Postgres, migrates, and constructs the
// initial set of live components. t.Cleanup ensures everything shuts down
// even on panic.
func newHarness(t *testing.T) *testHarness {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	port, err := freePort()
	if err != nil {
		t.Fatalf("free port: %v", err)
	}

	pgDataDir := t.TempDir()
	cacheDir := filepath.Join(os.TempDir(), "opendray-pg-cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("opendray").
			Password("testpw").
			Database("opendray").
			Port(uint32(port)).
			DataPath(pgDataDir).
			RuntimePath(filepath.Join(cacheDir, "runtime")).
			BinariesPath(cacheDir).
			StartTimeout(2 * time.Minute),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("pg start: %v", err)
	}
	t.Cleanup(func() { _ = pg.Stop() })

	db, err := store.New(ctx, store.Config{
		Host:     "127.0.0.1",
		Port:     fmt.Sprintf("%d", port),
		User:     "opendray",
		Password: "testpw",
		DBName:   "opendray",
	})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(db.Close)

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	h := &testHarness{
		t:       t,
		pgPort:  port,
		pg:      pg,
		db:      db,
		dataDir: t.TempDir(),
	}

	h.startServer(ctx)

	// Ensure the server stops even if the test panics mid-flight.
	t.Cleanup(func() { h.stopServer() })

	return h
}

// startServer (re)builds every in-memory component and a fresh httptest.Server.
// Safe to call repeatedly — used both on first boot and after restartServer.
func (h *testHarness) startServer(ctx context.Context) {
	h.t.Helper()

	h.registry = contributions.NewRegistry()
	h.hookBus = plugin.NewHookBus(slog.Default())

	// Runtime wires the contributions registry so every plugin load pushes
	// its declared contributes into the flat registry. This is the T12
	// contract the post-restart assertions verify.
	h.rt = plugin.NewRuntime(
		h.db,
		h.hookBus,
		"", // no filesystem pluginDir — we install via HTTP in this test
		slog.Default(),
		plugin.WithContributions(h.registry),
	)
	// LoadAll re-seeds bundled embedded manifests AND re-registers any
	// plugin stored in the DB — this is the key T12 restart behaviour.
	if err := h.rt.LoadAll(ctx); err != nil {
		h.t.Fatalf("runtime.LoadAll: %v", err)
	}

	h.gate = bridge.NewGate(
		&storeConsentReader{db: h.db},
		&storeAuditSink{db: h.db},
		slog.Default(),
	)

	h.installer = install.NewInstaller(h.dataDir, h.db, h.rt, h.gate, slog.Default())
	// T25: Installer.AllowLocal is the config-backed gate for local: sources.
	// The kernel/config loader normally populates this from
	// OPENDRAY_ALLOW_LOCAL_PLUGINS; in-process tests set it directly.
	h.installer.AllowLocal = true

	dp, err := commands.NewDispatcher(commands.Config{
		Registry: h.registry,
		Gate:     h.gate,
		Log:      slog.Default(),
	})
	if err != nil {
		h.t.Fatalf("commands.NewDispatcher: %v", err)
	}
	h.dispatcher = dp

	// Hub is constructed with just the DB — no resolver, no events. The
	// gateway server only uses hub for /api/health (which we don't hit)
	// and for DB() inside the audit handler (which we do hit).
	h.h = hub.New(hub.Config{
		DB:     h.db,
		Logger: slog.Default(),
	})

	// Construct the real gateway.Server via gateway.New. Auth=nil means the
	// protected route group runs without JWT middleware — fine for an
	// E2E test of business logic. Telegram watch loop spins harmlessly
	// because no "telegram" plugin is configured in this test.
	h.gwServer = gateway.New(gateway.Config{
		Hub:           h.h,
		Plugins:       h.rt,
		Auth:          nil,
		AdminUsername: "test",
		AdminPassword: "test",
		Logger:        slog.Default(),
		Installer:     h.installer,
		Contributions: h.registry,
		CommandInvoker: dispatcherInvoker{d: h.dispatcher},
	})

	h.httpServer = httptest.NewServer(h.gwServer.Handler())
	h.client = h.httpServer.Client()
	h.baseURL = h.httpServer.URL
}

// stopServer tears down the HTTP layer and any component goroutines. Safe
// to call more than once — idempotent via the embedded Installer.Stop()
// once-guard. Leaves the DB + DataDir intact so the harness can be restarted.
func (h *testHarness) stopServer() {
	if h.httpServer != nil {
		h.httpServer.Close()
		h.httpServer = nil
	}
	if h.installer != nil {
		h.installer.Stop()
	}
	// hub/runtime/hookBus have no explicit Stop hook in M1 — their
	// goroutines are bound to the process. Nothing to close here.
}

// restartServer stops the HTTP + in-memory components and reconstructs a
// fresh set against the same DB + same DataDir. Crucially the registry is
// a BRAND NEW instance — so any post-restart assertions on registry
// contents exercise LoadAll's re-registration path.
func (h *testHarness) restartServer() {
	h.t.Helper()
	h.stopServer()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	h.startServer(ctx)
}

// ─── HTTP helpers ────────────────────────────────────────────────────────────

// doJSON issues an HTTP request against the harness's baseURL and decodes
// the response body into `out`. Returns the status code for assertion.
func (h *testHarness) doJSON(method, path string, body any, out any) int {
	h.t.Helper()

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("marshal %s %s: %v", method, path, err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, h.baseURL+path, reqBody)
	if err != nil {
		h.t.Fatalf("new request %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatalf("read body %s %s: %v", method, path, err)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			h.t.Fatalf("decode %s %s body=%q: %v", method, path, string(raw), err)
		}
	}
	return resp.StatusCode
}

// ─── Test subject path ───────────────────────────────────────────────────────

// timeNinjaPath walks up from this test file's location until it finds the
// repo root (identified by go.mod), then joins plugins/examples/time-ninja.
// Using runtime.Caller keeps the test location-independent even if callers
// shift the working dir.
func timeNinjaPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	// Walk up looking for go.mod.
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			p := filepath.Join(dir, "plugins", "examples", "time-ninja")
			if _, err := os.Stat(filepath.Join(p, "manifest.json")); err != nil {
				t.Fatalf("time-ninja manifest missing at %s: %v", p, err)
			}
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root (no go.mod in any ancestor)")
	return ""
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// countRows is a tiny convenience for the post-uninstall assertions.
func countRows(t *testing.T, db *store.DB, q string, args ...any) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var n int
	if err := db.Pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		t.Fatalf("count rows %q: %v", q, err)
	}
	return n
}

// ─── TestE2E_TimeNinjaFullLifecycle ──────────────────────────────────────────

// TestE2E_TimeNinjaFullLifecycle exercises install → invoke → restart →
// invoke again → uninstall → no-trace verification. Each stage is a
// subtest so a failure points at the exact phase without obscuring the
// flow. Subtests share the same harness instance.
//
// Build tag `e2e` keeps this out of the default `go test ./...` run — CI
// opts in via `-tags=e2e`.
func TestE2E_TimeNinjaFullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full E2E under -short")
	}

	// OPENDRAY_ALLOW_LOCAL_PLUGINS=1 flips the install gate in
	// gateway/plugins_install.go so local: sources are accepted.
	t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", "1")

	h := newHarness(t)

	// Shared state passed between subtests.
	var installToken string
	ninjaDir := timeNinjaPath(t)

	t.Run("Boot", func(t *testing.T) {
		// newHarness already booted everything — just sanity-check that
		// every component is live and the HTTP layer is reachable.
		if h.db == nil || h.installer == nil || h.dispatcher == nil {
			t.Fatal("harness not fully wired")
		}
		var body map[string]any
		code := h.doJSON(http.MethodGet, "/api/health", nil, &body)
		if code != http.StatusOK {
			t.Fatalf("/api/health: got %d, want 200", code)
		}
	})

	t.Run("Install", func(t *testing.T) {
		req := map[string]string{"src": "local:" + ninjaDir}
		var got struct {
			Token        string                 `json:"token"`
			Name         string                 `json:"name"`
			Version      string                 `json:"version"`
			Perms        map[string]interface{} `json:"perms"`
			ExpiresAt    time.Time              `json:"expiresAt"`
			ManifestHash string                 `json:"manifestHash"`
		}
		code := h.doJSON(http.MethodPost, "/api/plugins/install", req, &got)
		if code != http.StatusAccepted {
			t.Fatalf("install: status %d (want 202), got=%+v", code, got)
		}
		if got.Token == "" {
			t.Fatal("install: empty token")
		}
		if got.Name != "time-ninja" {
			t.Errorf("install: name=%q, want time-ninja", got.Name)
		}
		if got.Version != "1.0.0" {
			t.Errorf("install: version=%q, want 1.0.0", got.Version)
		}
		// time-ninja declares {} permissions — every known cap key must be absent.
		for _, k := range []string{"fs", "exec", "http", "session", "storage", "secret"} {
			if v, ok := got.Perms[k]; ok && v != nil && v != false && v != "" {
				t.Errorf("install: perms[%q]=%v; expected empty/false/absent", k, v)
			}
		}
		installToken = got.Token
	})

	t.Run("Confirm", func(t *testing.T) {
		if installToken == "" {
			t.Skip("install failed; skipping confirm")
		}
		var got struct {
			Installed bool   `json:"installed"`
			Name      string `json:"name"`
		}
		code := h.doJSON(http.MethodPost, "/api/plugins/install/confirm",
			map[string]string{"token": installToken}, &got)
		if code != http.StatusOK {
			t.Fatalf("confirm: status %d (want 200), got=%+v", code, got)
		}
		if !got.Installed {
			t.Errorf("confirm: installed=%v", got.Installed)
		}
		if got.Name != "time-ninja" {
			t.Errorf("confirm: name=%q, want time-ninja", got.Name)
		}

		// DB side-effects: one plugins row, one plugin_consents row, perms_json
		// is a syntactically valid JSON object.
		if n := countRows(t, h.db,
			`SELECT count(*) FROM plugins WHERE name=$1`, "time-ninja"); n != 1 {
			t.Errorf("plugins row count=%d, want 1", n)
		}
		if n := countRows(t, h.db,
			`SELECT count(*) FROM plugin_consents WHERE plugin_name=$1`, "time-ninja"); n != 1 {
			t.Errorf("plugin_consents row count=%d, want 1", n)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var perms []byte
		err := h.db.Pool.QueryRow(ctx,
			`SELECT perms_json FROM plugin_consents WHERE plugin_name=$1`,
			"time-ninja").Scan(&perms)
		if err != nil {
			t.Fatalf("fetch perms_json: %v", err)
		}
		var anything map[string]any
		if err := json.Unmarshal(perms, &anything); err != nil {
			t.Errorf("perms_json is not valid JSON object: %v (raw=%q)", err, string(perms))
		}
	})

	t.Run("Contributions", func(t *testing.T) {
		assertTimeNinjaContributions(t, h, "after install")
	})

	t.Run("InvokeCommand", func(t *testing.T) {
		var raw json.RawMessage
		code := h.doJSON(http.MethodPost,
			"/api/plugins/time-ninja/commands/time.start/invoke",
			map[string]any{"args": map[string]any{}},
			&raw)
		if code != http.StatusOK {
			t.Fatalf("invoke: status %d, body=%s", code, string(raw))
		}
		var r struct {
			Kind    string `json:"kind"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("decode invoke body %q: %v", string(raw), err)
		}
		if r.Kind != "notify" {
			t.Errorf("invoke: kind=%q, want notify", r.Kind)
		}
		if r.Message != "Pomodoro started — 25 minutes" {
			t.Errorf("invoke: message=%q, want Pomodoro started — 25 minutes", r.Message)
		}
	})

	t.Run("SurviveRestart", func(t *testing.T) {
		// Tear down the HTTP server + in-memory state, then rebuild against
		// the same DB + DataDir. The key invariant: T12's loadIntoMemory
		// must push time-ninja's contributions into the brand-new registry.
		h.restartServer()
		assertTimeNinjaContributions(t, h, "after restart")
	})

	t.Run("InvokeAfterRestart", func(t *testing.T) {
		var raw json.RawMessage
		code := h.doJSON(http.MethodPost,
			"/api/plugins/time-ninja/commands/time.start/invoke",
			map[string]any{"args": map[string]any{}},
			&raw)
		if code != http.StatusOK {
			t.Fatalf("invoke post-restart: status %d, body=%s", code, string(raw))
		}
		var r struct {
			Kind    string `json:"kind"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("decode post-restart body %q: %v", string(raw), err)
		}
		if r.Kind != "notify" || r.Message != "Pomodoro started — 25 minutes" {
			t.Errorf("post-restart invoke: kind=%q msg=%q", r.Kind, r.Message)
		}
	})

	t.Run("Uninstall", func(t *testing.T) {
		var got struct {
			Status string `json:"status"`
			Name   string `json:"name"`
		}
		code := h.doJSON(http.MethodDelete, "/api/plugins/time-ninja", nil, &got)
		if code != http.StatusOK {
			t.Fatalf("uninstall: status %d, got=%+v", code, got)
		}
		if got.Status != "uninstalled" {
			t.Errorf("uninstall: status=%q, want uninstalled", got.Status)
		}
	})

	t.Run("UninstallNoTrace", func(t *testing.T) {
		// T24 merged assertions: DB + FS + registry are all clean of
		// time-ninja, but audit rows must survive (historical record).
		if n := countRows(t, h.db,
			`SELECT count(*) FROM plugins WHERE name=$1`, "time-ninja"); n != 0 {
			t.Errorf("plugins row count=%d, want 0", n)
		}
		if n := countRows(t, h.db,
			`SELECT count(*) FROM plugin_consents WHERE plugin_name=$1`, "time-ninja"); n != 0 {
			t.Errorf("plugin_consents row count=%d, want 0", n)
		}
		if n := countRows(t, h.db,
			`SELECT count(*) FROM plugin_audit WHERE plugin_name=$1`, "time-ninja"); n == 0 {
			t.Errorf("plugin_audit rows=%d, want > 0 (audit is historical)", n)
		}

		// Filesystem: final bundle path must be gone.
		finalPath := filepath.Join(h.dataDir, "time-ninja", "1.0.0")
		if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
			t.Errorf("expected %s to be gone (ErrNotExist), got err=%v", finalPath, err)
		}

		// In-memory registry: no entries with PluginName == time-ninja.
		flat := h.registry.Flatten()
		for _, c := range flat.Commands {
			if c.PluginName == "time-ninja" {
				t.Errorf("registry still has command %q from time-ninja", c.ID)
			}
		}
		for _, s := range flat.StatusBar {
			if s.PluginName == "time-ninja" {
				t.Errorf("registry still has statusBar %q from time-ninja", s.ID)
			}
		}
		for _, k := range flat.Keybindings {
			if k.PluginName == "time-ninja" {
				t.Errorf("registry still has keybinding %q from time-ninja", k.Key)
			}
		}
		for path, entries := range flat.Menus {
			for _, e := range entries {
				if e.PluginName == "time-ninja" {
					t.Errorf("registry still has menu entry in %s from time-ninja", path)
				}
			}
		}

		// HTTP: workbench endpoint also returns no time-ninja entries.
		var body map[string]json.RawMessage
		code := h.doJSON(http.MethodGet, "/api/workbench/contributions", nil, &body)
		if code != http.StatusOK {
			t.Fatalf("post-uninstall contributions: status %d", code)
		}
		var cmds []map[string]any
		_ = json.Unmarshal(body["commands"], &cmds)
		for _, c := range cmds {
			if c["pluginName"] == "time-ninja" {
				t.Errorf("/api/workbench/contributions still exposes time-ninja command: %v", c)
			}
		}
	})
}

// ─── Shared contribution-assertion helper ────────────────────────────────────

// assertTimeNinjaContributions fetches /api/workbench/contributions and
// asserts every slot time-ninja declares is present. Used both after the
// initial install and after the restart so both call sites share one
// source of truth for the expected shape.
func assertTimeNinjaContributions(t *testing.T, h *testHarness, label string) {
	t.Helper()

	var flat struct {
		Commands []struct {
			PluginName string `json:"pluginName"`
			ID         string `json:"id"`
		} `json:"commands"`
		StatusBar []struct {
			PluginName string `json:"pluginName"`
			ID         string `json:"id"`
			Text       string `json:"text"`
		} `json:"statusBar"`
		Keybindings []struct {
			PluginName string `json:"pluginName"`
			Command    string `json:"command"`
			Key        string `json:"key"`
		} `json:"keybindings"`
		Menus map[string][]struct {
			PluginName string `json:"pluginName"`
			Command    string `json:"command"`
		} `json:"menus"`
	}

	code := h.doJSON(http.MethodGet, "/api/workbench/contributions", nil, &flat)
	if code != http.StatusOK {
		t.Fatalf("%s: /api/workbench/contributions status %d", label, code)
	}

	// commands includes {id:"time.start", pluginName:"time-ninja"}
	foundCmd := false
	for _, c := range flat.Commands {
		if c.PluginName == "time-ninja" && c.ID == "time.start" {
			foundCmd = true
			break
		}
	}
	if !foundCmd {
		t.Errorf("%s: commands[] missing time-ninja/time.start, got %+v", label, flat.Commands)
	}

	// statusBar includes {id:"time.bar", text:"🍅 25:00", pluginName:"time-ninja"}
	foundSB := false
	for _, s := range flat.StatusBar {
		if s.PluginName == "time-ninja" && s.ID == "time.bar" && s.Text == "🍅 25:00" {
			foundSB = true
			break
		}
	}
	if !foundSB {
		t.Errorf("%s: statusBar[] missing time-ninja/time.bar, got %+v", label, flat.StatusBar)
	}

	// keybindings includes {command:"time.start", key:"ctrl+alt+p"}
	foundKB := false
	for _, k := range flat.Keybindings {
		if k.PluginName == "time-ninja" && k.Command == "time.start" && k.Key == "ctrl+alt+p" {
			foundKB = true
			break
		}
	}
	if !foundKB {
		t.Errorf("%s: keybindings[] missing time-ninja ctrl+alt+p, got %+v", label, flat.Keybindings)
	}

	// menus["appBar/right"] has one entry with command == time.start
	foundMenu := false
	for _, e := range flat.Menus["appBar/right"] {
		if e.PluginName == "time-ninja" && e.Command == "time.start" {
			foundMenu = true
			break
		}
	}
	if !foundMenu {
		t.Errorf("%s: menus[\"appBar/right\"] missing time-ninja/time.start, got %+v",
			label, flat.Menus["appBar/right"])
	}
}

// Compile-time check that sql is imported so the test binary links the pgx
// stdlib driver embedded-postgres needs. (Blank import above pulls it in,
// this just keeps the unused-import linter quiet across Go versions.)
var _ = sql.Drivers
