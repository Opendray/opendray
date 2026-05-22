// `opendray self-update --apply` is the privileged half of the
// in-dashboard upgrade. It is NOT meant to be run by hand — the
// opendray-selfupdate.service oneshot (root) runs it when the
// opendray-selfupdate.path unit sees the daemon drop a request file.
//
// The unprivileged daemon can't replace /usr/local/bin/opendray or restart
// the unit, so the dashboard only queues a request; this command, running
// as root, applies the official latest release (checksum-verified, via the
// same path as `opendray update`), ensures the W^X drop-in so the upgraded
// daemon doesn't relapse into the codex/gemini JIT crash (#239), restarts,
// and clears the request.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/selfupdate"
)

const noMdwxDropIn = "/etc/systemd/system/opendray.service.d/no-mdwx.conf"

func selfUpdateStateDir() string {
	if d := strings.TrimSpace(os.Getenv("OPENDRAY_STATE_DIR")); d != "" {
		return d
	}
	return "/var/lib/opendray"
}

func runSelfUpdate(args []string) int {
	fs := flag.NewFlagSet("self-update", flag.ExitOnError)
	apply := fs.Bool("apply", false, "apply a queued upgrade request (run as root by the systemd oneshot)")
	_ = fs.Parse(args)
	if !*apply {
		fmt.Fprintln(os.Stderr, "usage: opendray self-update --apply  (invoked by opendray-selfupdate.service; not for manual use)")
		return 2
	}

	reqPath := selfupdate.RequestPath(selfUpdateStateDir())
	req, err := selfupdate.ReadRequest(reqPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no/invalid self-update request at %s: %v\n", reqPath, err)
		return 1
	}
	// Always clear the trigger so the path unit doesn't re-fire, even if
	// the upgrade fails below.
	defer os.Remove(reqPath)
	fmt.Printf("self-update: queued upgrade to %s by %s at %s\n",
		req.Version, req.RequestedBy, req.RequestedAt.Format(time.RFC3339))

	// Ensure the W^X drop-in so the upgraded daemon doesn't relapse into the
	// codex/gemini JIT crash — `update` alone never touches the unit (#239).
	ensureNoMdwxDropIn()

	// Binary swap + restart via the existing checksum-verified path. As
	// root (the oneshot) the /usr/local/bin write and systemctl restart
	// both succeed. It re-resolves "latest" from the canonical repo and
	// verifies the download, so the request can't redirect it elsewhere.
	return runUpdate([]string{"--yes", "--restart"})
}

// ensureNoMdwxDropIn writes the W^X override if absent (idempotent;
// preserves an existing drop-in) and reloads systemd to pick it up.
func ensureNoMdwxDropIn() {
	if _, err := os.Stat(noMdwxDropIn); err == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(noMdwxDropIn), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "[!] couldn't create drop-in dir: %v\n", err)
		return
	}
	const content = "# Added by `opendray self-update`: V8/Node JIT (codex, gemini) needs\n" +
		"# W^X (RW->RX mprotect), which MemoryDenyWriteExecute blocks.\n" +
		"[Service]\nMemoryDenyWriteExecute=false\n"
	if err := os.WriteFile(noMdwxDropIn, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "[!] couldn't write %s: %v\n", noMdwxDropIn, err)
		return
	}
	fmt.Printf("[✓] ensured %s\n", noMdwxDropIn)
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[!] daemon-reload failed: %v\n", err)
	}
}
