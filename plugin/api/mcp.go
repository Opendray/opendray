package api

import "context"

// McpServer is a Model Context Protocol server provider. A plugin may
// register multiple McpServers — one per backend the plugin manages.
//
// The host is responsible for routing tool calls from agent sessions
// to the right McpServer instance based on the tool's owning server id.
type McpServer interface {
	// ID is the stable registry key (e.g. "filesystem", "github-mcp").
	// MUST equal the id declared in manifest.contributes.mcpServers[].id.
	ID() string

	// Start spawns or connects to the MCP server. Must be ready to
	// accept Tools() and CallTool() requests on return.
	Start(ctx context.Context) error

	// Stop terminates the server and releases all resources.
	Stop(ctx context.Context) error

	// Tools enumerates the tools the server currently exposes. The
	// host caches this and refreshes on plugin reload or explicit
	// invalidation.
	Tools(ctx context.Context) ([]McpTool, error)

	// CallTool invokes a server tool with the given arguments. The
	// host validates args against the tool's InputSchema before
	// dispatching.
	CallTool(ctx context.Context, name string, args map[string]any) (McpToolResult, error)
}

// McpTool is one tool exposed by an MCP server.
type McpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// McpToolResult is the result of a tool invocation.
type McpToolResult struct {
	Content []McpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// McpContent is one content fragment in a tool result.
type McpContent struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
}
