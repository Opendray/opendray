package channel

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// SessionInputter is the seam through which the channel hub forwards
// non-command inbound messages back into a session's stdin. The
// sessionMgr satisfies it; the hub does not import sessions to keep
// the dependency graph one-way.
type SessionInputter interface {
	Input(ctx context.Context, sessionID string, data []byte) error
}

// Hub manages the lifecycle of all configured channels in this process.
type Hub struct {
	log   *slog.Logger
	bus   *eventbus.Hub
	store *store
	cmds  *CommandRegistry
	input SessionInputter // nil = inbound text not routed to sessions

	mu        sync.RWMutex
	channels  map[string]Channel
	started   bool
	cancelOut context.CancelFunc
	outDone   chan struct{}

	notifyMu    sync.Mutex
	notifyState map[string]map[string]time.Time // channelID -> "topic|sessionID" -> sent-at

	lastSessMu sync.RWMutex
	lastSess   map[string]string // channelID -> session_id of the most-recent notification

	// outboundIndex maps the channel-platform's outbound message id
	// (string) → session_id of the notification we sent. Used so a
	// user replying to a *specific* notification in their chat
	// drives that specific session — even when the channel has
	// notified about several different sessions recently.
	//
	//   outboundIndex[channelID][outboundMsgID] = sessionID
	//
	// Bounded by outboundIndexMax per channel — oldest entries get
	// evicted when the cap is exceeded, so memory stays bounded.
	outboundMu    sync.Mutex
	outboundIndex map[string]map[string]outboundEntry

	// activeSess overrides routing for a channel when the user
	// explicitly selected a target via /select. Lookup priority:
	// reply-to > activeSess > lastSess.
	activeSessMu sync.RWMutex
	activeSess   map[string]string
}

// outboundEntry tracks a sent notification message so reply-to
// routing can find its session. ts is used for LRU eviction.
type outboundEntry struct {
	sessionID string
	ts        time.Time
}

// outboundIndexMax bounds how many outbound notifications we
// remember per channel for reply-to lookups. ~256 covers a busy
// week even on chatty channels; oldest entries evict first.
const outboundIndexMax = 256

// defaultCooldown is the per-channel cooldown applied when the
// channel's config does not explicitly set notify_cooldown_s. Keeps
// flapping CLIs (which emit periodic output between idle windows)
// from re-notifying every minute. Operators can lower or raise it
// per channel.
const defaultCooldown = 5 * time.Minute

// submitDelay is the settle window between the final typed rune
// and the carriage-return byte that fires Enter. Long enough that
// Ink-style input handlers (Gemini in particular) process the
// preceding text as keystrokes before the Enter arrives — short
// enough to feel instantaneous to the human on the other end.
// ~30 ms is the empirical sweet spot; the human-perception
// threshold for unrelated chat interactions is ~100 ms.
const submitDelay = 30 * time.Millisecond

// perRuneDelay paces the rune-by-rune typing in submitToSession.
// 5 ms is faster than any real human but slow enough that the
// Ink event loop processes each rune as its own keystroke event
// — i.e. the CLI cannot collapse the burst into a paste-mode
// classification. A 20-rune chat message ends up taking ~100 ms
// to "type", invisible against any chat round-trip.
const perRuneDelay = 5 * time.Millisecond

func NewHub(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger) *Hub {
	if log == nil {
		log = slog.Default()
	}
	h := &Hub{
		log:           log.With("component", "channel"),
		bus:           bus,
		store:         newStore(pool),
		cmds:          NewCommandRegistry(),
		channels:      make(map[string]Channel),
		notifyState:   make(map[string]map[string]time.Time),
		lastSess:      make(map[string]string),
		outboundIndex: make(map[string]map[string]outboundEntry),
		activeSess:    make(map[string]string),
	}
	h.registerBuiltinCommands()
	return h
}

// SetSessionInput installs a callback the Hub uses to forward non-
// command inbound channel messages into the originating session's
// stdin. Pass the session.Manager (which satisfies SessionInputter
// via its Input method).
func (h *Hub) SetSessionInput(in SessionInputter) { h.input = in }

// Commands returns the command registry so app code can wire additional
// session-aware commands (cancel, resume, status, ...) at startup.
func (h *Hub) Commands() *CommandRegistry { return h.cmds }

// RegisterCommand is a convenience wrapper around Commands().Register().
func (h *Hub) RegisterCommand(c Command) { h.cmds.Register(c) }

