// Subcommand `opendray mcp-dbtool` — a stdio MCP server exposing the
// Database tool (per-project external database connections) to agent
// CLIs, mirroring the mcp-memory architecture:
//
//	agent CLI ── stdio JSON-RPC ──> `opendray mcp-dbtool` ── HTTP ──> opendray gateway
//
// The subprocess is a thin forwarder — no DB connections, no state.
// Every tool call hits the gateway's /api/v1/dbtool/* endpoints with the
// integration bearer wired in by the catalog auto-attach. The key's
// db:read / db:write scopes and each connection's read_only flag are
// enforced gateway-side; this process adds nothing to the trust story.
//
// Required env vars:
//
//	OPENDRAY_BASE_URL   e.g. http://127.0.0.1:8770 (no trailing slash)
//	OPENDRAY_API_KEY    bearer token (admin or integration with db scopes)
//
// Optional env vars:
//
//	OPENDRAY_DBTOOL_CWD           project key (the session's cwd)
//	OPENDRAY_DBTOOL_CWD_FROM_CWD  "1" = derive an unset project key from
//	                              the process cwd (antigravity's shared
//	                              HOME-global config entry)
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
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// runMcpDbtool is the subcommand entry point. Returns a process exit code.
func runMcpDbtool(args []string) int {
	fs := flag.NewFlagSet("opendray mcp-dbtool", flag.ExitOnError)
	_ = fs.Parse(args)

	cfg, err := loadDbtoolMCPConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	srv := &dbtoolMCPServer{
		cfg: cfg,
		// The stdin loop dispatches synchronously, so a hung gateway
		// would otherwise freeze the whole MCP session — bound every
		// call so a stall surfaces as a tool error instead.
		client: &http.Client{Timeout: 60 * time.Second},
		out:    os.Stdout,
		outMu:  &sync.Mutex{},
		errLog: os.Stderr,
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1<<14), 1<<24)
	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		srv.handle(raw)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "mcp-dbtool: stdin: %v\n", err)
		return 1
	}
	return 0
}

type dbtoolMCPConfig struct {
	baseURL string
	apiKey  string
	cwd     string
}

func loadDbtoolMCPConfig() (dbtoolMCPConfig, error) {
	c := dbtoolMCPConfig{
		baseURL: strings.TrimRight(os.Getenv("OPENDRAY_BASE_URL"), "/"),
		apiKey:  os.Getenv("OPENDRAY_API_KEY"),
		cwd:     os.Getenv("OPENDRAY_DBTOOL_CWD"),
	}
	if c.baseURL == "" {
		return c, errors.New("mcp-dbtool: OPENDRAY_BASE_URL is required")
	}
	if c.apiKey == "" {
		return c, errors.New("mcp-dbtool: OPENDRAY_API_KEY is required")
	}
	// Same explicit opt-in as mcp-memory's SCOPE_FROM_CWD: antigravity's
	// entry lives in a HOME-global config shared by every session, so the
	// project key can't be baked in — but agy spawns MCP subprocesses
	// from the session workspace, so Getwd IS the session's project.
	if os.Getenv("OPENDRAY_DBTOOL_CWD_FROM_CWD") == "1" && c.cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			c.cwd = wd
		}
	}
	return c, nil
}

type dbtoolMCPServer struct {
	cfg    dbtoolMCPConfig
	client *http.Client

	out    io.Writer
	outMu  *sync.Mutex
	errLog io.Writer
}

func (s *dbtoolMCPServer) handle(raw []byte) {
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
				"name":    "opendray-dbtool",
				"version": "0.1.0",
			},
			"instructions": dbtoolInstructionsBlurb,
		})
	case "notifications/initialized":
	case "ping":
		s.respond(req.ID, map[string]any{})
	case "tools/list":
		s.respond(req.ID, map[string]any{"tools": dbtoolToolDefs})
	case "tools/call":
		s.handleToolCall(req)
	default:
		s.respondErr(req.ID, -32601, "Method not found", req.Method)
	}
}

