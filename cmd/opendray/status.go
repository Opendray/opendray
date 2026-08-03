// `opendray status` — a systemctl-style health summary.
//
// The platform service managers speak in their own dialects: macOS'
// `launchctl print` dumps ~80 lines of launchd internals, and `systemctl
// status` is readable but still process-centric. Neither answers the
// question an operator actually asks — "is opendray healthy right now?" —
// because the process being alive says nothing about whether the gateway is
// serving HTTP or can reach its database.
//
// This renders a compact checklist that fuses three sources:
//
//	launchd / systemd   → is the process up, pid, restart count, last exit
//	GET /health         → version, uptime, DB reachability (the REAL liveness)
//	config + TCC checks  → where config lives, privacy-folder gotchas
//
// `opendray status --raw` falls back to the native `launchctl print` /
// `systemctl status` for when the low-level detail is actually wanted.
//
// Exit code mirrors severity so scripts can gate on it (like systemctl):
// 0 = all green, 1 = degraded/warnings, 3 = not running / unreachable.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/version"
)

// checkState is the severity of a single dashboard row.
type checkState int

const (
	stateOK checkState = iota
	stateWarn
	stateFail
	stateUnknown
)

func (s checkState) icon() string {
	switch s {
	case stateOK:
		return "✓"
	case stateWarn:
		return "⚠"
	case stateFail:
		return "✗"
	default:
		return "·"
	}
}

// exitCode maps the worst row severity to a process exit code, matching
// systemd's convention (3 = not running).
func (s checkState) exitCode() int {
	switch s {
	case stateWarn:
		return 1
	case stateFail:
		return 3
	default:
		return 0
	}
}

// procInfo is the platform-agnostic view of the service process, distilled
// from `launchctl print` (macOS) or `systemctl show` (Linux).
type procInfo struct {
	found    bool // the unit is loaded/bootstrapped at all
	running  bool
	pid      int
	restarts int
	lastExit int
}

// healthInfo is the parsed GET /health response; reachable=false means the
// gateway did not answer HTTP (process may still exist but isn't serving).
type healthInfo struct {
	reachable bool
	status    string
	version   string
	commit    string
	uptime    time.Duration
	dbOK      bool
}

// ── rendering ──────────────────────────────────────────────────────

type statusRow struct {
	label string
	state checkState
	text  string
}

func renderStatusDashboard(system bool) int {
	home, _ := os.UserHomeDir()
	cfgPath := activeConfigPath()

	listen := "127.0.0.1:8770"
	if cfgPath != "" {
		if cfg, err := config.Load(cfgPath); err == nil && strings.TrimSpace(cfg.Listen) != "" {
			listen = cfg.Listen
		}
	}

	proc := gatherProc(system)
	health := probeHealth(listen)

	rows := buildStatusRows(proc, health, listen, cfgPath, home)

	// Header: prefer the running gateway's self-reported version, fall back
	// to this CLI binary's build info.
	ver, commit := version.Version, version.Commit
	if health.reachable && health.version != "" {
		ver, commit = health.version, health.commit
	}
	fmt.Printf("opendray  %s (%s)\n", ver, shortCommit(commit))
	fmt.Println(strings.Repeat("─", 42))

	worst := stateOK
	for _, r := range rows {
		fmt.Printf("%-10s %s %s\n", r.label, r.state.icon(), r.text)
		if r.state == stateWarn && worst < stateWarn {
			worst = stateWarn
		}
		if r.state == stateFail {
			worst = stateFail
		}
	}

	for _, h := range statusHints(proc, health, cfgPath, home) {
		fmt.Println()
		fmt.Println(h)
	}

	return worst.exitCode()
}

