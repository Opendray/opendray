package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// dispatchLoop handles autoloop.* events. It maintains the set of
// loop-driven sessions (so dispatch can suppress their routine idle cards)
// and broadcasts a "needs attention" card for the milestone outcomes that an
// operator actually wants pushed — escalated and failed. A clean done is
// deliberately silent: success is visible in the Loops UI and doesn't earn a
// chat ping.
func (h *Hub) dispatchLoop(ctx context.Context, ev eventbus.Event) {
	data, _ := ev.Data.(map[string]any)
	if data == nil {
		return
	}
	loopID, _ := data["loop_id"].(string)
	sessionID, _ := data["session_id"].(string)
	status, _ := data["status"].(string)
	kind, _ := data["kind"].(string)
	reason, _ := data["reason"].(string)

	// Track which sessions are loop-driven so dispatch() can keep their
	// per-turn idle cards suppressed.
	switch ev.Topic {
	case "autoloop.started", "autoloop.resumed":
		h.markLoopSession(sessionID, true)
	case "autoloop.done", "autoloop.failed", "autoloop.escalated", "autoloop.stopped":
		h.markLoopSession(sessionID, false)
	}

	// Only the "needs a human" outcomes broadcast a card.
	switch ev.Topic {
	case "autoloop.escalated", "autoloop.failed":
	default:
		return
	}

	text := loopMilestoneText(status, kind, loopID, reason)

	h.mu.RLock()
	chs := make([]Channel, 0, len(h.channels))
	for _, c := range h.channels {
		chs = append(chs, c)
	}
	h.mu.RUnlock()

	for _, c := range chs {
		if h.isMuted(ctx, c.ID()) {
			continue
		}
		// Dedup per loop+topic under the channel's repeat policy, so a
		// flapping loop can't spam the same escalation.
		if h.suppressByPolicy(ctx, c.ID(), ev.Topic, loopID) {
			continue
		}
		msg := ChannelMessage{
			ChannelID:      c.ID(),
			Direction:      DirectionOutbound,
			ConversationID: "default",
			Text:           text,
			Timestamp:      time.Now().UTC(),
			Metadata:       map[string]any{"loop_id": loopID},
		}
		if err := h.sendWithFallback(ctx, c, msg, nil); err != nil {
			h.log.Error("loop notify send failed", "id", c.ID(), "err", err)
			continue
		}
		if _, err := h.store.InsertMessage(ctx, msg); err != nil {
			h.log.Warn("loop notify persist failed", "id", c.ID(), "err", err)
		}
	}
}

// loopMilestoneText renders a concise chat line for a loop outcome.
func loopMilestoneText(status, kind, loopID string, reason string) string {
	var b strings.Builder
	switch status {
	case "escalated":
		b.WriteString("🔁 Loop needs your attention")
	case "failed":
		b.WriteString("🔁 Loop failed")
	default:
		b.WriteString("🔁 Loop update")
	}
	if kind != "" {
		fmt.Fprintf(&b, "\n%s loop %s", kind, loopID)
	} else {
		fmt.Fprintf(&b, "\nloop %s", loopID)
	}
	if strings.TrimSpace(reason) != "" {
		fmt.Fprintf(&b, "\n%s", strings.TrimSpace(reason))
	}
	return b.String()
}

func (h *Hub) markLoopSession(sessionID string, driven bool) {
	if sessionID == "" {
		return
	}
	h.loopMu.Lock()
	defer h.loopMu.Unlock()
	if driven {
		h.loopSessions[sessionID] = true
	} else {
		delete(h.loopSessions, sessionID)
	}
}

func (h *Hub) isLoopSession(sessionID string) bool {
	h.loopMu.Lock()
	defer h.loopMu.Unlock()
	return h.loopSessions[sessionID]
}
