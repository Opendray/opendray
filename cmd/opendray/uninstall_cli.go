// Package main — `opendray uninstall` — terminal uninstall wizard.
//
// Mirrors `opendray setup` but in reverse: enumerates the artefacts
// setup left behind, prints them for review, and removes them after
// confirmation. The visual language (ASCII logo, section headers,
// color palette) is shared with setup_cli.go so the two wizards feel
// like the same tool.
//
// What gets removed:
//   - the binary itself (self-delete on Unix; helper .cmd on Windows)
//   - ~/.opendray/ (data dir: PGDATA, cache, plugins, marketplace)
//   - config.toml at any of its search paths (XDG + home)
//
// What does NOT get removed automatically:
//   - the external PostgreSQL schema, if DB mode was "external"
//     (OpenDray's tables may share names with other apps). A helper
//     SQL script is written to the cwd with DROP TABLE statements
//     the operator can review and apply manually.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/opendray/opendray/kernel/config"
)

// opendrayTables is the flat list of tables OpenDray creates in its
// database. Hardcoded (rather than introspected from the DB) so the
// uninstall flow works even when the DB is unreachable. Kept in sync
// with kernel/store/migrations/*.sql.
var opendrayTables = []string{
	"sessions",
	"plugins",
	"mcp_servers",
	"claude_accounts",
	"llm_providers",
	"admin_auth",
	"plugin_consents",
	"plugin_kv",
	"plugin_secret",
	"plugin_audit",
	"plugin_secret_kek",
	"plugin_host_state",
	"plugin_tombstone",
}

type uninstallFlags struct {
	yes      bool
	dryRun   bool
	keepData bool
}

func parseUninstallFlags(args []string) (*uninstallFlags, error) {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	out := &uninstallFlags{}
	fs.BoolVar(&out.yes, "yes", false, "skip confirmation prompt")
	fs.BoolVar(&out.dryRun, "dry-run", false, "print the plan, remove nothing")
	fs.BoolVar(&out.keepData, "keep-data", false,
		"keep ~/.opendray/ (PG data + plugins + cache). Binary + config still removed.")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return out, nil
}

// runUninstallCLI is the `opendray uninstall` entry point.
func runUninstallCLI() int {
	flags, err := parseUninstallFlags(os.Args[2:])
	if err != nil {
		prf("uninstall: %v", err)
		return 2
	}

	uninstallBanner()

	// Load whatever config exists so we can report what mode was in
	// use (embedded vs external) and where the data dir sits. Missing
	// config isn't fatal — the user might be uninstalling precisely
	// because setup never completed.
	cfg, _, loadErr := config.Load()

	// Collect removal targets.
	targets := collectTargets(cfg, loadErr != nil)

	// Show plan.
	section(1, 3, "plan")
	describePlan(targets, cfg, flags)

	if flags.dryRun {
		prn("")
		prf("   %s  %s", warnMark(), styleYellow("--dry-run set; nothing will be removed."))
		prn("")
		return 0
	}

	// Confirm.
	if !flags.yes {
		prn("")
		in := bufio.NewReader(os.Stdin)
		if !askYN(in, "Proceed with removal?", false) {
			prn("")
			prf("   %s", styleYellow("Aborted. Nothing changed."))
			prn("")
			return 0
		}
	}

	// Teardown.
	section(2, 3, "teardown")

	// (1) Stop running processes so we don't delete files PG is
	//     still writing.
	if targets.embeddedMode {
		stopEmbeddedProcesses(cfg)
	}
	stopOpenDrayServer(cfg)

	// (2) Generate the external-DB drop script BEFORE we delete the
	//     config so the summary can print useful connection details
	//     even in the keep-data branch.
	var dropSQLPath string
	if targets.externalMode {
		path, err := writeDropSchemaSQL(cfg)
		if err != nil {
			prf("   %s could not write drop-schema SQL: %v", warnMark(), err)
		} else {
			dropSQLPath = path
		}
	}

	// (3) Delete files.
	deleteTargets(targets, flags)

	// (4) Binary self-delete (or defer via helper on Windows).
	section(3, 3, "remove binary")
	removeSelfBinary(targets.binaryPath)

	// (5) Farewell + external-DB reminder.
	farewell(targets, dropSQLPath)
	return 0
}

// ── banner ──────────────────────────────────────────────────────────

