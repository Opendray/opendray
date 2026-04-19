package bridge_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/opendray/opendray/plugin/bridge"
)

// ─────────────────────────────────────────────
// Fake implementations for test isolation
// ─────────────────────────────────────────────

// fakeConsent is a ConsentReader backed by a static PermissionsV1 JSON blob.
type fakeConsent struct {
	perms []byte
	found bool
	err   error
}

func (f *fakeConsent) Load(_ context.Context, _ string) ([]byte, bool, error) {
	return f.perms, f.found, f.err
}

// fakeAudit records every AuditEvent appended.
type fakeAudit struct {
	events []bridge.AuditEvent
	err    error
}

func (f *fakeAudit) Append(_ context.Context, ev bridge.AuditEvent) error {
	f.events = append(f.events, ev)
	return f.err
}

// permJSON builds a minimal PermissionsV1 JSON blob granting exec globs.
func execPermsJSON(t *testing.T, globs []string) []byte {
	t.Helper()
	raw, _ := json.Marshal(globs)
	return []byte(`{"exec":` + string(raw) + `}`)
}

func httpPermsJSON(t *testing.T, patterns []string) []byte {
	t.Helper()
	raw, _ := json.Marshal(patterns)
	return []byte(`{"http":` + string(raw) + `}`)
}

func fsReadPermsJSON(t *testing.T, patterns []string) []byte {
	t.Helper()
	raw, _ := json.Marshal(patterns)
	return []byte(`{"fs":{"read":` + string(raw) + `}}`)
}

// ─────────────────────────────────────────────
// 1. TestMatchExecGlobs — table-driven, ≥15 cases
// ─────────────────────────────────────────────

