package telegram

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray/kernel/hub"
	"github.com/opendray/opendray/plugin"
)

// Manager is the top-level entry point used by the gateway. It watches the
// "telegram" plugin's enabled flag + config in a loop and starts / stops /
// reloads the underlying Bot + Notifier + Forwarder accordingly.
//
// Single canonical instance per server. Safe for concurrent Snapshot calls.
type Manager struct {
	plugins *plugin.Runtime
	hub     *hub.Hub
	bus     *plugin.HookBus
	logger  *slog.Logger
	links   *LinkStore // shared across bot lifecycles — survives reloads

	mu        sync.Mutex
	bot       *Bot
	notifier  *Notifier
	forwarder *Forwarder
	cfgHash   string // cheap dedup so we don't reload every tick

	rootCtx context.Context
}

// NewManager constructs a manager. Call Start to begin the watch loop.
func NewManager(plugins *plugin.Runtime, h *hub.Hub, bus *plugin.HookBus, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		plugins: plugins,
		hub:     h,
		bus:     bus,
		logger:  logger,
		links:   NewLinkStore(""), // default path under os.TempDir()
	}
}

// Links exposes the link store for the HTTP layer (panel "active links" view).
func (m *Manager) Links() *LinkStore { return m.links }

// Start launches the watch loop. Returns immediately.
func (m *Manager) Start(ctx context.Context) {
	m.rootCtx = ctx
	m.reconcile() // one immediate pass so the bot is up before the first tick
	go m.watchLoop(ctx)
}

// Stop tears down the bot and notifier.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tearDownLocked()
}

// Snapshot returns the current bot status (or zero value if not running).
func (m *Manager) Snapshot() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.bot == nil {
		return Status{}
	}
	return m.bot.Snapshot()
}

// SendTest sends a test message to the configured notification chat.
// Returns the chat id used or an error when no bot is running / chat configured.
func (m *Manager) SendTest(ctx context.Context, body string) (int64, error) {
	m.mu.Lock()
	bot := m.bot
	m.mu.Unlock()
	if bot == nil {
		return 0, errBotNotRunning
	}
	chatID := bot.NotifyChatID()
	if chatID == 0 {
		return 0, errNoNotifyChat
	}
	if body == "" {
		body = "👋 Test message from OpenDray at " + time.Now().Format(time.RFC3339)
	}
	if _, err := bot.Send(ctx, chatID, body, &SendOpts{}); err != nil {
		return 0, err
	}
	return chatID, nil
}

// ── Internals ───────────────────────────────────────────────

func (m *Manager) watchLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.Stop()
			return
		case <-ticker.C:
			m.reconcile()
		}
	}
}

func (m *Manager) reconcile() {
	cfg, enabled, ok := m.findPluginConfig()
	hash := configHash(cfg, enabled, ok)

	m.mu.Lock()
	defer m.mu.Unlock()

	if hash == m.cfgHash {
		return
	}
	m.cfgHash = hash
	m.tearDownLocked()
	if !ok || !enabled {
		return
	}
	if cfg.BotToken == "" {
		m.logger.Info("telegram: plugin enabled but no bot token configured — skipping")
		return
	}
	if len(cfg.AllowedChatIDs) == 0 {
		m.logger.Warn("telegram: plugin enabled but no allowedChatIds — refusing to start (insecure)")
		return
	}
	m.bot = New(cfg, nil, m.logger)
	m.forwarder = NewForwarder(m.bot, m.hub, m.links, m.logger)
	m.notifier = NewNotifier(m.bot, m.hub, m.bus, m.links, m.forwarder)
	// Dispatcher gets notifier so writeToSession can reset idle flags.
	m.bot.handler = NewDispatcher(m.bot, m.hub, m.links, m.forwarder, m.notifier)
	if err := m.bot.Start(m.rootCtx); err != nil {
		m.logger.Warn("telegram: bot start failed", "error", err)
		m.bot = nil
		m.forwarder = nil
		m.notifier = nil
		return
	}
	m.notifier.Start()

	// Boot forwarders for any links that survived restart. If the session
	// is already gone, EnsureForSession is a no-op until the user /links
	// again.
	for _, l := range m.links.All() {
		m.forwarder.EnsureForSession(m.rootCtx, l.SessionID)
	}
}

