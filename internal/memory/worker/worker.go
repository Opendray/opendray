// Package worker defines a pluggable execution surface for the
// memory subsystem's LLM touchpoints (M25).
//
// Today four operations need to call a model:
//
//	gatekeeper   — pre-store classification, every memory_store
//	cleaner      — periodic LLM librarian (24h tick)
//	gitactivity  — git log → narrative summary (24h tick)
//	transcript   — per-session-end "what did the agent do" summary
//
// Before M25 each of these hardcoded a path through summarizer.
// Registry → HTTP → OpenAI-compatible endpoint (typically LM
// Studio). That's fine for high-frequency low-quality work
// (gatekeeper) but limits the long narrative tasks to whatever
// small local model the operator runs.
//
// M25 abstracts the call shape behind Worker so operators can
// route each touchpoint independently to either:
//
//	SummarizerWorker — the existing summarizer.Registry path.
//	                   Cheap, low-latency, local-private.
//	AgentWorker      — spawns a headless Claude/Antigravity agent in
//	                   `--print` mode for one-shot judgement.
//	                   Higher quality, higher latency, costs
//	                   API tokens / agent quota.
//
// Per-task configuration lives in the memory_workers table (see
// migration 0029). Defaults seed all tasks to SummarizerWorker so
// existing deployments behave identically until an operator opts
// into agents on a touchpoint.
package worker

import (
	"context"
	"errors"
	"time"
)

// TaskKind enumerates the four memory-system touchpoints that
// need an LLM. The string values are the persisted enum (see
// memory_workers.task CHECK constraint in migration 0029).
type TaskKind string

const (
	TaskGatekeeper       TaskKind = "gatekeeper"
	TaskCleaner          TaskKind = "cleaner"
	TaskGitActivity      TaskKind = "gitactivity"
	TaskTranscript       TaskKind = "transcript"
	TaskPlanDrift        TaskKind = "plan_drift"
	TaskConflictDetector TaskKind = "conflict_detector"
	// TaskCapture is the capture-engine touchpoint. Pre-M-PE the
	// capture engine talked to summarizer.Registry directly so it
	// could only use HTTP summarizer providers. Routing through
	// worker.Registry lets operators pick a headless Claude /
	// Antigravity agent for capture (higher quality but slower and
	// more expensive), matching the other 5 touchpoints.
	TaskCapture TaskKind = "capture"
	// TaskBlueprint is the Cortex doc-blueprint proposer: classify a
	// project from repo signals, propose a doc section set
	// (operator-triggered, applied only on operator accept).
	TaskBlueprint TaskKind = "blueprint"
	// TaskCuration is the Cortex curation-conversation channel: the
	// operator discusses a doc section or knowledge page with the AI
	// and the AI produces a revision (apply or proposal).
	TaskCuration TaskKind = "curation"
)

// AllTasks returns every recognised TaskKind in a stable order.
// Used by the UI to render the config rows + by registry bootstrap
// to seed missing entries.
func AllTasks() []TaskKind {
	return []TaskKind{
		TaskGatekeeper, TaskCleaner, TaskGitActivity, TaskTranscript,
		TaskPlanDrift, TaskConflictDetector, TaskCapture,
		TaskBlueprint, TaskCuration,
	}
}

// WorkerKind names the implementation strategy. Persisted as the
// memory_workers.kind column.
type WorkerKind string

const (
	WorkerSummarizer WorkerKind = "summarizer"
	WorkerAgent      WorkerKind = "agent"
)

// Request is the payload Worker implementations need to run one
// LLM call. Higher-level subsystems (gatekeeper, cleaner, …)
// build Request and hand it to the configured Worker without
// caring whether the underlying call is HTTP or a spawned agent.
type Request struct {
	// Task identifies which memory-system touchpoint is calling.
	// Used for routing (look up the right row in memory_workers)
	// and for metrics (memory_worker_calls.task).
	Task TaskKind

	// SystemPrompt is the role / instruction block. Summarizer
	// workers send this as the "system" message; agent workers
	// pass it via --append-system-prompt.
	SystemPrompt string

	// UserInput is the actual content to judge / summarise.
	UserInput string

	// MaxTokens caps the model's output. Summarizer workers
	// forward this as max_tokens; agent workers can't enforce
	// directly but use it as an output-size advisory.
	MaxTokens int

	// Timeout is the hard cap on the whole call. AgentWorker
	// uses this as the spawn timeout (kills the process); the
	// SummarizerWorker uses it as the HTTP timeout.
	Timeout time.Duration

	// ResponseFormatJSONSchema, when non-empty, asks the model
	// to return structured JSON conforming to this schema.
	// Summarizer workers translate this into the OpenAI-spec
	// response_format=json_schema field. Agent workers append
	// schema instructions to the system prompt instead (since
	// agent CLIs don't natively support response_format).
	ResponseFormatJSONSchema string

	// MCP, when non-nil, attaches MCP tool servers to this headless call.
	// The AgentWorker runs MCP.Provision after creating its scratch dir to
	// render the provider-specific config, applies the returned args + env,
	// and adds the per-provider flags that let the CLI execute the tools in
	// headless mode. Nil (the default) keeps the deliberately tool-less
	// one-shot behaviour every memory-touchpoint worker relies on.
	MCP *MCPAttach
}

