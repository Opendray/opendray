// Package app is opendray's composition root.
//
// All subsystems (config -> store -> eventbus -> session -> gateway) are
// constructed here. Subsystem packages must not import each other through
// globals; dependencies flow only via constructor parameters wired in
// this package.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/audit"
	"github.com/opendray/opendray-v2/internal/auth"
	"github.com/opendray/opendray-v2/internal/backup"
	"github.com/opendray/opendray-v2/internal/catalog"
	"github.com/opendray/opendray-v2/internal/channel"
	"github.com/opendray/opendray-v2/internal/channel/bridge"     // also registers kind=bridge via init()
	_ "github.com/opendray/opendray-v2/internal/channel/dingtalk" // register kind=dingtalk
	_ "github.com/opendray/opendray-v2/internal/channel/discord"  // register kind=discord
	_ "github.com/opendray/opendray-v2/internal/channel/feishu"   // register kind=feishu
	_ "github.com/opendray/opendray-v2/internal/channel/slack"    // register kind=slack
	_ "github.com/opendray/opendray-v2/internal/channel/telegram" // register kind=telegram
	_ "github.com/opendray/opendray-v2/internal/channel/wechat"   // register kind=wechat (wxpusher push)
	_ "github.com/opendray/opendray-v2/internal/channel/wecom"    // register kind=wecom
	"github.com/opendray/opendray-v2/internal/cliacct"
	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/cortex"
	customtask "github.com/opendray/opendray-v2/internal/customtask"
	"github.com/opendray/opendray-v2/internal/eventbus"
	fsapi "github.com/opendray/opendray-v2/internal/fs"
	"github.com/opendray/opendray-v2/internal/gateway"
	gitapi "github.com/opendray/opendray-v2/internal/git"
	"github.com/opendray/opendray-v2/internal/gitactivity"
	githost "github.com/opendray/opendray-v2/internal/githost"
	"github.com/opendray/opendray-v2/internal/integration"
	"github.com/opendray/opendray-v2/internal/knowledge"
	mcpapi "github.com/opendray/opendray-v2/internal/mcp"
	"github.com/opendray/opendray-v2/internal/memconflict"
	"github.com/opendray/opendray-v2/internal/memhealth"
	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/memory/capture"
	"github.com/opendray/opendray-v2/internal/memory/cleaner"
	"github.com/opendray/opendray-v2/internal/memory/injector"
	"github.com/opendray/opendray-v2/internal/memory/summarizer"
	memworker "github.com/opendray/opendray-v2/internal/memory/worker"
	"github.com/opendray/opendray-v2/internal/memquery"
	notesapi "github.com/opendray/opendray-v2/internal/notes"
	"github.com/opendray/opendray-v2/internal/projectdoc"
	"github.com/opendray/opendray-v2/internal/projectscan"
	"github.com/opendray/opendray-v2/internal/prwatcher"
	searchapi "github.com/opendray/opendray-v2/internal/search"
	"github.com/opendray/opendray-v2/internal/session"
	"github.com/opendray/opendray-v2/internal/settings"
	"github.com/opendray/opendray-v2/internal/skills"
	"github.com/opendray/opendray-v2/internal/store"
	vaultgit "github.com/opendray/opendray-v2/internal/vaultgit"
	"github.com/opendray/opendray-v2/internal/version"
)

type App struct {
	cfg             config.Config
	log             *slog.Logger
	store           *store.Store
	bus             *eventbus.Hub
	sessions        *session.Manager
	channels        *channel.Hub
	integrations    *integration.Service
	healthCheck     *integration.HealthChecker
	audit           *audit.Sink
	intgrCallLogger *integration.CallLogger
	vaultSync       *vaultgit.Syncer
	// liveBackup owns the backup Service + scheduler. Always non-nil
	// after New returns, but Service() returns nil when the feature
	// is off. Disarm on shutdown to stop the scheduler goroutine.
	liveBackup           *backup.LiveBackup
	captureEngine        *capture.Engine // ambient memory capture loop
	journaler            *projectdoc.Journaler
	memorySvc            *memory.Service     // shared pgvector memory; nil when memory disabled
	memoryMirror         *memory.Mirror      // M-U Phase 5 one-time file-memory import; nil when disabled
	projectDocSvc        *projectdoc.Service // owns the M-PB journal embed backfill loop
	cleanerScheduler     *cleaner.Scheduler  // optional; nil when scheduler is off
	gitActivityScheduler *gitactivity.Scheduler
	conflictScheduler    *memconflict.Scheduler // M-PC daily cross-layer conflict scan
	prWatcher            *prwatcher.Service     // polls open PRs' CI checks and emits pr.checks_completed
	cliacctWatcher       *cliacct.Watcher       // optional; nil when [providers.claude] watcher_enabled = false
	server               *http.Server
	knowledgeAnchorer    *knowledge.Anchorer            // M-KG Phase 1 anchor sweep; nil when [knowledge] disabled
	knowledgeCompiler    *knowledge.ExperienceCompiler  // experience compiler: cross-project recurrence mining; nil when disabled
	knowledgeSvc         *knowledge.Service             // M-KG Phase 6 embed backfill; nil when disabled
	knowledgeKBDrafter   *knowledge.KBDrafter           // M-KB curated KB-page drafting; nil when disabled
	knowledgeConsolidate *knowledge.ConsolidationEngine // P-C unified anchor→compile→KB loop; nil when disabled
}

// knowledgeMemorySource adapts *memory.Service to knowledge.MemorySource so
// the knowledge tier reads episodic memory through an interface it owns
// (one-way dependency: knowledge never imports internal/memory).
type knowledgeMemorySource struct{ mem *memory.Service }

func (a knowledgeMemorySource) ListProjectKeys(ctx context.Context) ([]string, error) {
	return a.mem.ListScopeKeys(ctx, memory.ScopeProject)
}

func (a knowledgeMemorySource) ListProjectMemories(ctx context.Context, scopeKey string, limit int) ([]knowledge.MemoryRow, error) {
	mems, err := a.mem.List(ctx, memory.ScopeProject, scopeKey, limit)
	if err != nil {
		return nil, err
	}
	out := make([]knowledge.MemoryRow, 0, len(mems))
	for _, m := range mems {
		out = append(out, knowledge.MemoryRow{ID: m.ID, Text: m.Text, ScopeKey: m.ScopeKey, CreatedAt: m.CreatedAt})
	}
	return out, nil
}

