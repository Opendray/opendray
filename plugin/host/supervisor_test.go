//go:build !windows

package host

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin"
)

// catBinary returns the absolute path of `cat` on the test host,
// skipping the test if not present (keeps the test portable across
// distros that split /bin vs /usr/bin).
func catBinary(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("cat")
	if err != nil {
		t.Skipf("cat not on PATH: %v", err)
	}
	return p
}

// fakeProviders stubs ProviderLookup with a static map.
type fakeProviders struct {
	m map[string]plugin.Provider
}

func (f *fakeProviders) Get(name string) (plugin.Provider, bool) {
	p, ok := f.m[name]
	return p, ok
}

func hostPluginWithEntry(name, entry string) plugin.Provider {
	return plugin.Provider{
		Name:      name,
		Version:   "1.0.0",
		Publisher: "test",
		Form:      plugin.FormHost,
		Engines:   &plugin.EnginesV1{Opendray: "^1.0.0"},
		Host: &plugin.HostV1{
			Entry:   entry,
			Runtime: plugin.HostRuntimeCustom,
		},
	}
}

func TestSupervisor_EnsureStartsProcessAndDedupes(t *testing.T) {
	cat := catBinary(t)
	providers := &fakeProviders{m: map[string]plugin.Provider{
		"cat": hostPluginWithEntry("cat", cat),
	}}
	sup := NewSupervisor(Config{DataDir: t.TempDir(), Providers: providers})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = sup.Stop(ctx)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sc, err := sup.Ensure(ctx, "cat")
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if sc == nil || !sc.isAlive() {
		t.Fatalf("expected live sidecar")
	}
	if sup.ActiveCount() != 1 {
		t.Errorf("ActiveCount = %d, want 1", sup.ActiveCount())
	}

	sc2, err := sup.Ensure(ctx, "cat")
	if err != nil {
		t.Fatalf("Ensure second: %v", err)
	}
	if sc2 != sc {
		t.Error("second Ensure returned a different sidecar (dedupe broken)")
	}
}

func TestSupervisor_KillTerminatesProcess(t *testing.T) {
	cat := catBinary(t)
	providers := &fakeProviders{m: map[string]plugin.Provider{
		"cat": hostPluginWithEntry("cat", cat),
	}}
	sup := NewSupervisor(Config{DataDir: t.TempDir(), Providers: providers})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sc, err := sup.Ensure(ctx, "cat")
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	if err := sup.Kill("cat", "test"); err != nil {
		t.Errorf("Kill: %v", err)
	}

	select {
	case <-sc.Exited():
	case <-time.After(3 * time.Second):
		t.Fatal("sidecar did not exit within 3s")
	}
	if sup.ActiveCount() != 0 {
		t.Errorf("ActiveCount = %d, want 0", sup.ActiveCount())
	}
}

func TestSupervisor_StopRejectsEnsure(t *testing.T) {
	cat := catBinary(t)
	providers := &fakeProviders{m: map[string]plugin.Provider{
		"cat": hostPluginWithEntry("cat", cat),
	}}
	sup := NewSupervisor(Config{DataDir: t.TempDir(), Providers: providers})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sup.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	_, err := sup.Ensure(ctx, "cat")
	if !errors.Is(err, ErrSupervisorStopped) {
		t.Errorf("expected ErrSupervisorStopped, got %v", err)
	}
}

func TestSupervisor_EnsureErrorsOnNonHostForm(t *testing.T) {
	providers := &fakeProviders{m: map[string]plugin.Provider{
		"ranger": {
			Name: "ranger", Version: "1.0.0", Publisher: "t",
			Type:    "panel",
			Engines: &plugin.EnginesV1{Opendray: "^1.0.0"},
		},
	}}
	sup := NewSupervisor(Config{DataDir: t.TempDir(), Providers: providers})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = sup.Stop(ctx)
	}()

	_, err := sup.Ensure(context.Background(), "ranger")
	if !errors.Is(err, ErrNoHost) {
		t.Errorf("expected ErrNoHost, got %v", err)
	}
}

func TestSupervisor_EnsureErrorsOnUnknownPlugin(t *testing.T) {
	sup := NewSupervisor(Config{
		DataDir:   t.TempDir(),
		Providers: &fakeProviders{m: map[string]plugin.Provider{}},
	})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = sup.Stop(ctx)
	}()
	_, err := sup.Ensure(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error for unknown plugin")
	}
}
