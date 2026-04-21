// Package main — `opendray service …` — OS-level background service
// lifecycle. Installs OpenDray as a system service so it auto-starts on
// boot, restarts on crash, and logs to a sensible place.
//
// Platform backends:
//   - Linux  → systemd unit at /etc/systemd/system/opendray.service
//   - macOS  → launchd plist at /Library/LaunchDaemons/com.opendray.opendray.plist
//
// Both backends run OpenDray under a non-root `User=` chosen at install
// time, matching setup's embedded-PG constraint (initdb refuses uid 0).
// When `opendray service install` is invoked via `sudo`, SUDO_USER is
// the default run-user; the operator can override via --user flag.
//
// All write operations (unit/plist file, systemctl/launchctl calls)
// require root — the subcommand bails early with a helpful hint if the
// effective uid isn't 0. Read-only operations (status, logs) work for
// any user.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	serviceName       = "opendray"
	systemdUnitPath   = "/etc/systemd/system/opendray.service"
	launchdLabel      = "com.opendray.opendray"
	launchdPlistPath  = "/Library/LaunchDaemons/com.opendray.opendray.plist"
	launchdLogDir     = "/var/log/opendray"
)

func runServiceCLI(args []string) int {
	if len(args) == 0 {
		printServiceHelp()
		return 2
	}
	switch args[0] {
	case "install":
		return serviceInstall(args[1:])
	case "uninstall", "remove":
		return serviceUninstall(args[1:])
	case "start":
		return serviceControl("start")
	case "stop":
		return serviceControl("stop")
	case "restart":
		return serviceControl("restart")
	case "status":
		return serviceStatus()
	case "logs", "log":
		return serviceLogs(args[1:])
	case "help", "-h", "--help":
		printServiceHelp()
		return 0
	default:
		prf("Unknown service subcommand: %s\n", args[0])
		printServiceHelp()
		return 2
	}
}

// ── install ─────────────────────────────────────────────────────────

type installFlags struct {
	user     string
	binary   string
	force    bool
	dryRun   bool
}

