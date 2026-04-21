package main

import (
	"fmt"
	"io"
	"os"
)

// runPluginCLI is the top-level dispatcher for `opendray plugin ...`.
// It delegates to runPluginCLIWith using the real os.Stdout / os.Stderr so
// that main.go can call it without caring about writer injection.
//
// Exit codes follow Unix conventions:
//
//	0 — success
//	1 — user error (bad args, unknown subcommand, missing required flag)
//	   M1 stubs also return 1 ("not yet implemented") for unfinished commands.
func runPluginCLI(args []string) int {
	return runPluginCLIWith(args, os.Stdout, os.Stderr)
}

// runPluginCLIWith is the injectable form used directly by tests.
// args is os.Args[2:] (everything after the "plugin" subcommand token).
func runPluginCLIWith(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printPluginUsage(stderr)
		return 1
	}

	switch args[0] {
	case "--help", "-h":
		printPluginUsage(stdout)
		return 0
	case "scaffold":
		return pluginCmdScaffoldWith(args[1:], stdout, stderr)
	case "install":
		return pluginCmdInstallWith(args[1:], stdout, stderr)
	case "validate":
		return pluginCmdValidateWith(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown plugin subcommand: %s\n\n", args[0])
		printPluginUsage(stderr)
		return 1
	}
}

// printPluginUsage writes a concise usage block to the given writer.
// Kept separate from the dispatcher so tests can capture it deterministically.
func printPluginUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: opendray plugin <command> [args]

Commands:
  scaffold --form <declarative>  <name>   Create a new plugin skeleton
  install  <path-or-url>                  Install a plugin from a local path
  validate [dir]                          Validate a plugin manifest

Use "opendray plugin <command> --help" for per-command help.
`)
}

// pluginCmdScaffold is the stub for T14.
// M1 returns 1 with a "not implemented yet" message so the flow is wired
// but inert. It will be replaced in T14 with real scaffold logic.
func pluginCmdScaffold(args []string) int {
	return pluginCmdScaffoldWith(args, os.Stdout, os.Stderr)
}

func pluginCmdScaffoldWith(args []string, stdout io.Writer, stderr io.Writer) int {
	return runScaffold(args, stdout, stderr)
}

// pluginCmdInstall is the stub for T15.
// M1 returns 1 with a "not implemented yet" message. It will be replaced
// in T15 with real HTTP-client install logic.
func pluginCmdInstall(args []string) int {
	return pluginCmdInstallWith(args, os.Stdout, os.Stderr)
}

func pluginCmdInstallWith(args []string, stdout, stderr io.Writer) int {
	return runInstall(args, stdout, stderr)
}

// pluginCmdValidate is the stub for T16.
// M1 returns 1 with a "not implemented yet" message. It will be replaced
// in T16 with ValidateV1-backed manifest validation.
func pluginCmdValidate(args []string) int {
	return pluginCmdValidateWith(args, os.Stdout, os.Stderr)
}

func pluginCmdValidateWith(args []string, stdout, stderr io.Writer) int {
	return runValidate(args, stdout, stderr)
}
