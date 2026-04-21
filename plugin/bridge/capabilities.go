// Package bridge implements the capability gate that every plugin bridge call
// passes through before execution. It is pure logic: no I/O, no HTTP handlers,
// no database access — all of that is wired in by later tasks (T10/T11).
//
// # Decoupling rule
//
// This package MUST NOT import github.com/opendray/opendray/kernel/store.
// It depends on two small interfaces (ConsentReader, AuditSink) whose
// implementations are wired in by integration tasks, keeping T5 buildable
// and testable in parallel with T3/T4.
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// ─────────────────────────────────────────────
// Dependency interfaces (defined where used — Go idiom)
// ─────────────────────────────────────────────

// ConsentReader is the minimum surface the gate needs from whatever stores
// granted permissions. Implementations wire store.DB to this in a later task
// (T4/T10 — not T5's concern).
type ConsentReader interface {
	// Load returns the raw granted-permissions JSON for a plugin, or
	// (nil, false, nil) when no consent row exists.
	Load(ctx context.Context, plugin string) (perms []byte, found bool, err error)
}

// AuditSink is the minimum write surface for audit events. Implementations
// wire store.DB.AppendAudit to this in a later task.
type AuditSink interface {
	Append(ctx context.Context, ev AuditEvent) error
}

// AuditEvent is the in-package shape written through AuditSink. A later
// adapter translates this to store.AuditEntry. Kept minimal here so T5
// has no cross-package types.
type AuditEvent struct {
	PluginName string
	Ns         string
	Method     string
	Caps       []string
	Result     string // "ok" | "denied" | "error"
	DurationMs int
	ArgsHash   string
	Message    string
}

// ─────────────────────────────────────────────
// Public API types
// ─────────────────────────────────────────────

// Need describes the specific capability invocation the bridge is about
// to perform. Cap is the top-level key from PermissionsV1 (e.g. "exec",
// "fs.read", "http", "session"). Target is the matcher input — the
// command line / URL / path the call will actually touch.
type Need struct {
	Cap    string
	Target string
}

// PermError is the structured denial result. Code is always "EPERM" for
// capability denials; Msg is a short user-safe message.
type PermError struct {
	Code string
	Msg  string
}

