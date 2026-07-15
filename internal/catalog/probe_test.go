package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtractSemverAndUpdateAvailable(t *testing.T) {
	cases := []struct {
		installed, latest string
		wantUpdate        bool
	}{
		{"2.1.146 (Claude Code)", "2.1.150", true},
		{"codex-cli 0.132.0", "0.132.0", false},
		{"0.42.0", "0.41.9", false}, // installed ahead
		{"", "1.0.0", false},        // not installed
		{"1.2.3", "", false},        // no latest
		{"1.2.3", "1.3.0", true},
		{"1.9.0", "1.10.0", true}, // numeric, not lexical
	}
	for _, c := range cases {
		if got := updateAvailable(c.installed, c.latest); got != c.wantUpdate {
			t.Errorf("updateAvailable(%q,%q)=%v want %v", c.installed, c.latest, got, c.wantUpdate)
		}
	}
}

func TestProber_InstalledCachesAndProbes(t *testing.T) {
	calls := 0
	p := NewProber()
	p.now = func() time.Time { return time.Unix(1000, 0) }
	p.lookPath = func(string) (string, error) { return "/usr/bin/claude", nil }
	p.runVer = func(context.Context, string) (string, error) {
		calls++
		return "2.1.146 (Claude Code)", nil
	}
	m := Manifest{Executable: "claude"}

	got := p.Installed(context.Background(), m)
	if !got.Installed || got.InstalledVersion != "2.1.146 (Claude Code)" || got.Path != "/usr/bin/claude" {
		t.Fatalf("unexpected info: %+v", got)
	}
	// Second call within TTL must hit the cache (no extra probe).
	_ = p.Installed(context.Background(), m)
	if calls != 1 {
		t.Errorf("expected 1 probe (cached), got %d", calls)
	}
}

func TestProber_NotInstalled(t *testing.T) {
	p := NewProber()
	p.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	got := p.Installed(context.Background(), Manifest{Executable: "ghost"})
	if got.Installed {
		t.Errorf("ghost should not be installed: %+v", got)
	}
}

func TestProber_CheckUpdate(t *testing.T) {
	p := NewProber()
	p.now = func() time.Time { return time.Unix(2000, 0) }
	p.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
	p.runVer = func(context.Context, string) (string, error) { return "codex-cli 0.132.0", nil }
	p.npmView = func(context.Context, string) (string, error) { return "0.140.0", nil }

	info := p.CheckUpdate(context.Background(), Manifest{Executable: "codex", NpmPackage: "@openai/codex"})
	if info.InstalledVersion != "codex-cli 0.132.0" || info.LatestVersion != "0.140.0" || !info.UpdateAvailable {
		t.Errorf("unexpected: %+v", info)
	}

	// No npm package → no latest lookup, no false update flag.
	none := p.CheckUpdate(context.Background(), Manifest{Executable: "codex"})
	if none.UpdateAvailable || none.LatestVersion != "" {
		t.Errorf("no-package provider should not report updates: %+v", none)
	}
}

