package main

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

// OPENDRAY_MEMORY_READONLY=1 flips the memory MCP into read-only mode.
func TestLoadMemMCPConfig_ReadOnly(t *testing.T) {
	t.Setenv("OPENDRAY_BASE_URL", "http://127.0.0.1:8770")
	t.Setenv("OPENDRAY_API_KEY", "k")

	t.Setenv("OPENDRAY_MEMORY_READONLY", "")
	if cfg, err := loadMemMCPConfig(); err != nil || cfg.readOnly {
		t.Fatalf("default should be read-write: readOnly=%v err=%v", cfg.readOnly, err)
	}
	t.Setenv("OPENDRAY_MEMORY_READONLY", "1")
	if cfg, err := loadMemMCPConfig(); err != nil || !cfg.readOnly {
		t.Fatalf("flag should enable read-only: readOnly=%v err=%v", cfg.readOnly, err)
	}
}

// A read-only session's tools/list must omit every write tool while keeping
// the read/search tools. A read-write session lists them all.
func TestToolsList_ReadOnlyFiltersWriteTools(t *testing.T) {
	list := func(readOnly bool) map[string]bool {
		var buf bytes.Buffer
		s := &memMCPServer{
			cfg:   memMCPConfig{readOnly: readOnly},
			out:   &buf,
			outMu: &sync.Mutex{},
		}
		s.handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))

		var resp struct {
			Result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal tools/list response: %v (raw: %s)", err, buf.String())
		}
		names := map[string]bool{}
		for _, tl := range resp.Result.Tools {
			names[tl.Name] = true
		}
		return names
	}

	ro := list(true)
	if len(ro) == 0 {
		t.Fatal("read-only tools/list returned nothing")
	}
	for name := range writeToolNames {
		if ro[name] {
			t.Errorf("read-only session must not list write tool %q", name)
		}
	}
	// Read tools survive.
	for _, want := range []string{"memory_search", "memory_list", "project_search", "doc_read"} {
		if !ro[want] {
			t.Errorf("read-only session must still list read tool %q", want)
		}
	}

	// A normal session lists the write tools.
	rw := list(false)
	if !rw["memory_store"] {
		t.Error("read-write session should list memory_store")
	}
}

// Defense in depth: even called by name, a write tool is refused in read-only
// mode — before any gateway HTTP call.
func TestDispatchTool_ReadOnlyRefusesWrites(t *testing.T) {
	s := &memMCPServer{cfg: memMCPConfig{readOnly: true}}
	for name := range writeToolNames {
		_, err, known := s.dispatchTool(name, nil)
		if !known {
			t.Errorf("%q should be a known tool name", name)
		}
		if err == nil {
			t.Errorf("read-only mode must refuse write tool %q", name)
		}
	}
}