// ListAllMemories gathers project-scoped memories across every project key so
// the cross-project KB pages distil straight from Memory (P-G). Per-key
// failures are skipped; the overall cap bounds the total returned.
func (a knowledgeMemorySource) ListAllMemories(ctx context.Context, limit int) ([]knowledge.MemoryRow, error) {
	keys, err := a.mem.ListScopeKeys(ctx, memory.ScopeProject)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 400
	}
	// Spread the budget across projects so one large project can't crowd out
	// the rest of the ecosystem's facts; at least a handful per project.
	perKey := limit
	if n := len(keys); n > 0 {
		if perKey = limit / n; perKey < 20 {
			perKey = 20
		}
	}
	out := make([]knowledge.MemoryRow, 0, limit)
	for _, k := range keys {
		if len(out) >= limit {
			break
		}
		rows, rerr := a.ListProjectMemories(ctx, k, perKey)
		if rerr != nil {
			continue
		}
		out = append(out, rows...)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// knowledgeJournalSource adapts *projectdoc.Service to knowledge.JournalSource
// so reflection distills playbooks from real session work-traces (the project
// journal), not just declarative memory facts. One-way dependency: knowledge
// owns the interface; the app adapts projectdoc to it.
type knowledgeJournalSource struct{ pd *projectdoc.Service }

func (a knowledgeJournalSource) ListJournal(ctx context.Context, scopeKey string, limit int) ([]knowledge.JournalEntry, error) {
	logs, err := a.pd.ListLogs(ctx, scopeKey, limit)
	if err != nil {
		return nil, err
	}
	out := make([]knowledge.JournalEntry, 0, len(logs))
	for _, l := range logs {
		out = append(out, knowledge.JournalEntry{
			Title:     l.Title,
			Content:   l.Content,
			Kind:      string(l.Kind),
			CreatedAt: l.CreatedAt,
		})
	}
	return out, nil
}

// knowledgeDocSink adapts *projectdoc.Service to knowledge.DocSink so the KB
// drafter writes curated pages INTO the note system (project_docs). A page the
// operator has edited (updated_by=operator) is reported HumanLocked so the
// drafter never overwrites it. One-way dependency (knowledge owns the interface).
type knowledgeDocSink struct{ pd *projectdoc.Service }

func (a knowledgeDocSink) GetKBDoc(ctx context.Context, cwd, kind string) (knowledge.KBDoc, error) {
	d, err := a.pd.GetDoc(ctx, cwd, projectdoc.Kind(kind))
	if errors.Is(err, projectdoc.ErrNotFound) {
		return knowledge.KBDoc{}, nil
	}
	if err != nil {
		return knowledge.KBDoc{}, err
	}
	return knowledge.KBDoc{
		Content:     d.Content,
		HumanLocked: d.UpdatedBy == projectdoc.AuthorOperator,
		Exists:      true,
	}, nil
}

func (a knowledgeDocSink) PutKBDoc(ctx context.Context, cwd, kind, content string) error {
	_, err := a.pd.PutDoc(ctx, cwd, projectdoc.Kind(kind), content, projectdoc.AuthorAgent)
	return err
}

// knowledgeLifecycle adapts *projectdoc.Service to knowledge.LifecycleFilter so
// the reflector skips frozen (paused/archived) projects during distillation
// (P-D). A status lookup error defaults to "not frozen" — we'd rather
// over-distill than silently drop an active project.
type knowledgeLifecycle struct{ pd *projectdoc.Service }

func (a knowledgeLifecycle) IsFrozen(ctx context.Context, cwd string) bool {
	status, err := a.pd.GetStatus(ctx, cwd)
	if err != nil {
		return false
	}
	return status.IsFrozen()
}

// knowledgeProposalSink adapts *projectdoc.Service to knowledge.ProposalSink so
// the KB drafter files an update PROPOSAL for a human-locked Knowledge page
// instead of overwriting it (B3 — Iterate). One-way dependency.
type knowledgeProposalSink struct{ pd *projectdoc.Service }

func (a knowledgeProposalSink) HasPendingKBProposal(ctx context.Context, cwd, kind string) (bool, error) {
	props, err := a.pd.ListPendingProposals(ctx, cwd)
	if err != nil {
		return false, err
	}
	for _, p := range props {
		if string(p.Kind) == kind {
			return true, nil
		}
	}
	return false, nil
}

func (a knowledgeProposalSink) ProposeKBDoc(ctx context.Context, cwd, kind, content, reason string) error {
	_, err := a.pd.ProposeDoc(ctx, cwd, projectdoc.Kind(kind), content, reason, "")
	return err
}

// knowledgeLLM adapts the memory worker registry to knowledge.LLM so the
// Phase 1B entity extractor gets a general completion path. It borrows the
// TaskCapture worker config (the closest existing touchpoint: extract
// structure from text) rather than introducing a dedicated worker TaskKind +
// migration. A missing/disabled worker surfaces as an error, which the
// anchorer treats as "skip fine-entity extraction" (degrades to 1A).
type knowledgeLLM struct {
	reg       *memworker.Registry
	maxTokens int                // 0 → 512 (small extraction default)
	timeout   time.Duration      // 0 → 20s
	task      memworker.TaskKind // 0-value → TaskCapture
}

func (a knowledgeLLM) Complete(ctx context.Context, system, user string) (string, error) {
	maxTokens := a.maxTokens
	if maxTokens == 0 {
		maxTokens = 512 // small entity-extraction default
	}
	timeout := a.timeout
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	task := a.task
	if task == "" {
		task = memworker.TaskCapture
	}
	resp, err := a.reg.Run(ctx, memworker.Request{
		Task:         task,
		SystemPrompt: system,
		UserInput:    user,
		MaxTokens:    maxTokens,
		Timeout:      timeout,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// knowledgeSkillSink writes a rendered SKILL.md into the vault skills dir so
// the skills loader picks up AI-promoted skills (one-way: knowledge owns the
// SkillSink interface; the app provides this filesystem impl).
type knowledgeSkillSink struct{ dir string }

func (s knowledgeSkillSink) WriteSkill(_ context.Context, id, markdown string) error {
	d := filepath.Join(s.dir, id)
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, "SKILL.md"), []byte(markdown), 0o600)
}

// WriteSkillAsset places an extra file next to SKILL.md — the experience
// compiler ships a skill's executable form as run.sh. Scripts get the
// executable bit; everything else stays 0600. The id/name are slugs minted
// server-side (never operator input), so the join stays inside the vault.
func (s knowledgeSkillSink) WriteSkillAsset(_ context.Context, id, name, content string) error {
	d := filepath.Join(s.dir, id)
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if strings.HasSuffix(name, ".sh") {
		mode = 0o700
	}
	return os.WriteFile(filepath.Join(d, name), []byte(content), mode)
}

func (s knowledgeSkillSink) DeleteSkill(_ context.Context, id string) error {
	if id == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(s.dir, id))
}

// New wires the runtime dependencies but does not start any goroutines.
// Caller is responsible for calling Run or Close.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	log, logRing, err := newLogger(cfg.Log)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(ctx, cfg.Database.URL, cfg.Database.MaxConns)
	if err != nil {
		return nil, err
	}

	// Auto-apply pending migrations on startup, fail-closed (M-U §7).
	// Previously migrations ran only via the explicit `opendray migrate`
	// command, so `opendray update` landed the new binary against the
	// old schema until the operator separately migrated — unacceptable
	// for a smooth upgrade. Migrations are idempotent, forward-only, and
	// transactional (tracked in schema_migrations), so running them here
	// is a no-op once applied. This must run BEFORE catalog.New below,
	// which upserts seed rows into tables migration 0001 creates (the
	// fresh-DB ordering bug from #162). `opendray migrate` stays as a
	// standalone command for operators who prefer to migrate explicitly.
	// Pre-migration safety snapshot (fail-closed). Runs before the
	// schema changes so an upgrade always has a restorable point; a
	// no-op once the DB is already up to date.
	pending, perr := st.PendingMigrations(ctx)
	if perr != nil {
		st.Close()
		return nil, fmt.Errorf("check pending migrations: %w", perr)
	}
	if len(pending) > 0 {
		preKey, kerr := backup.LoadPassphrase()
		if kerr != nil {
			// A configured-but-broken key source must not silently
			// degrade the snapshot to plaintext — fail closed.
			st.Close()
			return nil, fmt.Errorf("pre-migrate snapshot: backup key load: %w", kerr)
		}
		if gerr := backup.GuardPreMigrate(ctx, pending, backup.PreMigrateOptions{
			DSN:        cfg.Database.URL,
			Dir:        filepath.Join(defaultBackupDir(cfg.Backup.LocalDir, "backups"), "premigrate"),
			PgDumpPath: cfg.Backup.PgDumpPath,
			Passphrase: preKey.Passphrase,
			Log:        log,
		}); gerr != nil {
			st.Close()
			return nil, fmt.Errorf("pre-migrate snapshot: %w", gerr)
		}
	}

	if err := st.Migrate(ctx, log); err != nil {
		st.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	bus := eventbus.New(log)

	authSvc := auth.New(cfg.Admin, bus, log)
	authHandlers := auth.NewHandlers(authSvc, log)

	cat, err := catalog.New(st.Pool(), log)
	if err != nil {
		st.Close()
		return nil, err
	}
	if err := cat.Sync(ctx); err != nil {
		st.Close()
		return nil, err
	}
	catalogHandlers := catalog.NewHandlers(cat, bus, log)

	var cliacctOpts []cliacct.Option
	if d := strings.TrimSpace(cfg.Providers.Claude.AccountsDir); d != "" {
		cliacctOpts = append(cliacctOpts, cliacct.WithAccountsDir(expandPath(d)))
	}
	cliacctSvc := cliacct.NewService(st.Pool(), bus, log, cliacctOpts...)
	cliacctHandlers := cliacct.NewHandlers(cliacctSvc, log)
	// Accounts watcher: auto-registers a new account row when
	// ~/.claude-accounts/<name>/.credentials.json appears after a
	// `CLAUDE_CONFIG_DIR=… claude login` on the host. Construction
	// is cheap (no goroutines until Run); Run is started inside
	// App.Run with the other background services so its lifetime
	// tracks the gateway's. cliacctWatcher == nil when the operator
	// has set `[providers.claude] watcher_enabled = false`.
	var cliacctWatcher *cliacct.Watcher
	if cfg.Providers.Claude.WatcherIsEnabled() {
		cliacctWatcher = cliacct.NewWatcher(cliacctSvc, cliacctSvc.AccountsDir(), log)
	}

	// Vault + skills are needed by the SessionProvider so spawn-time
	// injection has them available. Constructed here (before the
	// session manager) so the manager's first Resolve call sees them.
	notesRoot, skillsRoot, gitRoot := resolveVaultPaths(cfg.Vault)
	vault, err := notesapi.New(notesRoot, notesapi.Options{
		PersonalPrefix: cfg.Vault.PersonalPrefix,
		ProjectsPrefix: cfg.Vault.ProjectsPrefix,
	})
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("init notes vault: %w", err)
	}
	log.Info("notes vault ready", "root", vault.Root())
	notesHandlers := notesapi.NewHandlers(vault, log)
	skillsLoader := skills.NewLoader(skillsRoot)
	if list, _ := skillsLoader.List(); len(list) > 0 {
		log.Info("agent skills loaded", "count", len(list),
			"vault", skillsLoader.VaultRoot())
	}

	mcpRoot, secretsFile := resolveMCPPaths(cfg.MCP, notesRoot, skillsRoot)
	mcpLoader := mcpapi.NewLoader(mcpRoot)
	if list, _ := mcpLoader.List(); len(list) > 0 {
		log.Info("mcp registry loaded", "count", len(list),
			"vault", mcpLoader.VaultRoot(), "secrets", secretsFile)
	}

	var sessionOpts []session.ManagerOption
	if d := cfg.Session.Threshold(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleThreshold(d))
	}
	if d := cfg.Session.Interval(); d > 0 {
		sessionOpts = append(sessionOpts, session.WithIdleInterval(d))
	}
	sessionOpts = append(sessionOpts,
		session.WithClaudeHistoryConfig(resolveClaudeHistoryConfig(cfg.Providers.Claude)),
		session.WithCodexHistoryConfig(resolveCodexHistoryConfig(cfg.Providers.Codex)),
		session.WithGeminiHistoryConfig(resolveGeminiHistoryConfig(cfg.Providers.Gemini)),
		session.WithAntigravityHistoryConfig(resolveAntigravityHistoryConfig(cfg.Providers.Antigravity)),
		// Lets Manager.SwitchClaudeAccount migrate the conversation
		// transcript JSONL into the new account's projects/ tree
		// before respawning, so --resume actually finds the
		// conversation and the dialog's "state will be lost" warning
		// stops being a self-fulfilling prophecy.
		session.WithClaudeAccountResolver(cliacctSvc),
		// Phase 2 Tier A: when [providers.claude] auto_failover_enabled
		// is true, pumpStdout scans each Claude session's PTY output
		// for the "session limit · resets HH:MM" banner and, on a
		// match, automatically switches the session to the next
		// non-throttled enabled account.
		session.WithAutoFailoverEnabled(cfg.Providers.Claude.AutoFailoverIsEnabled()),
	)
	sessionProvider := catalog.NewSessionProvider(cat, cliacctSvc, skillsLoader, mcpLoader, secretsFile, log)
	sessionMgr := session.NewManager(
		st.Pool(),
		bus,
		sessionProvider,
		log,
		sessionOpts...,
	)
	// Best-effort reconcile of leftover rows from a prior gateway
	// process — their PTYs are gone, so flip them to 'ended' so the
	// web UI can stop the WS reconnect loop and show EndedSessionView.
	if err := sessionMgr.ReconcileStartup(ctx); err != nil {
		log.Warn("session reconcile on startup failed", "err", err)
	}
	// Built before the session handlers so POST /sessions can inherit an
	// integration's configured spawn defaults (provider / model / claude
	// account). The handlers (proxy, events, health) that consume it are
	// constructed further below where the rest of the integration surface
	// is assembled.
	intgrSvc := integration.NewService(st.Pool(), bus, log)
	sessionHandlers := session.NewHandlers(sessionMgr, log,
		// Inject the cliacct validator so POST /sessions and
		// PATCH /sessions/{id}/claude-account fail fast with 400
		// when claude_account_id is bogus or disabled, instead of
		// letting the row be persisted/mutated and then erroring
		// at spawn time with 500.
		session.WithClaudeAccountChecker(cliacctSvc),
		// Fill provider/model/claude-account from the integration's
		// configured defaults for sessions an integration creates and
		// the request leaves those fields empty (request still wins).
		session.WithIntegrationDefaults(&integrationDefaultsLookup{svc: intgrSvc}),
	)
	// Now that the session manager exists, let the catalog handler
	// populate RuntimeInfo.ActiveSessions on update-check responses so
	// the UI can warn the operator before upgrading a CLI that running
	// sessions are using.
	catalogHandlers.WithSessionCounter(sessionMgr.ActiveCountByProvider)

	channelHub := channel.NewHub(st.Pool(), bus, log)
	// Plain-text inbound from a channel (e.g. a Telegram reply that
	// is not a slash command) gets forwarded to the last session that
	// notified that channel — letting the operator drive a running
	// CLI from chat without opening the web UI.
	channelHub.SetSessionInput(sessionMgr)
	// Session-aware slash commands (/list, /end, /resume) live in
	// this layer rather than internal/channel so the channel package
	// stays free of the session dependency. The matching idle-card
	// buttons (Resume / End) emit the same `cmd:/...` payloads.
	registerChannelCommands(channelHub, sessionMgr)
	// Optional single-owner gate: when OPENDRAY_CONTROL_OWNER is set to
	// a Telegram user id, only that account may interact with the bot at
	// all — send text to a session, run commands, or tap controls.
	// Unset = open (single-user/trusted default).
	if authz := senderAuthorizerFromEnv(log); authz != nil {
		channelHub.SetSenderAuthorizer(authz)
	}
	channelHandlers := channel.NewHandlers(channelHub, log)
	bridgeHandlers := bridge.NewHandlers(bridge.DefaultBroker(), log)

	intgrHandlers := integration.NewHandlers(intgrSvc, log)
	intgrCallLogger := integration.NewCallLogger(st.Pool(), log)
	intgrCallLogHandlers := integration.NewCallLogHandlers(intgrCallLogger, log)
	proxyHandlers := integration.NewProxyHandlers(intgrSvc, intgrCallLogger, log)
	eventsHandler := integration.NewEventsHandler(bus, log)
	healthCheck := integration.NewHealthChecker(intgrSvc, bus, log)

	auditSink := audit.NewSink(st.Pool(), bus, log)
	auditSvc := audit.NewService(st.Pool())
	auditHandlers := audit.NewHandlers(auditSvc, log)
	fsHandlers := fsapi.NewHandlers(log)
	gitHandlers := gitapi.NewHandlers(log)
	gitHostSvc := githost.NewService(st.Pool(), log)
	gitHostHandlers := githost.NewHandlers(gitHostSvc, log)
	customTaskSvc := customtask.NewService(st.Pool(), log)
	customTaskHandlers := customtask.NewHandlers(customTaskSvc, log)
	searchHandlers := searchapi.NewHandlers(log)
	skillsHandlers := skills.NewHandlers(skillsLoader, log)
	mcpHandlers := mcpapi.NewHandlers(mcpLoader, secretsFile, log)
	// Vault git ops are scoped to whatever the user told us is the
	// repo root (defaults: notes root if user pinned `notes` directly,
	// otherwise the parent that contains both notes/ and skills/).
	// The githost service is passed so private-repo HTTPS push/pull
	// picks up tokens stored in Plugins → Git hosts.
	vaultGitHandlers, err := vaultgit.NewHandlers(gitRoot, gitHostSvc, log)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("init vault git handlers: %w", err)
	}
	log.Info("vault git ready", "root", gitRoot)
	vaultSyncer := vaultgit.NewSyncer(st.Pool(), bus, vaultGitHandlers, log)

	// Settings: read/write the same config.toml the gateway booted
	// from. Empty FilePath (env-only mode) disables the API — Get
	// returns "no config path" so the UI shows a read-only banner.
	settingsSvc := settings.NewService(cfg.FilePath, log)
	settingsHandlers := settings.NewHandler(settingsSvc, logRing, log)

	// Memory subsystem — optional but built unconditionally so the
	// MCP server can advertise itself even when no memories exist
	// yet. resolveMemoryService picks the embedder + store from
	// cfg.Memory; the zero-value config gives BM25 + pgvector.
	memorySvc, err := resolveMemoryService(ctx, cfg.Memory, st, log)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("init memory: %w", err)
	}
	// memoryMirror is hoisted out of the block below so RunServices can
	// run the one-time M-U Phase 5 file-memory backfill import against
	// it. Nil when memory is disabled or the mirror wasn't wired.
	var memoryMirror *memory.Mirror
	if memorySvc != nil {
		log.Info("memory ready",
			"embedder", memorySvc.EmbedderName(),
			"dimensions", memorySvc.Dimensions(),
		)

		// Best-effort scan for local embedding services. Each probe
		// has its own tight timeout so the whole sweep stays under
		// a second even when none respond.
		probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		hits := memory.AutoDetect(probeCtx)
		cancel()
		memorySvc.SetAutoDetected(hits)
		for _, h := range hits {
			log.Info("memory auto-detected service",
				"service", h.Detected,
				"base_url", h.BaseURL,
				"models", len(h.Models))
		}

		// Mint a dedicated integration for the memory MCP subprocess
		// to authenticate with, then teach the SessionProvider to
		// auto-inject it into every spawned session's mcp.json.
		key, err := ensureMemoryIntegration(ctx, intgrSvc)
		if err != nil {
			log.Warn("memory MCP auto-attach disabled — could not mint integration key",
				"err", err)
		} else {
			binPath, perr := os.Executable()
			if perr != nil {
				log.Warn("memory MCP auto-attach disabled — os.Executable failed",
					"err", perr)
			} else {
				sessionProvider.WithMemoryAutoAttach(catalog.MemoryAutoAttach{
					Enabled:    true,
					BinaryPath: binPath,
					BaseURL:    listenLoopback(cfg.Listen),
					APIKey:     key,
					Scope:      cfg.Memory.Scope.Default,
				})
				log.Info("memory MCP auto-attach enabled",
					"bin", binPath,
					"base_url", listenLoopback(cfg.Listen))

				// Surface the auto-attached server in the Plugins → MCP
				// registry list so operators can see it exists. Command
				// only — the integration key is injected at spawn time
				// and never exposed through the registry API.
				mcpHandlers.SetBuiltins([]mcpapi.Server{{
					ID:        "opendray-memory",
					Name:      "opendray-memory",
					Transport: "stdio",
					Command:   binPath,
					Args:      []string{"mcp-memory"},
					Enabled:   true,
				}})

				// Wire the local-memory mirror so each session spawn
				// pulls Claude's <cwd>/.claude/projects/.../memory/*.md
				// files into the shared store. Cross-CLI search picks
				// them up automatically.
				mirror := memory.NewMirror(memorySvc, log)
				sessionProvider.WithMemoryMirror(mirror.SyncCwd)
				// Also expose the mirror through the Service so the
				// "Sync now" HTTP endpoint + UI button can trigger an
				// on-demand ingest without waiting for the next spawn.
				memorySvc.SetMirror(mirror)
				memoryMirror = mirror
			}
		}
	}
	memoryHandlers := memory.NewHandlers(memorySvc, log).
		// Cortex Phase 2 — direct /memory/store calls from integration
		// keys route into the tier their memory_policy declares.
		WithPolicyLookup(&capturePolicyAdapter{svc: intgrSvc})

	// Backup subsystem — opt-in. The passphrase resolution chain
	// (env > KEY_FILE > default keyfile, see internal/backup/keyfile.go)
	// is the single source of truth: presence of a passphrase from
	// any source turns the feature on. The legacy [backup] enabled =
	// true gate is kept only as a "you misconfigured something"
	// signal — if it's true but no passphrase is available we hard-
	// fail at startup, which matches the original behaviour.
	//
	// Since PR #50 the feature is hot-armable: LiveBackup owns the
	// Service + scheduler and can be Arm()'d from the /backup-setup
	// HTTP handler without a restart. The path here is just the
	// boot-time arm.
	keyLoad, kerr := backup.LoadPassphrase()
	if kerr != nil {
		st.Close()
		return nil, fmt.Errorf("backup key load: %w", kerr)
	}
	if cfg.Backup.Enabled && keyLoad.Passphrase == "" {
		st.Close()
		return nil, fmt.Errorf("backup: [backup] enabled = true but no passphrase found in OPENDRAY_BACKUP_KEY, $OPENDRAY_BACKUP_KEY_FILE, or %s", keyLoad.Path)
	}
	bcfg := backup.Config{
		Enabled:       true,
		LocalDir:      defaultBackupDir(cfg.Backup.LocalDir, "backups"),
		ExportDir:     defaultBackupDir(cfg.Backup.ExportDir, "exports"),
		PgDumpPath:    cfg.Backup.PgDumpPath,
		PgRestorePath: cfg.Backup.PgRestorePath,
		// full_instance bundles capture the vault + MCP secrets so a
		// restore rebuilds a working instance, not just its DB.
		VaultSources: []backup.VaultSource{
			{Logical: "notes", Dir: notesRoot},
			{Logical: "skills", Dir: skillsRoot},
			{Logical: "mcp", Dir: mcpRoot},
		},
		SecretsFile: secretsFile,
	}
	liveBackup := backup.NewLiveBackup(bcfg, st.Pool(), bus, cfg.Database.URL, cfg.FilePath, log)
	if keyLoad.Passphrase != "" {
		// Boot-time Arm: build the service eagerly so init errors
		// (DB migration failure, etc.) crash boot rather than wait
		// for the first /backups request.
		bsvc, berr := backup.NewService(bcfg, backup.ServiceDeps{
			Pool:       st.Pool(),
			Bus:        bus,
			Passphrase: keyLoad.Passphrase,
			DSN:        cfg.Database.URL,
			ConfigPath: cfg.FilePath,
			Log:        log,
		})
		if berr != nil {
			st.Close()
			return nil, fmt.Errorf("init backup: %w", berr)
		}
		if err := bsvc.Bootstrap(ctx); err != nil {
			st.Close()
			return nil, fmt.Errorf("backup bootstrap: %w", err)
		}
		if err := liveBackup.ArmWithService(ctx, bsvc); err != nil {
			st.Close()
			return nil, fmt.Errorf("arm backup: %w", err)
		}
		log.Info("backup ready",
			"local_dir", bcfg.LocalDir,
			"key_source", string(keyLoad.Source),
			"key_fingerprint", bsvc.CipherFingerprint())
	}
	backupHandlers := backup.NewHandlers(liveBackup)

	// Ambient memory subsystem (Phase A) — summarizer + capture +
	// injector. Always wired (admin endpoints work standalone) but
	// the capture engine only fires when at least one
	// memory_capture_rules row + one enabled summarizer provider
	// exist. Anthropic providers require backup cipher to encrypt
	// api keys; the cipher is now backed by LiveBackup so it Just
	// Works as soon as the operator arms backups via /backup-setup
	// — no restart required for anthropic provider creation either.
	ambientCipher := backup.NewLiveCipher(liveBackup)
	// Encrypt git-host API tokens at rest with the same live cipher
	// (no-op until the operator arms backups; tokens stay plaintext
	// until then, matching the historical trust model).
	gitHostSvc.SetCipher(ambientCipher)
	// Same at-rest encryption for channel config secrets (bot tokens,
	// app secrets, webhook keys).
	channelHub.SetCipher(ambientCipher)
	summarizerStore := summarizer.NewStore(st.Pool(), ambientCipher)
	summarizerRegistry := summarizer.NewRegistry(summarizerStore, log).
		WithIntegrationLookup(&summarizerIntegrationLookup{svc: intgrSvc})
	summarizerHandlers := summarizer.NewHandlers(summarizerRegistry, summarizerStore, log)

	// M25 — pluggable memory worker. Operators pick per-task
	// between the summarizer HTTP path (existing) and a headless
	// agent CLI (`claude --print` / `gemini --print`). All four
	// memory touchpoints (gatekeeper, cleaner, gitactivity,
	// transcript) read their config row from memory_workers.
	memoryWorkerRegistry := memworker.NewRegistry(
		st.Pool(), summarizerRegistry, cliacctSvc, log)
	memoryWorkerHandlers := memworker.NewHandlers(memoryWorkerRegistry, log)

	// M-PA — memory health dashboard. Aggregates "is the memory
	// system actually working?" metrics across both subsystems
	// (layer 5 + projectdoc) for one HTTP read.
	memhealthSvc, err := memhealth.New(st.Pool())
	if err != nil {
		return nil, fmt.Errorf("memhealth init: %w", err)
	}
	memhealthHandlers := memhealth.NewHandlers(memhealthSvc, log)

	// M-PB — cross-layer search composing memory facts + journal
	// + goal/plan. Initialised lazily after projectDocSvc is ready
	// to avoid a forward reference here; see further down.
	var memquerySvc *memquery.Service
	var memqueryHandlers *memquery.Handlers

	// M12 — Gatekeeper. Wired late because the summarizer registry
	// only exists after backup cipher + summarizer store are up.
	// When operators set [memory.gatekeeper] enabled = true, every
	// memory_store call gets a pre-write LLM judgement; otherwise
	// behaviour matches pre-M12 (no extra round-trip).
	if memorySvc != nil && cfg.Memory.Gatekeeper.Enabled {
		gk := memory.NewSummarizerGatekeeper(
			summarizerRegistry,
			cfg.Memory.Gatekeeper.SummarizerID,
			time.Duration(cfg.Memory.Gatekeeper.MaxLatencyMs)*time.Millisecond,
			log,
		)
		memorySvc.SetGatekeeper(gk)
		log.Info("memory gatekeeper enabled",
			"summarizer_id", cfg.Memory.Gatekeeper.SummarizerID,
			"max_latency_ms", cfg.Memory.Gatekeeper.MaxLatencyMs)
	}

	// M13 — Cleaner. Independent of the gatekeeper: even installs
	// that don't pre-judge writes can benefit from periodic review
	// of accumulated noise. We always wire the service when memory
	// + summarizer are up so the HTTP endpoints work; the scheduler
	// only fires when [memory.cleaner] enabled = true.
	var (
		cleanerSvc       *cleaner.Service
		cleanerHandlers  *cleaner.Handlers
		cleanerScheduler *cleaner.Scheduler
	)
	if memorySvc != nil {
		cc := cfg.Memory.Cleaner
		// M25 — cleaner dispatch goes through the memory worker
		// registry (memory_workers.cleaner picks the provider).
		cleanerSvc = cleaner.NewService(
			st.Pool(), memorySvc, memoryWorkerRegistry,
			cleaner.Config{
				BatchSize:            cc.BatchSize,
				MinAge:               time.Duration(cc.MinAgeHours) * time.Hour,
				SkipIfDecidedWithin:  time.Duration(cc.SkipIfDecidedWithinHours) * time.Hour,
				CallTimeout:          time.Duration(cc.CallTimeoutMs) * time.Millisecond,
				GracePeriod:          time.Duration(cc.GraceDays) * 24 * time.Hour,
				LifecycleDormantDays: cc.LifecycleDormantDays,
			},
			log,
		)
		cleanerHandlers = cleaner.NewHandlers(cleanerSvc, log)
		if cc.Enabled {
			cleanerScheduler = cleaner.NewScheduler(cleanerSvc, memorySvc, cleaner.SchedulerConfig{
				Interval:           time.Duration(cc.IntervalSeconds) * time.Second,
				InitialDelay:       time.Duration(cc.InitialDelaySeconds) * time.Second,
				IncludeGlobalScope: cc.IncludeGlobalScope,
			}, log)
			log.Info("memory cleaner scheduler enabled",
				"interval_seconds", cc.IntervalSeconds,
				"initial_delay_seconds", cc.InitialDelaySeconds,
				"include_global_scope", cc.IncludeGlobalScope)
		}
	}

	captureRuleStore := capture.NewRuleStore(st.Pool())
	captureSessionAdapter := &captureSessionAdapter{mgr: sessionMgr}
	captureHistoryAdapter := &captureHistoryAdapter{mgr: sessionMgr}
	captureEngine, ceErr := capture.NewEngine(capture.Deps{
		Rules:    captureRuleStore,
		Registry: summarizerRegistry,
		Memory:   memorySvc,
		Sessions: captureSessionAdapter,
		History:  captureHistoryAdapter,
		CallLog:  summarizerStore,
		Log:      log,
		// M-PE — route the engine's default provider through the
		// worker fabric so operators can switch capture between
		// summarizer-HTTP and Agent (CLI --print) at runtime from
		// /memory/workers. Rules that pin an explicit
		// SummarizerProviderID still win (pre-M-PE behaviour).
		WorkerProvider: memworker.NewSummarizerProvider(
			memoryWorkerRegistry, memworker.TaskCapture),
		// Cortex Phase 2 — integration-created sessions route their
		// facts by the integration's declared memory_policy.
		Policy: &capturePolicyAdapter{svc: intgrSvc},
	})
	if ceErr != nil {
		st.Close()
		return nil, fmt.Errorf("init capture engine: %w", ceErr)
	}
	captureHandlers := capture.NewHandlers(captureRuleStore, captureEngine, log)

	injectorProfileStore := injector.NewProfileStore(st.Pool())
	ambientInjector := injector.New(injectorProfileStore, memorySvc, log)
	sessionProvider.WithAmbientInjector(ambientInjector)
	injectorHandlers := injector.NewHandlers(injectorProfileStore, log)

	// Project docs / proposals / session journal — memory layers 2-4.
	// Composed at spawn time with memory layer 5 inside the catalog
	// adapter; here we just wire HTTP. Mounted under the dual-auth
	// group so the auto-attached opendray-memory MCP can reach it
	// with an integration bearer.
	projectDocSvc := projectdoc.NewService(st.Pool(), log)
	// M-PB — share the memory service's embedder so journal vectors
	// land in the same space as memory facts. Cross-layer search
	// then compares cosines apples-to-apples; otherwise BM25-vs-
	// bge-m3 hits would rank against each other meaninglessly.
	projectDocSvc.WithEmbedder(projectdocEmbedderAdapter{emb: memorySvc.Embedder()})

	// Lifecycle ⇄ memory bridge: archiving a project soft-archives its
	// project-scoped memories (they appear in the Archived view, exempt
	// from the grace purge), and unarchiving restores exactly those rows.
	// Without this the Archived view stayed empty after a project
	// archive — the two features looked broken when they were just
	// unwired.
	if memorySvc != nil {
		projectDocSvc.WithStatusChangeHook(func(ctx context.Context, cwd string, old, new projectdoc.ProjectStatus) {
			switch {
			case new == projectdoc.StatusArchived && old != projectdoc.StatusArchived:
				n, err := memorySvc.ArchiveByScope(ctx, memory.ScopeProject, cwd, memory.ReasonProjectArchived)
				if err != nil {
					log.Warn("project archive: memory archive failed", "cwd", cwd, "err", err)
					return
				}
				log.Info("project archived — memories soft-archived", "cwd", cwd, "count", n)
			case old == projectdoc.StatusArchived && new != projectdoc.StatusArchived:
				n, err := memorySvc.RestoreByScope(ctx, memory.ScopeProject, cwd, memory.ReasonProjectArchived)
				if err != nil {
					log.Warn("project unarchive: memory restore failed", "cwd", cwd, "err", err)
					return
				}
				log.Info("project unarchived — memories restored", "cwd", cwd, "count", n)
			}
		})

		// Backfill: projects archived before the bridge existed never
		// had their memories shelved (the hook only fires on a live
		// transition). Sweep every currently-archived project once at
		// startup and archive its still-active rows. Idempotent —
		// ArchiveByScope only touches archived_at IS NULL — so this is
		// safe to run on every boot; best-effort in the background so
		// it never blocks startup.
		go func() {
			ctx := context.Background()
			cwds, err := projectDocSvc.ListArchivedCwds(ctx)
			if err != nil {
				log.Warn("archive reconcile: list archived projects failed", "err", err)
				return
			}
			for _, cwd := range cwds {
				n, err := memorySvc.ArchiveByScope(ctx, memory.ScopeProject, cwd, memory.ReasonProjectArchived)
				if err != nil {
					log.Warn("archive reconcile failed", "cwd", cwd, "err", err)
					continue
				}
				if n > 0 {
					log.Info("archive reconcile — shelved memories of already-archived project",
						"cwd", cwd, "count", n)
				}
			}
		}()
	}

	// M-PB — now that projectDocSvc exists, build the cross-layer
	// search service. memquery.New is strict about nil deps so we
	// surface a misconfiguration at boot rather than at first hit.
	memquerySvc, err = memquery.New(memorySvc, projectDocSvc, st.Pool())
	if err != nil {
		return nil, fmt.Errorf("memquery init: %w", err)
	}
	memqueryHandlers = memquery.NewHandlers(memquerySvc, log)

	// M-PC — cross-layer conflict detector. Daily scheduler runs
	// across every project_docs cwd and asks the configured worker
	// LLM for contradictions; new findings land in
	// memory_conflicts pending the operator's verdict.
	conflictSvc, err := memconflict.New(st.Pool(), memorySvc, projectDocSvc, memoryWorkerRegistry, log)
	if err != nil {
		return nil, fmt.Errorf("memconflict init: %w", err)
	}
	conflictHandlers := memconflict.NewHandlers(conflictSvc, log)
	conflictScheduler := memconflict.NewScheduler(
		conflictSvc,
		memconflict.NewSQLCwdLister(st.Pool()),
		memconflict.SchedulerConfig{},
	)
	projectDocHandlers := projectdoc.NewHandlers(projectDocSvc, log)
	// M-KG Phase 0 — structured knowledge graph (arc: knowledge-graph-redesign).
	// OFF by default. Decoupled: reads memory, never the reverse; disabling
	// returns exact memory-only (M-U) behaviour.
	var knowledgeHandlers *knowledge.Handlers
	var knowledgeAnchorer *knowledge.Anchorer
	var knowledgeCompiler *knowledge.ExperienceCompiler
	var knowledgeSvc *knowledge.Service
	var knowledgeKBDrafter *knowledge.KBDrafter
	var knowledgeConsolidate *knowledge.ConsolidationEngine
	if cfg.Knowledge.Enabled {
		knowledgeSvc = knowledge.NewService(st.Pool(), log)
		knowledgeSvc.WithSkillSink(knowledgeSkillSink{dir: skillsRoot}) // Phase 4 — render promoted skills
		// Compiled skills (experience compiler) also land as click-runnable
		// custom tasks pointing at the rendered run.sh.
		knowledgeSvc.WithTaskSink(knowledgeTaskSink{tasks: customTaskSvc, skillsRoot: skillsRoot})
		// Skill promotion drafts a full structured SKILL.md via the
		// curation worker (overview / when-to-use / procedure /
		// pitfalls / verification) instead of copying the playbook
		// body verbatim.
		knowledgeSvc.WithSkillifyLLM(knowledgeLLM{
			reg: memoryWorkerRegistry, maxTokens: 4000, timeout: 180 * time.Second,
			task: memworker.TaskCuration,
		})
		sessionProvider.WithKnowledgeInjector(knowledgeSvc) // feed the brain into every spawn
		if memorySvc != nil {
			// Phase 1 — the anchorer reads episodic memory and lifts facts
			// into the graph. Needs memory; without it we still serve CRUD.
			knowledgeSvc.WithEmbedder(memorySvc.Embedder()) // Phase 6 — reuse memory's embedder
			kgLLM := knowledgeLLM{reg: memoryWorkerRegistry}
			// Playbooks + KB pages need bigger output and more time than the
			// small, frequent entity-extraction calls (which keep 512 tok/20s).
			kbLLM := knowledgeLLM{reg: memoryWorkerRegistry, maxTokens: 4000, timeout: 180 * time.Second}
			knowledgeAnchorer = knowledge.NewAnchorer(st.Pool(), knowledgeMemorySource{mem: memorySvc}, log).
				WithLLM(kgLLM).
				WithLifecycle(knowledgeLifecycle{pd: projectDocSvc}) // Cortex P2 — frozen projects stop feeding the graph
			// The experience compiler — outcome-driven distillation. It mines
			// session episodes ACROSS projects, clusters them by embedding
			// similarity, and only drafts a candidate when the same procedure
			// SUCCEEDED in ≥2 sessions (repetition + success evidence; never a
			// single session). Candidates are ranked by recurrence × manual
			// time cost so what saves the most operator time distills first.
			knowledgeCompiler = knowledge.NewExperienceCompiler(st.Pool(), kbLLM, log).
				WithEpisodes(knowledgeEpisodeSource{pool: st.Pool()}).
				WithEmbedder(memorySvc.Embedder()).
				WithLifecycle(knowledgeLifecycle{pd: projectDocSvc}) // P-D — frozen projects leave the feedstock
			knowledgeSvc.WithReanchor(func(c context.Context) error {
				return knowledgeAnchorer.AnchorAll(c, 500)
			})
			// Knowledge — the global cross-project KB pages (Experience Flywheel:
			// per-project docs live in Notes, so there is no handbook here).
			knowledgeKBDrafter = knowledge.NewKBDrafter(
				knowledge.NewStore(st.Pool()), kbLLM,
				knowledgeDocSink{pd: projectDocSvc}, log).
				WithMemory(knowledgeMemorySource{mem: memorySvc}).       // P-G — facts from Memory
				WithProposals(knowledgeProposalSink{pd: projectDocSvc}). // B3 — propose updates to locked pages
				WithLifecycle(knowledgeLifecycle{pd: projectDocSvc})     // Cortex P2 — frozen projects leave the feedstock
			knowledgeSvc.WithKBDrafter(knowledgeKBDrafter) // manual /kb/draft endpoint
			// The per-project Overview — the rich official document. Reads the
			// project's own goal/plan/tech + journal + memory and writes a
			// comprehensive Notes doc (kind=overview), lock-aware + propose-on-lock.
			knowledgeOverview := knowledge.NewOverviewDrafter(
				knowledge.NewStore(st.Pool()), kbLLM,
				knowledgeDocSink{pd: projectDocSvc}, log).
				WithMemory(knowledgeMemorySource{mem: memorySvc}).
				WithJournal(knowledgeJournalSource{pd: projectDocSvc}).
				WithProposals(knowledgeProposalSink{pd: projectDocSvc}).
				WithLifecycle(knowledgeLifecycle{pd: projectDocSvc})
			knowledgeSvc.WithOverviewDrafter(knowledgeOverview)
			// P-C — one ordered loop (anchor → compile → KB → overview) replaces
			// the independent sweep goroutines so each stage drafts from the prior
			// stage's fresh output instead of racing on separate timers.
			knowledgeConsolidate = knowledge.NewConsolidationEngine(
				knowledgeAnchorer, knowledgeCompiler, knowledgeKBDrafter,
				knowledgeOverview, log).
				// Hermes-style curator: skill lifecycle sweep
				// (active → stale 30d → auto-disabled 90d).
				WithCurator(knowledgeSvc)
		}
		knowledgeHandlers = knowledge.NewHandlers(knowledgeSvc, log)
		log.Info("knowledge graph (M-KG) enabled", "anchorer", knowledgeAnchorer != nil)
	}
	// Cortex — the unified module governing the Memory → Notes →
	// Knowledge flywheel. A facade only: re-mounts the three layer
	// handler sets under /api/v1/cortex and adds cross-layer endpoints
	// (status aggregation; quarantine/blueprint/conversations follow).
	// Legacy mounts stay for integrations + the not-yet-migrated mobile.
	cortexOpts := []cortex.Option{
		cortex.WithMemoryEnabled(memorySvc != nil),
		cortex.WithKnowledgeEnabled(knowledgeSvc != nil),
	}
	if memorySvc != nil {
		cortexOpts = append(cortexOpts, cortex.WithQuarantineCounter(memorySvc.CountQuarantined))
	}
	cortexSvc := cortex.NewService(projectDocSvc, log, cortexOpts...)
	// Phase 4 — curation conversations: the channel for actively
	// maintaining docs + re-drafting Foundational knowledge with the
	// AI. Lightweight worker LLM turns; escalation spawns a real
	// session in the project cwd (or the gateway's own workspace for
	// global knowledge pages).
	cortexConvStore := cortex.NewConversationStore(st.Pool())
	workspaceCwd, _ := os.Getwd()
	cortexCuration := cortex.NewCurationService(cortexConvStore, projectDocSvc, memoryWorkerRegistry, bus, log).
		WithSessionLauncher(&curationSessionLauncher{mgr: sessionMgr}, workspaceCwd)
	if memquerySvc != nil {
		cortexCuration.WithContextSource(&curationContextAdapter{mq: memquerySvc})
	}
	// Runtime settings (spawn injection mode) — the projectdoc renderer
	// resolves the mode per spawn, so flipping full↔lean needs no restart.
	cortexSettings := cortex.NewSettingsStore(st.Pool())
	projectDocSvc.WithSpawnMode(cortexSettings.SpawnModeSource())
	cortexHandlers := cortex.NewHandlers(cortexSvc, projectDocHandlers, memoryHandlers, knowledgeHandlers, log).
		WithDocs(projectDocSvc).
		WithBlueprintProposer(cortex.NewBlueprintProposer(projectDocSvc, memoryWorkerRegistry)).
		WithCuration(cortexCuration, cortexConvStore).
		WithSettings(cortexSettings)
	if memorySvc != nil {
		// Guarded: assigning a nil *memory.Service to the interface
		// field would dodge the handler's nil check (typed nil).
		cortexHandlers.WithQuarantine(memorySvc)
	}
	// Inject the cross-agent goal+plan+journal banner into every
	// spawned session's system prompt. Composed alongside the
	// memory-layer-5 banner (ambient injector) inside the catalog
	// adapter.
	sessionProvider.WithProjectDocInjector(projectDocSvc)
	// M16 — project scanner. Auto-detects tech stack + key dirs +
	// git head at spawn time so a fresh agent doesn't have to
	// re-index the repo. Stores the result as project_docs.kind=
	// 'tech_stack'; RenderForSpawn includes it in the system-prompt
	// banner. Re-scans on each spawn if the cached doc is older
	// than 6h.
	projectScanSvc := projectscan.NewService(projectDocSvc, log)
	projectScanHandlers := projectscan.NewHandlers(projectScanSvc, log)
	sessionProvider.WithProjectScanner(projectScanSvc, 6*time.Hour)

	// M16c — git activity summariser. Runs `git log --stat` over
	// the last 7 days, sends the parsed commits to an LLM (same
	// provider as the gatekeeper / cleaner), persists the prose
	// summary as project_docs.kind='recent_activity'. The LLM
	// client is built once at startup from the default summariser
	// provider — operators who add or change providers after boot
	// must restart to pick up the new config.
	gitActivityOpts := []gitactivity.ServiceOption{
		gitactivity.WithWindow("7 days ago"),
		gitactivity.WithCommitLimit(50),
	}
	// M25 — gitactivity LLM dispatch goes through the memory
	// worker registry. The registry handles provider selection
	// per-call from memory_workers.gitactivity, so operator
	// changes via the UI take effect on the next 24h tick (or
	// on /api/v1/git-activity/run if the operator forces it).
	gitActivityOpts = append(gitActivityOpts,
		gitactivity.WithLLM(gitactivity.NewClient(memoryWorkerRegistry)))
	gitActivitySvc := gitactivity.NewService(projectDocSvc, log, gitActivityOpts...)
	gitActivityHandlers := gitactivity.NewHandlers(gitActivitySvc, log)
	// Spawn-time async refresh — see catalog.SessionProvider.
	sessionProvider.WithGitActivityRefresher(gitActivitySvc, 12*time.Hour)
	// Background scheduler (24h tick by default).
	gitActivityScheduler := gitactivity.NewScheduler(
		gitActivitySvc, memorySvc,
		gitactivity.SchedulerConfig{
			Interval:     24 * time.Hour,
			InitialDelay: 10 * time.Minute,
			MaxAge:       12 * time.Hour,
		},
		log,
	)
	// PR watcher — polls open PRs' CI checks every ~90s and emits
	// pr.checks_completed when a suite finishes. The channel hub
	// turns that into chat-side notifications. Built without a
	// "start" call here; App.Run kicks it off alongside the
	// channel hub.
	prWatcher := prwatcher.New(
		&prwatcherSessionAdapter{mgr: sessionMgr},
		gitHostSvc,
		bus,
		log,
	)
	// Auto-journal: on every session.ended / session.stopped event
	// the Journaler writes a session_logs row so future sessions see
	// a chronological record of what just happened in this project.
	journaler := projectdoc.NewJournaler(
		projectDocSvc, bus,
		&projectdocSessionLookup{mgr: sessionMgr},
		log,
	)
	// M18 + M25 — transcript summariser routes through the
	// memory worker registry, so operator config in
	// memory_workers picks summarizer vs agent at call time.
	// No upfront provider check needed: the registry handles
	// degraded states (no summarizer configured → returns empty).
	journaler.WithSummariser(newTranscriptSummariser(memoryWorkerRegistry))
	// M-PA — same routing for the plan-drift detector. After each
	// successful session summary the journaler asks the detector
	// whether the project plan needs updating and files a proposal
	// when so.
	journaler.WithPlanDetector(newPlanDriftDetector(memoryWorkerRegistry))
	// Skill usage + outcome tracking — bump use counters for skills the
	// session's transcript referenced AND record whether that session
	// succeeded, so the workbench can propose retiring skills that are
	// never used or that keep getting loaded into failing sessions.
	if cfg.Knowledge.Enabled {
		knowledgeUsageStore := knowledge.NewStore(st.Pool())
		journaler.WithSkillUsage(func(c context.Context, transcript string, success bool) {
			if _, err := knowledgeUsageStore.RecordSkillUsage(c, transcript, success); err != nil {
				log.Debug("skill usage recording failed", "err", err)
			}
		})
	}
	log.Info("transcript-aware journaler enabled (worker-registry routing)",
		"plan_drift_enabled", true)

	gw := gateway.NewServer(gateway.Deps{
		Logger:    log,
		DB:        st,
		Version:   version.Current(),
		StartedAt: time.Now(),
		V1Routes: func(r chi.Router) {
			// Public: only login + bridge adapter WS endpoint
			// (token-authenticated via the register frame) +
			// per-channel webhook routes (feishu/dingtalk/wecom
			// receive events from the platform; channel impls verify
			// the request themselves).
			authHandlers.MountPublic(r)
			bridgeHandlers.Mount(r)
			channelHandlers.MountPublic(r)

			// Admin-only: integration CRUD + reverse proxy.
			r.Group(func(r chi.Router) {
				r.Use(authSvc.Middleware)
				intgrHandlers.MountAdmin(r)
				proxyHandlers.Mount(r)
				fsHandlers.Mount(r)
				gitHandlers.Mount(r)
				gitHandlers.MountWrite(r)
				gitHostHandlers.Mount(r)
				customTaskHandlers.Mount(r)
				searchHandlers.Mount(r)
				notesHandlers.Mount(r)
				skillsHandlers.Mount(r)
				mcpHandlers.Mount(r)
				vaultGitHandlers.Mount(r)
				vaultSyncer.Mount(r)
				auditHandlers.Mount(r)
				intgrCallLogHandlers.Mount(r)
				settingsHandlers.Mount(r)
				newVersionHandlers(selfUpdateStateDir(), bus, log, func() int {
					rows, err := sessionMgr.List(context.Background())
					if err != nil {
						return 0
					}
					live := 0
					for _, s := range rows {
						if !s.State.IsTerminal() {
							live++
						}
					}
					return live
				}).Mount(r)
				// SetupHandlers (status + setup + disable) is always
				// mounted — that's the whole point of PR #49 / #50.
				// Handlers (the data routes) is also always mounted
				// since PR #50; its requireArmed middleware 503s
				// when LiveBackup is disarmed, so the off-state is
				// safe without a nil-handlers branch here.
				backup.NewSetupHandlers(liveBackup, keyLoad.Source).Mount(r)
				backupHandlers.Mount(r)
				summarizerHandlers.Mount(r)
				memoryWorkerHandlers.Mount(r)
				memhealthHandlers.Mount(r)
				memqueryHandlers.Mount(r)
				conflictHandlers.Mount(r)
				captureHandlers.Mount(r)
				injectorHandlers.Mount(r)
				if cleanerHandlers != nil {
					cleanerHandlers.Mount(r)
				}
				projectScanHandlers.Mount(r)
				gitActivityHandlers.Mount(r)
			})

			// Dual-auth (admin OR integration API key): all business
			// endpoints. ADR 0006 §1 + ADR 0009 (events WS extended
			// to admin so the web Activity viewer rides the same
			// admin token). The integration call logger middleware
			// runs after auth so it can attribute requests to the
			// integration principal (admin requests are skipped
			// inside the middleware).
			r.Group(func(r chi.Router) {
				r.Use(integration.CombinedMiddleware(authSvc, intgrSvc))
				r.Use(intgrCallLogger.Middleware)
				authHandlers.MountProtected(r)
				sessionHandlers.Mount(r)
				catalogHandlers.Mount(r)
				cliacctHandlers.Mount(r)
				channelHandlers.Mount(r)
				memoryHandlers.Mount(r)
				projectDocHandlers.Mount(r)
				if knowledgeHandlers != nil {
					knowledgeHandlers.Mount(r)
				}
				cortexHandlers.Mount(r)
				r.Get("/integrations/_events", eventsHandler.Serve)
			})
		},
	})

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           gw.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{
		cfg:                  cfg,
		log:                  log,
		store:                st,
		bus:                  bus,
		sessions:             sessionMgr,
		channels:             channelHub,
		integrations:         intgrSvc,
		healthCheck:          healthCheck,
		audit:                auditSink,
		intgrCallLogger:      intgrCallLogger,
		vaultSync:            vaultSyncer,
		liveBackup:           liveBackup,
		captureEngine:        captureEngine,
		journaler:            journaler,
		memorySvc:            memorySvc,
		memoryMirror:         memoryMirror,
		projectDocSvc:        projectDocSvc,
		cleanerScheduler:     cleanerScheduler,
		gitActivityScheduler: gitActivityScheduler,
		conflictScheduler:    conflictScheduler,
		prWatcher:            prWatcher,
		cliacctWatcher:       cliacctWatcher,
		server:               srv,
		knowledgeAnchorer:    knowledgeAnchorer,
		knowledgeCompiler:    knowledgeCompiler,
		knowledgeSvc:         knowledgeSvc,
		knowledgeKBDrafter:   knowledgeKBDrafter,
		knowledgeConsolidate: knowledgeConsolidate,
	}, nil
}

