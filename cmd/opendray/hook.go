// Subcommand `opendray hook` — Claude Code hook integration that
// auto-writes journal entries when agents edit files / run bash /
// otherwise mutate state. Closes the "agent activity isn't being
// recorded" gap from M14 user feedback.
//
// Operator wiring (one-time, in ~/.claude/settings.json):
//
//	{
//	  "env": {
//	    "OPENDRAY_BASE_URL": "http://127.0.0.1:8770",
//	    "OPENDRAY_API_KEY":  "<integration token>"
//	  },
//	  "hooks": {
//	    "PostToolUse": [{
//	      "matcher": "Edit|Write|MultiEdit|Bash",
//	      "hooks": [{ "type": "command", "command": "opendray hook tool-use" }]
//	    }]
//	  }
//	}
//
// After that, every Edit / Write / Bash tool call inside any
// Claude Code session in this cwd appends one session_logs row.
// Operators see the running journal of "what the agent just did"
// directly in the Memory tab.
//
// Subcommands:
//
//	tool-use      — PostToolUse hook. Writes a one-line journal
//	                entry per tool call.
//	session-end   — Stop / SessionEnd hook. Writes a single
//	                session_summary entry summarising the session.
//
// stdin shape matches Claude Code's hook payload spec:
//
//	{
//	  "session_id":       "uuid",
//	  "transcript_path":  "/Users/.../sessions/xxx.jsonl",
//	  "cwd":              "/Users/me/projects/foo",
//	  "hook_event_name":  "PostToolUse",
//	  "tool_name":        "Edit",
//	  "tool_input":       { "file_path": "...", ... },
//	  "tool_response":    { ... }
//	}
//
// Required env vars:
//
//	OPENDRAY_BASE_URL  e.g. http://127.0.0.1:8770 (no trailing slash)
//	OPENDRAY_API_KEY   admin or integration bearer
//
// Optional env vars:
//
//	OPENDRAY_HOOK_DEBUG=1  echo POST body to stderr for diagnosis
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// runHook is the `opendray hook <subcommand>` entry point.
func runHook(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "opendray hook: expected subcommand (tool-use | session-end)")
		return 2
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "tool-use":
		return runHookToolUse(rest)
	case "session-end":
		return runHookSessionEnd(rest)
	default:
		fmt.Fprintf(os.Stderr, "opendray hook: unknown subcommand %q\n", sub)
		return 2
	}
}

// hookPayload is the union shape of every Claude Code hook event.
// Fields not relevant to the current event are simply absent.
type hookPayload struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	Cwd            string          `json:"cwd"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolResponse   json.RawMessage `json:"tool_response"`
}

// readPayload parses the Claude Code hook payload from stdin. Hooks
// fire fast and silent on a healthy system — anything that errors
// here logs to stderr but exits 0 so a misconfigured opendray host
// doesn't fail the agent's tool call.
func readPayload() (hookPayload, error) {
	var p hookPayload
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 4<<20))
	if err != nil {
		return p, fmt.Errorf("read stdin: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return p, errors.New("empty stdin")
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("parse json: %w", err)
	}
	return p, nil
}

// runHookToolUse — PostToolUse hook handler.
//
// Behaviour by tool:
//
//	Edit, Write, MultiEdit  → "Edit: <file basename>" with file path
//	                          in content
//	Bash                    → "Bash: <first 60 chars of command>"
//	other                   → "<tool_name> tool call" (generic)
//
// Skips when CWD is missing (we can't anchor the journal entry to a
// project without it).
func runHookToolUse(_ []string) int {
	p, err := readPayload()
	if err != nil {
		warn("readPayload", err)
		return 0
	}
	if p.Cwd == "" {
		warn("missing cwd", nil)
		return 0
	}
	title, content := summariseToolUse(p)
	if title == "" {
		// Some tool we don't care to log (e.g. Read, Grep).
		return 0
	}
	if err := postJournal(p.Cwd, p.SessionID, title, content, "manual"); err != nil {
		warn("postJournal", err)
	}
	return 0
}

// summariseToolUse picks a per-tool title + content. Returns empty
// title when the tool isn't interesting enough to log.
func summariseToolUse(p hookPayload) (string, string) {
	switch p.ToolName {
	case "Edit", "MultiEdit":
		var in struct {
			FilePath string `json:"file_path"`
		}
		_ = json.Unmarshal(p.ToolInput, &in)
		base := baseName(in.FilePath)
		if base == "" {
			base = "(unknown)"
		}
		return "Edit: " + base, "file: " + in.FilePath
	case "Write":
		var in struct {
			FilePath string `json:"file_path"`
		}
		_ = json.Unmarshal(p.ToolInput, &in)
		base := baseName(in.FilePath)
		if base == "" {
			base = "(unknown)"
		}
		return "Write: " + base, "file: " + in.FilePath
	case "Bash":
		var in struct {
			Command     string `json:"command"`
			Description string `json:"description"`
		}
		_ = json.Unmarshal(p.ToolInput, &in)
		cmd := strings.TrimSpace(in.Command)
		// Pull first non-empty line so multi-line scripts don't
		// dominate the journal title.
		if i := strings.IndexByte(cmd, '\n'); i >= 0 {
			cmd = cmd[:i]
		}
		short := cmd
		if len(short) > 60 {
			short = short[:60] + "…"
		}
		title := "Bash: " + short
		var body strings.Builder
		if in.Description != "" {
			body.WriteString(in.Description)
			body.WriteString("\n\n")
		}
		body.WriteString("```bash\n")
		body.WriteString(cmd)
		body.WriteString("\n```")
		return title, body.String()
	default:
		// Read / Grep / Glob / TodoWrite / etc. — these are
		// reconnaissance, not mutation. Don't pollute the journal.
		return "", ""
	}
}

