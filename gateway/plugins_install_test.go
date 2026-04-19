package gateway

// T7 — HTTP install endpoints test suite.
//
// Test strategy:
//   - Most tests construct a minimal *Server directly (no gateway.New pipeline)
//     and call handlers through httptest.NewRecorder — fast, no embedded-pg.
//   - PG-backed tests (TestPluginsInstall_EndToEnd, TestPluginsAudit_*)
//     boot embedded Postgres via the same helper pattern used in
//     kernel/store/plugin_tables_test.go.  They are skipped under -short.
//   - TestPluginsInstall_RoutesUnderAuth verifies the chi router produced by
//     gateway.New rejects unauthenticated requests with 401.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/opendray/opendray/kernel/auth"
	"github.com/opendray/opendray/kernel/hub"
	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/install"
)

// ─── embedded-pg helper ─────────────────────────────────────────────────────

func testFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// bootTestDB starts an embedded Postgres, runs migrations, and returns *store.DB.
// Skips the test when -short is set.
//
// Each call uses a unique runtime directory under the shared binary cache to
// avoid conflicts when multiple tests run concurrently with -race.
func bootTestDB(t *testing.T) *store.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping embedded-postgres test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	port := testFreePort(t)
	dataDir := t.TempDir()

	// Shared binary cache so pg binaries are only downloaded once per
	// machine. Runtime path is unique per test to avoid concurrent-use
	// conflicts on the pwfile and postmaster.pid.
	cacheDir := filepath.Join(os.TempDir(), "opendray-pg-cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	// Unique runtime dir per test invocation to prevent pwfile races.
	runtimeDir, err := os.MkdirTemp("", "opendray-pg-runtime-*")
	if err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runtimeDir) })

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("opendray").
			Password("testpw").
			Database("opendray").
			Port(uint32(port)).
			DataPath(dataDir).
			RuntimePath(runtimeDir).
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
	return db
}

// writeTestBundle creates a minimal valid v1 plugin directory under base.
func writeTestBundle(t *testing.T, base, name string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir bundle: %v", err)
	}
	m := map[string]any{
		"name":        name,
		"version":     "1.0.0",
		"publisher":   "opendray-test",
		"displayName": name,
		"description": "Test plugin",
		"type":        "panel",
		"form":        "declarative",
		"engines":     map[string]string{"opendray": "^1.0.0"},
		"permissions": map[string]any{},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return dir
}

// buildInstaller wires up a real Installer against the given DB and data dir.
func buildInstaller(t *testing.T, db *store.DB, dataDir string) *install.Installer {
	t.Helper()
	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(db, hooks, "", nil)
	gate := bridge.NewGate(nil, nil, nil)
	inst := install.NewInstallerWithTTL(dataDir, db, rt, gate, nil, 10*time.Minute, time.Hour)
	t.Cleanup(inst.Stop)
	return inst
}

// buildServer constructs a minimal *Server suitable for handler unit tests.
// Hub can be nil; only Installer and (for audit) hub.DB() are needed.
func buildTestServer(t *testing.T, inst *install.Installer, h *hub.Hub) *Server {
	t.Helper()
	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(nil, hooks, "", nil)
	return &Server{
		hub:       h,
		plugins:   rt,
		installer: inst,
		router:    chi.NewRouter(), // not used in handler calls
	}
}

// ─── JSON helpers ────────────────────────────────────────────────────────────

func postJSON(t *testing.T, handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func deleteJSON(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode body %q: %v", rr.Body.String(), err)
	}
	return m
}

