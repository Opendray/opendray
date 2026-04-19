package main

import (
	"fmt"
	"io"

	"github.com/opendray/opendray/plugin"
)

// printValidateHelp writes usage to w.
func printValidateHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: opendray plugin validate [dir]

Validate the manifest.json in the given directory (default ".") against
the OpenDray plugin manifest v1 schema. Prints one line per violation.

Exit codes:
  0  valid
  1  invalid
  2  manifest unreadable
`)
}

// runValidate validates the manifest.json in the given directory and
// prints one line per violation in the form "<path>: <msg>". Returns
// exit code: 0 if valid, 1 if invalid, 2 if the manifest can't even
// be read/parsed.
//
// args[0] (optional) is the target dir (default ".").
// --help / -h prints usage, returns 0.
func runValidate(args []string, stdout, stderr io.Writer) int {
	// Resolve the target directory from the first positional argument.
	// Help flags are handled before the dir resolution.
	dir := "."
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			printValidateHelp(stdout)
			return 0
		default:
			dir = arg
		}
	}

	// Load the manifest. Failures here are exit code 2 (unreadable).
	p, err := plugin.LoadManifest(dir)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	// Legacy/compat manifests (IsV1()==false) are treated as valid.
	// ValidateV1 short-circuits for them, but we also produce a distinct
	// "ok (legacy/compat manifest)" message so callers can distinguish.
	if !p.IsV1() {
		fmt.Fprintln(stdout, "ok (legacy/compat manifest)")
		return 0
	}

	// Run the v1 validator.
	errs := plugin.ValidateV1(p)
	if len(errs) == 0 {
		fmt.Fprintln(stdout, "ok")
		return 0
	}

	// Print each violation as "<path>: <msg>".
	for _, ve := range errs {
		fmt.Fprintf(stderr, "%s: %s\n", ve.Path, ve.Msg)
	}
	fmt.Fprintf(stderr, "%d validation error(s)\n", len(errs))
	return 1
}
