package bridge

// api_fs_watch.go — opendray.fs.watch streaming subscription (M5 D1-watch).
//
// Contract (see docs/plugin-platform/04-bridge-api.md §fs):
//
//	watch(glob: string) → { subId }
//	unwatch(subId: string)
//
// After a successful watch call, every filesystem change matching glob is
// pushed as a stream chunk envelope with body:
//
//	{ kind: "create"|"modify"|"delete", path: "<absolute>" }
//
// The subscription terminates on:
//   - explicit fs.unwatch(subId),
//   - the WS connection closing,
//   - consent revocation (DELETE /consents/fs) — a trailing EPERM
//     stream:"end" envelope is emitted by the Manager.
//
// # Authorisation
//
// Each watch requires the fs.read capability on the glob. The glob is
// expanded (${workspace}/etc), cleaned, and checked via Gate.CheckExpanded
// with Need{Cap:"fs.read", Target: cleaned}. A matching grant is mandatory.
//
// Per-event TOCTOU: when a change fires, the event path is EvalSymlinks'd
// and re-checked against the gate. If the resolved path is no longer
// covered by any fs.read grant (e.g. a symlink was swapped mid-watch), the
// chunk is suppressed — the plugin never sees the event.
//
// # Limits
//
//   - Max concurrent watch subs per API instance: 256. Beyond that the
//     call returns EINVAL (the plugin is likely leaking).
//   - Max directories added to one fsnotify.Watcher: 256. fsnotify falls
//     through to inotify on Linux, which has a per-user default of 8192 —
//     caps protect the shared table.
//   - Recursive expansion walks at most 4096 directory entries. Anything
//     deeper returns EINVAL.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Caps tuneable at build time; kept as vars so tests can lower them.
var (
	maxWatchSubs    = 256
	maxWatchDirs    = 256
	maxWatchWalkEnt = 4096
)

// watchSub tracks one live fs.watch subscription.
type watchSub struct {
	plugin  string
	subID   string
	pattern string
	conn    *Conn
	watcher *fsnotify.Watcher

	// cancel stops the pump goroutine when we unwind on our own (explicit
	// unwatch, conn close cleanup). done-channel from conn.Subscribe drives
	// the revoke path — closing it fires the same exit branch.
	cancel context.CancelFunc
}

