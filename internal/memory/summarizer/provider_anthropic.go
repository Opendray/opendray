package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AnthropicConfig holds the non-sensitive runtime knobs. APIKey is
// decrypted from the DB row (api_key_ciphertext) by the caller and
// passed in plaintext at construction time.
type AnthropicConfig struct {
	APIKey  string // plaintext; never persist
	Model   string // e.g. "claude-haiku-4-5"
	BaseURL string // optional; default "https://api.anthropic.com"
	Name    string // operator-friendly name shown in logs
	// MaxTokens caps the LLM's output. 1024 is plenty for a fact array
	// (≈ 50 facts at 20 tokens each) and bounds cost. Operators can
	// raise via extra_config.
	MaxTokens int
}

const (
	anthropicDefaultBaseURL   = "https://api.anthropic.com"
	anthropicVersionHeader    = "2023-06-01"
	anthropicDefaultMaxTokens = 1024
	anthropicCallTimeout      = 30 * time.Second
	anthropicHealthcheckPath  = "/v1/models"
	anthropicMaxRetries       = 1 // retry once on 5xx
	anthropicRateLimitBackoff = 8 * time.Second
)

// AnthropicProvider talks to Anthropic's Messages API directly via
// net/http, no anthropic-sdk-go dependency. The wire format is
// stable enough that hand-writing it is cheaper than tracking the
// SDK's interface churn.
type AnthropicProvider struct {
	cfg     AnthropicConfig
	client  *http.Client
	baseURL string
}

// NewAnthropicProvider validates required fields and returns the
// provider. Does not network. Returns an error when APIKey/Model
// are missing — runtime calls would never succeed without them.
func NewAnthropicProvider(cfg AnthropicConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("anthropic provider: APIKey required")
	}
	if cfg.Model == "" {
		return nil, errors.New("anthropic provider: Model required")
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = anthropicDefaultBaseURL
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = anthropicDefaultMaxTokens
	}
	if cfg.Name == "" {
		cfg.Name = "anthropic-" + cfg.Model
	}
	return &AnthropicProvider{
		cfg:     cfg,
		client:  &http.Client{Timeout: anthropicCallTimeout + 5*time.Second},
		baseURL: base,
	}, nil
}

func (p *AnthropicProvider) Name() string { return p.cfg.Name }
func (p *AnthropicProvider) Kind() string { return "anthropic" }

// Available pings GET /v1/models — the cheapest call that exercises
// auth + connectivity without committing tokens.
func (p *AnthropicProvider) Available(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+anthropicHealthcheckPath, nil)
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	p.setAuthHeaders(req)

	res, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrAuthFailed
	default:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("%w: HTTP %d: %s", ErrUnreachable, res.StatusCode, body)
	}
}

// Summarize sends the conversation as a single user message and
// forces a tool call to record_facts. Returns the parsed Facts +
// usage telemetry.
func (p *AnthropicProvider) Summarize(ctx context.Context, msgs []Message) (SummarizeResult, error) {
	if len(msgs) == 0 {
		return SummarizeResult{}, ErrEmptyConversation
	}
	transcript := MessagesToTranscriptText(msgs)
	if transcript == "" {
		return SummarizeResult{}, ErrEmptyConversation
	}

	body, err := p.buildRequestBody(transcript)
	if err != nil {
		return SummarizeResult{}, err
	}

	cctx, cancel := context.WithTimeout(ctx, anthropicCallTimeout)
	defer cancel()

	start := time.Now()
	res, raw, err := p.callWithRetry(cctx, body)
	latency := time.Since(start)
	if err != nil {
		return SummarizeResult{Latency: latency, RawResponse: TruncateRaw(raw)}, err
	}

	facts, in, out, parseErr := parseAnthropicResponse(res)
	usd := EstimateUSD(p.cfg.Model, in, out)
	result := SummarizeResult{
		Facts:        facts,
		InputTokens:  in,
		OutputTokens: out,
		EstimatedUSD: usd,
		Latency:      latency,
		RawResponse:  TruncateRaw(raw),
	}
	if parseErr != nil {
		return result, parseErr
	}
	return result, nil
}

func (p *AnthropicProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("x-api-key", p.cfg.APIKey)
	req.Header.Set("anthropic-version", anthropicVersionHeader)
	req.Header.Set("content-type", "application/json")
}

func (p *AnthropicProvider) buildRequestBody(transcript string) ([]byte, error) {
	body := map[string]any{
		"model":      p.cfg.Model,
		"max_tokens": p.cfg.MaxTokens,
		"system":     SystemPrompt(),
		"messages": []map[string]any{
			{"role": "user", "content": transcript},
		},
		"tools": []map[string]any{
			{
				"name":         FactsToolName,
				"description":  FactsToolDescription,
				"input_schema": json.RawMessage(FactsToolJSONSchema),
			},
		},
		"tool_choice": map[string]any{
			"type": "tool",
			"name": FactsToolName,
		},
	}
	return json.Marshal(body)
}

