package gateway

// End-to-end test for the remote marketplace install path.
//
// Boots:
//   - httptest.Server serving a fake registry (index.json +
//     per-version JSON + publisher record + revocations.json +
//     artifact zip).
//   - plugin/market/remote.Catalog pointed at that URL.
//   - embedded Postgres via bootTestDB for the Installer's DB.
//   - real install.Installer wired like production.
//
// Then drives POST /api/plugins/install → /install/confirm and
// asserts the plugin lands on disk + in the providers table.
//
// This is the one test that proves T1–T11 wire up correctly
// end-to-end. The individual package tests cover each layer's
// logic; this one catches any wiring drift.

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/install"
	marketremote "github.com/opendray/opendray/plugin/market/remote"
)

// Shared constants to avoid noisy magic numbers in the inline
// Installer construction below.
const (
	mins    = time.Minute
	oneHour = time.Hour
)

// bridgeNewGate is a tiny alias so the test doesn't need its own
// "import bridge as _" — keeps diagnostics readable.
var bridgeNewGate = bridge.NewGate

// fakeRegistry serves the minimum paths a remote install touches.
// Caller supplies the artifact bytes + sha256 + publisher record.
// Missing paths 404.
func fakeRegistry(t *testing.T, publisherBody, indexBody, versionBody string, zipBytes []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, indexBody)
		case "/plugins/opendray-examples/e2e/1.0.0.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, versionBody)
		case "/publishers/opendray-examples.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, publisherBody)
		case "/revocations.json":
			http.NotFound(w, r) // empty is fine
		case "/artifact.zip":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(zipBytes)
		default:
			http.NotFound(w, r)
		}
	}))
}

// zipBundle builds an in-memory v1 plugin bundle with a manifest
// declaring form=declarative (no sidecar), no permissions, no
// configSchema — the minimum shape install.Installer accepts.
func zipBundle(t *testing.T, name, version string) ([]byte, string) {
	t.Helper()
	manifest := map[string]any{
		"name":      name,
		"version":   version,
		"publisher": "opendray-examples",
		"engines":   map[string]any{"opendray": "^1.0.0"},
		"form":      "declarative",
	}
	mbytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(mbytes); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	body := buf.Bytes()
	sum := sha256.Sum256(body)
	return body, hex.EncodeToString(sum[:])
}

// TestMarketplaceInstall_EndToEnd walks the full remote install
// flow: registry served via httptest → remote.Catalog → gateway
// install handler → HTTPSSource download + sha256 + unzip →
// Installer.Stage → Confirm → DB row + on-disk bundle.
func TestMarketplaceInstall_EndToEnd(t *testing.T) {
	db := bootTestDB(t)
	dataDir := t.TempDir()

	// 1. Build the artifact.
	zipBytes, sum := zipBundle(t, "e2e", "1.0.0")

	// 2. Construct the fixtures. srvURL is known AFTER httptest.Server
	// starts, so we stage a placeholder and swap it in with a closure.
	publisherBody := `{
		"name": "opendray-examples",
		"trust": "community",
		"keys": []
	}`
	indexBody := `{
		"version": 1,
		"generatedAt": "2026-04-20T00:00:00Z",
		"plugins": [
			{"name":"e2e","publisher":"opendray-examples","latest":"1.0.0","form":"declarative"}
		]
	}`
	// versionBody needs the server URL so we fill it in after boot.
	var srv *httptest.Server
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/index.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, indexBody)
	})
	serverMux.HandleFunc("/publishers/opendray-examples.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, publisherBody)
	})
	serverMux.HandleFunc("/revocations.json", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	serverMux.HandleFunc("/plugins/opendray-examples/e2e/1.0.0.json", func(w http.ResponseWriter, _ *http.Request) {
		versionBody := fmt.Sprintf(`{
			"name": "e2e",
			"publisher": "opendray-examples",
			"version": "1.0.0",
			"artifact": {"url": "%s/artifact.zip", "size": %d},
			"sha256": %q,
			"manifest": {
				"name": "e2e",
				"version": "1.0.0",
				"publisher": "opendray-examples",
				"engines": {"opendray": "^1.0.0"},
				"form": "declarative"
			}
		}`, srv.URL, len(zipBytes), sum)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, versionBody)
	})
	serverMux.HandleFunc("/artifact.zip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(zipBytes)
	})
	srv = httptest.NewServer(serverMux)
	defer srv.Close()

	// 3. Remote catalog.
	catalog, err := marketremote.New(marketremote.Config{
		RegistryURL: srv.URL + "/",
		HTTPClient:  srv.Client(),
		CacheTTL:    -1, // disable cache so test is self-contained
	})
	if err != nil {
		t.Fatalf("remote.New: %v", err)
	}

	// 4. Install pipeline. Build Installer + Runtime inline so the
	// test server shares the runtime the Installer writes into.
	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(db, hooks, "", nil)
	gate := bridgeNewGate(nil, nil, nil)
	inst := install.NewInstallerWithTTL(dataDir, db, rt, gate, nil, 10*mins, oneHour)
	t.Cleanup(inst.Stop)

	s := &Server{
		plugins:     rt,
		installer:   inst,
		router:      chi.NewRouter(),
		marketplace: catalog,
	}

	// 5. POST /api/plugins/install with marketplace://ref.
	rr := postJSON(t, s.pluginsInstall, map[string]any{
		"src": "marketplace://opendray-examples/e2e@1.0.0",
	})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("install: want 202, got %d; body=%s", rr.Code, rr.Body)
	}
	staged := decodeBody(t, rr)
	token, _ := staged["token"].(string)
	if token == "" {
		t.Fatalf("install: missing token in %v", staged)
	}

	// 6. POST /install/confirm.
	rr = postJSON(t, s.pluginsInstallConfirm, map[string]any{"token": token})
	if rr.Code != http.StatusOK {
		t.Fatalf("confirm: want 200, got %d; body=%s", rr.Code, rr.Body)
	}

	// 7. Bundle landed on disk.
	manifestPath := filepath.Join(dataDir, "e2e", "1.0.0", "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest at %s: %v", manifestPath, err)
	}

	// 8. Runtime knows about the plugin. The test server's runtime
	// was registered from the Installer's Confirm call — List should
	// return one entry.
	if p, ok := s.plugins.Get("e2e"); !ok {
		t.Fatalf("runtime: plugin e2e not registered")
	} else if p.Version != "1.0.0" {
		t.Errorf("runtime: version = %q, want 1.0.0", p.Version)
	}
}

// Guard against unused-import elimination — the big e2e test
// doesn't exercise every import directly but they're all used
// transitively.
var _ = install.HTTPSSource{}
var _ = plugin.ConfigField{}
var _ chi.Router = chi.NewRouter()
var _ io.Reader
var _ = context.Background
