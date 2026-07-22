package roundtable

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Role-based collaborative execution: after the discussion, the group's work is
// broken into an ordered plan, each step assigned to the member whose strength
// fits it (claude=code, antigravity=UI, codex=review). The operator runs steps
// one at a time; each spawns a real agent session in the shared project cwd, so
// the specialists collaborate through the working tree.

// DraftPlan asks a member to break the discussion into an ordered, role-assigned
// task list and stores it as the table's plan (overwriting any prior draft).
// Runs in the background; the plan lands asynchronously. provider empty → the
// first seat drafts.
func (s *Service) DraftPlan(ctx context.Context, id, provider string) error {
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if len(rt.Seats) == 0 {
		return fmt.Errorf("roundtable: no members to draft a plan")
	}
	provider = strings.TrimSpace(provider)
	drafter := rt.Seats[0]
	if provider != "" {
		found := false
		for _, seat := range rt.Seats {
			if seat.Provider == provider {
				drafter, found = seat, true
				break
			}
		}
		if !found {
			return fmt.Errorf("roundtable: %q is not a member of this chat", provider)
		}
	}
	go s.runDraftPlan(id, drafter)
	return nil
}

func (s *Service) runDraftPlan(id string, drafter Seat) {
	ctx, cancel := context.WithTimeout(context.Background(), roundTimeout)
	defer cancel()
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		s.log.Error("roundtable plan: load failed", "id", id, "err", err)
		return
	}
	transcript, err := s.buildTranscript(ctx, rt, drafter.Provider)
	if err != nil {
		s.memberFailed(ctx, id, drafter, err)
		return
	}
	resp, err := s.invokeSeat(ctx, drafter, planSystemPrompt(rt), transcript, summaryMaxTokens, rt.Cwd)
	if err != nil {
		s.memberFailed(ctx, id, drafter, err)
		return
	}
	steps := parsePlan(resp, rt.Seats)
	if len(steps) == 0 {
		s.memberFailed(ctx, id, drafter, fmt.Errorf("could not parse a plan from the reply"))
		return
	}
	if err := s.store.SetPlan(ctx, id, steps); err != nil {
		s.log.Warn("roundtable: set plan failed", "id", id, "err", err)
		return
	}
	_, _ = s.store.AppendMessage(ctx, Message{
		RoundTableID: id, Role: RoleSystem,
		Content: fmt.Sprintf("Drafted a %d-step work plan. Review it, then run the steps in order.", len(steps)),
	})
	s.announce(id)
}

// RunStep launches (or, for a repeat of the same assignee's still-alive
// session, continues) a real agent session to carry out one plan step in the
// shared cwd. Marks the step running, auto-completes any earlier running steps
// (the operator moved on), and records the session id. Returns it so the UI can
// jump to the live session.
func (s *Service) RunStep(ctx context.Context, id string, index int, cwd, accountID string, args []string) (string, error) {
	if s.sessions == nil {
		return "", ErrHandoffUnavailable
	}
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if index < 0 || index >= len(rt.Plan) {
		return "", fmt.Errorf("roundtable: step %d out of range", index)
	}
	step := rt.Plan[index]

	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		cwd = strings.TrimSpace(rt.Cwd)
	}
	if cwd == "" {
		return "", fmt.Errorf("roundtable: a project path is required to run a step")
	}

	// Account: an explicit per-run choice wins (multi-account picker); else the
	// step's pin; else the matching seat's. Model inherits step → seat.
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		accountID = step.AccountID
	}
	model := step.Model
	for _, seat := range rt.Seats {
		if seat.Provider == step.Assignee {
			if model == "" {
				model = seat.Model
			}
			if accountID == "" {
				accountID = seat.AccountID
			}
			break
		}
	}

	seed := s.buildStepSeed(ctx, rt, index)
	name := fmt.Sprintf("round table %s: step %d (%s)",
		firstNonEmpty(rt.Topic, "chat"), index+1, step.Assignee)
	sid, err := s.sessions.Launch(ctx, LaunchSpec{
		Cwd: cwd, Name: name, SeedPrompt: seed,
		ProviderID: step.Assignee, Model: model, AccountID: accountID, Args: args,
	})
	if err != nil {
		return "", fmt.Errorf("roundtable: launch step session: %w", err)
	}

	// Mark this step running + record its session; auto-complete any earlier
	// running step (the operator advanced past it).
	plan := make([]PlanStep, len(rt.Plan))
	copy(plan, rt.Plan)
	for i := range plan {
		if i < index && plan[i].Status == StepRunning {
			plan[i].Status = StepDone
		}
	}
	plan[index].Status = StepRunning
	plan[index].SessionID = sid
	_ = s.store.SetPlan(ctx, id, plan)
	_ = s.store.SetResultingSession(ctx, id, sid)

	_, _ = s.store.AppendMessage(ctx, Message{
		RoundTableID: id, Role: RoleSystem,
		Content: fmt.Sprintf("Step %d — %s is working on it in session %s.", index+1, step.Assignee, sid),
	})
	s.announce(id)
	return sid, nil
}

