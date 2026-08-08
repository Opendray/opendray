package config

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeFS answers the two filesystem probes Resolve needs from a fixed
// set of paths, so every branch is reachable without building trees.
type fakeFS struct {
	withEntries map[string]bool
	existing    map[string]bool
}

func (f fakeFS) hasEntries(p string) bool { return f.withEntries[p] }
func (f fakeFS) exists(p string) bool     { return f.existing[p] }

func home(t *testing.T) string {
	t.Helper()
	h, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory in this environment")
	}
	return h
}

// A fresh install must land on the split layout: the Vault holds
// documents and nothing else, and the machinery lives beside it.
func TestResolve_FreshInstallSplitsTheRoots(t *testing.T) {
	h := home(t)
	p := Config{}.resolve(fakeFS{}.hasEntries, fakeFS{}.exists)

	want := Paths{
		Vault:      filepath.Join(h, ".opendray", "vault"),
		VaultGit:   filepath.Join(h, ".opendray", "vault"),
		Skills:     filepath.Join(h, ".opendray", "skills"),
		MCP:        filepath.Join(h, ".opendray", "mcp"),
		MCPSecrets: filepath.Join(h, ".opendray", "secrets.env"),
	}
	if p != want {
		t.Fatalf("fresh install resolved to\n %+v\nwant\n %+v", p, want)
	}
	if p.LegacyLayout() {
		t.Fatal("a fresh install must not be reported as legacy")
	}
}

// The whole point of the "has entries" probe: an install that already
// keeps notes and skills under the shared root keeps using them. Moving
// someone's files on upgrade is not an option.
func TestResolve_LegacyLayoutIsHonouredAndReported(t *testing.T) {
	h := home(t)
	root := filepath.Join(h, ".opendray", "vault")
	fs := fakeFS{withEntries: map[string]bool{
		filepath.Join(root, "notes"):  true,
		filepath.Join(root, "skills"): true,
		filepath.Join(root, "mcp"):    true,
	}}

	p := Config{}.resolve(fs.hasEntries, fs.exists)

	if p.Vault != filepath.Join(root, "notes") {
		t.Errorf("documents = %q, want the legacy notes/ dir", p.Vault)
	}
	if p.Skills != filepath.Join(root, "skills") {
		t.Errorf("skills = %q, want the legacy skills/ dir", p.Skills)
	}
	if p.MCP != filepath.Join(root, "mcp") {
		t.Errorf("mcp = %q, want the legacy mcp/ dir", p.MCP)
	}
	if !p.LegacyVault || !p.LegacySkills || !p.LegacyMCP || !p.LegacyLayout() {
		t.Errorf("legacy flags not set: %+v", p)
	}
}

// An EMPTY leftover directory must not pin an install to the old
// layout — otherwise a stray mkdir silently decides where documents
// live. This is why the probe asks for content, not existence.
func TestResolve_EmptyLegacyDirsDoNotPin(t *testing.T) {
	h := home(t)
	p := Config{}.resolve(fakeFS{}.hasEntries, fakeFS{}.exists)

	if p.Vault != filepath.Join(h, ".opendray", "vault") {
		t.Errorf("documents = %q, want the vault root itself", p.Vault)
	}
	if p.LegacyLayout() {
		t.Error("empty dirs must not mark the install legacy")
	}
}

// A pre-split install syncing from the shared root has a real git repo
// there with a real remote. Repointing the working tree at notes/ would
// break a setup that works, for tidiness.
func TestResolve_ExistingRepoAtSharedRootKeepsTheWorkingTree(t *testing.T) {
	h := home(t)
	root := filepath.Join(h, ".opendray", "vault")
	fs := fakeFS{
		withEntries: map[string]bool{filepath.Join(root, "notes"): true},
		existing:    map[string]bool{filepath.Join(root, ".git"): true},
	}

	p := fs.resolveWith(t, Config{})

	if p.VaultGit != root {
		t.Errorf("git root = %q, want the existing repo at %q", p.VaultGit, root)
	}
	if p.Vault != filepath.Join(root, "notes") {
		t.Errorf("documents = %q, want notes/", p.Vault)
	}
}

// Without a legacy notes/ dir there is no reason to keep the git root
// anywhere but on the documents themselves.
func TestResolve_GitRootFollowsTheDocuments(t *testing.T) {
	fs := fakeFS{}
	p := fs.resolveWith(t, Config{Vault: VaultConfig{Root: "/srv/docs"}})
	if p.VaultGit != "/srv/docs" {
		t.Errorf("git root = %q, want /srv/docs", p.VaultGit)
	}
}

func (f fakeFS) resolveWith(t *testing.T, c Config) Paths {
	t.Helper()
	return c.resolve(f.hasEntries, f.exists)
}

