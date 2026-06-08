package projectdoc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// PlanDriftDetector decides, post-session, whether the project plan
// document should be updated based on what the session accomplished.
// A nil detector is a no-op — the journaler falls back to "plan only
// updates when an agent explicitly calls project_plan_set".
//
// Implementations are expected to fail closed: on LLM error, return
// (DriftOutput{ShouldPropose:false}, error) — callers log and move
// on without filing a proposal.
type PlanDriftDetector interface {
	DetectDrift(ctx context.Context, in DriftInput) (DriftOutput, error)
}

// DriftInput bundles the context the detector needs. CurrentPlan
// being empty is the "no plan yet" signal — detectors should
// short-circuit and return ShouldPropose=false rather than
// hallucinating an initial plan.
type DriftInput struct {
	// Kind is the doc the detector is reviewing — KindPlan (default) or
	// KindGoal. CurrentPlan holds that doc's current content either way.
	Kind              Kind
	Cwd               string
	CurrentPlan       string
	TranscriptSummary string
	RecentJournal     []LogEntry
}

// DriftOutput is the detector's verdict. NewPlan is the full
// replacement markdown when ShouldPropose=true; otherwise ignored.
// Reason surfaces in the operator's inbox so they can decide
// whether to approve without re-reading the transcript.
type DriftOutput struct {
	ShouldPropose bool   `json:"should_propose"`
	NewPlan       string `json:"new_plan"`
	Reason        string `json:"reason"`
}

// PlanDriftSystemPrompt is the role block the detector ships with
// every call. Exposed for reuse by callers that want to construct
// the LLM Request themselves (e.g. the app-level worker wiring).
const PlanDriftSystemPrompt = `You are a project plan reviewer.

Given a project's CURRENT PLAN document, a summary of the agent
session that just ended, and the last few journal entries, decide
whether the plan should be updated.

UPDATE the plan ONLY when one of these is true:
1. The session COMPLETED a milestone the plan listed as upcoming —
   mark it done.
2. The session UNCOVERED work the plan didn't mention but should —
   add it.
3. The plan describes future work that the session made obsolete —
   remove it.
4. The plan's phase ordering needs to shift based on what was learned.

DO NOT update when:
- The session was exploratory / informational only.
- Nothing in the plan was touched.
- The agent merely answered a question without changing the project state.

When updating, REWRITE the plan in full — your output replaces the
existing document. Preserve unchanged sections verbatim.

Respond ONLY with a JSON object matching this schema (no prose, no
code fences, no commentary):

{
  "should_propose": <boolean>,
  "new_plan":       <full markdown of the proposed replacement plan>,
  "reason":         <one short sentence shown to the operator>
}

If should_propose is false, new_plan and reason MUST still be present
but may be empty strings.`

// GoalDriftSystemPrompt is the goal-document variant. The GOAL is the
// project's long-term intent — it changes rarely, so the bar is higher
// than for the plan.
const GoalDriftSystemPrompt = `You are a project goal reviewer.

Given a project's CURRENT GOAL document (its long-term intent — what we
are ultimately building and why), a summary of the session that just
ended, and recent journal entries, decide whether the GOAL should be
updated.

UPDATE the goal ONLY when the session reveals a genuine shift in
long-term intent or scope:
1. The project's purpose / target audience changed.
2. A major capability entered or left the project's scope.
3. The goal as written is now inaccurate about what we are building.

DO NOT update for routine progress, tactics, or step-by-step work —
that belongs in the PLAN, not the goal. The goal changes rarely; when
in doubt, do NOT propose.

When updating, REWRITE the goal in full — your output replaces the
document. Preserve unchanged parts verbatim.

Respond ONLY with a JSON object (no prose, no code fences):

{
  "should_propose": <boolean>,
  "new_plan":       <full markdown of the proposed replacement goal>,
  "reason":         <one short sentence shown to the operator>
}

If should_propose is false, new_plan and reason MUST still be present
but may be empty strings.`

// DriftSystemPrompt returns the role block for the given doc kind
// (goal vs plan). Defaults to the plan prompt.
func DriftSystemPrompt(kind Kind) string {
	if kind == KindGoal {
		return GoalDriftSystemPrompt
	}
	return PlanDriftSystemPrompt
}

// ErrDetectorParse is returned when the LLM response cannot be
// decoded into DriftOutput. Callers should treat it like any other
// detector failure — log and skip the proposal.
var ErrDetectorParse = errors.New("plan drift: detector returned unparseable response")

// ParseDriftResponse extracts a DriftOutput from a raw LLM response.
// Tolerates response_format=json_schema clean JSON and the common
// failure modes where models wrap JSON in code fences or prose.
// Returns ErrDetectorParse when nothing parseable is found.
func ParseDriftResponse(raw string) (DriftOutput, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return DriftOutput{}, ErrDetectorParse
	}
	if fenced := stripJSONFence(body); fenced != "" {
		body = fenced
	}
	if i := strings.IndexByte(body, '{'); i >= 0 {
		if j := strings.LastIndexByte(body, '}'); j > i {
			body = body[i : j+1]
		}
	}
	var out DriftOutput
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return DriftOutput{}, ErrDetectorParse
	}
	if out.ShouldPropose && strings.TrimSpace(out.NewPlan) == "" {
		return DriftOutput{}, ErrDetectorParse
	}
	out.NewPlan = strings.TrimSpace(out.NewPlan)
	out.Reason = strings.TrimSpace(out.Reason)
	return out, nil
}

// stripJSONFence pulls JSON out of a ```json ... ``` or ``` ... ```
// block. Returns "" when no fence is found so callers know to use
// the original body.
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
