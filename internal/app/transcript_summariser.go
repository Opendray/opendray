package app

import (
	"context"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/worker"
)

// transcriptSummariser is the M18 implementation of
// projectdoc.TranscriptSummariser. As of M25 it routes through
// worker.Registry so operators can pick per-task between the
// summarizer HTTP path and a headless agent CLI (Claude/Gemini
// `--print` mode). The prompt + tag-extraction logic stays local
// to this file because each touchpoint has its own
// idiosyncrasies; only the LLM call dispatch is shared.
//
// Wrapped in <summary>…</summary> tags so reasoning models that
// leak thinking process still get post-processed cleanly — same
// trick the gitactivity summariser uses.
type transcriptSummariser struct {
	registry *worker.Registry
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

// newTranscriptSummariser is M25-shape: just hold a worker
// registry reference and dispatch when SummariseTranscript is
// called. The actual provider lookup happens inside the worker
// registry per call, so operator UI changes apply immediately
// without restart.
func newTranscriptSummariser(reg *worker.Registry) *transcriptSummariser {
	return &transcriptSummariser{registry: reg}
}

// SummariseTranscript implements projectdoc.TranscriptSummariser.
func (s *transcriptSummariser) SummariseTranscript(ctx context.Context, transcript string) (string, error) {
	if strings.TrimSpace(transcript) == "" {
		return "", nil
	}
	if s.registry == nil {
		// Registry not wired → degrade silently. Journaler will
		// write a metadata-only entry, which is the right fail-
		// mode for "no LLM available".
		return "", nil
	}
	resp, err := s.registry.Run(ctx, worker.Request{
		Task:         worker.TaskTranscript,
		SystemPrompt: transcriptSystemPrompt,
		UserInput:    transcript,
		MaxTokens:    4096,
		Timeout:      5 * time.Minute,
	})
	if err != nil {
		return "", err
	}
	if resp.Content == "" {
		return "", nil
	}
	return extractTaggedSummary(resp.Content), nil
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
