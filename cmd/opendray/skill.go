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

	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/skills"
)

func runSkill(args []string) int {
	fs := flag.NewFlagSet("skill", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "path to config.toml (paths are resolved as the gateway does)")
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

// openLoader resolves the skills root through the same resolver the
// gateway uses, so `opendray skill` always operates on the directory
// the running gateway actually injects from.
func openLoader(cfgPath, override string) (*skills.Loader, error) {
	skillsDir := override
	if skillsDir == "" {
		skillsDir = loadCLIConfig(cfgPath).Resolve().Skills
	} else {
		skillsDir = config.ExpandPath(skillsDir)
	}
	// Don't use notes.New here — we don't want to create a directory
	// just for `skill` operations. Loader gracefully handles a missing
	// root (only built-ins return).
	return skills.NewLoader(skillsDir), nil
}

const skillUsage = `opendray skill — agent skills loader

usage:
  opendray skill [flags] <command> [args]

commands:
  path                    print the skills directory
  list                    list all skills (built-in + vault); cols: id source description
  describe <id>           print SKILL.md for one skill

flags:
  -config FILE            config.toml (only [vault] is consulted)
  --root PATH             vault root override
  --json                  list/describe: JSON output

skills load from (vault overrides built-in on conflict):
  - <opendray binary>/builtin/<id>/SKILL.md   (shipped)
  - <skills root>/<id>/SKILL.md               (user / git-versioned)`
