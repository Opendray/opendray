package actions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/opendray/opendray/plugin/market/revocation"
)

// recorder captures every callback invocation so tests assert
// behaviour without globals.
type recorder struct {
	mu          sync.Mutex
	uninstalls  []string
	setEnabledCalls [][2]any // [name, enabled]
	notifies    [][3]string // [kind, pluginName, reason]
	uninstErr   error
	enabledErr  error
}

func (r *recorder) uninstall(_ context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uninstalls = append(r.uninstalls, name)
	return r.uninstErr
}

func (r *recorder) setEnabled(_ context.Context, name string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setEnabledCalls = append(r.setEnabledCalls, [2]any{name, enabled})
	return r.enabledErr
}

func (r *recorder) notify(kind, pluginName, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifies = append(r.notifies, [3]string{kind, pluginName, reason})
}

func silent() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func buildHandler(t *testing.T, r *recorder) *Handler {
	t.Helper()
	h, err := New(Config{
		Uninstall:  r.uninstall,
		SetEnabled: r.setEnabled,
		Notify:     r.notify,
		Logger:     silent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// ─── New() guards ──────────────────────────────────────────────────────────

func TestNew_RequiresAllCallbacks(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"missing Uninstall", Config{
			SetEnabled: func(context.Context, string, bool) error { return nil },
			Notify:     func(string, string, string) {},
		}},
		{"missing SetEnabled", Config{
			Uninstall: func(context.Context, string) error { return nil },
			Notify:    func(string, string, string) {},
		}},
		{"missing Notify", Config{
			Uninstall:  func(context.Context, string) error { return nil },
			SetEnabled: func(context.Context, string, bool) error { return nil },
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.cfg); err == nil {
				t.Error("want error on missing callback")
			}
		})
	}
}

// ─── Dispatch by action ────────────────────────────────────────────────────

func TestDispatch_Uninstall(t *testing.T) {
	r := &recorder{}
	h := buildHandler(t, r)

	err := h.Dispatch(context.Background(),
		revocation.Entry{Name: "acme/evil", Action: revocation.ActionUninstall, Reason: "bad"},
		revocation.InstalledPlugin{Publisher: "acme", Name: "evil", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if len(r.uninstalls) != 1 || r.uninstalls[0] != "evil" {
		t.Errorf("uninstalls = %v; want [evil]", r.uninstalls)
	}
	if len(r.notifies) != 1 {
		t.Errorf("notifies = %v; want exactly 1", r.notifies)
	}
	if r.notifies[0][0] != "uninstall" || r.notifies[0][1] != "acme/evil" || r.notifies[0][2] != "bad" {
		t.Errorf("notify = %+v", r.notifies[0])
	}
	// SetEnabled should NOT have fired for an uninstall action.
	if len(r.setEnabledCalls) != 0 {
		t.Errorf("setEnabled = %v; want 0 calls", r.setEnabledCalls)
	}
}

func TestDispatch_Disable(t *testing.T) {
	r := &recorder{}
	h := buildHandler(t, r)

	err := h.Dispatch(context.Background(),
		revocation.Entry{Name: "acme/evil", Action: revocation.ActionDisable, Reason: "flaky"},
		revocation.InstalledPlugin{Publisher: "acme", Name: "evil", Version: "2.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.setEnabledCalls) != 1 || r.setEnabledCalls[0][0].(string) != "evil" || r.setEnabledCalls[0][1].(bool) != false {
		t.Errorf("setEnabled = %+v", r.setEnabledCalls)
	}
	if len(r.uninstalls) != 0 {
		t.Errorf("uninstalls should be empty on disable, got %v", r.uninstalls)
	}
	if r.notifies[0][0] != "disable" {
		t.Errorf("notify kind = %q, want disable", r.notifies[0][0])
	}
}

func TestDispatch_Warn(t *testing.T) {
	r := &recorder{}
	h := buildHandler(t, r)

	err := h.Dispatch(context.Background(),
		revocation.Entry{Name: "acme/evil", Action: revocation.ActionWarn, Reason: "heads up"},
		revocation.InstalledPlugin{Publisher: "acme", Name: "evil", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	// Warn fires notify only — no state change.
	if len(r.uninstalls) != 0 || len(r.setEnabledCalls) != 0 {
		t.Errorf("warn should only notify; got uninstalls=%v setEnabled=%v",
			r.uninstalls, r.setEnabledCalls)
	}
	if len(r.notifies) != 1 || r.notifies[0][0] != "warn" {
		t.Errorf("notifies = %+v", r.notifies)
	}
}

// ─── Unknown action normalises to warn ────────────────────────────────────

func TestDispatch_UnknownActionFallsToWarn(t *testing.T) {
	r := &recorder{}
	h := buildHandler(t, r)

	err := h.Dispatch(context.Background(),
		revocation.Entry{Name: "acme/x", Action: "exterminate"},
		revocation.InstalledPlugin{Publisher: "acme", Name: "x", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if r.notifies[0][0] != "warn" {
		t.Errorf("unknown action should normalise to warn, got %q", r.notifies[0][0])
	}
}

// ─── Notify fires even when the effect fails ──────────────────────────────

func TestDispatch_NotifyFiresBeforeUninstallError(t *testing.T) {
	r := &recorder{uninstErr: errors.New("locked")}
	h := buildHandler(t, r)

	err := h.Dispatch(context.Background(),
		revocation.Entry{Name: "acme/evil", Action: revocation.ActionUninstall},
		revocation.InstalledPlugin{Publisher: "acme", Name: "evil", Version: "1.0.0"})
	if err == nil {
		t.Fatal("want error when Uninstall fails")
	}
	if len(r.notifies) != 1 {
		t.Errorf("notify should still fire before the uninstall error, got %v", r.notifies)
	}
}

func TestDispatch_SetEnabledError(t *testing.T) {
	r := &recorder{enabledErr: errors.New("db down")}
	h := buildHandler(t, r)

	err := h.Dispatch(context.Background(),
		revocation.Entry{Name: "acme/evil", Action: revocation.ActionDisable},
		revocation.InstalledPlugin{Publisher: "acme", Name: "evil", Version: "1.0.0"})
	if err == nil {
		t.Fatal("want error when SetEnabled fails")
	}
}
