package cleaner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/worker"
)

// Verdict is the LLM's per-memory judgement.
type Verdict string

const (
	VerdictKeep      Verdict = "keep"
	VerdictStale     Verdict = "stale"
	VerdictDuplicate Verdict = "duplicate"
)

// ValidVerdict returns true for the three supported verdicts.
func ValidVerdict(v Verdict) bool {
	switch v {
	case VerdictKeep, VerdictStale, VerdictDuplicate:
		return true
	}
	return false
}

// BatchItem is one memory passed to the LLM for review.
type BatchItem struct {
	ID        string
	Text      string
	CreatedAt time.Time
	HitCount  int64
}

// llmDecision is the per-item judgement the LLM returns.
type llmDecision struct {
	MemoryID  string  `json:"memory_id"`
	Verdict   Verdict `json:"verdict"`
	Reason    string  `json:"reason"`
	MergeInto *string `json:"merge_into,omitempty"`
}

// Client wraps a worker.Registry with the cleaner-specific prompt
// + JSON-schema enforcement. M25 — replaced the direct HTTP
// client with a Registry dispatch so the cleaner touchpoint
// independently picks between the summarizer HTTP path and a
// headless Claude/Gemini agent based on the operator's choice in
// memory_workers.cleaner.
//
// The DecisionsJSONSchema constant is plumbed through worker.Request
// to enforce JSON-mode output (response_format=json_schema for the
// summarizer worker; appended-to-prompt schema for the agent
// worker since agent CLIs don't natively support response_format).
type Client struct {
	registry *worker.Registry
}

// NewClient builds a Client that dispatches through the given
// worker registry. Returns nil when registry is nil so callers
// can short-circuit gracefully.
func NewClient(reg *worker.Registry) *Client {
	if reg == nil {
		return nil
	}
	return &Client{registry: reg}
}

// Judge sends the whole batch to the registry-selected worker and
// returns the parsed per-item decisions. The returned slice has
// the same length as items unless the model refused / drifted, in
// which case missing memories are reported by the caller as
// "no decision".
//
// timeout caps the underlying worker call.
func (c *Client) Judge(ctx context.Context, items []BatchItem, timeout time.Duration) ([]llmDecision, error) {
	if c == nil || c.registry == nil {
		return nil, errors.New("cleaner: nil client / registry")
	}
	if len(items) == 0 {
		return nil, nil
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	resp, err := c.registry.Run(ctx, worker.Request{
		Task:                     worker.TaskCleaner,
		SystemPrompt:             SystemPrompt(),
		UserInput:                renderBatch(items),
		MaxTokens:                4096,
		Timeout:                  timeout,
		ResponseFormatJSONSchema: DecisionsJSONSchema,
	})
	if err != nil {
		return nil, fmt.Errorf("cleaner: worker call: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return nil, errors.New("cleaner: worker returned empty content")
	}
	// Agent workers may wrap output in markdown fences — strip them.
	content = stripJSONFence(content)

	var parsed struct {
		Decisions []llmDecision `json:"decisions"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("cleaner: parse llm decisions: %w (content=%s)", err, truncate(content, 400))
	}
	// Drop malformed entries quietly — the caller treats "missing
	// memory id in decisions" as "LLM refused; skip and try later".
	out := make([]llmDecision, 0, len(parsed.Decisions))
	for _, d := range parsed.Decisions {
		if d.MemoryID == "" || !ValidVerdict(d.Verdict) {
			continue
		}
		if d.Verdict == VerdictDuplicate && (d.MergeInto == nil || *d.MergeInto == "") {
			// duplicate verdict without merge_into is uninterpretable;
			// downgrade to keep so we don't drop something blindly.
			d.Verdict = VerdictKeep
			d.Reason = "downgraded from duplicate (no merge_into): " + d.Reason
			d.MergeInto = nil
		}
		out = append(out, d)
	}
	return out, nil
}

// renderBatch builds the USER message — one numbered block per
// memory. Includes hit_count + creation date so the LLM has
// signal to prioritise high-hit memories for "keep".
func renderBatch(items []BatchItem) string {
	var b strings.Builder
	for i, it := range items {
		fmt.Fprintf(&b, "[%d] %s | created %s | hit_count=%d\n    %s\n\n",
			i+1, it.ID,
			it.CreatedAt.UTC().Format("2006-01-02"),
			it.HitCount,
			strings.ReplaceAll(strings.TrimSpace(it.Text), "\n", " "),
		)
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// stripJSONFence removes ```json ... ``` or ``` ... ``` wrappers
// that agent CLIs sometimes add around JSON output. Idempotent on
// already-clean input.
func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Drop the opening fence (with optional language tag).
	if nl := strings.IndexByte(s, '\n'); nl != -1 {
		s = s[nl+1:]
	}
	if i := strings.LastIndex(s, "```"); i != -1 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
