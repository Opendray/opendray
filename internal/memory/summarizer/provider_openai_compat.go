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

// OpenAICompatConfig drives both OpenAI's real API and any service
// speaking the same wire shape — most notably LM Studio (no auth,
// runs on localhost:1234), but also Together / Groq / Fireworks /
// vLLM / Anyscale / Novita and many domestic-Chinese gateways.
//
// Kind controls log+UI labelling and decides whether an api_key
// is mandatory:
//   - "openai":   api_key required; default base_url
//     https://api.openai.com/v1
//   - "lmstudio": api_key optional (LM Studio ignores it); default
//     base_url http://localhost:1234/v1
//
// Any other Kind value is treated like "openai" for behaviour but
// preserved for display.
type OpenAICompatConfig struct {
	Kind      string // "openai" | "lmstudio"
	APIKey    string // plaintext; LM Studio may leave empty
	Model     string // e.g. "gpt-4o-mini"  or  "qwen2.5-7b-instruct"
	BaseURL   string // optional override
	Name      string // operator-friendly display
	MaxTokens int    // optional; default 1024
}

const (
	openaiDefaultBaseURL    = "https://api.openai.com/v1"
	lmstudioDefaultBaseURL  = "http://localhost:1234/v1"
	openaiCompatCallTimeout = 60 * time.Second
	openaiCompatMaxRetries  = 1
	openaiCompatRateBackoff = 8 * time.Second
)

// OpenAICompatProvider is one struct serving multiple Kind labels.
// The differences between OpenAI and LM Studio are entirely
// driven by config (base_url + auth required-ness); the wire
// format is identical, so duplicating two structs would be busy-
// work + churn.
type OpenAICompatProvider struct {
	cfg     OpenAICompatConfig
	client  *http.Client
	baseURL string
}

func NewOpenAICompatProvider(cfg OpenAICompatConfig) (*OpenAICompatProvider, error) {
	switch cfg.Kind {
	case "openai", "lmstudio":
	case "":
		cfg.Kind = "openai"
	default:
		// Other compatible kinds welcome — kept for future ext
		// without a registry change. Treat unknown as "openai-like".
	}
	if cfg.Model == "" {
		return nil, errors.New("openai-compat provider: Model required")
	}
	if cfg.Kind == "openai" && cfg.APIKey == "" {
		return nil, errors.New("openai provider: APIKey required")
	}

	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		switch cfg.Kind {
		case "openai":
			base = openaiDefaultBaseURL
		case "lmstudio":
			base = lmstudioDefaultBaseURL
		default:
			base = openaiDefaultBaseURL
		}
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 1024
	}
	if cfg.Name == "" {
		cfg.Name = cfg.Kind + "-" + cfg.Model
	}
	return &OpenAICompatProvider{
		cfg:     cfg,
		client:  &http.Client{Timeout: openaiCompatCallTimeout + 5*time.Second},
		baseURL: base,
	}, nil
}

func (p *OpenAICompatProvider) Name() string { return p.cfg.Name }
func (p *OpenAICompatProvider) Kind() string { return p.cfg.Kind }

// Available pings GET /models — the cheapest auth+reachability
// check that doesn't burn tokens. LM Studio also implements this,
// returning the loaded models list.
func (p *OpenAICompatProvider) Available(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	if p.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
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

// responseFormatFor returns the response_format block to send for a
// given provider kind. OpenAI accepts json_object — its loose-JSON
// mode; LM Studio (since 0.3.x) rejects json_object and only
// accepts json_schema or text. We send the strict schema from
// prompt.go for lmstudio so the model is guaranteed to emit the
// {"facts":[…]} envelope.
func responseFormatFor(kind string) map[string]any {
	switch kind {
	case "lmstudio":
		var schema map[string]any
		_ = json.Unmarshal([]byte(FactsToolJSONSchema), &schema)
		return map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   FactsToolName,
				"strict": true,
				"schema": schema,
			},
		}
	default:
		return map[string]any{"type": "json_object"}
	}
}

