package projectdoc

import (
	"testing"
	"time"
)

func TestParseLogOutcome(t *testing.T) {
	// Mirrors buildJournalBody's real output shape.
	body := "**Session metadata**\n\n" +
		"- id: `ses_abc123`\n" +
		"- provider: Claude\n" +
		"- cwd: `/home/proj`\n" +
		"- started: 2026-06-20T09:18:37Z\n" +
		"- ended: 2026-06-20T09:42:10Z\n" +
		"- duration: 23m33s\n" +
		"- exit_code: 0\n\n"

	state, exit, started, ended := parseLogOutcome("Session abc123 — Claude — ended", body)
	if state != "ended" {
		t.Errorf("state = %q, want ended", state)
	}
	if exit == nil || *exit != 0 {
		t.Errorf("exit = %v, want 0", exit)
	}
	if started == nil || !started.Equal(time.Date(2026, 6, 20, 9, 18, 37, 0, time.UTC)) {
		t.Errorf("started = %v", started)
	}
	if ended == nil || !ended.Equal(time.Date(2026, 6, 20, 9, 42, 10, 0, time.UTC)) {
		t.Errorf("ended = %v", ended)
	}
}

func TestParseLogOutcome_StoppedNoExit(t *testing.T) {
	// An operator-stopped session: state from title, no exit_code line, no end.
	body := "- started: 2026-06-19T08:00:00Z\n"
	state, exit, started, ended := parseLogOutcome("Session zz — Codex — stopped", body)
	if state != "stopped" {
		t.Errorf("state = %q, want stopped", state)
	}
	if exit != nil {
		t.Errorf("exit = %v, want nil", exit)
	}
	if started == nil {
		t.Error("started should parse")
	}
	if ended != nil {
		t.Errorf("ended = %v, want nil", ended)
	}
}

func TestParseLogOutcome_NonZeroExit(t *testing.T) {
	_, exit, _, _ := parseLogOutcome("Session q — Claude — ended", "- exit_code: 1\n")
	if exit == nil || *exit != 1 {
		t.Errorf("exit = %v, want 1", exit)
	}
}

func TestParseLogOutcome_Unparseable(t *testing.T) {
	// Title without the " — state" suffix and a body with no metadata lines.
	state, exit, started, ended := parseLogOutcome("freeform note", "just some text\n")
	if state != "" || exit != nil || started != nil || ended != nil {
		t.Errorf("expected all-zero, got state=%q exit=%v started=%v ended=%v", state, exit, started, ended)
	}
}
