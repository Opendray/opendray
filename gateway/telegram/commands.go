package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linivek/ntc/gateway/telegram/jsonl"
	"github.com/linivek/ntc/kernel/hub"
)

// Dispatcher implements UpdateHandler. It owns no terminal state itself —
// it consults the Hub for session lookups, the LinkStore for chat
// bindings, and the Forwarder for output streaming.
type Dispatcher struct {
	bot         *Bot
	hub         *hub.Hub
	links       *LinkStore
	forwarder   *Forwarder
	notifier    *Notifier
	resolver    *SessionResolver
	multiSelect *MultiSelectStore
}

// NewDispatcher wires a bot to a hub + link / forwarding plumbing.
func NewDispatcher(bot *Bot, h *hub.Hub, links *LinkStore, fwd *Forwarder, notifier *Notifier) *Dispatcher {
	ms := NewMultiSelectStore(24 * time.Hour)
	if fwd != nil {
		fwd.SetMultiSelectStore(ms)
	}
	return &Dispatcher{
		bot: bot, hub: h, links: links, forwarder: fwd, notifier: notifier,
		resolver:    NewSessionResolver(h),
		multiSelect: ms,
	}
}

// MultiSelect exposes the dispatcher's checkbox-state store so the
// Notifier / Forwarder can register a state record immediately after
// sending a PromptMultiSelect message.
func (d *Dispatcher) MultiSelect() *MultiSelectStore { return d.multiSelect }

// HandleUpdate is the only entry point invoked by the bot's poll loop.
func (d *Dispatcher) HandleUpdate(ctx context.Context, u Update) {
	// Callback query = user tapped an inline-keyboard button.
	if u.CallbackQuery != nil {
		d.handleCallback(ctx, u.CallbackQuery)
		return
	}

	if u.Message == nil || u.Message.Text == "" {
		return
	}
	msg := u.Message
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	if !strings.HasPrefix(text, "/") {
		// Plain text. Try the three routing paths in order:
		//   1. user replied to one of *our* notification messages
		//      → route to the session that notification was about
		//   2. chat is /link'd to a session → route to that session
		//   3. neither → hint the user
		if msg.ReplyToMessage != nil {
			if sid := d.links.ResolveReply(msg.ReplyToMessage.MessageID); sid != "" {
				d.forwardToSession(ctx, chatID, msg.MessageID, sid, text)
				return
			}
		}
		if sid := d.links.Get(chatID); sid != "" {
			d.forwardToSession(ctx, chatID, msg.MessageID, sid, text)
			return
		}
		d.reply(ctx, chatID, msg.MessageID,
			"No session linked to this chat.\n"+
				"• Use /link <session_id> to bind the chat, OR\n"+
				"• Reply directly to a notification message and your reply gets routed automatically.\n"+
				"Run /sessions to see ids.")
		return
	}

	cmd, arg := splitCommand(text)
	switch cmd {
	case "/start", "/help":
		d.cmdHelp(ctx, chatID, msg.MessageID)
	case "/status", "/sessions":
		d.cmdStatus(ctx, chatID, msg.MessageID)
	case "/tail":
		d.cmdTail(ctx, chatID, msg.MessageID, arg)
	case "/stop":
		d.cmdStop(ctx, chatID, msg.MessageID, arg)
	case "/whoami":
		d.cmdWhoami(ctx, chatID, msg)
	case "/link":
		d.cmdLink(ctx, chatID, msg.MessageID, arg)
	case "/unlink":
		d.cmdUnlink(ctx, chatID, msg.MessageID)
	case "/links":
		d.cmdLinks(ctx, chatID, msg.MessageID)
	case "/send":
		d.cmdSend(ctx, chatID, msg.MessageID, arg)
	case "/screen":
		d.cmdScreen(ctx, chatID, msg.MessageID, arg)
	// Control-key shortcuts — essential for interacting with TUI CLIs
	case "/cc", "/ctrl+c":
		d.sendControlKey(ctx, chatID, msg.MessageID, "\x03", "Ctrl+C")
	case "/cd", "/ctrl+d":
		d.sendControlKey(ctx, chatID, msg.MessageID, "\x04", "Ctrl+D")
	case "/tab":
		d.sendControlKey(ctx, chatID, msg.MessageID, "\t", "Tab")
	case "/enter":
		d.sendControlKey(ctx, chatID, msg.MessageID, "\r", "Enter")
	case "/yes", "/y":
		d.sendLinkedText(ctx, chatID, msg.MessageID, "yes\r")
	case "/no", "/n":
		d.sendLinkedText(ctx, chatID, msg.MessageID, "no\r")
	default:
		d.reply(ctx, chatID, msg.MessageID,
			"Unknown command: "+cmd+"\nTry /help.")
	}
}

