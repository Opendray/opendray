package worker

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

// SummarizerWorker routes the call through the existing
// summarizer.Registry → OpenAI-compatible HTTP path.
//
// We deliberately don't share code with the gatekeeper / cleaner /
// gitactivity HTTP helpers that existed before M25 — those each
// had bespoke prompt shapes and response parsers. M25 keeps the
// Worker surface generic (system prompt + user input → text out)
// so each subsystem still owns its own prompt; this layer is just
// "make the HTTP call and hand back the content".
type SummarizerWorker struct {
	cfg Config
	reg *summarizer.Registry
}

// NewSummarizerWorker builds a worker that uses the given config's
// summarizer_id to pick a provider row. Empty SummarizerID means
// "use the registry default" (IsDefault row, falling back to
// alphabetically first enabled).
func NewSummarizerWorker(reg *summarizer.Registry, cfg Config) *SummarizerWorker {
	return &SummarizerWorker{cfg: cfg, reg: reg}
}

func (w *SummarizerWorker) Kind() WorkerKind { return WorkerSummarizer }

func (w *SummarizerWorker) Run(ctx context.Context, req Request) (Response, error) {
	row, err := w.pickProvider(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("summarizer worker: pick provider: %w", err)
	}
	if row.BaseURL == "" || row.Model == "" {
		return Response{}, errors.New("summarizer worker: provider missing base_url or model")
	}

	body := map[string]any{
		"model": row.Model,
		"messages": []map[string]any{
			{"role": "system", "content": req.SystemPrompt},
			{"role": "user", "content": req.UserInput},
		},
		"stream": false,
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.ResponseFormatJSONSchema != "" {
		// json_schema form per LM Studio / OpenAI 2024 spec. We
		// embed the schema verbatim — callers must hand us a
		// valid JSON Schema string.
		var schema any
		if err := json.Unmarshal([]byte(req.ResponseFormatJSONSchema), &schema); err == nil {
			body["response_format"] = map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "memory_worker_response",
					"strict": true,
					"schema": schema,
				},
			}
		}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return Response{}, fmt.Errorf("summarizer worker: marshal: %w", err)
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	client := &http.Client{Timeout: timeout}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(row.BaseURL, "/")+"/chat/completions",
		bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if row.APIKeyPlaintext != "" {
		httpReq.Header.Set("Authorization", "Bearer "+row.APIKeyPlaintext)
	}

	t0 := time.Now()
	res, err := client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("summarizer worker: call: %w", err)
	}
	defer res.Body.Close()
	respRaw, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	dur := time.Since(t0).Milliseconds()
	if res.StatusCode/100 != 2 {
		return Response{}, fmt.Errorf("summarizer worker: HTTP %d: %s",
			res.StatusCode, truncate(string(respRaw), 300))
	}

	var envelope struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respRaw, &envelope); err != nil {
		return Response{}, fmt.Errorf("summarizer worker: parse: %w", err)
	}
	content := ""
	if len(envelope.Choices) > 0 {
		content = strings.TrimSpace(envelope.Choices[0].Message.Content)
		if content == "" {
			content = strings.TrimSpace(envelope.Choices[0].Message.ReasoningContent)
		}
	}
	return Response{
		Content:    content,
		DurationMS: dur,
		TokensIn:   envelope.Usage.PromptTokens,
		TokensOut:  envelope.Usage.CompletionTokens,
		WorkerKind: WorkerSummarizer,
		ProviderID: row.ID,
	}, nil
}

// pickProvider resolves the summarizer row to use for this call.
// Honors the per-task SummarizerID when set; otherwise picks the
// registry's default (IsDefault=true, else alphabetically first).
func (w *SummarizerWorker) pickProvider(ctx context.Context) (summarizer.ProviderRow, error) {
	if w.cfg.SummarizerID != "" {
		// Force decryption by going through Build().
		if _, err := w.reg.Build(ctx, w.cfg.SummarizerID); err != nil {
			return summarizer.ProviderRow{}, err
		}
		// Re-read with the now-decrypted plaintext key.
		fresh, err := w.reg.ListEnabledRows(ctx)
		if err != nil {
			return summarizer.ProviderRow{}, err
		}
		for _, r := range fresh {
			if r.ID == w.cfg.SummarizerID {
				return r, nil
			}
		}
		return summarizer.ProviderRow{}, fmt.Errorf("summarizer worker: configured provider %s not found / disabled", w.cfg.SummarizerID)
	}
	rows, err := w.reg.ListEnabledRows(ctx)
	if err != nil {
		return summarizer.ProviderRow{}, err
	}
	if len(rows) == 0 {
		return summarizer.ProviderRow{}, errors.New("summarizer worker: no enabled providers")
	}
	pick := rows[0]
	for _, r := range rows {
		if r.IsDefault {
			pick = r
			break
		}
	}
	// Build to decrypt API key.
	if _, err := w.reg.Build(ctx, pick.ID); err != nil {
		return summarizer.ProviderRow{}, err
	}
	fresh, _ := w.reg.ListEnabledRows(ctx)
	for _, r := range fresh {
		if r.ID == pick.ID {
			return r, nil
		}
	}
	return pick, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
