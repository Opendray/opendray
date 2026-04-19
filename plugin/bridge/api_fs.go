// Package bridge — FS namespace (M3 T9, read-path).
//
// # Scope
//
// This file implements the read-path of `opendray.fs.*`: readFile, exists,
// stat, readDir. Write-path (writeFile, mkdir, remove) and watch land in T10.
//
// # Capability gating
//
// Every method resolves path-vars through the injected PathVarResolver,
// cleans the input path with filepath.Clean, and calls Gate.CheckExpanded
// with Need{Cap:"fs.read", Target: cleaned}. After the initial check the
// cleaned path is run through filepath.EvalSymlinks and the gate is
// re-queried on the resolved path — the TOCTOU mitigation called out in
// M3-PLAN §6. If the second check denies, the operation fails with EPERM
// even though the first check passed (the symlink changed the effective
// target to somewhere the plugin is not allowed to read).
//
// # Caps
//
//   - readFile: 10 MiB hard byte cap.
//   - readDir:  4096 entry cap.
package bridge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Hard caps. Exported for tests and M3 operators who want visibility.
const (
	// MaxReadFileBytes is the ceiling for opendray.fs.readFile — 10 MiB.
	// Anything bigger returns EINVAL rather than streaming a partial file;
	// streaming belongs to a future T10 method if required.
	MaxReadFileBytes = 10 * 1024 * 1024
	// MaxReadDirEntries is the ceiling for opendray.fs.readDir — 4096.
	// Directories larger than this return their first 4096 entries and a
	// "truncated" marker in the result (no error). Plugins that need a
	// full listing should paginate via glob patterns in a future method.
	MaxReadDirEntries = 4096
)

// PathVarResolver returns the {workspace, home, dataDir, tmp} path-var
// context for a plugin at call time. Implemented in the gateway (T24);
// lives here so the bridge namespace can depend on the interface without
// importing gateway.
//
// Resolver MUST be safe for concurrent use.
type PathVarResolver interface {
	Resolve(ctx context.Context, plugin string) (PathVarCtx, error)
}

// FSConfig wires FSAPI's dependencies. All three fields are required —
// NewFSAPI will supply defaults only for Log (slog.Default()).
type FSConfig struct {
	Gate     *Gate
	Resolver PathVarResolver
	Log      *slog.Logger
}

// FSAPI implements the fs.* bridge namespace's read-path. Construct via
// NewFSAPI. Safe for concurrent use by many bridge connections.
type FSAPI struct {
	gate     *Gate
	resolver PathVarResolver
	log      *slog.Logger
}

// NewFSAPI constructs an FSAPI. Panics if cfg.Gate or cfg.Resolver is nil
// — a mis-wired namespace is a programming error, not a runtime one.
func NewFSAPI(cfg FSConfig) *FSAPI {
	if cfg.Gate == nil {
		panic("bridge: NewFSAPI: Gate is required")
	}
	if cfg.Resolver == nil {
		panic("bridge: NewFSAPI: Resolver is required")
	}
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	return &FSAPI{gate: cfg.Gate, resolver: cfg.Resolver, log: log}
}

// Dispatch implements gateway.Namespace. envID + conn are unused for
// read-path methods (none are stream-capable in T9); kept in the signature
// so the shape matches gateway.Namespace for T23 wire-up.
func (a *FSAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	_ = envID
	_ = conn
	switch method {
	case "readFile":
		return a.handleReadFile(ctx, plugin, args)
	case "exists":
		return a.handleExists(ctx, plugin, args)
	case "stat":
		return a.handleStat(ctx, plugin, args)
	case "readDir":
		return a.handleReadDir(ctx, plugin, args)
	default:
		we := &WireError{Code: "EUNAVAIL", Message: fmt.Sprintf("fs: method %q not available", method)}
		return nil, fmt.Errorf("fs %s: %w", method, we)
	}
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

// unpackPathArg returns the first positional path argument from an
// array-shaped args payload. The second return is the raw array so
// callers can pull subsequent opts. Errors are WireError-wrapped so the
// handler layer can map them cleanly.
func unpackPathArg(method string, args json.RawMessage) (string, []json.RawMessage, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.%s: args must be [path, ...]", method)}
		return "", nil, fmt.Errorf("fs.%s: %w", method, we)
	}
	var p string
	if err := json.Unmarshal(raw[0], &p); err != nil || p == "" {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.%s: path must be a non-empty string", method)}
		return "", nil, fmt.Errorf("fs.%s: %w", method, we)
	}
	return p, raw, nil
}