// ── Command implementations ────────────────────────────────────

func (d *Dispatcher) cmdHelp(ctx context.Context, chatID int64, replyTo int64) {
	uname := d.bot.Username()
	if uname == "" {
		uname = "this bot"
	}
	body := strings.Join([]string{
		"*NTC Telegram Bridge*",
		"",
		"You are talking to " + uname + ". Commands:",
		"",
		"*Read*",
		"/sessions           — numbered list of running sessions",
		"/screen [1]         — current screen (linked or by number)",
		"/tail 1 [n]         — last N lines of any session",
		"/whoami             — show your chat id",
		"",
		"*Control*",
		"/link 1             — bind THIS chat to session #1",
		"                     (also accepts name or ID prefix)",
		"/unlink             — remove the binding for this chat",
		"/links              — list all active bindings",
		"/send 1 <text>      — one-shot send without /link",
		"/stop 1             — stop a running session",
		"",
		"*Quick keys* (sent to linked session)",
		"/cc            — Ctrl+C (interrupt)",
		"/cd            — Ctrl+D (EOF)",
		"/tab           — Tab (cycle options)",
		"/enter         — Enter (empty line)",
		"/yes /no       — answer yes/no + Enter",
		"",
		"*Tip*: reply directly to any idle notification —",
		"the reply text goes to that session automatically.",
	}, "\n")
	d.reply(ctx, chatID, replyTo, body)
}

func (d *Dispatcher) cmdStatus(ctx context.Context, chatID int64, replyTo int64) {
	running, err := d.resolver.Refresh(ctx, chatID)
	if err != nil {
		d.reply(ctx, chatID, replyTo, "Error: "+err.Error())
		return
	}
	d.reply(ctx, chatID, replyTo, FormatSessionList(running))
}

func (d *Dispatcher) cmdTail(ctx context.Context, chatID int64, replyTo int64, arg string) {
	rawID, n := parseTailArg(arg)
	if rawID == "" {
		d.reply(ctx, chatID, replyTo, "Usage: /tail <number|name> [lines]\nTip: /sessions lists them.")
		return
	}
	id, err := d.resolver.Resolve(ctx, chatID, rawID)
	if err != nil {
		d.reply(ctx, chatID, replyTo, err.Error())
		return
	}
	text := d.cleanSnapshot(id, n)
	if text == "" {
		d.reply(ctx, chatID, replyTo, "(buffer empty)")
		return
	}
	d.replyHTML(ctx, chatID, replyTo, formatForTelegram(text))
}

