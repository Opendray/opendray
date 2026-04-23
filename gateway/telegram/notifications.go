package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray/gateway/telegram/jsonl"
	"github.com/opendray/opendray/kernel/hub"
	"github.com/opendray/opendray/plugin"
)

// Minimum gap between two idle notifications for the same session.
// 10 s is short enough to catch each Claude response cycle, long enough
// to avoid duplicate pings from sub-second idle re-fires.
const idleCooldown = 10 * time.Second

// Notifier subscribes to the hook bus and pushes formatted alerts to the
// configured Telegram chat. On idle events it ALSO calls
// forwarder.SendIfChanged() to push new content to linked chats — this
// is the only path output reaches Telegram (no polling, no goroutines).
type Notifier struct {
	bot         *Bot
	hub         *hub.Hub
	bus         *plugin.HookBus
	links       *LinkStore
	forwarder   *Forwarder
	unsubscribe func()

	mu           sync.Mutex
	lastNotify   map[string]time.Time
	pendingInput map[string]time.Time // sessionID → when input was sent; non-zero = waiting for real output
}

func NewNotifier(bot *Bot, h *hub.Hub, bus *plugin.HookBus, links *LinkStore, fwd *Forwarder) *Notifier {
	return &Notifier{
		bot: bot, hub: h, bus: bus, links: links, forwarder: fwd,
		lastNotify:   map[string]time.Time{},
		pendingInput: map[string]time.Time{},
	}
}

// Start subscribes to idle / stop hooks. Returns nil even if no chat is
// configured — the bot will still be queryable via /help.
func (n *Notifier) Start() {
	if n.unsubscribe != nil {
		return
	}
	hooks := []string{}
	if n.bot.Cfg().NotifyOnIdle {
		hooks = append(hooks, plugin.HookOnIdle)
	}
	if n.bot.Cfg().NotifyOnExit {
		hooks = append(hooks, plugin.HookOnSessionStop)
	}
	if len(hooks) == 0 {
		return
	}
	n.unsubscribe = n.bus.SubscribeLocal(hooks, n.handle)
}

// Stop cleanly unsubscribes.
func (n *Notifier) Stop() {
	if n.unsubscribe != nil {
		n.unsubscribe()
		n.unsubscribe = nil
	}
}

// ResetIdleFlag clears the cooldown for a session so the NEXT idle event
// fires a fresh notification immediately, and marks the session as
// "pending input" so the notifier knows to wait for real output: if an
// idle event fires during the "thinking gap" (before the session produces
// its actual response), the notification won't be consumed prematurely.
//
// Called by the dispatcher whenever user input is forwarded to a session
// from Telegram. Input from other channels (CLI typing, IDE terminal) is
// handled by the normal idle → handle() → cooldown flow; a dedicated
// flag-reset is not required.
func (n *Notifier) ResetIdleFlag(sessionID string) {
	n.mu.Lock()
	delete(n.lastNotify, sessionID) // clear cooldown so the next idle fires immediately
	n.pendingInput[sessionID] = time.Now()
	n.mu.Unlock()
	// A new turn begins: discard the forwarder's diff baseline so the next
	// SendIfChanged pushes the full reply instead of diffing it against the
	// previous turn's text (which can filter the whole message below the
	// 5-rune threshold when replies share boilerplate lines).
	if n.forwarder != nil {
		n.forwarder.ResetSnapshot(sessionID)
	}
}

func (n *Notifier) handle(ev plugin.HookEvent) {
	chatID := n.bot.NotifyChatID()
	if chatID == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch ev.Type {
	case plugin.HookOnIdle:
		n.mu.Lock()
		// Cooldown guard. watchIdle already gates re-fires per idle-period
		// (Phase 2 waits for State → Active before re-arming), so this only
		// catches poll-loop jitter around a single idle transition.
		if last, ok := n.lastNotify[ev.SessionID]; ok && time.Since(last) < idleCooldown {
			n.mu.Unlock()
			return
		}

		// Check pending input: if we sent input and are waiting for real
		// output, note it now. If the forwarder finds nothing new, we'll
		// keep the notification open for the next idle cycle.
		pendingSince, isPending := n.pendingInput[ev.SessionID]
		// Expire pending state after 2 minutes to avoid infinite retries.
		if isPending && time.Since(pendingSince) > 2*time.Minute {
			isPending = false
			delete(n.pendingInput, ev.SessionID)
		}
		_, hadPriorNotify := n.lastNotify[ev.SessionID]
		n.mu.Unlock()

		// 1. Push new content to every LINKED chat (diff-based).
		sent := 0
		if n.forwarder != nil {
			sent = n.forwarder.SendIfChanged(ctx, ev.SessionID)
		}

		// If we're waiting for output after user input but the forwarder
		// found nothing new, this idle was a "thinking gap" — the session
		// went quiet before producing its real response. Don't set the
		// cooldown so the next idle cycle retries immediately.
		if isPending && sent == 0 {
			return
		}

		// Suppress duplicate 🟡 bubbles. When a TUI (Claude Code, Codex,
		// …) is waiting for input it still paints cursor-blinks / spinner
		// frames, which flip state Active → Waiting every idleThreshold
		// and re-arm the hook. Without this guard the user would receive
		// the same "session idle" prompt every 10-20 s. Once the session
		// actually produces new output, sent > 0 and we fall through.
		if hadPriorNotify && sent == 0 {
			n.mu.Lock()
			n.lastNotify[ev.SessionID] = time.Now()
			n.mu.Unlock()
			return
		}

		// Real content was forwarded (or first notification). Commit the
		// cooldown + clear pending.
		n.mu.Lock()
		n.lastNotify[ev.SessionID] = time.Now()
		if isPending {
			delete(n.pendingInput, ev.SessionID)
		}
		n.mu.Unlock()

		// 2. Send 🟡 notification to the dedicated notification chat —
		//    but SKIP if the forwarder actually delivered content to this
		//    chat (sending again = duplicate). The sent > 0 guard is
		//    essential: when the diff is too small (<5 runes) the forwarder
		//    returns 0, and without this check the notification was silently
		//    swallowed for linked chats.
		alreadyCovered := false
		if sent > 0 && n.forwarder != nil {
			for _, c := range n.forwarder.store.ChatsFor(ev.SessionID) {
				if c == chatID {
					alreadyCovered = true
					break
				}
			}
		}
		if !alreadyCovered {
			n.notifyIdle(ctx, chatID, ev.SessionID)
		}
	case plugin.HookOnSessionStop:
		n.mu.Lock()
		delete(n.lastNotify, ev.SessionID)
		delete(n.pendingInput, ev.SessionID)
		n.mu.Unlock()
		n.notifyStop(ctx, chatID, ev.SessionID)
	}
}

