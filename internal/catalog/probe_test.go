package catalog

import (
	"context"
	"errors"
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