// buildStepSeed frames one step for its assignee: the agreed plan, what the
// earlier specialists already did (in the shared working tree), and this step's
// specific task + what to leave for whoever comes next.
func (s *Service) buildStepSeed(ctx context.Context, rt RoundTable, index int) string {
	step := rt.Plan[index]
	var summary string
	if msgs, err := s.store.Messages(ctx, rt.ID, 500); err == nil {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Kind == KindSummary {
				summary = strings.TrimSpace(msgs[i].Content)
				break
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "You are %q (%s), one specialist in a multi-model team implementing %q.\n\n",
		step.Assignee, vendorLabel(step.Assignee), firstNonEmpty(rt.Topic, "a group decision"))
	if strings.TrimSpace(rt.Framing) != "" {
		fmt.Fprintf(&b, "Team framing: %s\n\n", strings.TrimSpace(rt.Framing))
	}
	if summary != "" {
		b.WriteString("The agreed plan (from the discussion):\n")
		b.WriteString(summary)
		b.WriteString("\n\n")
	}
	b.WriteString("The full work plan, by step:\n")
	for i, st := range rt.Plan {
		marker := " "
		switch {
		case i == index:
			marker = "▶"
		case st.Status == StepDone:
			marker = "✓"
		}
		fmt.Fprintf(&b, "%s %d. [%s] %s\n", marker, i+1, st.Assignee, st.Task)
	}
	b.WriteString("\n")
	if index > 0 {
		b.WriteString("Earlier steps have already run in THIS project — read the working tree to see what your teammates produced before you build on it.\n\n")
	}
	fmt.Fprintf(&b, "YOUR STEP (%d): %s\n\n", index+1, step.Task)
	b.WriteString("Do exactly this step in this project: read the relevant files first, make the changes, and keep your work focused on your part. ")
	if index+1 < len(rt.Plan) {
		next := rt.Plan[index+1]
		fmt.Fprintf(&b, "When done, leave a short note of what you changed and what %s needs for the next step. ", next.Assignee)
	} else {
		b.WriteString("When done, summarize what you changed. ")
	}
	b.WriteString("If the task is ambiguous or a change is large/destructive, ask before proceeding.")
	return b.String()
}

// parsePlan extracts an ordered plan from a member's reply. It expects a JSON
// array of {assignee, task} (fenced or bare); unknown assignees fall back to the
// first seat so a near-miss still produces a runnable plan. Model/account are
// inherited from the matching seat.
func parsePlan(reply string, seats []Seat) []PlanStep {
	if len(seats) == 0 {
		return nil
	}
	raw := extractJSONArray(reply)
	if raw == "" {
		return nil
	}
	var items []struct {
		Assignee string `json:"assignee"`
		Task     string `json:"task"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	seatByProvider := map[string]Seat{}
	for _, s := range seats {
		seatByProvider[s.Provider] = s
	}
	var out []PlanStep
	for _, it := range items {
		task := strings.TrimSpace(it.Task)
		if task == "" {
			continue
		}
		assignee := strings.TrimSpace(it.Assignee)
		seat, ok := seatByProvider[assignee]
		if !ok {
			seat = seats[0] // near-miss → hand to the first member
		}
		out = append(out, PlanStep{
			Assignee: seat.Provider, Model: seat.Model, AccountID: seat.AccountID,
			Task: task, Status: StepPending,
		})
	}
	return out
}

// extractJSONArray returns the first top-level [...] block in s (tolerating
// prose or ```json fences around it), or "" if none is found.
func extractJSONArray(s string) string {
	start := strings.IndexByte(s, '[')
	end := strings.LastIndexByte(s, ']')
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}
