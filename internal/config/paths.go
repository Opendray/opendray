package config

import (
	"os"
	"path/filepath"
	"strings"
)

// The Vault used to be a container rather than a concept: one root held
// the operator's documents, the agent skills opendray injects at spawn,
// and the MCP registry. Three tenants with different owners, different
// lifecycles and different git needs, described in the settings UI as
// "notes, skills and git-versioned root" — a sentence that only parses
// if you already know the implementation.
//
// It was not merely confusing. Sharing a directory with the documents
// meant sharing their git repo: on the machine this was found on, the
// operator's private docs repo had opendray's own `skills/secretary/
// SKILL.md` committed into it, and a `git clean -fd` in that repo would
// have deleted the gateway's skills.
//
// So: the Vault is the documents. Skills and the MCP registry live
// beside it under ~/.opendray, next to the other machinery.
//
//	~/.opendray/vault/    documents — the doc library's root, and the
//	                      working tree Vault Sync commits
//	~/.opendray/skills/   agent skills
//	~/.opendray/mcp/      MCP registry
//
// Existing installs are NOT moved. Resolve() honours the old layout
// wherever it finds content there, and reports it so the gateway can
// say so out loud rather than silently reading a different directory
// than the settings page implies.
//
// This file is the ONLY place these paths are derived. They used to be
// computed in four — internal/app plus each of the three CLI commands —
// with subtly different precedence in each, which is how a layout
// becomes unpredictable.

// Paths are the resolved on-disk locations opendray reads and writes.
// Every field is absolute and ~-expanded.
type Paths struct {
	// Vault is the documents root: what the doc library browses, what
	// the notes API and MCP doc tools read and write.
	Vault string `json:"vault"`
	// VaultGit is the git working tree Vault Sync operates on.
	VaultGit string `json:"vault_git"`
	// Skills is the agent-skills root.
	Skills string `json:"skills"`
	// MCP is the MCP registry root.
	MCP string `json:"mcp"`
	// MCPSecrets is the ${KEY} substitution file for mcp.json. Kept
	// outside every other root on purpose, so no `git add .` anywhere
	// can pick it up.
	MCPSecrets string `json:"mcp_secrets"`

	// Legacy* record that a path resolved to the old shared layout
	// because content was already there. Surfaced to the operator
	// rather than silently honoured — the settings page otherwise
	// describes a layout the gateway is not using.
	LegacyVault  bool `json:"legacy_vault"`
	LegacySkills bool `json:"legacy_skills"`
	LegacyMCP    bool `json:"legacy_mcp"`
}

// LegacyLayout reports whether anything still resolves inside the old
// shared vault root.
func (p Paths) LegacyLayout() bool {
	return p.LegacyVault || p.LegacySkills || p.LegacyMCP
}

// NestedInVault returns the machinery roots that resolve INSIDE the
// documents root. The doc library hides these: on a legacy install
// `skills/` and `mcp/` are opendray's own configuration sitting in the
// middle of the operator's folder tree, and presenting them as
// documentation is the visible half of the problem this file exists to
// fix.
func (p Paths) NestedInVault() []string { return p.nestedIn(p.Vault) }

// NestedInGitRoot returns the machinery roots that resolve inside the
// Vault Sync working tree. This is NOT the same set as NestedInVault:
// on an install whose documents are `<root>/notes` while the repo is
// still `<root>`, skills/ is outside the documents yet squarely inside
// the repo — which is how one operator's private docs repo ended up
// with opendray's SKILL.md committed to it. The managed .gitignore
// uses this set.
func (p Paths) NestedInGitRoot() []string { return p.nestedIn(p.VaultGit) }

func (p Paths) nestedIn(base string) []string {
	if base == "" {
		return nil
	}
	var out []string
	for _, dir := range []string{p.Skills, p.MCP} {
		if dir != "" && dir != base && isUnder(base, dir) {
			out = append(out, dir)
		}
	}
	return out
}

// Resolve derives every path from the config. Filesystem state is part
// of the input — an install is "legacy" because content is sitting in
// the old place, which cannot be known from the config alone.
func (c Config) Resolve() Paths {
	return c.resolve(dirHasEntries, dirExists)
}

// resolve takes its filesystem probes as arguments so tests can drive
// every branch without building directory trees.
func (c Config) resolve(hasEntries, exists func(string) bool) Paths {
	root := ExpandPath(firstNonEmpty(c.Vault.Root, "~/.opendray/vault"))
	odHome := opendrayHome()

	var p Paths

	// Documents. `vault.notes` is deprecated but still authoritative
	// when set — it is how operators point opendray at an Obsidian
	// folder they already keep elsewhere.
	switch {
	case strings.TrimSpace(c.Vault.Notes) != "":
		p.Vault = ExpandPath(c.Vault.Notes)
	case hasEntries(filepath.Join(root, "notes")):
		p.Vault = filepath.Join(root, "notes")
		p.LegacyVault = true
	default:
		p.Vault = root
	}

	// Git working tree. An existing repo at the old shared root keeps
	// it: moving someone's sync target out from under them would break
	// a working remote for the sake of tidiness.
	switch {
	case strings.TrimSpace(c.Vault.GitRoot) != "":
		p.VaultGit = ExpandPath(c.Vault.GitRoot)
	case p.LegacyVault && exists(filepath.Join(root, ".git")):
		p.VaultGit = root
	default:
		p.VaultGit = p.Vault
	}

	// Skills.
	switch {
	case strings.TrimSpace(c.Skills.Root) != "":
		p.Skills = ExpandPath(c.Skills.Root)
	case strings.TrimSpace(c.Vault.Skills) != "":
		p.Skills = ExpandPath(c.Vault.Skills)
	case hasEntries(filepath.Join(root, "skills")):
		p.Skills = filepath.Join(root, "skills")
		p.LegacySkills = true
	default:
		p.Skills = filepath.Join(odHome, "skills")
	}

	// MCP registry.
	switch {
	case strings.TrimSpace(c.MCP.Root) != "":
		p.MCP = ExpandPath(c.MCP.Root)
	case hasEntries(filepath.Join(root, "mcp")):
		p.MCP = filepath.Join(root, "mcp")
		p.LegacyMCP = true
	default:
		p.MCP = filepath.Join(odHome, "mcp")
	}

	p.MCPSecrets = ExpandPath(firstNonEmpty(c.MCP.SecretsFile,
		filepath.Join(odHome, "secrets.env")))

	return p
}

// ExpandPath resolves a leading ~ against the calling user's home
// directory and makes the result absolute. Empty stays empty so
// callers can distinguish "unset" from "the current directory".
func ExpandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				p = home
			} else {
				p = filepath.Join(home, p[2:])
			}
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return filepath.Clean(p)
}

func opendrayHome() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".opendray")
	}
	return filepath.Clean(".opendray")
}

// dirHasEntries reports whether path is a directory holding anything at
// all. "Has content" rather than "exists" is the legacy test on
// purpose: an empty leftover directory should not pin an install to the
// old layout, while a directory with a single skill in it must.
func dirHasEntries(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		// Finder droppings are not content.
		if e.Name() == ".DS_Store" {
			continue
		}
		return true
	}
	return false
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// isUnder reports whether child sits inside parent.
func isUnder(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
