package app

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

// transcriptSummariser is the M18 implementation of
// projectdoc.TranscriptSummariser. It re-uses the OpenAI-compatible
// chat completions endpoint we already use for the gatekeeper and
// git activity summarisers, with a "session reviewer" prompt that
// turns a raw transcript into 1-3 paragraphs of "what the agent
// did in this session".
//
// Wrapped in <summary>…</summary> tags so reasoning models that
// leak thinking process still get post-processed cleanly — same
// trick the gitactivity summariser uses.
type transcriptSummariser struct {
	baseURL string
	apiKey  string
	model   string
	kind    string
	http    *http.Client
}

const transcriptSystemPrompt = `You are a session reviewer.

The user will give you the conversation transcript between an
operator (user) and an AI coding agent (assistant). Your job:
produce a 1-3 paragraph summary of what THE AGENT actually did in
this session — files edited, decisions made, problems debugged,
features shipped, blockers hit. The reader is a future agent
inheriting this project; they need to know what is already done.

Wrap your final answer in <summary>...</summary> tags. ANYTHING
outside the tags is discarded by the caller, so:

- Put NOTHING before <summary>
- Put NOTHING after </summary>
- Inside: 1-3 plain markdown paragraphs separated by blank lines.
  No bullet lists, no section headers, no code fences.
- Refer to files with backticks inline.
- Be specific and concrete. "Refactored memory module" is weak;
  "Extracted bcrypt helper to ` + "`internal/auth/bcrypt.go`" + ` and
  added a benchmark" is good.

If the transcript is too short / sparse to summarise (e.g. agent
just answered one question and the session ended), output an empty
<summary></summary> block.`

func newTranscriptSummariser(row summarizer.ProviderRow, apiKey string) (*transcriptSummariser, error) {
	switch row.Kind {
	case "openai", "lmstudio":
	default:
		return nil, fmt.Errorf("transcript summariser: unsupported provider kind %q", row.Kind)
	}
	base := strings.TrimRight(row.BaseURL, "/")
	if base == "" {
		return nil, errors.New("transcript summariser: provider has no base_url")
	}
	if row.Model == "" {
		return nil, errors.New("transcript summariser: provider has no model")
	}
	return &transcriptSummariser{
		baseURL: base,
		apiKey:  apiKey,
		model:   row.Model,
		kind:    row.Kind,
		http:    &http.Client{Timeout: 5 * time.Minute},
	}, nil
}

// SummariseTranscript implements projectdoc.TranscriptSummariser.
func (s *transcriptSummariser) SummariseTranscript(ctx context.Context, transcript string) (string, error) {
	if strings.TrimSpace(transcript) == "" {
		return "", nil
	}
	body := map[string]any{
		"model": s.model,
		"messages": []map[string]any{
			{"role": "system", "content": transcriptSystemPrompt},
			{"role": "user", "content": transcript},
		},
		"max_tokens": 4096,
		"stream":     false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("transcript summariser: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	res, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcript summariser: call: %w", err)
	}
	defer res.Body.Close()
	rawRes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("transcript summariser: HTTP %d: %s", res.StatusCode, truncate(string(rawRes), 300))
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
		return "", fmt.Errorf("transcript summariser: parse: %w", err)
	}
	if len(envelope.Choices) == 0 {
		return "", errors.New("transcript summariser: no choices")
	}
	content := strings.TrimSpace(envelope.Choices[0].Message.Content)
	if content == "" {
		content = strings.TrimSpace(envelope.Choices[0].Message.ReasoningContent)
	}
	if content == "" {
		return "", nil
	}
	return extractTaggedSummary(content), nil
}

// extractTaggedSummary pulls content between <summary>...</summary>
// — same logic as gitactivity.extractSummary. Kept inline rather
// than imported to avoid a circular-ish app → gitactivity import.
func extractTaggedSummary(s string) string {
	const open = "<summary>"
	const close = "</summary>"
	i := strings.Index(s, open)
	if i < 0 {
		return strings.TrimSpace(s)
	}
	rest := s[i+len(open):]
	j := strings.Index(rest, close)
	if j < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:j])
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
