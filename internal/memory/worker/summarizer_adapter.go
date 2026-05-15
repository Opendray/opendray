package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/summarizer"
)

// summarizerAdapter wraps Registry so callers that expect a
// summarizer.Provider (notably the capture engine) can transparently
// route through the worker fabric. When the worker for `task` is
// configured as kind=summarizer, behaviour is identical to calling
// the underlying summarizer.Provider directly; when kind=agent,
// the adapter assembles a textual prompt with the embedded JSON
// schema and parses the model's response into []summarizer.Fact.
type summarizerAdapter struct {
	reg  *Registry
	task TaskKind
}

// NewSummarizerProvider returns a summarizer.Provider that resolves
// its backing implementation lazily from Registry on each call.
// This means operators can change a task's worker (summarizer ⇄
// agent) from the UI without restarting any subsystem.
func NewSummarizerProvider(reg *Registry, task TaskKind) summarizer.Provider {
	return &summarizerAdapter{reg: reg, task: task}
}

func (a *summarizerAdapter) Name() string {
	return fmt.Sprintf("worker:%s", a.task)
}
func (a *summarizerAdapter) Kind() string {
	return fmt.Sprintf("worker:%s", a.task)
}

// Available pokes the registry the same way Summarize would. We
// don't fully resolve the agent / HTTP provider here — that would
// add a hot-path roundtrip — just confirm a worker row exists.
func (a *summarizerAdapter) Available(ctx context.Context) error {
	if a.reg == nil {
		return fmt.Errorf("%w: worker registry not wired", summarizer.ErrUnreachable)
	}
	if _, err := a.reg.WorkerFor(ctx, a.task); err != nil {
		if errors.Is(err, ErrNoWorkerConfigured) {
			return fmt.Errorf("%w: %v", summarizer.ErrUnreachable, err)
		}
		return err
	}
	return nil
}

// Summarize feeds the messages into the configured worker (either
// HTTP summarizer or headless agent CLI), expecting a JSON envelope
// matching summarizer.FactsToolJSONSchema. The 5 min timeout
// matches the other worker.Run callers — agents need it; HTTP
// summarizers usually finish in <10s.
func (a *summarizerAdapter) Summarize(ctx context.Context, msgs []summarizer.Message) (summarizer.SummarizeResult, error) {
	if a.reg == nil {
		return summarizer.SummarizeResult{}, fmt.Errorf("%w: worker registry not wired", summarizer.ErrUnreachable)
	}
	if len(msgs) == 0 {
		return summarizer.SummarizeResult{}, summarizer.ErrEmptyConversation
	}

	transcript := summarizer.MessagesToTranscriptText(msgs)
	if strings.TrimSpace(transcript) == "" {
		return summarizer.SummarizeResult{}, summarizer.ErrEmptyConversation
	}

	resp, err := a.reg.Run(ctx, Request{
		Task:                     a.task,
		SystemPrompt:             summarizer.SystemPrompt(),
		UserInput:                transcript,
		MaxTokens:                4096,
		Timeout:                  5 * time.Minute,
		ResponseFormatJSONSchema: capturePromptResponseSchema,
	})
	if err != nil {
		if errors.Is(err, ErrNoWorkerConfigured) {
			return summarizer.SummarizeResult{},
				fmt.Errorf("%w: %v", summarizer.ErrUnreachable, err)
		}
		return summarizer.SummarizeResult{}, err
	}

	facts, parseErr := parseFactsJSON(resp.Content)
	if parseErr != nil {
		return summarizer.SummarizeResult{
			RawResponse: truncate(resp.Content, 4096),
		}, fmt.Errorf("%w: %v", summarizer.ErrInvalidResponse, parseErr)
	}

	return summarizer.SummarizeResult{
		Facts:        facts,
		InputTokens:  resp.TokensIn,
		OutputTokens: resp.TokensOut,
		Latency:      time.Duration(resp.DurationMS) * time.Millisecond,
		RawResponse:  truncate(resp.Content, 4096),
	}, nil
}

// capturePromptResponseSchema wraps the summarizer's existing
// FactsToolJSONSchema in the response_format=json_schema envelope
// our workers expect. Summarizer-mode workers translate it to
// OpenAI-style response_format; agent-mode workers append the
// schema instructions to the system prompt.
var capturePromptResponseSchema = func() string {
	return `{
  "name": "capture_facts",
  "schema": ` + summarizer.FactsToolJSONSchema + `,
  "strict": true
}`
}()

// parseFactsJSON tolerates the same response-shape variations as
// projectdoc's drift parser: clean JSON, ```fenced``` blocks,
// leading or trailing prose. Returns ErrInvalidResponse when no
// JSON object can be located OR when the object doesn't contain a
// "facts" array.
func parseFactsJSON(raw string) ([]summarizer.Fact, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return nil, fmt.Errorf("empty response")
	}
	if fenced := stripJSONFence(body); fenced != "" {
		body = fenced
	}
	if i := strings.IndexByte(body, '{'); i >= 0 {
		if j := strings.LastIndexByte(body, '}'); j > i {
			body = body[i : j+1]
		}
	}
	var wrapper struct {
		Facts []struct {
			Text       string  `json:"text"`
			Category   string  `json:"category"`
			Confidence float32 `json:"confidence"`
		} `json:"facts"`
	}
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	out := make([]summarizer.Fact, 0, len(wrapper.Facts))
	for _, f := range wrapper.Facts {
		text := strings.TrimSpace(f.Text)
		if text == "" {
			continue
		}
		out = append(out, summarizer.Fact{
			Text:       text,
			Category:   summarizer.Category(f.Category),
			Confidence: f.Confidence,
		})
	}
	return out, nil
}

func stripJSONFence(s string) string {
	const fence = "```"
	i := strings.Index(s, fence)
	if i < 0 {
		return ""
	}
	rest := s[i+len(fence):]
	rest = strings.TrimPrefix(rest, "json")
	rest = strings.TrimLeft(rest, " \t\r\n")
	j := strings.Index(rest, fence)
	if j < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:j])
}
