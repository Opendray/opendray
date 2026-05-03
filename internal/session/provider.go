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