// (buildTranscriptSummariser was removed in M25 — the transcript
// summariser now routes through the memory worker registry
// directly. See newTranscriptSummariser in transcript_summariser.go.)

// (buildGitActivityClient was removed in M25 — gitactivity now
// routes through the memory worker registry. See
// gitactivity.NewClient(*worker.Registry).)

// Run starts the HTTP server, channel hub, and audit sink, then blocks
// until ctx is cancelled. Graceful shutdown order:
//
//	HTTP server -> session manager -> channel hub -> audit sink -> event bus -> store
func (a *App) Run(ctx context.Context) error {
	a.log.Info("opendray starting",
		"listen", a.cfg.Listen,
		"version", version.Version,
		"commit", version.Commit)

	// W^X preflight: if executable memory can't be mapped, the V8-based
	// CLIs (codex, gemini) will crash with a fatal SetPermissions error on
	// spawn while Claude keeps working. Surface it loudly here instead of
	// leaving operators to decode a V8 stack trace inside a dead session.
	// Common cause after an in-place `opendray update`: the systemd unit
	// still carries MemoryDenyWriteExecute=true (the unit isn't refreshed
	// by a binary update).
	if err := canMapExecutable(); err != nil {
		a.log.Warn("executable memory mapping is blocked — codex/gemini sessions will crash on spawn (Claude is unaffected). "+
			"Likely MemoryDenyWriteExecute=true on the opendray systemd unit, or an exhausted vm.max_map_count. "+
			"Fix: drop MemoryDenyWriteExecute (e.g. a no-mdwx.conf drop-in or re-run the installer), then `systemctl daemon-reload && systemctl restart opendray`; "+
			"if it's already off, raise vm.max_map_count.",
			"err", err)
	}

	if err := a.channels.Start(ctx); err != nil {
		a.log.Error("channel hub start", "err", err)
	}

	healthDone := make(chan struct{})
	go func() {
		a.healthCheck.Run(ctx)
		close(healthDone)
	}()

	auditDone := make(chan struct{})
	go func() {
		a.audit.Run(ctx)
		close(auditDone)
	}()

	vaultSyncDone := make(chan struct{})
	go func() {
		a.vaultSync.Run(ctx)
		close(vaultSyncDone)
	}()

	// Backup scheduler lifecycle is owned by LiveBackup itself
	// (started inside Arm / ArmWithService, stopped by Disarm or
	// by ctx cancellation). No goroutine to wrangle here.

	captureDone := make(chan struct{})
	go func() {
		a.captureEngine.Run(ctx)
		close(captureDone)
	}()

	journalerDone := make(chan struct{})
	go func() {
		a.journaler.Run(ctx)
		close(journalerDone)
	}()

	// P-C — one ordered consolidation loop (anchor → reflect → KB draft)
	// replaces the three independent M-KG/M-KB sweep goroutines, so each stage
	// distils from the prior stage's fresh output instead of racing on its own
	// timer. nil when the [knowledge] feature flag is off (no-op for M-U builds).
	if a.knowledgeConsolidate != nil {
		go a.knowledgeConsolidate.Run(ctx, knowledge.ConsolidateConfig{})
	}
	if a.knowledgeSvc != nil {
		go a.knowledgeSvc.RunEmbedBackfill(ctx, knowledge.EmbedBackfillConfig{})
	}

	// M-PB — backfill missing embeddings on the journal so the new
	// cross-layer project_search hits historical entries, not just
	// rows appended after this release shipped. Skips itself when
	// no embedder is wired (see RunLogEmbedBackfill guard).
	logEmbedBackfillDone := make(chan struct{})
	go func() {
		a.projectDocSvc.RunLogEmbedBackfill(ctx, projectdoc.LogEmbedBackfillConfig{})
		close(logEmbedBackfillDone)
	}()

	// M-U Phase 2 — same catch-up loop for goal/plan docs so historical
	// projects get their goal/plan embedded and join semantic search,
	// not just docs written after this release. Self-skips without an
	// embedder.
	docEmbedBackfillDone := make(chan struct{})
	go func() {
		a.projectDocSvc.RunDocEmbedBackfill(ctx, projectdoc.LogEmbedBackfillConfig{})
		close(docEmbedBackfillDone)
	}()

	// M-U Phase 5 — one-time import of pre-existing Claude file memories
	// into the single DB store, so an upgrading operator lands them
	// immediately rather than only after each project's next spawn. The
	// per-spawn mirror (WithMemoryMirror) stays wired as the ongoing
	// capture net, so file memory keeps flowing into the DB during the
	// transition; this just front-loads what's already on disk.
	// Idempotent (SyncCwd dedupes), so it self-skips once caught up.
	mirrorBackfillDone := make(chan struct{})
	go func() {
		defer close(mirrorBackfillDone)
		if a.memoryMirror == nil {
			return
		}
		if _, _, err := a.memoryMirror.BackfillAll(ctx); err != nil {
			a.log.Warn("memory: file-memory backfill import failed", "err", err)
		}
	}()

	// M-U Phase 6 — auto-converge re-embed loop. When the operator
	// switches the configured embedder, rows still carrying the old
	// embedder are invisible to search (pgvector partitions similarity
	// by (embedder, dim)). This loop detects the drift and re-embeds in
	// the background until every row is back on the current embedder —
	// no manual "Migrate" click. Self-skips in steady state.
	reembedConvergeDone := make(chan struct{})
	go func() {
		defer close(reembedConvergeDone)
		if a.memorySvc == nil {
			return
		}
		a.memorySvc.RunReembedConverge(ctx, memory.ReembedConvergeConfig{})
	}()

	cleanerDone := make(chan struct{})
	go func() {
		if a.cleanerScheduler != nil {
			a.cleanerScheduler.Run(ctx)
		}
		close(cleanerDone)
	}()

	gitActivityDone := make(chan struct{})
	go func() {
		if a.gitActivityScheduler != nil {
			a.gitActivityScheduler.Run(ctx)
		}
		close(gitActivityDone)
	}()

	// M-PC — daily conflict detector. Same goroutine pattern as
	// the git activity scheduler; nil-safe so disabling the
	// service skips cleanly.
	conflictDone := make(chan struct{})
	go func() {
		if a.conflictScheduler != nil {
			a.conflictScheduler.Run(ctx)
		}
		close(conflictDone)
	}()

	// PR watcher — polls open PRs' CI checks. Start() spawns its
	// own goroutine internally, so there's no done channel to
	// coordinate; the context cancellation drives shutdown.
	if a.prWatcher != nil {
		a.prWatcher.Start(ctx)
	}

	// Claude accounts fsnotify watcher — runs a startup
	// ImportLocal scan (catches accounts added while the gateway was
	// down) then watches AccountsDir for new credentials files.
	// Nil-safe: when [providers.claude] watcher_enabled = false the
	// field is nil and we skip cleanly.
	cliacctWatcherDone := make(chan struct{})
	go func() {
		if a.cliacctWatcher != nil {
			a.cliacctWatcher.Run(ctx)
		}
		close(cliacctWatcherDone)
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("http shutdown", "err", err)
	}
	if err := a.sessions.Shutdown(shutdownCtx); err != nil {
		a.log.Error("session shutdown", "err", err)
	}
	if err := a.channels.Shutdown(shutdownCtx); err != nil {
		a.log.Error("channel shutdown", "err", err)
	}
	// Drain the call log queue after the server stops accepting new
	// requests. Bounded by the queue size + per-row write timeout
	// (5s), so this returns quickly.
	a.intgrCallLogger.Close()

	select {
	case <-healthDone:
	case <-time.After(2 * time.Second):
		a.log.Warn("health checker shutdown timed out")
	}

	select {
	case <-auditDone:
	case <-time.After(5 * time.Second):
		a.log.Warn("audit shutdown timed out")
	}

	select {
	case <-vaultSyncDone:
	case <-time.After(5 * time.Second):
		a.log.Warn("vault auto-sync shutdown timed out")
	}

	// Disarm idempotently stops the backup scheduler if it's
	// running, otherwise no-ops. We do this explicitly rather
	// than relying on ctx cancellation alone so the shutdown
	// path doesn't race with a /backup-setup that arrives just
	// as the server starts to drain.
	a.liveBackup.Disarm()

	select {
	case <-captureDone:
	case <-time.After(5 * time.Second):
		a.log.Warn("capture engine shutdown timed out")
	}
	select {
	case <-journalerDone:
	case <-time.After(2 * time.Second):
		a.log.Warn("journaler shutdown timed out")
	}
	select {
	case <-cleanerDone:
	case <-time.After(2 * time.Second):
		a.log.Warn("cleaner scheduler shutdown timed out")
	}
	select {
	case <-gitActivityDone:
	case <-time.After(2 * time.Second):
		a.log.Warn("git activity scheduler shutdown timed out")
	}

	select {
	case <-cliacctWatcherDone:
	case <-time.After(2 * time.Second):
		a.log.Warn("cliacct watcher shutdown timed out")
	}

	a.bus.Close()
	a.store.Close()
	a.log.Info("opendray stopped")
	return nil
}

