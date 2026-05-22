package catalog

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestApplyConfigSchema(t *testing.T) {
	type tc struct {
		name     string
		schema   []ConfigField
		cfg      map[string]any
		wantArgs []string
		wantEnv  map[string]string
	}

	cases := []tc{
		{
			name: "empty schema and cfg",
		},
		{
			name: "boolean true with cliFlag",
			schema: []ConfigField{
				{Key: "bypass", Type: "boolean", CliFlag: "--dangerously-skip-permissions"},
			},
			cfg:      map[string]any{"bypass": true},
			wantArgs: []string{"--dangerously-skip-permissions"},
		},
		{
			name: "boolean false is skipped",
			schema: []ConfigField{
				{Key: "bypass", Type: "boolean", CliFlag: "--dangerously-skip-permissions"},
			},
			cfg: map[string]any{"bypass": false},
		},
		{
			name: "boolean true without cliFlag is skipped",
			schema: []ConfigField{
				{Key: "skills_disabled", Type: "boolean"},
			},
			cfg: map[string]any{"skills_disabled": true},
		},
		{
			name: "select with cliValue and non-empty value",
			schema: []ConfigField{
				{Key: "approval", Type: "select", CliFlag: "--approval-mode", CliValue: true},
			},
			cfg:      map[string]any{"approval": "full-auto"},
			wantArgs: []string{"--approval-mode", "full-auto"},
		},
		{
			name: "select with cliValue but empty string is skipped",
			schema: []ConfigField{
				{Key: "sandbox", Type: "select", CliFlag: "-s", CliValue: true},
			},
			cfg: map[string]any{"sandbox": ""},
		},
		{
			name: "secret with envVar and dependsOn satisfied",
			schema: []ConfigField{
				{Key: "apiKey", Type: "secret", EnvVar: "ANTHROPIC_API_KEY",
					DependsOn: "authType", DependsVal: "custom"},
			},
			cfg:     map[string]any{"authType": "custom", "apiKey": "sk-ant-xxx"},
			wantEnv: map[string]string{"ANTHROPIC_API_KEY": "sk-ant-xxx"},
		},
		{
			name: "secret with envVar but dependsOn unsatisfied",
			schema: []ConfigField{
				{Key: "apiKey", Type: "secret", EnvVar: "ANTHROPIC_API_KEY",
					DependsOn: "authType", DependsVal: "custom"},
			},
			cfg: map[string]any{"authType": "env", "apiKey": "sk-ant-xxx"},
		},
		{
			name: "secret with envVar but dependsOn key absent",
			schema: []ConfigField{
				{Key: "apiKey", Type: "secret", EnvVar: "ANTHROPIC_API_KEY",
					DependsOn: "authType", DependsVal: "custom"},
			},
			cfg: map[string]any{"apiKey": "sk-ant-xxx"},
		},
		{
			name: "args from []any",
			schema: []ConfigField{
				{Key: "extraArgs", Type: "args"},
			},
			cfg:      map[string]any{"extraArgs": []any{"--verbose", "--foo", ""}},
			wantArgs: []string{"--verbose", "--foo"},
		},
		{
			name: "args from []string",
			schema: []ConfigField{
				{Key: "extraArgs", Type: "args"},
			},
			cfg:      map[string]any{"extraArgs": []string{"--debug"}},
			wantArgs: []string{"--debug"},
		},
		{
			name: "number with cliFlag and cliValue (json.Number from JSONB)",
			schema: []ConfigField{
				{Key: "maxTurns", Type: "number", CliFlag: "--max-turns", CliValue: true},
			},
			cfg:      map[string]any{"maxTurns": float64(7)},
			wantArgs: []string{"--max-turns", "7"},
		},
		{
			name: "number 0 is rendered (caller decides semantics)",
			schema: []ConfigField{
				{Key: "n", Type: "number", CliFlag: "--n", CliValue: true},
			},
			cfg:      map[string]any{"n": float64(0)},
			wantArgs: []string{"--n", "0"},
		},
		{
			name: "string with envVar (non-secret) sets env",
			schema: []ConfigField{
				{Key: "model", Type: "string", EnvVar: "OPENDRAY_MODEL"},
			},
			cfg:     map[string]any{"model": "claude-opus-4-7"},
			wantEnv: map[string]string{"OPENDRAY_MODEL": "claude-opus-4-7"},
		},
		{
			name: "missing key in cfg is skipped",
			schema: []ConfigField{
				{Key: "bypass", Type: "boolean", CliFlag: "--bypass"},
				{Key: "extraArgs", Type: "args"},
			},
			cfg: map[string]any{},
		},
		{
			name: "schema iteration order is preserved",
			schema: []ConfigField{
				{Key: "yolo", Type: "boolean", CliFlag: "--yolo"},
				{Key: "sandbox", Type: "select", CliFlag: "-s", CliValue: true},
				{Key: "extraArgs", Type: "args"},
			},
			cfg: map[string]any{
				"yolo":      true,
				"sandbox":   "none",
				"extraArgs": []any{"--debug"},
			},
			wantArgs: []string{"--yolo", "-s", "none", "--debug"},
		},
		{
			name: "field with neither cliFlag nor envVar is skipped",
			schema: []ConfigField{
				{Key: "command", Type: "string"},
			},
			cfg: map[string]any{"command": "/usr/local/bin/claude"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotArgs, gotEnv := applyConfigSchema(c.schema, c.cfg)
			if !reflect.DeepEqual(gotArgs, c.wantArgs) {
				t.Errorf("args: got %#v, want %#v", gotArgs, c.wantArgs)
			}
			if !reflect.DeepEqual(gotEnv, c.wantEnv) {
				t.Errorf("env: got %#v, want %#v", gotEnv, c.wantEnv)
			}
		})
	}
}

