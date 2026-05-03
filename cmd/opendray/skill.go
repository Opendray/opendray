// Skill subcommand for the opendray binary. Wraps the skills loader
// for AI-agent and human use. Like the notes subcommand, this runs
// directly against the filesystem — no gateway required.
//
//	opendray skill list                       # all skills (built-in + vault)
//	opendray skill describe <id>              # full SKILL.md to stdout
//	opendray skill path                       # vault skills root

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/skills"
)

func runSkill(args []string) int {
	fs := flag.NewFlagSet("skill", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "path to config.toml (only [vault] is read)")
	root := fs.String("root", "", "override vault root (else config / env / default)")
	asJSON := fs.Bool("json", false, "list/describe: JSON output")
	fs.Usage = func() { fmt.Fprintln(os.Stderr, skillUsage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return 2
	}

	loader, err := openLoader(*cfgPath, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	switch rest[0] {
	case "path":
		fmt.Println(loader.VaultRoot())
		return 0

	case "list":
		all, err := loader.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(all)
			return 0
		}
		for _, s := range all {
			fmt.Printf("%s\t%s\t%s\n", s.ID, s.Source, s.Description)
		}
		return 0

	case "describe":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "describe: missing <id>")
			return 2
		}
		s, err := loader.Get(rest[1])
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "skill %q not found\n", rest[1])
				return 4
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(s)
			return 0
		}
		fmt.Print(s.Body)
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown skill command: %s\n", rest[0])
		fs.Usage()
		return 2
	}
}

// openLoader resolves the vault root the same way the notes CLI does.
// Skills live at <vault>/skills/ — using the same root keeps the user's
// vault one self-contained git-able directory.
func openLoader(cfgPath, override string) (*skills.Loader, error) {
	// Resolve skills directory using the same precedence as the
	// gateway: explicit override → config.toml [vault].skills →
	// env OPENDRAY_VAULT_SKILLS → derive from [vault].root.
	skillsDir := override
	if skillsDir == "" && cfgPath != "" {
		if cfg, err := config.Load(cfgPath); err == nil {
			if cfg.Vault.Skills != "" {
				skillsDir = cfg.Vault.Skills
			} else if cfg.Vault.Root != "" {
				skillsDir = filepath.Join(cfg.Vault.Root, "skills")
			}
		}
	}
	if skillsDir == "" {
		skillsDir = os.Getenv("OPENDRAY_VAULT_SKILLS")
	}
	if skillsDir == "" {
		root := os.Getenv("OPENDRAY_VAULT_ROOT")
		if root == "" {
			root = "~/.opendray/vault"
		}
		skillsDir = filepath.Join(root, "skills")
	}
	if expanded, err := expandHome(skillsDir); err == nil {
		skillsDir = expanded
	}
	// Don't use notes.New here — we don't want to create a notes dir
	// just for `skill` operations. Loader gracefully handles missing
	// vault dirs (only built-ins return).
	return skills.NewLoader(skillsDir), nil
}

func expandHome(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p, err
		}
		if p == "~" {
			return home, nil
		}
		return filepath.Join(home, p[2:]), nil
	}
	return filepath.Abs(p)
}

const skillUsage = `opendray skill — agent skills loader

usage:
  opendray skill [flags] <command> [args]

commands:
  path                    print the skills directory (vault/skills)
  list                    list all skills (built-in + vault); cols: id source description
  describe <id>           print SKILL.md for one skill

flags:
  -config FILE            config.toml (only [vault] is consulted)
  --root PATH             vault root override
  --json                  list/describe: JSON output

skills load from (vault overrides built-in on conflict):
  - <opendray binary>/builtin/<id>/SKILL.md   (shipped)
  - <vault>/skills/<id>/SKILL.md              (user / git-versioned)`
