package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeShell writes an executable stub at dir/name and returns its path.
// Content is deliberately empty + chmod 0755 — detectLoginShell only
// verifies PATH resolution, never actually runs the program.
func fakeShell(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("fakeShell: write %s: %v", p, err)
	}
	return p
}

func TestDetectLoginShell_UsesSHELLEnvWhenValid(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shells — not applicable on windows")
	}
	dir := t.TempDir()
	myShell := fakeShell(t, dir, "myshell")

	t.Setenv("SHELL", myShell)
	// Strip PATH so fallback candidates can't mask a $SHELL bug.
	t.Setenv("PATH", "/definitely/does/not/exist")

	got, err := DetectLoginShell()
	if err != nil {
		t.Fatalf("DetectLoginShell: %v", err)
	}
	if got != myShell {
		t.Errorf("got %q, want %q", got, myShell)
	}
}

func TestDetectLoginShell_IgnoresInvalidSHELL(t *testing.T) {
	dir := t.TempDir()
	bash := fakeShell(t, dir, "bash")

	// $SHELL points nowhere → should fall through to PATH lookup.
	t.Setenv("SHELL", "/no/such/shell")
	t.Setenv("PATH", dir)

	got, err := DetectLoginShell()
	if err != nil {
		t.Fatalf("DetectLoginShell: %v", err)
	}
	if got != bash {
		t.Errorf("got %q, want %q (bash on synthetic PATH)", got, bash)
	}
}

func TestDetectLoginShell_EmptySHELLFallsBackToPATH(t *testing.T) {
	dir := t.TempDir()
	zsh := fakeShell(t, dir, "zsh")

	t.Setenv("SHELL", "")
	t.Setenv("PATH", dir)

	got, err := DetectLoginShell()
	if err != nil {
		t.Fatalf("DetectLoginShell: %v", err)
	}
	if got != zsh {
		t.Errorf("got %q, want %q", got, zsh)
	}
}

func TestDetectLoginShell_CandidateOrderZshBeforeBash(t *testing.T) {
	dir := t.TempDir()
	zsh := fakeShell(t, dir, "zsh")
	_ = fakeShell(t, dir, "bash")

	t.Setenv("SHELL", "")
	t.Setenv("PATH", dir)

	got, err := DetectLoginShell()
	if err != nil {
		t.Fatalf("DetectLoginShell: %v", err)
	}
	if got != zsh {
		t.Errorf("expected zsh ahead of bash; got %q", got)
	}
}

func TestDetectLoginShell_FallsThroughToBashThenSh(t *testing.T) {
	dir := t.TempDir()
	sh := fakeShell(t, dir, "sh")

	t.Setenv("SHELL", "")
	t.Setenv("PATH", dir)

	got, err := DetectLoginShell()
	if err != nil {
		t.Fatalf("DetectLoginShell: %v", err)
	}
	if got != sh {
		t.Errorf("expected sh as last-resort; got %q", got)
	}
}

func TestDetectLoginShell_NoCandidateReturnsError(t *testing.T) {
	// Empty $SHELL + PATH without zsh/bash/sh → hard failure.
	t.Setenv("SHELL", "")
	t.Setenv("PATH", t.TempDir())

	got, err := DetectLoginShell()
	if err == nil {
		t.Fatalf("expected error, got %q", got)
	}
	if !strings.Contains(err.Error(), "no shell") {
		t.Errorf("error should mention missing shell; got %v", err)
	}
}

func TestIsAutoCommand(t *testing.T) {
	for _, s := range []string{"", " ", "auto", "AUTO", "Auto", "  auto  "} {
		if !isAutoCommand(s) {
			t.Errorf("isAutoCommand(%q) should be true", s)
		}
	}
	for _, s := range []string{"/bin/zsh", "zsh", "bash", "/usr/local/bin/fish"} {
		if isAutoCommand(s) {
			t.Errorf("isAutoCommand(%q) should be false", s)
		}
	}
}
