package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray/kernel/store"
	bundled "github.com/opendray/opendray/plugins"
)

// ProviderConfig holds user-customizable settings, keyed by ConfigField.Key.
type ProviderConfig map[string]any

// ProviderInfo is the API response for a provider.
type ProviderInfo struct {
	Provider  Provider       `json:"provider"`
	Config    ProviderConfig `json:"config"`
	Installed bool           `json:"installed"`
	Enabled   bool           `json:"enabled"`
}

// liveProvider tracks a loaded provider's runtime state.
type liveProvider struct {
	provider  Provider
	config    ProviderConfig
	installed bool
	enabled   bool
}

// RuntimeOption is a functional option for configuring a Runtime.
type RuntimeOption func(*Runtime)

// ContribRegistry is the minimal interface the Runtime requires to keep a
// contributions registry in sync. *contributions.Registry satisfies it.
// Defined here to avoid an import cycle (plugin ↛ plugin/contributions).
type ContribRegistry interface {
	Set(pluginName string, c ContributesV1)
	Remove(pluginName string)
}

// WithContributions wires a ContribRegistry into the Runtime.
// When set, Register and Remove automatically sync the registry after each
// successful DB write. Callers that omit this option get zero behaviour change.
func WithContributions(r ContribRegistry) RuntimeOption {
	return func(rt *Runtime) { rt.contributionsReg = r }
}

// SynthesizerFn projects a legacy (pre-v1) Provider into a v1 ContributesV1
// shape in-memory. main.go wires compat.Synthesize here. Keeping this a
// function rather than a direct compat import avoids a cycle
// (plugin/compat already imports plugin).
type SynthesizerFn func(Provider) ContributesV1

// WithSynthesizer wires a legacy→v1 contribution synthesizer into the
// Runtime. When a provider is loaded without a v1 `contributes` block,
// the synthesizer derives one so legacy panel / agent plugins still show
// up in the workbench. Omitting this option reverts to the previous
// behaviour (empty contributions for legacy manifests).
func WithSynthesizer(fn SynthesizerFn) RuntimeOption {
	return func(rt *Runtime) { rt.synthesizer = fn }
}

// ConfigOverlayFn returns the plugin's effective config — the legacy
// ProviderConfig (backed by the plugins.config JSONB column) merged
// with live values from plugin_kv / plugin_secret (where the v1
// Configure form writes). Without an overlay, ResolveCLI only sees
// the legacy cache — values saved via PUT /api/plugins/{name}/config
// never reach session spawn, so booleans like Claude's
// bypassPermissions and Gemini's yolo silently no-op.
//
// The overlay is called once per ResolveCLI invocation; implementations
// should keep it cheap (indexed KV reads, short DB round-trips).
type ConfigOverlayFn func(ctx context.Context, pluginName string, base ProviderConfig) ProviderConfig

// WithConfigOverlay wires a ConfigOverlayFn into the Runtime at
// construction time. Callers that can only build the overlay after
// NewRuntime (because it depends on services built later in the
// startup sequence — e.g. the gateway's secret store) should use
// SetConfigOverlay instead.
func WithConfigOverlay(fn ConfigOverlayFn) RuntimeOption {
	return func(rt *Runtime) { rt.configOverlay = fn }
}

// Runtime manages provider lifecycle and configuration.
type Runtime struct {
	db        *store.DB
	hookBus   *HookBus
	logger    *slog.Logger
	pluginDir string

	mu        sync.RWMutex
	providers map[string]*liveProvider // name → live state

	// contributionsReg is optional. Nil means no contribution tracking —
	// all existing callers that omit WithContributions are unaffected.
	contributionsReg ContribRegistry

	// synthesizer is optional. Nil means legacy manifests get an empty
	// ContributesV1 — matches pre-M2 behaviour. main.go wires
	// compat.Synthesize here so legacy panels/agents surface as
	// synthesized v1 contributions in the workbench.
	synthesizer SynthesizerFn

	// configOverlay is optional. Nil means ResolveCLI only sees the
	// legacy ProviderConfig cache. See ConfigOverlayFn for context.
	configOverlay ConfigOverlayFn
}

