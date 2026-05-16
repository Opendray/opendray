package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// gemini_jsonl.go: read the Google Gemini CLI's per-project log
// file and extract user prompts.
//
// Gemini stores per-project state under ~/.gemini/tmp/<dir>/, where
// <dir> can be either:
//
//   - a "short name" (e.g. "pdfstream") — the value the gemini CLI
//     assigns in ~/.gemini/projects.json for that cwd
//   - the lowercase hex SHA-256 of the cwd — used as a fallback
//     when no short name has been registered
//
// Each project dir holds a .project_root file containing the
// canonical cwd plus a logs.json (a JSON array of
// {sessionId, messageId, type, message, timestamp}). User prompts
// are records whose type == "user" (slash commands like /model count
// as user input — the operator typed them).
//
// Resolution order:
//   1. ~/.gemini/projects.json → short name → tmp/<name>/logs.json
//   2. tmp/<sha256(cwd)>/logs.json
//   3. Scan all tmp/*/.project_root files and match cwd

type geminiLogEntry struct {
	SessionID string    `json:"sessionId"`
	MessageID int       `json:"messageId"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// GeminiHistoryConfig points GeminiInputHistory at custom paths.
// Both fields optional; empty values use the upstream Gemini CLI
// defaults under ~/.gemini.
type GeminiHistoryConfig struct {
	TmpRoot      string // default: ~/.gemini/tmp
	ProjectsFile string // default: ~/.gemini/projects.json
}

// GeminiInputHistory returns up to `limit` user prompts from
// Gemini's per-project logs.json. Newest first.
func GeminiInputHistory(cfg GeminiHistoryConfig, cwd string, limit int) []ProjectInput {
	if limit <= 0 {
		limit = 200
	}
	tmpRoot := cfg.TmpRoot
	projectsFile := cfg.ProjectsFile
	if tmpRoot == "" || projectsFile == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return nil
		}
		if tmpRoot == "" {
			tmpRoot = filepath.Join(home, ".gemini", "tmp")
		}
		if projectsFile == "" {
			projectsFile = filepath.Join(home, ".gemini", "projects.json")
		}
	}
	dir := findGeminiProjectDir(projectsFile, tmpRoot, cwd)
	if dir == "" {
		return nil
	}
	body, err := os.ReadFile(filepath.Join(dir, "logs.json"))
	if err != nil {
		return nil
	}
	var raw []geminiLogEntry
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}
	out := make([]ProjectInput, 0, len(raw))
	for _, e := range raw {
		if e.Type != "user" || e.Message == "" {
			continue
		}
		out = append(out, ProjectInput{
			Ts:        e.Timestamp,
			Text:      e.Message,
			SessionID: e.SessionID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.After(out[j].Ts) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// geminiRecentResponse returns Gemini's most-recent "model" reply
// for `cwd`, suitable for embedding in a chat-side idle-notification
// snippet. Mirrors claudeRecentResponse for the Gemini provider so
// the snippet doesn't degrade to the ScreenSnapshot fallback (which
// is bounded by terminal grid size, ~1 screenful).
//
// Schema: ~/.gemini/tmp/<dir>/logs.json is a JSON array of entries
// `{sessionId, messageId, type, message, timestamp}`. We pick the
// model entry with the latest timestamp (the file isn't strictly
// sorted; Gemini appends as messages arrive, but ordering within
// the array is not contractual).
//
// Returns "" when no transcript exists, the file is unreadable, or
// no model reply is present yet. Callers fall through to the
// next snippet source in the priority chain.
func geminiRecentResponse(cwd string) string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	tmpRoot := filepath.Join(home, ".gemini", "tmp")
	projectsFile := filepath.Join(home, ".gemini", "projects.json")
	dir := findGeminiProjectDir(projectsFile, tmpRoot, cwd)
	if dir == "" {
		return ""
	}
	body, err := os.ReadFile(filepath.Join(dir, "logs.json"))
	if err != nil {
		return ""
	}
	var raw []geminiLogEntry
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}
	// Walk all entries and remember the latest model reply. We
	// compare by timestamp rather than trusting file order — a
	// long-running session can have model entries interleaved with
	// user prompts.
	var latest *geminiLogEntry
	for i := range raw {
		e := &raw[i]
		if e.Type != "model" {
			continue
		}
		if strings.TrimSpace(e.Message) == "" {
			continue
		}
		if latest == nil || e.Timestamp.After(latest.Timestamp) {
			latest = e
		}
	}
	if latest == nil {
		return ""
	}
	return strings.TrimSpace(latest.Message)
}

// findGeminiProjectDir resolves cwd to the matching tmp/<dir>/
// folder. Tries each strategy in order; returns "" when none hit.
func findGeminiProjectDir(projectsFile, tmpRoot, cwd string) string {
	// 1. projects.json maps cwd → short name. Cheapest path.
	if name := geminiShortName(projectsFile, cwd); name != "" {
		dir := filepath.Join(tmpRoot, name)
		if hasGeminiLogs(dir) {
			return dir
		}
	}
	// 2. Hash-named directory (older gemini versions).
	hash := sha256.Sum256([]byte(cwd))
	dir := filepath.Join(tmpRoot, hex.EncodeToString(hash[:]))
	if hasGeminiLogs(dir) {
		return dir
	}
	// 3. Scan all tmp/*/.project_root for a match. Slow but
	//    handles any other naming scheme gemini might use.
	if found := scanGeminiProjectRoots(tmpRoot, cwd); found != "" {
		return found
	}
	return ""
}

// geminiShortName looks up cwd in projectsFile and returns its
// short name, or "" when the file is missing or has no entry.
func geminiShortName(projectsFile, cwd string) string {
	body, err := os.ReadFile(projectsFile)
	if err != nil {
		return ""
	}
	var doc struct {
		Projects map[string]string `json:"projects"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return ""
	}
	return doc.Projects[cwd]
}

// scanGeminiProjectRoots walks tmpRoot and reads each child's
// .project_root file. Returns the dir whose canonical cwd matches.
func scanGeminiProjectRoots(tmpRoot, cwd string) string {
	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(tmpRoot, e.Name())
		body, err := os.ReadFile(filepath.Join(dir, ".project_root"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(body)) == cwd {
			return dir
		}
	}
	return ""
}

func hasGeminiLogs(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "logs.json"))
	return err == nil
}