// registerBuiltinCommands wires only the channel-scoped commands
// that don't need session access. /list, /end, and /resume live in
// the app layer (internal/app) so the channel package stays free of
// the session dependency.
//
// Slash-command set (intentionally small — operators get tired of
// long /help output and unmaintained shims):
//
//	/help    — list available commands (registered by NewCommandRegistry)
//	/notify  — toggle channel notifications on/off
//	/list    — active sessions (registered by app code)
//	/end     — end a session (registered by app code)
//	/resume  — resume a stopped/ended session (registered by app code)
//
// What we DELETED and why:
//   - /status   — channel-level diagnostic ("which capabilities does
//     this channel have"). Useful exactly once, then
//     noise in the /help output. The web admin shows
//     the same info with more context.
//   - /select   — pin a chat to a specific session_id. The reply-
//     to-message routing (outboundIndex) covers the
//     multi-session case more naturally; the pin was a
//     power-user feature that nobody used.
//   - /sessions — listed sessions that had previously NOTIFIED this
//     channel, not currently-active sessions. Confused
//     operators who expected /sessions == /list.
//
// The pinned-session machinery (activeSess + lookupActiveSession +
// setActiveSession) is retained as dead-code-but-load-bearing — if
// we ever bring /select back the routing is already wired. Cheap
// to keep; ripping it out would expand the diff for no gain.
func (h *Hub) registerBuiltinCommands() {
	h.cmds.Register(Command{
		Name:        "notify",
		Description: "Toggle notifications: /notify on|off",
		Source:      "builtin",
		Handler: func(ctx context.Context, cc CommandContext) (string, error) {
			if len(cc.Args) == 0 {
				return "Usage: /notify on|off", nil
			}
			on := cc.Args[0] == "on"
			if !on && cc.Args[0] != "off" {
				return "Usage: /notify on|off", nil
			}
			if err := h.setMuted(ctx, cc.Channel.ID(), !on); err != nil {
				return "", err
			}
			if on {
				return "Notifications enabled.", nil
			}
			return "Notifications muted.", nil
		},
	})
}

// Start loads enabled channels from DB, instantiates each via its
// registered factory, calls Channel.Start, and subscribes to outbound
// session.* events. Caller must call Shutdown to stop.
func (h *Hub) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()
		return nil
	}
	h.started = true
	h.mu.Unlock()

	rows, err := h.store.List(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if !r.Enabled {
			continue
		}
		if err := h.spawn(ctx, r); err != nil {
			h.log.Error("channel start failed", "id", r.ID, "kind", r.Kind, "err", err)
		}
	}

	outCtx, cancel := context.WithCancel(context.Background())
	h.cancelOut = cancel
	h.outDone = make(chan struct{})
	go h.runOutbound(outCtx)
	return nil
}

// Shutdown stops all channels and the outbound dispatcher.
func (h *Hub) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	if !h.started {
		h.mu.Unlock()
		return nil
	}
	h.started = false
	cancel := h.cancelOut
	done := h.outDone
	chs := make([]Channel, 0, len(h.channels))
	for _, c := range h.channels {
		chs = append(chs, c)
	}
	h.channels = make(map[string]Channel)
	h.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, c := range chs {
		if err := c.Stop(ctx); err != nil {
			h.log.Error("channel stop", "id", c.ID(), "err", err)
		}
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (h *Hub) spawn(ctx context.Context, r channelRow) error {
	factory := Lookup(r.Kind)
	if factory == nil {
		return fmt.Errorf("%w: %s", ErrUnknownKind, r.Kind)
	}
	ch, err := factory(r.ID, r.Config, h.log)
	if err != nil {
		return fmt.Errorf("factory: %w", err)
	}
	if err := ch.Start(ctx, h.handleInbound); err != nil {
		return fmt.Errorf("channel start: %w", err)
	}
	h.mu.Lock()
	h.channels[r.ID] = ch
	h.mu.Unlock()
	return nil
}

// handleInbound is invoked by Channel impls when a message arrives.
// The Hub persists every inbound message, then dispatches it through
// one of three paths:
//
//  1. Slash commands (`/help`, `/cancel`, ...) → CommandRegistry.
//  2. Non-command text + a known last-notified session → forward to
//     that session's stdin via SessionInputter, and reset the
//     once-mode suppression so the next idle re-notifies.
//  3. Otherwise → publish channel.message_received for any other
//     downstream consumer (no built-in routing today).
func (h *Hub) handleInbound(ctx context.Context, msg ChannelMessage) error {
	msg.Direction = DirectionInbound
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}
	id, err := h.store.InsertMessage(ctx, msg)
	if err != nil {
		h.log.Error("inbound persist failed", "channel", msg.ChannelID, "err", err)
		return err
	}
	if name, args, ok := ParseCommand(msg.Text); ok {
		h.handleCommand(ctx, msg, id, name, args)
		return nil
	}

	// Route plain text to the right session. Priority is:
	//   1) explicit reply-to-message in the chat platform → that
	//      specific notification's session (multi-session friendly)
	//   2) /select <sid> active pin for this channel
	//   3) most-recent notified session on this channel
	//
	// Terminator note: TUIs running in raw mode (Claude Code, Codex,
	// most modern CLIs) treat CR (\r) as Enter (submit) and LF (\n)
	// as shift-Enter (insert newline). Sending \r is what xterm.js
	// itself sends when the user hits Enter, so we mirror that.
	//
	// Why two separate writes (text, brief pause, then \r):
	// Some Ink-based input handlers (notably Gemini CLI) treat a
	// single combined "text+\r" PTY write as a paste-style burst
	// and swallow the trailing \r as part of the paste payload —
	// the text shows up at the prompt but the submit never fires.
	// xterm.js sidesteps this by issuing one PTY write per
	// keystroke; we approximate that by writing the body, briefly
	// yielding, then writing the Enter byte on its own. The pause
	// is below human-perception threshold and Claude/Codex behave
	// identically with or without it.
	if h.input != nil && msg.Text != "" {
		if sid, ok := h.resolveTargetSession(msg); ok {
			if err := h.submitToSession(ctx, sid, msg.Text); err != nil {
				h.log.Warn("forward to session failed",
					"channel", msg.ChannelID, "session", sid, "err", err)
				h.replyTextLookup(ctx, msg, fmt.Sprintf("Could not deliver to %s: %s", sid, err))
				return nil
			}
			h.forgetNotifyForSession(msg.ChannelID, sid)
			h.bus.Publish(eventbus.Event{
				Topic: "channel.message_forwarded",
				Data: map[string]any{
					"channel_id":         msg.ChannelID,
					"channel_message_id": id,
					"session_id":         sid,
					"text":               msg.Text,
				},
			})
			return nil
		}
	}

	h.bus.Publish(eventbus.Event{
		Topic: "channel.message_received",
		Data: map[string]any{
			"channel_id":         msg.ChannelID,
			"channel_message_id": id,
			"conversation_id":    msg.ConversationID,
			"author":             msg.Author,
			"text":               msg.Text,
		},
	})
	return nil
}

