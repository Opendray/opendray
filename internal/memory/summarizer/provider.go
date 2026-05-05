// Package summarizer turns conversation transcript turns into a JSON
// list of durable facts. Phase A ships two providers (Anthropic +
// ollama); Phase B will add OpenAI + an IntegrationProvider that
// lets any opendray-registered integration with the
// memory:summarize scope serve as a summarizer backend.
//
// Wire-level shape:
//
//	[]Message  ──> Provider.Summarize ──> SummarizeResult{Facts, Tokens, USD}
//
// Provider implementations enforce a 30s context deadline and
// return an error rather than partial results — capture engine
// uses the call log to decide retry vs. hard-fail.
package summarizer

import (
	"context"
	"errors"
	"time"
)

// Role is the speaker of a transcript message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message is one transcript turn. Lined up with Manager.History's
// ProjectInput shape but kept independent so the summarizer
// package doesn't import internal/session.
type Message struct {
	Role      Role      `json:"role"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	// ProviderRef is opaque (e.g. claude session UUID + line offset)
	// so dedup state can record "we summarised up to here".
	ProviderRef string `json:"provider_ref,omitempty"`
}

// Category classifies a fact for downstream filtering / display.
// Must match the JSON schema embedded in prompt.go.
type Category string

const (
	CategoryPreference Category = "preference"
	CategoryIdentifier Category = "identifier"
	CategoryDecision   Category = "decision"
	CategoryTask       Category = "task"
	CategoryOther      Category = "other"
)

// Fact is one durable claim extracted by the summarizer.
type Fact struct {
	Text       string   `json:"text"`
	Confidence float32  `json:"confidence"` // 0..1; provider self-reported
	Category   Category `json:"category"`
}

// SummarizeResult is the provider-call return shape.
//
// RawResponse holds the first 4 KiB of the provider's response
// body (truncated; for debugging / call_log audit). Latency is
// measured wall-clock around the HTTP exchange.
type SummarizeResult struct {
	Facts        []Fact
	InputTokens  int
	OutputTokens int
	EstimatedUSD float64
	Latency      time.Duration
	RawResponse  string
}

// Provider is the contract every summarizer LLM implementation
// satisfies. Implementations:
//
//   - enforce a 30s timeout via ctx (or a tighter caller timeout);
//   - return ErrInvalidResponse when the LLM returned malformed JSON;
//   - return ErrUnreachable / ctx.Err() for network failure;
//   - never panic — the capture engine relies on graceful errors.
type Provider interface {
	Summarize(ctx context.Context, msgs []Message) (SummarizeResult, error)
	Name() string
	Kind() string
	Available(ctx context.Context) error
}

// Sentinel errors. Every provider-level error wraps one of these
// so callers can errors.Is() instead of substring-matching.
var (
	ErrInvalidResponse  = errors.New("summarizer: invalid response from provider")
	ErrUnreachable      = errors.New("summarizer: provider unreachable")
	ErrAuthFailed       = errors.New("summarizer: provider authentication failed")
	ErrRateLimited      = errors.New("summarizer: provider rate limited")
	ErrModelNotFound    = errors.New("summarizer: provider does not have the requested model")
	ErrEmptyConversation = errors.New("summarizer: empty conversation, nothing to extract")
)

// truncateRaw caps a raw response body for the call log — we want
// enough bytes to debug a parse failure, not enough to bloat the DB.
const rawResponseCap = 4 * 1024

// truncateRaw is exported as a helper so each provider implementation
// can slice consistently before populating SummarizeResult.RawResponse.
func TruncateRaw(s string) string {
	if len(s) <= rawResponseCap {
		return s
	}
	return s[:rawResponseCap]
}
