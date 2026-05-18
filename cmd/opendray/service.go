// Service-control subcommands.
//
//	opendray start    start the opendray service (systemd on Linux, launchd on macOS)
//	opendray stop     stop the service
//	opendray restart  stop+start
//	opendray status   show service status
//
// These are thin shells over `systemctl` (Linux) and `launchctl`
// (macOS) — they exist so operators don't have to remember the
// platform-native incantation. Identical surface to the systemd / launchd
// units the install wizard sets up.
//
// On Linux: privilege is delegated to systemctl. If the operator isn't
// root, the binary prepends `sudo`; if there's no sudo and no root,
// the user is told to run with elevation.
//
// On macOS: assumes the user LaunchAgent at `gui/$UID/com.opendray.opendray`
// that the macOS installer creates by default. Pass `--system` to target
// the LaunchDaemon scope instead (matches install-macos.sh's
// `--launchd-daemon` flag).
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const (
	systemdUnitName = "opendray"
	launchdLabel    = "com.opendray.opendray"
)

func runStart(args []string) int   { return runServiceAction("start", args) }
func runStop(args []string) int    { return runServiceAction("stop", args) }
func runRestart(args []string) int { return runServiceAction("restart", args) }
func runStatus(args []string) int  { return runServiceAction("status", args) }

func runServiceAction(action string, args []string) int {
	fs := flag.NewFlagSet(action, flag.ExitOnError)
	system := fs.Bool("system", false, "macOS only: target the system LaunchDaemon instead of the user LaunchAgent")
	_ = fs.Parse(args)

	switch runtime.GOOS {
	case "linux":
		return linuxService(action)
	case "darwin":
		return macService(action, *system)
	default:
		fmt.Fprintf(os.Stderr, "`opendray %s` is not supported on %s — use the platform-native service manager directly.\n", action, runtime.GOOS)
		return 1
	}
}

func linuxService(action string) int {
	if _, err := exec.LookPath("systemctl"); err != nil {
		fmt.Fprintln(os.Stderr, "systemctl not found — `opendray start/stop/restart/status` requires systemd.")
		return 1
	}

	var cmd *exec.Cmd
	if os.Geteuid() == 0 {
		cmd = exec.Command("systemctl", action, systemdUnitName)
	} else if _, err := exec.LookPath("sudo"); err == nil {
		cmd = exec.Command("sudo", "systemctl", action, systemdUnitName)
	} else {
		fmt.Fprintf(os.Stderr, "service control needs root; rerun as root or install sudo.\n  systemctl %s %s\n", action, systemdUnitName)
		return 1
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "systemctl %s failed: %v\n", action, err)
		return 1
	}
	return 0
}

func macService(action string, system bool) int {
	if _, err := exec.LookPath("launchctl"); err != nil {
		fmt.Fprintln(os.Stderr, "launchctl not found — wrong macOS?")
		return 1
	}

	var target string
	if system {
		target = "system/" + launchdLabel
	} else {
		target = fmt.Sprintf("gui/%d/%s", os.Getuid(), launchdLabel)
	}

	var cmd *exec.Cmd
	switch action {
	case "start":
		cmd = exec.Command("launchctl", "kickstart", target)
	case "stop":
		// `bootout` removes the loaded unit so a future `start` has to
		// `bootstrap` it again. For a pure "pause", we'd use `kill TERM`,
		// but the user expectation of `stop` matches the install wizard's
		// "stop the service entirely" semantics.
		cmd = exec.Command("launchctl", "bootout", target)
	case "restart":
		cmd = exec.Command("launchctl", "kickstart", "-k", target)
	case "status":
		cmd = exec.Command("launchctl", "print", target)
	default:
		fmt.Fprintf(os.Stderr, "internal error: unknown action %q\n", action)
		return 2
	}

	if system && os.Geteuid() != 0 {
		if _, err := exec.LookPath("sudo"); err != nil {
			fmt.Fprintln(os.Stderr, "system-scope launchd commands need root; rerun with sudo.")
			return 1
		}
		// Prepend sudo by re-wrapping argv.
		newArgs := append([]string{cmd.Path}, cmd.Args[1:]...)
		cmd = exec.Command("sudo", newArgs...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "launchctl %s failed: %v\n", action, err)
		return 1
	}
	return 0
}
