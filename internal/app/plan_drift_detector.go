package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/worker"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// planDriftDetector is the M-PA implementation of
// projectdoc.PlanDriftDetector. It routes through worker.Registry
// so operators can pick per-task between the summarizer HTTP path
// (cheap, local) and a headless Claude/Gemini agent (higher
// quality, higher latency). Same shape as transcriptSummariser —
// each touchpoint owns its own prompt assembly; only the LLM
// dispatch is shared.
type planDriftDetector struct {
	registry *worker.Registry
}

// newPlanDriftDetector builds a detector backed by the given
// worker registry. Nil registry → detector silently no-ops; same
// degraded behaviour as transcriptSummariser.
func newPlanDriftDetector(reg *worker.Registry) *planDriftDetector {
	return &planDriftDetector{registry: reg}
}

// DetectDrift implements projectdoc.PlanDriftDetector.
func (d *planDriftDetector) DetectDrift(ctx context.Context, in projectdoc.DriftInput) (projectdoc.DriftOutput, error) {
	if d.registry == nil {
		return projectdoc.DriftOutput{}, nil
	}
	if strings.TrimSpace(in.CurrentPlan) == "" {
		// No plan to drift from — refuse to seed one to avoid
		// hallucinated initial plans. Operator should seed the plan
		// manually; subsequent sessions will then keep it fresh.
		return projectdoc.DriftOutput{}, nil
	}
	if strings.TrimSpace(in.TranscriptSummary) == "" {
		// Without a transcript summary the detector has nothing
		// concrete to evaluate against. Skip rather than ask the
		// LLM to guess.
		return projectdoc.DriftOutput{}, nil
	}

	userInput := buildDriftUserInput(in)
	resp, err := d.registry.Run(ctx, worker.Request{
		Task:                     worker.TaskPlanDrift,
		SystemPrompt:             projectdoc.PlanDriftSystemPrompt,
		UserInput:                userInput,
		MaxTokens:                4096,
		Timeout:                  5 * time.Minute,
		ResponseFormatJSONSchema: driftJSONSchema,
	})
	if err != nil {
		return projectdoc.DriftOutput{}, err
	}
	if strings.TrimSpace(resp.Content) == "" {
		return projectdoc.DriftOutput{}, nil
	}
	return projectdoc.ParseDriftResponse(resp.Content)
}

// driftJSONSchema is the response_format=json_schema body that
// asks the model for a strict-shape reply. Workers that don't
// support structured output (agent CLI mode) translate this to a
// trailing instruction in the system prompt; the parser tolerates
// both clean JSON and fenced/preambled variants either way.
const driftJSONSchema = `{
  "name": "plan_drift_decision",
  "schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "should_propose": {"type": "boolean"},
      "new_plan":       {"type": "string"},
      "reason":         {"type": "string"}
    },
    "required": ["should_propose", "new_plan", "reason"]
  },
  "strict": true
}`

// buildDriftUserInput assembles the user-message payload the
// detector sends to the LLM. Kept as a top-level function so unit
// tests can pin the exact rendered shape without spinning up a
// worker.
func buildDriftUserInput(in projectdoc.DriftInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Project cwd\n\n`%s`\n\n", in.Cwd)
	b.WriteString("## Current plan\n\n")
	b.WriteString(strings.TrimSpace(in.CurrentPlan))
	b.WriteString("\n\n## Latest session summary\n\n")
	b.WriteString(strings.TrimSpace(in.TranscriptSummary))
	if len(in.RecentJournal) > 0 {
		b.WriteString("\n\n## Recent journal entries (oldest first)\n\n")
		// Render oldest first so chronology reads top-to-bottom — the
		// list comes from projectdoc.ListLogs newest-first, so reverse.
		for i := len(in.RecentJournal) - 1; i >= 0; i-- {
			e := in.RecentJournal[i]
			body := strings.TrimSpace(e.Content)
			if len(body) > 600 {
				body = body[:600] + "…"
			}
			if e.Title != "" {
				fmt.Fprintf(&b, "- **%s** — %s\n", e.Title, body)
			} else {
				fmt.Fprintf(&b, "- %s\n", body)
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}
