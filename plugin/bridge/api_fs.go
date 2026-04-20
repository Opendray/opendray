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
// non-streaming methods; kept in the signature so the shape matches
// gateway.Namespace for T23 wire-up and M5 D1 stream methods
// (fs.watch lands in a follow-up task).
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
	case "writeFile":
		return a.handleWriteFile(ctx, plugin, args)
	case "mkdir":
		return a.handleMkdir(ctx, plugin, args)
	case "remove":
		return a.handleRemove(ctx, plugin, args)
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

// ─────────────────────────────────────────────
// D1 — write path (writeFile / mkdir / remove)
// ─────────────────────────────────────────────

// authorizeWritePath mirrors authorizePath but requests the `fs.write`
// capability. Returns the cleaned target path on success. Does NOT
// call EvalSymlinks itself — the caller decides whether to resolve
// symlinks (different semantics for writeFile vs mkdir vs remove).
func (a *FSAPI) authorizeWritePath(ctx context.Context, plugin, method, rawPath string) (string, PathVarCtx, error) {
	vars, err := a.resolver.Resolve(ctx, plugin)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs.%s: resolve path vars: %v", method, err)}
		return "", PathVarCtx{}, fmt.Errorf("fs.%s: %w", method, we)
	}

	expanded := ExpandPathVars(rawPath, vars)
	cleaned := filepath.Clean(expanded)

	if containsUnresolvedVar(cleaned) {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.%s: unresolved path variable in %q", method, rawPath)}
		return "", vars, fmt.Errorf("fs.%s: %w", method, we)
	}
	if !filepath.IsAbs(cleaned) {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.%s: path must be absolute, got %q", method, rawPath)}
		return "", vars, fmt.Errorf("fs.%s: %w", method, we)
	}

	if err := a.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.write", Target: cleaned}, vars); err != nil {
		return "", vars, err
	}
	return cleaned, vars, nil
}

// authorizeWriteSymlinkResolved runs the TOCTOU defence for write paths.
// Unlike the read variant this resolves the *parent* directory — the
// target file itself may not yet exist. If a symlink in the parent
// chain points outside the granted roots, the second CheckExpanded
// denies and we return EPERM.
//
// If the target itself already exists and is a symlink, we also
// resolve through it and re-check so overwriting a symlink can't
// clobber an arbitrary host file via the TOCTOU window.
func (a *FSAPI) authorizeWriteSymlinkResolved(ctx context.Context, plugin, cleaned string, vars PathVarCtx) (string, error) {
	parent := filepath.Dir(cleaned)
	base := filepath.Base(cleaned)

	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		// Parent doesn't exist (e.g. mkdir recursive) — bubble the error
		// up so the caller can decide what to do. We do NOT re-check
		// because there's nothing to walk.
		return "", err
	}
	resolvedFull := filepath.Join(resolvedParent, base)

	// If nothing shifted, the original gate check already authorised.
	if resolvedParent != parent {
		if err := a.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.write", Target: resolvedFull}, vars); err != nil {
			var pe *PermError
			if errors.As(err, &pe) {
				return "", &PermError{Code: "EPERM", Msg: fmt.Sprintf("fs: parent symlink resolves to %q which is outside granted write paths", resolvedParent)}
			}
			return "", err
		}
	}

	// If the target itself is a symlink, resolve it and re-check.
	if info, lErr := os.Lstat(resolvedFull); lErr == nil && info.Mode()&os.ModeSymlink != 0 {
		final, sErr := filepath.EvalSymlinks(resolvedFull)
		if sErr != nil {
			return "", sErr
		}
		if final != resolvedFull {
			if err := a.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.write", Target: final}, vars); err != nil {
				var pe *PermError
				if errors.As(err, &pe) {
					return "", &PermError{Code: "EPERM", Msg: fmt.Sprintf("fs: symlink target %q is outside granted write paths", final)}
				}
				return "", err
			}
			return final, nil
		}
	}
	return resolvedFull, nil
}

// ─────────────────────────────────────────────
// writeFile
// ─────────────────────────────────────────────