// MCPAttach carries the per-call MCP intent for an AgentWorker run. It holds
// a caller-supplied closure rather than a concrete server list so the worker
// package stays decoupled from the catalog rendering machinery (which owns
// every provider's config-file format).
type MCPAttach struct {
	// Cwd is the memory scope key — the project whose facts / journal the
	// agent should observe. It is also the working directory the CLI runs in
	// for providers that derive scope from the subprocess cwd (antigravity);
	// other providers run in the scratch dir and get the scope via env.
	Cwd string
	// Provision renders the MCP config for providerID into baseDir and
	// returns the extra CLI args + env. It is catalog.AttachMemoryMCP bound
	// to the memory config + read-only flag. runCwd is where the CLI will
	// run; scopeKey is the memory scope; home is the effective HOME.
	Provision MCPProvisionFunc
}

// MCPProvisionFunc renders a provider's MCP config and returns the extra CLI
// args + env needed to load it. See catalog.AttachMemoryMCP.
type MCPProvisionFunc func(providerID, baseDir, runCwd, scopeKey, home string) (args []string, env map[string]string, err error)

// Response is what every Worker returns on success. Latency +
// token counts are best-effort: AgentWorker can't always get
// reliable token info, so callers should treat them as hints.
type Response struct {
	Content    string
	DurationMS int64
	TokensIn   int
	TokensOut  int

	// Provenance metadata for metrics / UI.
	WorkerKind WorkerKind
	ProviderID string // "claude" / "antigravity" / summarizer-row id
	AccountID  string // empty for summarizer
}

// Sentinel errors callers can use to drive UX. Most failures
// just bubble up as wrapped errors; these flag the well-known
// degraded states.
var (
	// ErrNoWorkerConfigured means the memory_workers row for
	// this task is missing or disabled. Callers should treat
	// the touchpoint as "skip" and emit a metadata-only result
	// rather than calling some default fallback (operators
	// explicitly disabled it, respect that).
	ErrNoWorkerConfigured = errors.New("memory worker: no worker configured for task")

	// ErrAgentUnsupported is returned when an operator picked an
	// agent provider with no headless worker path (today: opencode).
	// The UI should validate before saving but we double-check
	// defensively.
	ErrAgentUnsupported = errors.New("memory worker: agent provider unsupported in headless mode")
)

// Worker is the single-method interface every implementation
// satisfies. Run is synchronous; callers manage their own
// background-goroutine semantics where needed (the journaler
// already does this for transcript summarisation).
type Worker interface {
	Kind() WorkerKind
	Run(ctx context.Context, req Request) (Response, error)
}

// Config carries the per-task configuration read from
// memory_workers. The Resolver builds a Worker from this.
type Config struct {
	Task         TaskKind   `json:"task"`
	Kind         WorkerKind `json:"kind"`
	SummarizerID string     `json:"summarizer_id"` // when Kind==WorkerSummarizer; "" → registry default
	ProviderID   string     `json:"provider_id"`   // when Kind==WorkerAgent: claude|codex|antigravity|grok|opencode
	AccountID    string     `json:"account_id"`    // multi-account pin (claude|antigravity); "" → CLI's default account
	// Model pins the agent CLI's model (`claude --model …` /
	// `agy --model …`) so cheap chores run on cheap models
	// (e.g. haiku) and only the judgement-heavy touchpoints pay for
	// frontier quality. Empty keeps the CLI default. Ignored for
	// summarizer-kind workers (their model lives on the provider row).
	Model     string    `json:"model"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Valid returns nil if the config is internally consistent.
// Caller (HTTP handler) should run this before INSERT/UPDATE.
func (c Config) Valid() error {
	switch c.Task {
	case TaskGatekeeper, TaskCleaner, TaskGitActivity, TaskTranscript,
		TaskPlanDrift, TaskConflictDetector, TaskCapture,
		TaskBlueprint, TaskCuration:
	default:
		return errors.New("memory worker: invalid task")
	}
	switch c.Kind {
	case WorkerSummarizer:
		return nil
	case WorkerAgent:
		// Must match AgentWorker.Run's headless provider switch.
		switch c.ProviderID {
		case "claude", "codex", "antigravity", "grok", "opencode":
			return nil
		default:
			return errors.New("memory worker: agent provider_id required (claude, codex, antigravity, grok, or opencode)")
		}
	default:
		return errors.New("memory worker: invalid kind")
	}
}
