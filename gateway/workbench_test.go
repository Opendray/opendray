package gateway

// T9 — Workbench contributions endpoint — unit tests.
//
// All tests construct a *Server directly by field so they need no DB, hub,
// or full gateway.New call. The handler under test is s.workbenchContributions.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/contributions"
)

// newTestServer builds a minimal Server wired with the given registry.
// contribReg may be nil (to test the 503 path).
func newTestServer(reg *contributions.Registry) *Server {
	return &Server{contribReg: reg}
}

// doGet fires a GET against s.workbenchContributions and returns the recorder.
func doGet(s *Server) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/workbench/contributions", nil)
	rr := httptest.NewRecorder()
	s.workbenchContributions(rr, req)
	return rr
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// fakeContributes returns a ContributesV1 with one entry in each slot.
// menuPath is the key used for the Menus map.
func fakeContributes(menuPath string) plugin.ContributesV1 {
	run := &plugin.CommandRunV1{Kind: "notify", Message: "hello"}
	return plugin.ContributesV1{
		Commands: []plugin.CommandV1{
			{
				ID:    "test.cmd",
				Title: "Test Command",
				Run:   run,
			},
		},
		StatusBar: []plugin.StatusBarItemV1{
			{ID: "test.bar", Text: "Test Bar", Alignment: "right", Priority: 10},
		},
		Keybindings: []plugin.KeybindingV1{
			{Command: "test.cmd", Key: "ctrl+shift+t"},
		},
		Menus: map[string][]plugin.MenuEntryV1{
			menuPath: {
				{Command: "test.cmd", Group: "test@1"},
			},
		},
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────────

// TestWorkbench_ContentType asserts that every response sets the JSON content type.
func TestWorkbench_ContentType(t *testing.T) {
	reg := contributions.NewRegistry()
	s := newTestServer(reg)
	rr := doGet(s)

	want := "application/json; charset=utf-8"
	if got := rr.Header().Get("Content-Type"); got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

// TestWorkbench_EmptyRegistry asserts 200 with all four slot keys as empty arrays/object.
func TestWorkbench_EmptyRegistry(t *testing.T) {
	reg := contributions.NewRegistry()
	s := newTestServer(reg)
	rr := doGet(s)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var flat contributions.FlatContributions
	if err := json.Unmarshal(rr.Body.Bytes(), &flat); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if flat.Commands == nil {
		t.Error("commands must not be null")
	}
	if flat.StatusBar == nil {
		t.Error("statusBar must not be null")
	}
	if flat.Keybindings == nil {
		t.Error("keybindings must not be null")
	}
	if flat.Menus == nil {
		t.Error("menus must not be null")
	}
	if len(flat.Commands) != 0 {
		t.Errorf("commands len = %d, want 0", len(flat.Commands))
	}
	if len(flat.StatusBar) != 0 {
		t.Errorf("statusBar len = %d, want 0", len(flat.StatusBar))
	}
	if len(flat.Keybindings) != 0 {
		t.Errorf("keybindings len = %d, want 0", len(flat.Keybindings))
	}
	if len(flat.Menus) != 0 {
		t.Errorf("menus len = %d, want 0", len(flat.Menus))
	}

	// Also assert that the raw JSON has all four keys (not omitted).
	raw := rr.Body.Bytes()
	for _, key := range []string{`"commands"`, `"statusBar"`, `"keybindings"`, `"menus"`} {
		found := false
		for i := 0; i < len(raw)-len(key)+1; i++ {
			if string(raw[i:i+len(key)]) == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("JSON body missing key %s: %s", key, raw)
		}
	}
}

// TestWorkbench_PopulatedAfterRegistering asserts 200 with correct content after
// registering one plugin.
func TestWorkbench_PopulatedAfterRegistering(t *testing.T) {
	reg := contributions.NewRegistry()
	reg.Set("time-ninja", fakeContributes("appBar/right"))

	s := newTestServer(reg)
	rr := doGet(s)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var flat contributions.FlatContributions
	if err := json.Unmarshal(rr.Body.Bytes(), &flat); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Commands: one entry with pluginName=time-ninja.
	if len(flat.Commands) != 1 {
		t.Fatalf("commands len = %d, want 1", len(flat.Commands))
	}
	if flat.Commands[0].PluginName != "time-ninja" {
		t.Errorf("commands[0].pluginName = %q, want %q", flat.Commands[0].PluginName, "time-ninja")
	}

	// StatusBar: one entry.
	if len(flat.StatusBar) != 1 {
		t.Fatalf("statusBar len = %d, want 1", len(flat.StatusBar))
	}
	if flat.StatusBar[0].PluginName != "time-ninja" {
		t.Errorf("statusBar[0].pluginName = %q, want %q", flat.StatusBar[0].PluginName, "time-ninja")
	}

	// Keybindings: one entry.
	if len(flat.Keybindings) != 1 {
		t.Fatalf("keybindings len = %d, want 1", len(flat.Keybindings))
	}

	// Menus: appBar/right key present with one entry.
	entries, ok := flat.Menus["appBar/right"]
	if !ok {
		t.Fatal("menus[\"appBar/right\"] missing")
	}
	if len(entries) != 1 {
		t.Errorf("menus[\"appBar/right\"] len = %d, want 1", len(entries))
	}
}

// TestWorkbench_StableOrdering sets 3 plugins (unsorted names) and asserts
// byte-identical response on two consecutive calls.
func TestWorkbench_StableOrdering(t *testing.T) {
	reg := contributions.NewRegistry()
	reg.Set("zebra-plugin", fakeContributes("commandPalette"))
	reg.Set("alpha-plugin", fakeContributes("commandPalette"))
	reg.Set("mango-plugin", fakeContributes("commandPalette"))

	s := newTestServer(reg)

	rr1 := doGet(s)
	rr2 := doGet(s)

	b1 := rr1.Body.String()
	b2 := rr2.Body.String()

	if b1 != b2 {
		t.Errorf("responses differ between calls:\ncall1: %s\ncall2: %s", b1, b2)
	}
}

// TestWorkbench_NilRegistryReturns503 asserts that a nil contribReg results in
// 503 Service Unavailable with code EREGISTRY.
func TestWorkbench_NilRegistryReturns503(t *testing.T) {
	s := newTestServer(nil) // nil registry — defensive path
	rr := doGet(s)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if body["code"] != "EREGISTRY" {
		t.Errorf("code = %q, want %q", body["code"], "EREGISTRY")
	}
	if body["msg"] == "" {
		t.Error("msg must not be empty")
	}
}

// TestWorkbench_MultipleMenus sets one plugin with entries under two distinct
// menu paths and asserts both keys appear in the response.
func TestWorkbench_MultipleMenus(t *testing.T) {
	reg := contributions.NewRegistry()

	multiRun := &plugin.CommandRunV1{Kind: "notify", Message: "hi"}
	c := plugin.ContributesV1{
		Commands: []plugin.CommandV1{
			{ID: "multi.cmd", Title: "Multi", Run: multiRun},
		},
		StatusBar:   []plugin.StatusBarItemV1{{ID: "multi.bar", Text: "bar"}},
		Keybindings: []plugin.KeybindingV1{{Command: "multi.cmd", Key: "ctrl+m"}},
		Menus: map[string][]plugin.MenuEntryV1{
			"appBar/right": {
				{Command: "multi.cmd", Group: "nav@1"},
			},
			"commandPalette": {
				{Command: "multi.cmd", Group: "palette@1"},
			},
		},
	}
	reg.Set("multi-plugin", c)

	s := newTestServer(reg)
	rr := doGet(s)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var flat contributions.FlatContributions
	if err := json.Unmarshal(rr.Body.Bytes(), &flat); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := flat.Menus["appBar/right"]; !ok {
		t.Error("menus[\"appBar/right\"] missing")
	}
	if _, ok := flat.Menus["commandPalette"]; !ok {
		t.Error("menus[\"commandPalette\"] missing")
	}
}