func (a *App) Logger() *slog.Logger { return a.log }

// Close releases resources without waiting on the HTTP server. Use Run for
// the normal lifecycle; Close is for failure paths after New succeeded.
func (a *App) Close() {
	if a.sessions != nil {
		_ = a.sessions.Shutdown(context.Background())
	}
	if a.channels != nil {
		_ = a.channels.Shutdown(context.Background())
	}
	if a.bus != nil {
		a.bus.Close()
	}
	if a.store != nil {
		a.store.Close()
	}
}

// parentOf returns the parent directory of an absolute path. Used to
// derive the vault base from <vault>/notes — skills live next to it
// at <vault>/skills, so both share one git-able root.
func parentOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return p
}

// resolveVaultPaths derives the three vault-related paths (notes root,
// skills root, git working tree) from VaultConfig. Each can be set
// explicitly; otherwise we fall back to the legacy layout under the
// shared root. Returns absolute paths suitable for filesystem ops.
//
// The defaults preserve opendray's original `<root>/notes` +
// `<root>/skills` layout. Users coming in with an existing Obsidian
// vault can pin `vault.notes = "~/Documents/MyVault"` (or similar)
// and opendray's notes API will read straight from that directory.
func resolveVaultPaths(c config.VaultConfig) (notes, skills, git string) {
	root := c.Root
	if root == "" {
		root = "~/.opendray/vault"
	}
	root = expandPath(root)

	notes = c.Notes
	if notes == "" {
		notes = filepath.Join(root, "notes")
	} else {
		notes = expandPath(notes)
	}

	skills = c.Skills
	if skills == "" {
		skills = filepath.Join(root, "skills")
	} else {
		skills = expandPath(skills)
	}

	git = c.GitRoot
	if git == "" {
		// Pick the most natural git working tree: if the user pinned a
		// custom notes path, that IS their vault repo (Obsidian-style);
		// otherwise the legacy combined root holds both notes + skills
		// under one repo.
		if c.Notes != "" {
			git = notes
		} else {
			git = root
		}
	} else {
		git = expandPath(git)
	}
	return notes, skills, git
}

