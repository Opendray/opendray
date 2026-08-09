// Notes subcommand for the opendray binary. Designed for AI agent
// invocation: zero-DB, fast, parses args via positional form so an
// LLM can construct the call without struggling with flags.
//
//	opendray notes list [--prefix=projects/]
//	opendray notes read <path>                 # body to stdout
//	opendray notes write <path>                # body from stdin
//	opendray notes append <path>               # body from stdin
//	opendray notes delete <path>
//	opendray notes daily                       # creates / opens today
//	opendray notes project <basename>          # creates / opens project note
//	opendray notes path                        # print vault root
//
// All operations talk directly to the vault filesystem — the gateway
// process doesn't have to be running. Vault root resolves from
// (in order) -config flag, OPENDRAY_VAULT_ROOT env, ~/.opendray/vault.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/notes"
)

func runNotes(args []string) int {
	fs := flag.NewFlagSet("notes", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "path to config.toml (only [vault] is read)")
	root := fs.String("root", "", "override vault root (else config / env / default)")
	prefix := fs.String("prefix", "", "list: filter by path prefix (e.g. projects/)")
	asJSON := fs.Bool("json", false, "list/read: emit JSON instead of plain text")
	apply := fs.Bool("apply", false, "flatten: perform the moves (default is a dry run)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, notesUsage)
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return 2
	}

	vault, err := openVault(*cfgPath, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	switch rest[0] {
	case "path":
		fmt.Println(vault.Root())
		return 0

	case "list":
		notes_, err := vault.List(*prefix)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(notes_)
			return 0
		}
		for _, n := range notes_ {
			fmt.Printf("%s\t%s\t%s\n",
				n.Path, n.Modified.Format(time.RFC3339), n.Title)
		}
		return 0

	case "read":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "read: missing <path>")
			return 2
		}
		n, err := vault.Read(rest[1])
		if err != nil {
			return reportErr(err)
		}
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(n)
			return 0
		}
		fmt.Print(n.Body)
		return 0

	case "write":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "write: missing <path>")
			return 2
		}
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		n, err := vault.Write(rest[1], string(body))
		if err != nil {
			return reportErr(err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", n.Path, n.Size)
		return 0

	case "append":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "append: missing <path>")
			return 2
		}
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		n, err := vault.Append(rest[1], string(body))
		if err != nil {
			return reportErr(err)
		}
		fmt.Fprintf(os.Stderr, "appended to %s (now %d bytes)\n", n.Path, n.Size)
		return 0

	case "delete":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "delete: missing <path>")
			return 2
		}
		if err := vault.Delete(rest[1]); err != nil {
			return reportErr(err)
		}
		fmt.Fprintf(os.Stderr, "deleted %s\n", rest[1])
		return 0

	case "daily":
		// Convenience: read or create today's daily note. Prints the
		// path so callers can pipe through to other tools.
		path := notes.DailyPath(time.Now())
		if _, err := vault.Read(path); errors.Is(err, notes.ErrNotFound) {
			template := dailyTemplate(time.Now())
			if _, err := vault.Write(path, template); err != nil {
				return reportErr(err)
			}
			fmt.Fprintf(os.Stderr, "created %s\n", path)
		}
		fmt.Println(path)
		return 0

	case "project":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "project: missing <basename>")
			return 2
		}
		// The vault's own derivation, not the package-level shim: the
		// shim hardcodes `projects/` and would file this outside the
		// configured layout.
		path := vault.ProjectPath(rest[1])
		if _, err := vault.Read(path); errors.Is(err, notes.ErrNotFound) {
			template := projectTemplate(rest[1])
			if _, err := vault.Write(path, template); err != nil {
				return reportErr(err)
			}
			fmt.Fprintf(os.Stderr, "created %s\n", path)
		}
		fmt.Println(path)
		return 0

	case "flatten":
		return runFlatten(vault, *cfgPath, *apply, *asJSON)

	default:
		fmt.Fprintf(os.Stderr, "unknown notes command: %s\n", rest[0])
		fs.Usage()
		return 2
	}
}