// Summarize uses chat completions with a kind-appropriate
// response_format to get strict JSON back. OpenAI's tool-use API
// would also work, but response_format is the lowest common
// denominator — every OpenAI-compatible provider supports either
// json_object (OpenAI itself, most third-party hosts) or
// json_schema (LM Studio).
func (p *OpenAICompatProvider) Summarize(ctx context.Context, msgs []Message) (SummarizeResult, error) {
	if len(msgs) == 0 {
		return SummarizeResult{}, ErrEmptyConversation
	}
	transcript := MessagesToTranscriptText(msgs)
	if transcript == "" {
		return SummarizeResult{}, ErrEmptyConversation
	}

	body := map[string]any{
		"model": p.cfg.Model,
		"messages": []map[string]any{
			{"role": "system", "content": SystemPrompt()},
			{"role": "user", "content": transcript},
		},
		"response_format": responseFormatFor(p.cfg.Kind),
		"max_tokens":      p.cfg.MaxTokens,
		"stream":          false,
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return SummarizeResult{}, fmt.Errorf("%w: marshal: %v", ErrInvalidResponse, err)
	}

	cctx, cancel := context.WithTimeout(ctx, openaiCompatCallTimeout)
	defer cancel()

	start := time.Now()
	res, raw, err := p.callWithRetry(cctx, rawBody)
	latency := time.Since(start)
	if err != nil {
		return SummarizeResult{Latency: latency, RawResponse: TruncateRaw(raw)}, err
	}

	facts, in, out, parseErr := parseOpenAICompatResponse(res)
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

func (p *OpenAICompatProvider) callWithRetry(ctx context.Context, body []byte) (map[string]any, string, error) {
	url := p.baseURL + "/chat/completions"
	var lastErr error
	for attempt := 0; attempt <= openaiCompatMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, "", fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
		}
		req.Header.Set("Content-Type", "application/json")
		if p.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
		}

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
			// LM Studio returns 404 when the requested model isn't
			// loaded; OpenAI returns 404 on a typo'd model name.
			return nil, string(raw), fmt.Errorf("%w: HTTP 404 — model %q not available", ErrModelNotFound, p.cfg.Model)

		case res.StatusCode == http.StatusTooManyRequests:
			wait := openaiCompatRateBackoff
			if h := res.Header.Get("Retry-After"); h != "" {
				if n, perr := strconv.Atoi(h); perr == nil && n > 0 && n < int(openaiCompatRateBackoff.Seconds()) {
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

// parseOpenAICompatResponse turns the JSON reply into facts +
// usage numbers. Shape:
//
//	{
//	  "choices":[{"message":{"content":"<JSON string with facts>"}}],
//	  "usage":{"prompt_tokens":N,"completion_tokens":M,"total_tokens":..}
//	}
//
// The "content" holds the JSON object we instructed the model to
// emit (response_format=json_object).
func parseOpenAICompatResponse(res map[string]any) ([]Fact, int, int, error) {
	usage, _ := res["usage"].(map[string]any)
	in, _ := numberToInt(usage["prompt_tokens"])
	out, _ := numberToInt(usage["completion_tokens"])

	choices, ok := res["choices"].([]any)
	if !ok || len(choices) == 0 {
		return nil, in, out, fmt.Errorf("%w: missing choices", ErrInvalidResponse)
	}
	first, _ := choices[0].(map[string]any)
	msg, _ := first["message"].(map[string]any)
	content, _ := msg["content"].(string)
	content = strings.TrimSpace(content)
	// LM Studio reasoning models (qwen3/r1/deepseek/etc.) emit the
	// actual structured answer in reasoning_content and leave content
	// empty. Fall back to that field so the gatekeeper + capture engine
	// work on reasoning-model backends without operator workarounds.
	if content == "" {
		if rc, ok := msg["reasoning_content"].(string); ok {
			content = strings.TrimSpace(rc)
		}
	}
	if content == "" {
		return nil, in, out, nil
	}

	var inner struct {
		Facts []map[string]any `json:"facts"`
	}
	if err := json.Unmarshal([]byte(content), &inner); err != nil {
		return nil, in, out, fmt.Errorf("%w: inner json: %v", ErrInvalidResponse, err)
	}
	rawFacts := make([]any, 0, len(inner.Facts))
	for _, f := range inner.Facts {
		rawFacts = append(rawFacts, f)
	}
	facts, _ := decodeFactsArray(rawFacts)
	return facts, in, out, nil
}
