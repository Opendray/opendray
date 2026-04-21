package plugin

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"
)

// newRuntimeForOverlay builds the smallest Runtime that ResolveCLI
// needs — no DB, no hook bus. Direct field writes keep the test off
// the embedded-Postgres boot path that the other tests in this
// package take (setupTestDB), so it runs in ms instead of seconds.
func newRuntimeForOverlay(t *testing.T) *Runtime {
	t.Helper()
	return &Runtime{
		providers: make(map[string]*liveProvider),
		mu:        sync.RWMutex{},
	}
}

// TestResolveCLI_OverlayMergesConfig verifies that values the v1
// Configure form writes to plugin_kv / plugin_secret (delivered via
// the ConfigOverlayFn) actually reach session spawn — this is the
// Claude bypassPermissions / Gemini yolo fix.
func TestResolveCLI_OverlayMergesConfig(t *testing.T) {
	cases := []struct {
		name       string
		baseCfg    ProviderConfig
		overlayCfg ProviderConfig // extra values the overlay injects on top of baseCfg
		wantArgs   []string
		wantEnv    map[string]string
	}{
		{
			name:    "bool flag — string-form true (kv overlay shape) triggers flag",
			baseCfg: ProviderConfig{},
			// Gateway's effectiveConfig canonicalises booleans to "true"/"false"
			// string. The overlay branch in ResolveCLI must still treat "true"
			// as truthy so the CLI flag is appended.
			overlayCfg: ProviderConfig{"bypassPermissions": "true"},
			wantArgs:   []string{"--dangerously-skip-permissions"},
			wantEnv:    map[string]string{},
		},
		{
			name:       "bool flag — real bool from legacy cache still works",
			baseCfg:    ProviderConfig{"bypassPermissions": true},
			overlayCfg: ProviderConfig{},
			wantArgs:   []string{"--dangerously-skip-permissions"},
			wantEnv:    map[string]string{},
		},
		{
			name:       "bool flag — string 'false' does not trigger flag",
			baseCfg:    ProviderConfig{},
			overlayCfg: ProviderConfig{"bypassPermissions": "false"},
			wantArgs:   []string{},
			wantEnv:    map[string]string{},
		},
		{
			name:    "secret apiKey + custom authType — overlay injects env var",
			baseCfg: ProviderConfig{},
			// authType is non-secret (kv → string). apiKey is secret
			// (plugin_secret → string). Both strings; bool-coerced flag
			// is unrelated.
			overlayCfg: ProviderConfig{
				"authType": "custom",
				"apiKey":   "sk-ant-live",
			},
			wantArgs: []string{},
			wantEnv:  map[string]string{"ANTHROPIC_API_KEY": "sk-ant-live"},
		},
		{
			name:       "authType=env → apiKey not injected even if stored",
			baseCfg:    ProviderConfig{},
			overlayCfg: ProviderConfig{"authType": "env", "apiKey": "ignored"},
			wantArgs:   []string{},
			wantEnv:    map[string]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt := newRuntimeForOverlay(t)

			provider := Provider{
				Name: "claude",
				CLI:  &CLISpec{Command: "claude"},
				ConfigSchema: []ConfigField{
					{Key: "authType", Type: "select", Options: []any{"env", "custom"}},
					{Key: "apiKey", Type: "secret", EnvVar: "ANTHROPIC_API_KEY"},
					{Key: "bypassPermissions", Type: "boolean", CLIFlag: "--dangerously-skip-permissions"},
				},
			}
			rt.providers["claude"] = &liveProvider{
				provider:  provider,
				config:    tc.baseCfg,
				installed: true,
				enabled:   true,
			}
			rt.SetConfigOverlay(func(_ context.Context, name string, base ProviderConfig) ProviderConfig {
				if name != "claude" {
					t.Fatalf("overlay called with %q, want claude", name)
				}
				merged := make(ProviderConfig, len(base)+len(tc.overlayCfg))
				for k, v := range base {
					merged[k] = v
				}
				for k, v := range tc.overlayCfg {
					merged[k] = v
				}
				return merged
			})

			got, ok := rt.ResolveCLI("claude")
			if !ok {
				t.Fatalf("ResolveCLI: not ok")
			}

			// Compare args without caring about ordering for the bool flag
			// case (it's appended after schema iteration, deterministic,
			// but safer to sort for equality).
			sort.Strings(got.Args)
			wantArgs := append([]string{}, tc.wantArgs...)
			sort.Strings(wantArgs)
			if len(got.Args) == 0 && len(wantArgs) == 0 {
				// both empty — ok
			} else if !reflect.DeepEqual(got.Args, wantArgs) {
				t.Errorf("Args = %v, want %v", got.Args, wantArgs)
			}

			if !reflect.DeepEqual(got.Env, tc.wantEnv) {
				t.Errorf("Env = %v, want %v", got.Env, tc.wantEnv)
			}
		})
	}
}

// TestResolveCLI_NoOverlay_UsesBaseConfig is a regression guard: the
// legacy code path (no overlay wired) must still produce identical
// output when callers don't opt in. Guards against accidental
// over-eager overlay lookups and against lock handling that would
// deadlock when the overlay is nil.
func TestResolveCLI_NoOverlay_UsesBaseConfig(t *testing.T) {
	rt := newRuntimeForOverlay(t)
	rt.providers["claude"] = &liveProvider{
		provider: Provider{
			Name: "claude",
			CLI:  &CLISpec{Command: "claude"},
			ConfigSchema: []ConfigField{
				{Key: "bypassPermissions", Type: "boolean", CLIFlag: "--dangerously-skip-permissions"},
			},
		},
		config:    ProviderConfig{"bypassPermissions": true},
		installed: true,
		enabled:   true,
	}

	got, ok := rt.ResolveCLI("claude")
	if !ok {
		t.Fatal("ResolveCLI: not ok")
	}
	want := []string{"--dangerously-skip-permissions"}
	if !reflect.DeepEqual(got.Args, want) {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

// TestBoolVal exercises the coercion helper directly — the overlay
// path depends on it accepting the kv store's canonical string form.
func TestBoolVal(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{true, true},
		{false, false},
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"", false},
		{nil, false},
		{42, false},
	}
	for _, tc := range cases {
		if got := boolVal(tc.in); got != tc.want {
			t.Errorf("boolVal(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