func uninstallBanner() {
	prn("")
	for _, line := range strings.Split(strings.TrimLeft(asciiLogo, "\n"), "\n") {
		prn(styleAccent(line))
	}
	prn("")
	prf("   %s  %s  %s",
		styleTitle("UNINSTALL"),
		styleDim("·"),
		styleDim(fmt.Sprintf("version %s", version)))
	prn("")
	prn("")
	prn(styleTitle("   This will:"))
	prn("")
	prn(styleDim("     1.") + "   Stop any running OpenDray server + bundled PostgreSQL")
	prn(styleDim("     2.") + "   Remove the data directory and config file")
	prn(styleDim("     3.") + "   Remove the " + styleCyan("opendray") + " binary from disk")
	prn("")
	prn(styleDim("   External databases (if used) are NOT touched — a"))
	prn(styleDim("   ") + styleCyan("drop_opendray_schema.sql") + styleDim(" helper is written to the current directory"))
	prn(styleDim("   so you can review and drop tables manually."))
	prn("")
}

// ── plan collection ─────────────────────────────────────────────────

// removalTarget is one file or directory queued for deletion plus a
// human label for the plan display.
type removalTarget struct {
	path  string
	label string
	kind  string // "file", "dir", "binary"
}

type plan struct {
	binaryPath    string            // os.Executable() result
	binaryHome    string            // install dir for PATH cleanup
	targets       []removalTarget   // everything else to rm
	configPaths   []string          // all found config files
	dataDir       string            // ~/.opendray
	embeddedMode  bool
	externalMode  bool
}

func collectTargets(cfg config.Config, cfgMissing bool) plan {
	p := plan{}

	// Mode detection — decides whether we print the external-DB
	// reminder and whether we try to stop embedded PG.
	if cfgMissing {
		p.embeddedMode = false
		p.externalMode = false
	} else {
		switch cfg.DB.Mode {
		case "external":
			p.externalMode = true
		default:
			p.embeddedMode = true
		}
	}

	// Binary path — the running executable.
	if exe, err := os.Executable(); err == nil {
		if abs, err := filepath.Abs(exe); err == nil {
			p.binaryPath = abs
			p.binaryHome = filepath.Dir(abs)
		} else {
			p.binaryPath = exe
			p.binaryHome = filepath.Dir(exe)
		}
	}

	// Data directory — the one real home for everything OpenDray
	// stores on the local box.
	home, _ := os.UserHomeDir()
	if home != "" {
		p.dataDir = filepath.Join(home, ".opendray")
		if _, err := os.Stat(p.dataDir); err == nil {
			p.targets = append(p.targets, removalTarget{
				path:  p.dataDir,
				label: "data dir (PG cluster, plugins, cache, marketplace)",
				kind:  "dir",
			})
		}
	}

	// Config file — check every candidate path; include each one that
	// actually exists AND sits outside the data dir (rm -rf dataDir
	// already kills the one inside it).
	for _, c := range config.DefaultPaths() {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		if p.dataDir != "" && strings.HasPrefix(abs, p.dataDir+string(filepath.Separator)) {
			continue // will be removed as part of the data dir
		}
		p.targets = append(p.targets, removalTarget{
			path:  abs,
			label: "config file",
			kind:  "file",
		})
		p.configPaths = append(p.configPaths, abs)
	}

	return p
}

func describePlan(p plan, cfg config.Config, flags *uninstallFlags) {
	if len(p.targets) == 0 && p.binaryPath == "" {
		prf("     %s Nothing to remove — OpenDray isn't installed here.", warnMark())
		return
	}

	row := func(kind, path string) {
		prf("     %s  %s", styleGreen("•"), styleBrightCyan(path))
		prf("        %s", styleDim(kind))
		prn("")
	}

	prn(styleTitle("   Will remove:"))
	prn("")

	if p.binaryPath != "" {
		row("binary", p.binaryPath)
	}
	if flags.keepData {
		// Only config files from the targets list — data dir stays.
		for _, t := range p.targets {
			if t.kind != "dir" {
				row(t.label, t.path)
			}
		}
		prf("     %s  %s", warnMark(), styleYellow("--keep-data: data directory will be left in place."))
		prn("")
	} else {
		for _, t := range p.targets {
			row(t.label, t.path)
		}
	}

	if p.externalMode {
		prn(styleTitle("   Will NOT touch:"))
		prn("")
		prf("     %s  external PostgreSQL at %s",
			warnMark(),
			styleBrightCyan(fmt.Sprintf("%s@%s:%d/%s",
				cfg.DB.External.User, cfg.DB.External.Host,
				cfg.DB.External.Port, cfg.DB.External.Name)))
		prf("        %s", styleDim("a drop_opendray_schema.sql helper will be written for manual cleanup"))
		prn("")
	}
}

// ── destructive actions ─────────────────────────────────────────────

func deleteTargets(p plan, flags *uninstallFlags) {
	if flags.keepData {
		// keep-data: only remove config files + skip data-dir tree.
		for _, t := range p.targets {
			if t.kind == "dir" {
				continue
			}
			progress(fmt.Sprintf("remove %s", t.path), func() error {
				return os.Remove(t.path)
			})
		}
		return
	}
	for _, t := range p.targets {
		target := t
		progress(fmt.Sprintf("remove %s", target.path), func() error {
			if target.kind == "dir" {
				return os.RemoveAll(target.path)
			}
			return os.Remove(target.path)
		})
	}
}

