package plugin

// Shell auto-detection for the built-in Terminal plugin.
//
// Previously the Terminal manifest hard-coded /bin/zsh. Linux minimal
// images (Alpine, Debian slim, LXC minrootfs) don't ship zsh, and some
// distributions install it at /usr/bin/zsh rather than /bin/zsh, so a
// fresh user would click Terminal and see "file not found" from the
// PTY spawn path. DetectLoginShell probes the user's real shell so the
// plugin works out of the box.

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// shellCandidates is the POSIX fallback list consulted when $SHELL is
// unset / unusable. Order matters: we prefer zsh (macOS default, many
// developer Linux setups), then bash (widely installed baseline), then
// sh (always present on any POSIX system).
//
// Names are looked up through exec.LookPath so /usr/bin/zsh,
// /usr/local/bin/zsh (Homebrew on Apple Silicon via /opt/homebrew/bin),
// and container-specific layouts all resolve correctly.
var shellCandidates = []string{"zsh", "bash", "sh"}

// DetectLoginShell returns an absolute path to a runnable login shell.
//
// Resolution order:
//  1. $SHELL env var, if it resolves to an executable on disk.
//  2. exec.LookPath("zsh") → ("bash") → ("sh").
//  3. Error "no shell available on PATH" — the caller should surface
//     this so the user can configure a specific shell in Providers.
func DetectLoginShell() (string, error) {
	if envShell := strings.TrimSpace(os.Getenv("SHELL")); envShell != "" {
		if info, err := os.Stat(envShell); err == nil && !info.IsDir() {
			return envShell, nil
		}
	}
	for _, name := range shellCandidates {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no shell available on PATH (tried $SHELL and %v)", shellCandidates)
}

// isAutoCommand reports whether a manifest / config value means "let
// the runtime pick a shell". Accepts empty string (shorthand for
// auto — easy to type in configSchema "default": "") plus the literal
// "auto" in any case, with surrounding whitespace trimmed.
func isAutoCommand(s string) bool {
	t := strings.TrimSpace(s)
	return t == "" || strings.EqualFold(t, "auto")
}