// cmdScreen shows the current screen of a session. Accepts an optional
// session argument; falls back to the linked session if none given.
//
//	/screen        — linked session
//	/screen 1      — session #1 (from /sessions)
//	/screen claude — session by name
func (d *Dispatcher) cmdScreen(ctx context.Context, chatID int64, replyTo int64, arg string) {
	arg = strings.TrimSpace(arg)

	var sid string
	if arg != "" {
		resolved, err := d.resolver.Resolve(ctx, chatID, arg)
		if err != nil {
			d.reply(ctx, chatID, replyTo, err.Error())
			return
		}
		sid = resolved
	} else {
		sid = d.links.Get(chatID)
		if sid == "" {
			d.reply(ctx, chatID, replyTo,
				"Usage: /screen [number|name]\n"+
					"Or /link a session first, then plain /screen.\n"+
					"Run /sessions to see the list.")
			return
		}
	}

	// Fetch session metadata for the header.
	sess, ok, err := d.hub.Get(ctx, sid)
	if err != nil || !ok {
		d.reply(ctx, chatID, replyTo, "Session not found.")
		return
	}
	name := sess.Name
	if name == "" {
		name = sess.SessionType
	}
	shortID := sid
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	header := fmt.Sprintf("📺 <b>%s</b> (%s) <code>%s</code>\n",
		escapeHTML(name), escapeHTML(sess.SessionType), shortID)
	const maxChunk = 3500

	// For Claude sessions, prefer JSONL → rich HTML (same quality as push
	// notifications). For terminal/shell sessions, keep PTY buffer + <pre>
	// monospace because terminal layout matters.
	if sess.SessionType == "claude" {
		text := d.cleanSnapshot(sid, 120)
		if text == "" {
			d.reply(ctx, chatID, replyTo, "(screen empty)")
			return
		}
		body := header + formatForTelegram(text)
		if len(body) <= maxChunk {
			d.replyHTML(ctx, chatID, replyTo, body)
			return
		}
		chunks := splitForTelegram(text, maxChunk-200)
		for i, chunk := range chunks {
			var msg string
			if i == 0 {
				msg = header + formatForTelegram(chunk)
			} else {
				msg = formatForTelegram(chunk)
			}
			d.replyHTML(ctx, chatID, replyTo, msg)
		}
		return
	}

	// Non-Claude: PTY buffer + monospace <pre> (preserves terminal layout).
	text := d.screenSnapshot(sid, 120)
	if text == "" {
		d.reply(ctx, chatID, replyTo, "(screen empty)")
		return
	}
	preContent := "<pre>" + escapeHTML(text) + "</pre>"
	firstBody := header + preContent
	if len(firstBody) <= maxChunk {
		d.replyHTML(ctx, chatID, replyTo, firstBody)
		return
	}
	chunks := splitForTelegram(text, maxChunk-200)
	for i, chunk := range chunks {
		var body string
		if i == 0 {
			body = header + "<pre>" + escapeHTML(chunk) + "</pre>"
		} else {
			body = "<pre>" + escapeHTML(chunk) + "</pre>"
		}
		d.replyHTML(ctx, chatID, replyTo, body)
	}
}

