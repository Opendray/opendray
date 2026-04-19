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

// TestRunPluginCLI_ScaffoldStub verifies the scaffold stub returns 1 with a
// "not yet implemented" message on stderr, passing args through intact.
func TestRunPluginCLI_ScaffoldStub(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"scaffold", "--form", "declarative", "foo"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not yet implemented") {
		t.Errorf("expected stderr to contain \"not yet implemented\", got %q", stderr.String())
	}
}

// TestRunPluginCLI_InstallStub verifies the install stub returns 1 with a
// "not yet implemented" message on stderr.
func TestRunPluginCLI_InstallStub(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"install", "./x"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not yet implemented") {
		t.Errorf("expected stderr to contain \"not yet implemented\", got %q", stderr.String())
	}
}

// TestRunPluginCLI_ValidateStub_NoArgs verifies the validate stub returns 1 with a
// "not yet implemented" message on stderr when called with no arguments.
func TestRunPluginCLI_ValidateStub_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"validate"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not yet implemented") {
		t.Errorf("expected stderr to contain \"not yet implemented\", got %q", stderr.String())
	}
}

// TestRunPluginCLI_ValidateStub_WithArg verifies the validate stub returns 1 with a
// "not yet implemented" message on stderr when called with a path argument.
func TestRunPluginCLI_ValidateStub_WithArg(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runPluginCLIWith([]string{"validate", "./x"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not yet implemented") {
		t.Errorf("expected stderr to contain \"not yet implemented\", got %q", stderr.String())
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
