package notes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTemplates_BuiltinsAndVaultOverrides(t *testing.T) {
	v := newTestVault(t)

	ids := map[string]Template{}
	for _, tpl := range v.Templates() {
		ids[tpl.ID] = tpl
	}
	for _, want := range []string{"blank", "feature", "decision", "runbook"} {
		if _, ok := ids[want]; !ok {
			t.Fatalf("built-in template %q missing", want)
		}
		if ids[want].Source != "builtin" {
			t.Fatalf("%q source = %q, want builtin", want, ids[want].Source)
		}
	}
	if got := v.Templates()[0].ID; got != "blank" {
		t.Fatalf("first template = %q, want blank (it is the safe default)", got)
	}

	// A vault file takes over its id, and a new file adds one — the
	// point being that a project can change the shape of its docs
	// without a gateway release.
	dir := filepath.Join(v.Root(), TemplateDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature.md"), []byte("MINE {{title}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "postmortem.md"), []byte("# {{title}}"), 0o600); err != nil {
		t.Fatal(err)
	}

	ids = map[string]Template{}
	for _, tpl := range v.Templates() {
		ids[tpl.ID] = tpl
	}
	if ids["feature"].Source != "vault" || !strings.HasPrefix(ids["feature"].Body, "MINE") {
		t.Fatalf("vault override ignored: %+v", ids["feature"])
	}
	if _, ok := ids["postmortem"]; !ok {
		t.Fatal("vault-only template not picked up")
	}
	if ids["blank"].Source != "builtin" {
		t.Fatal("unrelated built-in was clobbered")
	}
}

func TestNewFromTemplate_RendersPlaceholders(t *testing.T) {
	v := newTestVault(t)
	dir := filepath.Join(v.Root(), TemplateDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "t={{title}} s={{slug}} d={{date}} p={{path}}"
	if err := os.WriteFile(filepath.Join(dir, "probe.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := v.NewFromTemplate("features/host-power.md", "probe"); err != nil {
		t.Fatalf("NewFromTemplate: %v", err)
	}
	got := readFixture(t, v, "features/host-power.md")
	want := "t=Host power s=host-power d=" + time.Now().Format("2006-01-02") +
		" p=features/host-power.md"
	if got != want {
		t.Fatalf("rendered = %q, want %q", got, want)
	}
}

// The template shapes are the product here; a feature note that comes
// out without frontmatter isn't one.
func TestNewFromTemplate_BuiltinShapes(t *testing.T) {
	tests := []struct {
		template string
		contains []string
	}{
		{"feature", []string{"type: feature", "# Host power", "## Gotchas"}},
		{"decision", []string{"type: decision", "## Context", "## Consequences"}},
		{"runbook", []string{"type: runbook", "## Verification"}},
		{"blank", []string{"# Host power"}},
		{"", []string{"# Host power"}}, // empty = blank
	}
	for _, tt := range tests {
		t.Run("template="+tt.template, func(t *testing.T) {
			v := newTestVault(t)
			if _, err := v.NewFromTemplate("host-power.md", tt.template); err != nil {
				t.Fatalf("NewFromTemplate: %v", err)
			}
			got := readFixture(t, v, "host-power.md")
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q in:\n%s", want, got)
				}
			}
			if strings.Contains(got, "{{") {
				t.Fatalf("unrendered placeholder left behind:\n%s", got)
			}
		})
	}
}

func TestNewFromTemplate_Rejects(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		template string
		setup    func(*Vault)
		wantErr  error
	}{
		{
			name:     "existing note is never overwritten",
			path:     "a.md",
			template: "feature",
			setup:    func(v *Vault) { writeFixture(nil, v, "a.md", "PRECIOUS") },
			wantErr:  ErrAlreadyExists,
		},
		{
			name:     "unknown template",
			path:     "a.md",
			template: "nope",
			wantErr:  ErrInvalidPath,
		},
		{
			name:     "escape attempt",
			path:     "../a.md",
			template: "blank",
			wantErr:  ErrPathEscape,
		},
		{
			name:     "non markdown",
			path:     "a.txt",
			template: "blank",
			wantErr:  ErrNotMarkdown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newTestVault(t)
			if tt.setup != nil {
				tt.setup(v)
			}
			_, err := v.NewFromTemplate(tt.path, tt.template)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == ErrAlreadyExists {
				if got := readFixture(t, v, tt.path); got != "PRECIOUS" {
					t.Fatalf("existing note was modified: %q", got)
				}
			}
		})
	}
}

func TestTitleCase(t *testing.T) {
	tests := map[string]string{
		"host-power":   "Host power",
		"canvas":       "Canvas",
		"my_note_here": "My note here",
		"":             "",
	}
	for in, want := range tests {
		if got := titleCase(in); got != want {
			t.Fatalf("titleCase(%q) = %q, want %q", in, got, want)
		}
	}
}