// TestApplyConfigSchema_RealManifests pins the wiring against the actual
// embedded JSON so a manifest tweak can't silently break spawn behavior
// without flipping a test.
func TestApplyConfigSchema_RealManifests(t *testing.T) {
	manifests, _, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("claude bypassPermissions+extraArgs", func(t *testing.T) {
		m := manifests["claude"]
		args, env := applyConfigSchema(m.ConfigSchema, map[string]any{
			"bypassPermissions": true,
			"extraArgs":         []any{"--verbose"},
		})
		want := []string{"--dangerously-skip-permissions", "--verbose"}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args: got %#v, want %#v", args, want)
		}
		if env != nil {
			t.Errorf("env: got %#v, want nil", env)
		}
	})

	t.Run("claude apiKey custom", func(t *testing.T) {
		m := manifests["claude"]
		_, env := applyConfigSchema(m.ConfigSchema, map[string]any{
			"authType": "custom",
			"apiKey":   "sk-ant-test",
		})
		if got := env["ANTHROPIC_API_KEY"]; got != "sk-ant-test" {
			t.Errorf("ANTHROPIC_API_KEY: got %q, want sk-ant-test", got)
		}
	})

	t.Run("claude apiKey ignored when authType=env", func(t *testing.T) {
		m := manifests["claude"]
		_, env := applyConfigSchema(m.ConfigSchema, map[string]any{
			"authType": "env",
			"apiKey":   "sk-ant-test",
		})
		if _, ok := env["ANTHROPIC_API_KEY"]; ok {
			t.Errorf("expected no ANTHROPIC_API_KEY override; got %#v", env)
		}
	})

	t.Run("codex approval+sandbox", func(t *testing.T) {
		m := manifests["codex"]
		args, _ := applyConfigSchema(m.ConfigSchema, map[string]any{
			"approval": "never",
			"sandbox":  "workspace-write",
		})
		want := []string{"--ask-for-approval", "never", "-s", "workspace-write"}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args: got %#v, want %#v", args, want)
		}
	})

	t.Run("codex apiKey custom", func(t *testing.T) {
		m := manifests["codex"]
		_, env := applyConfigSchema(m.ConfigSchema, map[string]any{
			"authType": "custom",
			"apiKey":   "sk-test",
		})
		if got := env["OPENAI_API_KEY"]; got != "sk-test" {
			t.Errorf("OPENAI_API_KEY: got %q, want sk-test", got)
		}
	})

	t.Run("gemini yolo+sandbox+extraArgs", func(t *testing.T) {
		m := manifests["gemini"]
		args, _ := applyConfigSchema(m.ConfigSchema, map[string]any{
			"yolo":      true,
			"sandbox":   "none",
			"extraArgs": []any{"--debug"},
		})
		want := []string{"--yolo", "-s", "none", "--debug"}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args: got %#v, want %#v", args, want)
		}
	})

	t.Run("gemini empty sandbox is dropped", func(t *testing.T) {
		m := manifests["gemini"]
		args, _ := applyConfigSchema(m.ConfigSchema, map[string]any{
			"sandbox": "",
		})
		if args != nil {
			t.Errorf("got %#v, want nil", args)
		}
	})

	t.Run("claude command override is not surfaced as an arg", func(t *testing.T) {
		m := manifests["claude"]
		args, env := applyConfigSchema(m.ConfigSchema, map[string]any{
			"command": "/usr/local/bin/claude",
		})
		if args != nil {
			t.Errorf("args: got %#v, want nil (command is consumed by Resolve)", args)
		}
		if env != nil {
			t.Errorf("env: got %#v, want nil", env)
		}
	})

	t.Run("config decoded via encoding/json behaves the same", func(t *testing.T) {
		// Mimic JSONB → map[string]any round-trip from store.go: numbers
		// arrive as float64, arrays as []any, bools as bool. This guards
		// against regressions in stringifyArgs / stringifyScalar should
		// the store layer change its decoder.
		raw := `{"bypassPermissions": true, "extraArgs": ["--a", "--b"]}`
		var cfg map[string]any
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			t.Fatal(err)
		}
		args, _ := applyConfigSchema(manifests["claude"].ConfigSchema, cfg)
		want := []string{"--dangerously-skip-permissions", "--a", "--b"}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args: got %#v, want %#v", args, want)
		}
	})
}

func TestModelArgs(t *testing.T) {
	cli := Manifest{ModelFlag: "--model"}
	cases := []struct {
		name string
		m    Manifest
		cfg  map[string]any
		want []string
	}{
		{"default set", cli, map[string]any{"model": "opus"}, []string{"--model", "opus"}},
		{"no default", cli, map[string]any{}, nil},
		{"empty default", cli, map[string]any{"model": ""}, nil},
		{"non-string default", cli, map[string]any{"model": 3}, nil},
		{"no model flag", Manifest{}, map[string]any{"model": "opus"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := modelArgs(c.m, c.cfg)
			if len(got) != len(c.want) {
				t.Fatalf("modelArgs=%v want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("modelArgs[%d]=%q want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}