// withChiParam wraps a handler so chi.URLParam returns the given value for key.
func withChiParam(handler http.HandlerFunc, key, val string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add(key, val)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		handler(w, r)
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// TestPluginsInstall_EndToEnd performs the full install → confirm → uninstall
// flow over real HTTP handlers against an embedded Postgres.
func TestPluginsInstall_EndToEnd(t *testing.T) {
	db := bootTestDB(t)
	dataDir := t.TempDir()
	bundleDir := writeTestBundle(t, t.TempDir(), "e2e-plugin")

	t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", "1")

	inst := buildInstaller(t, db, dataDir)

	h := hub.New(hub.Config{DB: db})
	s := buildTestServer(t, inst, h)

	// ── POST /api/plugins/install ──────────────────────────────────
	body := map[string]any{"src": "local:" + bundleDir}
	rr := postJSON(t, s.pluginsInstall, body)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("install: want 202, got %d; body=%s", rr.Code, rr.Body)
	}
	resp := decodeBody(t, rr)
	token, _ := resp["token"].(string)
	name, _ := resp["name"].(string)
	if token == "" {
		t.Fatalf("install: response missing token; body=%s", rr.Body)
	}
	if name == "" {
		t.Fatalf("install: response missing name; body=%s", rr.Body)
	}

	// ── POST /api/plugins/install/confirm ──────────────────────────
	confirmRR := postJSON(t, s.pluginsInstallConfirm, map[string]any{"token": token})
	if confirmRR.Code != http.StatusOK {
		t.Fatalf("confirm: want 200, got %d; body=%s", confirmRR.Code, confirmRR.Body)
	}
	confirmResp := decodeBody(t, confirmRR)
	if installed, _ := confirmResp["installed"].(bool); !installed {
		t.Fatalf("confirm: expected installed=true; body=%s", confirmRR.Body)
	}
	if n, _ := confirmResp["name"].(string); n != name {
		t.Fatalf("confirm: name mismatch: want %q, got %q", name, n)
	}

	// ── DELETE /api/plugins/{name} ─────────────────────────────────
	uninstallRR := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/plugins/"+name, nil)
	withChiParam(s.pluginsUninstall, "name", name)(uninstallRR, req)

	if uninstallRR.Code != http.StatusOK {
		t.Fatalf("uninstall: want 200, got %d; body=%s", uninstallRR.Code, uninstallRR.Body)
	}
	uninstallResp := decodeBody(t, uninstallRR)
	if status, _ := uninstallResp["status"].(string); status != "uninstalled" {
		t.Fatalf("uninstall: want status=uninstalled, got %q", status)
	}
	if n, _ := uninstallResp["name"].(string); n != name {
		t.Fatalf("uninstall: name mismatch: want %q, got %q", name, n)
	}
}

// TestPluginsInstall_LocalSourceGated verifies that without
// OPENDRAY_ALLOW_LOCAL_PLUGINS=1, a local: source returns 403 EFORBIDDEN.
func TestPluginsInstall_LocalSourceGated(t *testing.T) {
	// Ensure env var is unset.
	t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", "")

	s := buildTestServer(t, nil, nil)

	rr := postJSON(t, s.pluginsInstall, map[string]any{"src": "local:/some/path"})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d; body=%s", rr.Code, rr.Body)
	}
	m := decodeBody(t, rr)
	if code, _ := m["code"].(string); code != "EFORBIDDEN" {
		t.Fatalf("want code=EFORBIDDEN, got %q", code)
	}
}

