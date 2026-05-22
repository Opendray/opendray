package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Check is one validation step (config sanity, reachability, handshake).
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// ValidationResult is the outcome of testing one MCP server — rendered
// by the Plugins UI as a Cursor-style "✓ connected · N tools" / "✗ …".
type ValidationResult struct {
	OK            bool     `json:"ok"`
	Transport     string   `json:"transport"`
	Checks        []Check  `json:"checks"`
	ToolCount     int      `json:"toolCount,omitempty"`
	Tools         []string `json:"tools,omitempty"`
	ServerName    string   `json:"serverName,omitempty"`
	ServerVersion string   `json:"serverVersion,omitempty"`
	Note          string   `json:"note,omitempty"`
	MissingEnv    []string `json:"missingEnv,omitempty"`
	LatencyMs     int64    `json:"latencyMs,omitempty"`
}

// Validate checks one MCP server from the daemon's vantage point (so it
// sees the same PATH + network a real session spawn would). stdio
// servers get a full MCP handshake (initialize + tools/list); sse/http
// servers get config-sanity + URL reachability (a full remote handshake
// is a later addition). `missing` is the set of unresolved ${SECRET}
// placeholders from Secrets.Resolve.
func Validate(ctx context.Context, srv Server, missing []string) ValidationResult {
	transport := strings.TrimSpace(srv.Transport)
	if transport == "" {
		transport = "stdio"
	}
	res := ValidationResult{Transport: transport, MissingEnv: missing}
	add := func(name string, ok bool, detail string) { res.Checks = append(res.Checks, Check{name, ok, detail}) }

	switch transport {
	case "sse", "http":
		// Config sanity — the #221 trap: address belongs in `url`.
		if strings.TrimSpace(srv.URL) == "" {
			detail := "sse/http transport needs a `url`"
			if strings.TrimSpace(srv.Command) != "" {
				detail = "address looks like it's in the `command` field — move it to `url`"
			}
			add("config", false, detail)
			res.OK = false
			return res
		}
		add("config", true, "url set")
		if len(missing) > 0 {
			add("secrets", false, "unresolved placeholders: "+strings.Join(missing, ", "))
		}
		// Reachability.
		start := time.Now()
		status, err := urlReachable(ctx, srv.URL)
		res.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			add("reachable", false, err.Error())
			res.OK = false
		} else {
			add("reachable", true, fmt.Sprintf("HTTP %d", status))
			res.OK = len(missing) == 0
		}
		res.Note = "Reachability only (no live handshake for sse/http yet). " +
			"Note: codex is stdio-only and won't see this server."
		return res

	default: // stdio
		if strings.TrimSpace(srv.Command) == "" {
			add("config", false, "stdio transport needs a `command`")
			res.OK = false
			return res
		}
		add("config", true, "command set")
		// Command resolvable on the daemon's PATH?
		path, err := exec.LookPath(srv.Command)
		if err != nil {
			add("command", false, fmt.Sprintf("%q not found on the service PATH", srv.Command))
			res.OK = false
			return res
		}
		add("command", true, path)
		if len(missing) > 0 {
			add("secrets", false, "unresolved placeholders: "+strings.Join(missing, ", "))
		}
		// Full MCP handshake.
		start := time.Now()
		hs, err := stdioHandshake(ctx, srv)
		res.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			add("handshake", false, err.Error())
			res.OK = false
			return res
		}
		res.ToolCount = len(hs.tools)
		res.Tools = hs.tools
		res.ServerName = hs.name
		res.ServerVersion = hs.version
		add("handshake", true, fmt.Sprintf("connected, %d tools", len(hs.tools)))
		res.OK = len(missing) == 0
		return res
	}
}

type handshakeResult struct {
	tools         []string
	name, version string
}

// stdioHandshake spawns the server and runs initialize + tools/list over
// stdio, returning the advertised tools. The process is always killed.
func stdioHandshake(ctx context.Context, srv Server) (handshakeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, srv.Command, srv.Args...)
	cmd.Env = append(os.Environ(), envSlice(srv.Env)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return handshakeResult{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return handshakeResult{}, err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return handshakeResult{}, fmt.Errorf("start: %w", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// stdout is the MCP channel; logs go to stderr (we keep it for errors).
	for _, msg := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"opendray-validator","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	} {
		if _, err := io.WriteString(stdin, msg+"\n"); err != nil {
			return handshakeResult{}, fmt.Errorf("write request: %w", err)
		}
	}

	var out handshakeResult
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1<<20), 8<<20) // tool lists can be large
	for sc.Scan() {
		var msg struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(sc.Bytes(), &msg) != nil || msg.ID == nil {
			continue // notifications / log lines / partial frames
		}
		switch *msg.ID {
		case 1:
			var init struct {
				ServerInfo struct{ Name, Version string } `json:"serverInfo"`
			}
			_ = json.Unmarshal(msg.Result, &init)
			out.name, out.version = init.ServerInfo.Name, init.ServerInfo.Version
		case 2:
			if msg.Error != nil {
				return out, errors.New(msg.Error.Message)
			}
			var tl struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			}
			if err := json.Unmarshal(msg.Result, &tl); err != nil {
				return out, fmt.Errorf("parse tools/list: %w", err)
			}
			for _, t := range tl.Tools {
				out.tools = append(out.tools, t.Name)
			}
			return out, nil
		}
	}
	if ctx.Err() != nil {
		return out, fmt.Errorf("timed out waiting for MCP response%s", stderrHint(stderr.String()))
	}
	return out, fmt.Errorf("server exited before answering tools/list%s", stderrHint(stderr.String()))
}

// urlReachable does a GET with a short timeout. http.Client returns once
// headers arrive, so an SSE endpoint that streams forever still reports
// promptly. Any status is "reachable"; a transport error is not.
func urlReachable(ctx context.Context, rawurl string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}

func envSlice(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

func stderrHint(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	return " — stderr: " + strings.Join(lines, " | ")
}
