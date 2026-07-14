package roundtable

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LaunchSpec describes the real agent session to spawn when handing a
// discussion off for execution. The executor runs with FULL tool access
// (unlike the read-only headless chat members), so it can actually read,
// edit, and add files.
type LaunchSpec struct {
	Cwd        string
	Name       string
	SeedPrompt string
	ProviderID string
	Model      string
	AccountID  string
}

// SessionLauncher spawns a full agent session seeded with a prompt.
// Implemented in the app over session.Manager — roundtable never imports
// internal/session.
type SessionLauncher interface {
	Launch(ctx context.Context, spec LaunchSpec) (sessionID string, err error)
}

// WithSessionLauncher enables the execution handoff. Nil → Handoff 503s.
func (s *Service) WithSessionLauncher(l SessionLauncher) *Service {
	s.sessions = l
	return s
}

// ErrHandoffUnavailable is returned when no session launcher is wired.
var ErrHandoffUnavailable = errors.New("roundtable: handoff not configured")

// Handoff spawns a real agent session in cwd, seeded with the discussion, to
// actually implement the changes the group agreed on. The round table
// members only ever chat (read-only, headless); this is the bridge from
// "议" to "做". Returns the new session id (link to it in the sessions view).
//
// The seed is the latest summary if one exists, else the whole transcript.
// The executor provider is operator-chosen; when it matches a seated member
// its model/account are inherited unless overridden.
func (s *Service) Handoff(ctx context.Context, id, cwd, provider, model, accountID string) (string, error) {
	if s.sessions == nil {
		return "", ErrHandoffUnavailable
	}
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		cwd = strings.TrimSpace(rt.Cwd)
	}
	if cwd == "" {
		return "", errors.New("roundtable: a project path is required to execute")
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return "", errors.New("roundtable: pick an executor")
	}
	// Inherit the seat's model/account when the executor is one of the
	// discussion's members and the caller didn't override them.
	for _, seat := range rt.Seats {
		if seat.Provider == provider {
			if model == "" {
				model = seat.Model
			}
			if accountID == "" {
				accountID = seat.AccountID
			}
			break
		}
	}

	seed, err := s.buildHandoffSeed(ctx, rt)
	if err != nil {
		return "", err
	}
	name := "round table: " + firstNonEmpty(rt.Topic, "chat")
	sid, err := s.sessions.Launch(ctx, LaunchSpec{
		Cwd: cwd, Name: name, SeedPrompt: seed,
		ProviderID: provider, Model: model, AccountID: accountID,
	})
	if err != nil {
		return "", fmt.Errorf("roundtable: launch session: %w", err)
	}
	_ = s.store.SetResultingSession(ctx, id, sid)
	_, _ = s.store.AppendMessage(ctx, Message{
		RoundTableID: id, Role: RoleSystem,
		Content: fmt.Sprintf("Handed off to session %s in %s to implement the plan.", sid, cwd),
	})
	s.announce(id)
	return sid, nil
}

// buildHandoffSeed prefers the most recent summary (concise, decisive);
// falls back to the full transcript when the operator never summarized.
func (s *Service) buildHandoffSeed(ctx context.Context, rt RoundTable) (string, error) {
	msgs, err := s.store.Messages(ctx, rt.ID, 500)
	if err != nil {
		return "", fmt.Errorf("load messages: %w", err)
	}

	var summary string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Kind == KindSummary {
			summary = strings.TrimSpace(msgs[i].Content)
			break
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "You are taking over from a multi-model group chat about %q to IMPLEMENT what was agreed.\n\n",
		firstNonEmpty(rt.Topic, "the discussion"))
	if summary != "" {
		b.WriteString("Here is the agreed plan (a summary of the discussion):\n\n")
		b.WriteString(summary)
		b.WriteString("\n\n")
	} else {
		b.WriteString("Here is the full discussion:\n\n")
		for _, m := range msgs {
			b.WriteString(speakerLabel(m, ""))
			b.WriteString(": ")
			b.WriteString(strings.TrimSpace(m.Content))
			b.WriteString("\n\n")
		}
	}
	b.WriteString("Now do the work in THIS project: read the relevant files first, then make the edits / add the files the plan calls for, and explain what you changed. ")
	b.WriteString("If the plan is ambiguous or a change is large/destructive, ask before proceeding.")
	return b.String(), nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