func TestMatchExecGlobs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		granted []string
		cmdline string
		want    bool
	}{
		// allow-all wildcard
		{name: "allow_all_wildcard", granted: []string{"*"}, cmdline: "rm -rf /", want: true},
		{name: "allow_all_wildcard_single_token", granted: []string{"*"}, cmdline: "ls", want: true},

		// exact literal (no wildcard — only matches identical cmdline)
		{name: "exact_match", granted: []string{"pnpm"}, cmdline: "pnpm", want: true},
		{name: "exact_no_extra_args", granted: []string{"pnpm"}, cmdline: "pnpm install", want: false},

		// prefix: "git *" → first token must be git, any remainder
		{name: "prefix_git_star_matches_subcommand", granted: []string{"git *"}, cmdline: "git status", want: true},
		{name: "prefix_git_star_matches_multi_args", granted: []string{"git *"}, cmdline: "git log --oneline", want: true},
		{name: "prefix_git_star_no_match_other_cmd", granted: []string{"git *"}, cmdline: "svn commit", want: false},
		{name: "prefix_git_star_bare_cmd_no_args", granted: []string{"git *"}, cmdline: "git", want: false},

		// partial sub-command prefix: "git log*"
		{name: "partial_subcommand_log_matches", granted: []string{"git log*"}, cmdline: "git log --oneline", want: true},
		{name: "partial_subcommand_log_exact", granted: []string{"git log*"}, cmdline: "git log", want: true},
		{name: "partial_subcommand_log_no_status", granted: []string{"git log*"}, cmdline: "git status", want: false},

		// empty granted list → always false
		{name: "empty_granted", granted: []string{}, cmdline: "git status", want: false},
		{name: "nil_granted", granted: nil, cmdline: "anything", want: false},

		// empty cmdline → always false
		{name: "empty_cmdline", granted: []string{"*"}, cmdline: "", want: false},

		// case-sensitivity (patterns are case-sensitive)
		{name: "case_sensitive_upper_no_match", granted: []string{"Git *"}, cmdline: "git status", want: false},
		{name: "case_sensitive_exact_upper", granted: []string{"Git *"}, cmdline: "Git status", want: true},

		// leading/trailing whitespace in cmdline is trimmed/normalised
		{name: "leading_whitespace_cmdline", granted: []string{"git *"}, cmdline: "  git status", want: false},
		// NOTE: leading-space makes first token empty, so it doesn't match "git" → deny (conservative)

		// multiple patterns — first match wins
		{name: "multi_pattern_second_matches", granted: []string{"npm *", "git *"}, cmdline: "git status", want: true},
		{name: "multi_pattern_none_matches", granted: []string{"npm *", "pnpm"}, cmdline: "yarn add react", want: false},

		// cmdline with multiple args
		{name: "multi_arg_cmdline", granted: []string{"cargo test*"}, cmdline: "cargo test --no-run --workspace", want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bridge.MatchExecGlobs(tc.granted, tc.cmdline)
			if got != tc.want {
				t.Errorf("MatchExecGlobs(%v, %q) = %v, want %v", tc.granted, tc.cmdline, got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────
// 2. TestMatchHTTPURL — table-driven, ≥12 cases
// ─────────────────────────────────────────────

func TestMatchHTTPURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		granted []string
		rawURL  string
		want    bool
	}{
		// happy path
		{name: "https_exact", granted: []string{"https://api.github.com/*"}, rawURL: "https://api.github.com/repos/foo", want: true},
		{name: "wildcard_host_and_path", granted: []string{"https://*.example.com/*"}, rawURL: "https://cdn.example.com/assets/main.js", want: true},

		// allow-all wildcard still denies RFC1918 (SSRF protection)
		{name: "deny_rfc1918_192_168_even_with_wildcard", granted: []string{"*"}, rawURL: "https://192.168.1.1/admin", want: false},
		{name: "deny_rfc1918_10_0_0_5", granted: []string{"*"}, rawURL: "https://10.0.0.5/api", want: false},
		{name: "deny_rfc1918_172_20_1_1", granted: []string{"*"}, rawURL: "https://172.20.1.1/secret", want: false},

		// deny loopback
		{name: "deny_loopback_127_0_0_1", granted: []string{"*"}, rawURL: "https://127.0.0.1:8080/health", want: false},
		{name: "deny_loopback_localhost", granted: []string{"*"}, rawURL: "https://localhost/api", want: false},
		{name: "deny_loopback_ipv6_1", granted: []string{"*"}, rawURL: "https://[::1]/api", want: false},

		// deny link-local (AWS IMDS endpoint is critical)
		{name: "deny_link_local_169_254", granted: []string{"*"}, rawURL: "https://169.254.169.254/latest/meta-data/", want: false},
		{name: "deny_link_local_fe80", granted: []string{"*"}, rawURL: "https://[fe80::1]/data", want: false},

		// non-https scheme without explicit http:// pattern → deny
		{name: "deny_http_without_explicit_pattern", granted: []string{"https://example.com/*"}, rawURL: "http://example.com/data", want: false},
		// explicit http:// pattern allows it (for dev/LAN use only)
		{name: "allow_explicit_http_pattern", granted: []string{"http://example.com/*"}, rawURL: "http://example.com/data", want: true},

		// invalid URL → false
		{name: "invalid_url", granted: []string{"*"}, rawURL: "not a url !!!", want: false},

		// query string is stripped before matching
		{name: "query_stripped_matches", granted: []string{"https://api.github.com/*"}, rawURL: "https://api.github.com/search?q=foo", want: true},
		// no match even with query included in wrong pattern
		{name: "no_match_different_host", granted: []string{"https://api.github.com/*"}, rawURL: "https://api.gitlab.com/repos", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bridge.MatchHTTPURL(tc.granted, tc.rawURL)
			if got != tc.want {
				t.Errorf("MatchHTTPURL(%v, %q) = %v, want %v", tc.granted, tc.rawURL, got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────
// 3. TestMatchFSPath — table-driven, ≥10 cases
// ─────────────────────────────────────────────

func TestMatchFSPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		granted []string
		absPath string
		want    bool
	}{
		// basic absolute path allow
		{name: "exact_file_match", granted: []string{"/home/user/.config/app.json"}, absPath: "/home/user/.config/app.json", want: true},
		// doublestar descendant
		{name: "double_star_descendant", granted: []string{"/workspace/**"}, absPath: "/workspace/src/main.go", want: true},
		{name: "double_star_nested", granted: []string{"/workspace/**"}, absPath: "/workspace/a/b/c/d.txt", want: true},
		// relative path → always false
		{name: "relative_path_denied", granted: []string{"*"}, absPath: "relative/path/file.txt", want: false},
		// path outside glob
		{name: "outside_glob", granted: []string{"/workspace/**"}, absPath: "/etc/passwd", want: false},
		// single-star wildcard in filename
		{name: "single_star_filename", granted: []string{"/workspace/src/*.go"}, absPath: "/workspace/src/main.go", want: true},
		{name: "single_star_no_cross_dir", granted: []string{"/workspace/src/*.go"}, absPath: "/workspace/src/sub/main.go", want: false},
		// path traversal via ".." → false (we clean and check absoluteness)
		{name: "path_traversal_dotdot", granted: []string{"/workspace/**"}, absPath: "/workspace/../etc/passwd", want: false},
		// no patterns → always false
		{name: "nil_granted", granted: nil, absPath: "/workspace/main.go", want: false},
		{name: "empty_granted", granted: []string{}, absPath: "/workspace/main.go", want: false},
		// double-star does not match parent dir itself (only descendants)
		{name: "double_star_root_itself", granted: []string{"/workspace/**"}, absPath: "/workspace", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bridge.MatchFSPath(tc.granted, tc.absPath)
			if got != tc.want {
				t.Errorf("MatchFSPath(%v, %q) = %v, want %v", tc.granted, tc.absPath, got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────
// 4. TestGate_CheckAllow
// ─────────────────────────────────────────────

func TestGate_CheckAllow(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: execPermsJSON(t, []string{"git *", "npm *"}),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{Cap: "exec", Target: "git status"})
	if err != nil {
		t.Fatalf("Check returned unexpected error: %v", err)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	ev := audit.events[0]
	if ev.Result != "ok" {
		t.Errorf("audit result = %q, want %q", ev.Result, "ok")
	}
	if ev.PluginName != "myplugin" {
		t.Errorf("audit plugin = %q, want %q", ev.PluginName, "myplugin")
	}
}

// ─────────────────────────────────────────────
// 5. TestGate_CheckDeny
// ─────────────────────────────────────────────

func TestGate_CheckDeny(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: execPermsJSON(t, []string{"git *"}),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{Cap: "exec", Target: "rm -rf /"})
	if err == nil {
		t.Fatal("Check should have returned an error for denied command")
	}
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PermError, got %T: %v", err, err)
	}
	if pe.Code != "EPERM" {
		t.Errorf("PermError.Code = %q, want %q", pe.Code, "EPERM")
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	if audit.events[0].Result != "denied" {
		t.Errorf("audit result = %q, want %q", audit.events[0].Result, "denied")
	}
}

// ─────────────────────────────────────────────
// 6. TestGate_CheckNoConsent
// ─────────────────────────────────────────────

func TestGate_CheckNoConsent(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: nil,
		found: false, // no consent row
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "unplugin", bridge.Need{Cap: "exec", Target: "git status"})
	if err == nil {
		t.Fatal("expected denial when no consent row exists")
	}
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PermError, got %T: %v", err, err)
	}
	if pe.Code != "EPERM" {
		t.Errorf("PermError.Code = %q, want %q", pe.Code, "EPERM")
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	if audit.events[0].Result != "denied" {
		t.Errorf("audit result = %q, want %q", audit.events[0].Result, "denied")
	}
}

// ─────────────────────────────────────────────
// 7. TestGate_CheckConsentError
// ─────────────────────────────────────────────

func TestGate_CheckConsentError(t *testing.T) {
	t.Parallel()

	consentErr := errors.New("database connection lost")
	audit := &fakeAudit{}
	consent := &fakeConsent{err: consentErr}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{Cap: "exec", Target: "git status"})
	if err == nil {
		t.Fatal("expected error when consent reader fails")
	}
	// Must NOT be a PermError — must be the wrapped underlying error
	var pe *bridge.PermError
	if errors.As(err, &pe) {
		t.Fatal("expected non-PermError when consent reader returns an error")
	}
	if !errors.Is(err, consentErr) {
		t.Errorf("error should wrap consentErr, got: %v", err)
	}
	// audit row with result "error" must be written
	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	if audit.events[0].Result != "error" {
		t.Errorf("audit result = %q, want %q", audit.events[0].Result, "error")
	}
}

// ─────────────────────────────────────────────
// 8. TestGate_NilAuditSinkIsSilent
// ─────────────────────────────────────────────

func TestGate_NilAuditSinkIsSilent(t *testing.T) {
	t.Parallel()

	consent := &fakeConsent{
		perms: execPermsJSON(t, []string{"git *"}),
		found: true,
	}
	// nil audit sink — must not panic
	g := bridge.NewGate(consent, nil, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{Cap: "exec", Target: "git status"})
	if err != nil {
		t.Fatalf("expected allow, got: %v", err)
	}

	// also test deny path doesn't panic with nil audit
	err = g.Check(context.Background(), "myplugin", bridge.Need{Cap: "exec", Target: "rm -rf /"})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PermError on deny, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// 9. TestNewGate_Defaults (nil logger must not panic)
// ─────────────────────────────────────────────

func TestNewGate_Defaults(t *testing.T) {
	t.Parallel()

	consent := &fakeConsent{
		perms: execPermsJSON(t, []string{"git *"}),
		found: true,
	}
	// nil logger must use slog.Default(), not panic
	g := bridge.NewGate(consent, nil, nil)

	err := g.Check(context.Background(), "myplugin", bridge.Need{Cap: "exec", Target: "git status"})
	if err != nil {
		t.Fatalf("nil logger gate returned unexpected error: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: PermError.Error() string form
// ─────────────────────────────────────────────

func TestPermError_Error(t *testing.T) {
	t.Parallel()
	pe := &bridge.PermError{Code: "EPERM", Msg: "exec not granted for: rm -rf /"}
	s := pe.Error()
	if s == "" {
		t.Error("PermError.Error() returned empty string")
	}
}

// ─────────────────────────────────────────────
// Extra: Gate.Check with http cap
// ─────────────────────────────────────────────

func TestGate_CheckHTTP_Allow(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: httpPermsJSON(t, []string{"https://api.github.com/*"}),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{
		Cap:    "http",
		Target: "https://api.github.com/repos",
	})
	if err != nil {
		t.Fatalf("expected allow for granted https URL, got: %v", err)
	}
	if len(audit.events) != 1 || audit.events[0].Result != "ok" {
		t.Errorf("audit result = %v, want 'ok'", audit.events)
	}
}

func TestGate_CheckHTTP_DenyRFC1918(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: httpPermsJSON(t, []string{"*"}),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{
		Cap:    "http",
		Target: "https://192.168.1.100/admin",
	})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PermError for RFC1918 target even with wildcard grant, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: Gate.Check with fs.read cap
// ─────────────────────────────────────────────

func TestGate_CheckFS_Allow(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: fsReadPermsJSON(t, []string{"/workspace/**"}),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{
		Cap:    "fs.read",
		Target: "/workspace/src/main.go",
	})
	if err != nil {
		t.Fatalf("expected allow for granted fs.read path, got: %v", err)
	}
	if len(audit.events) != 1 || audit.events[0].Result != "ok" {
		t.Errorf("audit result = %v, want 'ok'", audit.events)
	}
}

func TestGate_CheckFS_DenyOutsideGlob(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: fsReadPermsJSON(t, []string{"/workspace/**"}),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{
		Cap:    "fs.read",
		Target: "/etc/passwd",
	})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PermError for path outside grant, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: unknown cap → deny (conservative)
// ─────────────────────────────────────────────

func TestGate_CheckUnknownCap_Deny(t *testing.T) {
	t.Parallel()

	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())

	err := g.Check(context.Background(), "myplugin", bridge.Need{
		Cap:    "somefuturecap",
		Target: "anything",
	})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *PermError for unknown cap, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: session cap checks
// ─────────────────────────────────────────────

func TestGate_CheckSession_ReadGranted(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"session":"read"}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "session", Target: "read"}); err != nil {
		t.Fatalf("expected allow for session:read, got: %v", err)
	}
}

func TestGate_CheckSession_WriteImpliesRead(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"session":"write"}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	// write grant implies read is also allowed
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "session", Target: "read"}); err != nil {
		t.Fatalf("write grant should allow read, got: %v", err)
	}
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "session", Target: "write"}); err != nil {
		t.Fatalf("write grant should allow write, got: %v", err)
	}
}

func TestGate_CheckSession_ReadDeniedForWrite(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"session":"read"}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	err := g.Check(context.Background(), "p", bridge.Need{Cap: "session", Target: "write"})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected EPERM for write when only read granted, got: %v", err)
	}
}

