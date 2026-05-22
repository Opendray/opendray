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
		// CardHandler so Telegram (and other CardSender channels)
		// can render tap-to-act buttons per session — the nano-id
		// style session IDs are too long to retype on a phone, so
		// the buttons are the *intended* operating interface.
		CardHandler: listSessionsCardHandler(mgr),
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
	hub.RegisterCommand(channel.Command{
		Name: "select",
		Description: "Talk to a session: /select <session_id> " +
			"(pin where your messages go; /select off to clear)",
		Source:  "builtin",
		Handler: selectSessionHandler(mgr),
	})
}

// listSessionsMax caps how many rows /list returns. Telegram's
// inline keyboard supports ~100 buttons total; we emit at most 2
// per session (End + Resume — though only one applies at a time),
// so 12 sessions leaves comfortable headroom.
const listSessionsMax = 12

// sessionShortID returns the human-pronounceable head of a session
// id. Full nano IDs (`ses_jwwDK7iAGqA-`) are awkward in chat button
// labels; the first 6 chars after `ses_` are still distinctive
// across a typical operator's active set.
func sessionShortID(full string) string {
	stripped := strings.TrimPrefix(full, "ses_")
	const head = 6
	if len(stripped) <= head {
		return stripped
	}
	return stripped[:head] + "…"
}

func listSessionsCardHandler(mgr sessionOps) channel.CommandCardHandler {
	return func(ctx context.Context, _ channel.CommandContext) (*channel.Card, error) {
		all, err := mgr.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list sessions: %w", err)
		}
		if len(all) == 0 {
			return &channel.Card{
				Elements: []channel.CardElement{
					channel.CardMarkdown{Content: "No sessions yet."},
				},
			}, nil
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

		// Body: numbered list. We lead with the human-given name
		// when one exists — operators think in names, not nano ids —
		// and keep the full id on the line so it stays copyable for
		// `/end <full>`. The buttons below also carry the full id in
		// their callback data. Sessions started without a name fall
		// back to the bare id (there's nothing friendlier to show).
		var b strings.Builder
		// Header reads naturally whether or not terminated sessions are
		// mixed in: "3 sessions:" when all shown are live, otherwise
		// "0 active · 4 shown (incl. ended):" so "0 ... showing 4" can't
		// look like a contradiction.
		if liveCount == len(all) {
			fmt.Fprintf(&b, "%d session%s:\n", liveCount, plural(liveCount))
		} else {
			fmt.Fprintf(&b, "%d active · %d shown (incl. ended):\n", liveCount, len(all))
		}
		now := time.Now().UTC()
		for i, s := range all {
			label := s.ID
			if s.Name != "" {
				label = fmt.Sprintf("%s (%s)", s.Name, s.ID)
			}
			fmt.Fprintf(&b, "%d. %s — %s — %s — %s\n",
				i+1,
				label,
				s.ProviderID,
				s.State,
				relativeAge(sessionActivityTime(s), now),
			)
		}

		// Buttons: live sessions get a "Talk to" row (pins this chat's
		// active session so plain messages route there) and an End
		// row; terminated ones get a Resume row. Two per row keeps tap
		// targets large on mobile without forcing the keyboard taller
		// than the screen.
		var talkRow, endRow, resumeRow []channel.ButtonOption
		for i, s := range all {
			label := fmt.Sprintf("%d %s", i+1, sessionShortID(s.ID))
			if isLiveState(s.State) {
				talkRow = append(talkRow, channel.ButtonOption{
					Text:  "💬 Talk to " + label,
					Value: "cmd:/select " + s.ID,
					Style: "primary",
				})
				endRow = append(endRow, channel.ButtonOption{
					Text:  "End " + label,
					Value: "cmd:/end " + s.ID,
					Style: "danger",
				})
			} else {
				resumeRow = append(resumeRow, channel.ButtonOption{
					Text:  "Resume " + label,
					Value: "cmd:/resume " + s.ID,
					Style: "primary",
				})
			}
		}
		buttons := make([][]channel.ButtonOption, 0, 6)
		buttons = append(buttons, chunkButtons(talkRow, 2)...)
		buttons = append(buttons, chunkButtons(endRow, 2)...)
		buttons = append(buttons, chunkButtons(resumeRow, 2)...)

		elements := []channel.CardElement{
			channel.CardMarkdown{Content: strings.TrimRight(b.String(), "\n")},
		}
		if len(buttons) > 0 {
			elements = append(elements, channel.CardActions{Buttons: buttons})
		}
		return &channel.Card{Elements: elements}, nil
	}
}