func serviceInstall(args []string) int {
	fs := flag.NewFlagSet("service install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flags := &installFlags{}
	fs.StringVar(&flags.user, "user", "", "unix user to run opendray as (default: $SUDO_USER or current user)")
	fs.StringVar(&flags.binary, "binary", "", "path to the opendray binary (default: auto-detect)")
	fs.BoolVar(&flags.force, "force", false, "overwrite an existing unit/plist without asking")
	fs.BoolVar(&flags.dryRun, "dry-run", false, "print the unit/plist that would be written; change nothing")
	if err := fs.Parse(args); err != nil {
		prf("service install: %v", err)
		return 2
	}

	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		prf("%s service install: unsupported OS %q (only linux + macOS are wired today)",
			failMark(), runtime.GOOS)
		return 1
	}

	// Figure out which user will run the service. If invoked via sudo,
	// SUDO_USER is what the operator was before su'ing — almost always
	// who they meant to run the service as. Fall back to the current
	// logged-in user otherwise. A root run-user is always rejected.
	runUser := flags.user
	if runUser == "" {
		if s := os.Getenv("SUDO_USER"); s != "" && s != "root" {
			runUser = s
		} else {
			// Not running under sudo. If the operator IS root directly,
			// we can't auto-pick a safe run-user — force --user.
			if os.Geteuid() == 0 {
				prf("%s service install: running as root with no SUDO_USER", failMark())
				prf("  Cannot auto-detect which user should run the service.")
				prf("  Re-run with --user explicit, e.g.:")
				prf("      opendray service install --user opendray")
				return 2
			}
			u, err := user.Current()
			if err != nil {
				prf("%s cannot detect current user: %v", failMark(), err)
				return 1
			}
			runUser = u.Username
		}
	}
	if runUser == "root" {
		prf("%s service install: refusing to run as root (bundled PG won't start as uid 0)", failMark())
		prf("  Choose a different user with --user.")
		return 2
	}

	// Verify the user exists + resolve their home dir (WorkingDirectory).
	u, err := user.Lookup(runUser)
	if err != nil {
		prf("%s service install: user %q not found on this system: %v", failMark(), runUser, err)
		return 1
	}

	// Binary path. Must be an absolute file readable by the target
	// user. Usually auto-detected from os.Executable() which resolves
	// the current binary (whoever ran this command is installing the
	// very file they ran).
	binPath := flags.binary
	if binPath == "" {
		exe, err := os.Executable()
		if err != nil {
			prf("%s cannot detect current binary path: %v", failMark(), err)
			return 1
		}
		abs, err := filepath.Abs(exe)
		if err != nil {
			binPath = exe
		} else {
			binPath = abs
		}
	}
	if info, err := os.Stat(binPath); err != nil {
		prf("%s binary not found at %s: %v", failMark(), binPath, err)
		return 1
	} else if info.IsDir() {
		prf("%s %s is a directory, not a binary", failMark(), binPath)
		return 1
	}

	// Render the unit/plist contents.
	var unit, unitPath string
	if runtime.GOOS == "linux" {
		unit = renderSystemdUnit(runUser, u.HomeDir, binPath)
		unitPath = systemdUnitPath
	} else {
		unit = renderLaunchdPlist(runUser, u.HomeDir, binPath)
		unitPath = launchdPlistPath
	}

	if flags.dryRun {
		prn("")
		prf("# Would write to %s:", unitPath)
		prn(styleDim(strings.Repeat("─", 60)))
		prn(unit)
		prn(styleDim(strings.Repeat("─", 60)))
		prn("")
		return 0
	}

	// All write operations need root.
	if os.Geteuid() != 0 {
		prf("%s service install needs root — run with sudo:", failMark())
		prn("")
		prf("    sudo %s service install %s", binPath,
			strings.Join(os.Args[2:], " "))
		prn("")
		return 1
	}

	// Overwrite check.
	if _, err := os.Stat(unitPath); err == nil && !flags.force {
		prf("%s %s already exists — pass --force to overwrite.", warnMark(), unitPath)
		return 1
	}

	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		prf("%s could not write %s: %v", failMark(), unitPath, err)
		return 1
	}
	prf("%s wrote %s", okMark(), unitPath)

	// Platform-specific enable + start.
	if runtime.GOOS == "linux" {
		if err := runSilent("systemctl", "daemon-reload"); err != nil {
			prf("%s systemctl daemon-reload failed: %v", failMark(), err)
			return 1
		}
		if err := runSilent("systemctl", "enable", "--now", serviceName); err != nil {
			prf("%s systemctl enable --now opendray failed: %v", failMark(), err)
			prf("  Check: systemctl status opendray   journalctl -u opendray")
			return 1
		}
		prf("%s enabled + started via systemctl", okMark())
	} else {
		// Ensure the log dir exists with permissions the service user
		// can write to. launchd runs the binary as `runUser` but the
		// log paths live under /var/log which is root-owned by default.
		if err := os.MkdirAll(launchdLogDir, 0o755); err == nil {
			_ = chownToUser(launchdLogDir, u)
		}
		if err := runSilent("launchctl", "load", "-w", unitPath); err != nil {
			prf("%s launchctl load failed: %v", failMark(), err)
			prf("  Check: sudo launchctl print system/%s", launchdLabel)
			return 1
		}
		prf("%s loaded via launchctl", okMark())
	}

	prn("")
	prn(styleTitle("   OpenDray is now running as a background service."))
	prn("")
	prn(styleDim("   View status:"))
	if runtime.GOOS == "linux" {
		prf("       %s", styleBrightCyan("systemctl status opendray"))
		prn(styleDim("   Tail logs:"))
		prf("       %s", styleBrightCyan("journalctl -fu opendray"))
	} else {
		prf("       %s", styleBrightCyan("sudo launchctl print system/"+launchdLabel))
		prn(styleDim("   Tail logs:"))
		prf("       %s", styleBrightCyan("tail -f "+launchdLogDir+"/opendray.out.log "+launchdLogDir+"/opendray.err.log"))
	}
	prn("")
	return 0
}

// ── uninstall ───────────────────────────────────────────────────────

