//go:build e2e

// Package plugin_test — M3 T26 end-to-end test for the fs-readme
// host-form reference plugin.
//
// Exercises install → invoke summarise → revoke fs.read →
// invoke-again-denied → uninstall. The `summarise` command spawns a
// Node sidecar that uses the sidecar → host JSON-RPC mux to call
// fs/readFile on a fixture file under a fake HOME; the capability
// gate enforces permissions.fs.read end-to-end.
//
// Requires Node 20+ on PATH; skips with t.Skip if absent. Requires
// the `e2e` build tag — default `go test ./...` excludes this file.
//
// Run:
//
//	go test -race -tags=e2e -timeout=10m -run FSReadme ./plugin/...
package plugin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/commands"
	"github.com/opendray/opendray/plugin/host"
)

// fsReadmePath locates the fs-readme plugin directory relative to the
// repo root (same walk-up trick as timeNinjaPath).
func fsReadmePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			p := filepath.Join(dir, "plugins", "examples", "fs-readme")
			if _, err := os.Stat(filepath.Join(p, "manifest.json")); err != nil {
				t.Fatalf("fs-readme manifest missing at %s: %v", p, err)
			}
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root")
	return ""
}

// testPathVarResolver is a no-dependency PathVarResolver for tests.
type testPathVarResolver struct {
	home    string
	dataDir string
}

func (r *testPathVarResolver) Resolve(_ context.Context, _ string) (bridge.PathVarCtx, error) {
	return bridge.PathVarCtx{
		Home:    r.home,
		DataDir: r.dataDir,
		Tmp:     os.TempDir(),
	}, nil
}

