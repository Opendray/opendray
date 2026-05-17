package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGeminiInputHistory_ReadsHashedDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cwd := "/tmp/proj/Gemini"
	hash := sha256.Sum256([]byte(cwd))
	dir := filepath.Join(tmpHome, ".gemini", "tmp", hex.EncodeToString(hash[:]))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	logs := []map[string]any{
		{
			"sessionId": "sess-1",
			"messageId": 0,
			"type":      "user",
			"message":   "/model",
			"timestamp": "2026-05-04T10:00:00.000Z",
		},
		{
			"sessionId": "sess-1",
			"messageId": 1,
			"type":      "assistant",
			"message":   "Sure, let me help.",
			"timestamp": "2026-05-04T10:00:01.000Z",
		},
		{
			"sessionId": "sess-2",
			"messageId": 0,
			"type":      "user",
			"message":   "explain this code",
			"timestamp": "2026-05-04T11:00:00.000Z",
		},
	}
	body, err := json.Marshal(logs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}

	got := GeminiInputHistory(GeminiHistoryConfig{}, cwd, 10)
	if len(got) != 2 {
		t.Fatalf("want 2 user entries, got %d: %+v", len(got), got)
	}
	if got[0].Text != "explain this code" {
		t.Errorf("newest first failed: got %q", got[0].Text)
	}
	if got[1].Text != "/model" {
		t.Errorf("slash command not preserved: got %q", got[1].Text)
	}
	if got[0].SessionID != "sess-2" {
		t.Errorf("session id wrong: got %q", got[0].SessionID)
	}
}

func TestGeminiInputHistory_ShortNameFromProjectsJSON(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cwd := "/Users/alice/Documents/work/android/PDFStream"
	// projects.json maps cwd → "pdfstream"; logs.json lives under
	// tmp/pdfstream/, NOT under tmp/<sha256(cwd)>/.
	if err := os.MkdirAll(filepath.Join(tmpHome, ".gemini"), 0o755); err != nil {
		t.Fatal(err)
	}
	projDoc := map[string]any{
		"projects": map[string]string{cwd: "pdfstream"},
	}
	body, _ := json.Marshal(projDoc)
	if err := os.WriteFile(filepath.Join(tmpHome, ".gemini", "projects.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmpHome, ".gemini", "tmp", "pdfstream")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	logs := []map[string]any{
		{
			"sessionId": "s1", "messageId": 0, "type": "user",
			"message":   "简单描述这个项目",
			"timestamp": "2026-05-04T03:12:27.663Z",
		},
	}
	logsBody, _ := json.Marshal(logs)
	if err := os.WriteFile(filepath.Join(dir, "logs.json"), logsBody, 0o600); err != nil {
		t.Fatal(err)
	}

	got := GeminiInputHistory(GeminiHistoryConfig{}, cwd, 10)
	if len(got) != 1 || got[0].Text != "简单描述这个项目" {
		t.Errorf("short-name lookup failed: %+v", got)
	}
}

func TestGeminiInputHistory_ProjectRootScanFallback(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cwd := "/Users/alice/some/project"
	// No projects.json, no SHA dir — only a randomly-named tmp dir
	// with a .project_root file matching cwd.
	dir := filepath.Join(tmpHome, ".gemini", "tmp", "random-name-xyz")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".project_root"), []byte(cwd), 0o600); err != nil {
		t.Fatal(err)
	}
	logs := []map[string]any{{
		"sessionId": "s", "messageId": 0, "type": "user",
		"message": "hello", "timestamp": "2026-05-04T10:00:00Z",
	}}
	body, _ := json.Marshal(logs)
	if err := os.WriteFile(filepath.Join(dir, "logs.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}

	got := GeminiInputHistory(GeminiHistoryConfig{}, cwd, 10)
	if len(got) != 1 || got[0].Text != "hello" {
		t.Errorf(".project_root scan fallback failed: %+v", got)
	}
}

func TestGeminiInputHistory_CustomTmpAndProjectsFile(t *testing.T) {
	t.Setenv("HOME", "") // default discovery would yield nothing
	tmp := t.TempDir()
	customTmp := filepath.Join(tmp, "custom-gem-tmp")
	customProjects := filepath.Join(tmp, "custom-gem-projects.json")

	cwd := "/Users/x/CustomProj"
	if err := os.MkdirAll(filepath.Join(customTmp, "alias"), 0o755); err != nil {
		t.Fatal(err)
	}
	projDoc := map[string]any{"projects": map[string]string{cwd: "alias"}}
	body, _ := json.Marshal(projDoc)
	if err := os.WriteFile(customProjects, body, 0o600); err != nil {
		t.Fatal(err)
	}
	logs := []map[string]any{{
		"sessionId": "s", "messageId": 0, "type": "user",
		"message": "from custom gemini paths", "timestamp": "2026-05-04T10:00:00Z",
	}}
	logsBody, _ := json.Marshal(logs)
	if err := os.WriteFile(filepath.Join(customTmp, "alias", "logs.json"), logsBody, 0o600); err != nil {
		t.Fatal(err)
	}

	got := GeminiInputHistory(
		GeminiHistoryConfig{TmpRoot: customTmp, ProjectsFile: customProjects},
		cwd, 10,
	)
	if len(got) != 1 || got[0].Text != "from custom gemini paths" {
		t.Errorf("custom Gemini paths not honoured: %+v", got)
	}
}

func TestGeminiInputHistory_NoLogsFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if got := GeminiInputHistory(GeminiHistoryConfig{}, "/tmp/no/such/project", 10); got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestGeminiInputHistory_LimitTrims(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cwd := "/tmp/proj/L"
	hash := sha256.Sum256([]byte(cwd))
	dir := filepath.Join(tmpHome, ".gemini", "tmp", hex.EncodeToString(hash[:]))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	logs := make([]map[string]any, 0, 30)
	for i := 0; i < 30; i++ {
		logs = append(logs, map[string]any{
			"sessionId": "sess",
			"messageId": i,
			"type":      "user",
			"message":   "p" + twoDigit(i),
			"timestamp": "2026-05-04T10:" + twoDigit(i) + ":00.000Z",
		})
	}
	body, _ := json.Marshal(logs)
	if err := os.WriteFile(filepath.Join(dir, "logs.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}

	got := GeminiInputHistory(GeminiHistoryConfig{}, cwd, 5)
	if len(got) != 5 {
		t.Fatalf("limit not honoured: got %d, want 5", len(got))
	}
	if got[0].Text != "p29" {
		t.Errorf("newest first failed: %q", got[0].Text)
	}
}
