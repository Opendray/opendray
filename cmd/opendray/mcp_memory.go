// Subcommand `opendray mcp-memory` — a stdio MCP (Model Context
// Protocol) server that exposes opendray's in-process memory
// subsystem to any agent CLI that supports MCP servers (Claude
// Code, Codex, Gemini, Cursor, …).
//
// Architecture:
//
//	agent CLI ── stdio JSON-RPC ──> `opendray mcp-memory` ── HTTP ──> opendray gateway
//
// The subprocess is intentionally thin — it holds no state and no
// DB connection. Every tool call is forwarded to the gateway's
// /api/v1/memory/* endpoints, authenticated with whatever
// bearer the operator wired into the launching env. That keeps the
// memory layer's business logic in one place (internal/memory) and
// makes the MCP wrapper trivially easy to evolve as MCP itself does.
//
// Operators run this either:
//   - manually, by adding a server config to ~/.claude.json or per-
//     session mcp.json (see Settings → Memory tutorial), or
//   - via opendray's auto-attach (the catalog adapter renders an
//     mcp.json into each session's scratch dir at spawn time).
//
// Required env vars:
//
//	OPENDRAY_BASE_URL    e.g. http://127.0.0.1:8770 (no trailing slash)
//	OPENDRAY_API_KEY     bearer token (admin or integration with memory scopes)
//
// Optional env vars (defaults used when absent):
//
//	OPENDRAY_MEMORY_SCOPE        session | project | global   (default project)
//	OPENDRAY_MEMORY_SCOPE_KEY    cwd / session id / operator   (default empty for global, else required)
//
// MCP protocol coverage is the minimum that real clients exercise:
// initialize handshake, tools/list, tools/call. Resources, prompts,
// completions, and roots are NOT implemented — they're not needed
// for memory recall and adding them later is a no-op extension.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

// runMcpMemory is the subcommand entry point. Returns a process
// exit code. The whole thing is one readline loop — no goroutines —
// because MCP stdio is strictly request/response over a single
// channel.
func runMcpMemory(args []string) int {
	fs := flag.NewFlagSet("opendray mcp-memory", flag.ExitOnError)
	_ = fs.Parse(args)

	cfg, err := loadMemMCPConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	srv := &memMCPServer{
		cfg:     cfg,
		client:  &http.Client{},
		out:     os.Stdout,
		outMu:   &sync.Mutex{},
		errLog:  os.Stderr,
	}

	scanner := bufio.NewScanner(os.Stdin)
	// MCP messages are line-delimited JSON-RPC. Bump the buffer to
	// handle large tool-call payloads without crashing.
	scanner.Buffer(make([]byte, 0, 1<<14), 1<<24)

	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		srv.handle(raw)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "mcp-memory: stdin: %v\n", err)
		return 1
	}
	return 0
}

// memMCPConfig is the static config the subprocess reads at start.
type memMCPConfig struct {
	baseURL  string
	apiKey   string
	scope    string
	scopeKey string
}

func loadMemMCPConfig() (memMCPConfig, error) {
	c := memMCPConfig{
		baseURL:  strings.TrimRight(os.Getenv("OPENDRAY_BASE_URL"), "/"),
		apiKey:   os.Getenv("OPENDRAY_API_KEY"),
		scope:    os.Getenv("OPENDRAY_MEMORY_SCOPE"),
		scopeKey: os.Getenv("OPENDRAY_MEMORY_SCOPE_KEY"),
	}
	if c.baseURL == "" {
		return c, errors.New("mcp-memory: OPENDRAY_BASE_URL is required")
	}
	if c.apiKey == "" {
		return c, errors.New("mcp-memory: OPENDRAY_API_KEY is required")
	}
	if c.scope == "" {
		c.scope = "project"
	}
	return c, nil
}

// memMCPServer is the MCP request dispatcher. State held: nothing
// per-request beyond the response buffer; every tool call is a
// fresh HTTP call to the gateway.
type memMCPServer struct {
	cfg    memMCPConfig
	client *http.Client

	out    io.Writer
	outMu  *sync.Mutex // guards concurrent writes when we add notifications
	errLog io.Writer
}