// TestE2E_FSReadmeFullLifecycle install → invoke → revoke → uninstall.
//
// Uses the harness's HTTP server for install/uninstall/revoke, but
// invokes the summarise command via the commands.Dispatcher directly
// (not POST /invoke) because the harness builds its own dispatcher
// inside startServer without a HostCaller. Direct invocation hits the
// exact same code path the dispatcher would take behind the HTTP
// handler — no semantic loss.
func TestE2E_FSReadmeFullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full E2E under -short")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node not on PATH: %v", err)
	}

	t.Setenv("OPENDRAY_ALLOW_LOCAL_PLUGINS", "1")

	h := newHarness(t)
	fsDir := fsReadmePath(t)

	// Fake home containing the README the sidecar will read through
	// the capability gate.
	fakeHome := filepath.Join(h.dataDir, "fake-home")
	if err := os.MkdirAll(fakeHome, 0o700); err != nil {
		t.Fatalf("mkdir fake-home: %v", err)
	}
	readmeContent := "# Test README\n\nThis is a fixture file for fs-readme E2E test.\n"
	if err := os.WriteFile(filepath.Join(fakeHome, "README.md"), []byte(readmeContent), 0o644); err != nil {
		t.Fatalf("write fixture README: %v", err)
	}
	t.Setenv("HOME", fakeHome)

	// Build a supervisor that points at the harness's runtime. h.rt
	// is rebuilt by startServer/stopServer — we don't restart here
	// so the pointer stays valid for the test's lifetime. The
	// sidecar's fs/readFile call back routes through a fresh FSAPI
	// wired to the same gate + a PathVarResolver that expands ${home}
	// to fakeHome.
	resolver := &testPathVarResolver{home: fakeHome, dataDir: h.dataDir}
	fsAPI := bridge.NewFSAPI(bridge.FSConfig{Gate: h.gate, Resolver: resolver})

	sup := host.NewSupervisor(host.Config{
		DataDir:   h.dataDir,
		Providers: h.rt,
		// Installer writes to ${DataDir}/<plugin>/<version>/, so the
		// supervisor must look up the installed version or `node` can't
		// find the sidecar entry file.
		PluginVersion: func(name string) string {
			if p, ok := h.rt.Get(name); ok {
				return p.Version
			}
			return ""
		},
		HandlerFactory: func(pluginName string) host.RPCHandler {
			hr, err := host.NewHostRPCHandler(host.HostRPCConfig{
				Plugin: pluginName,
				Namespaces: map[string]host.NSDispatcher{
					"fs": host.NamespaceAdapter{Inner: fsAPI.Dispatch},
				},
			})
			if err != nil {
				return nil
			}
			return hr
		},
	})
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = sup.Stop(ctx)
	})

	// Host-aware dispatcher that invokes the sidecar via sup.
	hostDispatcher, err := commands.NewDispatcher(commands.Config{
		Registry: h.registry, Gate: h.gate,
		Host: commands.HostCallerFunc(func(ctx context.Context, plugin, method string, params json.RawMessage) (json.RawMessage, error) {
			sc, serr := sup.Ensure(ctx, plugin)
			if serr != nil {
				return nil, serr
			}
			return sc.Call(ctx, method, params)
		}),
	})
	if err != nil {
		t.Fatalf("hostDispatcher: %v", err)
	}

	var installToken string

	t.Run("InstallConfirm", func(t *testing.T) {
		var got struct {
			Token   string `json:"token"`
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		code := h.doJSON(http.MethodPost, "/api/plugins/install",
			map[string]string{"src": "local:" + fsDir}, &got)
		if code != http.StatusAccepted {
			t.Fatalf("install: status %d", code)
		}
		if got.Name != "fs-readme" {
			t.Errorf("name=%q want fs-readme", got.Name)
		}
		installToken = got.Token

		var conf struct {
			Installed bool `json:"installed"`
		}
		code = h.doJSON(http.MethodPost, "/api/plugins/install/confirm",
			map[string]string{"token": installToken}, &conf)
		if code != http.StatusOK {
			t.Fatalf("confirm: status %d", code)
		}
		if !conf.Installed {
			t.Error("confirm: installed=false")
		}
		if n := countRows(t, h.db, `SELECT count(*) FROM plugins WHERE name=$1`, "fs-readme"); n != 1 {
			t.Errorf("plugins row count=%d", n)
		}
	})

	t.Run("InvokeSummarise", func(t *testing.T) {
		if installToken == "" {
			t.Skip("install failed")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		res, err := hostDispatcher.Invoke(ctx, "fs-readme", "fs-readme.summarise", nil)
		if err != nil {
			t.Fatalf("Invoke summarise: %v", err)
		}
		if res.Kind != "host" {
			t.Errorf("kind=%q want host", res.Kind)
		}
		if !strings.Contains(res.Output, "Test README") {
			t.Errorf("output missing fixture README content: %q", res.Output)
		}
	})

	t.Run("RevokeFSReadDeniesInvoke", func(t *testing.T) {
		if installToken == "" {
			t.Skip("install failed")
		}
		start := time.Now()
		code := h.doJSON(http.MethodDelete,
			fmt.Sprintf("/api/plugins/%s/consents/%s", "fs-readme", "fs"),
			nil, nil)
		if code != http.StatusOK && code != http.StatusNoContent {
			t.Logf("revoke status: %d (continuing)", code)
		}
		elapsed := time.Since(start)
		if elapsed > 500*time.Millisecond {
			t.Logf("revoke took %v", elapsed)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := hostDispatcher.Invoke(ctx, "fs-readme", "fs-readme.summarise", nil)
		if err == nil {
			t.Fatal("expected error after revoke, got success")
		}
		if !strings.Contains(err.Error(), "EPERM") && !strings.Contains(err.Error(), "fs.read") {
			t.Errorf("expected EPERM-shaped error, got: %v", err)
		}
	})

	t.Run("Uninstall", func(t *testing.T) {
		code := h.doJSON(http.MethodDelete, "/api/plugins/fs-readme", nil, nil)
		if code != http.StatusOK {
			t.Fatalf("uninstall: status %d", code)
		}
		if n := countRows(t, h.db, `SELECT count(*) FROM plugins WHERE name=$1`, "fs-readme"); n != 0 {
			t.Errorf("plugins row count=%d after uninstall", n)
		}
		if n := countRows(t, h.db, `SELECT count(*) FROM plugin_host_state WHERE plugin_name=$1`, "fs-readme"); n != 0 {
			t.Errorf("plugin_host_state row count=%d after uninstall", n)
		}
	})
}