func serviceUninstall(args []string) int {
	fs := flag.NewFlagSet("service uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var keepLogs bool
	fs.BoolVar(&keepLogs, "keep-logs", false, "leave /var/log/opendray/ in place (macOS only)")
	_ = fs.Parse(args)

	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		prf("%s service uninstall: unsupported OS %q", failMark(), runtime.GOOS)
		return 1
	}
	if os.Geteuid() != 0 {
		prf("%s service uninstall needs root — run with sudo.", failMark())
		return 1
	}

	if runtime.GOOS == "linux" {
		// Best-effort stop + disable; both are idempotent and we don't
		// want to bail on the (mild) error of a service that wasn't
		// actually enabled.
		_ = runSilent("systemctl", "stop", serviceName)
		_ = runSilent("systemctl", "disable", serviceName)
		if _, err := os.Stat(systemdUnitPath); err == nil {
			if err := os.Remove(systemdUnitPath); err != nil {
				prf("%s could not remove %s: %v", failMark(), systemdUnitPath, err)
				return 1
			}
			prf("%s removed %s", okMark(), systemdUnitPath)
		}
		_ = runSilent("systemctl", "daemon-reload")
	} else {
		// launchctl unload removes from the running set; the -w flag
		// also clears Disabled=true in the plist database.
		_ = runSilent("launchctl", "unload", "-w", launchdPlistPath)
		if _, err := os.Stat(launchdPlistPath); err == nil {
			if err := os.Remove(launchdPlistPath); err != nil {
				prf("%s could not remove %s: %v", failMark(), launchdPlistPath, err)
				return 1
			}
			prf("%s removed %s", okMark(), launchdPlistPath)
		}
		if !keepLogs {
			_ = os.RemoveAll(launchdLogDir)
		}
	}
	prn("")
	prf("%s service removed.", okMark())
	prn("")
	return 0
}

// ── start / stop / restart ──────────────────────────────────────────

func serviceControl(action string) int {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		prf("%s service %s: unsupported OS %q", failMark(), action, runtime.GOOS)
		return 1
	}
	if os.Geteuid() != 0 {
		prf("%s service %s needs root — run with sudo.", failMark(), action)
		return 1
	}

	if runtime.GOOS == "linux" {
		if err := runInherit("systemctl", action, serviceName); err != nil {
			return 1
		}
	} else {
		// launchctl doesn't have a direct "restart". Do stop + start.
		switch action {
		case "start":
			if err := runInherit("launchctl", "load", "-w", launchdPlistPath); err != nil {
				return 1
			}
		case "stop":
			if err := runInherit("launchctl", "unload", launchdPlistPath); err != nil {
				return 1
			}
		case "restart":
			_ = runInherit("launchctl", "unload", launchdPlistPath)
			if err := runInherit("launchctl", "load", "-w", launchdPlistPath); err != nil {
				return 1
			}
		}
	}
	prf("%s %s", okMark(), action)
	return 0
}

// ── status ──────────────────────────────────────────────────────────

func serviceStatus() int {
	if runtime.GOOS == "linux" {
		if err := runInherit("systemctl", "status", serviceName, "--no-pager"); err != nil {
			// systemctl status returns 3 when the unit isn't active.
			// That's informational, not a failure of this CLI.
			return 0
		}
		return 0
	}
	if runtime.GOOS == "darwin" {
		// launchctl print works without sudo for reading. The verbose
		// output is the most user-friendly "here's what the service is
		// doing right now" we can get on mac.
		_ = runInherit("launchctl", "print", "system/"+launchdLabel)
		return 0
	}
	prf("%s service status: unsupported OS %q", failMark(), runtime.GOOS)
	return 1
}

// ── logs ────────────────────────────────────────────────────────────

