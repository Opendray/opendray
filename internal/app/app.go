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
	customtask "github.com/opendray/opendray-v2/internal/customtask"
	"github.com/opendray/opendray-v2/internal/eventbus"
	fsapi "github.com/opendray/opendray-v2/internal/fs"
	"github.com/opendray/opendray-v2/internal/gateway"
	gitapi "github.com/opendray/opendray-v2/internal/git"
	"github.com/opendray/opendray-v2/internal/gitactivity"
	githost "github.com/opendray/opendray-v2/internal/githost"
	"github.com/opendray/opendray-v2/internal/integration"
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
	projectDocSvc        *projectdoc.Service // owns the M-PB journal embed backfill loop
	cleanerScheduler     *cleaner.Scheduler  // optional; nil when scheduler is off
	gitActivityScheduler *gitactivity.Scheduler
	conflictScheduler    *memconflict.Scheduler // M-PC daily cross-layer conflict scan
	prWatcher            *prwatcher.Service     // polls open PRs' CI checks and emits pr.checks_completed
	server               *http.Server
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
	catalogHandlers := catalog.NewHandlers(cat, log)

	var cliacctOpts []cliacct.Option
	if d := strings.TrimSpace(cfg.Providers.Claude.AccountsDir); d != "" {
		cliacctOpts = append(cliacctOpts, cliacct.WithAccountsDir(expandPath(d)))
	}
	cliacctSvc := cliacct.NewService(st.Pool(), bus, log, cliacctOpts...)
	cliacctHandlers := cliacct.NewHandlers(cliacctSvc, log)

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
	sessionHandlers := session.NewHandlers(sessionMgr, log)

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
	channelHandlers := channel.NewHandlers(channelHub, log)
	bridgeHandlers := bridge.NewHandlers(bridge.DefaultBroker(), log)

	intgrSvc := integration.NewService(st.Pool(), bus, log)
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
			}
		}
	}
	memoryHandlers := memory.NewHandlers(memorySvc, log)

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
	}
	liveBackup := backup.NewLiveBackup(bcfg, st.Pool(), cfg.Database.URL, cfg.FilePath, log)
	if keyLoad.Passphrase != "" {
		// Boot-time Arm: build the service eagerly so init errors
		// (DB migration failure, etc.) crash boot rather than wait
		// for the first /backups request.
		bsvc, berr := backup.NewService(bcfg, backup.ServiceDeps{
			Pool:       st.Pool(),
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
		// registry; SummarizerID config is now ignored (lives in
		// memory_workers.cleaner.summarizer_id instead).
		cleanerSvc = cleaner.NewService(
			st.Pool(), memorySvc, memoryWorkerRegistry,
			cleaner.Config{
				SummarizerID:        cc.SummarizerID,
				BatchSize:           cc.BatchSize,
				MinAge:              time.Duration(cc.MinAgeHours) * time.Hour,
				SkipIfDecidedWithin: time.Duration(cc.SkipIfDecidedWithinHours) * time.Hour,
				CallTimeout:         time.Duration(cc.CallTimeoutMs) * time.Millisecond,
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
		projectDocSvc:        projectDocSvc,
		cleanerScheduler:     cleanerScheduler,
		gitActivityScheduler: gitActivityScheduler,
		conflictScheduler:    conflictScheduler,
		prWatcher:            prWatcher,
		server:               srv,
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

	// M-PB — backfill missing embeddings on the journal so the new
	// cross-layer project_search hits historical entries, not just
	// rows appended after this release shipped. Skips itself when
	// no embedder is wired (see RunLogEmbedBackfill guard).
	logEmbedBackfillDone := make(chan struct{})
	go func() {
		a.projectDocSvc.RunLogEmbedBackfill(ctx, projectdoc.LogEmbedBackfillConfig{})
		close(logEmbedBackfillDone)
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
	}

	// 1. Locate (or create) the integration row.
	all, err := svc.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list integrations: %w", err)
	}
	var id string
	for _, i := range all {
		if i.Name == name {
			id = i.ID
			break
		}
	}
	if id == "" {
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
//	backend = ""   | "auto"          → BM25 + pgvector store
//	backend = "bm25"                 → BM25 + pgvector store
//	backend = "http"                 → HTTP embedder + pgvector store
//	store   = "pgvector" (default)   → opendray's existing PG with vector ext
//	store   = "chromem"              → not yet implemented in v1
func resolveMemoryService(
	ctx context.Context,
	cfg config.MemoryConfig,
	st *store.Store,
	log *slog.Logger,
) (*memory.Service, error) {
	emb, err := buildEmbedder(cfg)
	if err != nil {
		return nil, err
	}
	storeKind := strings.ToLower(strings.TrimSpace(cfg.Store))
	if storeKind == "" {
		storeKind = "pgvector"
	}
	var memStore memory.Store
	switch storeKind {
	case "pgvector":
		memStore, err = memory.OpenPgvectorStore(ctx, st.Pool())
		if err != nil {
			return nil, fmt.Errorf("open pgvector store: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown memory.store=%q (valid: pgvector)", storeKind)
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
	return memory.New(opts)
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
			ID:         r.ID,
			ProviderID: r.ProviderID,
			Cwd:        r.Cwd,
			State:      string(r.State),
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