// chunkButtons rolls a flat slice into rows of at most `per`
// buttons each — Telegram's inline keyboard is a 2D matrix and
// dense rows scale better on mobile than one-per-row stacks.
func chunkButtons(in []channel.ButtonOption, per int) [][]channel.ButtonOption {
	if len(in) == 0 {
		return nil
	}
	out := make([][]channel.ButtonOption, 0, (len(in)+per-1)/per)
	for i := 0; i < len(in); i += per {
		end := i + per
		if end > len(in) {
			end = len(in)
		}
		out = append(out, in[i:end])
	}
	return out
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

// selectSessionHandler pins the chat's active session so plain
// (non-reply) messages route to it — the interactive way to switch
// between sessions. Backs the /select command and the "💬 Talk to"
// buttons on /list. No argument reports the current pin; "off"
// (or clear/none) removes it.
func selectSessionHandler(mgr sessionOps) channel.CommandHandler {
	return func(ctx context.Context, cc channel.CommandContext) (string, error) {
		chID := ""
		if cc.Channel != nil {
			chID = cc.Channel.ID()
		}
		if cc.Hub == nil || chID == "" {
			return "Session selection isn't available on this channel.", nil
		}

		arg, ok := singleSessionArg(cc.Args)
		if !ok {
			cur := cc.Hub.ActiveSession(chID)
			if cur == "" {
				return "No session selected. Use /list and tap “💬 Talk to”, or /select <session_id>.", nil
			}
			return fmt.Sprintf("Currently talking to %s. Send /select off to clear.",
				sessionDisplay(ctx, mgr, cur)), nil
		}

		switch strings.ToLower(arg) {
		case "off", "clear", "none":
			cc.Hub.SetActiveSession(chID, "")
			return "Cleared — messages now follow the most recent session again.", nil
		}

		// Validate against the live set so we never pin a stale/typo'd id
		// that would silently swallow the chat's messages.
		sess, found, err := findSession(ctx, mgr, arg)
		if err != nil {
			return "", fmt.Errorf("select %s: %w", arg, err)
		}
		if !found {
			return "Session " + arg + " not found — /list shows current sessions.", nil
		}
		if !isLiveState(sess.State) {
			return fmt.Sprintf("Session %s is %s — /resume it first.",
				sessionLabel(sess), sess.State), nil
		}
		cc.Hub.SetActiveSession(chID, sess.ID)
		return fmt.Sprintf("✅ Now talking to %s. Your messages go here until you /select another (or /select off).",
			sessionLabel(sess)), nil
	}
}

// findSession returns the session with the given id from the manager's
// current list.
func findSession(ctx context.Context, mgr sessionOps, id string) (session.Session, bool, error) {
	all, err := mgr.List(ctx)
	if err != nil {
		return session.Session{}, false, err
	}
	for _, s := range all {
		if s.ID == id {
			return s, true, nil
		}
	}
	return session.Session{}, false, nil
}

// sessionLabel renders a session as "name (id)", or the bare id when
// it has no name — matching how /list presents sessions.
func sessionLabel(s session.Session) string {
	if s.Name != "" {
		return fmt.Sprintf("%s (%s)", s.Name, s.ID)
	}
	return s.ID
}

// sessionDisplay looks a session up by id for display, falling back to
// the bare id when it's no longer listed.
func sessionDisplay(ctx context.Context, mgr sessionOps, id string) string {
	if s, ok, err := findSession(ctx, mgr, id); err == nil && ok {
		return sessionLabel(s)
	}
	return id
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