// screenSnapshot takes a session's buffer, extracts the last N lines,
// and runs the light cleaning pipeline (ANSI strip → box-drawing strip →
// Claude chrome filter only — NO dedup, NO noise filter). This preserves
// the actual terminal layout for /screen display.
func (d *Dispatcher) screenSnapshot(sessionID string, n int) string {
	ts, ok := d.hub.GetTerminalSession(sessionID)
	if !ok {
		return ""
	}
	if n <= 0 {
		n = 80
	}
	if n > 300 {
		n = 300
	}
	snap := ts.Buffer().Snapshot()
	lines := strings.Split(string(snap), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	text := CleanForScreen(strings.Join(lines, "\n"))
	return strings.TrimSpace(text)
}

// cleanSnapshot tries JSONL first (for Claude sessions), then falls back
// to the buffer snapshot cleaning pipeline.
func (d *Dispatcher) cleanSnapshot(sessionID string, n int) string {
	// Try JSONL (clean structured data, no ANSI/TUI chrome)
	ctx := context.Background()
	sess, ok, err := d.hub.Get(ctx, sessionID)
	if err == nil && ok && sess.SessionType == "claude" {
		bases := claudeBasePaths(ctx, d.hub.DB(), sess, d.bot.Cfg().ExtraClaudeDirs)
		if jp := jsonl.ResolveLatestJSONL(bases, sess.CWD); jp != "" {
			if text, _, err := jsonl.LastResponse(jp); err == nil && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	}

	// Fallback: buffer snapshot
	ts, running := d.hub.GetTerminalSession(sessionID)
	if !running {
		return ""
	}
	if n <= 0 {
		n = 40
	}
	if n > 200 {
		n = 200
	}
	snap := ts.Buffer().Snapshot()
	lines := strings.Split(string(snap), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	text := CleanForTelegram(strings.Join(lines, "\n"))
	return strings.TrimSpace(text)
}

func (d *Dispatcher) cmdStop(ctx context.Context, chatID int64, replyTo int64, arg string) {
	if strings.TrimSpace(arg) == "" {
		d.reply(ctx, chatID, replyTo, "Usage: /stop <number|name>")
		return
	}
	id, err := d.resolver.Resolve(ctx, chatID, arg)
	if err != nil {
		d.reply(ctx, chatID, replyTo, err.Error())
		return
	}
	if err := d.hub.Stop(ctx, id); err != nil {
		d.reply(ctx, chatID, replyTo, "Stop failed: "+err.Error())
		return
	}
	shortID := id
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	d.reply(ctx, chatID, replyTo, "✅ Stopped session `"+shortID+"`.")
}

// cmdLink: /link <session_id> — binds the current chat to a session.
// Re-linking replaces the prior binding (and tears down the prior
// forwarder if no other chat is listening).
func (d *Dispatcher) cmdLink(ctx context.Context, chatID int64, replyTo int64, arg string) {
	if strings.TrimSpace(arg) == "" {
		d.reply(ctx, chatID, replyTo, "Usage: /link <number|name>\nRun /sessions first.")
		return
	}
	id, err := d.resolver.Resolve(ctx, chatID, arg)
	if err != nil {
		d.reply(ctx, chatID, replyTo, err.Error())
		return
	}
	sess, ok, err := d.hub.Get(ctx, id)
	if err != nil {
		d.reply(ctx, chatID, replyTo, "Lookup failed: "+err.Error())
		return
	}
	_, running := d.hub.GetTerminalSession(id)
	if vErr := ValidateSessionExists(ok, running); vErr != nil {
		d.reply(ctx, chatID, replyTo, "Cannot link: "+vErr.Error())
		return
	}
	prior := d.links.Set(chatID, id)
	// Start forwarder for the new session, and tear down the prior one
	// if this was the last listener.
	d.forwarder.EnsureForSession(ctx, id)
	if prior != "" && prior != id && !d.forwarder.HasListeners(prior) {
		d.forwarder.StopForSession(prior)
	}
	name := sess.Name
	if name == "" {
		name = sess.SessionType
	}
	body := fmt.Sprintf("🔗  Linked this chat → session `%s` (%s).\n\n"+
		"Plain messages now go to the session.\n"+
		"Use /unlink to detach.", id, name)
	d.reply(ctx, chatID, replyTo, body)
}

// cmdUnlink: /unlink — removes the binding for this chat.
func (d *Dispatcher) cmdUnlink(ctx context.Context, chatID int64, replyTo int64) {
	prior := d.links.Remove(chatID)
	if prior == "" {
		d.reply(ctx, chatID, replyTo, "This chat is not linked to any session.")
		return
	}
	if !d.forwarder.HasListeners(prior) {
		d.forwarder.StopForSession(prior)
	}
	d.reply(ctx, chatID, replyTo,
		fmt.Sprintf("✗  Unlinked from session `%s`.", prior))
}

// cmdLinks: /links — show every active link (admin / debug view).
func (d *Dispatcher) cmdLinks(ctx context.Context, chatID int64, replyTo int64) {
	d.reply(ctx, chatID, replyTo, "*Active links*\n\n"+ShortLinkSummary(d.links.All()))
}

// cmdSend: /send <session_id> <text> — one-shot message to a session,
// without binding the current chat. Useful for kicking a single command
// without then having every reply forwarded.
func (d *Dispatcher) cmdSend(ctx context.Context, chatID int64, replyTo int64, arg string) {
	rawID, payload := splitFirst(arg)
	if rawID == "" || payload == "" {
		d.reply(ctx, chatID, replyTo, "Usage: /send <number|name> <text>")
		return
	}
	id, err := d.resolver.Resolve(ctx, chatID, rawID)
	if err != nil {
		d.reply(ctx, chatID, replyTo, err.Error())
		return
	}
	if err := d.writeTextThenEnter(id, payload); err != nil {
		d.reply(ctx, chatID, replyTo, "Send failed: "+err.Error())
		return
	}
	d.reply(ctx, chatID, replyTo,
		fmt.Sprintf("➡  Sent %d bytes to `%s`.", len(payload)+1, id))
}

func (d *Dispatcher) cmdWhoami(ctx context.Context, chatID int64, m *Message) {
	from := "?"
	if m.From != nil {
		from = "@" + m.From.Username
		if from == "@" {
			from = m.From.FirstName
		}
	}
	body := fmt.Sprintf("Chat ID: `%d`\nUser:   %s\nChat type: %s",
		chatID, from, m.Chat.Type)
	d.reply(ctx, chatID, m.MessageID, body)
}

// ── Helpers ────────────────────────────────────────────────────

// reply sends a Markdown-formatted message. Falls back to plain text on
// parse error (stray underscore in a path, etc.).
func (d *Dispatcher) reply(parent context.Context, chatID int64, replyTo int64, text string) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	opts := &SendOpts{ParseMode: "Markdown", ReplyToMessageID: replyTo}
	if _, err := d.bot.Send(ctx, chatID, text, opts); err != nil {
		_, _ = d.bot.Send(ctx, chatID, text, &SendOpts{ReplyToMessageID: replyTo})
	}
}

// replyHTML sends an HTML-formatted message. Falls back to plain text.
func (d *Dispatcher) replyHTML(parent context.Context, chatID int64, replyTo int64, html string) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	opts := &SendOpts{ParseMode: "HTML", ReplyToMessageID: replyTo, DisablePreview: true}
	if _, err := d.bot.Send(ctx, chatID, html, opts); err != nil {
		// Strip HTML tags for plain fallback
		plain := strings.NewReplacer(
			"<b>", "", "</b>", "",
			"<i>", "", "</i>", "",
			"<code>", "", "</code>", "",
			"&amp;", "&", "&lt;", "<", "&gt;", ">",
		).Replace(html)
		_, _ = d.bot.Send(ctx, chatID, plain, &SendOpts{ReplyToMessageID: replyTo})
	}
}

func splitCommand(text string) (cmd, arg string) {
	// Strip optional bot mention suffix: "/status@my_bot foo bar" → "/status", "foo bar"
	if i := strings.IndexAny(text, " @"); i > 0 {
		cmd = text[:i]
		switch text[i] {
		case '@':
			// drop everything until the first space (or EOL)
			rest := text[i+1:]
			if sp := strings.Index(rest, " "); sp >= 0 {
				arg = strings.TrimSpace(rest[sp:])
			}
		case ' ':
			arg = strings.TrimSpace(text[i+1:])
		}
		return
	}
	return text, ""
}

func parseTailArg(arg string) (id string, n int) {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		return "", 0
	}
	id = parts[0]
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &n)
	}
	return id, n
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func shortenPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 4 {
		return p
	}
	return ".../" + strings.Join(parts[len(parts)-3:], "/")
}

