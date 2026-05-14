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
//	AgentWorker      — spawns a headless Claude/Gemini agent in
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
)

// AllTasks returns every recognised TaskKind in a stable order.
// Used by the UI to render the config rows + by registry bootstrap
// to seed missing entries.
func AllTasks() []TaskKind {
	return []TaskKind{
		TaskGatekeeper, TaskCleaner, TaskGitActivity, TaskTranscript,
		TaskPlanDrift, TaskConflictDetector,
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
}

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
	ProviderID string // "claude" / "gemini" / summarizer-row id
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
	// agent provider that doesn't support --print mode (today:
	// codex). The UI should validate before saving but we
	// double-check defensively.
	ErrAgentUnsupported = errors.New("memory worker: agent provider unsupported in --print mode")
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
	ProviderID   string     `json:"provider_id"`   // when Kind==WorkerAgent: "claude" | "gemini"
	AccountID    string     `json:"account_id"`    // when ProviderID=="claude"; "" → catalog's default account
	Enabled      bool       `json:"enabled"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Valid returns nil if the config is internally consistent.
// Caller (HTTP handler) should run this before INSERT/UPDATE.
func (c Config) Valid() error {
	switch c.Task {
	case TaskGatekeeper, TaskCleaner, TaskGitActivity, TaskTranscript,
		TaskPlanDrift, TaskConflictDetector:
	default:
		return errors.New("memory worker: invalid task")
	}
	switch c.Kind {
	case WorkerSummarizer:
		return nil
	case WorkerAgent:
		switch c.ProviderID {
		case "claude", "gemini":
			return nil
		case "codex":
			return ErrAgentUnsupported
		default:
			return errors.New("memory worker: agent provider_id required (claude or gemini)")
		}
	default:
		return errors.New("memory worker: invalid kind")
	}
}
