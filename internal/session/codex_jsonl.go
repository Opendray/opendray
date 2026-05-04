package session

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// codex_jsonl.go: read the OpenAI Codex CLI's per-session rollout
// transcripts and extract user prompts, scoped to a project (cwd).
//
// Codex stores one .jsonl file per session at:
//
//   ~/.codex/sessions/YYYY/MM/DD/rollout-<iso>-<session-id>.jsonl
//
// Unlike Claude's per-cwd directory layout, codex pools sessions
// chronologically. We map files to a project by reading the first
// line of each .jsonl — `session_meta` carries `payload.cwd` —
// and keeping only the ones whose cwd matches.
//
// User prompts arrive as:
//
//   {"type":"response_item",
//    "payload":{"type":"message","role":"user",
//      "content":[{"type":"input_text","text":"..."}]}}
//
// The very first user message in every session is a synthetic
// bootstrap (AGENTS.md instructions + <environment_context> block)
// that we skip — it isn't something the operator typed.

type codexSessionMeta struct {
	Type    string `json:"type"`
	Payload struct {
		Cwd string `json:"cwd"`
	} `json:"payload"`
}

type codexLine struct {
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexResponseItem struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// CodexHistoryConfig points CodexInputHistory at a specific
// rollouts directory. Empty SessionsRoot falls back to
// ~/.codex/sessions (the upstream Codex default).
type CodexHistoryConfig struct {
	SessionsRoot string
}

// CodexInputHistory returns up to `limit` user prompts from every
// codex rollout JSONL whose session_meta.cwd matches `cwd`.
// Newest-first, bootstrap message stripped.
func CodexInputHistory(cfg CodexHistoryConfig, cwd string, limit int) []ProjectInput {
	if limit <= 0 {
		limit = 200
	}
	root := cfg.SessionsRoot
	if root == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return nil
		}
		root = filepath.Join(home, ".codex", "sessions")
	}
	files := collectCodexRollouts(root, cwd)
	var out []ProjectInput
	for _, f := range files {
		out = append(out, extractCodexUserInputs(f.path, f.sessionID)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.After(out[j].Ts) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

type codexRolloutFile struct {
	path      string
	sessionID string // derived from filename: rollout-<iso>-<id>.jsonl
}

// collectCodexRollouts walks the sessions tree and returns rollout
// files whose first line declares a matching cwd.
func collectCodexRollouts(root, cwd string) []codexRolloutFile {
	var files []codexRolloutFile
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		if !codexRolloutMatchesCwd(path, cwd) {
			return nil
		}
		files = append(files, codexRolloutFile{
			path:      path,
			sessionID: codexSessionIDFromName(d.Name()),
		})
		return nil
	})
	return files
}

// codexSessionIDFromName parses "rollout-2026-05-03T23-00-35-019deded-...jsonl"
// → the trailing UUIDv7 (last 5 dash-separated components).
func codexSessionIDFromName(name string) string {
	base := strings.TrimSuffix(name, ".jsonl")
	parts := strings.Split(base, "-")
	if len(parts) < 5 {
		return base
	}
	// UUIDv7 is 5 hex segments at the end.
	return strings.Join(parts[len(parts)-5:], "-")
}

// codexRolloutMatchesCwd reads only the first line of `path` and
// reports whether session_meta.payload.cwd equals `cwd`.
func codexRolloutMatchesCwd(path, cwd string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		return false
	}
	var meta codexLine
	if err := json.Unmarshal(scanner.Bytes(), &meta); err != nil {
		return false
	}
	if meta.Type != "session_meta" || len(meta.Payload) == 0 {
		return false
	}
	var sm codexSessionMeta
	if err := json.Unmarshal(meta.Payload, &sm.Payload); err != nil {
		// Fall back to decoding the whole envelope.
		if err := json.Unmarshal(scanner.Bytes(), &sm); err != nil {
			return false
		}
	}
	if sm.Payload.Cwd == "" {
		// session_meta nests cwd under payload; older versions might
		// put it at top level. Try the alternate shape.
		var alt struct {
			Payload codexSessionMeta `json:"payload"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &alt); err == nil {
			sm.Payload.Cwd = alt.Payload.Payload.Cwd
		}
	}
	return sm.Payload.Cwd == cwd
}

// extractCodexUserInputs pulls every `response_item.role=user` text
// from the rollout file, skipping the synthetic bootstrap message
// that codex injects at the start of every session.
func extractCodexUserInputs(path, sessionID string) []ProjectInput {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var out []ProjectInput
	for scanner.Scan() {
		var line codexLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "response_item" || len(line.Payload) == 0 {
			continue
		}
		var item codexResponseItem
		if err := json.Unmarshal(line.Payload, &item); err != nil {
			continue
		}
		if item.Type != "message" || item.Role != "user" {
			continue
		}
		text := codexUserText(item.Content)
		if text == "" || isCodexBootstrap(text) {
			continue
		}
		out = append(out, ProjectInput{
			Ts:        line.Timestamp,
			Text:      text,
			SessionID: sessionID,
		})
	}
	return out
}

func codexUserText(blocks []struct {
	Type string `json:"type"`
	Text string `json:"text"`
}) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "input_text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// isCodexBootstrap detects the synthetic AGENTS.md +
// environment_context message codex prepends to every session.
// Real prompts won't carry these markers, so a substring check
// is safe.
func isCodexBootstrap(text string) bool {
	if strings.Contains(text, "<environment_context>") {
		return true
	}
	if strings.Contains(text, "AGENTS.md instructions for") {
		return true
	}
	return false
}