// splitFirst splits "abc rest of text" → ("abc", "rest of text").
func splitFirst(s string) (first, rest string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx:])
}

// writeToSession sends raw bytes to a session's PTY input and resets the
// idle notification flag so the NEXT idle after Claude responds will
// trigger a fresh push to Telegram.
func (d *Dispatcher) writeToSession(sessionID, payload string) error {
	ts, ok := d.hub.GetTerminalSession(sessionID)
	if !ok {
		return fmt.Errorf("session %s is not running", sessionID)
	}
	if err := ts.WriteInput([]byte(payload)); err != nil {
		return err
	}
	// Input → output → next idle should send a fresh notification.
	if d.notifier != nil {
		d.notifier.ResetIdleFlag(sessionID)
	}
	return nil
}

// writeTextThenEnter writes user text to the PTY, pauses briefly, then
// sends \r (Enter) as a separate write. TUI applications in raw mode
// (Claude Code, Codex, etc.) process input character-by-character via
// their event loop. Sending "text\r" as a single atomic write sometimes
// causes the \r to be missed — the framework reads the whole payload in
// one read() and processes the text but doesn't fire the submit event.
// Splitting into two writes with a 100 ms gap mimics human typing and
// ensures the TUI has processed the text before Enter arrives.
func (d *Dispatcher) writeTextThenEnter(sessionID, text string) error {
	ts, ok := d.hub.GetTerminalSession(sessionID)
	if !ok {
		return fmt.Errorf("session %s is not running", sessionID)
	}
	// 1. Write the text content.
	if err := ts.WriteInput([]byte(text)); err != nil {
		return fmt.Errorf("write text: %w", err)
	}
	// 2. Brief pause so the TUI event loop processes the text insertion.
	time.Sleep(100 * time.Millisecond)
	// 3. Send Enter (carriage return) as a separate write.
	if err := ts.WriteInput([]byte("\r")); err != nil {
		return fmt.Errorf("write enter: %w", err)
	}
	// Input → output → next idle should send a fresh notification.
	if d.notifier != nil {
		d.notifier.ResetIdleFlag(sessionID)
	}
	return nil
}

