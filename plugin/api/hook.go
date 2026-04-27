package api

import "context"

// HookBus is the host event dispatcher. Plugins subscribe with
// Subscribe(); the host fires events on session lifecycle, tool calls,
// agent turns, message ingress/egress, and gateway lifecycle.
//
// Handlers run sequentially in descending priority; same-priority
// handlers run in registration order. A handler that returns
// HookResult{Block: true} or HookResult{Cancel: true} terminates the
// chain for that event (subsequent subscribers do not run).
//
// HookBus on this plugin's PluginAPI handle is scoped to this plugin:
// all subscriptions are auto-cancelled at plugin unload.
type HookBus interface {
	// Subscribe registers a handler for the named event. The returned
	// Subscription's Cancel() method removes the handler immediately.
	Subscribe(eventName string, handler HookHandler, opts ...HookOption) Subscription
}

// HookHandler is the per-event callback. Returning a zero HookResult
// is the "observe only, do not interfere" default.
type HookHandler func(ctx context.Context, event HookEvent) HookResult

// HookEvent is the dispatched payload. The Payload map's exact keys
// depend on the event name; see the Hook* constants below for documented
// shapes.
type HookEvent struct {
	Name      string         `json:"name"`
	Timestamp int64          `json:"timestamp"` // unix milliseconds
	SessionID string         `json:"sessionId,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// HookResult is the handler's reply. The host inspects the fields in
// this order: Block (terminate with deny), Cancel (terminate without
// continuing), Params (mutate continuing payload). Multiple subscribers
// can mutate Params in turn.
type HookResult struct {
	// Block, when true, stops the action that triggered the event
	// (e.g. block a tool call). BlockReason carries an operator-
	// readable explanation surfaced in logs and UI.
	Block       bool   `json:"block,omitempty"`
	BlockReason string `json:"blockReason,omitempty"`

	// Cancel, when true, stops dispatch without "blocking" semantics
	// (the caller treats it as a soft veto). Mainly useful for
	// outbound message hooks where "do not send" is the right verb.
	Cancel bool `json:"cancel,omitempty"`

	// Params is a mutated payload merged into HookEvent.Payload for
	// downstream subscribers and the originating call.
	Params map[string]any `json:"params,omitempty"`
}

// Subscription is the cancellation handle returned by Subscribe.
type Subscription interface {
	// Cancel removes the handler. Idempotent; safe to call after
	// the host has already cancelled (e.g. at plugin unload).
	Cancel()
}

// HookOption configures a subscription at registration time.
type HookOption func(*HookOptions)

// HookOptions is the materialised form of HookOptions, exposed so
// HookBus implementations can read it. Plugins should use the
// With* helpers and not construct this directly.
type HookOptions struct {
	Priority int
}

// WithPriority sets the subscription priority. Higher priorities run
// first. Default is 0; the host reserves priorities <0 for built-in
// safety hooks.
func WithPriority(p int) HookOption {
	return func(o *HookOptions) { o.Priority = p }
}

// Hook event names. Subscribers should reference these constants
// instead of raw strings so typos surface at compile time.
const (
	// HookBeforeToolCall fires before a tool is invoked. Subscribers
	// may rewrite parameters via HookResult.Params, block execution
	// via HookResult.Block, or observe.
	// Payload: { toolName, params, agentId, runId }
	HookBeforeToolCall = "before_tool_call"

	// HookAfterToolCall fires after a tool returns or errors.
	// Payload: { toolName, params, result, error, durationMs }
	HookAfterToolCall = "after_tool_call"

	// HookBeforeAgentReply fires before the agent's turn is sent to
	// the LLM provider. Subscribers may inject context or block.
	// Payload: { messages, system, model }
	HookBeforeAgentReply = "before_agent_reply"

	// HookMessageReceived fires when a channel ingests an inbound
	// message. Subscribers may observe; cancellation is meaningless.
	// Payload: { channelId, message }
	HookMessageReceived = "message_received"

	// HookMessageSending fires before a channel sends an outbound
	// message. Subscribers may rewrite via Params.text or cancel.
	// Payload: { channelId, message }
	HookMessageSending = "message_sending"

	// HookSessionStart fires when an agent session is created.
	// Payload: { sessionId, providerId, model }
	HookSessionStart = "session_start"

	// HookSessionEnd fires when an agent session terminates.
	// Payload: { sessionId, durationMs, reason }
	HookSessionEnd = "session_end"

	// HookGatewayStart fires once after the gateway has finished
	// loading all plugins. Useful for plugins that own background
	// services.
	HookGatewayStart = "gateway_start"

	// HookGatewayStop fires before the gateway shuts down. Plugins
	// should drain in-flight work here; the host gives a bounded
	// grace window before forcing unload.
	HookGatewayStop = "gateway_stop"

	// HookOutput fires for each stdout/stderr chunk from a CLI
	// session. Compatible with the existing terminal hook bus.
	// Payload: { sessionId, data }
	HookOutput = "output"

	// HookIdle fires when a CLI session has been idle for the
	// configured threshold.
	// Payload: { sessionId, idleMs }
	HookIdle = "idle"
)
