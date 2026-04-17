package plugin

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeManifest(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	body := `{"name":"` + name + `","displayName":"` + name + `","version":"1.0.0","type":"cli"}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestScanPluginDir_Recursive(t *testing.T) {
	root := t.TempDir()

	writeManifest(t, filepath.Join(root, "agents", "claude"), "claude")
	writeManifest(t, filepath.Join(root, "agents", "codex"), "codex")
	writeManifest(t, filepath.Join(root, "agents", "qwen-code"), "qwen-code")
	writeManifest(t, filepath.Join(root, "panels", "file-browser"), "file-browser")
	writeManifest(t, filepath.Join(root, "flat"), "flat")

	// Should be skipped
	writeManifest(t, filepath.Join(root, "_template"), "tmpl")
	writeManifest(t, filepath.Join(root, ".hidden"), "hidden")

	// Nested plugin inside another plugin — must not descend
	writeManifest(t, filepath.Join(root, "agents", "claude", "sub"), "nested")

	providers, err := ScanPluginDir(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	got := make([]string, 0, len(providers))
	for _, p := range providers {
		got = append(got, p.Name)
	}
	sort.Strings(got)

	want := []string{"claude", "codex", "file-browser", "flat", "qwen-code"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestScanPluginDir_MissingRoot(t *testing.T) {
	providers, err := ScanPluginDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected nil error for missing root, got %v", err)
	}
	if providers != nil {
		t.Fatalf("expected nil providers, got %v", providers)
	}
}
