// Command opendray is the gateway binary entry point.
//
// Subcommands:
//
//	opendray serve     [-config FILE]   start the gateway (foreground; for systemd/launchd ExecStart)
//	opendray migrate   [-config FILE]   apply pending DB migrations and exit
//	opendray start                      start the systemd/launchd service
//	opendray stop                       stop the systemd/launchd service
//	opendray restart                    restart the systemd/launchd service
//	opendray status                     show service status
//	opendray update    [--check] [--force] [--yes] [--restart]
//	                                    check + apply the latest released opendray binary
//	opendray providers <subcommand> ... list / update the AI CLIs opendray spawns
//	opendray notes     <subcommand> ... operate on the file-system notes vault (no gateway needed)
//	opendray skill     <subcommand> ... inspect / load agent skills (no gateway needed)
//	opendray mcp       <subcommand> ... inspect MCP server registry (no gateway needed)
//	opendray mcp-memory                 stdio MCP server bridging agents to opendray memory (run by Claude/Codex/etc.)
//	opendray version                    print build info and exit
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/opendray/opendray-v2/internal/app"
	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	switch cmd {
	case "serve":
		os.Exit(run(args, func(ctx context.Context, a *app.App) error { return a.Run(ctx) }))
	case "migrate":
		// Deliberately not routed through `run` — migrate must
		// bypass internal/app.New to work on a fresh database
		// where the catalog seed step would otherwise fail
		// against tables that don't exist yet (see #162).
		os.Exit(runMigrate(args))
	case "update":
		os.Exit(runUpdate(args))
	case "self-update":
		os.Exit(runSelfUpdate(args))
	case "providers":
		os.Exit(runProviders(args))
	case "start":
		os.Exit(runStart(args))
	case "stop":
		os.Exit(runStop(args))
	case "restart":
		os.Exit(runRestart(args))
	case "status":
		os.Exit(runStatus(args))
	case "notes":
		os.Exit(runNotes(args))
	case "skill":
		os.Exit(runSkill(args))
	case "mcp":
		os.Exit(runMcp(args))
	case "mcp-memory":
		os.Exit(runMcpMemory(args))
	case "hook":
		os.Exit(runHook(args))
	case "version":
		fmt.Printf("opendray %s (%s, %s)\n", version.Version, version.Commit, version.Date)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func run(args []string, fn func(context.Context, *app.App) error) int {
	fs := flag.NewFlagSet("opendray", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to config.toml (env-only mode if empty)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	a, err := app.New(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := fn(ctx, a); err != nil {
		a.Logger().Error("fatal", "err", err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, `opendray — multiplexer + integration gateway for AI agent CLIs

usage:
  opendray serve     [-config FILE]       (foreground; what the service unit's ExecStart calls)
  opendray migrate   [-config FILE]
  opendray start                          (start the systemd / launchd service)
  opendray stop                           (stop the service)
  opendray restart                        (stop + start)
  opendray status                         (show service status)
  opendray update    [--check] [--force] [--yes] [--restart]
                                          (download + replace this binary with the latest release;
                                           "opendray update --check" for a no-op version probe)
  opendray providers <subcommand> [args]  (run "opendray providers --help" — list / update AI CLIs)
  opendray notes     <subcommand> [args]  (run "opendray notes --help" for details)
  opendray skill     <subcommand> [args]  (run "opendray skill --help" for details)
  opendray mcp       <subcommand> [args]  (run "opendray mcp --help" for details)
  opendray mcp-memory                     (stdio MCP server — invoked by an agent CLI, not by humans)
  opendray hook      <event>              (Claude Code hook entry — auto-write journal entries)
  opendray version`)
}
