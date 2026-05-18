// `opendray providers` — manage the AI CLIs opendray spawns into sessions.
//
//	opendray providers list                       detect installed CLIs + their versions
//	opendray providers list --json                same but machine-readable
//	opendray providers update                     re-run `npm install -g` for each detected CLI
//	opendray providers update --check             show current vs npm-latest, don't install
//	opendray providers update --only claude,gemini  restrict to a subset (comma-separated)
//
// The "providers" here are the Claude / Codex / Gemini CLIs the user
// installed during the install wizard. opendray doesn't bundle them —
// it spawns whatever's on $PATH. This subcommand just centralises the
// "is it installed, what version, npm update it" busywork.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type providerSpec struct {
	Bin     string // CLI name on $PATH
	NpmPkg  string // npm package name
	Display string // human label
}

// Order here is the same order the install wizard offers them.
var providerCatalog = []providerSpec{
	{Bin: "claude", NpmPkg: "@anthropic-ai/claude-code", Display: "Claude Code"},
	{Bin: "gemini", NpmPkg: "@google/gemini-cli", Display: "Gemini CLI"},
	{Bin: "codex", NpmPkg: "@openai/codex", Display: "Codex CLI"},
}

func runProviders(args []string) int {
	if len(args) < 1 {
		providersUsage()
		return 2
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "list":
		return providersList(rest)
	case "update":
		return providersUpdate(rest)
	case "-h", "--help", "help":
		providersUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown providers subcommand: %s\n", sub)
		providersUsage()
		return 2
	}
}

func providersUsage() {
	fmt.Fprintln(os.Stderr, `opendray providers — manage the AI CLIs opendray spawns

usage:
  opendray providers list                       detect installed CLIs + their versions
  opendray providers list --json                machine-readable form
  opendray providers update                     re-run "npm install -g <pkg>" for each detected CLI
  opendray providers update --check             show current vs npm-latest; don't install
  opendray providers update --only claude,gemini  restrict to a subset (comma-separated CLI names)

Privilege: npm install -g typically writes under /usr/lib/node_modules
and needs root. Run via sudo (or whatever wrapper your npm install uses)
when --check is not set.`)
}

// ── list ─────────────────────────────────────────────────────────────

type providerRow struct {
	Bin       string `json:"bin"`
	NpmPkg    string `json:"npm_pkg"`
	Display   string `json:"display"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
}

func providersList(args []string) int {
	fs := flag.NewFlagSet("providers list", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "emit one row per provider as JSON")
	_ = fs.Parse(args)

	rows := make([]providerRow, 0, len(providerCatalog))
	for _, p := range providerCatalog {
		r := providerRow{Bin: p.Bin, NpmPkg: p.NpmPkg, Display: p.Display}
		if path, err := exec.LookPath(p.Bin); err == nil {
			r.Installed = true
			r.Path = path
			r.Version = cliVersion(p.Bin)
		}
		rows = append(rows, r)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return jsonOK(enc.Encode(rows))
	}

	// Human table.
	fmt.Printf("%-12s %-30s %-18s %s\n", "CLI", "NPM PACKAGE", "VERSION", "PATH")
	fmt.Printf("%-12s %-30s %-18s %s\n", "---", "-----------", "-------", "----")
	for _, r := range rows {
		path := r.Path
		if path == "" {
			path = "(not installed)"
		}
		ver := r.Version
		if ver == "" {
			ver = "-"
		}
		fmt.Printf("%-12s %-30s %-18s %s\n", r.Bin, r.NpmPkg, ver, path)
	}
	return 0
}

// ── update ───────────────────────────────────────────────────────────

func providersUpdate(args []string) int {
	fs := flag.NewFlagSet("providers update", flag.ExitOnError)
	checkOnly := fs.Bool("check", false, "print current vs npm-latest; don't install")
	onlyFlag := fs.String("only", "", "comma-separated CLI names to include (default: all)")
	_ = fs.Parse(args)

	only := splitCSV(*onlyFlag)
	rc := 0
	checkedAny := false

	for _, p := range providerCatalog {
		if len(only) > 0 && !contains(only, p.Bin) {
			continue
		}
		checkedAny = true

		path, _ := exec.LookPath(p.Bin)
		current := ""
		if path != "" {
			current = cliVersion(p.Bin)
		}

		if *checkOnly {
			latest := npmLatest(p.NpmPkg)
			label := p.Display
			fmt.Printf("\n%s\n", label)
			if path == "" {
				fmt.Printf("  current: (not installed)\n")
			} else {
				fmt.Printf("  current: %s\n", current)
			}
			fmt.Printf("  npm latest: %s\n", latest)
			continue
		}

		if path == "" {
			fmt.Printf("[skip] %s — not installed. Add it via the install wizard or:\n", p.Display)
			fmt.Printf("       npm install -g %s\n", p.NpmPkg)
			continue
		}

		fmt.Printf("\n=== %s — npm install -g %s ===\n", p.Display, p.NpmPkg)
		cmd := exec.Command("npm", "install", "-g", p.NpmPkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[!] %s update failed: %v\n", p.Display, err)
			rc = 1
			continue
		}
		after := cliVersion(p.Bin)
		switch after {
		case "":
			fmt.Printf("[✓] %s — upgraded (version probe failed; check `which %s`)\n", p.Display, p.Bin)
		case current:
			fmt.Printf("[✓] %s — already at npm-latest (%s)\n", p.Display, after)
		default:
			fmt.Printf("[✓] %s — %s → %s\n", p.Display, current, after)
		}
	}

	if !checkedAny {
		fmt.Fprintln(os.Stderr, "no providers matched --only — known names: claude, gemini, codex")
		return 2
	}
	return rc
}

// ── helpers ──────────────────────────────────────────────────────────

func cliVersion(bin string) string {
	cmd := exec.Command(bin, "--version")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return ""
}

// npmLatest shells out to `npm view <pkg> version`. Returns "" if npm is
// missing or the registry is unreachable — callers print "-" for empty.
func npmLatest(pkg string) string {
	if _, err := exec.LookPath("npm"); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "npm", "view", pkg, "version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func jsonOK(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "json encode failed: %v\n", err)
		return 1
	}
	return 0
}