// authorizePath is the gating spine every read-path method runs. It:
//
//  1. Resolves path-vars for the plugin.
//  2. Expands ${...} in the raw input path.
//  3. Cleans the path with filepath.Clean (collapses "..", "//" etc).
//  4. Rejects non-absolute inputs up front (the gate also rejects these,
//     but the distinct EINVAL code is friendlier than the generic EPERM).
//  5. Calls Gate.CheckExpanded with Need{Cap:"fs.read", Target: cleaned}.
//
// Returns the cleaned absolute path on success. The caller is expected to
// run the TOCTOU EvalSymlinks re-check via authorizeSymlinkResolved
// immediately before opening the file.
func (a *FSAPI) authorizePath(ctx context.Context, plugin, method, rawPath string) (string, error) {
	vars, err := a.resolver.Resolve(ctx, plugin)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs.%s: resolve path vars: %v", method, err)}
		return "", fmt.Errorf("fs.%s: %w", method, we)
	}

	expanded := ExpandPathVars(rawPath, vars)
	cleaned := filepath.Clean(expanded)

	// Leftover ${...} means the plugin referenced a variable the resolver
	// did not populate — treat that as a path it doesn't actually know,
	// fail closed with EINVAL rather than EPERM so the error message
	// points at the real cause.
	if containsUnresolvedVar(cleaned) {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.%s: unresolved path variable in %q", method, rawPath)}
		return "", fmt.Errorf("fs.%s: %w", method, we)
	}

	if !filepath.IsAbs(cleaned) {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.%s: path must be absolute, got %q", method, rawPath)}
		return "", fmt.Errorf("fs.%s: %w", method, we)
	}

	if err := a.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.read", Target: cleaned}, vars); err != nil {
		return "", err
	}
	return cleaned, nil
}

// authorizeSymlinkResolved is the TOCTOU defence. After the first gate
// check passed on `cleaned`, resolve symlinks with filepath.EvalSymlinks
// and re-check. If the resolved path differs AND the second check fails,
// return PermError — a plugin must not read through a symlink that
// escapes its granted roots, even if the original path matched.
//
// Missing-file cases (ENOENT) bubble through unchanged so stat / exists
// can distinguish between "blocked" and "absent".
func (a *FSAPI) authorizeSymlinkResolved(ctx context.Context, plugin, cleaned string) (string, error) {
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		// Not a symlink failure — bubble the OS error up so callers can
		// map ENOENT → friendly message while other errors (permission
		// denied on the host FS) surface EINTERNAL.
		return "", err
	}
	if resolved == cleaned {
		// No symlink in the path — first gate check already authorised.
		return resolved, nil
	}

	vars, vErr := a.resolver.Resolve(ctx, plugin)
	if vErr != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs: resolve path vars on EvalSymlinks recheck: %v", vErr)}
		return "", fmt.Errorf("fs: %w", we)
	}
	if err := a.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.read", Target: resolved}, vars); err != nil {
		// Symlink escape — wrap as EPERM with a clear message.
		var pe *PermError
		if errors.As(err, &pe) {
			return "", &PermError{Code: "EPERM", Msg: fmt.Sprintf("fs: symlink target %q is outside granted paths", resolved)}
		}
		return "", err
	}
	return resolved, nil
}

// ─────────────────────────────────────────────
// readFile
// ─────────────────────────────────────────────