// replyTextLookup is a defensive helper: it fetches the channel impl
// from the live map and posts a reply. Returns silently if the
// channel isn't running anymore (Stop racing with inbound).
func (h *Hub) replyTextLookup(ctx context.Context, src ChannelMessage, text string) {
	h.mu.RLock()
	ch := h.channels[src.ChannelID]
	h.mu.RUnlock()
	if ch == nil {
		return
	}
	h.replyText(ctx, ch, src, text)
}

// handleCommand dispatches a parsed command to its registered handler
// and ships the (optional) reply back through the originating channel.
// Unknown commands publish channel.command_unknown and reply with a
// hint pointing to /help.
func (h *Hub) handleCommand(ctx context.Context, msg ChannelMessage, mid int64, name string, args []string) {
	h.mu.RLock()
	ch := h.channels[msg.ChannelID]
	h.mu.RUnlock()

	cmd, ok := h.cmds.Lookup(name)
	if !ok {
		h.bus.Publish(eventbus.Event{
			Topic: "channel.command_unknown",
			Data: map[string]any{
				"channel_id":         msg.ChannelID,
				"channel_message_id": mid,
				"command":            name,
				"args":               args,
			},
		})
		if ch != nil {
			h.replyText(ctx, ch, msg, fmt.Sprintf("Unknown command /%s — try /help", name))
		}
		return
	}
	h.bus.Publish(eventbus.Event{
		Topic: "channel.command_received",
		Data: map[string]any{
			"channel_id":         msg.ChannelID,
			"channel_message_id": mid,
			"command":            name,
			"args":               args,
			"source":             cmd.Source,
		},
	})
	if ch == nil {
		return
	}
	cc := CommandContext{
		Channel: ch, Message: msg, Hub: h,
		Command: name, Args: args, Raw: msg.Text,
	}
	// CardHandler wins when both are set — structured reply
	// (buttons) is strictly more capable than plain text, and the
	// CardSender adapters degrade to Card.RenderText() on channels
	// that don't render buttons. We log the misconfig so future
	// contributors notice they accidentally provided both.
	if cmd.CardHandler != nil {
		if cmd.Handler != nil {
			h.log.Warn("command has both Handler and CardHandler; using CardHandler",
				"command", name)
		}
		card, err := cmd.CardHandler(ctx, cc)
		if err != nil {
			h.log.Error("card command handler failed", "command", name, "err", err)
			h.replyText(ctx, ch, msg,
				fmt.Sprintf("Error running /%s: %s", name, err))
			return
		}
		if card == nil {
			return
		}
		out := ChannelMessage{
			ChannelID: msg.ChannelID,
			Direction: DirectionOutbound,
			Text:      card.RenderText(),
			Timestamp: time.Now().UTC(),
			ReplyCtx:  msg.ReplyCtx,
		}
		if err := h.sendWithFallback(ctx, ch, out, card); err != nil {
			h.log.Error("send card reply", "command", name, "err", err)
		}
		return
	}
	if cmd.Handler == nil {
		h.log.Warn("command has no handler", "command", name)
		return
	}
	reply, err := cmd.Handler(ctx, cc)
	if err != nil {
		h.log.Error("command handler failed", "command", name, "err", err)
		h.replyText(ctx, ch, msg, fmt.Sprintf("Error running /%s: %s", name, err))
		return
	}
	if reply != "" {
		h.replyText(ctx, ch, msg, reply)
	}
}

// replyText posts a text reply back through the originating channel,
// preserving ReplyCtx so it threads correctly. Persisted to
// channel_messages and treated like any other outbound.
func (h *Hub) replyText(ctx context.Context, ch Channel, src ChannelMessage, text string) {
	out := ChannelMessage{
		ChannelID:      src.ChannelID,
		Direction:      DirectionOutbound,
		ConversationID: src.ConversationID,
		Text:           text,
		Timestamp:      time.Now().UTC(),
		ReplyCtx:       src.ReplyCtx,
	}
	if err := ch.Send(ctx, out); err != nil {
		h.log.Error("command reply send failed", "channel", ch.ID(), "err", err)
		return
	}
	if h.store == nil {
		return
	}
	if _, err := h.store.InsertMessage(ctx, out); err != nil {
		h.log.Warn("command reply persist failed", "err", err)
	}
}