// resolveMCPPaths picks the registry root + secrets file with the
// same precedence story as the vault paths above. Defaults:
//
//	root         = <vault root>/mcp        (next to notes/, skills/)
//	secrets_file = ~/.opendray/secrets.env (intentionally OUTSIDE the
//	               vault so a `git add .` in vault never picks it up)
//
// notesRoot / skillsRoot are passed only to derive `<vault root>` —
// we use parentOf(notes) which is the same dir all the other vault
// children live under in the default layout.
func resolveMCPPaths(c config.MCPConfig, notesRoot, skillsRoot string) (root, secrets string) {
	root = c.Root
	if root == "" {
		// Pick the same parent the vault siblings share. notes/ and
		// skills/ are always under <vault root>; using parentOf(notes)
		// works regardless of which the user pinned explicitly.
		base := parentOf(notesRoot)
		if base == "" || base == "/" {
			base = parentOf(skillsRoot)
		}
		if base == "" || base == "/" {
			base = expandPath("~/.opendray/vault")
		}
		root = filepath.Join(base, "mcp")
	} else {
		root = expandPath(root)
	}

	secrets = c.SecretsFile
	if secrets == "" {
		secrets = expandPath("~/.opendray/secrets.env")
	} else {
		secrets = expandPath(secrets)
	}
	return root, secrets
}