const dbtoolInstructionsBlurb = `Direct access to this project's registered databases (the operator
configures connections in opendray's Database tab; you cannot add or
remove connections).

  db_connections_list  ALWAYS CALL FIRST — the project's registered
                       connections and their ids. Empty list = the
                       project has no databases configured; say so
                       instead of guessing credentials.
  db_schema            introspect: schemas → tables → one table's
                       columns/PK/indexes/FKs, depending on which
                       arguments you pass.
  db_table_data        page through a table (limit/offset/sort/filters)
                       without writing SQL.
  db_query             run ONE read-only SQL statement (SELECT/EXPLAIN/
                       SHOW…). Executes inside a READ ONLY transaction.
  db_execute           run ONE mutating statement (INSERT/UPDATE/DELETE/
                       DDL). Requires the db:write scope and a writable
                       connection — connections marked read-only refuse
                       this. Confirm intent before destructive changes.

Rules: one statement per call (no "a; b"), no BEGIN/COMMIT (every call
already runs in a managed transaction), results are row-capped — check
"truncated" before claiming you saw everything.`

var dbtoolToolDefs = []map[string]any{
	{
		"name": "db_connections_list",
		"description": "List the current project's registered database " +
			"connections (id, name, engine, host, database, read_only). " +
			"Call this first — every other db_* tool needs a connection_id " +
			"from here. An empty list means the operator has not configured " +
			"any database for this project.",
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		"name": "db_schema",
		"description": "Introspect a connection's structure. With only " +
			"connection_id: list schemas. With schema: list that schema's " +
			"tables and views. With schema + table: full table metadata " +
			"(columns, primary key, indexes, foreign keys).",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{
					"type":        "string",
					"description": "Connection id from db_connections_list.",
				},
				"schema": map[string]any{
					"type":        "string",
					"description": "Schema name (optional).",
				},
				"table": map[string]any{
					"type":        "string",
					"description": "Table name (optional; requires schema).",
				},
			},
			"required": []string{"connection_id"},
		},
	},
	{
		"name": "db_table_data",
		"description": "Read rows from one table with paging, sorting and " +
			"simple filters — no SQL needed. Results are row-capped; check " +
			"\"truncated\".",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
				"schema":        map[string]any{"type": "string"},
				"table":         map[string]any{"type": "string"},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Rows per page (default 100).",
				},
				"offset": map[string]any{"type": "integer"},
				"sort": map[string]any{
					"type":        "array",
					"description": "Order terms, applied in sequence.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"column": map[string]any{"type": "string"},
							"desc":   map[string]any{"type": "boolean"},
						},
						"required": []string{"column"},
					},
				},
				"filters": map[string]any{
					"type":        "array",
					"description": "ANDed predicates.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"column": map[string]any{"type": "string"},
							"op": map[string]any{
								"type": "string",
								"description": "One of =, !=, <, >, <=, >=, LIKE, ILIKE, " +
									"NOT LIKE, NOT ILIKE, IS NULL, IS NOT NULL.",
							},
							"value": map[string]any{
								"description": "Comparison value (omit for IS NULL / IS NOT NULL).",
							},
						},
						"required": []string{"column", "op"},
					},
				},
			},
			"required": []string{"connection_id", "schema", "table"},
		},
	},
	{
		"name": "db_query",
		"description": "Run ONE read-only SQL statement (SELECT, EXPLAIN, " +
			"SHOW, VALUES, TABLE, or a non-mutating WITH). Executes inside " +
			"a READ ONLY transaction with a statement timeout — mutating " +
			"statements are rejected; use db_execute for those. One " +
			"statement per call.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
				"sql":           map[string]any{"type": "string"},
				"max_rows": map[string]any{
					"type":        "integer",
					"description": "Row cap for the result (default 500, max 10000).",
				},
			},
			"required": []string{"connection_id", "sql"},
		},
	},
	{
		"name": "db_execute",
		"description": "Run ONE mutating SQL statement (INSERT, UPDATE, " +
			"DELETE, DDL) against a WRITABLE connection. This CHANGES THE " +
			"TARGET DATABASE — be certain of the statement, prefer a WHERE " +
			"clause you have verified with db_query first, and never guess " +
			"table names. Fails on connections the operator marked " +
			"read-only or without the db:write scope. One statement per " +
			"call; no transaction control.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
				"sql":           map[string]any{"type": "string"},
			},
			"required": []string{"connection_id", "sql"},
		},
	},
}

