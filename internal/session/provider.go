package session

import "context"

// ProviderInfo is the resolved exec target for a session's provider_id.
// `Prepare`, when non-nil, runs after the manager creates the per-
// session scratch dir and before the PTY is started; it lets the
// provider write per-session config files (e.g. MCP server JSON for
// claude, codex's home-redirected TOML) and contribute extra args /
// env. The session manager owns the scratch dir lifecycle and removes
// it on session.ended.
type ProviderInfo struct {
	ID         string
	Executable string
	Args       []string

	// Conflicts declares provider-specific CLI argument-group rules.
	// When a user spawn arg matches a key flag, every flag listed in
	// the value slice is stripped from Args before exec (along with
	// the value following each flag, when applicable).
	//
	// Use this for CLI parsers that reject "this flag cannot be used
	// with that flag" (clap ArgGroup), where simple name-based dedup
	// is insufficient. E.g. codex's
	// --dangerously-bypass-approvals-and-sandbox is mutually exclusive
	// with --ask-for-approval and -s/--sandbox.
	Conflicts map[string][]string

	Prepare PrepareFunc
}

// PrepareFunc is the spawn-time hook signature.
type PrepareFunc func(ctx context.Context, sessionID, baseDir string) (PrepareOutput, error)

// PrepareOutput carries the bits the manager must merge into the
// exec.Command before pty.Start.
type PrepareOutput struct {
	Args []string
	Env  map[string]string

	// ClaudeSessionID is the agent-side session UUID for providers
	// that accept a `--session-id` flag (claude). When set,
	// the manager persists it onto the session row so the M18
	// transcript reader can find the right *.jsonl file without
	// fragile mtime-based guessing. Empty for providers that don't
	// support pre-assigned session IDs (e.g. codex).
	ClaudeSessionID string

	// Notices are one-time operator hints surfaced at the top of the
	// session terminal (and the ring buffer / transcript) before the
	// CLI's own output — e.g. "the CLI will disable MCP here because the
	// folder is untrusted". Plain text; the manager adds styling.
	Notices []string
}

// ProviderResolver maps a provider_id to its ProviderInfo. The catalog
// subsystem's adapter implements this interface; tests can supply a
// fake.
type ProviderResolver interface {
	Resolve(ctx context.Context, id string) (ProviderInfo, error)
}

// ── Account selection (multi-account providers like claude) ────────

type accountIDCtxKey struct{}

// WithAccountID attaches the spawn-time account selection to ctx so
// the ProviderResolver can look up the right credential without
// adding a parameter to Resolve(). Empty id is a no-op (resolver
// uses the provider's default).
func WithAccountID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, accountIDCtxKey{}, id)
}

