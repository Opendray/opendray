// Subcommand `opendray mcp-memory` — a stdio MCP (Model Context
// Protocol) server that exposes opendray's in-process memory
// subsystem to any agent CLI that supports MCP servers (Claude
// Code, Codex, Cursor, …).
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
//	OPENDRAY_MEMORY_SCOPE_KEY    cwd / session id / operator   (default: process
//	                             cwd when scope=project, else empty)
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
		cfg:    cfg,
		client: &http.Client{},
		out:    os.Stdout,
		outMu:  &sync.Mutex{},
		errLog: os.Stderr,
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
	// Project scope with no explicit key falls back to the process cwd.
	// This is antigravity's path: its MCP entry lives in the HOME-global
	// mcp_config.json shared by every session under that HOME, so the
	// adapter can't bake a per-session cwd into the entry's env — but agy
	// spawns MCP subprocesses from the session workspace, so Getwd IS the
	// session's scope key. Providers with per-session config files
	// (claude/codex/opencode) still pass the key explicitly.
	if c.scope == "project" && c.scopeKey == "" {
		if wd, err := os.Getwd(); err == nil {
			c.scopeKey = wd
		}
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
const instructionsBlurb = `Persistent cross-agent memory backed by opendray. Several layers,
each with a different rhythm and time-horizon — pick the right one:

  memory_*            short DISCRETE FACTS; retrieved top-K-relevant
                      (e.g. "user prefers pnpm", "DB at db.example:5432")
  project_goal_*      LONG-TERM intent (the North Star) — what we're
                      ultimately building. Rare changes; files a proposal.
  project_plan_*      MEDIUM-TERM roadmap / arc of phases. Update when the
                      roadmap SHAPE moves (phase done, scope shift); proposal.
  current_objective_* SHORT-TERM: what we're working on RIGHT NOW + its
                      immediate steps. Writes the live doc DIRECTLY (no
                      approval). Churns constantly — completing it = a
                      short-term objective done.
  session_log_append  PROJECT JOURNAL — append every time you finish a
                      meaningful step, fix a bug, hit a blocker, learn
                      something the next session should know.
  decision_record     ADR-style architectural locks-in (rare).
  project_search      CROSS-LAYER semantic search across facts, journal,
                      goal, plan, objective, AND the global knowledge
                      pages. A knowledge hit returns the matching SECTION
                      plus a doc_read(slug, section=…) pointer — follow it.
  doc_read            Pull ONE doc/section on demand. The knowledge index
                      lists global kb_* pages (op architecture, conventions,
                      integration contract, lessons, reusable features) by
                      slug — they are NOT loaded at spawn. Use
                      doc_read(slug="kb_integrations", section="…") to read
                      just one heading-section of a large page (~1K) rather
                      than the whole guide.

CRITICAL HABITS:

1. Call memory_load_context at session start so you don't repeat
   work prior sessions already addressed.

2. AS YOU WORK, call session_log_append liberally. The journal is
   the primary way future sessions know "where we are". A session
   that ends with no journal entries is a session that taught the
   next agent nothing.

3. Keep current_objective_set current — recognise the situation from
   the conversation, NO special keyword needed: call it when the
   conversation (a) sets a NEW immediate objective, (b) FINISHES the
   current one (roll it to the next), or (c) shifts its steps. This
   is the doc with the most context — you, mid-session — keeping it
   live. Reserve project_plan_set/project_goal_set for the rarer
   roadmap/North-Star changes (those file a proposal).

4. memory_store is for FUTURE-SESSION-USEFUL FACTS, not for
   tracking what you're currently doing. "Working on M5" is NOT
   a memory_store entry — that goes in session_log_append or as
   a current_objective_set update.

5. Before doing anything that touches how OUR SYSTEM works —
   integrating with opendray, provider/MCP/scope/auth wiring,
   following team conventions, or reusing a known pattern — FIRST
   scan the knowledge index for a relevant kb_* page and
   doc_read(slug, section=…) the matching section. Do NOT infer
   our system's design or rules from memory; the kb_* pages are the
   source of truth and they update.`

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
	{
		"name": "project_goal_get",
		"description": "Read the project's long-term goal document. " +
			"One markdown body per project. Empty when the operator has " +
			"not seeded one yet.",
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		"name": "project_plan_get",
		"description": "Read the project's current plan / roadmap " +
			"document. Same shape as project_goal_get — one body per " +
			"project, markdown.",
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		"name": "current_objective_get",
		"description": "Read the project's CURRENT OBJECTIVE document — the " +
			"short-term thing we're working on right now and its immediate " +
			"steps. One markdown body per project; empty when none is set yet.",
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		"name": "current_objective_set",
		"description": "Set the project's CURRENT OBJECTIVE — the short-term " +
			"thing we're working on right now and its immediate steps. Writes " +
			"the LIVE document directly (no operator approval) so it always " +
			"reflects where we are. Call it whenever the conversation, in its " +
			"own words, does any of these — you recognise the situation, there " +
			"is NO keyword to wait for: (1) establishes a NEW immediate " +
			"objective; (2) FINISHES the current one — roll it to the next and " +
			"note what was just done; (3) SHIFTS the objective's scope or " +
			"steps. It is expected to change often. For the medium-term " +
			"roadmap use project_plan_set; for long-term intent, project_goal_set.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The full current-objective markdown (objective + immediate steps).",
				},
				"reason": map[string]any{
					"type":        "string",
					"description": "Optional short note (used only if the doc is human-locked and this falls back to a proposal).",
				},
			},
			"required": []string{"content"},
		},
	},
	{
		"name": "project_goal_set",
		"description": "Update the project GOAL — the LONG-TERM intent / North " +
			"Star: what we are ultimately building and why. Files a proposal " +
			"the operator approves (does NOT overwrite the live doc directly). " +
			"Use RARELY — only when the long-term direction genuinely shifts, " +
			"or to seed an initial goal. Day-to-day progress goes to " +
			"current_objective_set, not here.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The full proposed goal markdown.",
				},
				"reason": map[string]any{
					"type":        "string",
					"description": "Short rationale shown to the operator in the inbox.",
				},
			},
			"required": []string{"content"},
		},
	},
	{
		"name": "project_plan_set",
		"description": "Update the project PLAN — the MEDIUM-TERM roadmap / arc " +
			"of phases. Files a proposal the operator approves (same flow as " +
			"project_goal_set). Call when the roadmap SHAPE changes: a phase " +
			"finishes, scope shifts, or phase ordering changes. For the " +
			"immediate thing you're working on right now, use " +
			"current_objective_set instead (that one writes live).",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The full proposed plan markdown.",
				},
				"reason": map[string]any{
					"type":        "string",
					"description": "Short rationale shown to the operator.",
				},
			},
			"required": []string{"content"},
		},
	},
	{
		"name": "session_log_append",
		"description": "Append a free-form journal entry to the " +
			"project's session log. Use when you want to record what " +
			"the session just accomplished, surface a question for the " +
			"next session, or note a non-decision finding. The entry " +
			"is visible to every future session in this project.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "One-line preview (e.g. \"M5 backend landed\").",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Full markdown body of the journal entry.",
				},
			},
			"required": []string{"content"},
		},
	},
	{
		"name": "decision_record",
		"description": "Append an ADR-style decision to the project " +
			"journal. Use when the session locked in a choice (\"we " +
			"picked Postgres over MySQL because X\") that future " +
			"sessions need to know about. Tagged kind=decision so the " +
			"operator UI can filter / index these separately from " +
			"general journal entries.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "ADR-style short title (e.g. \"Use pgvector for embeddings\").",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Body — context, decision, alternatives, consequences.",
				},
			},
			"required": []string{"title", "content"},
		},
	},
	{
		"name": "doc_read",
		"description": "Read ONE document on demand: a section of this " +
			"project's official doc (e.g. \"plan\", \"tech_stack\", or a " +
			"custom section slug from the project doc index) or a global " +
			"knowledge page (kb_* slug from the knowledge index). Use " +
			"this to pull exactly the document a task needs instead of " +
			"relying on whatever was injected at spawn. Pass `section` to " +
			"read just ONE heading-section of a large page — e.g. " +
			"doc_read(slug=\"kb_integrations\", section=\"Authentication\") " +
			"returns that section (~1K) instead of the whole 59K guide. A " +
			"wrong section returns the page's section list so you can retry.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"slug": map[string]any{
					"type": "string",
					"description": "Section slug (project doc) or kb_* slug " +
						"(global knowledge page).",
				},
				"section": map[string]any{
					"type": "string",
					"description": "Optional. A heading from the page to read " +
						"just that section (case/space-insensitive). Omit to " +
						"read the whole document.",
				},
			},
			"required": []string{"slug"},
		},
	},
	{
		"name": "skill_distill",
		"description": "Save a procedure from THIS session as a reusable " +
			"skill, when the operator asks you to (e.g. \"把刚才的过程存为技能\" " +
			"/ \"save this as a skill\"). You author the draft from your live " +
			"context; a structural quality gate validates it (≥3 concrete " +
			"steps with real commands/paths, a trigger, evidence quotes); it " +
			"lands DISABLED in the operator's workbench for review — their " +
			"enable click is the approval. Never include secrets.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "Short imperative title (e.g. 'Deploy a Nuxt app update to the PDA-web LXC').",
				},
				"applies_when": map[string]any{
					"type":        "string",
					"description": "The trigger/situation in which to reach for this skill.",
				},
				"steps": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "≥3 ordered, concrete steps reusing the REAL commands/paths from this session.",
				},
				"pitfalls": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Failure modes actually hit and how to avoid them.",
				},
				"evidence": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "1-3 short verbatim quotes from this session proving the procedure happened.",
				},
			},
			"required": []string{"title", "applies_when", "steps", "evidence"},
		},
	},
	{
		"name": "project_search",
		"description": "Search ACROSS all memory layers (facts + " +
			"journal entries + goal/plan documents) in the current " +
			"project. Use this when you want context that might live " +
			"anywhere — \"what did we decide about X\", \"have we " +
			"hit Y before\", \"how does our plan handle Z\". Returns " +
			"results ranked by semantic similarity with a time-decay " +
			"penalty, each tagged with which layer it came from.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Natural-language query.",
				},
				"top_k": map[string]any{
					"type":        "integer",
					"description": "Max hits across all layers combined (default 10, max 100).",
				},
			},
			"required": []string{"query"},
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

	result, err, known := s.dispatchTool(params.Name, params.Arguments)
	if !known {
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

// dispatchTool routes one tool call to its handler and returns the MCP
// tool result (a {"content":[...]} map), a tool-level error, and whether
// the tool name was recognised. Shared by the stdio MCP frontend
// (handleToolCall) and the argv frontend (`opendray memory call`), so the
// two surfaces can never drift in which tools they expose or how each
// forwards to the gateway. known=false means "no such tool".
func (s *memMCPServer) dispatchTool(name string, args json.RawMessage) (result any, err error, known bool) {
	if len(args) == 0 {
		// argv callers may omit arguments for no-arg tools; the per-tool
		// handlers unmarshal into a struct, which rejects empty input.
		args = json.RawMessage("{}")
	}
	switch name {
	case "memory_search":
		result, err = s.callSearch(args)
	case "memory_store":
		result, err = s.callStore(args)
	case "memory_list":
		result, err = s.callList(args)
	case "memory_load_context":
		result, err = s.callLoadContext(args)
	case "memory_get_provenance":
		result, err = s.callGetProvenance(args)
	case "project_goal_get":
		result, err = s.callProjectDocGet("goal")
	case "project_plan_get":
		result, err = s.callProjectDocGet("plan")
	case "current_objective_get":
		result, err = s.callProjectDocGet("current_objective")
	case "project_goal_set":
		result, err = s.callProjectDocSet("goal", args)
	case "project_plan_set":
		result, err = s.callProjectDocSet("plan", args)
	case "current_objective_set":
		result, err = s.callProjectDocSet("current_objective", args)
	case "session_log_append":
		result, err = s.callSessionLogAppend("manual", args)
	case "decision_record":
		result, err = s.callSessionLogAppend("decision", args)
	case "doc_read":
		result, err = s.callDocRead(args)
	case "skill_distill":
		result, err = s.callSkillDistill(args)
	case "project_search":
		result, err = s.callProjectSearch(args)
	default:
		return nil, nil, false
	}
	return result, err, true
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
			fmt.Fprintf(&b, "[%d] %s\n  similarity=%.3f id=%v\n",
				i+1, stringField(h.Memory, "text"), h.Similarity, h.Memory["id"])
			b.WriteString(foldedVariantsBlock(h.Memory, "  "))
			b.WriteByte('\n')
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
			b.WriteString(foldedVariantsBlock(memory, "  "))
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

// callProjectDocGet fetches the live goal or plan document for the
// active cwd (s.cfg.scopeKey). Returns the body as a text content
// block. The gateway returns an empty doc when the row doesn't
// exist, so the agent never gets a 404 — it just sees "(empty)".
func (s *memMCPServer) callProjectDocGet(kind string) (any, error) {
	cwd := s.cfg.scopeKey
	if cwd == "" {
		return nil, errors.New("project_doc requires OPENDRAY_MEMORY_SCOPE_KEY (cwd) to be set")
	}
	path := "/api/v1/project-docs/" + kind + "?cwd=" + urlQuery(cwd)
	var doc struct {
		Kind      string `json:"kind"`
		Content   string `json:"content"`
		UpdatedBy string `json:"updated_by"`
	}
	if err := s.gatewayGetJSON(path, &doc); err != nil {
		return nil, err
	}
	var b strings.Builder
	if strings.TrimSpace(doc.Content) == "" {
		fmt.Fprintf(&b, "(no %s set for this project yet)", kind)
	} else {
		fmt.Fprintf(&b, "# Project %s\n\n_last updated by %s_\n\n%s", kind, doc.UpdatedBy, doc.Content)
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
	}, nil
}

// callDocRead fetches ONE document on demand: a project doc section
// (slug from the project's blueprint) or a global knowledge page
// (kb_* slug). The lean spawn mode injects only an index; this is how
// the agent pulls the actual content it needs.
func (s *memMCPServer) callDocRead(args json.RawMessage) (any, error) {
	var in struct {
		Slug    string `json:"slug"`
		Section string `json:"section"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		return nil, errors.New("doc_read requires a slug")
	}
	cwd := s.cfg.scopeKey
	if strings.HasPrefix(slug, "kb_") {
		cwd = "__global__" // knowledge pages live under the global sentinel
	}
	if cwd == "" {
		return nil, errors.New("doc_read requires OPENDRAY_MEMORY_SCOPE_KEY (cwd) to be set")
	}
	path := "/api/v1/project-docs/" + urlQuery(slug) + "?cwd=" + urlQuery(cwd)
	// Optional section= pulls one heading-section instead of the whole page.
	if section := strings.TrimSpace(in.Section); section != "" {
		path += "&section=" + urlQuery(section)
	}
	var doc struct {
		Kind      string `json:"kind"`
		Content   string `json:"content"`
		UpdatedBy string `json:"updated_by"`
	}
	if err := s.gatewayGetJSON(path, &doc); err != nil {
		return nil, err
	}
	var b strings.Builder
	if strings.TrimSpace(doc.Content) == "" {
		fmt.Fprintf(&b, "(document %q is empty)", slug)
	} else {
		fmt.Fprintf(&b, "# %s\n\n_last updated by %s_\n\n%s", slug, doc.UpdatedBy, doc.Content)
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
	}, nil
}

// callSkillDistill posts an agent-authored skill draft for operator
// review (manual-trigger distillation).
func (s *memMCPServer) callSkillDistill(args json.RawMessage) (any, error) {
	var in struct {
		Title       string   `json:"title"`
		AppliesWhen string   `json:"applies_when"`
		Steps       []string `json:"steps"`
		Pitfalls    []string `json:"pitfalls"`
		Evidence    []string `json:"evidence"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("bad arguments: %w", err)
	}
	payload := map[string]any{
		"title":        in.Title,
		"applies_when": in.AppliesWhen,
		"steps":        in.Steps,
		"pitfalls":     in.Pitfalls,
		"evidence":     in.Evidence,
		// No session-id env is plumbed into the MCP subprocess; the
		// cwd is the meaningful provenance anchor.
		"session_id": "",
		"cwd":        s.cfg.scopeKey,
	}
	var node struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := s.gatewayPostJSON("/api/v1/knowledge/skills/distill", payload, &node); err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": "Skill draft \"" + node.Title + "\" submitted (id " + node.ID +
				"). It is DISABLED pending the operator's review in Cortex → 知识 → 蒸馏 — " +
				"tell the operator it is waiting there."},
		},
	}, nil
}