// JSON-RPC 2.0 envelopes. We accept loose IDs (string, number, null).
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// handle dispatches one inbound JSON-RPC message. Notifications
// (no id) are answered with no response per JSON-RPC spec.
func (s *memMCPServer) handle(raw []byte) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		s.respondErr(nil, -32700, "Parse error", err.Error())
		return
	}

	switch req.Method {
	case "initialize":
		s.respond(req.ID, map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "opendray-memory",
				"version": "0.1.0",
			},
			"instructions": instructionsBlurb,
		})
	case "notifications/initialized":
		// no-op; client signals it finished initialise handshake.
	case "ping":
		s.respond(req.ID, map[string]any{})
	case "tools/list":
		s.respond(req.ID, map[string]any{"tools": toolDefs})
	case "tools/call":
		s.handleToolCall(req)
	default:
		// MCP defines other namespaces (resources, prompts) — we
		// reply Method-not-found rather than crashing. Smart clients
		// fall back gracefully.
		s.respondErr(req.ID, -32601, "Method not found", req.Method)
	}
}

// instructionsBlurb shows up in the agent's system context so the
// model knows when to call the tools without explicit prompting.
const instructionsBlurb = `Persistent memory backed by opendray's pgvector store.
Use memory_search before answering anything that might benefit from past
context. Use memory_store after the user states a durable preference,
identifier, decision, relationship, or ongoing-task fact.`

// toolDefs is the static list returned for tools/list.
var toolDefs = []map[string]any{
	{
		"name": "memory_search",
		"description": "Search the operator's persistent memory for facts " +
			"relevant to the query. Returns up to top_k hits ranked by " +
			"semantic similarity. Use this BEFORE answering any question " +
			"that could benefit from past context.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Natural-language query.",
				},
				"top_k": map[string]any{
					"type":        "integer",
					"description": "Max hits to return (default 5).",
				},
			},
			"required": []string{"query"},
		},
	},
	{
		"name": "memory_store",
		"description": "Persist a single durable fact for future recall. " +
			"Use sparingly — only for genuinely durable items: user " +
			"preferences, identifiers (names, URLs, IDs), decisions made, " +
			"ongoing tasks. Do NOT store transient context, the current " +
			"conversation, or things you can re-derive easily.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "The fact to remember, as a self-contained sentence.",
				},
				"metadata": map[string]any{
					"type":        "object",
					"description": "Optional key/value tags for filtering later.",
				},
			},
			"required": []string{"text"},
		},
	},
	{
		"name": "memory_list",
		"description": "List recently stored facts in the active scope. " +
			"Useful when the agent wants to refresh its overall view of " +
			"what's been remembered.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{
					"type":        "integer",
					"description": "Max rows to return (default 50, max 200).",
				},
			},
		},
	},
	{
		"name": "memory_load_context",
		"description": "Load a markdown-formatted summary of the most " +
			"relevant memories for the active scope, suitable for " +
			"prepending to your reasoning. Use this when starting a " +
			"new task that may benefit from project-level context " +
			"the user has accumulated. Returns a single string.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Optional query to focus the context. Empty = use the active scope_key as the query.",
				},
				"top_k": map[string]any{
					"type":        "integer",
					"description": "Max memories to include (default 10, max 50).",
				},
			},
		},
	},
	{
		"name": "memory_get_provenance",
		"description": "Fetch the provenance metadata of one stored " +
			"memory: how it got into the store (manual / mcp_call / " +
			"summarizer / mirror_claude_md / imported), the source " +
			"reference, the originating session id (when extracted " +
			"by the summarizer), and the confidence score.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The memory id (mem_…).",
				},
			},
			"required": []string{"id"},
		},
	},
}

// handleToolCall dispatches one MCP tools/call invocation to the
// appropriate gateway endpoint. All tool calls share the same
// scope/scope_key envelope coming from env so the agent never has
// to guess.
func (s *memMCPServer) handleToolCall(req rpcRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.respondErr(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	var (
		result any
		err    error
	)
	switch params.Name {
	case "memory_search":
		result, err = s.callSearch(params.Arguments)
	case "memory_store":
		result, err = s.callStore(params.Arguments)
	case "memory_list":
		result, err = s.callList(params.Arguments)
	case "memory_load_context":
		result, err = s.callLoadContext(params.Arguments)
	case "memory_get_provenance":
		result, err = s.callGetProvenance(params.Arguments)
	default:
		s.respondErr(req.ID, -32601, "Unknown tool", params.Name)
		return
	}

	if err != nil {
		// Tool errors are NOT JSON-RPC errors — they're successful
		// responses with isError=true so the agent can recover.
		// (See MCP spec: tools/call result can carry isError.)
		s.respond(req.ID, map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "tool error: " + err.Error()},
			},
			"isError": true,
		})
		return
	}
	s.respond(req.ID, result)
}

