package mcp

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOneNormalizesRemoteCommandURL(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "remote")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte(`{
  "name": "remote",
  "transport": "sse",
  "command": "https://example.com/sse",
  "enabled": true
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := loadOne(root, "remote")
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://example.com/sse" {
		t.Fatalf("URL=%q, want remote endpoint", got.URL)
	}
	if got.Command != "" {
		t.Fatalf("Command=%q, want cleared after URL migration", got.Command)
	}
}

func TestMarshalNormalizesRemoteCommandURL(t *testing.T) {
	body, err := Marshal(Server{
		ID:        "remote",
		Name:      "remote",
		Transport: "http",
		Command:   "https://example.com/mcp",
		Enabled:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var got Server
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://example.com/mcp" {
		t.Fatalf("URL=%q, want remote endpoint", got.URL)
	}
	if got.Command != "" {
		t.Fatalf("Command=%q, want omitted after URL migration", got.Command)
	}
}

func TestPrepareServerForWriteRejectsRemoteCommand(t *testing.T) {
	srv := Server{
		ID:        "remote",
		Name:      "remote",
		Transport: "http",
		Command:   "node server.js",
		Enabled:   true,
	}
	if err := prepareServerForWrite(&srv); err == nil {
		t.Fatal("expected remote server without URL to be rejected")
	}
}

// Two servers with different ids but the same display name break Codex
// (duplicate TOML table) and silently drop in Claude. The loader must be
// able to report that a name is already taken by a *different* id so the
// create/update handlers can reject it before it ever reaches a session.
func TestLoader_NameTakenByAnother(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "notion")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"),
		[]byte(`{"name":"Notion API","command":"npx","enabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	l := NewLoader(root)

	if other, taken := l.NameTaken("Notion API", "motion"); !taken || other != "notion" {
		t.Errorf("a new id reusing an existing name must be reported taken; got other=%q taken=%v", other, taken)
	}
	if _, taken := l.NameTaken("Notion API", "notion"); taken {
		t.Error("the same id keeping its own name must not report a collision")
	}
	if _, taken := l.NameTaken("Something Else", "motion"); taken {
		t.Error("an unused name must be free")
	}
}

// End-to-end guard: creating a second server whose display name matches an
// existing one (the motion/notion bug) must be refused with 409, not
// written to disk — otherwise Codex sessions die on a duplicate TOML key.
func TestCreate_RejectsDuplicateDisplayName(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "notion")
	if err := os.MkdirAll(existing, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "mcp.json"),
		[]byte(`{"name":"Notion API","command":"npx","enabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	h := NewHandlers(NewLoader(root), "", nil)
	r := chi.NewRouter()
	h.Mount(r)

	body := `{"id":"motion","server":{"name":"Notion API","command":"npx","enabled":true}}`
	req := httptest.NewRequest(http.MethodPost, "/mcps/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409 Conflict on duplicate display name, got %d (%s)", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "motion", "mcp.json")); err == nil {
		t.Error("colliding server must not be written to disk")
	}
}
