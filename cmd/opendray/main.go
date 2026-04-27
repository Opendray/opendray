// Command opendray is the gateway binary entry point.
//
// Subcommands:
//
//	opendray serve [-config FILE]   start the gateway
//	opendray migrate [-config FILE] apply pending DB migrations and exit
//	opendray version                print build info and exit
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
		os.Exit(run(args, func(ctx context.Context, a *app.App) error {
			defer a.Close()
			return a.Migrate(ctx)
		}))
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
  opendray serve   [-config FILE]
  opendray migrate [-config FILE]
  opendray version`)
}