// AccountID retrieves the account selection set by WithAccountID, or
// "" if none.
func AccountID(ctx context.Context) string {
	if v, ok := ctx.Value(accountIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// ── Model selection ────────────────────────────────────────────────
//
// A session can pin its own model (Session.Model). Like the account
// selection it isn't a Resolve() parameter, so we thread it through
// context; the ProviderResolver renders it via the manifest's model
// flag, taking precedence over the provider config default. Empty is a
// no-op so the resolver falls back to the configured default.

type modelCtxKey struct{}

// WithModel attaches the session's pinned model to ctx for resolve-time
// use. Empty model is a no-op.
func WithModel(ctx context.Context, model string) context.Context {
	if model == "" {
		return ctx
	}
	return context.WithValue(ctx, modelCtxKey{}, model)
}

// ModelFromContext retrieves the value set by WithModel, or "" if none.
func ModelFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(modelCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// ── Cwd propagation ────────────────────────────────────────────────
//
// Some prepare-time decisions (notably the memory MCP auto-attach)
// need the session's working directory to scope memories correctly.
// The cwd lives on the Session struct but isn't part of the Prepare
// closure signature; we thread it through context to avoid breaking
// every existing PrepareFunc.

type cwdCtxKey struct{}

// WithCwd attaches the session's cwd to ctx for prepare-time use.
// Empty cwd is a no-op.
func WithCwd(ctx context.Context, cwd string) context.Context {
	if cwd == "" {
		return ctx
	}
	return context.WithValue(ctx, cwdCtxKey{}, cwd)
}

// Cwd retrieves the value set by WithCwd, or "" if none.
func Cwd(ctx context.Context) string {
	if v, ok := ctx.Value(cwdCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// ── Origin propagation ─────────────────────────────────────────────
//
// Prepare-time isolation decisions need the session's provenance: a
// third-party integration session is self-managed — it drives its own
// delivery and brings its own tools — so opendray must NOT auto-attach
// the cross-project memory MCP to it (see adapter Resolve). Like cwd,
// origin lives on the Session struct but isn't a Resolve() parameter,
// so we thread it through context.

type originCtxKey struct{}

// WithOrigin attaches the session's origin to ctx for resolve-time use.
// Empty origin is a no-op.
func WithOrigin(ctx context.Context, o Origin) context.Context {
	if o == "" {
		return ctx
	}
	return context.WithValue(ctx, originCtxKey{}, o)
}

// OriginFromContext retrieves the value set by WithOrigin, or "" if none.
func OriginFromContext(ctx context.Context) Origin {
	if v, ok := ctx.Value(originCtxKey{}).(Origin); ok {
		return v
	}
	return ""
}

// sessionIDCtxKey + WithSessionID + SessionIDFromContext mirror the
// cwd plumbing for the session.id, used by ambient-memory rendering
// at spawn time. Defined here (alongside WithCwd) so callers don't
// need a separate import.
type sessionIDCtxKey struct{}

// WithSessionID returns a derived context carrying the session id.
// Empty is a no-op so call sites needn't guard.
func WithSessionID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, sessionIDCtxKey{}, id)
}

// SessionIDFromContext returns the value set by WithSessionID, or
// "" when the key isn't present.
func SessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// resumeClaudeSessionIDCtxKey carries the agent-side session UUID that
// a reactivated (resumed) session should continue, so the provider's
// Prepare emits `--resume <id>` instead of minting a fresh
// `--session-id`. Without this, "resuming" a session spawned a brand
// new agent conversation and orphaned the original transcript.
type resumeClaudeSessionIDCtxKey struct{}

// WithResumeClaudeSessionID returns a derived context carrying the
// agent-side session UUID to resume. Empty is a no-op so call sites
// (fresh spawns) needn't guard.
func WithResumeClaudeSessionID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, resumeClaudeSessionIDCtxKey{}, id)
}

// ResumeClaudeSessionIDFromContext returns the UUID set by
// WithResumeClaudeSessionID, or "" when this is a fresh spawn.
func ResumeClaudeSessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(resumeClaudeSessionIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// carryoverContextCtxKey carries a block of prior-conversation text to
// seed a freshly spawned session's system prompt. Set ONLY by
// SwitchClaudeAccount when the operator opts into "carry context":
// switching accounts can't --resume the old conversation (the UUID
// isn't in the new account's registry), so instead we read the old
// transcript and inject a recap via --append-system-prompt. It's a
// one-shot — present only on the switch respawn, absent on later
// restarts (which --resume the new account's own UUID, whose
// transcript already contains the seeded recap).
type carryoverContextCtxKey struct{}

// WithCarryoverContext returns a derived context carrying the recap
// text. Empty is a no-op so call sites (the common fresh/restart path)
// needn't guard.
func WithCarryoverContext(ctx context.Context, text string) context.Context {
	if text == "" {
		return ctx
	}
	return context.WithValue(ctx, carryoverContextCtxKey{}, text)
}

// CarryoverContextFromContext returns the recap set by
// WithCarryoverContext, or "" when this spawn carries no prior context.
func CarryoverContextFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(carryoverContextCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// ── Integration spawn profile ──────────────────────────────────────
//
// The three keys below carry the provider-AGNOSTIC half of an
// integration's spawn profile (mcp_servers / system_prompt /
// bypass_permissions). The manager looks the profile up after resolving
// the creating integration and threads it through context, so the
// catalog adapter can translate each piece via the existing per-provider
// machinery (renderMCP, the system-prompt surface, the bypass flag)
// without these becoming Resolve()/Prepare() parameters. Each helper is a
// no-op on its zero value so the common (non-integration) spawn path
// needn't guard.

type integrationMCPServersCtxKey struct{}

// WithIntegrationMCPServers attaches the integration's raw mcp_servers
// JSON array string to ctx for resolve-time use. Empty is a no-op.
func WithIntegrationMCPServers(ctx context.Context, rawJSON string) context.Context {
	if rawJSON == "" {
		return ctx
	}
	return context.WithValue(ctx, integrationMCPServersCtxKey{}, rawJSON)
}

// IntegrationMCPServersFromContext returns the raw mcp_servers JSON set
// by WithIntegrationMCPServers, or "" if none.
func IntegrationMCPServersFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(integrationMCPServersCtxKey{}).(string); ok {
		return v
	}
	return ""
}

type integrationSystemPromptCtxKey struct{}

// WithIntegrationSystemPrompt attaches the integration's boot system
// prompt to ctx for prepare-time use. Empty is a no-op.
func WithIntegrationSystemPrompt(ctx context.Context, text string) context.Context {
	if text == "" {
		return ctx
	}
	return context.WithValue(ctx, integrationSystemPromptCtxKey{}, text)
}

// IntegrationSystemPromptFromContext returns the prompt set by
// WithIntegrationSystemPrompt, or "" if none.
func IntegrationSystemPromptFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(integrationSystemPromptCtxKey{}).(string); ok {
		return v
	}
	return ""
}

type permissionModeCtxKey struct{}

// WithPermissionMode attaches the integration's spawn-profile permission
// mode ("default"|"bypass") to ctx for resolve-time use. Empty/"default"
// is a no-op (the provider's normal approval flow stands).
func WithPermissionMode(ctx context.Context, mode string) context.Context {
	if mode == "" || mode == "default" {
		return ctx
	}
	return context.WithValue(ctx, permissionModeCtxKey{}, mode)
}

// PermissionModeFromContext returns the mode set by WithPermissionMode,
// or "" if none. "bypass" → the catalog appends the provider's bypass
// flag.
func PermissionModeFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(permissionModeCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// IntegrationSpawnProfile is the provider-agnostic injection config an
// integration declares once; the manager applies it to every session
// that integration creates.
type IntegrationSpawnProfile struct {
	MCPServersJSON string // raw mcp_servers JSON array
	SystemPrompt   string
	PermissionMode string // "default" | "bypass"
}

// IntegrationSpawnProfiles resolves an integration's spawn profile.
// Optional; nil disables integration spawn-profile injection.
type IntegrationSpawnProfiles interface {
	SpawnProfileFor(ctx context.Context, integrationID string) (IntegrationSpawnProfile, error)
}
