package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/channel"
	"github.com/opendray/opendray-v2/internal/session"
)

// sessionOps is the small slice of session.Manager that the chat
// commands need. Extracted as an interface so tests can stand in a
// fake without spinning up a real PTY-backed Manager.
type sessionOps interface {
	List(ctx context.Context) ([]session.Session, error)
	Start(ctx context.Context, id string) (session.Session, error)
	Stop(ctx context.Context, id string) error
}

// registerChannelCommands wires the session-aware slash commands
// (/list, /end, /resume) into the channel hub. We do it here rather
// than inside the channel package so internal/channel stays free of
// the session dependency — the channel layer is a transport, not a
// session controller.
//
// The deliberate set is small. Operators get tired of long /help
// dumps; chat-side actions are emergency / quick-check only, and
// the web admin remains the authoritative session console.
//
//   - /list           — what's alive right now
//   - /end <id>       — stop a running session (used by the End
//     inline button on the idle card as well)
//   - /resume <id>    — re-spawn a stopped/ended session under the
//     same id (used by the Resume inline button)
func registerChannelCommands(hub *channel.Hub, mgr sessionOps) {
	hub.RegisterCommand(channel.Command{
		Name:        "list",
		Description: "List active sessions",
		Source:      "builtin",
		Handler:     listSessionsHandler(mgr),
	})
	hub.RegisterCommand(channel.Command{
		Name:        "end",
		Description: "End a session: /end <session_id>",
		Source:      "builtin",
		Handler:     endSessionHandler(mgr),
	})
	hub.RegisterCommand(channel.Command{
		Name:        "resume",
		Description: "Resume a stopped or ended session: /resume <session_id>",
		Source:      "builtin",
		Handler:     resumeSessionHandler(mgr),
	})
}

// listSessionsHandler returns up to `listSessionsMax` sessions
// sorted by recency. Live states (pending/running/idle) come first
// so the operator can quickly find what's chewing CPU; recently
// terminated sessions follow so /resume <id> has visible targets.
const listSessionsMax = 12

func listSessionsHandler(mgr sessionOps) channel.CommandHandler {
	return func(ctx context.Context, _ channel.CommandContext) (string, error) {
		all, err := mgr.List(ctx)
		if err != nil {
			return "", fmt.Errorf("list sessions: %w", err)
		}
		if len(all) == 0 {
			return "No sessions yet.", nil
		}
		// Live first, then terminated; within each bucket newest first.
		sort.Slice(all, func(i, j int) bool {
			if isLiveState(all[i].State) != isLiveState(all[j].State) {
				return isLiveState(all[i].State)
			}
			return sessionActivityTime(all[i]).After(sessionActivityTime(all[j]))
		})
		if len(all) > listSessionsMax {
			all = all[:listSessionsMax]
		}
		liveCount := 0
		for _, s := range all {
			if isLiveState(s.State) {
				liveCount++
			}
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%d session%s (showing %d):\n",
			liveCount, plural(liveCount), len(all))
		now := time.Now().UTC()
		for _, s := range all {
			fmt.Fprintf(&b, "  %s — %s — %s — %s\n",
				s.ID,
				s.ProviderID,
				s.State,
				relativeAge(sessionActivityTime(s), now),
			)
		}
		return strings.TrimRight(b.String(), "\n"), nil
	}
}

func endSessionHandler(mgr sessionOps) channel.CommandHandler {
	return func(ctx context.Context, cc channel.CommandContext) (string, error) {
		sid, ok := singleSessionArg(cc.Args)
		if !ok {
			return "Usage: /end <session_id>", nil
		}
		if err := mgr.Stop(ctx, sid); err != nil {
			if errors.Is(err, session.ErrNotFound) {
				return "Session " + sid + " not found.", nil
			}
			if errors.Is(err, session.ErrAlreadyEnded) {
				return "Session " + sid + " is already ended.", nil
			}
			return "", fmt.Errorf("end %s: %w", sid, err)
		}
		return "Session " + sid + " stopped.", nil
	}
}

func resumeSessionHandler(mgr sessionOps) channel.CommandHandler {
	return func(ctx context.Context, cc channel.CommandContext) (string, error) {
		sid, ok := singleSessionArg(cc.Args)
		if !ok {
			return "Usage: /resume <session_id>", nil
		}
		s, err := mgr.Start(ctx, sid)
		if err != nil {
			// Manager.Start returns a specific error when the row is
			// already live. Surface it cleanly instead of leaking the
			// pgx wrap chain.
			if errors.Is(err, session.ErrAlreadyRunning) {
				return "Session " + sid + " is already running.", nil
			}
			if errors.Is(err, session.ErrNotFound) {
				return "Session " + sid + " not found.", nil
			}
			return "", fmt.Errorf("resume %s: %w", sid, err)
		}
		return fmt.Sprintf("Session %s resumed (state=%s).", sid, s.State), nil
	}
}

// ── helpers ──────────────────────────────────────────────────────

func isLiveState(s session.State) bool {
	switch s {
	case session.StatePending, session.StateRunning, session.StateIdle:
		return true
	}
	return false
}

// sessionActivityTime returns the most relevant timestamp for a
// recency sort. EndedAt for terminated sessions; StartedAt
// otherwise (the Manager doesn't currently expose a separate
// last-activity field at this level).
func sessionActivityTime(s session.Session) time.Time {
	if s.EndedAt != nil {
		return *s.EndedAt
	}
	return s.StartedAt
}

// singleSessionArg extracts a single session ID from a command's
// args, trimming any leading "/" that operators sometimes paste in.
// Returns ok=false when no usable id was supplied.
func singleSessionArg(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	sid := strings.TrimSpace(args[0])
	sid = strings.TrimPrefix(sid, "/")
	if sid == "" {
		return "", false
	}
	return sid, true
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// relativeAge renders durations the way chat readers expect ("3m
// ago", "12h ago", "2d ago"). Same buckets as the mobile/web side
// so users don't have to mentally translate between surfaces.
func relativeAge(ts, now time.Time) string {
	if ts.IsZero() {
		return "(unknown)"
	}
	d := now.Sub(ts)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	}
}