// Explicit config always wins over both the legacy probe and the
// defaults — including the deprecated spellings, which are how
// operators point opendray at an Obsidian folder kept elsewhere.
func TestResolve_ExplicitConfigWins(t *testing.T) {
	root := filepath.Join(home(t), ".opendray", "vault")
	fs := fakeFS{withEntries: map[string]bool{
		filepath.Join(root, "notes"):  true,
		filepath.Join(root, "skills"): true,
		filepath.Join(root, "mcp"):    true,
	}}

	// Each case pulls the one field it is about out of the result,
	// since the rest is covered by the layout tests above.
	tests := []struct {
		name string
		cfg  Config
		get  func(Paths) string
		want string
	}{
		{
			name: "vault.notes still points the documents elsewhere",
			cfg:  Config{Vault: VaultConfig{Notes: "/srv/obsidian"}},
			get:  func(p Paths) string { return p.Vault },
			want: "/srv/obsidian",
		},
		{
			name: "skills.root beats the legacy dir",
			cfg:  Config{Skills: SkillsConfig{Root: "/srv/skills"}},
			get:  func(p Paths) string { return p.Skills },
			want: "/srv/skills",
		},
		{
			name: "deprecated vault.skills still works",
			cfg:  Config{Vault: VaultConfig{Skills: "/srv/old-skills"}},
			get:  func(p Paths) string { return p.Skills },
			want: "/srv/old-skills",
		},
		{
			name: "skills.root beats vault.skills",
			cfg: Config{
				Skills: SkillsConfig{Root: "/srv/new"},
				Vault:  VaultConfig{Skills: "/srv/old"},
			},
			get:  func(p Paths) string { return p.Skills },
			want: "/srv/new",
		},
		{
			name: "mcp.root beats the legacy dir",
			cfg:  Config{MCP: MCPConfig{Root: "/srv/mcp"}},
			get:  func(p Paths) string { return p.MCP },
			want: "/srv/mcp",
		},
		{
			name: "git_root beats everything",
			cfg:  Config{Vault: VaultConfig{GitRoot: "/srv/repo"}},
			get:  func(p Paths) string { return p.VaultGit },
			want: "/srv/repo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.get(fs.resolveWith(t, tt.cfg)); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Secrets must never end up inside a directory anything git-adds.
func TestResolve_SecretsStayOutsideEveryRoot(t *testing.T) {
	root := filepath.Join(home(t), ".opendray", "vault")
	fs := fakeFS{withEntries: map[string]bool{
		filepath.Join(root, "notes"): true,
	}}
	p := fs.resolveWith(t, Config{})

	for _, dir := range []string{p.Vault, p.VaultGit, p.Skills, p.MCP} {
		if isUnder(dir, p.MCPSecrets) {
			t.Fatalf("secrets file %q sits under %q", p.MCPSecrets, dir)
		}
	}
}

// NestedInVault drives the doc library's filter. It must name exactly
// the machinery dirs that would otherwise show up as folders in the
// operator's documents — and nothing on a clean split install.
func TestNestedInVault(t *testing.T) {
	root := filepath.Join(home(t), ".opendray", "vault")

	legacy := fakeFS{withEntries: map[string]bool{
		filepath.Join(root, "notes"):  true,
		filepath.Join(root, "skills"): true,
		filepath.Join(root, "mcp"):    true,
	}}.resolveWith(t, Config{Vault: VaultConfig{Notes: root}})

	nested := legacy.NestedInVault()
	if len(nested) != 2 {
		t.Fatalf("legacy install: nested = %v, want skills/ and mcp/", nested)
	}

	clean := fakeFS{}.resolveWith(t, Config{})
	if got := clean.NestedInVault(); len(got) != 0 {
		t.Fatalf("split install: nested = %v, want none", got)
	}
}

// dirHasEntries is the real probe; a directory holding only Finder
// droppings is empty as far as layout decisions go.
func TestDirHasEntries(t *testing.T) {
	dir := t.TempDir()
	if dirHasEntries(filepath.Join(dir, "missing")) {
		t.Error("missing dir reported as having entries")
	}

	empty := filepath.Join(dir, "empty")
	if err := os.MkdirAll(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	if dirHasEntries(empty) {
		t.Error("empty dir reported as having entries")
	}

	if err := os.WriteFile(filepath.Join(empty, ".DS_Store"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if dirHasEntries(empty) {
		t.Error(".DS_Store alone must not count as content")
	}

	if err := os.WriteFile(filepath.Join(empty, "note.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !dirHasEntries(empty) {
		t.Error("dir with a file reported as empty")
	}
}