func (n *Notifier) notifyIdle(ctx context.Context, chatID int64, sessionID string) {
	sess, ok, err := n.hub.Get(ctx, sessionID)
	if err != nil || !ok {
		return
	}
	name := sess.Name
	if name == "" {
		name = sess.SessionType
	}

	// Try JSONL first (clean structured output for Claude sessions),
	// fall back to buffer snapshot for terminal/other sessions.
	tail := ""
	var prompt jsonl.PromptInfo
	if sess.SessionType == "claude" {
		bases := claudeBasePaths(ctx, n.hub.DB(), sess, n.bot.Cfg().ExtraClaudeDirs)
		if jp := jsonl.ResolveLatestJSONL(bases, sess.CWD); jp != "" {
			if text, p, err := jsonl.LastResponseDetail(jp); err == nil && strings.TrimSpace(text) != "" {
				tail = strings.TrimSpace(text)
				prompt = p
			}
		}
	}
	if tail == "" {
		if ts, running := n.hub.GetTerminalSession(sessionID); running {
			snap := ts.Buffer().Snapshot()
			raw := tailLines(string(snap), n.bot.Cfg().TailLines)
			tail = strings.TrimSpace(CleanForTelegram(raw))
			// PTY path: use text-based detection
			prompt = DetectPrompt(tail)
		}
	}

	// For free-text prompts, append a hint so the user knows to type a reply.
	if prompt.Kind == jsonl.PromptAskUser || prompt.Kind == jsonl.PromptGenericQuestion {
		tail = tail + "\n\n💬 Reply to this message to answer."
	}
	if prompt.Kind == jsonl.PromptMultiSelect {
		tail = tail + "\n\n☑ Check the options you want, then tap <b>Submit</b>. You can also type a reply directly in the chat (it will be sent as a separate message)."
	}

	body := fmt.Sprintf(
		"🟡 Session <b>%s</b> (%s) is <b>idle</b>\n\n",
		escapeHTML(name), escapeHTML(sess.SessionType),
	)
	if tail != "" {
		body += formatForTelegram(tail) + "\n\n"
	}
	shortID := sess.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	body += fmt.Sprintf("<i>id: %s · reply to this message → input goes to the session</i>",
		shortID)

	// Build the correct inline keyboard based on prompt type.
	keyboard := KeyboardFromPrompt(prompt, sess.ID)
	msgID := n.sendHTMLWithKeyboard(ctx, chatID, body, keyboard)
	// Remember the outbound message id so a Telegram reply gets routed
	// back to this session even when no /link is in force.
	if msgID > 0 && n.links != nil {
		n.links.RememberNotification(msgID, sess.ID)
	}
	// Multi-select: register checkbox state so toggle/submit callbacks
	// can find it.
	if msgID > 0 && prompt.Kind == jsonl.PromptMultiSelect &&
		n.forwarder != nil && n.forwarder.multiSelect != nil {
		n.forwarder.multiSelect.Create(chatID, msgID, sess.ID, prompt.Options)
	}
}

func (n *Notifier) notifyStop(ctx context.Context, chatID int64, sessionID string) {
	sess, ok, err := n.hub.Get(ctx, sessionID)
	if err != nil || !ok {
		return
	}
	name := sess.Name
	if name == "" {
		name = sess.SessionType
	}
	icon := "⚪"
	switch strings.ToLower(sess.Status) {
	case "stopped":
		icon = "🟢"
	case "error":
		icon = "🔴"
	}
	body := fmt.Sprintf("%s Session <b>%s</b> exited (status: <b>%s</b>)",
		icon, escapeHTML(name), escapeHTML(sess.Status))
	n.sendHTML(ctx, chatID, body)
}

// send delivers a notification and returns the resulting message id (or 0
// on failure) so the caller can hook reply-routing.
func (n *Notifier) sendHTML(ctx context.Context, chatID int64, html string) int64 {
	return n.sendHTMLWithKeyboard(ctx, chatID, html, nil)
}

func (n *Notifier) sendHTMLWithKeyboard(ctx context.Context, chatID int64, html string, kb *InlineKeyboardMarkup) int64 {
	opts := &SendOpts{ParseMode: "HTML", DisablePreview: true}
	if kb != nil {
		opts.ReplyMarkup = kb
	}
	id, err := n.bot.Send(ctx, chatID, html, opts)
	if err != nil {
		plain := strings.NewReplacer("<b>", "", "</b>", "", "<i>", "", "</i>", "",
			"<pre>", "", "</pre>", "",
			"&amp;", "&", "&lt;", "<", "&gt;", ">").Replace(html)
		id, _ = n.bot.Send(ctx, chatID, plain, &SendOpts{ReplyMarkup: kb})
	}
	return id
}