func (s *memMCPServer) callSearch(args json.RawMessage) (any, error) {
	var in struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Query == "" {
		return nil, errors.New("query is required")
	}
	body := map[string]any{
		"query":     in.Query,
		"scope":     s.cfg.scope,
		"scope_key": s.cfg.scopeKey,
	}
	if in.TopK > 0 {
		body["top_k"] = in.TopK
	}
	var out struct {
		Hits []struct {
			Memory     map[string]any `json:"memory"`
			Similarity float32        `json:"similarity"`
		} `json:"hits"`
	}
	if err := s.gatewayPostJSON("/api/v1/memory/search", body, &out); err != nil {
		return nil, err
	}

	// Render hits as one text block — easier for agents to consume
	// than nested JSON. The MCP spec says content is a list of
	// {type, text, ...} blocks.
	var b strings.Builder
	if len(out.Hits) == 0 {
		b.WriteString("(no memories matched)")
	} else {
		fmt.Fprintf(&b, "Found %d memory hit(s):\n\n", len(out.Hits))
		for i, h := range out.Hits {
			fmt.Fprintf(&b, "[%d] %s\n  similarity=%.3f id=%v\n\n",
				i+1, stringField(h.Memory, "text"), h.Similarity, h.Memory["id"])
		}
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
	}, nil
}

func (s *memMCPServer) callStore(args json.RawMessage) (any, error) {
	var in struct {
		Text     string                 `json:"text"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if in.Text == "" {
		return nil, errors.New("text is required")
	}
	body := map[string]any{
		"text":      in.Text,
		"scope":     s.cfg.scope,
		"scope_key": s.cfg.scopeKey,
	}
	if in.Metadata != nil {
		body["metadata"] = in.Metadata
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := s.gatewayPostJSON("/api/v1/memory/store", body, &out); err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": "stored as " + out.ID},
		},
	}, nil
}

func (s *memMCPServer) callList(args json.RawMessage) (any, error) {
	var in struct {
		Limit int `json:"limit"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &in)
	}
	if in.Limit <= 0 {
		in.Limit = 50
	}
	if in.Limit > 200 {
		in.Limit = 200
	}
	url := fmt.Sprintf(
		"/api/v1/memory/list?scope=%s&scope_key=%s&n=%d",
		s.cfg.scope, urlQuery(s.cfg.scopeKey), in.Limit,
	)
	var out struct {
		Memories []map[string]any `json:"memories"`
	}
	if err := s.gatewayGetJSON(url, &out); err != nil {
		return nil, err
	}
	var b strings.Builder
	if len(out.Memories) == 0 {
		b.WriteString("(no memories yet)")
	} else {
		fmt.Fprintf(&b, "%d memory record(s):\n\n", len(out.Memories))
		for i, m := range out.Memories {
			fmt.Fprintf(&b, "[%d] %s\n", i+1, stringField(m, "text"))
		}
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
	}, nil
}

// callLoadContext renders a markdown banner of relevant memories
// for the active scope. Wraps memory_search + a short formatter
// so the agent gets one ready-to-prepend string instead of an
// array it has to format itself.
func (s *memMCPServer) callLoadContext(args json.RawMessage) (any, error) {
	var in struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &in)
	}
	if in.TopK <= 0 {
		in.TopK = 10
	}
	if in.TopK > 50 {
		in.TopK = 50
	}
	if in.Query == "" {
		// Default: use the scope_key (cwd basename for project,
		// session id otherwise) so the search is at least loosely
		// targeted at the active context.
		in.Query = s.cfg.scopeKey
		if idx := strings.LastIndex(in.Query, "/"); idx >= 0 && idx+1 < len(in.Query) {
			in.Query = in.Query[idx+1:]
		}
		if in.Query == "" {
			in.Query = "context"
		}
	}
	body := map[string]any{
		"query":     in.Query,
		"scope":     s.cfg.scope,
		"scope_key": s.cfg.scopeKey,
		"top_k":     in.TopK,
	}
	var resp struct {
		Hits []map[string]any `json:"hits"`
	}
	if err := s.gatewayPostJSON("/api/v1/memory/search", body, &resp); err != nil {
		return nil, err
	}
	var b strings.Builder
	if len(resp.Hits) == 0 {
		b.WriteString("(no relevant memories found for this scope)")
	} else {
		fmt.Fprintf(&b, "## Relevant project memory\n\nopendray pulled %d memories matching `%s`:\n\n", len(resp.Hits), in.Query)
		for _, hit := range resp.Hits {
			memory, _ := hit["memory"].(map[string]any)
			text := stringField(memory, "text")
			if i := strings.IndexByte(text, '\n'); i >= 0 {
				text = text[:i]
			}
			fmt.Fprintf(&b, "- %s\n", text)
		}
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
	}, nil
}

