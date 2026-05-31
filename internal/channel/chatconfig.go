package channel

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

// chatConfig is the subset of a channel's config JSON that governs
// two-way chat behavior. Every field is optional with a safe default,
// and all of it is editable from the dashboard Channels form (the keys
// match the Telegram KindDef fields) — nothing here is hardcoded.
type chatConfig struct {
	// OwnerUserIDs is an allowlist of platform user ids permitted to
	// interact with the bot (drive sessions, run commands, tap buttons).
	// Comma / space / newline separated. Empty = no per-channel gate
	// (falls back to the global env authorizer; see authorizeSender).
	OwnerUserIDs string `json:"owner_user_ids"`
	// ChatEnabled routes inbound plain text into a session's stdin.
	// nil/absent = enabled (preserves existing behavior).
	ChatEnabled *bool `json:"chat_enabled"`
	// ChatTyping shows the "typing…" indicator while awaiting a reply.
	// nil/absent = enabled.
	ChatTyping *bool `json:"chat_typing"`
	// NotifyEnabled controls the idle / ended / PR broadcast cards. When
	// two-way chat is on these are pure noise (every session dumps its
	// running output every idle window), so the default is OFF whenever
	// chat is enabled. nil/absent = default-by-mode; set true to opt back
	// in to activity notifications.
	NotifyEnabled *bool `json:"notify_enabled"`
	// ReplyMaxChars caps how much of an agent's turn reply is sent to the
	// chat before it's trimmed with a "…(truncated)" footer. Kept as raw
	// JSON because the dashboard form submits it as a string ("3500") but
	// a hand-edited config may store a number — replyMaxChars() parses
	// either. Absent = defaultReplyMaxChars; "0" = unlimited (the reply is
	// chunked into multiple messages instead).
	ReplyMaxChars json.RawMessage `json:"reply_max_chars,omitempty"`
}

// defaultReplyMaxChars trims a turn reply to roughly one Telegram
// message by default, so a runaway agent response can't dump dozens of
// chunks into the chat. Operators raise it (or set 0 for unlimited)
// from the dashboard.
const defaultReplyMaxChars = 3500

// chatConfigFor reads the chat-related config for a channel. A single
// DB read per inbound message — callers pass the result down rather
// than re-reading.
func (h *Hub) chatConfigFor(ctx context.Context, channelID string) chatConfig {
	var cfg chatConfig
	if h.store == nil {
		return cfg
	}
	row, err := h.store.Get(ctx, channelID)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(row.Config, &cfg)
	return cfg
}

// chatEnabled reports whether inbound text should be routed to a
// session (default true).
func (c chatConfig) chatEnabled() bool { return c.ChatEnabled == nil || *c.ChatEnabled }

// typingEnabled reports whether to show the typing indicator (default
// true).
func (c chatConfig) typingEnabled() bool { return c.ChatTyping == nil || *c.ChatTyping }

// notificationsEnabled reports whether idle/ended/PR broadcast cards
// should be sent. Explicit setting wins; otherwise notifications are on
// only when two-way chat is OFF — so a chat channel stays clean unless
// the operator opts in.
func (c chatConfig) notificationsEnabled() bool {
	if c.NotifyEnabled != nil {
		return *c.NotifyEnabled
	}
	return !c.chatEnabled()
}

// replyMaxChars returns the configured turn-reply cap in characters.
// Accepts either a JSON number or a numeric string (the dashboard form
// submits text). Absent / unparesable → defaultReplyMaxChars; a
// negative value is treated as the default; 0 means unlimited.
func (c chatConfig) replyMaxChars() int {
	if len(c.ReplyMaxChars) == 0 {
		return defaultReplyMaxChars
	}
	var n int
	if err := json.Unmarshal(c.ReplyMaxChars, &n); err == nil {
		if n < 0 {
			return defaultReplyMaxChars
		}
		return n
	}
	var s string
	if err := json.Unmarshal(c.ReplyMaxChars, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return defaultReplyMaxChars
		}
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			return v
		}
	}
	return defaultReplyMaxChars
}

// ownerSet parses OwnerUserIDs into a lookup set. Empty when no owners
// are configured.
func (c chatConfig) ownerSet() map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.FieldsFunc(c.OwnerUserIDs, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t' || r == '\r'
	}) {
		if f != "" {
			out[f] = true
		}
	}
	return out
}
