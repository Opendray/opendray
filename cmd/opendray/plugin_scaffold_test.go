package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/opendray/opendray/plugin"
)

// ─── TestParseScaffoldArgs ───────────────────────────────────────────────────

// TestParseScaffoldArgs_HappyPath verifies that the minimal required args
// produce a correctly-populated scaffoldOpts.
func TestParseScaffoldArgs_HappyPath(t *testing.T) {
	opts, help, err := parseScaffoldArgs([]string{"--form", "declarative", "my-plugin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if help {
		t.Fatal("help should be false")
	}
	if opts.Form != "declarative" {
		t.Errorf("Form: got %q, want %q", opts.Form, "declarative")
	}
	if opts.Name != "my-plugin" {
		t.Errorf("Name: got %q, want %q", opts.Name, "my-plugin")
	}
	if opts.Publisher != "opendray-you" {
		t.Errorf("Publisher: got %q, want %q (default)", opts.Publisher, "opendray-you")
	}
	if opts.OutDir != "." {
		t.Errorf("OutDir: got %q, want %q (default)", opts.OutDir, ".")
	}
}

// TestParseScaffoldArgs_WithPublisher verifies that --publisher sets the
// publisher field correctly.
func TestParseScaffoldArgs_WithPublisher(t *testing.T) {
	opts, _, err := parseScaffoldArgs([]string{"--form", "declarative", "--publisher", "acme", "my-plugin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Publisher != "acme" {
		t.Errorf("Publisher: got %q, want %q", opts.Publisher, "acme")
	}
}

// TestParseScaffoldArgs_WithOutDir verifies that --out-dir sets OutDir.
func TestParseScaffoldArgs_WithOutDir(t *testing.T) {
	opts, _, err := parseScaffoldArgs([]string{"--form", "declarative", "--out-dir", "/tmp/foo", "my-plugin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.OutDir != "/tmp/foo" {
		t.Errorf("OutDir: got %q, want %q", opts.OutDir, "/tmp/foo")
	}
}

// TestParseScaffoldArgs_MissingName verifies that omitting the positional name
// returns an error mentioning "name required".
func TestParseScaffoldArgs_MissingName(t *testing.T) {
	_, _, err := parseScaffoldArgs([]string{"--form", "declarative"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "name required") {
		t.Errorf("error should mention 'name required', got: %v", err)
	}
}

// TestParseScaffoldArgs_UnsupportedForm verifies that a form other than
// "declarative" returns ErrScaffoldUnsupportedForm.
func TestParseScaffoldArgs_UnsupportedForm(t *testing.T) {
	_, _, err := parseScaffoldArgs([]string{"--form", "webview", "foo"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrScaffoldUnsupportedForm {
		t.Errorf("expected ErrScaffoldUnsupportedForm, got: %v", err)
	}
}

// TestParseScaffoldArgs_InvalidName_Uppercase verifies that a name with
// uppercase letters is rejected.
func TestParseScaffoldArgs_InvalidName_Uppercase(t *testing.T) {
	_, _, err := parseScaffoldArgs([]string{"--form", "declarative", "Foo"})
	if err == nil {
		t.Fatal("expected error for uppercase name, got nil")
	}
	if !strings.Contains(err.Error(), "invalid plugin name") {
		t.Errorf("error should mention 'invalid plugin name', got: %v", err)
	}
}

// TestParseScaffoldArgs_InvalidName_LeadingHyphen verifies that a name
// starting with a hyphen is rejected.
func TestParseScaffoldArgs_InvalidName_LeadingHyphen(t *testing.T) {
	_, _, err := parseScaffoldArgs([]string{"--form", "declarative", "-foo"})
	if err == nil {
		t.Fatal("expected error for leading-hyphen name, got nil")
	}
}

// TestParseScaffoldArgs_InvalidName_Empty verifies that an empty string name
// is rejected (caught via missing-name path).
func TestParseScaffoldArgs_InvalidName_Empty(t *testing.T) {
	_, _, err := parseScaffoldArgs([]string{"--form", "declarative", ""})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

// TestParseScaffoldArgs_ValidName_SingleChar verifies that a single lowercase
// alphanumeric character is valid per the regex.
func TestParseScaffoldArgs_ValidName_SingleChar(t *testing.T) {
	opts, _, err := parseScaffoldArgs([]string{"--form", "declarative", "a"})
	if err != nil {
		t.Fatalf("single-char name 'a' should be valid, got: %v", err)
	}
	if opts.Name != "a" {
		t.Errorf("Name: got %q, want %q", opts.Name, "a")
	}
}

// TestParseScaffoldArgs_InvalidPublisher verifies that a publisher with invalid
// characters (e.g. "BadPub!") is rejected.
func TestParseScaffoldArgs_InvalidPublisher(t *testing.T) {
	_, _, err := parseScaffoldArgs([]string{"--form", "declarative", "--publisher", "BadPub!", "my-plugin"})
	if err == nil {
		t.Fatal("expected error for invalid publisher, got nil")
	}
	if !strings.Contains(err.Error(), "invalid publisher") {
		t.Errorf("error should mention 'invalid publisher', got: %v", err)
	}
}

// TestParseScaffoldArgs_Help_LongFlag verifies --help returns help=true and
// err=nil.
func TestParseScaffoldArgs_Help_LongFlag(t *testing.T) {
	_, help, err := parseScaffoldArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !help {
		t.Fatal("expected help=true")
	}
}

// TestParseScaffoldArgs_Help_ShortFlag verifies -h returns help=true.
func TestParseScaffoldArgs_Help_ShortFlag(t *testing.T) {
	_, help, err := parseScaffoldArgs([]string{"-h"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !help {
		t.Fatal("expected help=true")
	}
}

// ─── TestDeriveDisplayName ───────────────────────────────────────────────────

// TestDeriveDisplayName verifies title-casing and hyphen-to-space conversion.
func TestDeriveDisplayName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"my-plugin", "My Plugin"},
		{"foo", "Foo"},
		{"x-y-z", "X Y Z"},
		{"a", "A"},
		{"hello-world", "Hello World"},
		{"abc", "Abc"},
	}
	for _, tc := range cases {
		got := deriveDisplayName(tc.input)
		if got != tc.want {
			t.Errorf("deriveDisplayName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── TestWriteScaffold ───────────────────────────────────────────────────────

// TestWriteScaffold_CreatesWorkingPlugin verifies that writeScaffold produces a
// directory containing manifest.json and README.md, and that the manifest
// passes plugin.ValidateV1.
func TestWriteScaffold_CreatesWorkingPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	opts := scaffoldOpts{
		Form:      "declarative",
		Name:      "test-plug",
		Publisher: "opendray-you",
		OutDir:    tmpDir,
	}

	if err := writeScaffold(opts); err != nil {
		t.Fatalf("writeScaffold failed: %v", err)
	}

	plugDir := filepath.Join(tmpDir, "test-plug")

	// manifest.json must exist
	manifestPath := filepath.Join(plugDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest.json missing: %v", err)
	}

	// README.md must exist
	readmePath := filepath.Join(plugDir, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Fatalf("README.md missing: %v", err)
	}

	// Manifest must parse and pass ValidateV1
	p, err := plugin.LoadManifest(plugDir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}
	if errs := plugin.ValidateV1(p); len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("ValidateV1 error: %v", e)
		}
	}

	// Verify the plugin name matches what we requested
	if p.Name != "test-plug" {
		t.Errorf("Name: got %q, want %q", p.Name, "test-plug")
	}
	if p.Publisher != "opendray-you" {
		t.Errorf("Publisher: got %q, want %q", p.Publisher, "opendray-you")
	}
}

// TestWriteScaffold_OutputExists verifies that writeScaffold returns
// ErrScaffoldOutputExists when the target directory already exists, and that
// no partial state is left behind.
func TestWriteScaffold_OutputExists(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "my-plugin")

	// Pre-create the target directory
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("pre-create dir: %v", err)
	}

	opts := scaffoldOpts{
		Form:      "declarative",
		Name:      "my-plugin",
		Publisher: "opendray-you",
		OutDir:    tmpDir,
	}

	err := writeScaffold(opts)
	if err == nil {
		t.Fatal("expected ErrScaffoldOutputExists, got nil")
	}
	if err != ErrScaffoldOutputExists {
		t.Errorf("expected ErrScaffoldOutputExists, got: %v", err)
	}

	// No tmp dirs should remain in outDir (atomic cleanup)
	entries, readErr := os.ReadDir(tmpDir)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".scaffold-") {
			t.Errorf("partial temp dir not cleaned up: %s", e.Name())
		}
	}
}

// TestWriteScaffold_AtomicOnFailure verifies that when the final os.Rename
// fails (because the parent dir is read-only), no partial scaffold directory
// remains. Skipped on Windows where chmod semantics differ.
func TestWriteScaffold_AtomicOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod read-only semantics differ on Windows")
	}
	// Skip if running as root (root ignores read-only bits)
	if os.Getuid() == 0 {
		t.Skip("cannot test read-only directories as root")
	}

	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readonlyDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Make parent read-only so Rename into it fails
	if err := os.Chmod(readonlyDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		// Restore permissions so t.TempDir cleanup can delete it
		_ = os.Chmod(readonlyDir, 0o755)
	})

	opts := scaffoldOpts{
		Form:      "declarative",
		Name:      "my-plugin",
		Publisher: "opendray-you",
		OutDir:    readonlyDir,
	}

	err := writeScaffold(opts)
	if err == nil {
		// If it somehow succeeded (e.g. on a permissive FS), skip
		t.Log("writeScaffold succeeded unexpectedly; skipping atomicity assertion")
		return
	}

	// No partial scaffold dir should remain in readonlyDir after cleanup
	// We need read permission to check, which we have (0o555 = r-xr-xr-x)
	entries, _ := os.ReadDir(readonlyDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".scaffold-") {
			t.Errorf("partial temp dir not cleaned up: %s", e.Name())
		}
		if e.Name() == "my-plugin" {
			t.Errorf("partial plugin dir not cleaned up: %s", e.Name())
		}
	}
}