// memoryKeyPath is where we cache the plaintext API key for the
// internal opendray-memory integration. mode 0600 + parent dir
// 0700 — same convention as the existing secrets.env file.
const memoryKeyFile = "~/.opendray/memory.key"

// ensureMemoryIntegration guarantees an integration row named
// "opendray-memory" exists and returns a working plaintext API key.
//
// Why we DON'T rotate on every startup: rotating would invalidate
// the api_key already baked into every running session's mcp.json
// (the gateway auto-attaches it at spawn time). Instead we cache
// the plaintext in ~/.opendray/memory.key (mode 0600, same threat
// model as secrets.env) and reuse it across restarts.
//
// The cache and the DB hash can drift in a few edge cases:
//   - Operator deletes the integration row from the UI / SQL
//   - Operator restores PG from a backup that pre-dates the cache
//   - Operator manually rotates via the Integrations UI
//
// All three surface as a 401 next time an agent calls a memory
// tool. Recovery: delete ~/.opendray/memory.key and restart, or
// hit the UI's "Reset opendray-memory" button (planned).
func ensureMemoryIntegration(ctx context.Context, svc *integration.Service) (string, error) {
	const name = "opendray-memory"
	scopes := []string{
		"session:read", // session metadata visibility (future)
		// The auto-attached opendray-memory MCP runs as this key, so it
		// needs the memory read+write scopes — otherwise every agent
		// memory_search/memory_store returns 403 and the cross-agent
		// shared brain never accumulates anything. Writing to the global
		// scope is still admin-only, enforced at the store handler.
		"memory:read",
		"memory:write",
	}

	// 1. Locate (or create) the integration row.
	all, err := svc.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list integrations: %w", err)
	}
	var existing *integration.Integration
	for i := range all {
		if all[i].Name == name {
			existing = &all[i]
			break
		}
	}
	if existing == nil {
		// Brand-new install or migrated DB. Register + cache the key
		// — no rotate needed because Register itself returns a fresh
		// plaintext.
		res, err := svc.Register(ctx, integration.RegisterRequest{
			Name:     name,
			Scopes:   scopes,
			Version:  "internal",
			IsSystem: true,
		})
		if err != nil {
			return "", fmt.Errorf("register %s: %w", name, err)
		}
		_ = writeMemoryKey(res.APIKey)
		return res.APIKey, nil
	}
	id := existing.ID

	// Reconcile scopes. Installs that registered this key before
	// memory:read/write were required are stuck on the old scope set,
	// so every agent memory_search/memory_store 403s. Patch the row up
	// to the desired scopes if any are missing. Cheap no-op once aligned.
	if !scopesCover(existing.Scopes, scopes) {
		if _, err := svc.Update(ctx, id, integration.UpdatePatch{Scopes: &scopes}); err != nil {
			return "", fmt.Errorf("update %s scopes: %w", name, err)
		}
	}

	// 2. Row exists. Reuse cache if present — the ONE thing we know
	//    is the row's bcrypt hash hasn't changed since last write
	//    (we never rotate from this code path), so the cached
	//    plaintext is valid by construction unless the operator did
	//    something to the row out of band.
	if cached, ok := readMemoryKey(); ok {
		return cached, nil
	}

	// 3. Cache missing (first run after upgrade, or operator nuked it).
	//    We can't recover the previous plaintext, so rotate once to
	//    get a fresh one, then cache it.
	res, err := svc.RotateKey(ctx, id)
	if err != nil {
		return "", fmt.Errorf("rotate %s: %w", name, err)
	}
	_ = writeMemoryKey(res.APIKey)
	return res.APIKey, nil
}