// handleWatch implements fs.watch(glob, opts?). envID is the subscription
// id the client correlates with. Returns {subId} on success; pushes chunks
// out-of-band via conn.WriteEnvelope once the pump spins up.
func (a *FSAPI) handleWatch(ctx context.Context, plugin string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	if conn == nil {
		we := &WireError{Code: "EUNAVAIL", Message: "fs.watch: streaming requires a WebSocket connection"}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}
	if envID == "" {
		we := &WireError{Code: "EINVAL", Message: "fs.watch: envelope id is required for stream correlation"}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}

	rawGlob, _, err := unpackPathArg("watch", args)
	if err != nil {
		return nil, err
	}

	// Authorise: fs.read against the cleaned glob. We deliberately gate on
	// the glob itself — CheckExpanded already matches grants against a
	// pattern, and MatchFSPath handles the "/**" suffix.
	vars, err := a.resolver.Resolve(ctx, plugin)
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs.watch: resolve path vars: %v", err)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}
	expanded := ExpandPathVars(rawGlob, vars)
	cleaned := filepath.Clean(expanded)
	if containsUnresolvedVar(cleaned) {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.watch: unresolved path variable in %q", rawGlob)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}
	if !filepath.IsAbs(cleaned) {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.watch: glob must be absolute, got %q", rawGlob)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}
	if err := a.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.read", Target: cleaned}, vars); err != nil {
		return nil, err
	}

	// Global sub cap.
	a.watchMu.Lock()
	if len(a.watches) >= maxWatchSubs {
		a.watchMu.Unlock()
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.watch: too many active subscriptions (max %d)", maxWatchSubs)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}
	a.watchMu.Unlock()

	// Walk the glob into a concrete set of directories to register with
	// fsnotify. The walk is bounded (maxWatchWalkEnt) so a broken /** on a
	// huge tree can't DoS the host.
	rootDir, recursive := watchDirsForGlob(cleaned)
	dirs, err := enumerateWatchDirs(rootDir, recursive)
	if err != nil {
		return nil, mapFSError("watch", err)
	}
	if len(dirs) == 0 {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.watch: no watchable directory for glob %q", rawGlob)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}
	if len(dirs) > maxWatchDirs {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.watch: glob expands to %d directories (max %d)", len(dirs), maxWatchDirs)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("fs.watch: new watcher: %v", err)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}
	for _, d := range dirs {
		if err := w.Add(d); err != nil {
			_ = w.Close()
			return nil, mapFSError("watch", err)
		}
	}

	// Register with conn (capability name = "fs" so DELETE /consents/fs
	// tears every watcher down synchronously within the 200 ms SLO).
	done, err := conn.Subscribe(envID, "fs")
	if err != nil {
		_ = w.Close()
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("fs.watch: duplicate subscription id %q", envID)}
		return nil, fmt.Errorf("fs.watch: %w", we)
	}

	// Store in API registry so unwatch can find us.
	pumpCtx, cancel := context.WithCancel(context.Background())
	sub := &watchSub{
		plugin:  plugin,
		subID:   envID,
		pattern: cleaned,
		conn:    conn,
		watcher: w,
		cancel:  cancel,
	}
	key := watchKey(plugin, envID)
	a.watchMu.Lock()
	a.watches[key] = sub
	a.watchMu.Unlock()

	// Pump goroutine.
	go a.watchPump(pumpCtx, sub, done, vars)

	return map[string]string{"subId": envID}, nil
}

// handleUnwatch implements fs.unwatch(subId). Idempotent — a subID that
// isn't tracked returns ENOENT so plugins can detect a double-unsub.
func (a *FSAPI) handleUnwatch(plugin string, args json.RawMessage) (any, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(args, &raw); err != nil || len(raw) < 1 {
		we := &WireError{Code: "EINVAL", Message: "fs.unwatch: args must be [subId]"}
		return nil, fmt.Errorf("fs.unwatch: %w", we)
	}
	var subID string
	if err := json.Unmarshal(raw[0], &subID); err != nil || subID == "" {
		we := &WireError{Code: "EINVAL", Message: "fs.unwatch: subId must be a non-empty string"}
		return nil, fmt.Errorf("fs.unwatch: %w", we)
	}

	key := watchKey(plugin, subID)
	a.watchMu.Lock()
	sub, ok := a.watches[key]
	if ok {
		delete(a.watches, key)
	}
	a.watchMu.Unlock()
	if !ok {
		we := &WireError{Code: "ENOENT", Message: fmt.Sprintf("fs.unwatch: no subscription %q", subID)}
		return nil, fmt.Errorf("fs.unwatch: %w", we)
	}

	sub.cancel()
	_ = sub.watcher.Close()
	sub.conn.Unsubscribe(sub.subID)
	return nil, nil
}

// watchPump reads fsnotify events and forwards matched ones as stream
// chunks until done is closed or pumpCtx is cancelled. Cleanup happens
// inside the deferred block so every exit path releases the watcher.
func (a *FSAPI) watchPump(pumpCtx context.Context, sub *watchSub, done <-chan struct{}, vars PathVarCtx) {
	defer func() {
		_ = sub.watcher.Close()
		// Remove from map if still present (unwatch deletes ahead of us).
		a.watchMu.Lock()
		delete(a.watches, watchKey(sub.plugin, sub.subID))
		a.watchMu.Unlock()
	}()

	for {
		select {
		case <-done:
			// Hot-revoke or Unsubscribe-from-conn. Manager already writes
			// the EPERM stream:"end" for revoke; for plain Unsubscribe we
			// don't emit a trailing envelope (the client asked us to stop).
			return
		case <-pumpCtx.Done():
			return
		case ev, ok := <-sub.watcher.Events:
			if !ok {
				return
			}
			a.emitWatchEvent(sub, ev, vars)
		case err, ok := <-sub.watcher.Errors:
			if !ok {
				return
			}
			if a.log != nil {
				a.log.Warn("fs.watch: watcher error",
					slog.String("plugin", sub.plugin),
					slog.String("subID", sub.subID),
					slog.Any("err", err),
				)
			}
		}
	}
}

// emitWatchEvent filters one fsnotify event and, if it passes the glob +
// TOCTOU recheck, writes a stream chunk on the conn.
func (a *FSAPI) emitWatchEvent(sub *watchSub, ev fsnotify.Event, vars PathVarCtx) {
	kind := mapFSNotifyOp(ev.Op)
	if kind == "" {
		return
	}
	if !matchFSPattern(sub.pattern, ev.Name) {
		return
	}
	// TOCTOU recheck: skip for delete events (path no longer exists).
	if kind != "delete" {
		if resolved, err := filepath.EvalSymlinks(ev.Name); err == nil && resolved != ev.Name {
			if err := a.gate.CheckExpanded(context.Background(), sub.plugin,
				Need{Cap: "fs.read", Target: resolved}, vars); err != nil {
				return // silently suppress symlink escape
			}
		}
	}

	env, err := NewStreamChunk(sub.subID, map[string]any{
		"kind": kind,
		"path": ev.Name,
	})
	if err != nil {
		return
	}
	_ = sub.conn.WriteEnvelope(env)
}

// watchKey namespaces subIDs across plugins so the API-level map can't
// collide when two plugins happen to pick the same envelope id.
func watchKey(plugin, subID string) string {
	return plugin + "\x00" + subID
}

// mapFSNotifyOp translates fsnotify bitmask ops into the contract's event
// kind. Returns "" for ops we ignore (Chmod — v1 spec has no chmod event).
func mapFSNotifyOp(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create != 0:
		return "create"
	case op&fsnotify.Write != 0:
		return "modify"
	case op&fsnotify.Remove != 0, op&fsnotify.Rename != 0:
		return "delete"
	default:
		return ""
	}
}

// watchDirsForGlob extracts the directory root + recursion flag from a
// cleaned absolute glob. Rules:
//   - "/a/b/**" or "/a/b/**/..." → root=/a/b, recursive=true.
//   - "/a/b/*.md" → root=/a/b, recursive=false.
//   - "/a/b/file.txt" (no meta) → root=/a/b, recursive=false.
func watchDirsForGlob(glob string) (root string, recursive bool) {
	if idx := strings.Index(glob, "/**"); idx >= 0 {
		return glob[:idx], true
	}
	// Find the first meta char.
	if i := strings.IndexAny(glob, "*?["); i >= 0 {
		return filepath.Dir(glob[:i]), false
	}
	// No meta → watch the file's parent (file-level watch).
	return filepath.Dir(glob), false
}

// enumerateWatchDirs returns the absolute dir list fsnotify must watch.
// Non-recursive: only root. Recursive: root + every subdirectory found
// by a bounded walk. Returns ENOENT-ish errors verbatim so the caller
// can map them.
func enumerateWatchDirs(root string, recursive bool) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("fs.watch: root %q is not a directory", root)
	}
	if !recursive {
		return []string{root}, nil
	}

	dirs := []string{root}
	count := 0
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Skip inaccessible subtrees but keep walking.
			if errors.Is(walkErr, os.ErrPermission) {
				return filepath.SkipDir
			}
			return walkErr
		}
		count++
		if count > maxWatchWalkEnt {
			return fmt.Errorf("fs.watch: walk exceeded %d entries", maxWatchWalkEnt)
		}
		if !d.IsDir() || path == root {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

