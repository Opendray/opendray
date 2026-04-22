package gateway

import (
	"reflect"
	"testing"

	"github.com/opendray/opendray/plugin"
)

func TestExpandUserPath(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("OPENDRAY_TEST_VAR", "/custom/root")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"absolute path untouched", "/var/log", "/var/log"},
		{"$HOME expands", "$HOME", "/home/test"},
		{"${HOME} expands", "${HOME}", "/home/test"},
		{"$HOME prefix", "$HOME/projects", "/home/test/projects"},
		{"~ alone", "~", "/home/test"},
		{"~/ prefix", "~/projects", "/home/test/projects"},
		{"~user form left as-is", "~alice/docs", "~alice/docs"},
		{"other env var", "$OPENDRAY_TEST_VAR/logs", "/custom/root/logs"},
		{"undefined env var drops to empty", "$OPENDRAY_UNDEFINED_VAR_XYZ/x", "/x"},
		{"trims surrounding whitespace", "  $HOME/x  ", "/home/test/x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := expandUserPath(tc.in)
			if got != tc.want {
				t.Errorf("expandUserPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveRoots(t *testing.T) {
	t.Setenv("HOME", "/home/test")

	schema := []plugin.ConfigField{
		{Key: "allowedRoots", Type: "string", Default: "$HOME"},
		{Key: "noDefault", Type: "string"},
		{Key: "multiDefault", Type: "string", Default: "$HOME,/var/log,/tmp"},
		{Key: "nonStringDefault", Type: "number", Default: 42},
	}

	tests := []struct {
		name string
		cfg  map[string]any
		key  string
		want []string
	}{
		{
			name: "user value takes priority",
			cfg:  map[string]any{"allowedRoots": "/a,/b"},
			key:  "allowedRoots",
			want: []string{"/a", "/b"},
		},
		{
			name: "empty user value falls back to schema default",
			cfg:  map[string]any{"allowedRoots": ""},
			key:  "allowedRoots",
			want: []string{"/home/test"},
		},
		{
			name: "whitespace-only user value falls back to schema default",
			cfg:  map[string]any{"allowedRoots": "   "},
			key:  "allowedRoots",
			want: []string{"/home/test"},
		},
		{
			name: "missing cfg key falls back to schema default",
			cfg:  map[string]any{},
			key:  "allowedRoots",
			want: []string{"/home/test"},
		},
		{
			name: "multi default expands each entry",
			cfg:  map[string]any{},
			key:  "multiDefault",
			want: []string{"/home/test", "/var/log", "/tmp"},
		},
		{
			name: "user value with $HOME expands",
			cfg:  map[string]any{"allowedRoots": "$HOME/projects,/data"},
			key:  "allowedRoots",
			want: []string{"/home/test/projects", "/data"},
		},
		{
			name: "no default and empty cfg yields empty slice",
			cfg:  map[string]any{},
			key:  "noDefault",
			want: nil,
		},
		{
			name: "non-string default ignored",
			cfg:  map[string]any{},
			key:  "nonStringDefault",
			want: nil,
		},
		{
			name: "unknown key yields empty",
			cfg:  map[string]any{},
			key:  "unknownKey",
			want: nil,
		},
		{
			name: "blank entries filtered",
			cfg:  map[string]any{"allowedRoots": ",,/a, ,/b,"},
			key:  "allowedRoots",
			want: []string{"/a", "/b"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveRoots(tc.cfg, schema, tc.key)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("resolveRoots(%v, %q) = %v, want %v", tc.cfg, tc.key, got, tc.want)
			}
		})
	}
}

func TestResolveDefaultPath(t *testing.T) {
	t.Setenv("HOME", "/home/test")

	schema := []plugin.ConfigField{
		{Key: "defaultPath", Type: "string", Default: "$HOME"},
		{Key: "noDefault", Type: "string"},
	}

	tests := []struct {
		name string
		cfg  map[string]any
		key  string
		want string
	}{
		{"user value expands", map[string]any{"defaultPath": "$HOME/x"}, "defaultPath", "/home/test/x"},
		{"empty user falls back to default", map[string]any{"defaultPath": ""}, "defaultPath", "/home/test"},
		{"missing key falls back to default", map[string]any{}, "defaultPath", "/home/test"},
		{"no default returns empty", map[string]any{}, "noDefault", ""},
		{"user absolute path", map[string]any{"defaultPath": "/abs/path"}, "defaultPath", "/abs/path"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveDefaultPath(tc.cfg, schema, tc.key)
			if got != tc.want {
				t.Errorf("resolveDefaultPath(%v, %q) = %q, want %q", tc.cfg, tc.key, got, tc.want)
			}
		})
	}
}
