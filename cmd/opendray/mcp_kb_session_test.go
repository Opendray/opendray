package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// blueprintStub serves a global knowledge blueprint so tools/list can be
// exercised without a live gateway.
func blueprintStub(t *testing.T, sections []kbSection) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/project-docs/blueprint") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"sections": sections})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func listTools(t *testing.T, cfg memMCPConfig) (map[string]map[string]any, string) {
	t.Helper()
	var buf bytes.Buffer
	s := &memMCPServer{cfg: cfg, out: &buf, outMu: &sync.Mutex{}, client: &http.Client{}}
	s.handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))

	var resp struct {
		Result struct {
			Tools []map[string]any `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal tools/list: %v (raw %s)", err, buf.String())
	}
	byName := map[string]map[string]any{}
	for _, tl := range resp.Result.Tools {
		name, _ := tl["name"].(string)
		byName[name] = tl
	}
	return byName, buf.String()
}

// The writable-page set comes from blueprint metadata alone — no slug is
// special-cased, so an operator's own kb_* page qualifies and the classic
// pages do not.
func TestSessionWritableKBPages_MetadataDriven(t *testing.T) {
	srv := blueprintStub(t, []kbSection{
		{Slug: "kb_conventions", Title: "Conventions", Maintainer: "ai", Nature: "foundational", Pinned: true},
		{Slug: "kb_lessons", Title: "Lessons", Maintainer: "ai", Nature: "emergent", Pinned: true},
		{Slug: "kb_integrations", Title: "Integrations", Maintainer: "human", Nature: "emergent", Pinned: true},
		{Slug: "kb_projects", Title: "Local Projects", Maintainer: "session", Nature: "emergent",
			Description: "Per-project records.", WritePolicy: "direct"},
		{Slug: "kb_a", Title: "Anything", Maintainer: "session", Nature: "emergent"},
		// A page mislabelled session but marked foundational stays out.
		{Slug: "kb_rules", Title: "Rules", Maintainer: "session", Nature: "foundational"},
		// Non-kb slugs under the global blueprint are not knowledge pages.
		{Slug: "overview", Title: "Overview", Maintainer: "session"},
	})

	s := &memMCPServer{
		cfg:    memMCPConfig{baseURL: srv.URL, apiKey: "k"},
		out:    &bytes.Buffer{},
		outMu:  &sync.Mutex{},
		client: &http.Client{},
	}
	pages := s.sessionWritableKBPages()

	var got []string
	for _, p := range pages {
		got = append(got, p.Slug)
	}
	want := []string{"kb_projects", "kb_a"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("sessionWritableKBPages() = %v, want %v", got, want)
	}
}

// No session-maintained page → the tool is not offered at all, so an
// install that never created one sees an unchanged tool surface.
func TestToolsList_NoSessionPagesOmitsTool(t *testing.T) {
	srv := blueprintStub(t, []kbSection{
		{Slug: "kb_lessons", Title: "Lessons", Maintainer: "ai", Nature: "emergent", Pinned: true},
	})
	tools, _ := listTools(t, memMCPConfig{baseURL: srv.URL, apiKey: "k"})
	if _, ok := tools["kb_page_set"]; ok {
		t.Error("kb_page_set must not be listed when no page is session-maintained")
	}
	if _, ok := tools["memory_search"]; !ok {
		t.Error("the normal tool surface must be unaffected")
	}
}

// With session-maintained pages the tool appears, constrained to exactly
// those slugs and describing each one from its blueprint metadata.
func TestToolsList_SessionPagesShapeTheTool(t *testing.T) {
	srv := blueprintStub(t, []kbSection{
		{Slug: "kb_lessons", Title: "Lessons", Maintainer: "ai", Nature: "emergent", Pinned: true},
		{Slug: "kb_projects", Title: "Local Projects", Maintainer: "session", Nature: "emergent",
			Description: "Per-project records: databases, deploy targets, domains.", WritePolicy: "direct"},
	})
	tools, raw := listTools(t, memMCPConfig{baseURL: srv.URL, apiKey: "k"})

	def, ok := tools["kb_page_set"]
	if !ok {
		t.Fatalf("kb_page_set missing from tools/list: %s", raw)
	}
	desc, _ := def["description"].(string)
	for _, want := range []string{"kb_projects", "Local Projects", "Per-project records"} {
		if !strings.Contains(desc, want) {
			t.Errorf("description missing %q — agents learn which page to write from it:\n%s", want, desc)
		}
	}
	if strings.Contains(desc, "kb_lessons") {
		t.Error("description must not advertise a page the session cannot write")
	}

	schema, _ := def["inputSchema"].(map[string]any)
	props, _ := schema["properties"].(map[string]any)
	slug, _ := props["slug"].(map[string]any)
	enum, _ := slug["enum"].([]any)
	if len(enum) != 1 || enum[0] != "kb_projects" {
		t.Errorf("slug enum = %v, want exactly [kb_projects]", enum)
	}
}

// The gateway being unreachable must degrade to "no tool", never block or
// break the handshake — tools/list is on the session's startup path.
func TestToolsList_GatewayDownDegradesGracefully(t *testing.T) {
	tools, raw := listTools(t, memMCPConfig{baseURL: "http://127.0.0.1:1", apiKey: "k"})
	if _, ok := tools["kb_page_set"]; ok {
		t.Error("kb_page_set must not be listed when the blueprint cannot be read")
	}
	if _, ok := tools["memory_search"]; !ok {
		t.Errorf("handshake must still return the normal tools: %s", raw)
	}
}

// Read-only sessions (Round Table members) never get a write tool.
func TestToolsList_ReadOnlyOmitsKBPageSet(t *testing.T) {
	srv := blueprintStub(t, []kbSection{
		{Slug: "kb_projects", Title: "Local Projects", Maintainer: "session", Nature: "emergent"},
	})
	tools, _ := listTools(t, memMCPConfig{baseURL: srv.URL, apiKey: "k", readOnly: true})
	if _, ok := tools["kb_page_set"]; ok {
		t.Error("read-only session must not list kb_page_set")
	}
}

// Calling by name is refused for a read-only session even though the tool
// was never listed — a read-only member runs with CLI permissions open.
func TestDispatch_ReadOnlyRefusesKBPageSet(t *testing.T) {
	s := &memMCPServer{
		cfg:    memMCPConfig{readOnly: true},
		out:    &bytes.Buffer{},
		outMu:  &sync.Mutex{},
		client: &http.Client{},
	}
	_, err, known := s.dispatchTool("kb_page_set", json.RawMessage(`{"slug":"kb_projects","content":"x"}`))
	if !known {
		t.Fatal("kb_page_set should be a known tool")
	}
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Errorf("want read-only refusal, got %v", err)
	}
}