// setMuted patches the channel's config JSON to set or clear the
// muted flag. dispatch() honours this flag by skipping muted channels.
func (h *Hub) setMuted(ctx context.Context, channelID string, muted bool) error {
	row, err := h.store.Get(ctx, channelID)
	if err != nil {
		return err
	}
	var cfg map[string]any
	if len(row.Config) > 0 {
		_ = json.Unmarshal(row.Config, &cfg)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	cfg["muted"] = muted
	patched, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal channel config: %w", err)
	}
	return h.store.Update(ctx, channelID, patched, nil)
}

// isMuted returns true when the channel's config has muted=true.
func (h *Hub) isMuted(ctx context.Context, channelID string) bool {
	row, err := h.store.Get(ctx, channelID)
	if err != nil {
		return false
	}
	var cfg struct {
		Muted bool `json:"muted"`
	}
	_ = json.Unmarshal(row.Config, &cfg)
	return cfg.Muted
}

// NotifyMode is the per-channel suppression policy for repeat
// session.* notifications.
//
//	once     — fire once per (channel, topic, session) tuple. Stays
//	           suppressed until either the channel is updated, the
//	           session ends, or user input arrives back via the same
//	           channel (which resets the entry).
//	cooldown — time-window suppression keyed off notify_cooldown_s.
//	every    — never suppress; every event fires a notification.
//
// Default is `once`: matches user expectations ("notify me once when
// the session needs me, not every minute").
type NotifyMode string

const (
	NotifyModeOnce     NotifyMode = "once"
	NotifyModeCooldown NotifyMode = "cooldown"
	NotifyModeEvery    NotifyMode = "every"
)

// onceModeTTL bounds how long a `once`-mode suppression record sits
// in memory before GC reclaims it. Channels left running for weeks
// shouldn't leak memory. After TTL, a fresh idle event would notify
// again — usually fine, since by then the user has either responded
// or stopped caring.
const onceModeTTL = 24 * time.Hour

// notifyPolicy returns the resolved (mode, cooldown) for the channel.
// When the channel config does not set notify_mode, the legacy field
// notify_cooldown_s decides: > 0 → cooldown mode, anything else →
// once mode. This keeps existing channels saved before notify_mode
// existed working sanely (no spam).
func (h *Hub) notifyPolicy(ctx context.Context, channelID string) (NotifyMode, time.Duration) {
	row, err := h.store.Get(ctx, channelID)
	if err != nil {
		return NotifyModeOnce, 0
	}
	var cfg struct {
		Mode    string `json:"notify_mode"`
		Seconds *int   `json:"notify_cooldown_s"`
	}
	_ = json.Unmarshal(row.Config, &cfg)
	cooldown := time.Duration(0)
	if cfg.Seconds != nil && *cfg.Seconds > 0 {
		cooldown = time.Duration(*cfg.Seconds) * time.Second
	}
	switch NotifyMode(cfg.Mode) {
	case NotifyModeOnce:
		return NotifyModeOnce, 0
	case NotifyModeCooldown:
		if cooldown <= 0 {
			cooldown = defaultCooldown
		}
		return NotifyModeCooldown, cooldown
	case NotifyModeEvery:
		return NotifyModeEvery, 0
	}
	// Mode missing — infer from legacy cooldown field. Pre-feature
	// channels (no notify_cooldown_s either) get `once`.
	if cooldown > 0 {
		return NotifyModeCooldown, cooldown
	}
	return NotifyModeOnce, 0
}

// suppressByPolicy reports whether the (channel, topic, session)
// notification should be skipped under the channel's notify mode.
//   - every    → never
//   - cooldown → suppressed if last fire was within `cd` ago
//   - once     → suppressed forever after the first fire (until
//     forgetNotifyState* / TTL elapses)
//
// The state map is opportunistically GC'd: entries older than the
// effective TTL (cooldown→2×cd, once→onceModeTTL) are dropped on
// every call, so memory stays bounded under session churn.
func (h *Hub) suppressByPolicy(ctx context.Context, channelID, topic, sessionID string) bool {
	mode, cd := h.notifyPolicy(ctx, channelID)
	if mode == NotifyModeEvery {
		return false
	}
	now := time.Now()
	key := topic + "|" + sessionID

	h.notifyMu.Lock()
	defer h.notifyMu.Unlock()
	chState := h.notifyState[channelID]
	if chState == nil {
		chState = make(map[string]time.Time)
		h.notifyState[channelID] = chState
	}
	if last, ok := chState[key]; ok {
		switch mode {
		case NotifyModeOnce:
			// Suppressed forever, until forgetNotifyForSession is
			// called (user input arrives) or the entry ages out.
			if now.Sub(last) < onceModeTTL {
				return true
			}
		case NotifyModeCooldown:
			if cd > 0 && now.Sub(last) < cd {
				return true
			}
		}
	}
	chState[key] = now

	// GC. Use the larger of the two windows so we don't lose state
	// the same call just wrote.
	ttl := onceModeTTL
	if mode == NotifyModeCooldown {
		ttl = 2 * cd
	}
	if ttl > 0 {
		cutoff := now.Add(-ttl)
		for k, t := range chState {
			if t.Before(cutoff) {
				delete(chState, k)
			}
		}
	}
	return false
}