type readFileOpts struct {
	Encoding string `json:"encoding,omitempty"` // "utf8" (default) or "base64"
}

// handleReadFile implements fs.readFile(path, opts?{encoding}) → string.
//
// utf8 (default): the file bytes are returned as a Go string. Non-UTF-8
// bytes are preserved verbatim — SDKs decide how to handle invalid
// sequences on their side.
// base64: the file bytes are base64-encoded (standard alphabet, padding on).
func (a *FSAPI) handleReadFile(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	rawPath, rawArgs, err := unpackPathArg("readFile", args)
	if err != nil {
		return nil, err
	}
	opts := readFileOpts{Encoding: "utf8"}
	if len(rawArgs) >= 2 && len(rawArgs[1]) > 0 && string(rawArgs[1]) != "null" {
		if uErr := json.Unmarshal(rawArgs[1], &opts); uErr != nil {
			we := &WireError{Code: "EINVAL", Message: "fs.readFile: opts must be {encoding?}"}
			return nil, fmt.Errorf("fs.readFile: %w", we)
		}
	}
	if opts.Encoding == "" {
		opts.Encoding = "utf8"
	}
	if opts.Encoding != "utf8" && opts.Encoding != "base64" {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.readFile: encoding must be utf8 or base64, got %q", opts.Encoding)}
		return nil, fmt.Errorf("fs.readFile: %w", we)
	}

	cleaned, err := a.authorizePath(ctx, plugin, "readFile", rawPath)
	if err != nil {
		return nil, err
	}
	resolved, err := a.authorizeSymlinkResolved(ctx, plugin, cleaned)
	if err != nil {
		return nil, mapFSError("readFile", err)
	}

	info, err := os.Lstat(resolved)
	if err != nil {
		return nil, mapFSError("readFile", err)
	}
	if info.IsDir() {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.readFile: %q is a directory", rawPath)}
		return nil, fmt.Errorf("fs.readFile: %w", we)
	}
	if info.Size() > MaxReadFileBytes {
		we := &WireError{Code: "EINVAL", Message: "fs.readFile: file exceeds 10 MiB cap"}
		return nil, fmt.Errorf("fs.readFile: %w", we)
	}

	// #nosec G304 — resolved path has passed two Gate.CheckExpanded calls
	// (pre- and post-EvalSymlinks). The whole purpose of this package is
	// to open plugin-supplied paths after capability validation.
	f, err := os.Open(resolved)
	if err != nil {
		return nil, mapFSError("readFile", err)
	}
	defer f.Close()

	// Bounded read — even if Lstat lied about the size (e.g. racy grow),
	// we never read past the hard cap.
	buf, err := io.ReadAll(io.LimitReader(f, MaxReadFileBytes+1))
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs.readFile: read: %v", err)}
		return nil, fmt.Errorf("fs.readFile: %w", we)
	}
	if int64(len(buf)) > MaxReadFileBytes {
		we := &WireError{Code: "EINVAL", Message: "fs.readFile: file exceeds 10 MiB cap"}
		return nil, fmt.Errorf("fs.readFile: %w", we)
	}

	if opts.Encoding == "base64" {
		return base64.StdEncoding.EncodeToString(buf), nil
	}
	return string(buf), nil
}

// ─────────────────────────────────────────────
// exists
// ─────────────────────────────────────────────

// handleExists implements fs.exists(path) → bool.
// Returns false on ENOENT. Any other OS error propagates as EINTERNAL
// so the plugin knows the answer is "unknown", not "absent".
func (a *FSAPI) handleExists(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	rawPath, _, err := unpackPathArg("exists", args)
	if err != nil {
		return nil, err
	}
	cleaned, err := a.authorizePath(ctx, plugin, "exists", rawPath)
	if err != nil {
		return nil, err
	}
	resolved, err := a.authorizeSymlinkResolved(ctx, plugin, cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return nil, mapFSError("exists", err)
	}
	if _, sErr := os.Lstat(resolved); sErr != nil {
		if os.IsNotExist(sErr) {
			return false, nil
		}
		return nil, mapFSError("exists", sErr)
	}
	return true, nil
}

