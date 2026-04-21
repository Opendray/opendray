package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/opendray/opendray/cmd/opendray/templates"
)

// ─── Sentinel errors ─────────────────────────────────────────────────────────

// ErrScaffoldUnsupportedForm is returned when --form is set to a value other
// than "declarative". Only "declarative" is supported in M1.
var ErrScaffoldUnsupportedForm = errors.New("only --form declarative is supported in M1")

// ErrScaffoldOutputExists is returned when the target output directory
// already exists. scaffold is pure — it never overwrites.
var ErrScaffoldOutputExists = errors.New("scaffold output directory already exists")

// ─── Validation regexes ──────────────────────────────────────────────────────
// Patterns mirror docs/plugin-platform/02-manifest.md §JSON Schema verbatim.

// reScaffoldName — 02-manifest.md line 18.
var reScaffoldName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// reScaffoldPublisher — 02-manifest.md line 20.
var reScaffoldPublisher = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$`)

// ─── scaffoldOpts ────────────────────────────────────────────────────────────

// scaffoldOpts holds all validated inputs for the scaffold command.
type scaffoldOpts struct {
	Form      string // only "declarative" supported in M1
	Name      string // required positional — must match plugin-name regex
	Publisher string // --publisher flag, defaults to "opendray-you"
	OutDir    string // --out-dir flag, defaults to "."
}

// ─── templateData ────────────────────────────────────────────────────────────

// templateData is the data context passed to each template file.
type templateData struct {
	Name        string
	Publisher   string
	DisplayName string
}

// ─── parseScaffoldArgs ───────────────────────────────────────────────────────

// parseScaffoldArgs parses flags and the positional plugin-name argument.
// Returns the populated opts, a helpRequested flag, and any validation error.
// When helpRequested is true, the caller should print help and exit 0.
func parseScaffoldArgs(args []string) (scaffoldOpts, bool, error) {
	// Intercept -h / --help before flag.Parse so we can exit 0 cleanly.
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return scaffoldOpts{}, true, nil
		}
	}

	fs := flag.NewFlagSet("opendray plugin scaffold", flag.ContinueOnError)
	// Suppress the default error output — we handle it ourselves.
	fs.SetOutput(io.Discard)

	form := fs.String("form", "", "Plugin form (only \"declarative\" is supported in M1)")
	publisher := fs.String("publisher", "opendray-you", "Publisher id (matches ^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$)")
	outDir := fs.String("out-dir", ".", "Output directory (plugin dir is created inside this dir)")

	if err := fs.Parse(args); err != nil {
		return scaffoldOpts{}, false, fmt.Errorf("scaffold: %w", err)
	}

	// Validate --form
	if *form != "declarative" {
		return scaffoldOpts{}, false, ErrScaffoldUnsupportedForm
	}

	// Remaining positional args: the plugin name is the first (and only) one.
	positional := fs.Args()
	if len(positional) == 0 || positional[0] == "" {
		return scaffoldOpts{}, false, fmt.Errorf("name required: provide the plugin name as a positional argument")
	}
	name := positional[0]

	// Validate name against the manifest regex.
	if !reScaffoldName.MatchString(name) {
		return scaffoldOpts{}, false, fmt.Errorf(
			"invalid plugin name: must match ^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$, got %q", name,
		)
	}

	// Validate publisher.
	if !reScaffoldPublisher.MatchString(*publisher) {
		return scaffoldOpts{}, false, fmt.Errorf(
			"invalid publisher: must match ^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$, got %q", *publisher,
		)
	}

	return scaffoldOpts{
		Form:      *form,
		Name:      name,
		Publisher: *publisher,
		OutDir:    *outDir,
	}, false, nil
}

// ─── deriveDisplayName ───────────────────────────────────────────────────────

// deriveDisplayName converts a kebab-case plugin name to a title-cased display
// name by replacing hyphens with spaces and capitalising each word.
// Examples: "my-plugin" → "My Plugin", "foo" → "Foo", "x-y-z" → "X Y Z".
func deriveDisplayName(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// ─── printScaffoldHelp ───────────────────────────────────────────────────────

// printScaffoldHelp writes the scaffold subcommand usage to w.
// Kept separate so tests can verify the help text without running the full command.
func printScaffoldHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: opendray plugin scaffold --form declarative [flags] <name>

Create a new OpenDray plugin skeleton in the current directory (or --out-dir).

Arguments:
  <name>            Plugin name. Must match ^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$

Flags:
  --form declarative   Plugin form (only "declarative" is supported in M1) [required]
  --publisher <id>     Publisher id (default: "opendray-you")
                       Must match ^[a-z0-9]([a-z0-9-]{0,38}[a-z0-9])?$
  --out-dir <dir>      Output directory; the plugin dir is created inside (default: ".")
  --help, -h           Print this help text and exit 0

Examples:
  opendray plugin scaffold --form declarative my-plugin
  opendray plugin scaffold --form declarative --publisher acme my-plugin --out-dir /tmp
`)
}

