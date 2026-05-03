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
		path := notes.ProjectPath(rest[1])
		if _, err := vault.Read(path); errors.Is(err, notes.ErrNotFound) {
			template := projectTemplate(rest[1])
			if _, err := vault.Write(path, template); err != nil {
				return reportErr(err)
			}
			fmt.Fprintf(os.Stderr, "created %s\n", path)
		}
		fmt.Println(path)
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown notes command: %s\n", rest[0])
		fs.Usage()
		return 2
	}
}

func openVault(cfgPath, override string) (*notes.Vault, error) {
	root := override
	opts := notes.Options{}
	// Try config file (cheap — we only read [vault]); fall through
	// to env / default if the file doesn't exist.
	if cfgPath != "" {
		if cfg, err := config.Load(cfgPath); err == nil {
			if root == "" {
				if cfg.Vault.Notes != "" {
					root = cfg.Vault.Notes
				} else if cfg.Vault.Root != "" {
					root = cfg.Vault.Root + "/notes"
				}
			}
			opts.PersonalPrefix = cfg.Vault.PersonalPrefix
			opts.ProjectsPrefix = cfg.Vault.ProjectsPrefix
		}
	}
	if root == "" {
		if v := os.Getenv("OPENDRAY_VAULT_NOTES"); v != "" {
			root = v
		} else if v := os.Getenv("OPENDRAY_VAULT_ROOT"); v != "" {
			root = v + "/notes"
		} else {
			root = "~/.opendray/vault/notes"
		}
	}
	return notes.New(root, opts)
}

func reportErr(err error) int {
	switch {
	case errors.Is(err, notes.ErrNotFound):
		fmt.Fprintln(os.Stderr, err)
		return 4
	case errors.Is(err, notes.ErrPathEscape),
		errors.Is(err, notes.ErrInvalidPath),
		errors.Is(err, notes.ErrNotMarkdown):
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

flags:
  -config FILE                  config.toml (only [vault] is consulted)
  --root PATH                   vault root override
  --prefix STRING               list filter (e.g. projects/)
  --json                        list/read JSON output

paths are relative to <vault>/notes and must end in .md.
operates on the filesystem directly — the gateway does not need to be running.`