// ─── TestRunScaffold_Integration ────────────────────────────────────────────

// TestRunScaffold_Integration exercises the full runScaffold flow with
// --out-dir and verifies exit code 0, files on disk, and success line on
// stdout.
func TestRunScaffold_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := runScaffold(
		[]string{"--form", "declarative", "--out-dir", tmpDir, "my-plugin"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	// stdout must contain success line
	if !strings.Contains(stdout.String(), "scaffolded") {
		t.Errorf("stdout should contain 'scaffolded', got: %q", stdout.String())
	}

	// Files must be present
	plugDir := filepath.Join(tmpDir, "my-plugin")
	for _, name := range []string{"manifest.json", "README.md"} {
		if _, err := os.Stat(filepath.Join(plugDir, name)); err != nil {
			t.Errorf("expected file %q to exist: %v", name, err)
		}
	}

	// Manifest must pass validation
	p, err := plugin.LoadManifest(plugDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if errs := plugin.ValidateV1(p); len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("ValidateV1: %v", e)
		}
	}
}

// TestRunScaffold_NoArgs verifies that calling with no args exits 1 with
// usage text on stderr.
func TestRunScaffold_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runScaffold(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("expected Usage in stderr, got: %q", stderr.String())
	}
}

// TestRunScaffold_HelpFlag verifies --help exits 0.
func TestRunScaffold_HelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runScaffold([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", code)
	}
}