// buildStatusRows is the pure heart of the dashboard: given the gathered
// facts it decides each row's severity and text. Kept side-effect-free so
// the severity logic is unit-testable without launchd/HTTP.
func buildStatusRows(proc procInfo, health healthInfo, listen, cfgPath, home string) []statusRow {
	var rows []statusRow

	// Process.
	switch {
	case !proc.found:
		rows = append(rows, statusRow{"Process", stateFail, "not loaded"})
	case proc.running:
		rows = append(rows, statusRow{"Process", stateOK, fmt.Sprintf("running   pid %d", proc.pid)})
	default:
		rows = append(rows, statusRow{"Process", stateFail, "stopped"})
	}

	// Health (the real liveness signal).
	switch {
	case !health.reachable:
		rows = append(rows, statusRow{"Health", stateFail, "unreachable (no HTTP response)"})
	case health.status == "ok":
		rows = append(rows, statusRow{"Health", stateOK, "ok        up " + humanDuration(health.uptime)})
	default:
		label := health.status
		if label == "" {
			label = "degraded"
		}
		rows = append(rows, statusRow{"Health", stateWarn, label + "   up " + humanDuration(health.uptime)})
	}

	// Database.
	switch {
	case !health.reachable:
		rows = append(rows, statusRow{"Database", stateUnknown, "— (gateway unreachable)"})
	case health.dbOK:
		rows = append(rows, statusRow{"Database", stateOK, "reachable"})
	default:
		rows = append(rows, statusRow{"Database", stateFail, "unreachable"})
	}

	// Listen.
	switch {
	case health.reachable:
		rows = append(rows, statusRow{"Listen", stateOK, listen})
	case proc.running:
		rows = append(rows, statusRow{"Listen", stateWarn, listen + " (not answering)"})
	default:
		rows = append(rows, statusRow{"Listen", stateUnknown, listen})
	}

	// Restarts — only meaningful once the unit is loaded.
	if proc.found {
		text := strconv.Itoa(proc.restarts)
		st := stateOK
		if proc.lastExit != 0 {
			st = stateWarn
			text += fmt.Sprintf("  (last exit code %d)", proc.lastExit)
		}
		rows = append(rows, statusRow{"Restarts", st, text})
	}

	// Config.
	switch {
	case cfgPath == "":
		rows = append(rows, statusRow{"Config", stateWarn, "(none found)"})
	case isTCCProtectedPath(home, cfgPath):
		rows = append(rows, statusRow{"Config", stateWarn, tildeAbbrev(home, cfgPath) + " (privacy-gated folder)"})
	default:
		rows = append(rows, statusRow{"Config", stateOK, tildeAbbrev(home, cfgPath)})
	}

	return rows
}

// statusHints returns actionable follow-up lines for anything not green.
func statusHints(proc procInfo, health healthInfo, cfgPath, home string) []string {
	var hints []string
	errLog := "~/.opendray/logs/opendray.err"

	if !proc.running || !health.reachable {
		hints = append(hints, "Hint: service not fully up — see `opendray status --raw` for launchd/systemd\n"+
			"      detail, and check "+errLog)
	} else if proc.lastExit != 0 {
		hints = append(hints, fmt.Sprintf("Hint: last exit was non-zero (code %d) — the service recovered, but\n"+
			"      check %s if restarts keep climbing.", proc.lastExit, errLog))
	}

	if cfgPath != "" && isTCCProtectedPath(home, cfgPath) {
		hints = append(hints, "Hint: config sits in a macOS privacy-gated folder — run `opendray doctor`\n"+
			"      or move it under ~/.opendray/ so startup never blocks on a prompt.")
	}
	return hints
}

// ── gathering (side-effecting) ─────────────────────────────────────

func gatherProc(system bool) procInfo {
	switch runtime.GOOS {
	case "darwin":
		out, _ := exec.Command("launchctl", "print", launchdTarget(system)).CombinedOutput()
		return parseLaunchctlPrint(string(out))
	case "linux":
		out, _ := exec.Command("systemctl", "show", systemdUnitName,
			"--property=LoadState,ActiveState,SubState,MainPID,NRestarts,ExecMainStatus").CombinedOutput()
		return parseSystemctlShow(string(out))
	default:
		return procInfo{}
	}
}

