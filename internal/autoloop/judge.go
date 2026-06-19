package autoloop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/memory/worker"
)

// Verdict is the strict response a goal-mode judge returns after inspecting
// one turn's output. NextPrompt is only meaningful when Decision == continue.
type Verdict struct {
	Decision   string `json:"decision"` // continue | done | escalate | fail
	Reason     string `json:"reason"`
	NextPrompt string `json:"next_prompt"`
}

// validDecision normalises an unknown/blank decision to a safe escalate: when
// the judge is unsure, pull a human in rather than loop blindly.
func (v Verdict) normalise() Verdict {
	switch v.Decision {
	case DecisionContinue, DecisionDone, DecisionEscalate, DecisionFail:
		return v
	default:
		out := v
		out.Decision = DecisionEscalate
		if out.Reason == "" {
			out.Reason = "judge returned an unrecognised decision; escalating"
		}
		return out
	}
}

// verdictJSONSchema is the OpenAI-spec response_format schema. Summarizer
// workers translate it to response_format=json_schema; agent workers append
// it to the system prompt. Mirrors the strict-schema pattern Cortex curation
// already uses.
const verdictJSONSchema = `{
  "name": "loop_verdict",
  "schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "decision":    {"type": "string", "enum": ["continue", "done", "escalate", "fail"]},
      "reason":      {"type": "string"},
      "next_prompt": {"type": "string"}
    },
    "required": ["decision", "reason", "next_prompt"]
  },
  "strict": true
}`

const judgeSystemPrompt = `You are the verifier in an autonomous agent loop. You are given a GOAL and the OUTPUT of the agent's most recent turn. Decide one of:
- "continue": the goal is not met yet and the agent should keep going. Put the concrete next instruction in "next_prompt".
- "done": the goal is fully met. No further turns are needed.
- "escalate": you are unsure, the agent appears stuck, or a human decision is needed. Prefer this over guessing.
- "fail": this turn clearly failed or produced an error that the agent cannot recover from on its own.
Be conservative: if you cannot positively confirm the goal is met, do NOT return "done". Reply ONLY with the structured JSON.`

// WorkerJudge implements Judger over worker.Registry, routing to whichever
// worker (summarizer by default, or an agent CLI) the operator configured for
// the loop's JudgeTask. The judge is decoupled from the driven session's CLI,
// so a goal loop over any provider can be verified by the same cheap worker.
type WorkerJudge struct {
	reg     *worker.Registry
	timeout time.Duration
}

// NewWorkerJudge builds a judge. A zero timeout defaults to 90s — long enough
// for a headless agent judge, bounded so a hung worker can't stall the loop.
func NewWorkerJudge(reg *worker.Registry, timeout time.Duration) *WorkerJudge {
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &WorkerJudge{reg: reg, timeout: timeout}
}

// Judge runs the configured worker over the goal + turn output and parses the
// structured verdict. A worker that isn't configured, errors, or returns
// unparseable output yields an escalate verdict (never a silent continue).
func (j *WorkerJudge) Judge(ctx context.Context, loop Loop, turnOutput string) (Verdict, error) {
	task := worker.TaskKind(loop.JudgeTask)
	if task == "" {
		task = worker.TaskLoopJudge
	}
	resp, err := j.reg.Run(ctx, worker.Request{
		Task:                     task,
		SystemPrompt:             judgeSystemPrompt,
		UserInput:                buildJudgeInput(loop, turnOutput),
		MaxTokens:                500,
		Timeout:                  j.timeout,
		ResponseFormatJSONSchema: verdictJSONSchema,
	})
	if err != nil {
		return Verdict{
			Decision: DecisionEscalate,
			Reason:   fmt.Sprintf("judge worker error: %v", err),
		}, err
	}
	var v Verdict
	if uerr := json.Unmarshal([]byte(extractJSON(resp.Content)), &v); uerr != nil {
		return Verdict{
			Decision: DecisionEscalate,
			Reason:   "judge returned unparseable output; escalating",
		}, nil
	}
	return v.normalise(), nil
}

func buildJudgeInput(loop Loop, turnOutput string) string {
	var b strings.Builder
	b.WriteString("GOAL:\n")
	b.WriteString(strings.TrimSpace(loop.Goal))
	b.WriteString("\n\nITERATION: ")
	fmt.Fprintf(&b, "%d of %d", loop.Iteration, loop.MaxIterations)
	if loop.LastReason != "" {
		b.WriteString("\n\nPREVIOUS VERDICT: ")
		b.WriteString(loop.LastVerdict)
		b.WriteString(" — ")
		b.WriteString(loop.LastReason)
	}
	b.WriteString("\n\nMOST RECENT TURN OUTPUT:\n")
	if strings.TrimSpace(turnOutput) == "" {
		b.WriteString("(the agent produced no visible output this turn)")
	} else {
		b.WriteString(turnOutput)
	}
	return b.String()
}

// extractJSON pulls the first {...} block out of a worker response. Agent
// workers sometimes wrap JSON in prose or a ```json fence; summarizer workers
// using response_format return clean JSON. Defensive either way.
func extractJSON(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