// ─────────────────────────────────────────────
// stat
// ─────────────────────────────────────────────

type statResult struct {
	Size  int64  `json:"size"`
	Mtime string `json:"mtime"` // RFC3339
	IsDir bool   `json:"isDir"`
}

// handleStat implements fs.stat(path) → {size, mtime, isDir}.
func (a *FSAPI) handleStat(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	rawPath, _, err := unpackPathArg("stat", args)
	if err != nil {
		return nil, err
	}
	cleaned, err := a.authorizePath(ctx, plugin, "stat", rawPath)
	if err != nil {
		return nil, err
	}
	resolved, err := a.authorizeSymlinkResolved(ctx, plugin, cleaned)
	if err != nil {
		return nil, mapFSError("stat", err)
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return nil, mapFSError("stat", err)
	}
	return statResult{
		Size:  info.Size(),
		Mtime: info.ModTime().UTC().Format(time.RFC3339),
		IsDir: info.IsDir(),
	}, nil
}

// ─────────────────────────────────────────────
// readDir
// ─────────────────────────────────────────────

type readDirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

// handleReadDir implements fs.readDir(path) → [{name, isDir}].
// Returns up to MaxReadDirEntries entries. No dot/double-dot filtering —
// the plugin sees exactly what os.ReadDir returned.
func (a *FSAPI) handleReadDir(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	rawPath, _, err := unpackPathArg("readDir", args)
	if err != nil {
		return nil, err
	}
	cleaned, err := a.authorizePath(ctx, plugin, "readDir", rawPath)
	if err != nil {
		return nil, err
	}
	resolved, err := a.authorizeSymlinkResolved(ctx, plugin, cleaned)
	if err != nil {
		return nil, mapFSError("readDir", err)
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, mapFSError("readDir", err)
	}
	out := make([]readDirEntry, 0, len(entries))
	for i, e := range entries {
		if i >= MaxReadDirEntries {
			break
		}
		out = append(out, readDirEntry{Name: e.Name(), IsDir: e.IsDir()})
	}
	return out, nil
}

// ─────────────────────────────────────────────
// error mapping
// ─────────────────────────────────────────────

// mapFSError translates filesystem errors into WireError envelopes. The
// mapping is deliberately narrow: ENOENT becomes a visible EINVAL/ENOENT
// rather than a vague EINTERNAL so plugins can distinguish "missing" from
// "internal error".
func mapFSError(method string, err error) error {
	// Permission-gate denials flow through unchanged — the gateway wraps
	// them as EPERM envelopes at the boundary.
	var pe *PermError
	if errors.As(err, &pe) {
		return err
	}
	var we *WireError
	if errors.As(err, &we) {
		return err
	}
	if os.IsNotExist(err) {
		w := &WireError{Code: "ENOENT", Message: fmt.Sprintf("fs.%s: %v", method, err)}
		return fmt.Errorf("fs.%s: %w", method, w)
	}
	if os.IsPermission(err) {
		// Host FS denied — surface as EINTERNAL so the plugin knows this is
		// the host's local permission, not a capability issue.
		w := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs.%s: host denied: %v", method, err)}
		return fmt.Errorf("fs.%s: %w", method, w)
	}
	w := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs.%s: %v", method, err)}
	return fmt.Errorf("fs.%s: %w", method, w)
}

// containsUnresolvedVar reports whether s still carries a literal ${...}
// pattern after expansion. ExpandPathVars leaves unknown vars literal by
// design (fail-closed) — we catch that here so the gate doesn't waste a
// round-trip on a path that can never match anyway.
func containsUnresolvedVar(s string) bool {
	// Cheap pre-check before scanning for the closing brace.
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '$' && s[i+1] == '{' {
			// Find matching '}' within a reasonable window.
			for j := i + 2; j < len(s); j++ {
				if s[j] == '}' {
					return true
				}
			}
		}
	}
	return false
}
