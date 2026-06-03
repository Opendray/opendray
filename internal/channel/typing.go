package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// TurnExpecter is an optional companion to SessionInputter: a session
// backend that can tell the hub "watch this session and let me know
// when its next reply turn settles". The session.Manager satisfies it
// via ExpectTurn. When the wired SessionInputter does not also
// implement this, the hub simply skips the typing/turn machinery and
// behaves exactly as before (idle-card notifications only).
type TurnExpecter interface {
	ExpectTurn(sessionID string)
}

// typingCap bounds how long a "typing…" indicator runs without a
// turn-complete signal. A turn normally settles in seconds; the cap is
// the safety net for a session that never replies (hung, waiting on a
// human, crashed mid-turn) so the chat doesn't show a perpetual fake
// "typing…". On expiry we stop the indicator and post a short note but
// KEEP the pending entry, so the reply is still delivered when the turn
// finally settles (see onReplyTimeout) — only a session end / supersede
// tears it down.
const typingCap = 90 * time.Second

// pendingReply is a chat message awaiting its agent reply. One per
// (channel, session) — a newer message for the same pair supersedes
// the older one (re-arm + reset cap).
type pendingReply struct {
	channelID  string
	sessionID  string
	src        ChannelMessage // the user's message, for reply threading
	stopTyping func()         // cancels the "typing…" indicator
	timer      *time.Timer    // the typingCap safety net
}

func pendingKey(channelID, sessionID string) string {
	return channelID + "\x00" + sessionID
}

// sessionControlButtons is the quick-action row attached to a chat
// reply card so the operator can drive the session from their phone
// without retyping its id. Stop/Restart route through /confirm (a
// fat-fingered tap must never interrupt a live session — see the
// confirm command); Switch opens the /list picker to retarget which
// session the chat talks to.
func sessionControlButtons(sessionID string) []ButtonOption {
	return []ButtonOption{
		{Text: "⏸ Stop", Value: "cmd:/confirm stop " + sessionID, Style: "danger"},
		{Text: "🔄 Restart", Value: "cmd:/confirm restart " + sessionID},
		{Text: "🔀 Switch", Value: "cmd:/list"},
	}
}

// trimReply caps an agent turn reply at max characters, keeping the
// HEAD of the reply (it reads top-to-bottom, unlike idle cards which
// keep the most-recent tail) and cutting on a line boundary when one is
// reasonably close to the cap. Returns the trimmed body and a footer
// note describing how much was dropped (empty when nothing was). max<=0
// means unlimited — the reply is left whole for the per-platform
// chunker to split.
func trimReply(s string, max int) (body, footer string) {
	s = strings.TrimSpace(s)
	if max <= 0 {
		return s, ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s, ""
	}
	cut := string(runes[:max])
	// Prefer a line boundary so we don't slice mid-sentence, but only
	// when one exists past the halfway mark (else we'd drop too much).
	if i := strings.LastIndexByte(cut, '\n'); i > max/2 {
		cut = cut[:i]
	}
	cut = strings.TrimRight(cut, "\n ")
	omitted := len(runes) - len([]rune(cut))
	footer = fmt.Sprintf("\n\n… (truncated %d more characters — open the dashboard for the full reply)", omitted)
	return cut, footer
}

// buildReplyCard renders an agent turn as a chat-style reply: the
// response text followed by the session-control action row. Kept
// header-less so it reads like a message, not an alert.
func buildReplyCard(sessionID, reply string) *Card {
	return &Card{
		Elements: []CardElement{
			CardMarkdown{Content: reply},
			CardActions{Buttons: [][]ButtonOption{sessionControlButtons(sessionID)}},
		},
	}
}

// beginReplyWait is called right after a chat message has been
// submitted into a session's stdin. If the backend supports turn
// detection it arms the session and registers a pending reply (with a
// cap timer) so the agent's reply is delivered as a prompt chat
// message. When typingOn and the channel supports it, a "typing…"
// indicator runs until the reply settles. A no-op when the backend
// can't report turn completion — the reply then arrives via the idle
// card, as before.
func (h *Hub) beginReplyWait(ch Channel, src ChannelMessage, sessionID string, typingOn bool) {
	expecter, ok := h.input.(TurnExpecter)
	if !ok {
		return
	}

	expecter.ExpectTurn(sessionID)

	// Typing indicator is optional (per-channel chat_typing). Default to
	// a no-op stop so teardown is uniform whether or not it's running.
	stop := func() {}
	if typingOn {
		if typer, ok := ch.(TypingIndicator); ok {
			// context.Background (not the inbound request ctx): the
			// indicator must outlive delivery of the inbound message and
			// is torn down explicitly on turn-complete / cap / session end.
			stop = typer.StartTyping(context.Background(), src)
		}
	}

	key := pendingKey(ch.ID(), sessionID)
	h.pendingMu.Lock()
	if prev := h.pending[key]; prev != nil {
		prev.stop()
	}
	pr := &pendingReply{
		channelID:  ch.ID(),
		sessionID:  sessionID,
		src:        src,
		stopTyping: stop,
	}
	pr.timer = time.AfterFunc(typingCap, func() { h.onReplyTimeout(key) })
	h.pending[key] = pr
	h.pendingMu.Unlock()
}

// stop tears down the indicator and the cap timer. Must be called with
// pendingMu held (callers below do). Safe to call once per entry.
func (pr *pendingReply) stop() {
	if pr.stopTyping != nil {
		pr.stopTyping()
	}
	if pr.timer != nil {
		pr.timer.Stop()
	}
}

