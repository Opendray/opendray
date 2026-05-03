package gateway

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Pure-logic helpers ──────────────────────────────────────────────

func TestOAuthURLPattern(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"verified CLI output line v2.1.126",
			`If the browser didn't open, visit: https://claude.com/cai/oauth/authorize?code=true&client_id=9d1c250a-e61b-44d9-88ed-5944d1962f5e&response_type=code&redirect_uri=https%3A%2F%2Fplatform.claude.com%2Foauth%2Fcode%2Fcallback&scope=org%3Acreate_api_key+user%3Aprofile+user%3Ainference+user%3Asessions%3Aclaude_code+user%3Amcp_servers+user%3Afile_upload&code_challenge=dTYZDhtYVpIw2A-4q64H65SBiJgiraR1JZG2iJTZwck&code_challenge_method=S256&state=SNyJo8_eT4mhpIr6EpodQ05_hKjOb0PbcXwWqRsvjjA`,
			"https://claude.com/cai/oauth/authorize?code=true&client_id=9d1c250a-e61b-44d9-88ed-5944d1962f5e&response_type=code&redirect_uri=https%3A%2F%2Fplatform.claude.com%2Foauth%2Fcode%2Fcallback&scope=org%3Acreate_api_key+user%3Aprofile+user%3Ainference+user%3Asessions%3Aclaude_code+user%3Amcp_servers+user%3Afile_upload&code_challenge=dTYZDhtYVpIw2A-4q64H65SBiJgiraR1JZG2iJTZwck&code_challenge_method=S256&state=SNyJo8_eT4mhpIr6EpodQ05_hKjOb0PbcXwWqRsvjjA",
		},
		{
			"plain URL on a line",
			"https://claude.com/cai/oauth/authorize?code=true&x=1",
			"https://claude.com/cai/oauth/authorize?code=true&x=1",
		},
		{
			"no URL",
			"Opening browser to sign in…",
			"",
		},
		{
			"wrong host — must not match",
			"https://anthropic.com/oauth/authorize?evil=1",
			"",
		},
		{
			"trailing whitespace stripped naturally by regex",
			"  https://claude.com/cai/oauth/authorize?abc",
			"https://claude.com/cai/oauth/authorize?abc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := oauthURLPattern.FindString(tc.in)
			if got != tc.want {
				t.Errorf("FindString:\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestSanitizeFsName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		// '@' is replaced by '-', '.' is preserved (filesystem-safe).
		// Use deriveAccountName for the email→name conversion that
		// also strips dots.
		{"navid@example.com", "navid-example.com"},
		{"a/b/c", "a-b-c"},
		{"  spaces  ", "spaces"},
		{"...dots...", "dots"},
		{"UPPER_lower-Mix", "upper_lower-mix"},
		{"emoji-🚀-strip", "emoji-strip"},
		{"", ""},
		{"---", ""},
		{"valid_name.123", "valid_name.123"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := sanitizeFsName(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeFsName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDeriveAccountName(t *testing.T) {
	t.Parallel()
	t.Run("from email", func(t *testing.T) {
		got := deriveAccountName("navid@example.com", "/tmp/whatever")
		want := "navid-example-com"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("falls back to timestamp when email empty", func(t *testing.T) {
		got := deriveAccountName("", "/tmp/whatever")
		if !strings.HasPrefix(got, "claude-") {
			t.Errorf("expected claude-<timestamp> fallback, got %q", got)
		}
	})
	t.Run("falls back to timestamp when email is unsanitisable", func(t *testing.T) {
		got := deriveAccountName("@@@", "/tmp/whatever")
		if !strings.HasPrefix(got, "claude-") {
			t.Errorf("expected claude-<timestamp> fallback, got %q", got)
		}
	})
}

func TestLastErrorLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in string
		wantSub  string
	}{
		{
			"invalid code",
			"Opening browser…\nPaste code here if prompted > Invalid code. Please make sure the full code was copied.\n",
			"Invalid code",
		},
		{
			"failed login",
			"Opening browser…\nLogin failed: network timeout\n",
			"Login failed",
		},
		{
			"no error pattern → generic fallback",
			"All went smoothly.\nGoodbye.\n",
			"check your paste",
		},
		{
			"empty",
			"",
			"check your paste",
		},
		{
			"picks LAST error, not first",
			"Error: first\nSomething benign\nError: second is the latest\n",
			"second is the latest",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lastErrorLine(tc.in)
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("lastErrorLine:\n got: %q\nwant substring: %q", got, tc.wantSub)
			}
		})
	}
}

func TestNewOAuthFlowID(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := newOAuthFlowID()
		if !strings.HasPrefix(id, "fl-") {
			t.Fatalf("flow id missing prefix: %q", id)
		}
		if len(id) < 10 {
			t.Fatalf("flow id too short: %q", id)
		}
		if seen[id] {
			t.Fatalf("flow id collision after %d iterations: %q", i, id)
		}
		seen[id] = true
	}
}

// ── Preflight handler ───────────────────────────────────────────────

func TestPreflightClaudeOAuth_CLIMissing(t *testing.T) {
	// Force PATH to a directory we know contains no `claude`.
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)
	resetClaudeCLICache()

	srv := &Server{}
	rec := httptest.NewRecorder()
	srv.preflightClaudeOAuth(rec, httptest.NewRequest(http.MethodGet, "/api/claude-accounts/oauth/preflight", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"available":false`) {
		t.Errorf("expected available:false, got: %s", body)
	}
	if !strings.Contains(body, "installHint") {
		t.Errorf("expected installHint to surface, got: %s", body)
	}
}

func TestPreflightClaudeOAuth_CLIPresent(t *testing.T) {
	// Drop a fake `claude` script onto a temp PATH that prints a version.
	tmp := t.TempDir()
	fakeClaude := filepath.Join(tmp, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\necho '2.1.126 (Claude Code Test Stub)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)
	resetClaudeCLICache()

	srv := &Server{}
	rec := httptest.NewRecorder()
	srv.preflightClaudeOAuth(rec, httptest.NewRequest(http.MethodGet, "/api/claude-accounts/oauth/preflight", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"available":true`) {
		t.Errorf("expected available:true, got: %s", body)
	}
	if !strings.Contains(body, "2.1.126") {
		t.Errorf("expected version 2.1.126 to surface, got: %s", body)
	}
	if !strings.Contains(body, fakeClaude) {
		t.Errorf("expected resolved path %q in body, got: %s", fakeClaude, body)
	}
}

// ── Helpers cleanup edge cases ──────────────────────────────────────

func TestCleanupFlow_NilSafe(t *testing.T) {
	t.Parallel()
	// Should not panic on nil flow.
	srv := &Server{}
	srv.cleanupFlow(nil, "nil safety")
}

func TestKillOAuthFlow_NotFound(t *testing.T) {
	t.Parallel()
	srv := &Server{}
	if srv.killOAuthFlow("does-not-exist", "test") {
		t.Errorf("killOAuthFlow returned true for non-existent flowID")
	}
}

// ── Concurrency cap ─────────────────────────────────────────────────

func TestStartClaudeOAuth_ConcurrencyCap(t *testing.T) {
	// Pre-populate the registry up to the cap.
	oauthFlowsMu.Lock()
	for i := 0; i < oauthMaxConcurrent; i++ {
		id := newOAuthFlowID()
		oauthFlows[id] = &oauthFlow{id: id}
	}
	oauthFlowsMu.Unlock()
	t.Cleanup(func() {
		oauthFlowsMu.Lock()
		oauthFlows = make(map[string]*oauthFlow)
		oauthFlowsMu.Unlock()
	})

	// Force PATH to include a fake claude so the CLI-absent check passes
	// and we hit the concurrency check.
	tmp := t.TempDir()
	fakeClaude := filepath.Join(tmp, "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\nsleep 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)
	resetClaudeCLICache()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/claude-accounts/oauth/start",
		strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.startClaudeOAuth(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429 (Too Many Requests); body: %s", rec.Code, rec.Body.String())
	}
}
