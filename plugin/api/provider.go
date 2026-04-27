package api

import "context"

// Provider is an LLM provider — the abstraction over Anthropic, OpenAI,
// Gemini, Azure, custom OpenAI-compatible endpoints, on-device
// runtimes, etc.
//
// One plugin may register one or many Providers. A vendor plugin
// (e.g. "anthropic") typically registers a single Provider whose
// Models() enumerates that vendor's lineup; an aggregator plugin
// (e.g. "openrouter") may also register a single Provider but expose
// many models through it.
//
// Stream is the only invocation surface in v1. Non-streaming completion
// is modelled as a stream that emits one chunk; this keeps the contract
// narrow.
type Provider interface {
	// ID is the stable registry key (e.g. "anthropic", "openai").
	// MUST equal the id declared in manifest.contributes.providers[].id.
	ID() string

	// Models lists model ids this provider currently exposes. Called
	// by the model picker UI and by the routing layer when a session
	// requests a model by short name. Implementations may cache; the
	// host calls this on a cold-cache schedule it controls.
	Models(ctx context.Context) ([]ProviderModel, error)

	// Stream initiates a streaming completion. Implementations write
	// chunks to the returned channel and close it when the response
	// terminates (whether by stop reason, error, or context cancel).
	//
	// Cancelling ctx MUST abort the upstream call and close the
	// channel promptly.
	Stream(ctx context.Context, req ProviderRequest) (<-chan ProviderChunk, error)
}

// ProviderModel describes one model exposed by a Provider.
type ProviderModel struct {
	ID            string   `json:"id"`
	DisplayName   string   `json:"displayName,omitempty"`
	ContextWindow int      `json:"contextWindow,omitempty"`
	MaxTokens     int      `json:"maxTokens,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"` // "text", "vision", "tool-use", "reasoning"
}

// ProviderRequest is the v1 minimum. Multimodal content, advanced
// sampling parameters, and cache hints are reserved for v1.x additions.
type ProviderRequest struct {
	Model       string            `json:"model"`
	Messages    []ProviderMessage `json:"messages"`
	System      string            `json:"system,omitempty"`
	Tools       []ProviderTool    `json:"tools,omitempty"`
	MaxTokens   int               `json:"maxTokens,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
}

// ProviderMessage is a single conversation turn.
type ProviderMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant", "tool"
	Content string `json:"content"`
}

// ProviderTool declares a tool that the model may invoke during the
// turn. Tool dispatch happens above this layer — the provider only
// transports declarations and the model's tool_use chunks back.
type ProviderTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// ProviderChunk is one element of a streaming response. Exactly one
// of the kind-specific fields is populated for each non-error chunk.
type ProviderChunk struct {
	Kind string `json:"kind"` // "text", "tool_call", "stop", "error"

	// Kind == "text"
	Text string `json:"text,omitempty"`

	// Kind == "tool_call"
	ToolCall *ProviderToolCall `json:"toolCall,omitempty"`

	// Kind == "stop"
	Stop *ProviderStop `json:"stop,omitempty"`

	// Kind == "error"
	Err error `json:"-"`
}

// ProviderToolCall is the model's request to invoke a declared tool.
type ProviderToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ProviderStop terminates a stream cleanly with a reason and usage.
type ProviderStop struct {
	Reason string         `json:"reason"` // "end_turn", "max_tokens", "stop_sequence", "tool_use"
	Usage  *ProviderUsage `json:"usage,omitempty"`
}

// ProviderUsage is the token accounting for a completed turn.
type ProviderUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}
