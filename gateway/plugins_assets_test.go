package gateway

// T8 — Asset handler test suite.
//
// Test strategy:
//   - All tests construct a chi router wired to a closure that calls
//     assetsHandler (the free function in plugins_assets.go) with a
//     fakeVersioner and a temp data directory. This avoids any DB dependency.
//   - TestAssets_AuthRequired uses gateway.New() so the JWT middleware is
//     exercised end-to-end (same pattern as T7's RoutesUnderAuth test).
//
// File layout used across all happy-path tests:
//
//	${tmp}/
//	  k/
//	    1.0.0/
//	      ui/
//	        index.html    → "<!doctype html><title>hi</title>"
//	        main.js       → "console.log('hi')"
//	        styles.css    → "body{color:red}"
//	        image.png     → PNG magic bytes
//	        data.json     → `{"key":"value"}`
//	        icon.svg      → "<svg/>"
//	        unknown.weird → "weird content"

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/auth"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
	"github.com/opendray/opendray/plugin/install"
)

// ─── Constants ───────────────────────────────────────────────────────────────

const (
	testPluginName    = "k"
	testPluginVersion = "1.0.0"

	contentIndexHTML = "<!doctype html><title>hi</title>"
	contentMainJS    = "console.log('hi')"
	contentCSS       = "body{color:red}"
	contentJSON      = `{"key":"value"}`
	contentSVG       = "<svg/>"
	contentWeird     = "weird content"
)

// cspGolden is the EXACT Content-Security-Policy value from M2-PLAN §T8.
// Any character-level divergence in the handler is a bug.
const cspGolden = "default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'"

// ─── Fake versioner ───────────────────────────────────────────────────────────

// fakeVersioner implements the pluginVersioner interface (defined in
// plugins_assets.go). It serves a fixed map without any DB.
type fakeVersioner struct {
	providers map[string]plugin.Provider
}

func newFakeVersioner(providers map[string]plugin.Provider) *fakeVersioner {
	return &fakeVersioner{providers: providers}
}

func (f *fakeVersioner) Get(name string) (plugin.Provider, bool) {
	p, ok := f.providers[name]
	return p, ok
}

// testVersioner returns a fakeVersioner pre-seeded with plugin "k" v1.0.0.
func testVersioner() *fakeVersioner {
	return newFakeVersioner(map[string]plugin.Provider{
		testPluginName: {
			Name:    testPluginName,
			Version: testPluginVersion,
			Type:    "panel",
		},
	})
}

// ─── Fixtures ─────────────────────────────────────────────────────────────────

// makeTestTree writes the plugin ui/ directory tree inside a new temp dir
// and returns the plugins data root (the dir passed as Installer.DataDir).
func makeTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	uiDir := filepath.Join(root, testPluginName, testPluginVersion, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("makeTestTree: mkdir: %v", err)
	}
	files := map[string]string{
		"index.html":    contentIndexHTML,
		"main.js":       contentMainJS,
		"styles.css":    contentCSS,
		"image.png":     "\x89PNG\r\n\x1a\n",
		"data.json":     contentJSON,
		"icon.svg":      contentSVG,
		"unknown.weird": contentWeird,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(uiDir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("makeTestTree: write %s: %v", name, err)
		}
	}
	return root
}

// ─── Router helpers ───────────────────────────────────────────────────────────

