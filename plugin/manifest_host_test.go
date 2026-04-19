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
