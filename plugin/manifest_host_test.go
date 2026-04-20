package plugin

import (
	"strings"
	"testing"
)

// TestValidateV1_HostValid covers the happy paths for a host-form
// manifest declaring the minimum set of fields.
func TestValidateV1_HostValid(t *testing.T) {
	if !HostFormAllowed {
		t.Skip("host-form disabled on this build")
	}
	t.Run("minimal node sidecar", func(t *testing.T) {
		p := mustParseProvider(t, `{
			"name": "fs-readme",
			"version": "1.0.0",
			"publisher": "opendray-examples",
			"engines": { "opendray": "^1.0.0" },
			"form": "host",
			"host": {
				"entry": "sidecar.js",
				"runtime": "node",
				"protocol": "jsonrpc-stdio"
			}
		}`)
		if errs := ValidateV1(p); len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("binary with platform map", func(t *testing.T) {
		p := mustParseProvider(t, `{
			"name": "rust-analyzer-od",
			"version": "0.1.0",
			"publisher": "acme",
			"engines": { "opendray": "^1.0.0" },
			"form": "host",
			"host": {
				"entry": "bin/rust-analyzer",
				"runtime": "binary",
				"platforms": {
					"linux-x64": "bin/linux-x64/rust-analyzer",
					"darwin-arm64": "bin/darwin-arm64/rust-analyzer"
				},
				"restart": "on-failure",
				"env": { "RUST_LOG": "info" },
				"idleShutdownMinutes": 20
			}
		}`)
		if errs := ValidateV1(p); len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})
}

// TestValidateV1_HostInvalid table covers every rejection path in
// validateHostV1. Each row asserts the validator produces at least
// one ValidationError whose Path contains the wantPath substring.
func TestValidateV1_HostInvalid(t *testing.T) {
	if !HostFormAllowed {
		t.Skip("host-form disabled on this build")
	}
	tests := []struct {
		name     string
		manifest string
		wantPath string
	}{
		{
			name: "host block missing",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host"
			}`,
			wantPath: "host",
		},
		{
			name: "entry missing",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": { "runtime": "node" }
			}`,
			wantPath: "host.entry",
		},
		{
			name: "entry contains traversal",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": { "entry": "../../etc/passwd", "runtime": "binary" }
			}`,
			wantPath: "host.entry",
		},
		{
			name: "unknown runtime",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": { "entry": "x", "runtime": "ruby" }
			}`,
			wantPath: "host.runtime",
		},
		{
			name: "unsupported protocol",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": { "entry": "x", "runtime": "binary", "protocol": "grpc" }
			}`,
			wantPath: "host.protocol",
		},
		{
			name: "unknown restart",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": { "entry": "x", "runtime": "binary", "restart": "sometimes" }
			}`,
			wantPath: "host.restart",
		},
		{
			name: "bad platform key",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": {
					"entry": "x", "runtime": "binary",
					"platforms": { "bsd-riscv": "bin/x" }
				}
			}`,
			wantPath: "host.platforms",
		},
		{
			name: "bad env key",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": {
					"entry": "x", "runtime": "binary",
					"env": { "lowercase-bad": "v" }
				}
			}`,
			wantPath: "host.env",
		},
		{
			name: "cwd with traversal",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": {
					"entry": "x", "runtime": "binary",
					"cwd": "../outside"
				}
			}`,
			wantPath: "host.cwd",
		},
		{
			name: "negative idle shutdown",
			manifest: `{
				"name": "p", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "host",
				"host": {
					"entry": "x", "runtime": "binary",
					"idleShutdownMinutes": -1
				}
			}`,
			wantPath: "host.idleShutdownMinutes",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := mustParseProvider(t, tc.manifest)
			errs := ValidateV1(p)
			if len(errs) == 0 {
				t.Fatalf("expected at least one error")
			}
			found := false
			for _, e := range errs {
				if strings.Contains(e.Path, tc.wantPath) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error on path containing %q, got %v", tc.wantPath, errs)
			}
		})
	}
}

// TestValidateV1_WebviewHostCombined — M5 B1 combined form: a
// form:"webview" manifest may optionally ship a host:{} block so
// the webview UI can call its own privileged sidecar methods via
// opendray.commands.execute. The host block has to pass the same
// validation as a pure form:"host" plugin when present.
func TestValidateV1_WebviewHostCombined(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		manifest string
		wantOK   bool
		wantPath string
	}{
		{
			name: "webview no host — legal pure webview",
			manifest: `{
				"name": "ui-only", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "webview"
			}`,
			wantOK: true,
		},
		{
			name: "webview + valid host — combined form",
			manifest: `{
				"name": "combo", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "webview",
				"host": {"entry": "sidecar.js", "runtime": "node"}
			}`,
			wantOK: true,
		},
		{
			name: "webview + malformed host — combined form still validates",
			manifest: `{
				"name": "combo-bad", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "webview",
				"host": {"entry": "../escape", "runtime": "node"}
			}`,
			wantOK:   false,
			wantPath: "host.entry",
		},
		{
			name: "declarative + host — still ignored (neither form needs one)",
			manifest: `{
				"name": "dec", "version": "1.0.0", "publisher": "x",
				"engines": {"opendray": "^1.0.0"}, "form": "declarative",
				"host": {"entry": "junk", "runtime": "lolcode"}
			}`,
			wantOK: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := mustParseProvider(t, tc.manifest)
			errs := ValidateV1(p)
			if tc.wantOK {
				if len(errs) > 0 {
					t.Fatalf("want ok, got errors: %v", errs)
				}
				return
			}
			found := false
			for _, e := range errs {
				if strings.Contains(e.Path, tc.wantPath) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("want error on %q, got %v", tc.wantPath, errs)
			}
		})
	}
}

// TestProvider_HasHostBackend — the helper centralising the
// "does this plugin spawn a sidecar" predicate. Keep the table
// small but covering every form × host-presence combo.
func TestProvider_HasHostBackend(t *testing.T) {
	t.Parallel()
	host := &HostV1{Entry: "s.js", Runtime: "node"}
	cases := []struct {
		name string
		p    Provider
		want bool
	}{
		{"host + block", Provider{Form: FormHost, Host: host}, true},
		{"host + no block (invalid but truthful)", Provider{Form: FormHost}, false},
		{"webview + block (combined)", Provider{Form: FormWebview, Host: host}, true},
		{"webview + no block", Provider{Form: FormWebview}, false},
		{"declarative + block (ignored)", Provider{Form: FormDeclarative, Host: host}, false},
		{"declarative + no block", Provider{Form: FormDeclarative}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.p.HasHostBackend(); got != tc.want {
				t.Errorf("HasHostBackend() = %v, want %v", got, tc.want)
			}
		})
	}
}
