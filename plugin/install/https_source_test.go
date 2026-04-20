package install

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildZip constructs an in-memory zip with the given name→content
// entries. Helper keeps test bodies focused on behaviour.
func buildZip(t *testing.T, entries map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	bytes := buf.Bytes()
	sum := sha256.Sum256(bytes)
	return bytes, hex.EncodeToString(sum[:])
}

// serveBytes serves the given bytes at /bundle.zip (any other path
// 404s). Returns the server URL + cleanup attached to t.
func serveBytes(t *testing.T, body []byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bundle.zip" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/bundle.zip"
}

// ─── HTTPSSource.Fetch — happy ─────────────────────────────────────────────

func TestHTTPS_Fetch_HappyPath(t *testing.T) {
	zipBytes, sum := buildZip(t, map[string]string{
		"manifest.json": `{"name":"fake"}`,
		"README.md":     "hello",
		"ui/index.html": "<html></html>",
	})
	url := serveBytes(t, zipBytes)

	src := HTTPSSource{URL: url, ExpectedSHA256: sum}
	dir, cleanup, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer cleanup()

	// manifest.json lands at the root (spec §Required contents).
	manifest, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if string(manifest) != `{"name":"fake"}` {
		t.Errorf("manifest contents = %q", manifest)
	}

	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil || string(readme) != "hello" {
		t.Errorf("README = %q, err=%v", readme, err)
	}

	// Nested directory extraction.
	html, err := os.ReadFile(filepath.Join(dir, "ui", "index.html"))
	if err != nil || string(html) != "<html></html>" {
		t.Errorf("ui/index.html = %q err=%v", html, err)
	}
}

// ─── HTTPSSource.Fetch — validation errors ─────────────────────────────────

func TestHTTPS_Fetch_MissingURL(t *testing.T) {
	_, _, err := HTTPSSource{ExpectedSHA256: "x"}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "URL is empty") {
		t.Errorf("err = %v; want URL empty", err)
	}
}

func TestHTTPS_Fetch_MissingSHA256(t *testing.T) {
	_, _, err := HTTPSSource{URL: "http://x/y.zip"}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "ExpectedSHA256") {
		t.Errorf("err = %v; want ExpectedSHA256 required", err)
	}
}

func TestHTTPS_Fetch_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, _, err := HTTPSSource{URL: srv.URL, ExpectedSHA256: "a"}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Errorf("err = %v; want HTTP 503", err)
	}
}

func TestHTTPS_Fetch_SHAMismatch(t *testing.T) {
	zipBytes, _ := buildZip(t, map[string]string{"manifest.json": "{}"})
	url := serveBytes(t, zipBytes)

	wrongSum := strings.Repeat("f", 64)
	_, _, err := HTTPSSource{URL: url, ExpectedSHA256: wrongSum}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("err = %v; want sha256 mismatch", err)
	}
}

func TestHTTPS_Fetch_SizeMismatchEarly(t *testing.T) {
	// Server advertises 1024 via Content-Length but ExpectedSize
	// says 50. Early-reject before streaming.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1024")
		_, _ = w.Write(bytes.Repeat([]byte("x"), 1024))
	}))
	defer srv.Close()
	_, _, err := HTTPSSource{
		URL:            srv.URL,
		ExpectedSHA256: strings.Repeat("a", 64),
		ExpectedSize:   50,
	}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "size") {
		t.Errorf("err = %v; want size mismatch", err)
	}
}

func TestHTTPS_Fetch_ExceedsMaxBytes(t *testing.T) {
	// Serve 2 KiB under a 1 KiB cap. Download abort before hashing.
	bigZip := bytes.Repeat([]byte("x"), 2<<10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bigZip)
	}))
	defer srv.Close()
	_, _, err := HTTPSSource{
		URL:            srv.URL,
		ExpectedSHA256: strings.Repeat("a", 64),
		MaxBytes:       1 << 10,
	}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "exceeds MaxBytes") {
		t.Errorf("err = %v; want MaxBytes exceed", err)
	}
}

// ─── extractZipBundle — security posture ──────────────────────────────────

func TestExtract_RejectsPathTraversal(t *testing.T) {
	zipBytes, sum := buildZip(t, map[string]string{
		"manifest.json":  "{}",
		"../evil":        "pwn",
	})
	url := serveBytes(t, zipBytes)
	_, _, err := HTTPSSource{URL: url, ExpectedSHA256: sum}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "escape") {
		t.Errorf("err = %v; want escape reject", err)
	}
}

func TestExtract_RejectsAbsolutePath(t *testing.T) {
	// archive/zip's Writer will refuse to write an entry starting
	// with "/" — build the raw header ourselves via a deliberately
	// malformed entry name.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "/etc/passwd", Method: zip.Deflate}
	fw, err := w.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("root:x:0")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	url := serveBytes(t, buf.Bytes())
	_, _, err = HTTPSSource{URL: url, ExpectedSHA256: hex.EncodeToString(sum[:])}.Fetch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Errorf("err = %v; want absolute-path reject", err)
	}
}

func TestExtract_StripsSetuid(t *testing.T) {
	// Build a zip with a file carrying setuid mode. After extract
	// the setuid bit MUST be gone.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "bin/tool", Method: zip.Deflate}
	// setuid + rwxr-xr-x
	h.SetMode(os.ModeSetuid | 0o755)
	fw, err := w.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("elf-stub")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	url := serveBytes(t, buf.Bytes())
	dir, cleanup, err := HTTPSSource{URL: url, ExpectedSHA256: hex.EncodeToString(sum[:])}.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer cleanup()

	fi, err := os.Stat(filepath.Join(dir, "bin", "tool"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode()&os.ModeSetuid != 0 {
		t.Errorf("setuid still set: %v", fi.Mode())
	}
	// But executable bit stays.
	if fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("executable bit stripped: %v", fi.Mode())
	}
}

// ─── Sanity: AllowLocal gate doesn't trip on HTTPSSource ───────────────────

func TestInstallerStage_DoesNotGateHTTPSSource(t *testing.T) {
	// The existing install tests cover the LocalSource gating; this
	// is a regression sentinel proving HTTPSSource is not caught by
	// that type assertion. We don't build a full Installer here —
	// just assert the source isn't a LocalSource.
	var s Source = HTTPSSource{URL: "http://x/y.zip", ExpectedSHA256: "a"}
	if _, ok := s.(LocalSource); ok {
		t.Fatal("HTTPSSource should not satisfy LocalSource assertion")
	}

	// And confirm the sentinel we export for not-implemented paths
	// is still importable (other callers use errors.Is).
	var e error = ErrNotImplemented
	if !errors.Is(e, ErrNotImplemented) {
		t.Error("ErrNotImplemented sentinel broken")
	}
	_ = fmt.Sprintf("%v", e) // use fmt to avoid unused import if err check short-circuits
}
