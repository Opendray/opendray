package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validV1ManifestJSON is the time-ninja manifest verbatim, used as a
// known-valid v1 fixture. Content is from plugins/examples/time-ninja/manifest.json.
const validV1ManifestJSON = `{
  "$schema": "https://opendray.dev/schemas/plugin-manifest-v1.json",
  "name": "time-ninja",
  "version": "1.0.0",
  "publisher": "opendray-examples",
  "displayName": "Time Ninja",
  "description": "Pomodoro reminder that lives in the status bar. The M1 reference plugin.",
  "icon": "🍅",
  "engines": { "opendray": "^1.0.0" },
  "form": "declarative",
  "activation": ["onStartup"],
  "contributes": {
    "commands": [
      {
        "id": "time.start",
        "title": "Start Pomodoro",
        "category": "Time Ninja",
        "run": { "kind": "notify", "message": "Pomodoro started — 25 minutes" }
      }
    ],
    "statusBar": [
      {
        "id": "time.bar",
        "text": "🍅 25:00",
        "tooltip": "Start a pomodoro",
        "command": "time.start",
        "alignment": "right",
        "priority": 50
      }
    ],
    "keybindings": [
      { "command": "time.start", "key": "ctrl+alt+p", "mac": "cmd+alt+p" }
    ],
    "menus": {
      "appBar/right": [
        { "command": "time.start", "group": "timer@1" }
      ]
    }
  },
  "permissions": {}
}`

// legacyManifestJSON is a bare legacy manifest — IsV1()==false because it has
// no publisher or engines.opendray. ValidateV1 short-circuits and returns nil.
const legacyManifestJSON = `{"name":"x","type":"cli","version":"1.0.0"}`

// writeManifest writes the given JSON content to manifest.json inside dir.
func writeManifest(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}
}

// repoRoot walks up from the test's source directory to find the repo root
// (identified by the presence of go.mod). Used to locate the real
// plugins/examples/time-ninja/ fixture.
func repoRoot(t *testing.T) string {
	t.Helper()
	// Start from the test binary working directory.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("repoRoot: getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repoRoot: could not find go.mod by walking up from " + dir)
		}
		dir = parent
	}
}

// ─── Test cases ─────────────────────────────────────────────────────────────

// TestRunValidate_ValidV1Manifest verifies that a known-good v1 manifest
// prints "ok" to stdout, leaves stderr empty, and returns exit code 0.
func TestRunValidate_ValidV1Manifest(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, validV1ManifestJSON)

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{dir}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Errorf("expected stdout to contain \"ok\", got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr, got %q", stderr.String())
	}
}

// TestRunValidate_LegacyManifest verifies that a legacy (non-v1) manifest
// prints "ok" (compat short-circuit) to stdout and returns exit code 0.
func TestRunValidate_LegacyManifest(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, legacyManifestJSON)

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{dir}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0 for legacy manifest, got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Errorf("expected stdout to contain \"ok\", got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr for legacy manifest, got %q", stderr.String())
	}
}

// TestRunValidate_InvalidV1_BadCommandID verifies that a v1 manifest with an
// invalid command id (contains uppercase) produces an error on stderr that
// names the path "contributes.commands[0].id:", exits 1, and the final line
// contains "1 validation error".
func TestRunValidate_InvalidV1_BadCommandID(t *testing.T) {
	const badIDManifest = `{
  "name": "time-ninja",
  "version": "1.0.0",
  "publisher": "opendray-examples",
  "engines": { "opendray": "^1.0.0" },
  "contributes": {
    "commands": [
      {
        "id": "Bad-ID",
        "title": "Bad Command",
        "run": { "kind": "notify", "message": "test" }
      }
    ]
  }
}`
	dir := t.TempDir()
	writeManifest(t, dir, badIDManifest)

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{dir}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "contributes.commands[0].id:") {
		t.Errorf("expected stderr to contain \"contributes.commands[0].id:\", got %q", errOut)
	}
	lines := strings.Split(strings.TrimSpace(errOut), "\n")
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "1 validation error") {
		t.Errorf("expected final stderr line to contain \"1 validation error\", got %q", lastLine)
	}
}

// TestRunValidate_InvalidV1_MissingPublisher verifies that a manifest that is
// v1 (has publisher + engines.opendray) but has an invalid name (uppercase)
// produces a validation error on the "name" path and exits 1.
func TestRunValidate_InvalidV1_MissingPublisher(t *testing.T) {
	// Name "Bad" contains uppercase — fails the name pattern even though
	// publisher and engines.opendray are both present (so IsV1()==true).
	const badNameManifest = `{
  "name": "Bad",
  "version": "1.0.0",
  "publisher": "ok",
  "engines": { "opendray": "^1.0.0" }
}`
	dir := t.TempDir()
	writeManifest(t, dir, badNameManifest)

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{dir}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "name:") {
		t.Errorf("expected stderr to contain \"name:\", got %q", errOut)
	}
}