// TestRunScaffold_UnsupportedForm verifies non-declarative form exits 1.
func TestRunScaffold_UnsupportedForm(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runScaffold([]string{"--form", "webview", "my-plugin"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "declarative") {
		t.Errorf("stderr should mention 'declarative', got: %q", stderr.String())
	}
}

// TestRunScaffold_PrintsScaffoldHelp verifies that printScaffoldHelp writes
// meaningful content (Usage + flags).
func TestRunScaffold_PrintsScaffoldHelp(t *testing.T) {
	var buf bytes.Buffer
	printScaffoldHelp(&buf)
	out := buf.String()
	for _, want := range []string{"Usage", "--form", "--publisher", "--out-dir"} {
		if !strings.Contains(out, want) {
			t.Errorf("printScaffoldHelp output missing %q: %q", want, out)
		}
	}
}

// TestWriteScaffold_GeneratesCorrectCommandID verifies that the scaffolded
// manifest uses the plugin name as the command prefix (e.g. "myplug.hello").
func TestWriteScaffold_GeneratesCorrectCommandID(t *testing.T) {
	tmpDir := t.TempDir()
	opts := scaffoldOpts{
		Form:      "declarative",
		Name:      "myplug",
		Publisher: "test-pub",
		OutDir:    tmpDir,
	}

	if err := writeScaffold(opts); err != nil {
		t.Fatalf("writeScaffold: %v", err)
	}

	p, err := plugin.LoadManifest(filepath.Join(tmpDir, "myplug"))
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// Verify command ID is "myplug.hello"
	if p.Contributes == nil || len(p.Contributes.Commands) == 0 {
		t.Fatal("no commands in manifest")
	}
	if p.Contributes.Commands[0].ID != "myplug.hello" {
		t.Errorf("command ID: got %q, want %q", p.Contributes.Commands[0].ID, "myplug.hello")
	}

	// Verify statusBar uses the plugin name prefix
	if len(p.Contributes.StatusBar) == 0 {
		t.Fatal("no statusBar items in manifest")
	}
	if p.Contributes.StatusBar[0].ID != "myplug.bar" {
		t.Errorf("statusBar ID: got %q, want %q", p.Contributes.StatusBar[0].ID, "myplug.bar")
	}

	// Verify keybinding references the command
	if len(p.Contributes.Keybindings) == 0 {
		t.Fatal("no keybindings in manifest")
	}
	if p.Contributes.Keybindings[0].Command != "myplug.hello" {
		t.Errorf("keybinding command: got %q, want %q", p.Contributes.Keybindings[0].Command, "myplug.hello")
	}
}

// TestWriteScaffold_READMEContainsPluginName verifies the README.md contains
// the plugin name for the install instructions.
func TestWriteScaffold_READMEContainsPluginName(t *testing.T) {
	tmpDir := t.TempDir()
	opts := scaffoldOpts{
		Form:      "declarative",
		Name:      "readme-test",
		Publisher: "opendray-you",
		OutDir:    tmpDir,
	}

	if err := writeScaffold(opts); err != nil {
		t.Fatalf("writeScaffold: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "readme-test", "README.md"))
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}

	readme := string(data)
	if !strings.Contains(readme, "readme-test") {
		t.Errorf("README.md should contain plugin name 'readme-test', got:\n%s", readme)
	}
	if !strings.Contains(readme, "Readme Test") {
		t.Errorf("README.md should contain display name 'Readme Test', got:\n%s", readme)
	}
}

// TestRunScaffold_InvalidName_Error verifies that an invalid name produces
// a non-zero exit code with an error message.
func TestRunScaffold_InvalidName_Error(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runScaffold(
		[]string{"--form", "declarative", "--out-dir", tmpDir, "BADNAME"},
		&stdout, &stderr,
	)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid plugin name") {
		t.Errorf("stderr should contain 'invalid plugin name', got: %q", stderr.String())
	}
}

// TestRunScaffold_OutputAlreadyExists verifies that running scaffold twice on
// the same name returns exit 1 with ErrScaffoldOutputExists error message.
func TestRunScaffold_OutputAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	args := []string{"--form", "declarative", "--out-dir", tmpDir, "dupe-plugin"}

	// First run: should succeed
	var stdout1, stderr1 bytes.Buffer
	if code := runScaffold(args, &stdout1, &stderr1); code != 0 {
		t.Fatalf("first run failed: %d; %s", code, stderr1.String())
	}

	// Second run: should fail with output-exists error
	var stdout2, stderr2 bytes.Buffer
	code := runScaffold(args, &stdout2, &stderr2)
	if code != 1 {
		t.Errorf("expected exit code 1 on duplicate, got %d", code)
	}
	if !strings.Contains(stderr2.String(), "already exists") {
		t.Errorf("stderr should contain 'already exists', got: %q", stderr2.String())
	}
}