// takePending removes and returns the pending entries for a session
// across all channels (a session is usually driven from one chat, but
// keying by channel keeps multi-channel honest). Caller owns teardown.
func (h *Hub) takePendingForSession(sessionID string) []*pendingReply {
	h.pendingMu.Lock()
	defer h.pendingMu.Unlock()
	var out []*pendingReply
	for key, pr := range h.pending {
		if pr.sessionID == sessionID {
			out = append(out, pr)
			delete(h.pending, key)
		}
	}
	return out
}

// onReplyTimeout fires when typingCap elapses with no turn-complete.
// Stops the indicator and posts a brief "still working" note so the
// user knows the bot is alive; the eventual output arrives via the
// idle card.
func (h *Hub) onReplyTimeout(key string) {
	h.pendingMu.Lock()
	pr := h.pending[key]
	if pr == nil {
		h.pendingMu.Unlock()
		return
	}
	// Do NOT drop the pending here: a long turn that merely exceeded the
	// typing cap still settles eventually, and deliverTurnReply must find
	// this entry to post the reply. Dropping it (the old behaviour) lost
	// the reply for any turn longer than typingCap, because the idle-card
	// fallback is unreliable under once-mode. We only stop the visual
	// "typing…" indicator here (the cap timer has already fired); swap
	// stopTyping for a no-op so a later pr.stop() doesn't double-cancel.
	if pr.stopTyping != nil {
		pr.stopTyping()
		pr.stopTyping = func() {}
	}
	h.pendingMu.Unlock()

	h.mu.RLock()
	ch := h.channels[pr.channelID]
	h.mu.RUnlock()
	if ch == nil {
		return
	}
	h.replyText(context.Background(), ch, pr.src,
		"⏳ Still working — I'll post the result when it settles.")
}

// deliverTurnReply handles session.turn_completed: it stops the
// "typing…" indicator for any chat waiting on this session and posts
// the agent's reply as a threaded chat message with a session-control
// action row. Sessions nobody is waiting on are ignored (the event is
// only armed by a chat message, so this is just a late-cap race).
func (h *Hub) deliverTurnReply(ctx context.Context, ev eventbus.Event) {
	sessionID := sessionIDFromEvent(ev)
	if sessionID == "" {
		return
	}
	pendings := h.takePendingForSession(sessionID)
	if len(pendings) == 0 {
		return
	}
	reply := ""
	if data, ok := ev.Data.(map[string]any); ok {
		reply, _ = data["recent_output"].(string)
	}
	reply = strings.TrimSpace(reply)

	for _, pr := range pendings {
		pr.stop()
		h.mu.RLock()
		ch := h.channels[pr.channelID]
		h.mu.RUnlock()
		if ch == nil {
			continue
		}
		// This session is now the chat's current target, so a bare
		// keyboard tap (Stop/Restart) and a reply-to-bubble both resolve
		// to it. In a chat-only flow (notifications off) nothing else
		// records this — the broadcast path that normally sets lastSess
		// never runs.
		h.lastSessMu.Lock()
		h.lastSess[pr.channelID] = sessionID
		h.lastSessMu.Unlock()

		if reply == "" {
			// Turn settled but produced no extractable text (e.g. a
			// pure tool run). Acknowledge rather than leave the user
			// hanging after the typing indicator vanishes.
			h.replyControlText(ctx, ch, pr.src, "✅ Done — no text output for that turn.")
			continue
		}
		// Record the full reply for idle-card dedup (the follow-up idle
		// card echoes the same untrimmed output), then trim what we
		// actually post per the channel's reply_max_chars.
		h.markDelivered(sessionID, reply)
		body, footer := trimReply(reply, h.chatConfigFor(ctx, pr.channelID).replyMaxChars())
		card := buildReplyCard(sessionID, body+footer)
		out := ChannelMessage{
			ChannelID:      pr.channelID,
			Direction:      DirectionOutbound,
			ConversationID: pr.src.ConversationID,
			Text:           card.RenderText(),
			Timestamp:      time.Now().UTC(),
			ReplyCtx:       pr.src.ReplyCtx,
			// Ask Telegram to attach its persistent control keyboard (and
			// drop the card's inline row); other channels ignore the flag
			// and render the inline buttons from buildReplyCard.
			Metadata: map[string]any{MetaControlKeyboard: true},
		}
		if err := h.sendWithFallback(ctx, ch, out, card); err != nil {
			h.log.Error("turn reply send failed", "channel", pr.channelID, "session", sessionID, "err", err)
			continue
		}
		// Index the reply bubble so a reply *to it* routes back to this
		// session (the transport stamped outbound_msg_id during send).
		h.recordOutbound(pr.channelID, out.Metadata, sessionID)
		if h.store != nil {
			if _, err := h.store.InsertMessage(ctx, out); err != nil {
				h.log.Warn("turn reply persist failed", "err", err)
			}
		}
	}
}

// cancelReplyWait tears down any pending typing for a session that has
// ended/stopped/interrupted, so the indicator doesn't run to the cap.
func (h *Hub) cancelReplyWait(_ context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	for _, pr := range h.takePendingForSession(sessionID) {
		pr.stop()
	}
}

// markDelivered records the reply text last shown for a session so the
// follow-up idle card can suppress an identical echo (see dispatch).
func (h *Hub) markDelivered(sessionID, text string) {
	h.lastDeliveredMu.Lock()
	h.lastDelivered[sessionID] = text
	h.lastDeliveredMu.Unlock()
}

// alreadyDelivered reports whether text matches the last reply shown
// for this session (so an idle card doesn't repeat a turn reply).
func (h *Hub) alreadyDelivered(sessionID, text string) bool {
	if sessionID == "" || text == "" {
		return false
	}
	h.lastDeliveredMu.Lock()
	defer h.lastDeliveredMu.Unlock()
	return h.lastDelivered[sessionID] == text
}