// Error implements the error interface.
func (e *PermError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

// ─────────────────────────────────────────────
// Gate
// ─────────────────────────────────────────────

// Gate is the installed capability checker. Hold one per Runtime instance.
type Gate struct {
	consents ConsentReader
	audit    AuditSink // may be nil; nil means drop rows silently
	log      *slog.Logger
}

// NewGate constructs a Gate. audit may be nil (audit rows are silently
// dropped — only acceptable in tests). A nil log uses slog.Default().
func NewGate(consents ConsentReader, audit AuditSink, log *slog.Logger) *Gate {
	if log == nil {
		log = slog.Default()
	}
	return &Gate{
		consents: consents,
		audit:    audit,
		log:      log,
	}
}

// Check consults the stored consent, matches Need.Target against the
// grant globs, writes an audit row, and returns:
//   - nil on allow
//   - *PermError{Code:"EPERM"} on deny
//   - wrapped error (not PermError) when ConsentReader / AuditSink returns
//     an unexpected error
func (g *Gate) Check(ctx context.Context, plugin string, need Need) error {
	start := time.Now()

	// Load consent row.
	rawPerms, found, loadErr := g.consents.Load(ctx, plugin)
	if loadErr != nil {
		elapsed := int(time.Since(start).Milliseconds())
		g.appendAudit(ctx, AuditEvent{
			PluginName: plugin,
			Ns:         needNs(need.Cap),
			Method:     need.Cap,
			Caps:       []string{need.Cap},
			Result:     "error",
			DurationMs: elapsed,
			Message:    loadErr.Error(),
		})
		return fmt.Errorf("bridge: load consent for %q: %w", plugin, loadErr)
	}

	// No consent row → deny.
	if !found {
		elapsed := int(time.Since(start).Milliseconds())
		msg := fmt.Sprintf("no consent record for plugin %q; install the plugin first", plugin)
		g.appendAudit(ctx, AuditEvent{
			PluginName: plugin,
			Ns:         needNs(need.Cap),
			Method:     need.Cap,
			Caps:       []string{need.Cap},
			Result:     "denied",
			DurationMs: elapsed,
			Message:    msg,
		})
		return &PermError{Code: "EPERM", Msg: msg}
	}

	// Parse the stored permissions JSON.
	var perms permissionsV1Wire
	if len(rawPerms) > 0 {
		if err := json.Unmarshal(rawPerms, &perms); err != nil {
			elapsed := int(time.Since(start).Milliseconds())
			g.appendAudit(ctx, AuditEvent{
				PluginName: plugin,
				Ns:         needNs(need.Cap),
				Method:     need.Cap,
				Caps:       []string{need.Cap},
				Result:     "error",
				DurationMs: elapsed,
				Message:    err.Error(),
			})
			return fmt.Errorf("bridge: parse consent JSON for %q: %w", plugin, err)
		}
	}

	// Evaluate the capability check.
	allowed, denyMsg := evaluate(need, perms)
	elapsed := int(time.Since(start).Milliseconds())

	if allowed {
		g.appendAudit(ctx, AuditEvent{
			PluginName: plugin,
			Ns:         needNs(need.Cap),
			Method:     need.Cap,
			Caps:       []string{need.Cap},
			Result:     "ok",
			DurationMs: elapsed,
		})
		return nil
	}

	g.appendAudit(ctx, AuditEvent{
		PluginName: plugin,
		Ns:         needNs(need.Cap),
		Method:     need.Cap,
		Caps:       []string{need.Cap},
		Result:     "denied",
		DurationMs: elapsed,
		Message:    denyMsg,
	})
	return &PermError{Code: "EPERM", Msg: denyMsg}
}

// appendAudit writes an audit event, silently dropping it when the sink is nil
// or returns an error (audit failures must not block the primary code path).
func (g *Gate) appendAudit(ctx context.Context, ev AuditEvent) {
	if g.audit == nil {
		return
	}
	if err := g.audit.Append(ctx, ev); err != nil {
		g.log.Warn("bridge: audit append failed", "err", err, "plugin", ev.PluginName)
	}
}

// ─────────────────────────────────────────────
// Internal wire type for parsing PermissionsV1 JSON
// ─────────────────────────────────────────────

// permissionsV1Wire is the internal shape for deserialising the
// PermissionsV1 JSON blob stored in plugin_consents.perms_json.
//
// Each capability field uses json.RawMessage so we can handle the
// polymorphic bool | object | array shapes the spec allows:
//   - exec: true | ["git *", "npm *"]
//   - http: true | ["https://api.github.com/*"]
//   - fs:   true | { "read": [...], "write": [...] }
//
// See plugin/manifest.go PermissionsV1 for the canonical type used at install time.
type permissionsV1Wire struct {
	Exec    json.RawMessage `json:"exec,omitempty"`
	HTTP    json.RawMessage `json:"http,omitempty"`
	Fs      json.RawMessage `json:"fs,omitempty"`
	Session string          `json:"session,omitempty"`
	Storage bool            `json:"storage,omitempty"`
	Secret  bool            `json:"secret,omitempty"`
	LLM     bool            `json:"llm,omitempty"`
	// Other caps (clipboard, telegram, git, events) are not matched
	// by the matchers in T5 — they are simple bool/string comparisons
	// handled in the evaluate switch below.
}

// fsPermsWire handles the { "read": [...], "write": [...] } shape.
type fsPermsWire struct {
	All   bool     `json:"-"`
	Read  []string `json:"read,omitempty"`
	Write []string `json:"write,omitempty"`
}

// parseExecGlobs extracts the []string glob list from exec's raw JSON.
// Accepts: null/absent → nil, true → ["*"], false → nil, []string → as-is.
func parseExecGlobs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	// try bool first
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		if b {
			return []string{"*"}
		}
		return nil
	}
	// try []string
	var globs []string
	if json.Unmarshal(raw, &globs) == nil {
		return globs
	}
	return nil
}