// NewRuntime creates a provider runtime. Accepts zero or more RuntimeOption
// values; callers that pass no options get identical behaviour to before
// this parameter was added.
func NewRuntime(db *store.DB, hookBus *HookBus, pluginDir string, logger *slog.Logger, opts ...RuntimeOption) *Runtime {
	if logger == nil {
		logger = slog.Default()
	}
	rt := &Runtime{
		db:        db,
		hookBus:   hookBus,
		logger:    logger,
		pluginDir: pluginDir,
		providers: make(map[string]*liveProvider),
	}
	for _, opt := range opts {
		opt(rt)
	}
	return rt
}

// ── Loading ─────────────────────────────────────────────────────

// LoadAll loads providers from DB first, then seeds new ones from filesystem.
func (rt *Runtime) LoadAll(ctx context.Context) error {
	dbPlugins, err := rt.db.ListPlugins(ctx, false)
	if err != nil {
		return fmt.Errorf("plugin runtime: load from db: %w", err)
	}

	// Load tombstones — names the user has explicitly uninstalled.
	// We skip seeding these even if they reappear in embeddedProviders,
	// so an uninstall survives gateway restarts. A tombstone-read
	// failure is non-fatal: we log and proceed with an empty set,
	// preserving the pre-M5 re-seed behaviour rather than blocking
	// startup on a schema issue.
	tombstoneList, err := rt.db.ListPluginTombstones(ctx)
	if err != nil {
		rt.logger.Warn("plugin tombstone list failed, skipping tombstone check", "error", err)
		tombstoneList = nil
	}
	tombstones := make(map[string]bool, len(tombstoneList))
	for _, n := range tombstoneList {
		tombstones[n] = true
	}

	loaded := make(map[string]bool)
	for _, dbp := range dbPlugins {
		var p Provider
		if err := json.Unmarshal(dbp.Manifest, &p); err != nil {
			rt.logger.Warn("invalid manifest in DB", "plugin", dbp.Name, "error", err)
			continue
		}
		// The in-memory ProviderConfig starts empty and stays that
		// way — live config values come from plugin_kv via gateway's
		// Server.effectiveConfig on every HTTP handler call, so the
		// runtime never needs to cache them. This also means a user
		// saving through Configure sees their new value on the next
		// request, no restart required.
		rt.loadIntoMemory(p, ProviderConfig{}, dbp.Enabled)
		loaded[p.Name] = true
		rt.logger.Info("provider loaded from DB", "name", p.Name, "enabled", dbp.Enabled)
	}

	// Seed new providers and sync manifests for plugins that already exist
	// in the DB — so schema/capabilities edits in code flow into the DB
	// on every restart without clobbering user config or the enabled flag.
	//
	// Two sources, filesystem wins by name:
	//   1. Embedded: bundled with the binary, so release-binary deploys
	//      (LXC, Docker, GitHub Release) ship every core plugin by default.
	//   2. Filesystem `rt.pluginDir`: optional overlay for user-added or
	//      forked plugins — absent on a fresh install, fine.
	providers := mergeProviders(
		embeddedProviders(rt.logger),
		filesystemProviders(rt.logger, rt.pluginDir),
	)
	for _, p := range providers {
		if loaded[p.Name] {
			manifestJSON, err := json.Marshal(p)
			if err != nil {
				rt.logger.Warn("marshal for sync failed", "name", p.Name, "error", err)
				continue
			}
			if err := rt.db.SyncManifest(ctx, p.Name, p.Version, manifestJSON); err != nil {
				rt.logger.Warn("manifest sync failed", "name", p.Name, "error", err)
				continue
			}
			// Refresh the in-memory live provider so the running process
			// sees the new schema without needing a restart round-trip.
			rt.mu.Lock()
			if lp, ok := rt.providers[p.Name]; ok {
				lp.provider = p
			}
			rt.mu.Unlock()
			continue
		}
		if tombstones[p.Name] {
			// User explicitly uninstalled this plugin. Respect that —
			// re-seeding here is exactly the bug the tombstone fixes.
			rt.logger.Info("provider seed skipped (tombstoned)", "name", p.Name)
			continue
		}
		if err := rt.seed(ctx, p); err != nil {
			rt.logger.Warn("seed failed", "name", p.Name, "error", err)
			continue
		}
		rt.logger.Info("provider seeded", "name", p.Name)
	}
	return nil
}