func TestGate_CheckSession_Denied_NoGrant(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	err := g.Check(context.Background(), "p", bridge.Need{Cap: "session", Target: "read"})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected EPERM when no session grant, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: storage + secret cap checks
// ─────────────────────────────────────────────

func TestGate_CheckStorage_Granted(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"storage":true}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "storage", Target: ""}); err != nil {
		t.Fatalf("expected allow for storage:true, got: %v", err)
	}
}

func TestGate_CheckStorage_Denied(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	err := g.Check(context.Background(), "p", bridge.Need{Cap: "storage", Target: ""})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected EPERM when storage not granted, got: %v", err)
	}
}

func TestGate_CheckSecret_Granted(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"secret":true}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "secret", Target: ""}); err != nil {
		t.Fatalf("expected allow for secret:true, got: %v", err)
	}
}

func TestGate_CheckSecret_Denied(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	err := g.Check(context.Background(), "p", bridge.Need{Cap: "secret", Target: ""})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected EPERM when secret not granted, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: exec with bool true/false JSON
// ─────────────────────────────────────────────

func TestGate_CheckExec_BoolTrue(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"exec":true}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "exec", Target: "rm -rf /"}); err != nil {
		t.Fatalf("exec:true should allow any command, got: %v", err)
	}
}