// removeSelfBinary unlinks the running binary. On Unix, unlinking a
// running executable is supported — the process keeps its file
// descriptor until exit. On Windows, the filesystem keeps the file
// locked, so we schedule a helper batch that waits for us to exit
// and then deletes both the .exe and itself.
func removeSelfBinary(path string) {
	if path == "" {
		prf("   %s could not locate running binary; nothing removed", warnMark())
		return
	}

	if runtime.GOOS == "windows" {
		if err := scheduleWindowsSelfDelete(path); err != nil {
			prf("   %s could not schedule delete of %s: %v", failMark(), path, err)
			prf("         delete it manually.")
			return
		}
		prf("   %s scheduled removal of %s (applied after this process exits)", okMark(), path)
		return
	}

	if err := os.Remove(path); err != nil {
		prf("   %s could not remove %s: %v", failMark(), path, err)
		return
	}
	prf("   %s removed %s", okMark(), path)
}

// scheduleWindowsSelfDelete writes a small batch file to %TEMP% that
// polls for the parent process to exit, deletes the .exe, then deletes
// itself. Uses `start /b` to detach from the console so closing the
// current terminal doesn't kill it.
func scheduleWindowsSelfDelete(exePath string) error {
	tmpDir, err := os.MkdirTemp("", "opendray-uninstall-")
	if err != nil {
		return err
	}
	batPath := filepath.Join(tmpDir, "remove-opendray.cmd")
	pid := os.Getpid()
	// The batch waits for our PID to exit (tasklist returns ERRORLEVEL
	// 1 when no matching process found), then retries the delete a few
	// times in case AV has a lock.
	script := fmt.Sprintf(`@echo off
:wait
tasklist /FI "PID eq %d" 2>nul | find "%d" >nul
if not errorlevel 1 (
    timeout /t 1 /nobreak >nul
    goto wait
)
del /q "%s" >nul 2>&1
if exist "%s" (
    timeout /t 2 /nobreak >nul
    del /q "%s" >nul 2>&1
)
rmdir /q /s "%s" >nul 2>&1
`, pid, pid, exePath, exePath, exePath, tmpDir)

	if err := os.WriteFile(batPath, []byte(script), 0o700); err != nil {
		return err
	}

	cmd := exec.Command("cmd.exe", "/c", "start", "/b", "", batPath)
	return cmd.Start()
}

// stopOpenDrayServer tries to gracefully shut down a running server
// that's bound to cfg.Server.ListenAddr. Best-effort — nothing
// downstream blocks on its success, just nicer than hitting
// "permission denied" when we try to remove a file PG still has open.
func stopOpenDrayServer(cfg config.Config) {
	addr := cfg.Server.ListenAddr
	if addr == "" {
		addr = "127.0.0.1:8640"
	}

	// Quick probe: connect, if nothing answers we're done.
	ln, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		// Nothing listening → no server running → nothing to do.
		return
	}
	_ = ln.Close()

	pid := findPIDByListenAddr(addr)
	if pid <= 0 {
		prf("   %s a process is listening on %s, but we couldn't identify its PID.",
			warnMark(), addr)
		prf("         stop it manually if removal fails.")
		return
	}
	if pid == os.Getpid() {
		// Shouldn't happen — the uninstall wizard doesn't bind a port.
		return
	}
	killGracefully(pid)
}

// stopEmbeddedProcesses terminates any bundled-Postgres backend that
// still has files in cfg.DB.Embedded.DataDir. The embedded-postgres
// library doesn't leave a registered service — we grep the process
// list for postgres children whose DATA path matches ours.
func stopEmbeddedProcesses(cfg config.Config) {
	dataDir := cfg.DB.Embedded.DataDir
	if dataDir == "" {
		return
	}
	port := cfg.DB.Embedded.Port
	if port == 0 {
		port = 5433
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	if _, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err != nil {
		return // not running
	}

	pid := findPIDByListenAddr(addr)
	if pid > 0 {
		killGracefully(pid)
		return
	}
	prf("   %s bundled PostgreSQL appears to still be running on %s;",
		warnMark(), addr)
	prf("         stop it manually before re-running if removal fails.")
}

// findPIDByListenAddr uses `lsof -i :<port>` on Unix and
// `netstat -ano | findstr` on Windows to map a TCP listener to a PID.
// Returns 0 when nothing matches.
func findPIDByListenAddr(addr string) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}

	if runtime.GOOS == "windows" {
		out, err := exec.Command("netstat", "-ano").Output()
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, ":"+port) {
				continue
			}
			if !strings.Contains(line, "LISTENING") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			if pid, err := strconv.Atoi(fields[len(fields)-1]); err == nil {
				return pid
			}
		}
		return 0
	}

	// Unix: lsof is available on macOS + most Linux distros.
	if _, err := exec.LookPath("lsof"); err != nil {
		return 0
	}
	out, err := exec.Command("lsof", "-t", "-i", ":"+port, "-sTCP:LISTEN").Output()
	if err != nil {
		return 0
	}
	firstLine := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	if pid, err := strconv.Atoi(firstLine); err == nil {
		return pid
	}
	return 0
}