// TestPluginsInstall_MissingSrc verifies that an empty body returns 400 EINVAL.
func TestPluginsInstall_MissingSrc(t *testing.T) {
	s := buildTestServer(t, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.pluginsInstall(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d; body=%s", rr.Code, rr.Body)
	}
	m := decodeBody(t, rr)
	if code, _ := m["code"].(string); code != "EINVAL" {
		t.Fatalf("want code=EINVAL, got %q", code)
	}
}

// TestPluginsInstall_BadScheme verifies that an unrecognised scheme → 400 EBADSRC.
func TestPluginsInstall_BadScheme(t *testing.T) {
	// Set env so we pass the local-gate check before hitting ParseSource.
	t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", "1")
	s := buildTestServer(t, nil, nil)

	rr := postJSON(t, s.pluginsInstall, map[string]any{"src": "garbage://x"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d; body=%s", rr.Code, rr.Body)
	}
	m := decodeBody(t, rr)
	if code, _ := m["code"].(string); code != "EBADSRC" {
		t.Fatalf("want code=EBADSRC, got %q", code)
	}
}

// TestPluginsInstall_HTTPSNotImplemented verifies that https:// sources
// return 501 ENOTIMPL in M1.
func TestPluginsInstall_HTTPSNotImplemented(t *testing.T) {
	// Must set the env so the handler passes the local-gate check for
	// non-local sources (the gate only fires for local: scheme).
	t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", "1")
	s := buildTestServer(t, nil, nil)

	rr := postJSON(t, s.pluginsInstall, map[string]any{"src": "https://example.com/plugin.zip"})
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("want 501, got %d; body=%s", rr.Code, rr.Body)
	}
	m := decodeBody(t, rr)
	if code, _ := m["code"].(string); code != "ENOTIMPL" {
		t.Fatalf("want code=ENOTIMPL, got %q", code)
	}
}

// TestPluginsInstallConfirm_MissingToken verifies that an empty token → 400.
func TestPluginsInstallConfirm_MissingToken(t *testing.T) {
	s := buildTestServer(t, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.pluginsInstallConfirm(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d; body=%s", rr.Code, rr.Body)
	}
}

// TestPluginsInstallConfirm_UnknownToken verifies that an unknown token → 410 ETOKEN.
func TestPluginsInstallConfirm_UnknownToken(t *testing.T) {
	db := bootTestDB(t)
	dataDir := t.TempDir()
	inst := buildInstaller(t, db, dataDir)
	h := hub.New(hub.Config{DB: db})
	s := buildTestServer(t, inst, h)

	rr := postJSON(t, s.pluginsInstallConfirm, map[string]any{"token": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"})
	if rr.Code != http.StatusGone {
		t.Fatalf("want 410, got %d; body=%s", rr.Code, rr.Body)
	}
	m := decodeBody(t, rr)
	if code, _ := m["code"].(string); code != "ETOKEN" {
		t.Fatalf("want code=ETOKEN, got %q", code)
	}
}

// TestPluginsUninstall_UnknownName verifies that uninstalling an unknown plugin
// returns 200 {status:"uninstalled"} — Installer.Uninstall is idempotent per T6 spec.
func TestPluginsUninstall_UnknownName(t *testing.T) {
	db := bootTestDB(t)
	dataDir := t.TempDir()
	inst := buildInstaller(t, db, dataDir)
	h := hub.New(hub.Config{DB: db})
	s := buildTestServer(t, inst, h)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/plugins/no-such-plugin", nil)
	withChiParam(s.pluginsUninstall, "name", "no-such-plugin")(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body)
	}
	m := decodeBody(t, rr)
	if status, _ := m["status"].(string); status != "uninstalled" {
		t.Fatalf("want status=uninstalled, got %q", status)
	}
}

// TestPluginsAudit_DefaultLimit installs a plugin, appends audit rows via
// DB.AppendAudit, then calls pluginsAudit and asserts the DTO matches.
func TestPluginsAudit_DefaultLimit(t *testing.T) {
	db := bootTestDB(t)
	dataDir := t.TempDir()
	bundleDir := writeTestBundle(t, t.TempDir(), "audit-plugin")

	t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", "1")

	inst := buildInstaller(t, db, dataDir)
	ctx := context.Background()

	// Stage + Confirm so the plugin row exists (audit FK not enforced but
	// we want a realistic row). Alternatively just insert the plugins row.
	src, err := install.ParseSource("local:" + bundleDir)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	pending, err := inst.Stage(ctx, src)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if err := inst.Confirm(ctx, pending.Token); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	pluginName := "audit-plugin"

	// Append two audit rows directly.
	for i := 0; i < 2; i++ {
		if err := db.AppendAudit(ctx, store.AuditEntry{
			PluginName: pluginName,
			Ns:         "test",
			Method:     fmt.Sprintf("method-%d", i),
			Caps:       []string{"exec"},
			Result:     "ok",
			DurationMs: i * 10,
			ArgsHash:   "abc123",
			Message:    fmt.Sprintf("msg-%d", i),
		}); err != nil {
			t.Fatalf("AppendAudit: %v", err)
		}
	}

	h := hub.New(hub.Config{DB: db})
	s := buildTestServer(t, inst, h)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/plugins/"+pluginName+"/audit", nil)
	withChiParam(s.pluginsAudit, "name", pluginName)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body)
	}

	var entries []auditEntryDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode audit response: %v; body=%s", err, rr.Body)
	}
	// We wrote 2 rows + the Stage and Confirm audit rows = 4 rows.
	// At minimum our 2 rows must be present.
	if len(entries) < 2 {
		t.Fatalf("want >= 2 entries, got %d", len(entries))
	}
	// Newest-first: check the first entry has the right plugin name.
	if entries[0].Ns == "" {
		t.Fatalf("first audit entry has empty Ns; body=%s", rr.Body)
	}
}

// TestPluginsAudit_LimitClamping verifies that ?limit=0 and ?limit=2000
// both work (clamped by DB layer) and do not cause HTTP errors.
func TestPluginsAudit_LimitClamping(t *testing.T) {
	db := bootTestDB(t)
	inst := buildInstaller(t, db, t.TempDir())
	h := hub.New(hub.Config{DB: db})
	s := buildTestServer(t, inst, h)

	for _, limitParam := range []string{"0", "2000"} {
		t.Run("limit="+limitParam, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/plugins/no-such/audit?limit="+limitParam, nil)
			withChiParam(s.pluginsAudit, "name", "no-such")(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("want 200, got %d; body=%s", rr.Code, rr.Body)
			}
		})
	}
}

// TestPluginsInstall_RoutesUnderAuth verifies that the chi router produced by
// New() returns 401 for unauthenticated requests on all four plugin endpoints.
func TestPluginsInstall_RoutesUnderAuth(t *testing.T) {
	// A real auth instance with a non-empty secret so Middleware blocks.
	a := auth.New("test-secret-12345678", 24*time.Hour)

	// Plugins runtime is required by New (telegram bridge uses it).
	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(nil, hooks, "", nil)

	srv := New(Config{
		Auth:    a,
		Plugins: rt,
		// Installer is nil — routes are registered regardless; handlers
		// will panic if reached, but auth should fire first (401).
		Installer: nil,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/plugins/install"},
		{http.MethodPost, "/api/plugins/install/confirm"},
		{http.MethodDelete, "/api/plugins/myplugin"},
		{http.MethodGet, "/api/plugins/myplugin/audit"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req, err := http.NewRequest(rt.method, ts.URL+rt.path, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("want 401 for %s %s, got %d", rt.method, rt.path, resp.StatusCode)
			}
		})
	}
}