// ─── writeScaffold ───────────────────────────────────────────────────────────

// writeScaffold renders the declarative scaffold templates into a new directory
// at opts.OutDir/opts.Name. The write is atomic: templates are rendered into a
// temporary directory first, then renamed into place in a single os.Rename
// call. If anything fails, the temporary directory is removed so no partial
// state is left on disk.
//
// Returns ErrScaffoldOutputExists if opts.OutDir/opts.Name already exists.
func writeScaffold(opts scaffoldOpts) error {
	finalDir := filepath.Join(opts.OutDir, opts.Name)

	// Guard: fail immediately if the target already exists.
	if _, err := os.Stat(finalDir); err == nil {
		return ErrScaffoldOutputExists
	}

	data := templateData{
		Name:        opts.Name,
		Publisher:   opts.Publisher,
		DisplayName: deriveDisplayName(opts.Name),
	}

	// Stage into a hidden temp dir inside outDir so that os.Rename is
	// always on the same filesystem (no cross-device rename).
	tmpDir, err := os.MkdirTemp(opts.OutDir, ".scaffold-*")
	if err != nil {
		return fmt.Errorf("scaffold: create temp dir: %w", err)
	}

	// Cleanup on any error path after this point.
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	// Walk the embedded declarative FS and render each template into tmpDir.
	embedFS := templates.DeclarativeFS
	err = fs.WalkDir(embedFS, "declarative", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil // nothing to create; tmpDir is flat for M1
		}

		// Read template source.
		src, readErr := embedFS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("scaffold: read template %s: %w", path, readErr)
		}

		// Parse with Go text/template.
		tmpl, parseErr := template.New(path).Parse(string(src))
		if parseErr != nil {
			return fmt.Errorf("scaffold: parse template %s: %w", path, parseErr)
		}

		// Derive the output filename: strip the "declarative/" prefix and the
		// ".tmpl" suffix to get the real filename (e.g. manifest.json.tmpl →
		// manifest.json).
		rel := strings.TrimPrefix(path, "declarative/")
		outName := strings.TrimSuffix(rel, ".tmpl")
		outPath := filepath.Join(tmpDir, outName)

		// Create output file.
		f, createErr := os.Create(outPath)
		if createErr != nil {
			return fmt.Errorf("scaffold: create %s: %w", outPath, createErr)
		}

		execErr := tmpl.Execute(f, data)
		closeErr := f.Close()
		if execErr != nil {
			return fmt.Errorf("scaffold: render template %s: %w", path, execErr)
		}
		if closeErr != nil {
			return fmt.Errorf("scaffold: close %s: %w", outPath, closeErr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Atomic rename: move the temp dir to the final location.
	if err := os.Rename(tmpDir, finalDir); err != nil {
		return fmt.Errorf("scaffold: rename %s → %s: %w", tmpDir, finalDir, err)
	}
	ok = true
	return nil
}

// ─── runScaffold ─────────────────────────────────────────────────────────────

// runScaffold is the top-level entry point for `opendray plugin scaffold`.
// It parses args, validates all inputs, and writes the scaffold. Returns a
// Unix exit code (0 = success, 1 = error).
func runScaffold(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: name required")
		fmt.Fprintln(stderr, "")
		printScaffoldHelp(stderr)
		return 1
	}

	opts, helpRequested, err := parseScaffoldArgs(args)
	if helpRequested {
		printScaffoldHelp(stdout)
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if err := writeScaffold(opts); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	finalDir := filepath.Join(opts.OutDir, opts.Name)
	absDir, absErr := filepath.Abs(finalDir)
	if absErr != nil {
		absDir = finalDir
	}
	fmt.Fprintf(stdout, "scaffolded %q at %s\n", opts.Name, absDir)
	return 0
}
