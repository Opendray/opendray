package mcp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestMain intercepts the test binary when used as a fake MCP server.
// When OPENDRAY_TEST_FAKE_MCP=1, the process acts as a minimal stdio
// MCP server and exits instead of running tests.
func TestMain(m *testing.M) {
	if os.Getenv("OPENDRAY_TEST_FAKE_MCP") == "1" {
		scanner := bufio.NewScanner(os.Stdin)
		// initialize request
		scanner.Scan()
		fmt.Fprintln(os.Stdout, `{"jsonrpc":"2.0","id":1,"result":{"serverInfo":{"name":"fake-vault","version":"9.9"},"capabilities":{}}}`)
		// initialized notification
		scanner.Scan()
		// tools/list request
		scanner.Scan()
		fmt.Fprintln(os.Stdout, `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"read_secret"},{"name":"list_mounts"}]}}`)
		return
	}
	os.Exit(m.Run())
}

func checkByName(res ValidationResult, name string) (Check, bool) {
	for _, c := range res.Checks {
		if c.Name == name {
			return c, true
		}
	}
	return Check{}, false
}

func TestValidate_ConfigSanity(t *testing.T) {
	// sse with the address mistakenly in `command` (the #221 trap).
	res := Validate(context.Background(),
		Server{Transport: "sse", Command: "http://host:3100/sse"}, nil)
	if res.OK {
		t.Error("sse with no url should fail")
	}
	if c, ok := checkByName(res, "config"); !ok || c.OK || !strings.Contains(c.Detail, "command") {
		t.Errorf("expected config check pointing at the command field, got %+v", res.Checks)
	}

	// stdio with no command.
	res = Validate(context.Background(), Server{Transport: "stdio"}, nil)
	if res.OK {
		t.Error("stdio with no command should fail")
	}
}

func TestValidate_SSEReachability_Unreachable(t *testing.T) {
	// Nothing listening here → reachability fails, but config passes.
	res := Validate(context.Background(),
		Server{Transport: "sse", URL: "http://127.0.0.1:1/sse"}, nil)
	if c, ok := checkByName(res, "config"); !ok || !c.OK {
		t.Errorf("config should pass when url is set: %+v", res.Checks)
	}
	if res.OK {
		t.Error("unreachable endpoint should not be OK")
	}
	if !strings.Contains(res.Note, "codex") {
		t.Errorf("sse note should mention the codex caveat: %q", res.Note)
	}
}

func TestValidate_StdioHandshake(t *testing.T) {
	// Use the test binary itself as the fake MCP server (cross-platform).
	// When invoked with OPENDRAY_TEST_FAKE_MCP=1, TestMain handles the
	// MCP handshake and exits before running any tests.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	res := Validate(context.Background(), Server{
		Transport: "stdio",
		Command:   exe,
		Env:       map[string]string{"OPENDRAY_TEST_FAKE_MCP": "1"},
	}, nil)
	if !res.OK {
		t.Fatalf("handshake should succeed: %+v", res.Checks)
	}
	if res.ToolCount != 2 {
		t.Errorf("toolCount = %d, want 2 (%v)", res.ToolCount, res.Tools)
	}
	if res.ServerName != "fake-vault" || res.ServerVersion != "9.9" {
		t.Errorf("serverInfo = %q %q", res.ServerName, res.ServerVersion)
	}
	if c, ok := checkByName(res, "handshake"); !ok || !c.OK {
		t.Errorf("handshake check missing/failed: %+v", res.Checks)
	}
}

func TestValidate_StdioCommandNotFound(t *testing.T) {
	res := Validate(context.Background(),
		Server{Transport: "stdio", Command: "definitely-not-a-real-binary-xyz"}, nil)
	if res.OK {
		t.Error("missing command should fail")
	}
	if c, ok := checkByName(res, "command"); !ok || c.OK {
		t.Errorf("expected failing command check: %+v", res.Checks)
	}
}
