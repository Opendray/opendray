// Package integration is the registry + reverse proxy + event push
// for external apps that consume opendray.
//
// Per design §8.2 and ADR 0006:
//   - External apps register via admin REST and receive a one-time
//     API key (`odk_live_<b64u>`).
//   - The API key is a parallel valid Principal alongside admin bearer
//     tokens — any business endpoint accepts either via the combined
//     middleware.
//   - The reverse proxy `/api/v1/proxy/{prefix}/*` lets the operator
//     reach a peer app without leaving opendray.
//   - The event WS at `/api/v1/integrations/_events` lets an
//     integration subscribe to bus topics over its own API key.
//
// Per-handler scope enforcement is deferred to v1.1 (see ADR 0006);
// the only enforced scope in M3 is `event:subscribe:<topic>` on the
// event WS.
package integration

import (
	"errors"
	"time"
)

// HealthStatus reflects the most recent /health probe result.
type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// MemoryPolicy declares what the memory capture pipeline does with
// sessions an integration creates (Cortex Phase 2 — quarantine by
// default so third-party temp sessions can't pollute durable memory).
type MemoryPolicy string

const (
	// MemoryPolicyNone — sessions never produce memory.
	MemoryPolicyNone MemoryPolicy = "none"
	// MemoryPolicyQuarantine — facts land in the quarantine tier:
	// excluded from consolidation + spawn injection, reviewable and
	// promotable, auto-expired after a TTL. The default.
	MemoryPolicyQuarantine MemoryPolicy = "quarantine"
	// MemoryPolicyFull — trusted: facts are durable, same as operator
	// sessions.
	MemoryPolicyFull MemoryPolicy = "full"
)

// ValidMemoryPolicy reports whether p is one of the declared policies.
func ValidMemoryPolicy(p MemoryPolicy) bool {
	switch p {
	case MemoryPolicyNone, MemoryPolicyQuarantine, MemoryPolicyFull:
		return true
	}
	return false
}

// Integration is the public view of one registered external app.
// `APIKeyHash` is excluded from JSON; the plaintext key is returned
// once at registration and never again.
type Integration struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	BaseURL        string         `json:"base_url"`
	RoutePrefix    string         `json:"route_prefix"`
	Scopes         []string       `json:"scopes"`
	Version        string         `json:"version,omitempty"`
	Enabled        bool           `json:"enabled"`
	HealthStatus   HealthStatus   `json:"health_status"`
	HealthPayload  map[string]any `json:"health_payload,omitempty"`
	HealthLastSeen *time.Time     `json:"health_last_seen,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	RotatedAt      *time.Time     `json:"rotated_at,omitempty"`

	// MemoryPolicy routes memory capture for sessions this integration
	// creates: none | quarantine (default) | full.
	MemoryPolicy MemoryPolicy `json:"memory_policy"`

	// IsSystem flags rows opendray manages itself (e.g. the
	// auto-registered opendray-memory MCP integration). The UI
	// renders system rows in a separate group and disables
	// delete/rotate so operators don't break their own running
	// sessions by destroying a key the gateway just baked into
	// every spawn's mcp.json.
	IsSystem bool `json:"is_system"`

	apiKeyHash string // bcrypt; never serialised
}

// HealthPing is the parsed body of an integration's /health endpoint
// (per design §11). All fields are optional.
type HealthPing struct {
	Status     string  `json:"status"`
	Version    string  `json:"version,omitempty"`
	BusyRatio  float64 `json:"busy_ratio,omitempty"`
	QueueDepth int     `json:"queue_depth,omitempty"`
}

// Errors surfaced by handlers / services.
var (
	ErrNotFound          = errors.New("integration not found")
	ErrPrefixTaken       = errors.New("route prefix already in use")
	ErrNameTaken         = errors.New("integration name already in use")
	ErrReservedPrefix    = errors.New("route prefix is reserved")
	ErrInvalidAPIKey     = errors.New("invalid api key")
	ErrInsufficientScope = errors.New("insufficient scope")
	// ErrSystemIntegration is returned when an operator tries to
	// delete or rotate the key on a row flagged is_system=true.
	// These rows are owned by opendray itself; mutating them out
	// of band would orphan running sessions whose mcp.json holds
	// the previous key.
	ErrSystemIntegration = errors.New("system integration: not operator-mutable")
)

// reservedPrefixes is the allowlist of prefixes that would collide
// with non-proxy routes if registered.
var reservedPrefixes = map[string]bool{
	"":          true,
	"_events":   true,
	"_kinds":    true,
	"_internal": true,
	"_":         true,
}

func isReservedPrefix(p string) bool { return reservedPrefixes[p] }