// scopesCover reports whether every scope in want is granted by have
// (honouring integration.HasScope wildcard semantics).
func scopesCover(have, want []string) bool {
	for _, w := range want {
		if !integration.HasScope(have, w) {
			return false
		}
	}
	return true
}

// readMemoryKey loads the cached plaintext key from
// ~/.opendray/memory.key. Returns (key, true) on success; missing
// or unreadable file → ("", false).
func readMemoryKey() (string, bool) {
	body, err := os.ReadFile(expandPath(memoryKeyFile))
	if err != nil {
		return "", false
	}
	key := strings.TrimSpace(string(body))
	if key == "" {
		return "", false
	}
	return key, true
}

// writeMemoryKey persists the plaintext key with mode 0600 inside
// ~/.opendray/. Errors are non-fatal — a write failure means the
// next startup will rotate again, which the operator notices via
// existing mcp.json suddenly returning 401.
func writeMemoryKey(key string) error {
	path := expandPath(memoryKeyFile)
	if err := os.MkdirAll(filepathDir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(key+"\n"), 0o600)
}

// filepathDir returns the parent directory of p, tolerating empty
// input by returning ".". Same job as filepath.Dir but kept here
// to avoid pulling the import into a non-fs hot path.
func filepathDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "."
}

// listenLoopback turns the gateway's bind address ("0.0.0.0:8770",
// "[::]:8770", ":8770", "127.0.0.1:8770") into a loopback URL the
// MCP subprocess can dial reliably regardless of NIC binding.
func listenLoopback(listen string) string {
	host, port, ok := strings.Cut(listen, ":")
	if !ok {
		// e.g. ":8770" → SplitN once on the first colon
		port = strings.TrimPrefix(listen, ":")
		host = ""
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	return "http://" + host + ":" + port
}

// resolveMemoryService translates [memory] into a live
// memory.Service. Returns (nil, nil) when memory is explicitly
// disabled; an error when the chosen backend can't be initialised
// (caller treats as fatal — the operator picked something they
// didn't actually have).
//
// Choice matrix:
//
//	backend = ""   | "auto"          → tiered: configured dense if reachable, else BM25 (upgrade-only)
//	backend = "bm25"                 → BM25 keyword floor
//	backend = "http"                 → HTTP dense embedder ([memory.http])
//	backend = "local"                → cgo ONNX embedder ([memory.local], -tags local_onnx)
//	store: pgvector only — the single supported vector store
func resolveMemoryService(
	ctx context.Context,
	cfg config.MemoryConfig,
	st *store.Store,
	log *slog.Logger,
) (*memory.Service, error) {
	storeKind := strings.ToLower(strings.TrimSpace(cfg.Store))
	if storeKind == "" {
		storeKind = "pgvector"
	}
	var memStore memory.Store
	var err error
	switch storeKind {
	case "pgvector":
		memStore, err = memory.OpenPgvectorStore(ctx, st.Pool())
		if err != nil {
			return nil, fmt.Errorf("open pgvector store: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown memory.store=%q (valid: pgvector)", storeKind)
	}

	// Resolve the embedder AFTER the store is open: backend="auto" is
	// store-aware — it inspects existing rows so it never silently drops
	// below the tier the data already uses (M-U Phase 7 §11.2).
	emb, err := resolveAutoEmbedder(ctx, cfg, memStore, log)
	if err != nil {
		return nil, err
	}

	opts := memory.Options{
		Embedder:            emb,
		Store:               memStore,
		SimilarityThreshold: float32(cfg.SimilarityThreshold),
		DefaultTopK:         cfg.DefaultTopK,
		DedupThreshold:      float32(cfg.DedupThreshold),
		Scope: memory.ScopeDefaults{
			Default: memory.Scope(cfg.Scope.Default),
		},
		Logger: log,
	}
	svc, err := memory.New(opts)
	if err != nil {
		return nil, err
	}
	// Record the configured embedder so EmbedderHealth can report the
	// effective-vs-configured tier and live-probe the dense endpoint for
	// the settings UI (even when "auto" has degraded to the BM25 floor).
	svc.SetEmbedderConfig(cfg.Backend, cfg.HTTP.BaseURL, cfg.HTTP.Model, cfg.HTTP.APIKey)
	return svc, nil
}

// buildEmbedder picks the live Embedder per cfg.Backend.
//
//	"" / "auto"  → BM25 today (will swap to LocalONNX in a future
//	                build that ships the model)
//	"bm25"       → BM25 hash-bucket
//	"http"       → OpenAI-compatible /v1/embeddings client
//	"local"      → LocalONNX (only resolves to a real embedder
//	                when the binary was compiled with
//	                `-tags local_onnx`; otherwise the stub returns
//	                a clear error pointing at setup docs)
func buildEmbedder(cfg config.MemoryConfig) (memory.Embedder, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	if backend == "" || backend == "auto" {
		return memory.NewBM25Embedder(384), nil
	}
	switch backend {
	case "bm25":
		return memory.NewBM25Embedder(384), nil
	case "http":
		return memory.NewOpenAICompatibleEmbedder(memory.HTTPEmbedderConfig{
			BaseURL:    cfg.HTTP.BaseURL,
			Model:      cfg.HTTP.Model,
			APIKey:     cfg.HTTP.APIKey,
			Dimensions: cfg.HTTP.Dimensions,
		})
	case "local":
		return memory.NewLocalONNXEmbedder(memory.LocalONNXConfig{
			LibraryPath:   expandPath(cfg.Local.LibraryPath),
			ModelPath:     expandPath(cfg.Local.ModelPath),
			TokenizerPath: expandPath(cfg.Local.TokenizerPath),
			MaxSeqLen:     cfg.Local.MaxSeqLen,
		})
	}
	return nil, fmt.Errorf("unknown memory.backend=%q (valid: auto, bm25, http, local)", cfg.Backend)
}

// embedderDecision is the outcome of the backend="auto" tiering table.
type embedderDecision int

const (
	// decideDenseHTTP — use the configured [memory.http] dense embedder.
	decideDenseHTTP embedderDecision = iota
	// decideBM25Floor — use the pure-Go BM25 keyword floor.
	decideBM25Floor
	// decideFailClosed — the store holds dense rows but no dense endpoint
	// is configured; refuse to start rather than silently churn to BM25.
	decideFailClosed
)

// decideAutoEmbedder is the pure decision table for backend="auto"
// (availability-tiered, upgrade-only — M-U Phase 7 §11.2). It is kept free
// of I/O so it can be table-tested exhaustively; resolveAutoEmbedder
// gathers the live inputs (endpoint probe + per-embedder row counts) and
// acts on the verdict.
//
//	httpConfigured | httpReachable | intentDense | verdict
//	---------------+---------------+-------------+-------------------------------
//	     true      |     true      |      *      | dense  (steady state / upgrade)
//	     true      |     false     |     true    | dense  (degrade-keep-dense, no churn)
//	     true      |     false     |     false   | bm25   (floor; upgrades on a later boot)
//	     false     |      *        |     true    | fail-closed (can't reconstruct dense)
//	     false     |      *        |     false   | bm25   (fresh install, no model)
func decideAutoEmbedder(httpConfigured, httpReachable, intentDense bool) embedderDecision {
	if httpConfigured {
		if httpReachable || intentDense {
			return decideDenseHTTP
		}
		return decideBM25Floor
	}
	if intentDense {
		return decideFailClosed
	}
	return decideBM25Floor
}

// denseIntent inspects the per-embedder row counts and reports whether the
// store already holds rows from a dense (non-BM25) embedder, and which one
// has the most rows. That embedder is the "intended" tier: backend="auto"
// must never silently drop below it, because doing so would hide those
// rows behind the WHERE embedder=$active predicate and trigger a lossy
// re-embed (M-U Phase 7 §11.2).
func denseIntent(counts map[string]int) (name string, dense bool) {
	bestN := 0
	for n, c := range counts {
		if c <= 0 || strings.HasPrefix(strings.ToLower(n), "bm25") {
			continue
		}
		if c > bestN {
			name, bestN = n, c
		}
	}
	return name, name != ""
}

// resolveAutoEmbedder resolves the live Embedder, applying the smart
// backend="auto" tiering (M-U Phase 7 §11.2). Any explicit backend
// (bm25 / http / local) defers to buildEmbedder unchanged.
func resolveAutoEmbedder(ctx context.Context, cfg config.MemoryConfig, st memory.Store, log *slog.Logger) (memory.Embedder, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	if backend != "" && backend != "auto" {
		return buildEmbedder(cfg)
	}

	httpConfigured := strings.TrimSpace(cfg.HTTP.BaseURL) != "" && strings.TrimSpace(cfg.HTTP.Model) != ""

	// Derive the intended tier from existing rows. Best-effort: a count
	// error is treated as "no dense rows" so a transient DB hiccup cannot
	// flip an install into fail-closed.
	intentName, intentDense := "", false
	if counts, cErr := st.CountByEmbedder(ctx); cErr == nil {
		intentName, intentDense = denseIntent(counts)
	} else {
		log.Warn("memory: could not read embedder row counts; assuming no dense rows", "err", cErr)
	}

	// Probe the configured dense endpoint (bounded so a hung endpoint
	// can't stall startup).
	httpReachable := false
	if httpConfigured {
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		httpReachable = memory.ProbeEndpoint(probeCtx, cfg.HTTP.BaseURL, cfg.HTTP.APIKey).Reachable
		cancel()
	}

	switch decideAutoEmbedder(httpConfigured, httpReachable, intentDense) {
	case decideDenseHTTP:
		emb, err := memory.NewOpenAICompatibleEmbedder(memory.HTTPEmbedderConfig{
			BaseURL:    cfg.HTTP.BaseURL,
			Model:      cfg.HTTP.Model,
			APIKey:     cfg.HTTP.APIKey,
			Dimensions: cfg.HTTP.Dimensions,
		})
		if err != nil {
			// base_url+model were non-empty, so this is unexpected; fall
			// back to the floor rather than disabling memory entirely.
			log.Warn("memory: auto could not build the configured dense embedder; using BM25 floor", "err", err)
			return memory.NewBM25Embedder(384), nil
		}
		if httpReachable {
			log.Info("memory: auto selected the configured dense embedder",
				"embedder", emb.Name(), "base_url", cfg.HTTP.BaseURL)
		} else {
			log.Warn("memory: configured dense endpoint unreachable — keeping dense active (degraded). Reads/injection keep working and existing vectors are preserved (no re-embed); new writes and similarity search pause until the endpoint responds.",
				"embedder", emb.Name(), "base_url", cfg.HTTP.BaseURL)
		}
		return emb, nil

	case decideFailClosed:
		return nil, fmt.Errorf("memory: the store holds rows embedded with %q but [memory.http] is not configured; restore the dense endpoint config, or set memory.backend=%q to abandon dense vectors (they will be re-embedded to BM25)", intentName, "bm25")

	default: // decideBM25Floor
		switch {
		case httpConfigured && !httpReachable:
			log.Warn("memory: configured dense endpoint unreachable and the store has no dense rows yet — using the BM25 floor; it will auto-upgrade to dense on a restart once the endpoint responds.",
				"base_url", cfg.HTTP.BaseURL)
		case !httpConfigured:
			log.Info("memory: no dense embedder configured — using the BM25 keyword floor (semantic memory off). Configure [memory.http] to enable dense retrieval.")
		}
		return memory.NewBM25Embedder(384), nil
	}
}

// resolveClaudeHistoryConfig translates the operator's
// [providers.claude] TOML section into a session-package config,
// expanding ~/ in any path. Empty fields stay empty so the
// session package falls back to its built-in HOME defaults.
func resolveClaudeHistoryConfig(c config.ClaudeProviderConfig) session.ClaudeHistoryConfig {
	out := session.ClaudeHistoryConfig{}
	if len(c.HistoryRoots) > 0 {
		out.HistoryRoots = make([]string, 0, len(c.HistoryRoots))
		for _, r := range c.HistoryRoots {
			if r = strings.TrimSpace(r); r != "" {
				out.HistoryRoots = append(out.HistoryRoots, expandPath(r))
			}
		}
	}
	if c.AccountsDir != "" {
		out.AccountsDir = expandPath(c.AccountsDir)
	}
	return out
}

// resolveCodexHistoryConfig expands ~/ in [providers.codex].
func resolveCodexHistoryConfig(c config.CodexProviderConfig) session.CodexHistoryConfig {
	out := session.CodexHistoryConfig{}
	if c.SessionsRoot != "" {
		out.SessionsRoot = expandPath(c.SessionsRoot)
	}
	return out
}

// resolveGeminiHistoryConfig expands ~/ in [providers.gemini].
func resolveGeminiHistoryConfig(c config.GeminiProviderConfig) session.GeminiHistoryConfig {
	out := session.GeminiHistoryConfig{}
	if c.TmpRoot != "" {
		out.TmpRoot = expandPath(c.TmpRoot)
	}
	if c.ProjectsFile != "" {
		out.ProjectsFile = expandPath(c.ProjectsFile)
	}
	return out
}

// resolveAntigravityHistoryConfig expands ~/ in [providers.antigravity].
func resolveAntigravityHistoryConfig(c config.AntigravityProviderConfig) session.AntigravityHistoryConfig {
	out := session.AntigravityHistoryConfig{}
	if c.ConversationsRoot != "" {
		out.ConversationsRoot = expandPath(c.ConversationsRoot)
	}
	return out
}

// integrationDefaultsLookup adapts integration.Service to the session
// package's IntegrationDefaults interface, so a session an integration
// creates inherits that integration's configured provider / model /
// claude account when the POST /sessions request omits them.
type integrationDefaultsLookup struct {
	svc *integration.Service
}

func (l *integrationDefaultsLookup) DefaultsFor(ctx context.Context, integrationID string) (provider, model, claudeAccount string, err error) {
	i, err := l.svc.Get(ctx, integrationID)
	if err != nil {
		return "", "", "", err
	}
	return i.DefaultProviderID, i.DefaultModel, i.DefaultClaudeAccountID, nil
}

// summarizerIntegrationLookup adapts integration.Service to the
// summarizer.IntegrationLookup interface so the summarizer
// registry can resolve integration-kind providers.
type summarizerIntegrationLookup struct {
	svc *integration.Service
}

func (a *summarizerIntegrationLookup) LookupBaseURL(ctx context.Context, id string) (string, bool, error) {
	row, err := a.svc.Get(ctx, id)
	if err != nil {
		// integration.Service.Get returns ErrNotFound for unknown
		// rows; surface that as (_, false, nil) so the registry
		// can give a clear error message.
		if errors.Is(err, integration.ErrNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return row.BaseURL, row.Enabled, nil
}

// curationContextAdapter implements cortex.ContextSource over the
// cross-layer memquery service: top-K facts/journal/doc hits rendered
// as a compact bullet list for the curation prompt.
type curationContextAdapter struct {
	mq *memquery.Service
}

func (a *curationContextAdapter) RelevantContext(ctx context.Context, cwd, query string, topK int) (string, error) {
	hits, err := a.mq.Search(ctx, memquery.SearchRequest{Cwd: cwd, Query: query, TopK: topK})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, h := range hits {
		text := strings.TrimSpace(h.Text)
		if len(text) > 400 {
			text = text[:400] + "…"
		}
		if h.Title != "" {
			fmt.Fprintf(&b, "- [%s] **%s** — %s\n", h.Source, h.Title, text)
		} else {
			fmt.Fprintf(&b, "- [%s] %s\n", h.Source, text)
		}
	}
	return b.String(), nil
}

// curationSessionLauncher implements cortex.SessionLauncher over the
// session manager: spawn a claude session in cwd, give the CLI a
// moment to boot, then type the seed prompt and submit it.
type curationSessionLauncher struct {
	mgr *session.Manager
}

func (l *curationSessionLauncher) Launch(ctx context.Context, cwd, name, seedPrompt string) (string, error) {
	sess, err := l.mgr.Create(ctx, session.CreateRequest{
		Name:       name,
		ProviderID: "claude",
		Cwd:        cwd,
	})
	if err != nil {
		return "", err
	}
	// Seed the prompt once the CLI has had a moment to draw its input
	// box. Detached goroutine: the conversation row already links the
	// session; a failed seed leaves a usable (just unprimed) session.
	go func(sid, prompt string) {
		time.Sleep(4 * time.Second)
		bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := l.mgr.Input(bg, sid, []byte(prompt)); err != nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
		_ = l.mgr.Input(bg, sid, []byte{'\r'})
	}(sess.ID, seedPrompt)
	return sess.ID, nil
}

// capturePolicyAdapter implements capture.PolicyResolver against the
// integration registry. Unknown integrations resolve to "" so the
// runner applies its quarantine default.
type capturePolicyAdapter struct {
	svc *integration.Service
}

func (a *capturePolicyAdapter) MemoryPolicy(ctx context.Context, integrationID string) (string, error) {
	if integrationID == "" {
		return "", nil
	}
	i, err := a.svc.Get(ctx, integrationID)
	if err != nil {
		return "", err
	}
	return string(i.MemoryPolicy), nil
}

// captureSessionAdapter implements capture.SessionLister by
// translating session.Manager.List into capture.SessionInfo.
type captureSessionAdapter struct {
	mgr *session.Manager
}

func (a *captureSessionAdapter) List(ctx context.Context) ([]capture.SessionInfo, error) {
	rows, err := a.mgr.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]capture.SessionInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, capture.SessionInfo{
			ID:            r.ID,
			ProviderID:    r.ProviderID,
			Cwd:           r.Cwd,
			State:         string(r.State),
			Origin:        string(r.Origin),
			IntegrationID: r.IntegrationID,
		})
	}
	return out, nil
}

// prwatcherSessionAdapter satisfies prwatcher.SessionLister by
// projecting session.Manager rows onto the smaller surface the
// watcher needs (id / cwd / state).
type prwatcherSessionAdapter struct {
	mgr *session.Manager
}

func (a *prwatcherSessionAdapter) List(ctx context.Context) ([]prwatcher.SessionInfo, error) {
	rows, err := a.mgr.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]prwatcher.SessionInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, prwatcher.SessionInfo{
			ID:    r.ID,
			Cwd:   r.Cwd,
			State: string(r.State),
		})
	}
	return out, nil
}