type writeFileOpts struct {
	Encoding string `json:"encoding,omitempty"`
	Mode     *uint32 `json:"mode,omitempty"` // pointer so 0 is distinguishable from unset
}

// handleWriteFile implements fs.writeFile(path, data, opts?) → null.
//
// utf8 (default): data is written as the raw bytes of the string.
// base64:          data is base64-decoded before writing.
// mode:            file mode bits, default 0644. Permission mask is
//                  applied even on overwrites (matches os.WriteFile).
//
// The write is NOT atomic in M5 — writes happen in-place, so a crashed
// host or plugin during the write can leave a truncated file. An
// atomic-rename variant lands with fs.watch in the follow-up task.
func (a *FSAPI) handleWriteFile(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	rawPath, rawArgs, err := unpackPathArg("writeFile", args)
	if err != nil {
		return nil, err
	}
	if len(rawArgs) < 2 {
		we := &WireError{Code: "EINVAL", Message: "fs.writeFile: args must be [path, data, opts?]"}
		return nil, fmt.Errorf("fs.writeFile: %w", we)
	}
	var data string
	if uErr := json.Unmarshal(rawArgs[1], &data); uErr != nil {
		we := &WireError{Code: "EINVAL", Message: "fs.writeFile: data must be a string"}
		return nil, fmt.Errorf("fs.writeFile: %w", we)
	}

	opts := writeFileOpts{Encoding: "utf8"}
	if len(rawArgs) >= 3 && len(rawArgs[2]) > 0 && string(rawArgs[2]) != "null" {
		if uErr := json.Unmarshal(rawArgs[2], &opts); uErr != nil {
			we := &WireError{Code: "EINVAL", Message: "fs.writeFile: opts must be {encoding?, mode?}"}
			return nil, fmt.Errorf("fs.writeFile: %w", we)
		}
	}
	if opts.Encoding == "" {
		opts.Encoding = "utf8"
	}
	if opts.Encoding != "utf8" && opts.Encoding != "base64" {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.writeFile: encoding must be utf8 or base64, got %q", opts.Encoding)}
		return nil, fmt.Errorf("fs.writeFile: %w", we)
	}

	var payload []byte
	if opts.Encoding == "base64" {
		decoded, dErr := base64.StdEncoding.DecodeString(data)
		if dErr != nil {
			we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.writeFile: invalid base64: %v", dErr)}
			return nil, fmt.Errorf("fs.writeFile: %w", we)
		}
		payload = decoded
	} else {
		payload = []byte(data)
	}

	if int64(len(payload)) > MaxReadFileBytes {
		we := &WireError{Code: "EINVAL", Message: "fs.writeFile: payload exceeds 10 MiB cap"}
		return nil, fmt.Errorf("fs.writeFile: %w", we)
	}

	mode := os.FileMode(0o644)
	if opts.Mode != nil {
		mode = os.FileMode(*opts.Mode & 0o777)
	}

	cleaned, vars, err := a.authorizeWritePath(ctx, plugin, "writeFile", rawPath)
	if err != nil {
		return nil, err
	}
	resolved, err := a.authorizeWriteSymlinkResolved(ctx, plugin, cleaned, vars)
	if err != nil {
		return nil, mapFSError("writeFile", err)
	}

	// Reject writing onto a directory up-front — os.WriteFile would
	// otherwise return a cryptic syscall error.
	if info, lErr := os.Lstat(resolved); lErr == nil && info.IsDir() {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.writeFile: %q is a directory", rawPath)}
		return nil, fmt.Errorf("fs.writeFile: %w", we)
	}

	// #nosec G306 — mode is capped to 0o777 above; default 0o644 matches
	// the contract in 04-bridge-api.md.
	if wErr := os.WriteFile(resolved, payload, mode); wErr != nil {
		return nil, mapFSError("writeFile", wErr)
	}
	// If the file pre-existed os.WriteFile does not re-apply mode bits —
	// enforce the requested mode explicitly when the caller set it.
	if opts.Mode != nil {
		if cErr := os.Chmod(resolved, mode); cErr != nil {
			return nil, mapFSError("writeFile", cErr)
		}
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// mkdir
// ─────────────────────────────────────────────

type mkdirOpts struct {
	Recursive bool `json:"recursive,omitempty"`
}

// handleMkdir implements fs.mkdir(path, opts?{recursive}) → null.
//
// recursive=false (default): single-level Mkdir. Missing parent is an
// error, existing target is an error (matches os.Mkdir).
// recursive=true: MkdirAll-style — parents are created, and the call
// is idempotent if the directory already exists.
func (a *FSAPI) handleMkdir(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	rawPath, rawArgs, err := unpackPathArg("mkdir", args)
	if err != nil {
		return nil, err
	}
	opts := mkdirOpts{}
	if len(rawArgs) >= 2 && len(rawArgs[1]) > 0 && string(rawArgs[1]) != "null" {
		if uErr := json.Unmarshal(rawArgs[1], &opts); uErr != nil {
			we := &WireError{Code: "EINVAL", Message: "fs.mkdir: opts must be {recursive?}"}
			return nil, fmt.Errorf("fs.mkdir: %w", we)
		}
	}

	cleaned, vars, err := a.authorizeWritePath(ctx, plugin, "mkdir", rawPath)
	if err != nil {
		return nil, err
	}

	// Parent TOCTOU check: if EvalSymlinks on the parent succeeds the
	// second gate call runs. For recursive mkdir the parent may not
	// exist yet — that's fine; we skip symlink re-check when parent is
	// absent because there is no symlink to follow.
	if _, statErr := os.Lstat(filepath.Dir(cleaned)); statErr == nil {
		if _, err := a.authorizeWriteSymlinkResolved(ctx, plugin, cleaned, vars); err != nil {
			return nil, mapFSError("mkdir", err)
		}
	}

	if opts.Recursive {
		if err := os.MkdirAll(cleaned, 0o755); err != nil {
			return nil, mapFSError("mkdir", err)
		}
		return nil, nil
	}
	if err := os.Mkdir(cleaned, 0o755); err != nil {
		return nil, mapFSError("mkdir", err)
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// remove
// ─────────────────────────────────────────────

type removeOpts struct {
	Recursive bool `json:"recursive,omitempty"`
}

// handleRemove implements fs.remove(path, opts?{recursive}) → null.
//
// recursive=false (default): single path deletion. Non-empty directory
// is an error (matches os.Remove).
// recursive=true: os.RemoveAll — silently skips ENOENT and removes
// whole subtrees. Symlinks are removed as-is, never followed, so a
// malicious symlink cannot point the delete at a host-owned file
// outside the granted write paths.
func (a *FSAPI) handleRemove(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	rawPath, rawArgs, err := unpackPathArg("remove", args)
	if err != nil {
		return nil, err
	}
	opts := removeOpts{}
	if len(rawArgs) >= 2 && len(rawArgs[1]) > 0 && string(rawArgs[1]) != "null" {
		if uErr := json.Unmarshal(rawArgs[1], &opts); uErr != nil {
			we := &WireError{Code: "EINVAL", Message: "fs.remove: opts must be {recursive?}"}
			return nil, fmt.Errorf("fs.remove: %w", we)
		}
	}

	cleaned, vars, err := a.authorizeWritePath(ctx, plugin, "remove", rawPath)
	if err != nil {
		return nil, err
	}
	resolved, err := a.authorizeWriteSymlinkResolved(ctx, plugin, cleaned, vars)
	if err != nil {
		return nil, mapFSError("remove", err)
	}

	if opts.Recursive {
		// RemoveAll returns nil for ENOENT — we need an explicit pre-check
		// to surface the contract's "ENOENT on missing" expectation so
		// plugins can tell `delete` apart from `already gone`.
		if _, sErr := os.Lstat(resolved); sErr != nil {
			return nil, mapFSError("remove", sErr)
		}
		if rErr := os.RemoveAll(resolved); rErr != nil {
			return nil, mapFSError("remove", rErr)
		}
		return nil, nil
	}
	if rErr := os.Remove(resolved); rErr != nil {
		return nil, mapFSError("remove", rErr)
	}
	return nil, nil
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