// parseHTTPPatterns extracts the []string pattern list from http's raw JSON.
// Accepts: null/absent → nil, true → ["*"], false → nil, []string → as-is.
func parseHTTPPatterns(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		if b {
			return []string{"*"}
		}
		return nil
	}
	var patterns []string
	if json.Unmarshal(raw, &patterns) == nil {
		return patterns
	}
	return nil
}

// parseFSPerm extracts read/write glob lists from fs's raw JSON.
// Accepts: null/absent, true, false, { "read":[], "write":[] }.
func parseFSPerm(raw json.RawMessage) (readGlobs, writeGlobs []string) {
	if len(raw) == 0 {
		return nil, nil
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		if b {
			return []string{"/**"}, []string{"/**"}
		}
		return nil, nil
	}
	var fw fsPermsWire
	if json.Unmarshal(raw, &fw) == nil {
		return fw.Read, fw.Write
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// evaluate dispatches on Cap and runs the appropriate matcher.
// Returns (true, "") on allow; (false, humanMsg) on deny.
// Semantics per 05-capabilities.md.
// ─────────────────────────────────────────────
func evaluate(need Need, perms permissionsV1Wire) (allowed bool, denyMsg string) {
	switch need.Cap {
	case "exec":
		globs := parseExecGlobs(perms.Exec)
		if MatchExecGlobs(globs, need.Target) {
			return true, ""
		}
		return false, fmt.Sprintf("exec not granted for: %s", need.Target)

	case "http":
		patterns := parseHTTPPatterns(perms.HTTP)
		if MatchHTTPURL(patterns, need.Target) {
			return true, ""
		}
		return false, fmt.Sprintf("http not granted for: %s", need.Target)

	case "fs.read":
		readGlobs, _ := parseFSPerm(perms.Fs)
		if MatchFSPath(readGlobs, need.Target) {
			return true, ""
		}
		return false, fmt.Sprintf("fs.read not granted for: %s", need.Target)

	case "fs.write":
		_, writeGlobs := parseFSPerm(perms.Fs)
		if MatchFSPath(writeGlobs, need.Target) {
			return true, ""
		}
		return false, fmt.Sprintf("fs.write not granted for: %s", need.Target)

	case "session":
		// session cap is a simple string: "" | "read" | "write"
		// "write" implies read (05-capabilities.md §session).
		switch need.Target {
		case "read":
			if perms.Session == "read" || perms.Session == "write" {
				return true, ""
			}
		case "write":
			if perms.Session == "write" {
				return true, ""
			}
		}
		return false, fmt.Sprintf("session %q not granted", need.Target)

	case "storage":
		if perms.Storage {
			return true, ""
		}
		return false, "storage not granted"

	case "secret":
		if perms.Secret {
			return true, ""
		}
		return false, "secret not granted"

	case "llm":
		// Platform capability — read-only list of LLM endpoints shared
		// across agents. Granted via `"permissions": {"llm": true}` in
		// the manifest. Secrets never leave the kernel — only endpoint
		// metadata flows to plugins.
		if perms.LLM {
			return true, ""
		}
		return false, "llm not granted"

	default:
		// Conservative: unknown / future capabilities are denied.
		// Per 05-capabilities.md design principle: when in doubt, deny.
		return false, fmt.Sprintf("capability %q not recognised or not granted", need.Cap)
	}
}

// needNs maps a cap key to an audit namespace string (matches 05-capabilities.md audit shape).
func needNs(cap string) string {
	// Cap keys like "fs.read" → "fs"; "exec" → "exec"; "http" → "http"
	if i := strings.IndexByte(cap, '.'); i >= 0 {
		return cap[:i]
	}
	return cap
}

// ─────────────────────────────────────────────
// Pure matchers (stateless)
// ─────────────────────────────────────────────

// MatchExecGlobs returns true if cmdline matches any allow-pattern.
//
// Semantics per 05-capabilities.md §Command patterns:
//   - patterns are space-tokenised
//   - "*" matches everything (any cmdline with at least one non-whitespace token)
//   - "git *" → first token must be exactly "git", any further args allowed
//   - "git log*" → first token is "git" AND second token starts with "log"
//   - "pnpm" (no wildcard) → cmdline must be exactly "pnpm" (no args)
//
// Backslashes, shell metacharacters, and environment expansion are NOT
// interpreted — cmdline is treated as literal bytes split on ASCII spaces.
//
// Importantly: splitting is done with strings.Fields (collapses all whitespace),
// but the spec mandates that a cmdline with leading whitespace is treated
// conservatively as not matching any sane pattern. We enforce this by
// checking whether the raw cmdline has a leading space character — if it
// does, the normalised token sequence would silently suppress the indicator,
// so we deny it immediately.
func MatchExecGlobs(granted []string, cmdline string) bool {
	if len(granted) == 0 {
		return false
	}
	// Empty or blank cmdline — nothing to run.
	if cmdline == "" {
		return false
	}
	// Leading whitespace: conservative deny. The caller is expected to pass
	// the verbatim command string from the plugin bridge call. Leading
	// whitespace is not a valid command prefix and could mask attempts to
	// bypass pattern matching (e.g. " rm -rf /"). Deny unconditionally.
	if cmdline[0] == ' ' || cmdline[0] == '\t' {
		return false
	}

	tokens := strings.Fields(cmdline)
	if len(tokens) == 0 {
		return false
	}

	for _, pattern := range granted {
		if matchExecPattern(pattern, tokens) {
			return true
		}
	}
	return false
}

// matchExecPattern matches a single exec pattern against pre-tokenised cmdline tokens.
func matchExecPattern(pattern string, tokens []string) bool {
	if pattern == "*" {
		// Allow-all: any non-empty cmdline passes.
		return len(tokens) > 0
	}

	patTokens := strings.Fields(pattern)
	if len(patTokens) == 0 {
		return false
	}

	if len(patTokens) == 1 {
		// No wildcard in pattern, or single token pattern.
		p := patTokens[0]
		if !strings.ContainsAny(p, "*?[") {
			// Exact match: cmdline must have exactly one token equal to p.
			return len(tokens) == 1 && tokens[0] == p
		}
		// Single-token pattern with wildcard: match first token only (if only one token).
		match, _ := filepath.Match(p, tokens[0])
		return match && len(tokens) == 1
	}

	// Multi-token pattern: first tokens must match, last pattern token may contain wildcard.
	// e.g. "git *"   → patTokens = ["git", "*"]
	//      "git log*" → patTokens = ["git", "log*"]
	//
	// Rules:
	//  1. All pattern tokens except the last must exactly match the corresponding cmdline token.
	//  2. The last pattern token is matched with filepath.Match against the corresponding cmdline token.
	//  3. If the last pattern token is "*", then the cmdline may have additional tokens beyond the pattern.
	//  4. The cmdline must have at least len(patTokens) tokens for a multi-token pattern to match
	//     (unless the last pattern token is "*" meaning "at least one more arg").

	lastIdx := len(patTokens) - 1
	lastPat := patTokens[lastIdx]

	// Check all fixed prefix tokens.
	for i := 0; i < lastIdx; i++ {
		if i >= len(tokens) || tokens[i] != patTokens[i] {
			return false
		}
	}

	// Now handle the last pattern token.
	if lastPat == "*" {
		// "git *" requires at least one more token after the prefix.
		return len(tokens) > lastIdx
	}

	// "git log*" → tokens[lastIdx] must match "log*", and no extra tokens required.
	if lastIdx >= len(tokens) {
		return false
	}
	match, _ := filepath.Match(lastPat, tokens[lastIdx])
	return match
}

// MatchHTTPURL returns true if rawURL is permitted by the granted patterns.
//
// Rules per 05-capabilities.md §URL patterns and 04-bridge-api.md §http:
//
//  1. Parse URL via net/url; invalid URLs return false.
//  2. RFC1918 (10.*, 172.16–31.*, 192.168.*), loopback (127.*, ::1,
//     localhost), and link-local (169.254.*, fe80::*) are ALWAYS denied,
//     even when granted == ["*"]. This protects against SSRF.
//  3. Scheme must be https unless the matched pattern explicitly starts with
//     "http://" (literal).
//  4. Matching uses filepath.Match-style globbing on host+path. Query strings
//     are stripped before matching.
func MatchHTTPURL(granted []string, rawURL string) bool {
	if len(granted) == 0 {
		return false
	}
	if rawURL == "" {
		return false
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return false
	}

	// Strip brackets from IPv6 host for net.ParseIP; url.Parse keeps them.
	host := u.Hostname() // returns host without port, strips []

	// ── Deny list (SSRF protection) — checked BEFORE the allow list ──
	// Per 04-bridge-api.md §http (Locked): RFC1918 and link-local are denied
	// even if pattern-matched.
	if isPrivateHost(host) {
		return false
	}

	// ── Allow list ──
	// Build the match target: scheme://host+path (no query, no fragment).
	// This matches patterns like "https://api.github.com/*".
	matchTarget := u.Scheme + "://" + u.Host + u.Path

	for _, pattern := range granted {
		if matchHTTPPattern(pattern, u, matchTarget) {
			return true
		}
	}
	return false
}

// matchHTTPPattern matches one URL against one granted pattern.
//
// The pattern has the form:  scheme://hostglob/pathglob
// (e.g. "https://*.example.com/*" or "http://example.com/api/*")
//
// Matching is decomposed into three parts:
//  1. scheme: must match exactly.
//  2. host: matched with path.Match (no "/" in hostnames, so "*" works).
//  3. path: matched with path.Match but with special handling:
//     - a trailing "/*" means "any path with at least one segment"
//     - path.Match is used for specific prefix patterns like "/api/*"
//
// Per 05-capabilities.md: "*" wildcard alone (without scheme prefix) allows
// only https (conservative); http must be explicitly in the pattern.
func matchHTTPPattern(pattern string, u *url.URL, _ string) bool {
	if pattern == "*" {
		// Wildcard-all without scheme: conservative — allow only https.
		return u.Scheme == "https"
	}

	// Decompose the pattern: find "://"
	schemeSep := strings.Index(pattern, "://")
	if schemeSep < 0 {
		// Malformed pattern — conservative deny.
		return false
	}
	patScheme := pattern[:schemeSep]
	rest := pattern[schemeSep+3:] // everything after "://"

	// Scheme must match exactly.
	if patScheme != u.Scheme {
		return false
	}

	// Split pattern rest into host glob and path glob at the first "/".
	var patHostGlob, patPathGlob string
	if idx := strings.Index(rest, "/"); idx >= 0 {
		patHostGlob = rest[:idx]
		patPathGlob = rest[idx:] // includes leading "/"
	} else {
		patHostGlob = rest
		patPathGlob = ""
	}

	// Match host using path.Match (no "/" in hostnames, so "*" matches any segment).
	hostMatched, err := path.Match(patHostGlob, u.Hostname())
	if err != nil || !hostMatched {
		return false
	}

	// Match the URL path against the pattern path glob.
	urlPath := u.Path
	if urlPath == "" {
		urlPath = "/"
	}

	if patPathGlob == "" || patPathGlob == "/" {
		// No path restriction — allow any path.
		return true
	}

	// "/*" means "any path" (one or more path segments).
	if patPathGlob == "/*" {
		return strings.HasPrefix(urlPath, "/")
	}

	// For patterns like "/api/*", use path.Match. path.Match's "*" does not
	// cross "/" boundaries, so "/api/*" matches "/api/foo" but not "/api/foo/bar".
	// This matches the spec's filepath.Match semantics.
	pathMatched, err := path.Match(patPathGlob, urlPath)
	if err != nil {
		return false
	}
	return pathMatched
}

// isPrivateHost returns true if the host literal is an RFC1918, loopback,
// or link-local address. We check the literal string — no DNS resolution.
// This prevents SSRF via URL tricks; if a plugin tries to connect to
// "169.254.169.254" (AWS IMDS), it is denied regardless of grants.
//
// Per 05-capabilities.md §URL patterns: "RFC1918 / link-local are denied
// even if pattern-matched".
func isPrivateHost(host string) bool {
	// localhost (by name)
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Not an IP literal — cannot be private by address; allow name resolution
		// to the network layer (which is outside T5 scope).
		return false
	}

	// Loopback: 127.0.0.0/8 and ::1
	if ip.IsLoopback() {
		return true
	}
	// Link-local unicast: 169.254.0.0/16 and fe80::/10
	if ip.IsLinkLocalUnicast() {
		return true
	}
	// Link-local multicast: 224.0.0.0/24, ff02::/16 etc.
	if ip.IsLinkLocalMulticast() {
		return true
	}

	// RFC1918 private ranges:
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// MatchFSPath returns true if absPath is permitted by the granted patterns.
//
// Rules per 05-capabilities.md §Path patterns:
//   - absPath must be an absolute cleaned path; relative paths return false.
//   - We clean the path with filepath.Clean and re-check absoluteness to
//     defend against "../" traversal tricks (e.g. "/workspace/../etc/passwd").
//   - Globs use filepath.Match.
//   - A pattern ending in "/**" matches any descendant (not the dir itself).
//   - No symlink following — caller's responsibility to resolve before calling.
func MatchFSPath(granted []string, absPath string) bool {
	if len(granted) == 0 {
		return false
	}
	if absPath == "" {
		return false
	}

	// Reject non-absolute paths immediately.
	if !filepath.IsAbs(absPath) {
		return false
	}

	// Clean the path. If it still contains ".." after cleaning it means the
	// path was not absolute to begin with (filepath.Clean on an absolute path
	// always produces an absolute path without ".."), but we double-check
	// absoluteness as a defence-in-depth measure.
	cleaned := filepath.Clean(absPath)
	if !filepath.IsAbs(cleaned) {
		return false
	}
	// If cleaning changed the path (e.g. "/workspace/../etc/passwd" →
	// "/etc/passwd"), we use the cleaned version for matching, but
	// if the cleaned path escapes what the original appeared to be, it
	// should still be matched against the grant list normally.
	// The key defence is: we always match the CLEANED path, never the raw one.
	absPath = cleaned

	for _, pattern := range granted {
		if matchFSPattern(pattern, absPath) {
			return true
		}
	}
	return false
}

// matchFSPattern matches a single FS path pattern against an absolute path.
func matchFSPattern(pattern, absPath string) bool {
	// "/**" suffix means "any descendant but not the directory itself".
	// Transform "/**" into a filepath.Match-compatible glob.
	if strings.HasSuffix(pattern, "/**") {
		// Base dir is everything before the "/**".
		base := pattern[:len(pattern)-3]
		// absPath must start with base+"/" (be a proper descendant).
		if !strings.HasPrefix(absPath, base+"/") {
			return false
		}
		return true
	}

	// Standard filepath.Match for everything else.
	matched, err := filepath.Match(pattern, absPath)
	if err != nil {
		// Invalid glob in grant → conservative deny.
		return false
	}
	return matched
}