// callGetProvenance asks /api/v1/memory/{id} for one memory's
// provenance metadata (source_kind, source_ref, summarizer_session,
// confidence). The agent can use this to decide how much to trust
// a particular memory ("did the user type this themselves, or did
// the summarizer extract it?").
func (s *memMCPServer) callGetProvenance(args json.RawMessage) (any, error) {
	var in struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &in); err != nil || in.ID == "" {
		return nil, fmt.Errorf("memory_get_provenance requires an id")
	}
	url := "/api/v1/memory/" + urlQuery(in.ID)
	var memory map[string]any
	if err := s.gatewayGetJSON(url, &memory); err != nil {
		return nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Memory %s:\n", stringField(memory, "id"))
	fmt.Fprintf(&b, "  text:               %s\n", stringField(memory, "text"))
	fmt.Fprintf(&b, "  source_kind:        %s\n", stringField(memory, "source_kind"))
	if v := stringField(memory, "source_ref"); v != "" {
		fmt.Fprintf(&b, "  source_ref:         %s\n", v)
	}
	if v := stringField(memory, "summarizer_session"); v != "" {
		fmt.Fprintf(&b, "  summarizer_session: %s\n", v)
	}
	if v, ok := memory["confidence"].(float64); ok && v > 0 {
		fmt.Fprintf(&b, "  confidence:         %.2f\n", v)
	}
	fmt.Fprintf(&b, "  scope:              %s/%s\n", stringField(memory, "scope"), stringField(memory, "scope_key"))
	fmt.Fprintf(&b, "  embedder:           %s\n", stringField(memory, "embedder"))
	fmt.Fprintf(&b, "  created_at:         %s\n", stringField(memory, "created_at"))
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
	}, nil
}

// gatewayPostJSON sends body as JSON to a path on the gateway and
// decodes the response into out (when out != nil). Errors include
// the response body verbatim so the operator can debug.
func (s *memMCPServer) gatewayPostJSON(path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode body: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.cfg.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.apiKey)
	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("gateway %s: %w", path, err)
	}
	defer res.Body.Close()
	rawRes, _ := io.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("gateway %s returned %d: %s", path, res.StatusCode, string(rawRes))
	}
	if out != nil {
		if err := json.Unmarshal(rawRes, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// gatewayGetJSON is the GET twin of gatewayPostJSON.
func (s *memMCPServer) gatewayGetJSON(path string, out any) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, s.cfg.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.apiKey)
	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("gateway %s: %w", path, err)
	}
	defer res.Body.Close()
	rawRes, _ := io.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("gateway %s returned %d: %s", path, res.StatusCode, string(rawRes))
	}
	if out != nil {
		if err := json.Unmarshal(rawRes, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (s *memMCPServer) respond(id json.RawMessage, result any) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *memMCPServer) respondErr(id json.RawMessage, code int, msg string, data any) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg, Data: data}})
}

func (s *memMCPServer) write(resp rpcResponse) {
	raw, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(s.errLog, "mcp-memory: encode response: %v\n", err)
		return
	}
	s.outMu.Lock()
	defer s.outMu.Unlock()
	_, _ = s.out.Write(raw)
	_, _ = s.out.Write([]byte{'\n'})
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func urlQuery(s string) string {
	// Minimal escaping — full url.QueryEscape pulls in net/url
	// which is fine but the path is short and predictable.
	r := strings.NewReplacer(
		" ", "%20",
		"/", "%2F",
		"?", "%3F",
		"&", "%26",
		"=", "%3D",
		"#", "%23",
	)
	return r.Replace(s)
}