// buildAssetsRouter creates a chi router that dispatches
// GET /api/plugins/{name}/assets/* to assetsHandler with the given
// pluginVersioner and data directory. No *Server or database required.
func buildAssetsRouter(fv pluginVersioner, dataDir string) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/plugins/{name}/assets/*", func(w http.ResponseWriter, req *http.Request) {
		assetsHandler(w, req, fv, dataDir)
	})
	return r
}

// getAsset is a test helper that fires a GET against the router.
func getAsset(router http.Handler, urlPath string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, urlPath, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// decodeJSON unmarshals the recorder body into map[string]any.
func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("decodeJSON: %v; body=%s", err, rr.Body)
	}
	return m
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// 1. TestAssets_HappyHTML — GET index.html → 200 + correct body/Content-Type/CSP.
func TestAssets_HappyHTML(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/index.html")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	if body := rr.Body.String(); body != contentIndexHTML {
		t.Errorf("body mismatch:\nwant: %q\ngot:  %q", contentIndexHTML, body)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type: want text/html..., got %q", ct)
	}
	csp := rr.Header().Get("Content-Security-Policy")
	if csp != cspGolden {
		t.Errorf("CSP mismatch:\nwant: %q\ngot:  %q", cspGolden, csp)
	}
}

// 2. TestAssets_DefaultsToIndexHtml — bare /assets/ → 200 serving index.html.
func TestAssets_DefaultsToIndexHtml(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	// chi captures "" for the wildcard when URL ends at /assets/
	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	if body := rr.Body.String(); body != contentIndexHTML {
		t.Errorf("body mismatch:\nwant: %q\ngot:  %q", contentIndexHTML, body)
	}
}

// 3. TestAssets_JSContentType — .js → application/javascript.
func TestAssets_JSContentType(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/main.js")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/javascript") {
		t.Errorf("Content-Type: want application/javascript, got %q", ct)
	}
}

// 4. TestAssets_CSSContentType — .css → text/css.
func TestAssets_CSSContentType(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/styles.css")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/css") {
		t.Errorf("Content-Type: want text/css, got %q", ct)
	}
}

// 5. TestAssets_UnknownExtensionFallback — .weird → application/octet-stream.
func TestAssets_UnknownExtensionFallback(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/unknown.weird")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("Content-Type: want application/octet-stream, got %q", ct)
	}
}

// 6. TestAssets_PathTraversalAttempts — table-driven, 12 attack strings.
//
// Outcome guide:
//   - 400 EBADPATH  → fragment is dangerous; handler rejects before touching FS
//   - 404 ENOFILE   → fragment is safe after Clean, but the file doesn't exist
//
// Note on newline/NUL cases: httptest.NewRequest rejects URLs containing
// control characters (same as real HTTP clients do). For these two cases we
// test validateAssetFragment directly rather than sending an HTTP request.
func TestAssets_PathTraversalAttempts(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	// httpCases are attack strings that can be encoded in a valid HTTP URL.
	httpCases := []struct {
		name     string
		fragment string
		want     int
		wantCode string
	}{
		// ── Rejected with 400 EBADPATH ────────────────────────────────────────
		{"double-dot traversal", "../../etc/passwd", http.StatusBadRequest, "EBADPATH"},
		{"url-encoded double-dot", "..%2fetc%2fpasswd", http.StatusBadRequest, "EBADPATH"},
		{"absolute path slash", "/etc/passwd", http.StatusBadRequest, "EBADPATH"},
		{"deep traversal ui subdir", "ui/../../../../etc/passwd", http.StatusBadRequest, "EBADPATH"},
		{"dot-slash traversal", "./../etc/passwd", http.StatusBadRequest, "EBADPATH"},
		{"windows backslash traversal", `\\..\\.\\..\\etc\\passwd`, http.StatusBadRequest, "EBADPATH"},
		{"bare double-dot", "..", http.StatusBadRequest, "EBADPATH"},
		{"absolute path variant", "/absolute/path", http.StatusBadRequest, "EBADPATH"},
		{"subdir traversal outside", "subdir/../../../outside", http.StatusBadRequest, "EBADPATH"},
		// ── Safe after Clean — file absent → 404 ENOFILE ─────────────────────
		// a/b/../c → filepath.Clean → a/c; no ".." in result, valid subpath.
		{"clean subdir missing file", "a/b/../c", http.StatusNotFound, "ENOFILE"},
	}

	for _, tc := range httpCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/plugins/"+testPluginName+"/assets/"+tc.fragment, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tc.want {
				t.Errorf("fragment %q: want %d, got %d; body=%s",
					tc.fragment, tc.want, rr.Code, rr.Body)
				return
			}
			if tc.wantCode != "" {
				m := decodeJSON(t, rr)
				got, _ := m["code"].(string)
				if got != tc.wantCode {
					t.Errorf("fragment %q: want code=%q, got %q; body=%s",
						tc.fragment, tc.wantCode, got, rr.Body)
				}
			}
		})
	}

	// controlCases test fragments that carry control characters (NUL, CR, LF)
	// which httptest.NewRequest refuses to construct as HTTP requests. We test
	// these by calling validateAssetFragment directly — the path the handler
	// would take if such bytes somehow reached it (e.g. smuggled via a proxy).
	controlCases := []struct {
		name     string
		fragment string
	}{
		{"path with newline", "with\nnewline"},
		{"path with null byte", "with\x00null"},
	}
	for _, tc := range controlCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateAssetFragment(tc.fragment)
			if err == nil {
				t.Errorf("fragment %q: want validateAssetFragment to reject, got nil", tc.fragment)
			}
		})
	}
}

// 7. TestAssets_MissingPluginReturns404 — unknown plugin → 404 ENOPLUGIN.
func TestAssets_MissingPluginReturns404(t *testing.T) {
	dataDir := makeTestTree(t)
	// Empty versioner — no plugins registered.
	r := buildAssetsRouter(newFakeVersioner(map[string]plugin.Provider{}), dataDir)

	rr := getAsset(r, "/api/plugins/no-such-plugin/assets/index.html")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d; body=%q", rr.Code, rr.Body)
	}
	m := decodeJSON(t, rr)
	if code, _ := m["code"].(string); code != "ENOPLUGIN" {
		t.Errorf("want code=ENOPLUGIN, got %q; body=%s", code, rr.Body)
	}
}

// 8. TestAssets_MissingFileReturns404 — known plugin, nonexistent file → 404 ENOFILE.
func TestAssets_MissingFileReturns404(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/does-not-exist.html")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d; body=%q", rr.Code, rr.Body)
	}
	m := decodeJSON(t, rr)
	if code, _ := m["code"].(string); code != "ENOFILE" {
		t.Errorf("want code=ENOFILE, got %q; body=%s", code, rr.Body)
	}
}

// 9. TestAssets_CSPHeaderExact — byte-exact golden compare on CSP header.
func TestAssets_CSPHeaderExact(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/index.html")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	got := rr.Header().Get("Content-Security-Policy")
	if got != cspGolden {
		t.Errorf("CSP byte-exact mismatch:\nwant: %q\ngot:  %q", cspGolden, got)
	}
}

// 10. TestAssets_SecurityHeaders — X-Content-Type-Options + X-Frame-Options present.
func TestAssets_SecurityHeaders(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/main.js")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	if v := rr.Header().Get("X-Content-Type-Options"); v != "nosniff" {
		t.Errorf("X-Content-Type-Options: want nosniff, got %q", v)
	}
	if v := rr.Header().Get("X-Frame-Options"); v != "DENY" {
		t.Errorf("X-Frame-Options: want DENY, got %q", v)
	}
}

// 11. TestAssets_CacheControl — .html → no-store; .js/.css → public, max-age=3600.
func TestAssets_CacheControl(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	cases := []struct {
		file      string
		wantCache string
	}{
		{"index.html", "no-store"},
		{"main.js", "public, max-age=3600"},
		{"styles.css", "public, max-age=3600"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/"+tc.file)
			if rr.Code != http.StatusOK {
				t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
			}
			cc := rr.Header().Get("Cache-Control")
			if cc != tc.wantCache {
				t.Errorf("Cache-Control: want %q, got %q", tc.wantCache, cc)
			}
		})
	}
}

// 12. TestAssets_AuthRequired — no JWT → 401 via full gateway.New() pipeline.
func TestAssets_AuthRequired(t *testing.T) {
	a := auth.New("test-secret-12345678", 24*time.Hour)
	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(nil, hooks, "", nil)

	srv := New(Config{
		Auth:      a,
		Plugins:   rt,
		Installer: nil, // auth middleware fires before the handler
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet,
		ts.URL+"/api/plugins/kanban/assets/index.html", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 without JWT, got %d", resp.StatusCode)
	}
}

// ─── Additional content-type coverage ────────────────────────────────────────

// TestAssets_JSONContentType — .json → application/json.
func TestAssets_JSONContentType(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/data.json")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}
}

// TestAssets_SVGContentType — .svg → image/svg+xml.
func TestAssets_SVGContentType(t *testing.T) {
	dataDir := makeTestTree(t)
	r := buildAssetsRouter(testVersioner(), dataDir)

	rr := getAsset(r, "/api/plugins/"+testPluginName+"/assets/icon.svg")

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d; body=%q", rr.Code, rr.Body)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/svg+xml") {
		t.Errorf("Content-Type: want image/svg+xml, got %q", ct)
	}
}

// TestAssets_MethodWrapper exercises the pluginsAssets *Server method wrapper
// with a valid JWT via a full gateway.New() server + real temp files, covering
// the s.installer nil-guard branch and ensuring the method delegates correctly.
func TestAssets_MethodWrapper(t *testing.T) {
	dataDir := makeTestTree(t)

	a := auth.New("test-secret-wrapper-abc", 24*time.Hour)
	token, err := a.Issue("testuser")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	hooks := plugin.NewHookBus(nil)
	rt := plugin.NewRuntime(nil, hooks, "", nil)

	gate := bridge.NewGate(nil, nil, nil)
	inst := install.NewInstallerWithTTL(dataDir, nil, nil, gate, nil, 10*time.Minute, time.Hour)
	t.Cleanup(inst.Stop)

	srv := New(Config{
		Auth:      a,
		Plugins:   rt,
		Installer: inst,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// The runtime has no plugin registered so we expect 404 ENOPLUGIN —
	// but the important thing is the request gets past auth (not 401).
	req, err := http.NewRequest(http.MethodGet,
		ts.URL+"/api/plugins/kanban/assets/index.html", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	// Not 401 (auth passed); 404 ENOPLUGIN is expected (no plugin registered).
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("want non-401 with valid JWT, got 401")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 ENOPLUGIN (no plugin in runtime), got %d", resp.StatusCode)
	}
}
