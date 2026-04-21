package main

import (
	"bytes"
	"strings"
	"testing"
)

// runPluginCLIWith is the testable entry point that accepts injectable writers.
// Tests call this directly instead of runPluginCLI so no subprocess is needed.

// TestRunPluginCLI_NoArgs verifies that calling with no arguments prints usage
// to stderr and returns exit code 1.
func TestRunPluginCLI_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith(nil, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("expected stderr to contain \"Usage:\", got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "scaffold") {
		t.Errorf("expected stderr to contain \"scaffold\", got %q", stderr.String())
	}
}

// TestRunPluginCLI_Help_LongFlag verifies --help prints usage to stdout and returns 0.
func TestRunPluginCLI_Help_LongFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected stdout to contain \"Usage:\", got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr, got %q", stderr.String())
	}
}

// TestRunPluginCLI_Help_ShortFlag verifies -h prints usage to stdout and returns 0.
func TestRunPluginCLI_Help_ShortFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"-h"}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected stdout to contain \"Usage:\", got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr, got %q", stderr.String())
	}
}

// TestRunPluginCLI_UnknownSubcommand verifies that an unknown subcommand prints
// an error message to stderr and returns exit code 1.
func TestRunPluginCLI_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"doesnotexist"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown plugin subcommand: doesnotexist") {
		t.Errorf("expected stderr to contain unknown subcommand message, got %q", stderr.String())
	}
}

// TestRunPluginCLI_ScaffoldDispatch verifies that the scaffold subcommand is
// wired correctly via runPluginCLIWith and succeeds (exit 0) when given valid
// args pointing at a fresh temp directory. T14 replaces the old stub test.
func TestRunPluginCLI_ScaffoldDispatch(t *testing.T) {
	tmpDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith(
		[]string{"scaffold", "--form", "declarative", "--out-dir", tmpDir, "my-wired-plugin"},
		&stdout, &stderr,
	)

	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "scaffolded") {
		t.Errorf("expected stdout to contain \"scaffolded\", got %q", stdout.String())
	}
}

// TestRunPluginCLI_InstallDispatch verifies the install subcommand is wired
// and runs real logic (T15). With no server available, it exits non-zero.
// We only assert the command is dispatched (not the old "not yet implemented" stub).
func TestRunPluginCLI_InstallDispatch(t *testing.T) {
	// Point at a non-listening port so the install attempt fails fast.
	t.Setenv("OPENDRAY_SERVER_URL", "http://127.0.0.1:0")

	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"install", "--yes", "./x"}, &stdout, &stderr)

	// Non-zero: install tried to reach the server and failed (exit 2) or
	// the server rejected the request (exit 1). Either is fine — we only
	// verify the stub is gone and real logic ran (no "not yet implemented").
	if code == 0 {
		t.Errorf("expected non-zero exit code when server is unavailable, got 0")
	}
	if strings.Contains(stderr.String(), "not yet implemented") {
		t.Errorf("stub message still present — T15 implementation not wired: %q", stderr.String())
	}
}

// TestRunPluginCLI_ValidateNoArgs verifies that "opendray plugin validate" with
// no args and no manifest.json in CWD returns exit code 2 (unreadable manifest)
// now that T16 is implemented (the "not yet implemented" stub is gone).
func TestRunPluginCLI_ValidateNoArgs(t *testing.T) {
	// Use a CWD with no manifest.json so we get exit 2.
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := runPluginCLIWith([]string{"validate"}, &stdout, &stderr)

	// exit 2 = manifest unreadable (no manifest.json in empty temp dir)
	if code != 2 {
		t.Errorf("expected exit code 2 (no manifest), got %d; stderr=%q", code, stderr.String())
	}
}

// TestRunPluginCLI_ValidateWithNonExistentDir verifies that pointing validate
// at a non-existent path returns exit code 2 (unreadable manifest).
func TestRunPluginCLI_ValidateWithNonExistentDir(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"validate", "/nonexistent/path/abc123"}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("expected exit code 2 (no manifest), got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "error: ") {
		t.Errorf("expected stderr to contain \"error: \", got %q", stderr.String())
	}
}

// TestPrintPluginUsage_ContainsAllSubcommands verifies that the usage text
// mentions all three subcommand names.
func TestPrintPluginUsage_ContainsAllSubcommands(t *testing.T) {
	var buf bytes.Buffer

	printPluginUsage(&buf)

	usage := buf.String()
	for _, name := range []string{"scaffold", "install", "validate"} {
		if !strings.Contains(usage, name) {
			t.Errorf("expected usage to contain subcommand %q, got %q", name, usage)
		}
	}
}