// callProjectDocSet writes goal / plan / current_objective through the
// gateway's policy-aware set endpoint. The gateway routes on the
// section's blueprint write_policy: a "direct" section (current_objective)
// updates the LIVE doc immediately when unlocked; "proposal" sections
// (goal/plan) — or any doc a human has hand-edited — file a proposal that
// the operator approves. The response's `action` tells us which happened
// so the agent gets accurate feedback.
func (s *memMCPServer) callProjectDocSet(kind string, args json.RawMessage) (any, error) {
	cwd := s.cfg.scopeKey
	if cwd == "" {
		return nil, errors.New("project_doc_set requires OPENDRAY_MEMORY_SCOPE_KEY (cwd) to be set")
	}
	var in struct {
		Content string `json:"content"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(in.Content) == "" {
		return nil, errors.New("content is required")
	}
	body := map[string]any{
		"cwd":     cwd,
		"content": in.Content,
		"reason":  in.Reason,
	}
	var out struct {
		Action   string `json:"action"`
		Proposal struct {
			ID string `json:"id"`
		} `json:"proposal"`
	}
	if err := s.gatewayPostJSON("/api/v1/project-docs/"+kind+"/set", body, &out); err != nil {
		return nil, err
	}
	var text string
	switch out.Action {
	case "applied":
		text = fmt.Sprintf("Updated the live %s document directly.", kind)
	case "proposed":
		if out.Proposal.ID == "" {
			text = fmt.Sprintf("Filed a %s proposal (gateway returned no id) — check the opendray inbox.", kind)
		} else {
			text = fmt.Sprintf(
				"Filed %s proposal %s — this doc is human-locked or proposal-gated, so the live doc is unchanged until the operator approves.",
				kind, out.Proposal.ID)
		}
	default:
		text = fmt.Sprintf("Gateway returned an unexpected action %q for the %s set — nothing confirmed.", out.Action, kind)
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	}, nil
}

// callSessionLogAppend writes one journal entry. Used by both
// session_log_append (kind=manual) and decision_record (kind=
// decision); the schema then tags it author=agent so the operator
// UI can distinguish agent-written entries from operator-written
// ones at a glance.
func (s *memMCPServer) callSessionLogAppend(kind string, args json.RawMessage) (any, error) {
	cwd := s.cfg.scopeKey
	if cwd == "" {
		return nil, errors.New("session_log requires OPENDRAY_MEMORY_SCOPE_KEY (cwd) to be set")
	}
	var in struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(in.Content) == "" {
		return nil, errors.New("content is required")
	}
	if kind == "decision" && strings.TrimSpace(in.Title) == "" {
		return nil, errors.New("decision_record requires a title")
	}
	body := map[string]any{
		"cwd":        cwd,
		"kind":       kind,
		"title":      in.Title,
		"content":    in.Content,
		"updated_by": "agent",
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := s.gatewayPostJSON("/api/v1/session-logs", body, &out); err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": fmt.Sprintf("Appended %s journal entry %s.", kind, out.ID)},
		},
	}, nil
}

// callProjectSearch wraps /api/v1/project-search. Surfaces hits
// from all five memory layers in one ranked list so the agent can
// answer "have we touched X before" without choosing a layer
// first. Output is rendered as a markdown bullet list so the
// model can quote pieces back into its response naturally.
func (s *memMCPServer) callProjectSearch(args json.RawMessage) (any, error) {
	cwd := s.cfg.scopeKey
	if cwd == "" {
		return nil, errors.New("project_search requires OPENDRAY_MEMORY_SCOPE_KEY (cwd) to be set")
	}
	var in struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, errors.New("query is required")
	}
	topK := in.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 100 {
		topK = 100
	}

	url := fmt.Sprintf("/api/v1/project-search?cwd=%s&q=%s&top_k=%d",
		urlQuery(cwd), urlQuery(q), topK)
	var resp struct {
		Hits []struct {
			Source         string  `json:"source"`
			ID             string  `json:"id"`
			Text           string  `json:"text"`
			Title          string  `json:"title"`
			Similarity     float32 `json:"similarity"`
			EffectiveScore float32 `json:"effective_score"`
			CreatedAt      string  `json:"created_at"`
			Slug           string  `json:"slug"`
			Section        string  `json:"section"`
		} `json:"hits"`
	}
	if err := s.gatewayGetJSON(url, &resp); err != nil {
		return nil, err
	}
	if len(resp.Hits) == 0 {
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": fmt.Sprintf("No cross-layer hits for %q.", q)},
			},
		}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Top %d cross-layer matches for %q (effective_score · source — preview):\n\n",
		len(resp.Hits), q)
	for _, h := range resp.Hits {
		text := strings.TrimSpace(h.Text)
		if h.Title != "" && !strings.HasPrefix(text, h.Title) {
			text = h.Title + " — " + text
		}
		if len(text) > 240 {
			text = text[:240] + "…"
		}
		fmt.Fprintf(&b, "- **%.2f · %s** — %s\n", h.EffectiveScore, h.Source, text)
		// Knowledge hits carry a slug (+ section when the page has
		// subsections) — surface the exact doc_read call so the agent pulls
		// the full section cheaply instead of guessing a slug or swallowing
		// the whole page.
		if h.Slug != "" {
			if h.Section != "" {
				fmt.Fprintf(&b, "    → doc_read(slug=%q, section=%q)\n", h.Slug, h.Section)
			} else {
				fmt.Fprintf(&b, "    → doc_read(slug=%q)  # full page on demand\n", h.Slug)
			}
		}
	}
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

// mergedFromTexts extracts the absorbed variant texts from a memory's
// metadata.merged_from audit list (oldest first), or nil when the row was
// never folded. Write-time dedup (memory.Service.Store) overwrites the
// canonical text with the newest write and parks the superseded text here —
// lossless in the DB, but invisible to search until we surface it. Two facts
// that differ only in a critical token (a port, a provider tag, a version)
// embed as near-duplicates and fold, so without this the distinguishing one
// silently vanishes from recall.
func mergedFromTexts(memory map[string]any) []string {
	meta, ok := memory["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := meta["merged_from"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	var out []string
	for _, e := range raw {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		if t, ok := em["text"].(string); ok && strings.TrimSpace(t) != "" {
			out = append(out, t)
		}
	}
	return out
}

// foldedVariantsBlock renders the "folded in N earlier writes" suffix for a
// search / load_context hit, or "" when the row was never deduped (the common
// case, so normal output is unchanged). indent is prepended to every line so
// it nests under the hit. Each variant is collapsed to a single line.
func foldedVariantsBlock(memory map[string]any, indent string) string {
	texts := mergedFromTexts(memory)
	if len(texts) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s↳ folded in %d earlier write(s) (deduped, kept for recall):\n", indent, len(texts))
	for _, t := range texts {
		oneLine := strings.Join(strings.Fields(t), " ")
		fmt.Fprintf(&b, "%s    • %s\n", indent, oneLine)
	}
	return b.String()
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func urlQuery(s string) string {
	// Minimal escaping for path segments + query values. `%` and `+` MUST
	// be escaped: an unescaped `%` makes the server's url.ParseQuery choke
	// (silent empty value), and an unescaped `+` decodes to a space — both
	// would silently break a section= heading like "100% done" / "Auth +
	// OAuth". NewReplacer scans the input once and never re-scans its own
	// output, so escaping `%`→`%25` here can't double-escape the `%20`/`%2F`
	// it also emits. Order: `%` first as defensive documentation of intent.
	r := strings.NewReplacer(
		"%", "%25",
		"+", "%2B",
		" ", "%20",
		"/", "%2F",
		"?", "%3F",
		"&", "%26",
		"=", "%3D",
		"#", "%23",
	)
	return r.Replace(s)
}