// TestRunValidate_MultipleErrors verifies that a v1 manifest with two distinct
// violations emits both paths on stderr and the final line contains
// "2 validation error(s)".
func TestRunValidate_MultipleErrors(t *testing.T) {
	// Bad name AND bad command id → 2 violations.
	const twoErrorManifest = `{
  "name": "Bad",
  "version": "1.0.0",
  "publisher": "ok",
  "engines": { "opendray": "^1.0.0" },
  "contributes": {
    "commands": [
      {
        "id": "Bad-ID",
        "title": "Bad Command",
        "run": { "kind": "notify", "message": "test" }
      }
    ]
  }
}`
	dir := t.TempDir()
	writeManifest(t, dir, twoErrorManifest)

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{dir}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "name:") {
		t.Errorf("expected stderr to contain \"name:\", got %q", errOut)
	}
	if !strings.Contains(errOut, "contributes.commands[0].id:") {
		t.Errorf("expected stderr to contain \"contributes.commands[0].id:\", got %q", errOut)
	}
	lines := strings.Split(strings.TrimSpace(errOut), "\n")
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "2 validation error") {
		t.Errorf("expected final stderr line to contain \"2 validation error\", got %q", lastLine)
	}
}

// TestRunValidate_NoManifestFile verifies that pointing to an empty directory
// (no manifest.json) exits 2 and stderr contains "error: ".
func TestRunValidate_NoManifestFile(t *testing.T) {
	dir := t.TempDir() // empty — no manifest.json

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{dir}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("expected exit code 2, got %d; stderr=%q", code, stderr.String())
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "error: ") {
		t.Errorf("expected stderr to contain \"error: \", got %q", errOut)
	}
}

// TestRunValidate_MalformedJSON verifies that a manifest.json with invalid
// JSON exits 2 and stderr mentions JSON.
func TestRunValidate_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "{not json")

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{dir}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("expected exit code 2, got %d; stderr=%q", code, stderr.String())
	}
	errOut := stderr.String()
	// Must mention JSON in some form — the loader error wraps the json parse error.
	if !strings.Contains(strings.ToLower(errOut), "json") &&
		!strings.Contains(errOut, "parse") &&
		!strings.Contains(errOut, "invalid") {
		t.Errorf("expected stderr to hint at JSON parse failure, got %q", errOut)
	}
}

// TestRunValidate_NoArgsUsesCwd verifies that when no arguments are passed,
// runValidate validates manifest.json in the current working directory.
// Uses t.Chdir (Go 1.24+) to set CWD to a temp dir with a valid manifest.
func TestRunValidate_NoArgsUsesCwd(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, validV1ManifestJSON)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0 (valid manifest in CWD), got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Errorf("expected stdout to contain \"ok\", got %q", stdout.String())
	}
}

// TestRunValidate_DirArgument verifies that the positional dir argument is
// used regardless of the current working directory.
func TestRunValidate_DirArgument(t *testing.T) {
	// Create a separate dir for the manifest and a different CWD.
	manifestDir := t.TempDir()
	writeManifest(t, manifestDir, validV1ManifestJSON)

	cwdDir := t.TempDir() // no manifest here
	t.Chdir(cwdDir)

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{manifestDir}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0 (dir arg overrides CWD), got %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Errorf("expected stdout to contain \"ok\", got %q", stdout.String())
	}
}

// TestRunValidate_HelpFlag verifies that --help and -h print usage to stdout
// and return exit code 0.
func TestRunValidate_HelpFlag(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runValidate([]string{flag}, &stdout, &stderr)

			if code != 0 {
				t.Errorf("expected exit code 0 for %s, got %d", flag, code)
			}
			if !strings.Contains(stdout.String(), "Usage:") {
				t.Errorf("expected stdout to contain \"Usage:\" for %s, got %q", flag, stdout.String())
			}
			if stderr.Len() != 0 {
				t.Errorf("expected empty stderr for %s, got %q", flag, stderr.String())
			}
		})
	}
}

// TestRunValidate_PassesTimeNinja validates the actual repo fixture at
// plugins/examples/time-ninja/ and asserts it is spec-compliant (exit 0).
func TestRunValidate_PassesTimeNinja(t *testing.T) {
	root := repoRoot(t)
	timeNinjaDir := filepath.Join(root, "plugins", "examples", "time-ninja")

	if _, err := os.Stat(filepath.Join(timeNinjaDir, "manifest.json")); err != nil {
		t.Skipf("time-ninja fixture not found at %s: %v", timeNinjaDir, err)
	}

	var stdout, stderr bytes.Buffer
	code := runValidate([]string{timeNinjaDir}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("expected exit code 0 for time-ninja fixture, got %d; stderr=%q stderr", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Errorf("expected stdout to contain \"ok\" for time-ninja, got %q", stdout.String())
	}
}

// TestPrintValidateHelp verifies that the usage text contains "Exit codes:".
func TestPrintValidateHelp(t *testing.T) {
	var buf bytes.Buffer
	printValidateHelp(&buf)

	if !strings.Contains(buf.String(), "Exit codes:") {
		t.Errorf("expected usage text to contain \"Exit codes:\", got %q", buf.String())
	}
}
