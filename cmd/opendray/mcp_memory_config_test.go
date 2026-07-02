package main

import (
	"os"
	"testing"
)

// loadMemMCPConfig derives the scope key from the process cwd ONLY on
// explicit opt-in (OPENDRAY_MEMORY_SCOPE_FROM_CWD=1 — antigravity's
// HOME-global entry, which can't carry a per-session cwd). Providers
// that pass OPENDRAY_MEMORY_SCOPE_KEY per-session must keep their exact
// value, and an empty key without the flag must stay empty.
func TestLoadMemMCPConfig_ScopeKey(t *testing.T) {
	t.Setenv("OPENDRAY_BASE_URL", "http://127.0.0.1:8770")
	t.Setenv("OPENDRAY_API_KEY", "k")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		scope   string
		key     string
		fromCwd string
		want    string
	}{
		{"explicit key wins", "project", "/some/session/cwd", "", "/some/session/cwd"},
		{"explicit key wins over flag", "project", "/some/session/cwd", "1", "/some/session/cwd"},
		{"no flag, empty key stays empty", "project", "", "", ""},
		{"flag derives from cwd", "project", "", "1", wd},
		{"flag applies regardless of scope", "global", "", "1", wd},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("OPENDRAY_MEMORY_SCOPE", c.scope)
			t.Setenv("OPENDRAY_MEMORY_SCOPE_KEY", c.key)
			t.Setenv("OPENDRAY_MEMORY_SCOPE_FROM_CWD", c.fromCwd)
			cfg, err := loadMemMCPConfig()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.scopeKey != c.want {
				t.Errorf("scopeKey = %q, want %q", cfg.scopeKey, c.want)
			}
		})
	}
}

func TestLoadMemMCPConfig_RequiredEnv(t *testing.T) {
	t.Setenv("OPENDRAY_BASE_URL", "")
	t.Setenv("OPENDRAY_API_KEY", "k")
	if _, err := loadMemMCPConfig(); err == nil {
		t.Error("missing base URL should error")
	}
	t.Setenv("OPENDRAY_BASE_URL", "http://127.0.0.1:8770/")
	t.Setenv("OPENDRAY_API_KEY", "")
	if _, err := loadMemMCPConfig(); err == nil {
		t.Error("missing API key should error")
	}
}
