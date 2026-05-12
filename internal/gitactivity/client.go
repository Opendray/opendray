package gitactivity

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

// Client wraps an OpenAI-compatible chat completions endpoint with
// the "project historian" prompt below. Built per Service from a
// summarizer.ProviderRow so operators can reuse whichever provider
// they configured for the gatekeeper / cleaner.
//
// Why a separate client rather than re-using summarizer.Provider:
// summarizer.Provider's contract is "extract facts" which is the
// wrong shape — we want a prose summary, not a fact array. The
// HTTP envelope is identical to cleaner's, so this is a small
// focused copy.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	kind    string
	http    *http.Client
}

// NewClient builds a Client from a summarizer provider row. Same
// supported kinds as the cleaner: openai / lmstudio.
func NewClient(row summarizer.ProviderRow, apiKey string) (*Client, error) {
	switch row.Kind {
	case "openai", "lmstudio":
	default:
		return nil, fmt.Errorf("gitactivity: unsupported provider kind %q", row.Kind)
	}
	base := strings.TrimRight(row.BaseURL, "/")
	if base == "" {
		return nil, errors.New("gitactivity: provider has no base_url")
	}
	if row.Model == "" {
		return nil, errors.New("gitactivity: provider has no model")
	}
	return &Client{
		baseURL: base,
		apiKey:  apiKey,
		model:   row.Model,
		kind:    row.Kind,
		http:    &http.Client{Timeout: 5 * time.Minute},
	}, nil
}

const systemPromptText = `You are a project historian.

The user will give you a numbered list of recent git commits with
their file change stats. Produce a 2-3 paragraph PROSE summary
covering: what major areas saw work, recurring themes, and
anything an incoming session should know to avoid duplicating
completed work.

OUTPUT FORMAT — strictly enforced:

Wrap your final summary in <summary>...</summary> tags. ANY
content outside the tags will be discarded by the caller, so:

- Put NOTHING before <summary>
- Put NOTHING after </summary>
- Inside the tags: 2-3 plain markdown paragraphs separated by
  blank lines. No bullet lists, no section headers, no code fences.
- Refer to file paths with backticks inline.

If you want to think first, do it OUTSIDE the tags — the caller
will throw that away. Only the tagged region survives.

Example shape:

<summary>
First paragraph about the major themes.

Second paragraph about recurring patterns and specific files
that saw heavy activity, like ` + "`internal/foo/bar.go`" + `.

Third paragraph with advice for incoming sessions.
</summary>`

// Summarise sends the rolled-up summary to the LLM and returns the
// 2-3 paragraph narrative.
func (c *Client) Summarise(ctx context.Context, s Summary) (string, error) {
	user := renderUserPrompt(s)
	body := map[string]any{
		"model": c.model,
		"messages": []map[string]any{
			{"role": "system", "content": systemPromptText},
			{"role": "user", "content": user},
		},
		"max_tokens": 4096,
		"stream":     false,
	}
	// LM Studio rejects json_object for plain-text answers; we
	// don't need a JSON schema here so just leave response_format
	// unset.
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("gitactivity: marshal request: %w", err)
	}

	// Caller controls the timeout via ctx — gitactivity.Service
	// gives us a 5-minute deadline because reasoning models on
	// LM Studio commonly need 2-3 minutes on a 50-commit batch.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("gitactivity: call llm: %w", err)
	}
	defer res.Body.Close()
	rawRes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("gitactivity: llm HTTP %d: %s", res.StatusCode, truncate(string(rawRes), 400))
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
		return "", fmt.Errorf("gitactivity: parse llm envelope: %w", err)
	}
	if len(envelope.Choices) == 0 {
		return "", errors.New("gitactivity: llm returned no choices")
	}
	content := strings.TrimSpace(envelope.Choices[0].Message.Content)
	if content == "" {
		// Reasoning models put their answer in reasoning_content.
		content = strings.TrimSpace(envelope.Choices[0].Message.ReasoningContent)
	}
	if content == "" {
		return "", errors.New("gitactivity: llm returned empty content")
	}
	return extractSummary(content), nil
}

// extractSummary pulls the content inside <summary>...</summary>
// tags. Reasoning models often leak thinking process around the
// actual answer; the tag dance lets us keep the final summary and
// discard everything else.
//
// When tags are absent (older non-reasoning models that follow
// instructions exactly) we return the trimmed input as-is.
func extractSummary(s string) string {
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

// renderUserPrompt formats the Summary as a numbered commit list
// the LLM can scan. We cap at the configured Summary.Commits
// length (Service already enforces a limit upstream).
func renderUserPrompt(s Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b,
		"Repository activity since %s. %d commits, %d files changed, +%d / -%d lines.\n\n",
		s.WindowSince, s.TotalCommits, s.TotalFiles, s.TotalInserts, s.TotalDeletes)
	if len(s.HotPaths) > 0 {
		b.WriteString("Most-touched paths:\n")
		for _, p := range s.HotPaths {
			fmt.Fprintf(&b, "- %s (%d commits)\n", p.Path, p.Hits)
		}
		b.WriteString("\n")
	}
	b.WriteString("Commits:\n")
	for i, c := range s.Commits {
		fmt.Fprintf(&b, "[%d] %s — %s _(%s, +%d/-%d, %d files)_\n",
			i+1, c.SHA, c.Subject,
			c.AuthoredAt.Format("Jan 02"),
			c.Insertions, c.Deletions, c.FilesChanged)
		// Append a few of the changed files inline if available.
		if len(c.Files) > 0 {
			fmt.Fprintf(&b, "    files: %s\n", strings.Join(c.Files, ", "))
		}
	}
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
