package revocation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// fakeCatalog is the minimum market.Catalog surface the poller
// touches: FetchRevocations returns canned bytes + err. Every other
// method returns zero values — the poller never calls them.
type fakeCatalog struct {
	body []byte
	err  error
}

func (f *fakeCatalog) List(_ context.Context) ([]market.Entry, error) {
	return nil, nil
}
func (f *fakeCatalog) Resolve(_ context.Context, _ market.Ref) (market.Entry, error) {
	return market.Entry{}, nil
}
func (f *fakeCatalog) BundlePath(_ context.Context, _ market.Ref) (string, bool, error) {
	return "", false, nil
}
func (f *fakeCatalog) FetchPublisher(_ context.Context, _ string) (market.PublisherRecord, error) {
	return market.PublisherRecord{}, nil
}
func (f *fakeCatalog) FetchRevocations(_ context.Context) ([]byte, error) {
	return f.body, f.err
}

// silentLogger keeps test output clean.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ─── Construction guards ──────────────────────────────────────────────────

func TestNew_RequiresCatalog(t *testing.T) {
	_, err := New(Config{
		Installed: func() []InstalledPlugin { return nil },
		OnAction:  func(context.Context, Entry, InstalledPlugin) error { return nil },
	})
	if err == nil {
		t.Fatal("want error on nil Catalog")
	}
}

func TestNew_RequiresInstalled(t *testing.T) {
	_, err := New(Config{
		Catalog:  &fakeCatalog{},
		OnAction: func(context.Context, Entry, InstalledPlugin) error { return nil },
	})
	if err == nil {
		t.Fatal("want error on nil Installed")
	}
}

func TestNew_RequiresOnAction(t *testing.T) {
	_, err := New(Config{
		Catalog:   &fakeCatalog{},
		Installed: func() []InstalledPlugin { return nil },
	})
	if err == nil {
		t.Fatal("want error on nil OnAction")
	}
}

// ─── Interval clamping ────────────────────────────────────────────────────

func TestClampInterval(t *testing.T) {
	cases := []struct{ in, want time.Duration }{
		{0, DefaultPollInterval},
		{30 * time.Minute, MinPollInterval},
		{3 * time.Hour, 3 * time.Hour},
		{200 * time.Hour, MaxPollInterval},
	}
	for _, tc := range cases {
		if got := clampInterval(tc.in); got != tc.want {
			t.Errorf("clampInterval(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ─── Sweep behaviour ──────────────────────────────────────────────────────

func TestSweep_FiresActionForMatch(t *testing.T) {
	body := []byte(`{"version":1,"entries":[
		{"name":"acme/evil","versions":"<=1.2.3","reason":"bad","recordedAt":"","action":"uninstall"}
	]}`)
	var hits int32
	p, err := New(Config{
		Catalog: &fakeCatalog{body: body},
		Installed: func() []InstalledPlugin {
			return []InstalledPlugin{{Publisher: "acme", Name: "evil", Version: "1.0.0"}}
		},
		OnAction: func(_ context.Context, e Entry, _ InstalledPlugin) error {
			if e.Action != ActionUninstall {
				t.Errorf("want uninstall action, got %q", e.Action)
			}
			atomic.AddInt32(&hits, 1)
			return nil
		},
		Logger: silentLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Sweep(context.Background())
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("OnAction hits = %d, want 1", hits)
	}
}

func TestSweep_NoActionForMiss(t *testing.T) {
	body := []byte(`{"version":1,"entries":[
		{"name":"acme/evil","versions":">=2.0.0","reason":"","recordedAt":"","action":"uninstall"}
	]}`)
	var hits int32
	p, _ := New(Config{
		Catalog: &fakeCatalog{body: body},
		Installed: func() []InstalledPlugin {
			return []InstalledPlugin{{Publisher: "acme", Name: "evil", Version: "1.0.0"}}
		},
		OnAction: func(context.Context, Entry, InstalledPlugin) error {
			atomic.AddInt32(&hits, 1)
			return nil
		},
		Logger: silentLogger(),
	})
	p.Sweep(context.Background())
	if atomic.LoadInt32(&hits) != 0 {
		t.Errorf("expected no hits, got %d", hits)
	}
}

func TestSweep_SkipsMalformedEntries(t *testing.T) {
	// First entry has garbage Name; second is valid. Poller should
	// log + skip the first and still fire for the second.
	body := []byte(`{"version":1,"entries":[
		{"name":"a/b/c","versions":"*","action":"warn"},
		{"name":"acme/evil","versions":"*","action":"disable"}
	]}`)
	var hits int32
	p, _ := New(Config{
		Catalog: &fakeCatalog{body: body},
		Installed: func() []InstalledPlugin {
			return []InstalledPlugin{{Publisher: "acme", Name: "evil", Version: "1.0.0"}}
		},
		OnAction: func(context.Context, Entry, InstalledPlugin) error {
			atomic.AddInt32(&hits, 1)
			return nil
		},
		Logger: silentLogger(),
	})
	p.Sweep(context.Background())
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("hits = %d, want 1 (malformed skipped, valid fired)", hits)
	}
}

func TestSweep_IgnoresEmptyBody(t *testing.T) {
	var hits int32
	p, _ := New(Config{
		Catalog: &fakeCatalog{body: nil},
		Installed: func() []InstalledPlugin {
			return []InstalledPlugin{{Publisher: "acme", Name: "evil", Version: "1.0.0"}}
		},
		OnAction: func(context.Context, Entry, InstalledPlugin) error {
			atomic.AddInt32(&hits, 1)
			return nil
		},
		Logger: silentLogger(),
	})
	p.Sweep(context.Background())
	if atomic.LoadInt32(&hits) != 0 {
		t.Errorf("empty body should produce zero hits, got %d", hits)
	}
}

func TestSweep_SurvivesFetchError(t *testing.T) {
	// Network-level failure → log + return; no panics.
	p, _ := New(Config{
		Catalog:   &fakeCatalog{err: errors.New("boom")},
		Installed: func() []InstalledPlugin { return nil },
		OnAction:  func(context.Context, Entry, InstalledPlugin) error { return nil },
		Logger:    silentLogger(),
	})
	p.Sweep(context.Background()) // must not panic
}

func TestSweep_ActionErrorKeepsSweeping(t *testing.T) {
	// Two entries both match; first OnAction errors; second must
	// still be invoked.
	body := []byte(`{"version":1,"entries":[
		{"name":"acme/a","versions":"*","action":"uninstall"},
		{"name":"acme/b","versions":"*","action":"disable"}
	]}`)
	var hits int32
	p, _ := New(Config{
		Catalog: &fakeCatalog{body: body},
		Installed: func() []InstalledPlugin {
			return []InstalledPlugin{
				{Publisher: "acme", Name: "a", Version: "1.0.0"},
				{Publisher: "acme", Name: "b", Version: "1.0.0"},
			}
		},
		OnAction: func(_ context.Context, e Entry, _ InstalledPlugin) error {
			atomic.AddInt32(&hits, 1)
			if e.Name == "acme/a" {
				return errors.New("simulated")
			}
			return nil
		},
		Logger: silentLogger(),
	})
	p.Sweep(context.Background())
	if atomic.LoadInt32(&hits) != 2 {
		t.Errorf("hits = %d, want 2 (error on first shouldn't abort)", hits)
	}
}