// captureHistoryAdapter implements capture.HistoryReader by
// translating session.Manager.History entries.
type captureHistoryAdapter struct {
	mgr *session.Manager
}

func (a *captureHistoryAdapter) History(ctx context.Context, sessionID string, limit int) ([]capture.TranscriptEntry, error) {
	resp, err := a.mgr.History(ctx, sessionID, limit)
	if err != nil {
		return nil, err
	}
	if resp.UnsupportedProvider {
		return nil, nil
	}
	out := make([]capture.TranscriptEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		out = append(out, capture.TranscriptEntry{Ts: e.Ts, Text: e.Text})
	}
	return out, nil
}

// projectdocSessionLookup adapts session.Manager.Get + History to the
// projectdoc.SessionLookup interface the journaler depends on.
// Decoupled from session.Session so projectdoc doesn't have to
// import internal/session (avoids a future cycle when session needs
// to read the journal at startup).
type projectdocSessionLookup struct {
	mgr *session.Manager
}

func (a *projectdocSessionLookup) Get(ctx context.Context, id string) (projectdoc.SessionInfo, error) {
	s, err := a.mgr.Get(ctx, id)
	if err != nil {
		return projectdoc.SessionInfo{}, err
	}
	return projectdoc.SessionInfo{
		ID:         s.ID,
		ProviderID: s.ProviderID,
		Cwd:        s.Cwd,
		StartedAt:  s.StartedAt,
		EndedAt:    s.EndedAt,
		ExitCode:   s.ExitCode,
	}, nil
}

func (a *projectdocSessionLookup) History(ctx context.Context, id string, limit int) ([]projectdoc.HistoryEntry, error) {
	resp, err := a.mgr.History(ctx, id, limit)
	if err != nil {
		return nil, err
	}
	if resp.UnsupportedProvider {
		return nil, nil
	}
	out := make([]projectdoc.HistoryEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		out = append(out, projectdoc.HistoryEntry{Ts: e.Ts, Text: e.Text})
	}
	return out, nil
}

// TranscriptText (M18) returns the full conversation transcript
// for the session — Claude / Codex / Gemini each have their own
// JSONL reader; Manager.TranscriptText dispatches by provider.
// Returns "" for providers we haven't taught yet rather than an
// error so the journaler falls back to metadata-only.
func (a *projectdocSessionLookup) TranscriptText(ctx context.Context, id string, maxBytes int) (string, error) {
	return a.mgr.TranscriptText(ctx, id, maxBytes)
}

// defaultBackupDir returns expandPath(configured) when set, else
// ~/.opendray/<sub>. The backup feature falls back to a per-user
// directory so a fresh dev machine works with zero config beyond
// OPENDRAY_BACKUP_KEY + OPENDRAY_BACKUP_ENABLED.
func defaultBackupDir(configured, sub string) string {
	if v := strings.TrimSpace(configured); v != "" {
		return expandPath(v)
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".opendray", sub)
	}
	return filepath.Join(".", sub)
}

// expandPath resolves ~/ prefixes against the calling user's home
// dir, then makes the path absolute. Mirrors what notes.expand does
// internally — kept here so app-level path resolution doesn't reach
// into the notes package's privates.
func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				p = home
			} else {
				p = filepath.Join(home, p[2:])
			}
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
