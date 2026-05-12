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
	Prepare    PrepareFunc
}

// PrepareFunc is the spawn-time hook signature.
type PrepareFunc func(ctx context.Context, sessionID, baseDir string) (PrepareOutput, error)

// PrepareOutput carries the bits the manager must merge into the
// exec.Command before pty.Start.
type PrepareOutput struct {
	Args []string
	Env  map[string]string

	// ClaudeSessionID is the agent-side session UUID for providers
	// that accept a `--session-id` flag (claude, gemini). When set,
	// the manager persists it onto the session row so the M18
	// transcript reader can find the right *.jsonl file without
	// fragile mtime-based guessing. Empty for providers that don't
	// support pre-assigned session IDs (e.g. codex).
	ClaudeSessionID string
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