func (m *Manager) tearDownLocked() {
	if m.notifier != nil {
		m.notifier.Stop()
		m.notifier = nil
	}
	if m.forwarder != nil {
		m.forwarder.StopAll()
		m.forwarder = nil
	}
	if m.bot != nil {
		m.bot.Stop()
		m.bot = nil
	}
}

// findPluginConfig pulls the running plugin's config, applies the
// `OPENDRAY_TELEGRAM_BOT_TOKEN` env-var fallback, and returns a parsed Config.
func (m *Manager) findPluginConfig() (Config, bool, bool) {
	for _, pi := range m.plugins.ListInfo() {
		if pi.Provider.Name != "telegram" {
			continue
		}
		raw := pi.Config
		token := stringFrom(raw, "botToken")
		if token == "" {
			token = os.Getenv("OPENDRAY_TELEGRAM_BOT_TOKEN")
		}
		allowed := parseChatIDs(stringFrom(raw, "allowedChatIds"))
		notify := parseChatID(stringFrom(raw, "notifyChatId"))
		cfg := Config{
			BotToken:        token,
			AllowedChatIDs:  allowed,
			NotifyChatID:    notify,
			NotifyOnIdle:    boolFrom(raw, "notifyOnIdle", true),
			NotifyOnExit:    boolFrom(raw, "notifyOnExit", true),
			TailLines:       intFrom(raw, "tailLines", 20),
			PollInterval:    time.Duration(intFrom(raw, "pollInterval", 25)) * time.Second,
			ExtraClaudeDirs: parseExtraClaudeDirs(stringFrom(raw, "extraClaudeDirs")),
		}
		return cfg, pi.Enabled, true
	}
	return Config{}, false, false
}

// ── Plain helpers (kept in this file to avoid sprawling utils.go) ──

func stringFrom(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intFrom(m map[string]any, key string, fallback int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return fallback
}

func boolFrom(m map[string]any, key string, fallback bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return fallback
}

func parseChatIDs(s string) map[int64]bool {
	out := map[int64]bool{}
	for _, p := range strings.Split(s, ",") {
		id := parseChatID(strings.TrimSpace(p))
		if id != 0 {
			out[id] = true
		}
	}
	return out
}

func parseChatID(s string) int64 {
	if s == "" {
		return 0
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// configHash is a coarse fingerprint of the bits the bot cares about,
// so a tick that finds nothing changed is a no-op.
func configHash(c Config, enabled, found bool) string {
	if !found {
		return "missing"
	}
	if !enabled {
		return "disabled"
	}
	var b strings.Builder
	b.WriteString("on:")
	b.WriteString(c.BotToken)
	b.WriteString("|")
	for id := range c.AllowedChatIDs {
		b.WriteString(strconv.FormatInt(id, 10))
		b.WriteString(",")
	}
	b.WriteString("|n=")
	b.WriteString(strconv.FormatInt(c.NotifyChatID, 10))
	b.WriteString("|i=")
	b.WriteString(strconv.FormatBool(c.NotifyOnIdle))
	b.WriteString("|x=")
	b.WriteString(strconv.FormatBool(c.NotifyOnExit))
	b.WriteString("|t=")
	b.WriteString(strconv.Itoa(c.TailLines))
	b.WriteString("|p=")
	b.WriteString(c.PollInterval.String())
	b.WriteString("|ecd=")
	b.WriteString(strings.Join(c.ExtraClaudeDirs, ","))
	return b.String()
}

// Sentinel errors so the HTTP layer can map to specific status codes.
var (
	errBotNotRunning = newErr("telegram bot is not running — check plugin enabled & token configured")
	errNoNotifyChat  = newErr("no notification chat configured — set notifyChatId or allowedChatIds")
)

type strErr string

func (s strErr) Error() string { return string(s) }

func newErr(s string) strErr { return strErr(s) }