// forgetNotifyState clears the per-channel suppression record. Called
// on channel update / delete so config changes (e.g. switching modes)
// take effect immediately and deleted channels don't leak memory.
func (h *Hub) forgetNotifyState(channelID string) {
	h.notifyMu.Lock()
	defer h.notifyMu.Unlock()
	delete(h.notifyState, channelID)
}

// forgetNotifyForSession clears the suppression entry for one
// (channel, topic, session) triple. Called when the user replies
// through the channel — the next time that session goes idle we
// want to ping them again, not stay quiet forever. Topic prefix
// "session." is matched on so a single user reply clears `idle`,
// `started` and `ended` entries together.
func (h *Hub) forgetNotifyForSession(channelID, sessionID string) {
	h.notifyMu.Lock()
	defer h.notifyMu.Unlock()
	state, ok := h.notifyState[channelID]
	if !ok {
		return
	}
	suffix := "|" + sessionID
	for k := range state {
		if strings.HasSuffix(k, suffix) {
			delete(state, k)
		}
	}
}

// sessionIDFromEvent pulls the session_id field out of an event's
// data payload (best effort). Empty string when the topic doesn't
// carry one.
func sessionIDFromEvent(ev eventbus.Event) string {
	data, _ := ev.Data.(map[string]any)
	if data == nil {
		return ""
	}
	if s, ok := data["session_id"].(string); ok {
		return s
	}
	return ""
}

func (h *Hub) runOutbound(ctx context.Context) {
	defer close(h.outDone)
	chIdle, unsubI := h.bus.Subscribe("session.idle", 64)
	defer unsubI()
	chEnded, unsubE := h.bus.Subscribe("session.ended", 64)
	defer unsubE()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-chIdle:
			if !ok {
				return
			}
			h.dispatch(ctx, ev)
		case ev, ok := <-chEnded:
			if !ok {
				return
			}
			h.dispatch(ctx, ev)
		}
	}
}

func (h *Hub) dispatch(ctx context.Context, ev eventbus.Event) {
	h.mu.RLock()
	chs := make([]Channel, 0, len(h.channels))
	for _, c := range h.channels {
		chs = append(chs, c)
	}
	h.mu.RUnlock()

	sessionID := sessionIDFromEvent(ev)

	for _, c := range chs {
		if h.isMuted(ctx, c.ID()) {
			continue
		}
		topics, err := h.notifyTopicsFor(ctx, c.ID())
		if err != nil {
			continue
		}
		if len(topics) > 0 && !contains(topics, ev.Topic) {
			continue
		}
		if h.suppressByPolicy(ctx, c.ID(), ev.Topic, sessionID) {
			continue
		}

		// Render the card per-channel — different channels can have
		// different snippet preferences (omit snippet, smaller cap…).
		snip := h.snippetPrefs(ctx, c.ID())
		card := buildSessionCard(ev, snip)
		textFallback := card.RenderText()

		msg := ChannelMessage{
			ChannelID:      c.ID(),
			Direction:      DirectionOutbound,
			ConversationID: "default",
			Text:           textFallback,
			Timestamp:      time.Now().UTC(),
			// Initialise so channel impls (Telegram etc.) can drop
			// "outbound_msg_id" here for reply-to indexing.
			Metadata: map[string]any{},
		}
		if err := h.sendWithFallback(ctx, c, msg, card); err != nil {
			h.log.Error("channel send failed", "id", c.ID(), "err", err)
			continue
		}
		if _, err := h.store.InsertMessage(ctx, msg); err != nil {
			h.log.Warn("outbound persist failed", "id", c.ID(), "err", err)
		}
		// Record routing hints so a future Telegram-style reply lands
		// on the right session.
		if sessionID != "" {
			h.lastSessMu.Lock()
			h.lastSess[c.ID()] = sessionID
			h.lastSessMu.Unlock()
			h.recordOutbound(c.ID(), msg.Metadata, sessionID)
		}
		h.bus.Publish(eventbus.Event{
			Topic: "channel.message_sent",
			Data: map[string]any{
				"channel_id": c.ID(),
				"topic":      ev.Topic,
			},
		})
	}
}

func (h *Hub) lookupLastSession(channelID string) string {
	h.lastSessMu.RLock()
	defer h.lastSessMu.RUnlock()
	return h.lastSess[channelID]
}

// recordOutbound stashes (channelID, outboundMsgID) -> sessionID so
// the next inbound that's a reply to outboundMsgID routes to that
// specific session. Caller passes channel-supplied msg.Metadata after
// a successful send; we look for "outbound_msg_id" by convention.
func (h *Hub) recordOutbound(channelID string, meta map[string]any, sessionID string) {
	if sessionID == "" || meta == nil {
		return
	}
	mid, _ := meta["outbound_msg_id"].(string)
	if mid == "" {
		return
	}
	now := time.Now()
	h.outboundMu.Lock()
	defer h.outboundMu.Unlock()
	chMap := h.outboundIndex[channelID]
	if chMap == nil {
		chMap = make(map[string]outboundEntry)
		h.outboundIndex[channelID] = chMap
	}
	chMap[mid] = outboundEntry{sessionID: sessionID, ts: now}
	// Cheap LRU: when over the cap, drop the oldest 25% in one pass.
	if len(chMap) > outboundIndexMax {
		evictOldest(chMap, len(chMap)/4)
	}
}