// forwardToSession is the routing path for plain text in linked chats
// and for replies to notifications. Acks the user with a small ✓ via a
// reply-marker message (Telegram doesn't have a "delivery receipt" but
// the bot should at least confirm receipt — silent send is confusing on
// mobile when the user can't see the cursor land in the terminal).
func (d *Dispatcher) forwardToSession(ctx context.Context, chatID, replyTo int64, sessionID, text string) {
	if err := d.writeTextThenEnter(sessionID, text); err != nil {
		d.reply(ctx, chatID, replyTo, "Send failed: "+err.Error())
		return
	}
	// Ensure forwarder is running so the session's reply comes back to
	// this chat (covers the reply-to-notification case where no /link
	// was previously set up).
	d.forwarder.EnsureForSession(ctx, sessionID)
	if d.links.Get(chatID) == "" {
		// Reply-to-notification path with no /link: keep the answer
		// flowing into THIS chat for at least the next coalescing
		// window by binding implicitly. User can /unlink at any time.
		d.links.Set(chatID, sessionID)
	}
	// React with a tiny acknowledgement so the user knows it landed.
	_, _ = d.bot.Send(ctx, chatID,
		"✓ "+fmt.Sprintf("%d", len(text)+1)+" bytes → `"+sessionID+"`",
		&SendOpts{ParseMode: "Markdown", ReplyToMessageID: replyTo, DisablePreview: true},
	)
}