// killGracefully sends SIGTERM, waits up to 3 seconds, then SIGKILL.
// On Windows os.Process.Signal(syscall.SIGTERM) is a no-op — we use
// Kill() directly there, which is the MS-blessed way to terminate
// another process short of a clean service Stop.
func killGracefully(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	prf("   %s  stopping PID %d …", styleAccent("→"), pid)

	if runtime.GOOS == "windows" {
		_ = proc.Kill()
		// Give the kernel a moment to flush.
		time.Sleep(200 * time.Millisecond)
		prf("   %s  stopped (PID %d)", okMark(), pid)
		return
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Already gone?
		return
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			prf("   %s  stopped (PID %d)", okMark(), pid)
			return
		}
	}
	_ = proc.Kill()
	prf("   %s  force-stopped (PID %d)", okMark(), pid)
}

// ── external-DB drop script ─────────────────────────────────────────

func writeDropSchemaSQL(cfg config.Config) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := filepath.Join(cwd, "drop_opendray_schema.sql")

	var b strings.Builder
	fmt.Fprintln(&b, "-- OpenDray schema-drop helper")
	fmt.Fprintln(&b, "--")
	fmt.Fprintf (&b, "-- Generated by `opendray uninstall` at %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintln(&b, "--")
	fmt.Fprintf (&b, "-- Target: %s@%s:%d/%s\n",
		cfg.DB.External.User, cfg.DB.External.Host,
		cfg.DB.External.Port, cfg.DB.External.Name)
	fmt.Fprintln(&b, "--")
	fmt.Fprintln(&b, "-- WARNING")
	fmt.Fprintln(&b, "--   These table names are generic (sessions, plugins, admin_auth, …)")
	fmt.Fprintln(&b, "--   and may conflict with other applications sharing this database.")
	fmt.Fprintln(&b, "--   Review the list below before applying. Remove any line you want")
	fmt.Fprintln(&b, "--   to keep.")
	fmt.Fprintln(&b, "--")
	fmt.Fprintln(&b, "-- Apply with:")
	fmt.Fprintf (&b, "--   psql -h %s -p %d -U %s -d %s -f %s\n",
		cfg.DB.External.Host, cfg.DB.External.Port,
		cfg.DB.External.User, cfg.DB.External.Name,
		filepath.Base(path))
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "BEGIN;")
	for _, t := range opendrayTables {
		fmt.Fprintf(&b, "DROP TABLE IF EXISTS %s CASCADE;\n", t)
	}
	fmt.Fprintln(&b, "COMMIT;")

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ── farewell ────────────────────────────────────────────────────────

func farewell(p plan, dropSQLPath string) {
	prn("")
	prn("")
	prn(styleGreen(divider))
	prn("")
	prf("     %s   %s", okMark(), styleTitle("UNINSTALL COMPLETE"))
	prn("")
	prn(styleGreen(divider))
	prn("")
	prn("")

	if p.externalMode && dropSQLPath != "" {
		prn(styleTitle("   External PostgreSQL tables left in place."))
		prn("")
		prf("   Review and apply the drop script with:")
		prn("")
		prf("       %s", styleBrightCyan(fmt.Sprintf(
			"psql -h <host> -U <user> -d <db> -f %s", dropSQLPath)))
		prn("")
	}

	if runtime.GOOS == "windows" && p.binaryHome != "" {
		prn(styleTitle("   Windows PATH reminder:"))
		prn("")
		prf("     %s is still listed in your user PATH.", styleBrightCyan(p.binaryHome))
		prf("     Clean it up manually with System Properties → Environment Variables.")
		prn("")
	}

	// Unix PATH hint — ~/.local/bin is a common default.
	if runtime.GOOS != "windows" && p.binaryHome != "" {
		prf("   %s", styleDim(
			fmt.Sprintf("(Your PATH may still reference %s — safe to leave, harmless.)",
				p.binaryHome)))
		prn("")
	}
}

// ── compile-time unused import silencer ─────────────────────────────
// Keeps imports we bring in for their side effects / future use from
// sliding out during iteration.
var _ = context.Background
