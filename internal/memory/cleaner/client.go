package cleaner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/summarizer"
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

// Client wraps an OpenAI-compatible chat completions endpoint with
// the cleaner prompt + JSON-schema enforcement. Built per Run call
// (cheap — just a config struct) from a summarizer.ProviderRow so
// operators can reuse the same provider they configured for the
// gatekeeper.
//
// Why a separate client rather than re-using summarizer.Provider:
// summarizer's Summarize method is locked to the "extract facts"
// prompt, and changing the system prompt path through that
// abstraction would force every provider to re-implement to a
// new shape. The cleaner only needs OpenAI-compatible chat
// completions; that's a small focused HTTP call we keep here.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	kind    string
	http    *http.Client
}

// NewClient builds a Client from a summarizer provider row. Only
// kinds with an OpenAI-compatible chat completions endpoint are
// supported (openai / lmstudio / ollama-with-openai-shim); other
// kinds return an error so the operator gets a clear startup
// failure instead of a silent gatekeeper degradation.
func NewClient(row summarizer.ProviderRow, apiKey string) (*Client, error) {
	switch row.Kind {
	case "openai", "lmstudio":
		// OK — both speak chat completions; lmstudio uses the same
		// /v1/chat/completions path.
	default:
		return nil, fmt.Errorf("cleaner: unsupported provider kind %q (expected openai or lmstudio)", row.Kind)
	}
	base := strings.TrimRight(row.BaseURL, "/")
	if base == "" {
		return nil, errors.New("cleaner: provider has no base_url")
	}
	if row.Model == "" {
		return nil, errors.New("cleaner: provider has no model")
	}
	return &Client{
		baseURL: base,
		apiKey:  apiKey,
		model:   row.Model,
		kind:    row.Kind,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Judge sends the whole batch to the LLM and returns the parsed
// per-item decisions. The returned slice has the same length as
// items unless the model refused / drifted, in which case missing
// memories are reported by the caller as "no decision".
//
// timeout caps the HTTP call; LM Studio reasoning models can take
// 5-10s on a warm pass, longer on cold-start, so callers should
// allow at least 30s when running against local models.
func (c *Client) Judge(ctx context.Context, items []BatchItem, timeout time.Duration) ([]llmDecision, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	body := map[string]any{
		"model": c.model,
		"messages": []map[string]any{
			{"role": "system", "content": SystemPrompt()},
			{"role": "user", "content": renderBatch(items)},
		},
		"response_format": responseFormatFor(c.kind),
		"max_tokens":      4096,
		"stream":          false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("cleaner: marshal request: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("cleaner: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cleaner: call llm: %w", err)
	}
	defer res.Body.Close()
	rawRes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("cleaner: llm HTTP %d: %s", res.StatusCode, truncate(string(rawRes), 400))
	}

	var envelope struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rawRes, &envelope); err != nil {
		return nil, fmt.Errorf("cleaner: parse llm envelope: %w", err)
	}
	if len(envelope.Choices) == 0 {
		return nil, errors.New("cleaner: llm returned no choices")
	}
	content := strings.TrimSpace(envelope.Choices[0].Message.Content)
	// Reasoning models (qwen3 / deepseek-r1 / etc.) on LM Studio
	// emit the structured answer in reasoning_content; fall back
	// (mirrors the same fix in summarizer/provider_openai_compat.go).
	if content == "" {
		content = strings.TrimSpace(envelope.Choices[0].Message.ReasoningContent)
	}
	if content == "" {
		return nil, errors.New("cleaner: llm returned empty content")
	}

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

// responseFormatFor mirrors summarizer/provider_openai_compat —
// LM Studio rejects json_object, OpenAI accepts both. We always
// send the schema for "lmstudio" so reasoning models get the
// strict shape they need to stay aligned.
func responseFormatFor(kind string) map[string]any {
	switch kind {
	case "lmstudio":
		var schema map[string]any
		_ = json.Unmarshal([]byte(DecisionsJSONSchema), &schema)
		return map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "memory_cleanup_decisions",
				"strict": true,
				"schema": schema,
			},
		}
	default:
		return map[string]any{"type": "json_object"}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
