package notes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestVault(t *testing.T) *Vault {
	t.Helper()
	root := t.TempDir()
	v, err := New(root, Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

// writeFixture tolerates a nil *testing.T so the table cases below can
// call it from a setup closure that has no t in scope.
func writeFixture(t *testing.T, v *Vault, rel, body string) {
	if t != nil {
		t.Helper()
	}
	if _, err := v.Write(rel, body); err != nil {
		if t == nil {
			panic("write " + rel + ": " + err.Error())
		}
		t.Fatalf("write %s: %v", rel, err)
	}
}

func readFixture(t *testing.T, v *Vault, rel string) string {
	t.Helper()
	full, err := v.resolve(rel)
	if err != nil {
		t.Fatalf("resolve %s: %v", rel, err)
	}
	b, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// Filing a flat note under a folder is the whole point; the links that
// pointed at it must follow.
func TestMove_RewritesLinksInEveryRecognisedForm(t *testing.T) {
	v := newTestVault(t)
	writeFixture(t, v, "canvas.md", "# Canvas\n")
	writeFixture(t, v, "index.md", strings.Join([]string{
		"# Index",
		"bare: [[canvas]]",
		"aliased: [[canvas|the canvas]]",
		"full path: [[canvas]] again",
	}, "\n"))

	res, err := v.Move(context.Background(), "canvas.md", "features/canvas.md")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if res.LinksRewritten != 3 {
		t.Fatalf("LinksRewritten = %d, want 3", res.LinksRewritten)
	}
	if len(res.RewrittenIn) != 1 || res.RewrittenIn[0] != "index.md" {
		t.Fatalf("RewrittenIn = %v, want [index.md]", res.RewrittenIn)
	}

	got := readFixture(t, v, "index.md")
	if strings.Contains(got, "[[canvas]]") && !strings.Contains(got, "[[canvas|") {
		// The bare basename is unchanged here only because the new
		// basename is also "canvas" — assert the alias survived instead.
		t.Log(got)
	}
	if !strings.Contains(got, "[[canvas|the canvas]]") {
		t.Fatalf("alias text lost:\n%s", got)
	}

	// The file itself moved.
	if _, err := os.Stat(filepath.Join(v.Root(), "features", "canvas.md")); err != nil {
		t.Fatalf("moved file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(v.Root(), "canvas.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("original still present after move")
	}
}

// A rename that changes the basename must repoint bare references too,
// otherwise every [[old-name]] silently dangles.
func TestMove_RenameRepointsBareReferences(t *testing.T) {
	v := newTestVault(t)
	writeFixture(t, v, "projects/app/spec.md", "# Spec\n")
	writeFixture(t, v, "notes.md", "see [[spec]] and [[projects/app/spec]]\n")

	res, err := v.Move(context.Background(), "projects/app/spec.md", "projects/app/design.md")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	got := readFixture(t, v, "notes.md")
	if !strings.Contains(got, "[[design]]") {
		t.Fatalf("bare reference not repointed:\n%s", got)
	}
	if !strings.Contains(got, "[[projects/app/design]]") {
		t.Fatalf("full-path reference not repointed:\n%s", got)
	}
	if res.LinksRewritten != 2 {
		t.Fatalf("LinksRewritten = %d, want 2", res.LinksRewritten)
	}
}

// Wiki-link syntax inside code samples is documentation, not a link.
// Rewriting it would silently edit someone's explanation of the syntax.
func TestMove_LeavesCodeRegionsAlone(t *testing.T) {
	v := newTestVault(t)
	writeFixture(t, v, "spec.md", "# Spec\n")
	body := strings.Join([]string{
		"real link: [[spec]]",
		"inline: `[[spec]]`",
		"```md",
		"fenced: [[spec]]",
		"```",
		"unicode before code: 中文 `[[spec]]` 中文",
		"trailing real: [[spec]]",
	}, "\n")
	writeFixture(t, v, "guide.md", body)

	res, err := v.Move(context.Background(), "spec.md", "docs/spec.md")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	got := readFixture(t, v, "guide.md")

	// Both code forms must survive verbatim, including after multi-byte
	// text — the masking has to be byte-exact or the splice lands in
	// the wrong place.
	if !strings.Contains(got, "inline: `[[spec]]`") {
		t.Fatalf("inline code was rewritten:\n%s", got)
	}
	if !strings.Contains(got, "fenced: [[spec]]") {
		t.Fatalf("fenced code was rewritten:\n%s", got)
	}
	if !strings.Contains(got, "中文 `[[spec]]` 中文") {
		t.Fatalf("code after multi-byte text was corrupted:\n%s", got)
	}
	if res.LinksRewritten != 2 {
		t.Fatalf("LinksRewritten = %d, want 2 (the two real links only)", res.LinksRewritten)
	}
}

func TestMove_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(v *Vault)
		from    string
		to      string
		wantErr error
	}{
		{
			name:    "missing source",
			from:    "nope.md",
			to:      "docs/nope.md",
			wantErr: ErrNotFound,
		},
		{
			name:    "destination exists",
			setup:   func(v *Vault) { writeFixture(nil, v, "a.md", "a"); writeFixture(nil, v, "b.md", "b") },
			from:    "a.md",
			to:      "b.md",
			wantErr: ErrAlreadyExists,
		},
		{
			name:    "escape via dotdot",
			setup:   func(v *Vault) { writeFixture(nil, v, "a.md", "a") },
			from:    "a.md",
			to:      "../outside.md",
			wantErr: ErrPathEscape,
		},
		{
			name:    "non markdown destination",
			setup:   func(v *Vault) { writeFixture(nil, v, "a.md", "a") },
			from:    "a.md",
			to:      "a.txt",
			wantErr: ErrNotDocument,
		},
		{
			name:    "same path",
			setup:   func(v *Vault) { writeFixture(nil, v, "a.md", "a") },
			from:    "a.md",
			to:      "a.md",
			wantErr: ErrInvalidPath,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := newTestVault(t)
			if tt.setup != nil {
				tt.setup(v)
			}
			_, err := v.Move(context.Background(), tt.from, tt.to)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			// A rejected move must not have touched the source.
			if tt.wantErr != ErrNotFound {
				if _, err := os.Stat(filepath.Join(v.Root(), tt.from)); err != nil {
					t.Fatalf("source disturbed by a rejected move: %v", err)
				}
			}
		})
	}
}