// callWithRetry: 1 retry on 5xx, exponential backoff on 429.
// Returns the parsed map[string]any + raw body string + error.
func (p *AnthropicProvider) callWithRetry(ctx context.Context, body []byte) (map[string]any, string, error) {
	url := p.baseURL + "/v1/messages"
	var lastErr error
	for attempt := 0; attempt <= anthropicMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, "", fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
		}
		p.setAuthHeaders(req)

		res, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrUnreachable, err)
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 64*1024))
		res.Body.Close()

		switch {
		case res.StatusCode == http.StatusOK:
			var parsed map[string]any
			if jerr := json.Unmarshal(raw, &parsed); jerr != nil {
				return nil, string(raw), fmt.Errorf("%w: json: %v", ErrInvalidResponse, jerr)
			}
			return parsed, string(raw), nil

		case res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusForbidden:
			return nil, string(raw), fmt.Errorf("%w: HTTP %d", ErrAuthFailed, res.StatusCode)

		case res.StatusCode == http.StatusNotFound:
			return nil, string(raw), fmt.Errorf("%w: HTTP %d", ErrModelNotFound, res.StatusCode)

		case res.StatusCode == http.StatusTooManyRequests:
			// Honour Retry-After header up to our cap.
			wait := anthropicRateLimitBackoff
			if h := res.Header.Get("Retry-After"); h != "" {
				if n, perr := strconv.Atoi(h); perr == nil && n > 0 && n < int(anthropicRateLimitBackoff.Seconds()) {
					wait = time.Duration(n) * time.Second
				}
			}
			lastErr = fmt.Errorf("%w: HTTP 429", ErrRateLimited)
			select {
			case <-ctx.Done():
				return nil, string(raw), ctx.Err()
			case <-time.After(wait):
			}
			continue

		case res.StatusCode >= 500:
			lastErr = fmt.Errorf("%w: HTTP %d: %s", ErrUnreachable, res.StatusCode, raw)
			continue

		default:
			return nil, string(raw), fmt.Errorf("%w: HTTP %d: %s", ErrInvalidResponse, res.StatusCode, raw)
		}
	}
	return nil, "", lastErr
}

// parseAnthropicResponse extracts (facts, input_tokens, output_tokens)
// from the parsed JSON. The shape we expect:
//
//	{
//	  "content": [{"type":"tool_use", "name":"record_facts",
//	               "input": {"facts":[...]}}],
//	  "usage": {"input_tokens": N, "output_tokens": M}
//	}
func parseAnthropicResponse(res map[string]any) ([]Fact, int, int, error) {
	usage, _ := res["usage"].(map[string]any)
	in, _ := numberToInt(usage["input_tokens"])
	out, _ := numberToInt(usage["output_tokens"])

	content, ok := res["content"].([]any)
	if !ok {
		return nil, in, out, fmt.Errorf("%w: missing content array", ErrInvalidResponse)
	}
	for _, blk := range content {
		blkMap, ok := blk.(map[string]any)
		if !ok {
			continue
		}
		if blkMap["type"] != "tool_use" {
			continue
		}
		input, ok := blkMap["input"].(map[string]any)
		if !ok {
			return nil, in, out, fmt.Errorf("%w: tool_use without input object", ErrInvalidResponse)
		}
		factsRaw, ok := input["facts"].([]any)
		if !ok {
			// Empty / absent — treat as zero facts; not an error.
			return nil, in, out, nil
		}
		facts, err := decodeFactsArray(factsRaw)
		if err != nil {
			return nil, in, out, err
		}
		return facts, in, out, nil
	}
	// No tool_use block — model may have generated a stop_sequence.
	// Treat as zero facts, not an error.
	return nil, in, out, nil
}

// decodeFactsArray converts the JSON facts array into []Fact,
// dropping items that fail validation rather than aborting the
// whole batch — partial extraction is more useful than nothing.
func decodeFactsArray(raw []any) ([]Fact, error) {
	out := make([]Fact, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text, _ := m["text"].(string)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		conf, _ := numberToFloat32(m["confidence"])
		if conf < 0 {
			conf = 0
		}
		if conf > 1 {
			conf = 1
		}
		category, _ := m["category"].(string)
		c := Category(category)
		switch c {
		case CategoryPreference, CategoryIdentifier, CategoryDecision, CategoryTask, CategoryOther:
		default:
			c = CategoryOther
		}
		out = append(out, Fact{Text: text, Confidence: conf, Category: c})
	}
	return out, nil
}

func numberToInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return int(i), true
		}
	}
	return 0, false
}

func numberToFloat32(v any) (float32, bool) {
	switch n := v.(type) {
	case float64:
		return float32(n), true
	case float32:
		return n, true
	case int:
		return float32(n), true
	case json.Number:
		f, err := n.Float64()
		if err == nil {
			return float32(f), true
		}
	}
	return 0, false
}
