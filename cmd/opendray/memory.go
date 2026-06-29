// Subcommand `opendray memory` — an argv frontend to opendray's shared
// cross-agent memory, for agent CLIs that CANNOT load the stdio
// `opendray mcp-memory` MCP server (notably antigravity/agy, whose MCP
// is a closed plugin/protobuf system with no per-session config file).
//
// It reuses the exact same gateway-forwarding handlers as the MCP server
// (memMCPServer.dispatchTool), so the two surfaces expose identical tools
// and can never drift. The agent calls these through its Bash/shell tool:
//
//	opendray memory tools
//	opendray memory call memory_search '{"query":"db url"}'
//	opendray memory call memory_store  '{"text":"...","metadata":{"type":"project_fact"}}'
//	opendray memory call session_log_append '{"content":"fixed X"}'
//	opendray memory call current_objective_get
//
// Auth + scope come from the same env the MCP server reads
// (OPENDRAY_BASE_URL / OPENDRAY_API_KEY / OPENDRAY_MEMORY_SCOPE /
// OPENDRAY_MEMORY_SCOPE_KEY), which opendray exports into the session at
// spawn time for non-MCP providers.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

const memoryUsage = `opendray memory — call opendray's shared cross-agent memory from a shell.

Usage:
  opendray memory tools [--json]          List available tools.
  opendray memory call <tool> [<json>]    Invoke a tool. <json> is the
                                          arguments object; omit it for
                                          no-arg tools, or pass '-' to read
                                          the JSON from stdin.

Auth/scope come from the environment (set automatically inside an
opendray session):
  OPENDRAY_BASE_URL, OPENDRAY_API_KEY                 (required)
  OPENDRAY_MEMORY_SCOPE, OPENDRAY_MEMORY_SCOPE_KEY    (optional)
Flags --scope / --scope-key override the env when given.

Examples:
  opendray memory call memory_search '{"query":"deploy topology"}'
  opendray memory call memory_store '{"text":"deploys on LXC 86xx","metadata":{"type":"project_fact"}}'
  opendray memory call session_log_append '{"content":"shipped codex bypass fix"}'
  opendray memory call current_objective_get
`

func runMemory(args []string) int {
	fs := flag.NewFlagSet("memory", flag.ContinueOnError)
	scope := fs.String("scope", "", "override OPENDRAY_MEMORY_SCOPE (session|project|global)")
	scopeKey := fs.String("scope-key", "", "override OPENDRAY_MEMORY_SCOPE_KEY (cwd / session id)")
	asJSON := fs.Bool("json", false, "tools: emit JSON instead of text")
	fs.Usage = func() { fmt.Fprint(os.Stderr, memoryUsage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return 2
	}

	switch rest[0] {
	case "tools", "list":
		return memoryListTools(*asJSON)
	case "call":
		return memoryCall(rest[1:], *scope, *scopeKey)
	default:
		fmt.Fprintf(os.Stderr, "opendray memory: unknown subcommand %q\n\n", rest[0])
		fs.Usage()
		return 2
	}
}

// memoryListTools prints the shared tool catalogue (same toolDefs the MCP
// server returns for tools/list) so an agent can discover what to call.
func memoryListTools(asJSON bool) int {
	if asJSON {
		b, err := json.MarshalIndent(toolDefs, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(b))
		return 0
	}
	for _, t := range toolDefs {
		name, _ := t["name"].(string)
		desc, _ := t["description"].(string)
		if i := strings.IndexByte(desc, '\n'); i >= 0 {
			desc = desc[:i]
		}
		fmt.Printf("%-22s %s\n", name, desc)
	}
	return 0
}

// memoryCall invokes one tool by name with a JSON arguments object,
// reusing the MCP server's gateway-forwarding dispatch. Result text goes
// to stdout; a tool-level error (or isError result) exits non-zero so the
// agent's shell sees the failure.
func memoryCall(args []string, scopeOverride, scopeKeyOverride string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "opendray memory call: tool name required (try `opendray memory tools`)")
		return 2
	}
	tool := args[0]

	rawArgs := "{}"
	if len(args) >= 2 {
		if args[1] == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "opendray memory call: read stdin:", err)
				return 2
			}
			if s := strings.TrimSpace(string(data)); s != "" {
				rawArgs = s
			}
		} else if s := strings.TrimSpace(args[1]); s != "" {
			rawArgs = s
		}
	}
	if !json.Valid([]byte(rawArgs)) {
		fmt.Fprintf(os.Stderr, "opendray memory call: arguments are not valid JSON: %s\n", rawArgs)
		return 2
	}

	cfg, err := loadMemMCPConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if scopeOverride != "" {
		cfg.scope = scopeOverride
	}
	if scopeKeyOverride != "" {
		cfg.scopeKey = scopeKeyOverride
	}
	srv := &memMCPServer{
		cfg:    cfg,
		client: &http.Client{},
		out:    os.Stdout,
		outMu:  &sync.Mutex{},
		errLog: os.Stderr,
	}

	result, callErr, known := srv.dispatchTool(tool, json.RawMessage(rawArgs))
	if !known {
		fmt.Fprintf(os.Stderr, "opendray memory call: unknown tool %q (try `opendray memory tools`)\n", tool)
		return 2
	}
	if callErr != nil {
		fmt.Fprintln(os.Stderr, "tool error:", callErr)
		return 1
	}
	text, isErr := memoryResultText(result)
	if isErr {
		fmt.Fprintln(os.Stderr, text)
		return 1
	}
	fmt.Println(text)
	return 0
}

// memoryResultText flattens an MCP tool result ({"content":[{text}], ...})
// into a single string, and reports whether the result was flagged
// isError. Falls back to JSON for any unexpected shape.
func memoryResultText(result any) (string, bool) {
	m, ok := result.(map[string]any)
	if !ok {
		b, _ := json.Marshal(result)
		return string(b), false
	}
	isErr, _ := m["isError"].(bool)

	var b strings.Builder
	appendText := func(c any) {
		cm, ok := c.(map[string]any)
		if !ok {
			return
		}
		if t, ok := cm["text"].(string); ok {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(t)
		}
	}
	switch content := m["content"].(type) {
	case []map[string]any:
		for _, c := range content {
			appendText(c)
		}
	case []any:
		for _, c := range content {
			appendText(c)
		}
	default:
		raw, _ := json.Marshal(m)
		return string(raw), isErr
	}
	return b.String(), isErr
}