// embeddedProviders returns the manifests bundled inside the binary.
// All built-in plugins live under `plugins/builtin/` (flattened — the
// old agents/panels split at directory level became redundant once
// the bundle-ships-code question was settled at the manifest's `form`
// field). Fails gracefully if the embed can't be walked — logs a
// warning and returns nil so the runtime can still limp along on
// filesystem plugins.
func embeddedProviders(logger *slog.Logger) []Provider {
	ps, err := ScanFS(bundled.FS, "builtin")
	if err != nil {
		logger.Warn("embedded plugin scan failed", "root", "builtin", "error", err)
		return nil
	}
	return ps
}

// filesystemProviders walks the (optional) user-managed plugin dir.
func filesystemProviders(logger *slog.Logger, dir string) []Provider {
	if dir == "" {
		return nil
	}
	ps, err := ScanPluginDir(dir)
	if err != nil {
		logger.Warn("filesystem plugin scan failed", "dir", dir, "error", err)
		return nil
	}
	return ps
}

// mergeProviders combines embedded + filesystem lists with filesystem
// taking precedence by name. Forks / overrides can replace a core
// plugin by dropping the same-named manifest into pluginDir.
func mergeProviders(embedded, overlay []Provider) []Provider {
	byName := make(map[string]Provider, len(embedded)+len(overlay))
	for _, p := range embedded {
		byName[p.Name] = p
	}
	for _, p := range overlay {
		byName[p.Name] = p // overlay wins
	}
	out := make([]Provider, 0, len(byName))
	for _, p := range byName {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (rt *Runtime) seed(ctx context.Context, p Provider) error {
	manifestJSON, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = rt.db.UpsertPlugin(ctx, store.Plugin{
		Name: p.Name, Version: p.Version, Manifest: manifestJSON, Enabled: true,
	})
	if err != nil {
		return err
	}
	rt.loadIntoMemory(p, ProviderConfig{}, true)
	return nil
}

func (rt *Runtime) loadIntoMemory(p Provider, cfg ProviderConfig, enabled bool) {
	installed := detectInstalled(p, cfg)

	rt.mu.Lock()
	rt.providers[p.Name] = &liveProvider{
		provider: p, config: cfg, installed: installed, enabled: enabled,
	}
	rt.mu.Unlock()

	// Push contributions. v1 manifests declare their own block; legacy
	// manifests fall back to the synthesizer (M2 T4) so panels/agents
	// appear in the workbench as synthesized views. When no synthesizer
	// is wired, legacy plugins get an empty ContributesV1 (pre-M2
	// behaviour). Overlay is in-memory; disk is never rewritten.
	if rt.contributionsReg == nil {
		return
	}
	var c ContributesV1
	switch {
	case p.Contributes != nil:
		c = *p.Contributes
	case rt.synthesizer != nil:
		c = rt.synthesizer(p)
	}
	rt.contributionsReg.Set(p.Name, c)
}

func detectInstalled(p Provider, cfg ProviderConfig) bool {
	if p.CLI == nil {
		return true
	}
	cmd := p.CLI.Command
	if v, ok := cfg["command"].(string); ok && v != "" {
		cmd = v
	}
	detectScript := "which " + cmd + " || ls " + cmd + " 2>/dev/null"
	if p.CLI.DetectCmd != "" && (cfg["command"] == nil || cfg["command"] == "") {
		detectScript = p.CLI.DetectCmd
	}
	return exec.Command("sh", "-c", detectScript).Run() == nil
}

// ── Query ───────────────────────────────────────────────────────

// Get returns a provider by name.
func (rt *Runtime) Get(name string) (Provider, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	lp, ok := rt.providers[name]
	if !ok {
		return Provider{}, false
	}
	return lp.provider, true
}

// List returns all providers.
func (rt *Runtime) List() []Provider {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	result := make([]Provider, 0, len(rt.providers))
	for _, lp := range rt.providers {
		result = append(result, lp.provider)
	}
	return result
}

// ListInfo returns all providers with runtime status and config.
// Results are sorted by Provider.Name so clients see a stable order —
// without this, the random map iteration would cause the mobile UI to
// reshuffle cards after every toggle/config update.
func (rt *Runtime) ListInfo() []ProviderInfo {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	result := make([]ProviderInfo, 0, len(rt.providers))
	for _, lp := range rt.providers {
		result = append(result, ProviderInfo{
			Provider: lp.provider, Config: lp.config,
			Installed: lp.installed, Enabled: lp.enabled,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Provider.Name < result[j].Provider.Name
	})
	return result
}

// ── Mutation ────────────────────────────────────────────────────

// ErrRequiredPlugin is returned when a caller tries to disable or
// remove a plugin that has `required:true` in its manifest. The three
// built-ins (claude / terminal / file-browser) rely on this so the
// mobile shell can't be broken by an accidental tap.
var ErrRequiredPlugin = fmt.Errorf("required plugin cannot be modified")

// SetEnabled toggles a provider's enabled state.
//
// Returns ErrRequiredPlugin (wrapped) when the caller tries to disable
// a required plugin — re-enabling is always allowed.
func (rt *Runtime) SetEnabled(ctx context.Context, name string, enabled bool) error {
	rt.mu.Lock()
	lp, ok := rt.providers[name]
	if !ok {
		rt.mu.Unlock()
		return fmt.Errorf("provider %q not found", name)
	}
	if !enabled && lp.provider.Required {
		rt.mu.Unlock()
		return fmt.Errorf("disable %q: %w", name, ErrRequiredPlugin)
	}
	lp.enabled = enabled
	rt.mu.Unlock()
	return rt.db.UpdatePluginEnabled(ctx, name, enabled)
}

// Register adds a new provider at runtime (no filesystem needed).
func (rt *Runtime) Register(ctx context.Context, p Provider) error {
	manifestJSON, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = rt.db.UpsertPlugin(ctx, store.Plugin{
		Name: p.Name, Version: p.Version, Manifest: manifestJSON, Enabled: true,
	})
	if err != nil {
		return err
	}
	rt.loadIntoMemory(p, ProviderConfig{}, true)
	return nil
}

// Remove deletes a provider from runtime and DB.
//
// Returns ErrRequiredPlugin (wrapped) when the caller tries to remove a
// required plugin.
func (rt *Runtime) Remove(ctx context.Context, name string) error {
	rt.mu.Lock()
	lp, ok := rt.providers[name]
	if !ok {
		rt.mu.Unlock()
		return fmt.Errorf("provider %q not found", name)
	}
	if lp.provider.Required {
		rt.mu.Unlock()
		return fmt.Errorf("remove %q: %w", name, ErrRequiredPlugin)
	}
	delete(rt.providers, name)
	rt.mu.Unlock()
	rt.hookBus.Unregister(name)
	if err := rt.db.DeletePlugin(ctx, name); err != nil {
		return err
	}
	if rt.contributionsReg != nil {
		rt.contributionsReg.Remove(name)
	}
	return nil
}

// ── Built-in restore ────────────────────────────────────────────

// BuiltinState labels a bundled plugin's relationship to the current
// runtime: present and running, present but disabled, or wholly
// uninstalled (user-initiated Uninstall + tombstoned so LoadAll
// doesn't re-seed on boot).
type BuiltinState string

const (
	BuiltinInstalled   BuiltinState = "installed"
	BuiltinDisabled    BuiltinState = "disabled"
	BuiltinUninstalled BuiltinState = "uninstalled"
)

// BuiltinInfo is the wire shape returned by ListBuiltins — a bundled
// manifest paired with its current state. Enough for the Settings
// → Built-in Plugins page to render the row + pick the right action
// (none / Enable / Restore).
type BuiltinInfo struct {
	Provider Provider     `json:"provider"`
	State    BuiltinState `json:"state"`
}

// ErrNotBuiltin is returned when RestoreBuiltin gets a name that
// doesn't match any manifest under plugins/builtin/. Callers should
// map this to HTTP 404.
var ErrNotBuiltin = fmt.Errorf("plugin: not a built-in")

// ErrAlreadyInstalled is returned when RestoreBuiltin is called on a
// plugin that already exists in the runtime map — the caller should
// use the regular /toggle endpoint instead. Maps to HTTP 409.
var ErrAlreadyInstalled = fmt.Errorf("plugin: already installed")

// ListBuiltins returns every manifest bundled in the binary's embed.FS
// paired with its current state (installed / disabled / uninstalled).
// Drives the Settings → Built-in Plugins page so users can re-enable
// or restore anything they removed.
//
// State derivation mirrors LoadAll: anything present in rt.providers
// is either installed or disabled depending on its enabled flag;
// anything absent is reported as uninstalled. Tombstone presence is
// NOT consulted — for UI purposes "not in the runtime map" is the
// definition of uninstalled, tombstone or not.
func (rt *Runtime) ListBuiltins() []BuiltinInfo {
	providers := embeddedProviders(rt.logger)
	out := make([]BuiltinInfo, 0, len(providers))
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	for _, p := range providers {
		state := BuiltinUninstalled
		if lp, ok := rt.providers[p.Name]; ok {
			state = BuiltinInstalled
			if !lp.enabled {
				state = BuiltinDisabled
			}
		}
		out = append(out, BuiltinInfo{Provider: p, State: state})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Provider.Name < out[j].Provider.Name
	})
	return out
}

// RestoreBuiltin brings a previously-uninstalled built-in plugin back
// online: clears its tombstone (idempotent) and re-seeds the manifest
// from embed.FS. The caller must route this only from the Settings
// → Built-in Plugins page; user-facing semantics are "undo the
// Uninstall I did earlier".
//
// Returns:
//   - ErrNotBuiltin — the name isn't bundled in embed.FS (404)
//   - ErrAlreadyInstalled — the plugin is already in the runtime;
//     use /toggle instead (409)
//   - generic error — tombstone delete or seed failed (500)
//
// On success the reinstated Provider is returned so the handler can
// surface the canonical manifest (icon, version, contributes block)
// back to the UI without an extra round-trip.
func (rt *Runtime) RestoreBuiltin(ctx context.Context, name string) (Provider, error) {
	var target Provider
	found := false
	for _, p := range embeddedProviders(rt.logger) {
		if p.Name == name {
			target = p
			found = true
			break
		}
	}
	if !found {
		return Provider{}, fmt.Errorf("%w: %s", ErrNotBuiltin, name)
	}

	rt.mu.RLock()
	_, exists := rt.providers[name]
	rt.mu.RUnlock()
	if exists {
		return Provider{}, fmt.Errorf("%w: %s", ErrAlreadyInstalled, name)
	}

	// Tombstone cleared before seed so a crash between the two leaves
	// the system in a consistent "will re-seed on next boot" state
	// rather than "tombstoned + half-seeded".
	if err := rt.db.RemovePluginTombstone(ctx, name); err != nil {
		return Provider{}, fmt.Errorf("remove tombstone: %w", err)
	}

	if err := rt.seed(ctx, target); err != nil {
		return Provider{}, fmt.Errorf("seed: %w", err)
	}

	return target, nil
}

// SetConfigOverlay registers (or clears, when fn is nil) a ConfigOverlayFn
// after Runtime construction. Needed because the gateway-side merge
// helper — which has access to both plugin_kv and plugin_secret — is
// built later in the startup sequence than the Runtime itself. Must be
// called before any session spawns so RecoverOnStartup and the first
// user-driven session both see the overlaid config.
func (rt *Runtime) SetConfigOverlay(fn ConfigOverlayFn) {
	rt.mu.Lock()
	rt.configOverlay = fn
	rt.mu.Unlock()
}

// ── CLI Tool Resolution (used by Hub) ───────────────────────────

// ResolvedCLI is the final CLI specification with config overrides applied.
type ResolvedCLI struct {
	Command string
	Args    []string
	Env     map[string]string
}

// ResolveCLI returns the CLI launch spec for a provider, with config overrides applied.
// Handles: command override, auth type, boolean→cliFlag, select→cliFlag, env var injection.
//
// When a ConfigOverlayFn is wired, the legacy in-memory ProviderConfig is
// merged with live plugin_kv / plugin_secret values before args are built.
// This is how values saved via PUT /api/plugins/{name}/config (Configure
// form) actually reach `claude --dangerously-skip-permissions`, `gemini
// --yolo`, etc. — the legacy cache alone is empty for every v1-form write.
func (rt *Runtime) ResolveCLI(name string) (ResolvedCLI, bool) {
	rt.mu.RLock()
	lp, ok := rt.providers[name]
	overlay := rt.configOverlay
	rt.mu.RUnlock()

	if !ok || lp.provider.CLI == nil || !lp.enabled {
		return ResolvedCLI{}, false
	}

	p := lp.provider
	cfg := lp.config
	if overlay != nil {
		// Short bound so a stuck DB can't hold up session spawn —
		// the user would rather launch with stale config than see
		// the Start button freeze.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cfg = overlay(ctx, name, cfg)
		cancel()
	}

	// 1. Command override + shell auto-detect.
	//
	// Terminal's manifest ships cli.command:"auto" so a fresh user gets
	// a working shell regardless of distro (/bin/zsh vs /usr/bin/zsh,
	// Alpine without zsh, etc.). A user-saved explicit path wins over
	// the manifest default, and if that explicit path has gone missing
	// (zsh uninstalled, Homebrew prefix changed) we fall back to
	// DetectLoginShell rather than let the PTY spawn fail. The warning
	// log surfaces the downgrade so operators can clean up stale config.
	command := p.CLI.Command
	if v, ok := cfg["command"].(string); ok && strings.TrimSpace(v) != "" {
		command = v
	}
	if isAutoCommand(command) {
		if resolved, err := DetectLoginShell(); err == nil {
			command = resolved
		} else if rt.logger != nil {
			rt.logger.Warn("plugin: shell auto-detect failed",
				"plugin", name, "err", err)
		}
	} else if p.Type == ProviderTypeShell {
		// Only shell-type plugins get the "path-missing → auto" rescue.
		// Applying this to CLI providers (claude, codex) would silently
		// launch a shell in place of the missing agent binary and mask
		// real installation problems.
		if _, err := exec.LookPath(command); err != nil {
			if resolved, autoErr := DetectLoginShell(); autoErr == nil {
				if rt.logger != nil {
					rt.logger.Warn("plugin: configured shell not found, falling back to detected shell",
						"plugin", name, "configured", command, "resolved", resolved)
				}
				command = resolved
			}
		}
	}

	// 2. Base args
	args := make([]string, len(p.CLI.DefaultArgs))
	copy(args, p.CLI.DefaultArgs)

	// 3. Process configSchema fields → args and env
	env := make(map[string]string)

	for _, field := range p.ConfigSchema {
		val, hasVal := cfg[field.Key]

		// Auth type: only inject env var when authType = "custom"
		if field.Key == "apiKey" && field.EnvVar != "" {
			authType, _ := cfg["authType"].(string)
			if authType == "custom" {
				if s, ok := val.(string); ok && s != "" {
					env[field.EnvVar] = s
				}
			}
			// "env" = don't set (use system), "oauth" = don't set (tool handles it)
			continue
		}

		// Boolean with cliFlag → append flag when true.
		// The overlay canonicalises every non-secret value to a
		// string ("true"/"false"), while the legacy in-memory cache
		// may deliver a real bool — accept both so either source
		// still triggers the flag.
		if field.Type == "boolean" && field.CLIFlag != "" && hasVal {
			if boolVal(val) {
				args = append(args, field.CLIFlag)
			}
			continue
		}

		// Select with cliFlag → append flag + value
		if field.Type == "select" && field.CLIFlag != "" && hasVal {
			if s, ok := val.(string); ok && s != "" {
				if field.CLIValue {
					args = append(args, field.CLIFlag, s)
				} else {
					args = append(args, field.CLIFlag)
				}
			}
			continue
		}

		// EnvVar mapping (non-auth fields)
		if field.EnvVar != "" && hasVal {
			if s, ok := val.(string); ok && s != "" {
				env[field.EnvVar] = s
			}
		}
	}

	// 4. Extra args (freeform)
	if v, ok := cfg["extraArgs"].(string); ok && v != "" {
		args = append(args, strings.Fields(v)...)
	}

	return ResolvedCLI{Command: command, Args: args, Env: env}, true
}

// boolVal coerces a ProviderConfig entry to a bool. Accepts real
// bools (legacy JSONB cache) and canonical string forms emitted by
// the gateway's kv overlay ("true" / "false" / "1" / "0"). Anything
// else is false — intentionally conservative; unexpected shapes
// shouldn't flip a CLI flag the user didn't ask for.
func boolVal(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "true" || x == "1"
	}
	return false
}

// ── Model Detection ─────────────────────────────────────────────

// DetectModels runs runtime detection for providers with dynamicModels capability.
func (rt *Runtime) DetectModels(name string) ([]ModelDef, error) {
	rt.mu.RLock()
	lp, ok := rt.providers[name]
	rt.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	if !lp.provider.Capabilities.DynamicModels {
		return lp.provider.Capabilities.Models, nil
	}

	cmd := lp.provider.CLI.Command
	if v, ok := lp.config["command"].(string); ok && v != "" {
		cmd = v
	}

	var script string
	switch lp.provider.Name {
	case "lmstudio":
		script = cmd + " ls 2>/dev/null"
	case "ollama":
		script = cmd + " list 2>/dev/null | tail -n +2 | awk '{print $1}'"
	default:
		return lp.provider.Capabilities.Models, nil
	}

	out, err := exec.Command("sh", "-c", script).Output()
	if err != nil {
		return lp.provider.Capabilities.Models, nil
	}

	var models []ModelDef
	if lp.provider.Name == "lmstudio" {
		inLLM := false
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "LLM") {
				inLLM = true
				continue
			}
			if strings.HasPrefix(line, "EMBEDDING") {
				break
			}
			if inLLM && line != "" {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					models = append(models, ModelDef{ID: parts[0], Name: parts[0]})
				}
			}
		}
	} else {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				models = append(models, ModelDef{ID: line, Name: line})
			}
		}
	}

	if len(models) == 0 {
		return lp.provider.Capabilities.Models, nil
	}
	return models, nil
}

// ── Health Check ────────────────────────────────────────────────

func (rt *Runtime) StartHealthCheck(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rt.mu.RLock()
				for name, lp := range rt.providers {
					newInstalled := detectInstalled(lp.provider, lp.config)
					if newInstalled != lp.installed {
						rt.logger.Info("provider install status changed", "name", name, "installed", newInstalled)
					}
				}
				rt.mu.RUnlock()
			}
		}
	}()
}

// HookBus returns the event bus.
func (rt *Runtime) HookBus() *HookBus {
	return rt.hookBus
}

// Proxy is kept for panel-type plugins (future).
func (rt *Runtime) Proxy(name string) (http.Handler, error) {
	return nil, fmt.Errorf("proxy not implemented for provider %q", name)
}
