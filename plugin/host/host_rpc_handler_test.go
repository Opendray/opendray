package host

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/opendray/opendray/plugin/bridge"
)

// fakeDispatcher records what it saw and returns a preset response.
type fakeDispatcher struct {
	gotPlugin string
	gotMethod string
	gotArgs   json.RawMessage
	result    any
	err       error
}

func (f *fakeDispatcher) Dispatch(_ context.Context, plugin, method string, args json.RawMessage) (any, error) {
	f.gotPlugin = plugin
	f.gotMethod = method
	f.gotArgs = args
	return f.result, f.err
}

func TestHostRPCHandler_RoutesToNamespace(t *testing.T) {
	t.Parallel()
	fs := &fakeDispatcher{result: map[string]string{"ok": "yes"}}
	h, err := NewHostRPCHandler(HostRPCConfig{
		Plugin:     "fs-readme",
		Namespaces: map[string]NSDispatcher{"fs": fs},
	})
	if err != nil {
		t.Fatalf("NewHostRPCHandler: %v", err)
	}
	res, err := h.Handle(context.Background(), "fs/readFile", json.RawMessage(`["/x"]`))
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if fs.gotPlugin != "fs-readme" {
		t.Errorf("plugin=%q, want fs-readme (sidecar must not override)", fs.gotPlugin)
	}
	if fs.gotMethod != "readFile" {
		t.Errorf("method=%q, want readFile", fs.gotMethod)
	}
	if string(fs.gotArgs) != `["/x"]` {
		t.Errorf("args round-trip mismatch: %q", fs.gotArgs)
	}
	if res == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestHostRPCHandler_SidecarCannotImpersonate(t *testing.T) {
	t.Parallel()
	fs := &fakeDispatcher{result: "ok"}
	h, _ := NewHostRPCHandler(HostRPCConfig{
		Plugin:     "fs-readme",
		Namespaces: map[string]NSDispatcher{"fs": fs},
	})
	// Even if the sidecar somehow encoded a plugin field in params,
	// the handler sends its bound Plugin — never reads params for it.
	_, err := h.Handle(context.Background(), "fs/readFile", json.RawMessage(`{"plugin":"attacker"}`))
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if fs.gotPlugin != "fs-readme" {
		t.Errorf("plugin impersonation succeeded: %q", fs.gotPlugin)
	}
}

func TestHostRPCHandler_UnknownNamespace(t *testing.T) {
	t.Parallel()
	h, _ := NewHostRPCHandler(HostRPCConfig{
		Plugin:     "p",
		Namespaces: map[string]NSDispatcher{"fs": &fakeDispatcher{}},
	})
	_, err := h.Handle(context.Background(), "nope/do", nil)
	var rerr *RPCError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RPCError, got %v", err)
	}
	if rerr.Code != RPCErrMethodNotFound {
		t.Errorf("code=%d, want %d", rerr.Code, RPCErrMethodNotFound)
	}
}

func TestHostRPCHandler_MethodInjectionRejected(t *testing.T) {
	t.Parallel()
	h, _ := NewHostRPCHandler(HostRPCConfig{
		Plugin:     "p",
		Namespaces: map[string]NSDispatcher{"fs": &fakeDispatcher{}},
	})
	for _, method := range []string{
		"fs/readFile/extra",   // two slashes
		"fs\x00/readFile",      // null byte
		"fs/../readFile",       // traversal
		"fs",                   // no slash
		"/readFile",            // empty ns
		"fs/",                  // empty method
		"",                     // empty
	} {
		_, err := h.Handle(context.Background(), method, nil)
		var rerr *RPCError
		if !errors.As(err, &rerr) {
			t.Errorf("method=%q expected *RPCError, got %v", method, err)
			continue
		}
		if rerr.Code != RPCErrInvalidRequest && rerr.Code != RPCErrMethodNotFound {
			t.Errorf("method=%q code=%d, want InvalidRequest or MethodNotFound", method, rerr.Code)
		}
	}
}

func TestHostRPCHandler_PermErrorMapsToEPERM(t *testing.T) {
	t.Parallel()
	fs := &fakeDispatcher{err: &bridge.PermError{Code: "EPERM", Msg: "fs.read not granted"}}
	h, _ := NewHostRPCHandler(HostRPCConfig{
		Plugin:     "p",
		Namespaces: map[string]NSDispatcher{"fs": fs},
	})
	_, err := h.Handle(context.Background(), "fs/readFile", nil)
	var rerr *RPCError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RPCError, got %v", err)
	}
	if rerr.Code != RPCErrEPERM {
		t.Errorf("code=%d, want %d", rerr.Code, RPCErrEPERM)
	}
}

func TestHostRPCHandler_WireErrorMapsByCode(t *testing.T) {
	t.Parallel()
	cases := map[string]int{
		"EINVAL":    RPCErrInvalidParams,
		"EUNAVAIL":  RPCErrEUNAVAIL,
		"EINTERNAL": RPCErrInternal,
		"EFOOBAR":   RPCErrInternal, // unknown → internal
	}
	for wireCode, wantRPC := range cases {
		t.Run(wireCode, func(t *testing.T) {
			fs := &fakeDispatcher{err: &bridge.WireError{Code: wireCode, Message: "x"}}
			h, _ := NewHostRPCHandler(HostRPCConfig{
				Plugin:     "p",
				Namespaces: map[string]NSDispatcher{"fs": fs},
			})
			_, err := h.Handle(context.Background(), "fs/readFile", nil)
			var rerr *RPCError
			if !errors.As(err, &rerr) {
				t.Fatalf("expected *RPCError, got %v", err)
			}
			if rerr.Code != wantRPC {
				t.Errorf("wire %q → rpc %d, want %d", wireCode, rerr.Code, wantRPC)
			}
		})
	}
}

func TestNewHostRPCHandler_Validation(t *testing.T) {
	t.Parallel()
	if _, err := NewHostRPCHandler(HostRPCConfig{}); err == nil {
		t.Error("expected error on empty config")
	}
	if _, err := NewHostRPCHandler(HostRPCConfig{Plugin: "p"}); err == nil {
		t.Error("expected error on empty Namespaces")
	}
	if _, err := NewHostRPCHandler(HostRPCConfig{
		Namespaces: map[string]NSDispatcher{"fs": &fakeDispatcher{}},
	}); err == nil {
		t.Error("expected error on empty Plugin")
	}
}