func evictOldest(m map[string]outboundEntry, n int) {
	if n <= 0 || len(m) == 0 {
		return
	}
	type kv struct {
		k string
		t time.Time
	}
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k, v.ts})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].t.Before(all[j].t) })
	for i := 0; i < n && i < len(all); i++ {
		delete(m, all[i].k)
	}
}

// lookupOutbound returns the session_id associated with a previous
// outbound notification on this channel.
func (h *Hub) lookupOutbound(channelID, outboundMsgID string) string {
	h.outboundMu.Lock()
	defer h.outboundMu.Unlock()
	if chMap, ok := h.outboundIndex[channelID]; ok {
		return chMap[outboundMsgID].sessionID
	}
	return ""
}

func (h *Hub) lookupActiveSession(channelID string) string {
	h.activeSessMu.RLock()
	defer h.activeSessMu.RUnlock()
	return h.activeSess[channelID]
}

func (h *Hub) setActiveSession(channelID, sessionID string) {
	h.activeSessMu.Lock()
	defer h.activeSessMu.Unlock()
	if sessionID == "" {
		delete(h.activeSess, channelID)
		return
	}
	h.activeSess[channelID] = sessionID
}

// submitToSession types `text` into a session's PTY rune-by-rune,
// mimicking what xterm.js does for keyboard input — each rune is
// its own write, with a brief inter-key pause — then sends a
// final \r on its own as the Enter keypress.
//
// Why not a single text+\r write: Gemini's Ink-based input handler
// classifies any multi-byte PTY write as a paste burst. In paste
// mode the trailing Enter is swallowed as part of the paste
// payload (multi-line paste with embedded newline) instead of
// firing the submit handler. xterm.js sidesteps this by emitting
// one PTY write per real keystroke; we mirror that exactly so the
// CLI cannot tell our input apart from a human typing.
//
// Cost: ~5 ms per rune means a 20-rune chat message takes ~100 ms
// to "type" — at or below the human-perception threshold for
// chat interactions, and trivial compared to the round-trip to a
// remote Gemini API. Long pastes (a multi-KB blob) would degrade,
// but operators paste those into the web admin, not Telegram.
//
// Cancellable via ctx — partial sends are surfaced as errors so
// the caller can report failure back to the chat. Submitting an
// empty body is a no-op for the typing loop but still emits the
// final Enter (useful for chat platforms that surface tap-Enter
// gestures as empty submissions).
func (h *Hub) submitToSession(ctx context.Context, sid, text string) error {
	if h.input == nil {
		return errors.New("session input not configured")
	}
	for _, r := range text {
		if err := h.input.Input(ctx, sid, []byte(string(r))); err != nil {
			return err
		}
		select {
		case <-time.After(perRuneDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	// A slightly larger settle window between the last keystroke
	// and the Enter byte, mirroring the natural human pause before
	// pressing Return. Empirically this is what makes Gemini fire
	// the submit handler instead of treating Enter as part of the
	// paste payload.
	select {
	case <-time.After(submitDelay):
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := h.input.Input(ctx, sid, []byte{'\r'}); err != nil {
		return fmt.Errorf("submit: %w", err)
	}
	return nil
}

// resolveTargetSession picks the session for an inbound non-command
// message. Priority:
//
//  1. reply-to-message — the user replied to a specific notification
//  2. /select override — explicit session pin for this channel
//  3. last-notified session — the simple "most recent" fallback
//
// Returns ("", false) when no target can be determined.
func (h *Hub) resolveTargetSession(msg ChannelMessage) (string, bool) {
	if rid, _ := msg.Metadata["reply_to_outbound_msg_id"].(string); rid != "" {
		if sid := h.lookupOutbound(msg.ChannelID, rid); sid != "" {
			return sid, true
		}
	}
	if sid := h.lookupActiveSession(msg.ChannelID); sid != "" {
		return sid, true
	}
	if sid := h.lookupLastSession(msg.ChannelID); sid != "" {
		return sid, true
	}
	return "", false
}

// sendWithFallback ships a card via CardSender when supported, else
// falls back to Channel.Send with the rendered text. Centralised so
// every outbound code path observes the same degradation rule.
//
// CardSender impls that return ErrNotSupported (e.g. a bridge whose
// adapter did not claim the "card" capability) trigger the same
// fallback as channels that don't implement CardSender at all.
func (h *Hub) sendWithFallback(ctx context.Context, c Channel, msg ChannelMessage, card *Card) error {
	if cs, ok := c.(CardSender); ok && card != nil {
		err := cs.SendCard(ctx, msg, card)
		if err == nil || !errors.Is(err, ErrNotSupported) {
			return err
		}
	}
	return c.Send(ctx, msg)
}

func (h *Hub) notifyTopicsFor(ctx context.Context, channelID string) ([]string, error) {
	row, err := h.store.Get(ctx, channelID)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		NotifyOn []string `json:"notify_on"`
	}
	_ = json.Unmarshal(row.Config, &cfg)
	return cfg.NotifyOn, nil
}

// CreateChannel registers a new channel and starts it if enabled.
func (h *Hub) CreateChannel(ctx context.Context, kind string, config json.RawMessage, enabled bool) (string, error) {
	if Lookup(kind) == nil {
		return "", fmt.Errorf("%w: %s", ErrUnknownKind, kind)
	}
	id := newID()
	if err := h.store.Insert(ctx, id, kind, config, enabled); err != nil {
		return "", err
	}
	if enabled && h.isStarted() {
		if err := h.spawn(ctx, channelRow{ID: id, Kind: kind, Config: config, Enabled: true}); err != nil {
			return "", err
		}
	}
	return id, nil
}

// UpdateChannel persists changes and restarts the impl when running.
// Pass nil for any unchanged field.
func (h *Hub) UpdateChannel(ctx context.Context, id string, config json.RawMessage, enabled *bool) error {
	if err := h.store.Update(ctx, id, config, enabled); err != nil {
		return err
	}
	row, err := h.store.Get(ctx, id)
	if err != nil {
		return err
	}
	h.mu.Lock()
	existing, running := h.channels[id]
	delete(h.channels, id)
	h.mu.Unlock()
	if running {
		_ = existing.Stop(ctx)
	}
	// Cooldown bookkeeping is keyed by channelID — drop it on update so
	// a freshly-lowered cooldown isn't blocked by an old timestamp.
	h.forgetNotifyState(id)
	if row.Enabled && h.isStarted() {
		return h.spawn(ctx, row)
	}
	return nil
}

// DeleteChannel stops the running impl (if any) and removes the row.
func (h *Hub) DeleteChannel(ctx context.Context, id string) error {
	h.mu.Lock()
	ch, ok := h.channels[id]
	delete(h.channels, id)
	h.mu.Unlock()
	if ok {
		_ = ch.Stop(ctx)
	}
	h.forgetNotifyState(id)
	return h.store.Delete(ctx, id)
}

// LookupWebhook returns the WebhookHandler-implementing channel with
// the given id, or (nil, false) when the channel either does not exist
// or does not accept webhooks. Used by the public webhook route to
// route inbound HTTP POSTs to the right impl.
func (h *Hub) LookupWebhook(id string) (WebhookHandler, bool) {
	h.mu.RLock()
	c, ok := h.channels[id]
	h.mu.RUnlock()
	if !ok {
		return nil, false
	}
	wh, ok := c.(WebhookHandler)
	return wh, ok
}

// SendTest pushes a fixed text message via channel.Send.
func (h *Hub) SendTest(ctx context.Context, id string) error {
	h.mu.RLock()
	ch, ok := h.channels[id]
	h.mu.RUnlock()
	if !ok {
		return ErrNotFound
	}
	return ch.Send(ctx, ChannelMessage{
		ChannelID:      id,
		Direction:      DirectionOutbound,
		ConversationID: "default",
		Text:           "OpenDray channel test ✓",
		Timestamp:      time.Now().UTC(),
	})
}

// List returns the persisted channels along with a "running" flag.
func (h *Hub) List(ctx context.Context) ([]ChannelView, error) {
	rows, err := h.store.List(ctx)
	if err != nil {
		return nil, err
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]ChannelView, 0, len(rows))
	for _, r := range rows {
		ch, running := h.channels[r.ID]
		out = append(out, viewOf(r, ch, running))
	}
	return out, nil
}

// Get returns one channel view.
func (h *Hub) Get(ctx context.Context, id string) (ChannelView, error) {
	r, err := h.store.Get(ctx, id)
	if err != nil {
		return ChannelView{}, err
	}
	h.mu.RLock()
	ch, running := h.channels[id]
	h.mu.RUnlock()
	return viewOf(r, ch, running), nil
}

// viewOf renders the public REST shape for one channel row, including
// the capability list and muted flag (both read live from the running
// impl + config JSON respectively). Channels that have not been
// instantiated yet report only the text capability.
func viewOf(r channelRow, ch Channel, running bool) ChannelView {
	caps := []Capability{CapText}
	if ch != nil {
		caps = Capabilities(ch)
	}
	muted := false
	if len(r.Config) > 0 {
		var cfg struct {
			Muted bool `json:"muted"`
		}
		_ = json.Unmarshal(r.Config, &cfg)
		muted = cfg.Muted
	}
	return ChannelView{
		ID:           r.ID,
		Kind:         r.Kind,
		Config:       r.Config,
		Enabled:      r.Enabled,
		Running:      running,
		Capabilities: caps,
		Muted:        muted,
	}
}

// ChannelView is the public wire shape for REST.
type ChannelView struct {
	ID           string          `json:"id"`
	Kind         string          `json:"kind"`
	Config       json.RawMessage `json:"config"`
	Enabled      bool            `json:"enabled"`
	Running      bool            `json:"running"`
	Capabilities []Capability    `json:"capabilities"`
	Muted        bool            `json:"muted"`
}

func (h *Hub) isStarted() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.started
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// snippetPrefs is the per-channel preference for embedding the
// terminal tail in session.* notifications.
//
// MaxChars semantics: 0 (the default) means "no cap — channel impl
// handles platform chunking". A positive value applies a hard cap
// with the "[…]" elided-prefix marker.
type snippetPrefs struct {
	Include  bool
	MaxChars int
}

// snippetPrefsFor reads notify_include_snippet (default true) and
// notify_snippet_max_chars (default 0 = no cap → multi-message
// chunking takes over) from the channel's config JSON.
func (h *Hub) snippetPrefs(ctx context.Context, channelID string) snippetPrefs {
	out := snippetPrefs{Include: true, MaxChars: 0}
	row, err := h.store.Get(ctx, channelID)
	if err != nil {
		return out
	}
	var cfg struct {
		Include  *bool `json:"notify_include_snippet"`
		MaxChars *int  `json:"notify_snippet_max_chars"`
	}
	_ = json.Unmarshal(row.Config, &cfg)
	if cfg.Include != nil {
		out.Include = *cfg.Include
	}
	if cfg.MaxChars != nil && *cfg.MaxChars >= 0 {
		out.MaxChars = *cfg.MaxChars
	}
	return out
}

// buildSessionCard turns a session.* event into a structured Card.
// Cards always carry a body and (for known events) a row of inline
// action buttons. Channels without CardSender get the RenderText()
// fallback.
//
// session.idle bodies are rendered as plain text (no code-fence
// wrapper, no parse_mode-specific markdown). The reasoning:
//
//   - Wrapping a long Claude reply in ``` makes Telegram show it as
//     monospace, which is fine for shell output but wrong for
//     prose responses (the actual JSONL content).
//   - Telegram legacy parse_mode=Markdown breaks on Claude's
//     `**bold**` (GitHub-flavored) and can return 400 mid-message,
//     killing the rest of the chunked stream.
//
// Long content is left intact when MaxChars==0 (the new default) —
// channel impls (Telegram, etc.) chunk into multiple messages
// automatically. Operators who want a hard cap can still set
// notify_snippet_max_chars from the UI.
func buildSessionCard(ev eventbus.Event, snip snippetPrefs) *Card {
	data, _ := ev.Data.(map[string]any)
	sid, _ := data["session_id"].(string)
	switch ev.Topic {
	case "session.idle":
		ms := toInt64(data["idle_for_ms"])
		body := fmt.Sprintf("Session %s went idle (silent for %ds).", sid, ms/1000)
		if snip.Include {
			recent, _ := data["recent_output"].(string)
			if snip.MaxChars > 0 {
				recent = trimForCardN(recent, snip.MaxChars)
			}
			if recent = strings.TrimSpace(recent); recent != "" {
				body += "\n\n" + recent
			}
		}
		return &Card{
			Header: &CardHeader{Title: "Session idle", Color: "yellow"},
			Elements: []CardElement{
				CardMarkdown{Content: body},
				// Buttons emit slash-command payloads that the app
				// wires via session.RegisterChannelCommands. Resume
				// re-spawns a stopped/idle session; End stops a
				// running one; Mute silences the channel.
				CardActions{Buttons: [][]ButtonOption{{
					{Text: "Resume", Value: "cmd:/resume " + sid, Style: "primary"},
					{Text: "End", Value: "cmd:/end " + sid, Style: "danger"},
					{Text: "Mute", Value: "cmd:/notify off"},
				}}},
			},
		}
	case "session.ended":
		exit := toInt64(data["exit_code"])
		color := "green"
		if exit != 0 {
			color = "red"
		}
		return &Card{
			Header: &CardHeader{Title: "Session ended", Color: color},
			Elements: []CardElement{
				CardMarkdown{Content: fmt.Sprintf("Session `%s` ended with exit_code=%d.", sid, exit)},
				// Resume re-spawns the ended session under the same
				// id. The legacy "Spawn similar" button was dropped
				// because its /spawn-like handler was never wired.
				CardActions{Buttons: [][]ButtonOption{{
					{Text: "Resume", Value: "cmd:/resume " + sid, Style: "primary"},
					{Text: "Open log", Value: "nav:/sessions/" + sid},
				}}},
			},
		}
	}
	return &Card{
		Elements: []CardElement{
			CardMarkdown{Content: fmt.Sprintf("%s: %s", ev.Topic, sid)},
		},
	}
}

// trimForCardN hard-caps the recent_output snippet so the card stays
// within the per-channel message limit. When trimming, keeps the
// *trailing* portion (most recent / most relevant) and prepends a
// "[…]" marker so the reader knows content was elided.
func trimForCardN(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	tail := s[len(s)-max:]
	// Avoid splitting in the middle of a UTF-8 rune.
	for i := 0; i < 4 && len(tail) > 0; i++ {
		if tail[0]&0xC0 == 0x80 {
			tail = tail[1:]
			continue
		}
		break
	}
	return "[…]\n" + tail
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}

func newID() string {
	var b [9]byte
	_, _ = rand.Read(b[:])
	return "ch_" + base64.RawURLEncoding.EncodeToString(b[:])
}
