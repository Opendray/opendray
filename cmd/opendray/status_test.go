package main

import (
	"testing"
	"time"
)

func TestParseLaunchctlPrint(t *testing.T) {
	// Real dumps repeat `state = active` for nested coalitions after the
	// top-level `state = running`; the parser must not let those override it.
	running := `gui/501/com.opendray.opendray = {
	active count = 1
	state = running
	program = /Users/linivek/.opendray/bin/opendray
	pid = 46781
	runs = 201
	last exit code = 1
	resource coalition = {
		state = active
	}
	jetsam coalition = {
		state = active
	}
}`
	got := parseLaunchctlPrint(running)
	if !got.found || !got.running {
		t.Fatalf("expected found+running, got %+v", got)
	}
	if got.pid != 46781 || got.restarts != 201 || got.lastExit != 1 {
		t.Fatalf("field mismatch: %+v", got)
	}

	// Not-loaded: launchctl prints an error, no fields.
	empty := parseLaunchctlPrint("Could not find service \"com.opendray.opendray\"")
	if empty.found || empty.running {
		t.Fatalf("expected not-found, got %+v", empty)
	}

	// Loaded but stopped: state present, no pid line.
	stopped := parseLaunchctlPrint("state = not running\nruns = 3\nlast exit code = 0")
	if !stopped.found {
		t.Fatalf("expected found (state seen), got %+v", stopped)
	}
	if stopped.running {
		t.Fatalf("expected stopped, got %+v", stopped)
	}
}

func TestParseSystemctlShow(t *testing.T) {
	out := "LoadState=loaded\nActiveState=active\nSubState=running\nMainPID=1234\nNRestarts=2\nExecMainStatus=0"
	got := parseSystemctlShow(out)
	if !got.found || !got.running || got.pid != 1234 || got.restarts != 2 || got.lastExit != 0 {
		t.Fatalf("mismatch: %+v", got)
	}

	dead := parseSystemctlShow("LoadState=loaded\nActiveState=failed\nSubState=failed\nMainPID=0\nNRestarts=5\nExecMainStatus=1")
	if !dead.found || dead.running || dead.lastExit != 1 {
		t.Fatalf("expected loaded+not-running+exit1, got %+v", dead)
	}

	notLoaded := parseSystemctlShow("LoadState=not-found\nActiveState=inactive")
	if notLoaded.found {
		t.Fatalf("expected not-found, got %+v", notLoaded)
	}
}

func TestNormalizeHealthURL(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:8770": "http://127.0.0.1:8770/api/v1/health",
		"0.0.0.0:8770":   "http://127.0.0.1:8770/api/v1/health",
		":8770":          "http://127.0.0.1:8770/api/v1/health",
		"192.168.3.1:80": "http://192.168.3.1:80/api/v1/health",
		"":               "",
		"garbage":        "",
	}
	for in, want := range cases {
		if got := normalizeHealthURL(in); got != want {
			t.Errorf("normalizeHealthURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	cases := map[time.Duration]string{
		45 * time.Second:             "45s",
		90 * time.Second:             "1m30s",
		4*time.Hour + 21*time.Minute: "4h21m",
		3*24*time.Hour + 2*time.Hour: "3d2h",
		-5 * time.Second:             "0s",
	}
	for in, want := range cases {
		if got := humanDuration(in); got != want {
			t.Errorf("humanDuration(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildStatusRowsSeverity(t *testing.T) {
	home := "/Users/x"

	// Healthy: process running, health ok, db ok.
	healthy := buildStatusRows(
		procInfo{found: true, running: true, pid: 100, restarts: 2, lastExit: 0},
		healthInfo{reachable: true, status: "ok", dbOK: true, uptime: time.Hour},
		"127.0.0.1:8770", "/Users/x/.opendray/config.toml", home,
	)
	for _, r := range healthy {
		if r.state != stateOK {
			t.Errorf("healthy row %q not OK: %+v", r.label, r)
		}
	}

	// Non-zero last exit → Restarts row warns.
	warned := buildStatusRows(
		procInfo{found: true, running: true, pid: 100, restarts: 201, lastExit: 1},
		healthInfo{reachable: true, status: "ok", dbOK: true},
		"127.0.0.1:8770", "/Users/x/.opendray/config.toml", home,
	)
	if got := rowState(warned, "Restarts"); got != stateWarn {
		t.Errorf("Restarts row = %v, want warn", got)
	}

	// Gateway unreachable → Health fails, DB unknown.
	down := buildStatusRows(
		procInfo{found: true, running: true, pid: 100},
		healthInfo{reachable: false},
		"127.0.0.1:8770", "/Users/x/.opendray/config.toml", home,
	)
	if got := rowState(down, "Health"); got != stateFail {
		t.Errorf("Health row = %v, want fail", got)
	}
	if got := rowState(down, "Database"); got != stateUnknown {
		t.Errorf("Database row = %v, want unknown", got)
	}

	// Not loaded → Process fails, no Restarts row.
	off := buildStatusRows(
		procInfo{found: false},
		healthInfo{reachable: false},
		"127.0.0.1:8770", "", home,
	)
	if got := rowState(off, "Process"); got != stateFail {
		t.Errorf("Process row = %v, want fail", got)
	}
	if hasRow(off, "Restarts") {
		t.Error("expected no Restarts row when unit not loaded")
	}
}

func rowState(rows []statusRow, label string) checkState {
	for _, r := range rows {
		if r.label == label {
			return r.state
		}
	}
	return stateUnknown
}

func hasRow(rows []statusRow, label string) bool {
	for _, r := range rows {
		if r.label == label {
			return true
		}
	}
	return false
}