// handleCallback processes an inline-keyboard button press.
// Data format: "action:sessionID:payload"
//   send:abc123:y\r   → write "y\r" to session abc123
//   screen:abc123     → send cleaned screen to this chat
func (d *Dispatcher) handleCallback(ctx context.Context, cq *CallbackQuery) {
	chatID := int64(0)
	replyTo := int64(0)
	if cq.Message != nil {
		chatID = cq.Message.Chat.ID
		replyTo = cq.Message.MessageID
	}

	parts := strings.SplitN(cq.Data, ":", 3)
	if len(parts) < 2 {
		d.bot.AnswerCallbackQuery(ctx, cq.ID, "Invalid button data")
		return
	}
	action := parts[0]
	sessionID := parts[1]

	switch action {
	case "send":
		if len(parts) < 3 {
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Missing payload")
			return
		}
		payload := parts[2]
		if err := d.writeToSession(sessionID, payload); err != nil {
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Failed: "+err.Error())
			return
		}
		// Show which button was pressed
		label := callbackLabel(payload)
		d.bot.AnswerCallbackQuery(ctx, cq.ID, "✓ Sent: "+label)

	case "screen":
		d.bot.AnswerCallbackQuery(ctx, cq.ID, "Loading screen...")
		text := d.cleanSnapshot(sessionID, 60)
		if text == "" {
			d.reply(ctx, chatID, replyTo, "(screen empty)")
			return
		}
		d.replyHTML(ctx, chatID, replyTo, formatForTelegram(text))

	case "multi_toggle":
		if len(parts) < 3 {
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Missing key")
			return
		}
		key := parts[2]
		st := d.multiSelect.Toggle(chatID, replyTo, key)
		if st == nil {
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Selection expired")
			return
		}
		kb := makeMultiSelectKeyboard(st.SessionID, st.Options, st.Checked)
		if err := d.bot.EditMessageReplyMarkup(ctx, chatID, replyTo, kb); err != nil {
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Refresh failed")
			return
		}
		mark := "☐"
		if st.Checked[key] {
			mark = "☑"
		}
		d.bot.AnswerCallbackQuery(ctx, cq.ID, mark+" "+key)

	case "multi_submit":
		st := d.multiSelect.Submit(chatID, replyTo)
		if st == nil {
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Already submitted or expired")
			return
		}
		var picks []string
		for _, o := range st.Options {
			if st.Checked[o.Key] {
				picks = append(picks, o.Key)
			}
		}
		if len(picks) == 0 {
			// Put the state back so the user can still pick something.
			d.multiSelect.Create(chatID, replyTo, st.SessionID, st.Options)
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Pick at least one option")
			return
		}
		payload := strings.Join(picks, " ")
		if err := d.writeTextThenEnter(sessionID, payload); err != nil {
			d.multiSelect.Create(chatID, replyTo, st.SessionID, st.Options)
			d.bot.AnswerCallbackQuery(ctx, cq.ID, "Send failed: "+err.Error())
			return
		}
		receipt := "✅ Submitted: " + strings.Join(picks, ", ")
		// Pass an explicit empty keyboard so the checkboxes disappear
		// atomically with the text edit (Telegram keeps the old markup
		// if reply_markup is omitted).
		emptyKB := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{}}
		if err := d.bot.EditMessageText(ctx, chatID, replyTo, receipt, &SendOpts{ReplyMarkup: emptyKB}); err != nil {
			// Editing failed (message too old, etc.) — fall back to
			// clearing just the buttons so the user can't re-submit.
			_ = d.bot.EditMessageReplyMarkup(ctx, chatID, replyTo, nil)
		}
		d.bot.AnswerCallbackQuery(ctx, cq.ID, "Submitted")

	default:
		d.bot.AnswerCallbackQuery(ctx, cq.ID, "Unknown action")
	}
}

// sendControlKey sends a control character to the linked session.
// Used by /cc, /cd, /tab, /enter.
func (d *Dispatcher) sendControlKey(ctx context.Context, chatID, replyTo int64, key, label string) {
	sid := d.links.Get(chatID)
	if sid == "" {
		d.reply(ctx, chatID, replyTo,
			"No linked session. Use /link <id> first, then /cc /tab etc.")
		return
	}
	if err := d.writeToSession(sid, key); err != nil {
		d.reply(ctx, chatID, replyTo, label+" failed: "+err.Error())
		return
	}
	d.reply(ctx, chatID, replyTo, "⚡ "+label+" → `"+sid+"`")
}

// sendLinkedText sends a fixed text to the linked session.
// Used by /yes, /no.
func (d *Dispatcher) sendLinkedText(ctx context.Context, chatID, replyTo int64, text string) {
	sid := d.links.Get(chatID)
	if sid == "" {
		d.reply(ctx, chatID, replyTo,
			"No linked session. Use /link <id> first.")
		return
	}
	if err := d.writeToSession(sid, text); err != nil {
		d.reply(ctx, chatID, replyTo, "Send failed: "+err.Error())
		return
	}
	display := strings.TrimSpace(text)
	d.reply(ctx, chatID, replyTo,
		fmt.Sprintf("✓ `%s` → `%s`", display, sid))
}

// callbackLabel maps a raw PTY payload to a human-readable label
// for the Telegram callback acknowledgement toast.
func callbackLabel(payload string) string {
	switch payload {
	case "y\r":
		return "Yes"
	case "n\r":
		return "No"
	case "a\r":
		return "Always"
	case "\x03":
		return "Ctrl+C"
	case "\r":
		return "Enter"
	default:
		clean := strings.TrimSuffix(payload, "\r")
		if len(clean) == 1 && clean[0] >= '1' && clean[0] <= '9' {
			return "Option " + clean
		}
		if len(clean) == 1 && clean[0] >= 'a' && clean[0] <= 'z' {
			return "Option (" + clean + ")"
		}
		return clean
	}
}
