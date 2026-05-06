package summarizer

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
)

// OllamaConfig — local-runs-on-operator-hardware backend.
// No api_key. The base_url points at an ollama HTTP daemon
// (typically http://localhost:11434).
type OllamaConfig struct {
	Model     string // e.g. "llama3.1:8b" or "qwen2.5:7b"
	BaseURL   string // e.g. "http://localhost:11434"
	Name      string // operator-friendly display
	MaxTokens int    // optional ollama num_predict (no hard limit if 0)
}

const (
	ollamaCallTimeout     = 60 * time.Second // local models are typically slower
	ollamaHealthcheckPath = "/api/tags"
	ollamaChatPath        = "/api/chat"
)

// OllamaProvider talks to a local ollama daemon. Reuses our HTTP
// pattern from the embedder_http.go style — no SDK, just net/http.
type OllamaProvider struct {
	cfg     OllamaConfig
	client  *http.Client
	baseURL string
}

func NewOllamaProvider(cfg OllamaConfig) (*OllamaProvider, error) {
	if cfg.Model == "" {
		return nil, errors.New("ollama provider: Model required")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("ollama provider: BaseURL required")
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Name == "" {
		cfg.Name = "ollama-" + cfg.Model
	}
	return &OllamaProvider{
		cfg:     cfg,
		client:  &http.Client{Timeout: ollamaCallTimeout + 5*time.Second},
		baseURL: base,
	}, nil
}

func (p *OllamaProvider) Name() string { return p.cfg.Name }
func (p *OllamaProvider) Kind() string { return "ollama" }

// Available pings GET /api/tags — confirms the daemon is up + the
// configured model is loaded. We don't fail when the model isn't
// in the list (operator may use `ollama pull` lazily) but we do
// fail when the daemon doesn't respond.
func (p *OllamaProvider) Available(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+ollamaHealthcheckPath, nil)
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	res, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("%w: HTTP %d: %s", ErrUnreachable, res.StatusCode, body)
	}
	return nil
}

// Summarize sends a non-streaming chat request with format:"json"
// so the model returns a JSON object directly (instead of
// generating prose containing JSON).
//
// ollama's response shape:
//
//	{
//	  "model": "...", "created_at": "...",
//	  "message": {"role":"assistant", "content":"<JSON string>"},
//	  "done": true,
//	  "prompt_eval_count": 100,   // input tokens
//	  "eval_count": 50            // output tokens
//	}
//
// We unmarshal message.content as the {"facts":[...]} payload.
func (p *OllamaProvider) Summarize(ctx context.Context, msgs []Message) (SummarizeResult, error) {
	if len(msgs) == 0 {
		return SummarizeResult{}, ErrEmptyConversation
	}
	transcript := MessagesToTranscriptText(msgs)
	if transcript == "" {
		return SummarizeResult{}, ErrEmptyConversation
	}

	body := map[string]any{
		"model":  p.cfg.Model,
		"format": "json",
		"stream": false,
		"messages": []map[string]any{
			{"role": "system", "content": SystemPrompt()},
			{"role": "user", "content": transcript},
		},
	}
	if p.cfg.MaxTokens > 0 {
		body["options"] = map[string]any{"num_predict": p.cfg.MaxTokens}
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return SummarizeResult{}, fmt.Errorf("%w: marshal: %v", ErrInvalidResponse, err)
	}

	cctx, cancel := context.WithTimeout(ctx, ollamaCallTimeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, p.baseURL+ollamaChatPath, bytes.NewReader(rawBody))
	if err != nil {
		return SummarizeResult{}, fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	req.Header.Set("content-type", "application/json")

	res, err := p.client.Do(req)
	if err != nil {
		return SummarizeResult{Latency: time.Since(start)}, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	res.Body.Close()
	latency := time.Since(start)

	if res.StatusCode == http.StatusNotFound {
		return SummarizeResult{Latency: latency, RawResponse: TruncateRaw(string(raw))},
			fmt.Errorf("%w: model %q not pulled (run `ollama pull %s`)", ErrModelNotFound, p.cfg.Model, p.cfg.Model)
	}
	if res.StatusCode != http.StatusOK {
		return SummarizeResult{Latency: latency, RawResponse: TruncateRaw(string(raw))},
			fmt.Errorf("%w: HTTP %d: %s", ErrUnreachable, res.StatusCode, raw)
	}

	facts, in, out, parseErr := parseOllamaResponse(raw)
	usd := EstimateUSD(p.cfg.Model, in, out) // typically 0 for local
	result := SummarizeResult{
		Facts:        facts,
		InputTokens:  in,
		OutputTokens: out,
		EstimatedUSD: usd,
		Latency:      latency,
		RawResponse:  TruncateRaw(string(raw)),
	}
	if parseErr != nil {
		return result, parseErr
	}
	return result, nil
}

func parseOllamaResponse(raw []byte) ([]Fact, int, int, error) {
	var envelope struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, 0, 0, fmt.Errorf("%w: envelope: %v", ErrInvalidResponse, err)
	}

	in := envelope.PromptEvalCount
	out := envelope.EvalCount

	content := strings.TrimSpace(envelope.Message.Content)
	if content == "" {
		return nil, in, out, nil
	}

	// ollama's format:"json" returns a JSON object as a string.
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