func serviceLogs(args []string) int {
	fs := flag.NewFlagSet("service logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var follow bool
	var lines int
	fs.BoolVar(&follow, "follow", true, "tail logs continuously (Ctrl-C to stop)")
	fs.BoolVar(&follow, "f", true, "alias for --follow")
	fs.IntVar(&lines, "lines", 50, "how many lines of history to show before tailing")
	fs.IntVar(&lines, "n", 50, "alias for --lines")
	_ = fs.Parse(args)

	if runtime.GOOS == "linux" {
		cmdArgs := []string{"-u", serviceName, fmt.Sprintf("-n%d", lines)}
		if follow {
			cmdArgs = append(cmdArgs, "-f")
		}
		return execPassthrough("journalctl", cmdArgs...)
	}
	if runtime.GOOS == "darwin" {
		out := filepath.Join(launchdLogDir, "opendray.out.log")
		errLog := filepath.Join(launchdLogDir, "opendray.err.log")
		cmdArgs := []string{fmt.Sprintf("-n%d", lines)}
		if follow {
			cmdArgs = append(cmdArgs, "-f")
		}
		cmdArgs = append(cmdArgs, out, errLog)
		return execPassthrough("tail", cmdArgs...)
	}
	prf("%s service logs: unsupported OS %q", failMark(), runtime.GOOS)
	return 1
}

// ── renderers ───────────────────────────────────────────────────────

func renderSystemdUnit(runUser, homeDir, binPath string) string {
	// After=network-online.target — we need outbound DNS / HTTPS to
	//   reach agent marketplaces, OAuth endpoints, etc.
	// Restart=on-failure — don't thrash on normal `opendray uninstall`
	//   shutdowns, but recover from crashes.
	// Environment=HOME/LOGNAME — some subcommands (gh CLI, ssh) look up
	//   the user via these vars; systemd doesn't set them by default.
	// ProtectSystem=full + NoNewPrivileges=true — tighten what a
	//   compromised opendray process can touch on the host. We still
	//   need write access to the home dir so config + PG data + logs
	//   reach disk, hence `full` rather than `strict`.
	return fmt.Sprintf(`[Unit]
Description=OpenDray — self-hosted AI workbench
Documentation=https://github.com/Opendray/opendray
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s
ExecStart=%s
Environment=HOME=%s
Environment=LOGNAME=%s
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=read-only
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, runUser, runUser, homeDir, binPath, homeDir, runUser, homeDir)
}

func renderLaunchdPlist(runUser, homeDir, binPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>UserName</key>
    <string>%s</string>
    <key>GroupName</key>
    <string>staff</string>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>HOME</key>
        <string>%s</string>
        <key>LOGNAME</key>
        <string>%s</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>%s/opendray.out.log</string>
    <key>StandardErrorPath</key>
    <string>%s/opendray.err.log</string>
</dict>
</plist>
`, launchdLabel, runUser, homeDir, binPath, homeDir, runUser, launchdLogDir, launchdLogDir)
}

// ── exec helpers ────────────────────────────────────────────────────

// runSilent runs a command, capturing and discarding output on
// success; surfaces stderr on failure. Intended for one-shot calls
// like `systemctl daemon-reload` where no output is expected.
func runSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		if len(out) > 0 {
			prn(string(out))
		}
		return err
	}
	return nil
}

// runInherit runs a command with stdin/out/err attached to the current
// terminal — useful for commands whose output the operator wants to
// see (systemctl status, launchctl print).
func runInherit(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// execPassthrough is for long-running commands that follow / tail.
// Replaces the current process on Unix so Ctrl-C behaves naturally.
// (If exec fails for some reason, fall back to a child process.)
func execPassthrough(name string, args ...string) int {
	// Try the system-wide PATH resolution first so we exec an absolute
	// path (syscall.Exec requires it on some kernels).
	path, err := exec.LookPath(name)
	if err != nil {
		prf("%s %s: %v", failMark(), name, err)
		return 1
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Don't print — the child has already written its own error
		// stream to stderr.
		return 1
	}
	return 0
}

// chownToUser is a best-effort shell out to `chown` so we get the
// same behavior the user would from the command line.
func chownToUser(path string, u *user.User) error {
	cmd := exec.Command("chown", fmt.Sprintf("%s:staff", u.Username), path)
	return cmd.Run()
}

// ── help ────────────────────────────────────────────────────────────

func printServiceHelp() {
	fmt.Println(`opendray service — manage the OpenDray background service.

Usage:
  opendray service <command> [flags]

Commands:
  install      Install as systemd (Linux) or launchd (macOS) service.
               Runs as an unprivileged user; needs sudo for install itself.
               Flags: --user, --binary, --force, --dry-run
  uninstall    Stop + remove the service file. Needs sudo.
  start        Start the service now. Needs sudo.
  stop         Stop the running service. Needs sudo.
  restart      Stop then start. Needs sudo.
  status       Show what the service is doing right now.
  logs         Tail service logs. Flags: -f/--follow (default on), -n/--lines N.

Examples:
  sudo opendray service install --user linivek
  opendray service status
  opendray service logs -n 200
  sudo opendray service restart
  sudo opendray service uninstall

Notes:
  • The run-user defaults to $SUDO_USER (the account that invoked sudo).
  • Running as root is refused — bundled PostgreSQL's initdb rejects uid 0.
  • Linux uses systemd + journald; macOS uses launchd + /var/log/opendray/.`)
}
