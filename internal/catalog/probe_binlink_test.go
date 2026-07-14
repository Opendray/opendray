package catalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// npmPrefix builds a realistic npm global layout under a temp dir and
// returns (npmRoot, binDir). `npm root -g` reports <prefix>/lib/node_modules;
// the executables live in <prefix>/bin.
func npmPrefix(t *testing.T) (root, bin string) {
	t.Helper()
	prefix := t.TempDir()
	root = filepath.Join(prefix, "lib", "node_modules")
	bin = filepath.Join(prefix, "bin")
	for _, d := range []string{root, bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root, bin
}

func TestUnmanagedBinLink_NothingInstalled(t *testing.T) {
	root, _ := npmPrefix(t)
	link, isFile := unmanagedBinLink(root, "grok")
	if link != "" || isFile {
		t.Errorf("empty bin dir must be clear for npm; got link=%q isFile=%v", link, isFile)
	}
}

func TestUnmanagedBinLink_NpmOwnedLinkIsLeftAlone(t *testing.T) {
	root, bin := npmPrefix(t)
	// What npm itself creates: bin/grok -> ../lib/node_modules/<pkg>/bin/grok
	target := filepath.Join(root, "@xai-official", "grok", "bin", "grok")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(bin, "grok")); err != nil {
		t.Fatal(err)
	}

	link, isFile := unmanagedBinLink(root, "grok")
	if link != "" || isFile {
		t.Errorf("npm-owned link must not be reported as unmanaged; got link=%q isFile=%v", link, isFile)
	}
}

// The grok case: `curl -fsSL https://x.ai/cli/install.sh | bash` drops a
// symlink into the npm bin dir that points at its own download tree. npm
// refuses to overwrite a bin entry it does not own (EEXIST), so the update
// fails and silently leaves the old binary in place.
func TestUnmanagedBinLink_VendorInstallerLinkIsReported(t *testing.T) {
	root, bin := npmPrefix(t)
	vendor := filepath.Join(t.TempDir(), "grok-downloads", "grok")
	if err := os.MkdirAll(filepath.Dir(vendor), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(vendor, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(bin, "grok")
	if err := os.Symlink(vendor, want); err != nil {
		t.Fatal(err)
	}

	link, isFile := unmanagedBinLink(root, "grok")
	if link != want {
		t.Errorf("vendor symlink not reported: got %q, want %q", link, want)
	}
	if isFile {
		t.Error("a symlink must not be reported as a regular file")
	}
}

func TestUnmanagedBinLink_DanglingLinkIsReplaceable(t *testing.T) {
	root, bin := npmPrefix(t)
	want := filepath.Join(bin, "grok")
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), want); err != nil {
		t.Fatal(err)
	}

	link, isFile := unmanagedBinLink(root, "grok")
	if link != want || isFile {
		t.Errorf("dangling link should be replaceable; got link=%q isFile=%v", link, isFile)
	}
}

// A real file (not a symlink) is something a human or another package
// manager put there deliberately. We must report it but never delete it.
func TestUnmanagedBinLink_RegularFileIsFlaggedNotDeletable(t *testing.T) {
	root, bin := npmPrefix(t)
	want := filepath.Join(bin, "grok")
	if err := os.WriteFile(want, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	link, isFile := unmanagedBinLink(root, "grok")
	if link != want {
		t.Errorf("regular file not reported: got %q, want %q", link, want)
	}
	if !isFile {
		t.Error("a regular file must be flagged as such so Update refuses to remove it")
	}
}

// End-to-end: Update must clear the vendor symlink so `npm install -g`
// doesn't die with EEXIST, and must tell the operator it did so.
func TestProber_UpdateReplacesVendorBinLink(t *testing.T) {
	root, bin := npmPrefix(t)
	vendor := filepath.Join(t.TempDir(), "grok")
	if err := os.WriteFile(vendor, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(bin, "grok")
	if err := os.Symlink(vendor, link); err != nil {
		t.Fatal(err)
	}

	installs := 0
	p := NewProber()
	p.lookPath = func(string) (string, error) { return link, nil }
	p.runVer = func(context.Context, string) (string, error) { return "grok 0.2.101 (5bc4b5dfad)", nil }
	p.npmRoot = func(context.Context) (string, error) { return root, nil }
	p.npmInstall = func(_ context.Context, pkg string) (string, error) {
		installs++
		if _, err := os.Lstat(link); !os.IsNotExist(err) {
			t.Error("the unmanaged bin link must be gone BEFORE npm install runs (else EEXIST)")
		}
		return "added 3 packages", nil
	}

	res, err := p.Update(context.Background(),
		Manifest{ID: "grok", Executable: "grok", NpmPackage: "@xai-official/grok"})
	if err != nil {
		t.Fatal(err)
	}
	if installs != 1 {
		t.Fatalf("expected one npm install, got %d", installs)
	}
	if !strings.Contains(res.Output, link) {
		t.Errorf("operator must be told which unmanaged link was replaced; output=%q", res.Output)
	}
}

func TestProber_UpdateRefusesToDeleteRegularBinFile(t *testing.T) {
	root, bin := npmPrefix(t)
	real := filepath.Join(bin, "grok")
	if err := os.WriteFile(real, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	installed := false
	p := NewProber()
	p.lookPath = func(string) (string, error) { return real, nil }
	p.runVer = func(context.Context, string) (string, error) { return "grok 0.2.101", nil }
	p.npmRoot = func(context.Context) (string, error) { return root, nil }
	p.npmInstall = func(context.Context, string) (string, error) { installed = true; return "", nil }

	_, err := p.Update(context.Background(),
		Manifest{ID: "grok", Executable: "grok", NpmPackage: "@xai-official/grok"})
	if err == nil {
		t.Fatal("Update must refuse when the bin entry is a real file it did not create")
	}
	if installed {
		t.Error("npm install must not run when we cannot clear the bin path")
	}
	if _, statErr := os.Stat(real); statErr != nil {
		t.Error("Update must never delete a regular file it did not create")
	}
}

// grok ships as @xai-official/grok on npm (maintainer xai-security@x.ai).
// Without the package name the whole update path is dead: CheckUpdate
// returns early and the dashboard can never offer an update.
func TestBuiltinGrokIsUpdatable(t *testing.T) {
	manifests, _, err := LoadBuiltin()
	if err != nil {
		t.Fatalf("LoadBuiltin: %v", err)
	}
	m, ok := manifests["grok"]
	if !ok {
		t.Fatal("grok manifest not found in builtins")
	}
	if m.NpmPackage != "@xai-official/grok" {
		t.Fatalf("grok manifest npmPackage = %q, want %q", m.NpmPackage, "@xai-official/grok")
	}
}