func TestGate_CheckExec_BoolFalse(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"exec":false}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	err := g.Check(context.Background(), "p", bridge.Need{Cap: "exec", Target: "git status"})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("exec:false should deny, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: http with bool true (grants all https)
// ─────────────────────────────────────────────

func TestGate_CheckHTTP_BoolTrue(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"http":true}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "http", Target: "https://example.com/api"}); err != nil {
		t.Fatalf("http:true should allow https, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: fs.write path
// ─────────────────────────────────────────────

func TestGate_CheckFSWrite_Allow(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"fs":{"write":["/workspace/**"]}}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "fs.write", Target: "/workspace/out.txt"}); err != nil {
		t.Fatalf("expected allow for fs.write, got: %v", err)
	}
}

func TestGate_CheckFSWrite_Deny(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"fs":{"read":["/workspace/**"]}}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	err := g.Check(context.Background(), "p", bridge.Need{Cap: "fs.write", Target: "/workspace/out.txt"})
	var pe *bridge.PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected EPERM for fs.write when only read granted, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: fs bool true
// ─────────────────────────────────────────────

func TestGate_CheckFS_BoolTrue(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"fs":true}`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	if err := g.Check(context.Background(), "p", bridge.Need{Cap: "fs.read", Target: "/etc/passwd"}); err != nil {
		t.Fatalf("fs:true should allow any read, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: audit sink error is swallowed (primary path not broken)
// ─────────────────────────────────────────────

func TestGate_AuditSinkError_DoesNotBlockDecision(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{err: errors.New("disk full")}
	consent := &fakeConsent{
		perms: execPermsJSON(t, []string{"git *"}),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	// Check must still return allow even if audit fails.
	err := g.Check(context.Background(), "myplugin", bridge.Need{Cap: "exec", Target: "git status"})
	if err != nil {
		t.Fatalf("audit failure must not fail the primary check, got: %v", err)
	}
}

// ─────────────────────────────────────────────
// Extra: malformed consent JSON → error (not PermError)
// ─────────────────────────────────────────────

func TestGate_MalformedConsentJSON(t *testing.T) {
	t.Parallel()
	audit := &fakeAudit{}
	consent := &fakeConsent{
		perms: []byte(`{"exec": INVALID JSON`),
		found: true,
	}
	g := bridge.NewGate(consent, audit, slog.Default())
	err := g.Check(context.Background(), "p", bridge.Need{Cap: "exec", Target: "git status"})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	var pe *bridge.PermError
	if errors.As(err, &pe) {
		t.Fatal("malformed JSON should return non-PermError")
	}
	// Audit row should record "error"
	if len(audit.events) != 1 || audit.events[0].Result != "error" {
		t.Errorf("expected audit error row, got: %v", audit.events)
	}
}

// ─────────────────────────────────────────────
// Extra: MatchHTTPURL with specific path patterns
// ─────────────────────────────────────────────

func TestMatchHTTPURL_SpecificPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		granted []string
		rawURL  string
		want    bool
	}{
		{
			name:    "specific_path_matches",
			granted: []string{"https://api.github.com/repos/*"},
			rawURL:  "https://api.github.com/repos/foo",
			want:    true,
		},
		{
			name:    "specific_path_no_cross_slash",
			granted: []string{"https://api.github.com/repos/*"},
			rawURL:  "https://api.github.com/repos/foo/bar",
			want:    false,
		},
		{
			name:    "no_path_pattern_allows_root",
			granted: []string{"https://example.com"},
			rawURL:  "https://example.com/anything",
			want:    true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bridge.MatchHTTPURL(tc.granted, tc.rawURL)
			if got != tc.want {
				t.Errorf("MatchHTTPURL(%v, %q) = %v, want %v", tc.granted, tc.rawURL, got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────
// Extra: MatchExecGlobs edge case — single-token pattern with wildcard
// ─────────────────────────────────────────────

func TestMatchExecGlobs_SingleTokenWildcard(t *testing.T) {
	t.Parallel()
	// "git*" as a single-token pattern should match only "git" with no args
	got := bridge.MatchExecGlobs([]string{"git*"}, "git")
	if !got {
		t.Error("single-token 'git*' should match bare 'git'")
	}
	// "git*" should NOT match "git status" (multi-token cmdline with single-token pattern)
	got2 := bridge.MatchExecGlobs([]string{"git*"}, "git status")
	if got2 {
		t.Error("single-token 'git*' should not match 'git status' (extra arg)")
	}
}
