package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
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