func (s *dbtoolMCPServer) handleToolCall(req rpcRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.respondErr(req.ID, -32602, "Invalid params", err.Error())
		return
	}
	if len(params.Arguments) == 0 {
		params.Arguments = json.RawMessage("{}")
	}

	var result any
	var err error
	switch params.Name {
	case "db_connections_list":
		result, err = s.callConnectionsList()
	case "db_schema":
		result, err = s.callSchema(params.Arguments)
	case "db_table_data":
		result, err = s.callTableData(params.Arguments)
	case "db_query", "db_execute":
		// Both forward to the same gateway endpoint; the gateway
		// classifies the statement and applies the matching gate, so
		// the tools can't drift from the REST behaviour.
		result, err = s.callQuery(params.Arguments)
	default:
		s.respondErr(req.ID, -32601, "Unknown tool", params.Name)
		return
	}

	if err != nil {
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

// cwdQuery returns the "?cwd=…" project-binding parameter the gateway
// requires from non-admin callers on every dbtool call, or "" when no
// project key is bound (which will make the gateway reject the call —
// the surfaced 403 tells the agent the session has no project context).
func (s *dbtoolMCPServer) cwdQuery() string {
	if s.cfg.cwd == "" {
		return ""
	}
	return "?cwd=" + url.QueryEscape(s.cfg.cwd)
}

func (s *dbtoolMCPServer) callConnectionsList() (any, error) {
	var out struct {
		Connections []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Driver   string `json:"driver"`
			Host     string `json:"host"`
			Port     int    `json:"port"`
			DBName   string `json:"db_name"`
			ReadOnly bool   `json:"read_only"`
		} `json:"connections"`
	}
	if err := s.gatewayGetJSON("/api/v1/dbtool/connections"+s.cwdQuery(), &out); err != nil {
		return nil, err
	}
	var b strings.Builder
	if len(out.Connections) == 0 {
		b.WriteString("(no database connections configured for this project)")
	} else {
		fmt.Fprintf(&b, "%d connection(s):\n\n", len(out.Connections))
		for _, c := range out.Connections {
			mode := "read-write"
			if c.ReadOnly {
				mode = "READ-ONLY"
			}
			fmt.Fprintf(&b, "- id=%s  %s  (%s @ %s:%d/%s, %s)\n",
				c.ID, c.Name, c.Driver, c.Host, c.Port, c.DBName, mode)
		}
	}
	return textResult(b.String()), nil
}

