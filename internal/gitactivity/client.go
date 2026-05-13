package gitactivity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/worker"
)

// Client wraps a worker.Registry with the gitactivity-specific
// prompt + response parser. M25 — replaced the direct HTTP client
// with a Registry dispatch so the gitactivity touchpoint
// independently picks between the summarizer HTTP path and a
// headless Claude/Gemini agent based on the operator's choice in
// memory_workers.gitactivity.
//
// Why a separate client rather than re-using summarizer.Provider:
// summarizer.Provider's contract is "extract facts" which is the
// wrong shape — we want a prose summary, not a fact array. So we
// keep our own prompt + tag extraction here; only the LLM call
// dispatch is shared via the registry.
type Client struct {
	registry *worker.Registry
}

// NewClient builds a Client that dispatches through the given
// worker registry. Returns nil when registry is nil so callers
// (gitactivity.Service.WithLLM) can drop in defensively without
// a guard on every call site.
func NewClient(reg *worker.Registry) *Client {
	if reg == nil {
		return nil
	}
	return &Client{registry: reg}
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

// Summarise sends the rolled-up summary to the registry-selected
// worker and returns the 2-3 paragraph narrative.
func (c *Client) Summarise(ctx context.Context, s Summary) (string, error) {
	if c == nil || c.registry == nil {
		return "", errors.New("gitactivity: nil client / registry")
	}
	user := renderUserPrompt(s)
	resp, err := c.registry.Run(ctx, worker.Request{
		Task:         worker.TaskGitActivity,
		SystemPrompt: systemPromptText,
		UserInput:    user,
		MaxTokens:    4096,
		Timeout:      5 * time.Minute,
	})
	if err != nil {
		return "", fmt.Errorf("gitactivity: worker call: %w", err)
	}
	if strings.TrimSpace(resp.Content) == "" {
		return "", errors.New("gitactivity: worker returned empty content")
	}
	return extractSummary(resp.Content), nil
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