// runHookSessionEnd — Stop / SessionEnd hook handler. Writes a
// single session_summary entry. For now the summary is just a
// pointer to the transcript path — a follow-up could read the
// transcript and run a summarizer pass.
func runHookSessionEnd(_ []string) int {
	p, err := readPayload()
	if err != nil {
		warn("readPayload", err)
		return 0
	}
	if p.Cwd == "" {
		warn("missing cwd", nil)
		return 0
	}
	title := "Claude Code session ended"
	if p.SessionID != "" {
		shortID := p.SessionID
		if len(shortID) > 8 {
			shortID = shortID[len(shortID)-8:]
		}
		title = fmt.Sprintf("Claude Code session %s ended", shortID)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Session id: `%s`\n", p.SessionID)
	if p.TranscriptPath != "" {
		fmt.Fprintf(&b, "Transcript: `%s`\n", p.TranscriptPath)
	}
	b.WriteString("\n_Auto-generated by opendray hook session-end._\n")
	if err := postJournal(p.Cwd, p.SessionID, title, b.String(), "session_summary"); err != nil {
		warn("postJournal", err)
	}
	return 0
}

// postJournal sends one session_logs entry to the opendray gateway.
// updated_by is hardcoded to "agent" so the operator UI can tell
// hook-written entries apart from operator-typed ones. kind defaults
// to "manual" for tool-use events and "session_summary" for
// session-end events.
func postJournal(cwd, sessionID, title, content, kind string) error {
	base := strings.TrimRight(os.Getenv("OPENDRAY_BASE_URL"), "/")
	if base == "" {
		return errors.New("OPENDRAY_BASE_URL not set")
	}
	key := os.Getenv("OPENDRAY_API_KEY")
	if key == "" {
		return errors.New("OPENDRAY_API_KEY not set")
	}
	body := map[string]any{
		"cwd":        cwd,
		"kind":       kind,
		"title":      title,
		"content":    content,
		"updated_by": "agent",
	}
	if sessionID != "" {
		// session_logs.session_id is FK to sessions(id); the
		// Claude Code session id won't match opendray's session
		// id space, so we deliberately do NOT send it. The
		// transcript_path in the content is enough to trace.
		_ = sessionID
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if os.Getenv("OPENDRAY_HOOK_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "opendray-hook: POST %s/api/v1/session-logs %s\n", base, string(raw))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/session-logs", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("HTTP %d: %s", res.StatusCode, string(b))
	}
	return nil
}

func warn(prefix string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "opendray-hook: %s: %v\n", prefix, err)
	} else {
		fmt.Fprintf(os.Stderr, "opendray-hook: %s\n", prefix)
	}
}

func baseName(p string) string {
	if p == "" {
		return ""
	}
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}