func (s *dbtoolMCPServer) callSchema(args json.RawMessage) (any, error) {
	var in struct {
		ConnectionID string `json:"connection_id"`
		Schema       string `json:"schema"`
		Table        string `json:"table"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if in.ConnectionID == "" {
		return nil, errors.New("connection_id is required")
	}
	base := "/api/v1/dbtool/connections/" + url.PathEscape(in.ConnectionID)
	q := s.cwdQuery()
	switch {
	case in.Schema == "":
		var out struct {
			Schemas []struct {
				Name string `json:"name"`
			} `json:"schemas"`
		}
		if err := s.gatewayGetJSON(base+"/schemas"+q, &out); err != nil {
			return nil, err
		}
		names := make([]string, 0, len(out.Schemas))
		for _, sc := range out.Schemas {
			names = append(names, sc.Name)
		}
		return textResult("schemas: " + strings.Join(names, ", ")), nil
	case in.Table == "":
		var out json.RawMessage
		if err := s.gatewayGetJSON(base+"/schemas/"+url.PathEscape(in.Schema)+"/tables"+q, &out); err != nil {
			return nil, err
		}
		return jsonResult(out), nil
	default:
		var out json.RawMessage
		if err := s.gatewayGetJSON(base+"/schemas/"+url.PathEscape(in.Schema)+
			"/tables/"+url.PathEscape(in.Table)+"/meta"+q, &out); err != nil {
			return nil, err
		}
		return jsonResult(out), nil
	}
}

func (s *dbtoolMCPServer) callTableData(args json.RawMessage) (any, error) {
	var in struct {
		ConnectionID string          `json:"connection_id"`
		Schema       string          `json:"schema"`
		Table        string          `json:"table"`
		Limit        int             `json:"limit"`
		Offset       int             `json:"offset"`
		Sort         json.RawMessage `json:"sort"`
		Filters      json.RawMessage `json:"filters"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if in.ConnectionID == "" || in.Schema == "" || in.Table == "" {
		return nil, errors.New("connection_id, schema and table are required")
	}
	body := map[string]any{
		"schema": in.Schema,
		"table":  in.Table,
		"limit":  in.Limit,
		"offset": in.Offset,
	}
	if len(in.Sort) > 0 {
		body["sort"] = in.Sort
	}
	if len(in.Filters) > 0 {
		body["filters"] = in.Filters
	}
	var out json.RawMessage
	if err := s.gatewayPostJSON("/api/v1/dbtool/connections/"+
		url.PathEscape(in.ConnectionID)+"/table-data"+s.cwdQuery(), body, &out); err != nil {
		return nil, err
	}
	return jsonResult(out), nil
}

func (s *dbtoolMCPServer) callQuery(args json.RawMessage) (any, error) {
	var in struct {
		ConnectionID string `json:"connection_id"`
		SQL          string `json:"sql"`
		MaxRows      int    `json:"max_rows"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if in.ConnectionID == "" {
		return nil, errors.New("connection_id is required")
	}
	if strings.TrimSpace(in.SQL) == "" {
		return nil, errors.New("sql is required")
	}
	body := map[string]any{"sql": in.SQL}
	if in.MaxRows > 0 {
		body["max_rows"] = in.MaxRows
	}
	var out json.RawMessage
	if err := s.gatewayPostJSON("/api/v1/dbtool/connections/"+
		url.PathEscape(in.ConnectionID)+"/query"+s.cwdQuery(), body, &out); err != nil {
		return nil, err
	}
	return jsonResult(out), nil
}

func textResult(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	}
}

// jsonResult pretty-prints a raw gateway response as the tool text.
func jsonResult(raw json.RawMessage) map[string]any {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return textResult(string(raw))
	}
	return textResult(buf.String())
}

// ---- gateway HTTP (same thin forwarding as mcp-memory) ----

func (s *dbtoolMCPServer) gatewayPostJSON(path string, body, out any) error {
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
	return s.doJSON(req, path, out)
}

func (s *dbtoolMCPServer) gatewayGetJSON(path string, out any) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, s.cfg.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.apiKey)
	return s.doJSON(req, path, out)
}

func (s *dbtoolMCPServer) doJSON(req *http.Request, path string, out any) error {
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

func (s *dbtoolMCPServer) respond(id json.RawMessage, result any) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *dbtoolMCPServer) respondErr(id json.RawMessage, code int, msg string, data any) {
	s.write(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg, Data: data}})
}

func (s *dbtoolMCPServer) write(resp rpcResponse) {
	raw, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(s.errLog, "mcp-dbtool: encode response: %v\n", err)
		return
	}
	s.outMu.Lock()
	defer s.outMu.Unlock()
	_, _ = s.out.Write(raw)
	_, _ = s.out.Write([]byte{'\n'})
}