func TestProber_Update(t *testing.T) {
	versions := []string{"0.132.0", "0.140.0"} // before, after
	call := 0
	installs := 0
	p := NewProber()
	p.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
	p.runVer = func(context.Context, string) (string, error) {
		v := versions[call]
		if call < len(versions)-1 {
			call++
		}
		return "codex-cli " + v, nil
	}
	p.npmInstall = func(_ context.Context, pkg string) (string, error) {
		installs++
		if pkg != "@openai/codex" {
			t.Errorf("npm install got wrong package: %q", pkg)
		}
		return "+ @openai/codex@0.140.0\nadded 1 package", nil
	}
	// Preflight: point npm root at a writable temp dir so Update proceeds.
	p.npmRoot = func(context.Context) (string, error) { return t.TempDir(), nil }
	m := Manifest{ID: "codex", Executable: "codex", NpmPackage: "@openai/codex"}

	res, err := p.Update(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	if installs != 1 {
		t.Errorf("expected one npm install, got %d", installs)
	}
	if res.BeforeVersion != "codex-cli 0.132.0" || res.AfterVersion != "codex-cli 0.140.0" || !res.Changed {
		t.Errorf("unexpected result: %+v", res)
	}
	if res.Output == "" {
		t.Error("expected npm output tail")
	}
}

func TestProber_UpdateNoPackage(t *testing.T) {
	p := NewProber()
	_, err := p.Update(context.Background(), Manifest{ID: "shell", Executable: "bash"})
	if err == nil {
		t.Error("update of a provider without an npm package should error")
	}
}

func TestProber_UpdateReadonlyPrefix(t *testing.T) {
	installed := false
	p := NewProber()
	p.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
	p.runVer = func(context.Context, string) (string, error) { return "codex-cli 0.132.0", nil }
	p.npmInstall = func(context.Context, string) (string, error) { installed = true; return "", nil }
	// Point npm root at a path that can't be written (a file, not a dir).
	ro := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(ro, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	p.npmRoot = func(context.Context) (string, error) { return ro, nil }

	_, err := p.Update(context.Background(),
		Manifest{ID: "codex", Executable: "codex", NpmPackage: "@openai/codex"})
	if !errors.Is(err, ErrUpdatePrefixReadonly) {
		t.Fatalf("expected ErrUpdatePrefixReadonly, got %v", err)
	}
	if installed {
		t.Error("npm install must NOT run when the prefix is read-only")
	}
}

// Regression: the update handler passes the HTTP request's context down to
// `npm install -g`. If the client disconnects mid-install (browser closed,
// proxy timeout), cancelling that ctx SIGKILLs npm — which leaves a partial
// global tree (a stale `.<pkg>-XXXXXX` temp dir) that makes EVERY later
// install fail with ENOTEMPTY. That is exactly how a codex update wedged
// itself for a week. The install must be detached from the caller's
// cancellation.
func TestProber_UpdateDetachesInstallFromCallerCancellation(t *testing.T) {
	p := NewProber()
	p.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
	p.runVer = func(context.Context, string) (string, error) { return "0.1.0", nil }
	p.npmRoot = func(context.Context) (string, error) { return "", nil }

	var installCtxErr error
	p.npmInstall = func(ctx context.Context, _ string) (string, error) {
		installCtxErr = ctx.Err()
		return "ok", nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // client already gone

	m := Manifest{ID: "codex", Executable: "codex", NpmPackage: "@openai/codex"}
	if _, err := p.Update(ctx, m); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if installCtxErr != nil {
		t.Fatalf("install ctx was cancelled by the caller (%v) — npm would be SIGKILLed mid-install", installCtxErr)
	}
}

// Regression: a CLI can be on PATH but not runnable — a broken npm install
// missing its platform binary (codex threw "Missing optional dependency
// @openai/codex-linux-x64"). Installed() must surface that instead of
// silently reporting a blank version, which reads as "fine" in the UI.
func TestProber_InstalledSurfacesBrokenCLI(t *testing.T) {
	p := NewProber()
	p.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
	p.runVer = func(context.Context, string) (string, error) {
		return "", errors.New("exit status 1: Error: Missing optional dependency @openai/codex-linux-x64")
	}
	info := p.Installed(context.Background(), Manifest{ID: "codex", Executable: "codex"})
	if !info.Installed {
		t.Fatal("expected Installed=true (the binary is on PATH)")
	}
	if info.InstalledVersion != "" {
		t.Fatalf("expected empty version for a broken CLI, got %q", info.InstalledVersion)
	}
	if info.VersionError == "" {
		t.Fatal("expected VersionError to surface the broken CLI, got empty")
	}
	if !strings.Contains(info.VersionError, "Missing optional dependency") {
		t.Fatalf("VersionError should carry the CLI's own message, got %q", info.VersionError)
	}
}