// runFlatten converts a nested vault to the flat layout. It defaults
// to a dry run: this rewrites every project document's path, and a
// migration that starts moving files because someone typed the command
// to see what it would do is not one anybody should trust.
//
// The config is left alone even on success. Recording `layout = "flat"`
// belongs to the gateway's startup detection, which sees a vault with
// no `projects/` directory and reaches the same conclusion — one place
// deciding, rather than two that can disagree if this run half-fails.
func runFlatten(vault *notes.Vault, cfgPath string, apply, asJSON bool) int {
	res, err := vault.Flatten(context.Background(), !apply)
	if err != nil {
		return reportErr(err)
	}
	if asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(res)
		return 0
	}

	for _, m := range res.Moves {
		if m.LinksRewritten > 0 {
			fmt.Printf("%s -> %s (%d links repointed)\n",
				m.From, m.To, m.LinksRewritten)
			continue
		}
		fmt.Printf("%s -> %s\n", m.From, m.To)
	}
	for _, s := range res.Skips {
		fmt.Fprintf(os.Stderr, "skipped %s: %s\n", s.Path, s.Reason)
	}

	switch {
	case res.DryRun && len(res.Moves) == 0:
		fmt.Fprintln(os.Stderr, "nothing to move")
	case res.DryRun:
		fmt.Fprintf(os.Stderr,
			"\ndry run: %d document(s) would move. Re-run with --apply.\n",
			len(res.Moves))
	default:
		fmt.Fprintf(os.Stderr, "\nmoved %d document(s)", len(res.Moves))
		if res.MappingsRewritten > 0 {
			fmt.Fprintf(os.Stderr, ", repointed %d project override(s)",
				res.MappingsRewritten)
		}
		fmt.Fprintf(os.Stderr, ".\nRestart the gateway so it picks up the "+
			"new layout.\n")
	}
	return 0
}

func openVault(cfgPath, override string) (*notes.Vault, error) {
	cfg := loadCLIConfig(cfgPath)
	paths := cfg.Resolve()
	root := override
	if root == "" {
		root = paths.Vault
	}
	// The CLI reads the layout the gateway recorded rather than
	// detecting one of its own: two components disagreeing about where
	// a project's notes live is precisely what the recorded setting
	// exists to prevent. An unset value means no gateway has started
	// yet, and Options falls back to the nested shape.
	return notes.New(root, notes.Options{
		Layout:         notes.Layout(cfg.Vault.Layout),
		PersonalPrefix: cfg.Vault.PersonalPrefix,
		ProjectsPrefix: cfg.Vault.ProjectsPrefix,
		HiddenDirs:     paths.NestedInVault(),
	})
}

// loadCLIConfig reads the gateway's config so the CLI resolves paths
// identically. A missing or unreadable file is not fatal: Load("")
// returns defaults with the environment applied, which is exactly the
// behaviour the CLI wants when run before the gateway is configured.
//
// This exists so that every `opendray notes|skill|mcp` invocation goes
// through config.Resolve. They each used to derive their own paths
// with slightly different precedence, which meant the CLI could read
// and write a different directory than the running gateway.
func loadCLIConfig(cfgPath string) config.Config {
	if cfgPath != "" {
		if cfg, err := config.Load(cfgPath); err == nil {
			return cfg
		}
	}
	cfg, _ := config.Load("")
	return cfg
}

func reportErr(err error) int {
	switch {
	case errors.Is(err, notes.ErrNotFound):
		fmt.Fprintln(os.Stderr, err)
		return 4
	case errors.Is(err, notes.ErrPathEscape),
		errors.Is(err, notes.ErrInvalidPath),
		errors.Is(err, notes.ErrNotDocument):
		fmt.Fprintln(os.Stderr, err)
		return 2
	default:
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
}

func dailyTemplate(t time.Time) string {
	return fmt.Sprintf(`---
date: %s
type: daily
---

# %s

## What I'm doing

## What I learned

## TODO

`, t.Format("2006-01-02"), t.Format("Monday, January 2, 2006"))
}

func projectTemplate(basename string) string {
	return fmt.Sprintf(`---
project: %s
type: project
created: %s
---

# %s

This is the project's main note (README.md). Drop additional
markdown files in the same directory for specs, decisions, retros, etc.

## Overview

## Status

## Notes

## Open questions

`, basename, time.Now().Format("2006-01-02"), basename)
}

const notesUsage = `opendray notes — file-system notes vault

usage:
  opendray notes [flags] <command> [args]

commands:
  path                          print the vault root
  list [--prefix=PFX]           list notes (newest first)
  read <path>                   write note body to stdout
  write <path>                  read body from stdin, replace note
  append <path>                 read body from stdin, append to note
  delete <path>                 delete a note
  daily                         create or print today's daily note path
  project <basename>            create or print a project note's path
  flatten                       convert a nested vault to the flat layout
                                (dry run unless --apply)

flags (must come BEFORE the command):
  -config FILE                  config.toml (only [vault] is consulted)
  --root PATH                   vault root override
  --prefix STRING               list filter (e.g. projects/)
  --json                        list/read/flatten JSON output
  --apply                       flatten: perform the moves

paths are relative to the vault root and end in .md, .markdown, .html or .htm.
operates on the filesystem directly — the gateway does not need to be running.`