func probeHealth(listen string) healthInfo {
	url := normalizeHealthURL(listen)
	if url == "" {
		return healthInfo{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return healthInfo{}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return healthInfo{}
	}
	defer resp.Body.Close()

	// The gateway answers /health with 200 (ok) or 503 (degraded). Anything
	// else means we hit something that isn't our health endpoint (a 404, a
	// stray proxy) — not a healthy gateway.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return healthInfo{}
	}

	var body struct {
		Status    string `json:"status"`
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		UptimeSec int64  `json:"uptime_s"`
		DBOK      bool   `json:"db_ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return healthInfo{}
	}
	return healthInfo{
		reachable: true,
		status:    body.Status,
		version:   body.Version,
		commit:    body.Commit,
		uptime:    time.Duration(body.UptimeSec) * time.Second,
		dbOK:      body.DBOK,
	}
}

func launchdTarget(system bool) string {
	if system {
		return "system/" + launchdLabel
	}
	return fmt.Sprintf("gui/%d/%s", os.Getuid(), launchdLabel)
}

// ── pure helpers (unit-tested) ─────────────────────────────────────

// parseLaunchctlPrint distills a `launchctl print <target>` dump into a
// procInfo. The dump is `key = value` lines at varying indentation; we only
// care about a handful of top-level fields.
func parseLaunchctlPrint(out string) procInfo {
	var p procInfo
	sawState := false
	for _, ln := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(ln), " = ")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "state":
			// The dump repeats `state = …` for nested coalitions; only the
			// first (top-level) line is the service's own state.
			if !sawState {
				sawState = true
				p.running = strings.HasPrefix(v, "running")
			}
		case "pid":
			p.pid, _ = strconv.Atoi(v)
		case "runs":
			p.restarts, _ = strconv.Atoi(v)
		case "last exit code":
			p.lastExit, _ = strconv.Atoi(v)
		}
	}
	p.found = sawState || p.pid > 0
	return p
}

// parseSystemctlShow distills `systemctl show <unit> --property=...`
// key=value output into a procInfo.
func parseSystemctlShow(out string) procInfo {
	m := map[string]string{}
	for _, ln := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(ln), "="); ok {
			m[k] = v
		}
	}
	p := procInfo{
		found:   m["LoadState"] == "loaded",
		running: m["ActiveState"] == "active" && m["SubState"] == "running",
	}
	p.pid, _ = strconv.Atoi(m["MainPID"])
	p.restarts, _ = strconv.Atoi(m["NRestarts"])
	p.lastExit, _ = strconv.Atoi(m["ExecMainStatus"])
	return p
}

// normalizeHealthURL turns a listen address into the /health URL to probe.
// A wildcard/empty host is dialled on the loopback interface.
func normalizeHealthURL(listen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return ""
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/api/v1/health"
}

// humanDuration renders a duration the way systemctl/uptime do: the two most
// significant units, e.g. "4h21m", "3d2h", "45s".
func humanDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int64(d.Seconds())
	days := sec / 86400
	hrs := (sec % 86400) / 3600
	mins := (sec % 3600) / 60
	secs := sec % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd%dh", days, hrs)
	case hrs > 0:
		return fmt.Sprintf("%dh%dm", hrs, mins)
	case mins > 0:
		return fmt.Sprintf("%dm%ds", mins, secs)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

// shortCommit trims a git SHA to 7 chars for display.
func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	if c == "" {
		return "unknown"
	}
	return c
}

// tildeAbbrev replaces a leading home dir with "~" for compact display.
func tildeAbbrev(home, p string) string {
	if home != "" && (p == home || strings.HasPrefix(p, home+string(filepath.Separator))) {
		return "~" + p[len(home):]
	}
	return p
}
