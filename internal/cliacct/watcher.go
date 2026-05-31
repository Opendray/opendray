package cliacct

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher reacts to filesystem changes under the configured accounts dir
// and feeds the existing ImportLocal pipeline. The MCP UI claims that
// new ~/.claude-accounts/<name>/ directories register automatically;
// this watcher makes that promise real.
//
// Design notes:
//   - We trigger ImportLocal on a debounce timer (default 500ms) so a
//     burst of fsnotify events (mkdir → claude login → write credentials)
//     collapses to one DB-touching scan.
//   - Symlinks are rejected at every level: a malicious symlink under
//     ~/.claude-accounts/foo could otherwise trick the gateway into
//     reading arbitrary files. The Service-level helpers also Lstat;
//     this is defense in depth.
//   - The watched root may not exist yet at startup (fresh install) or
//     may be deleted at runtime. Run() retries with exponential backoff
//     so a missing dir does not silently disable the feature.
//   - A single Service.mu (lock-around-ImportLocal) makes concurrent
//     fsnotify events, button clicks, and startup scans race-free.
type Watcher struct {
	svc      *Service
	dir      string
	log      *slog.Logger
	debounce time.Duration
}

// NewWatcher constructs a Watcher around svc. dir is the accounts root
// (typically the result of Service.resolveAccountsDir()); empty disables
// the watcher entirely. Log is required; pass slog.Default() if nothing
// fancier is wired.
func NewWatcher(svc *Service, dir string, log *slog.Logger) *Watcher {
	if log == nil {
		log = slog.Default()
	}
	return &Watcher{
		svc:      svc,
		dir:      dir,
		log:      log.With("component", "cliacct.watcher"),
		debounce: 500 * time.Millisecond,
	}
}

// Run blocks until ctx is cancelled. Idempotent on early exit when dir
// is empty (caller can always spawn it without conditionals).
//
// Lifecycle:
//  1. ImportLocal once at startup so accounts added while the gateway
//     was down are picked up immediately (otherwise a real watch would
//     only catch *future* changes — agents flagged this P0).
//  2. Start an fsnotify watch on dir; on Create events for new subdirs
//     also add them as sub-watches so we see .credentials.json appear
//     inside.
//  3. On Create/Write events anywhere in the tree, reset a debounce
//     timer; when it fires, run ImportLocal once.
//  4. On any unrecoverable watcher error or the root vanishing, sleep
//     a backoff and retry from step 2.
func (w *Watcher) Run(ctx context.Context) {
	if w == nil || w.dir == "" || w.svc == nil {
		return
	}

	// Catch-up scan first. ImportLocal tolerates a missing dir (returns
	// nil, error from os.ReadDir wrapped as ErrNotExist-class), but the
	// service-level resolveAccountsDir returns an empty string only
	// when HOME is unset — in that case we already bailed above.
	if _, err := w.svc.ImportLocal(ctx); err != nil && !errors.Is(err, os.ErrNotExist) {
		w.log.Warn("startup import-local failed", "err", err)
	}

	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second
	for ctx.Err() == nil {
		err := w.watchOnce(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		w.log.Warn("watcher restarting after error", "err", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// watchOnce runs one fsnotify lifecycle: open a watcher, register the
// root + existing subdirs, react to events, and return when ctx is
// cancelled or fsnotify reports an unrecoverable error.
func (w *Watcher) watchOnce(ctx context.Context) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("new fsnotify watcher: %w", err)
	}
	defer fsw.Close()

	if err := addDirWatch(fsw, w.dir); err != nil {
		// Root missing or unreadable — caller retries.
		return fmt.Errorf("watch root %q: %w", w.dir, err)
	}

	// Pre-register existing subdirs so a .credentials.json appearing in
	// a directory created BEFORE the gateway started will still trigger
	// a debounce. New subdirs are added in the event loop below.
	if entries, err := os.ReadDir(w.dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "tokens" {
				continue
			}
			child := filepath.Join(w.dir, e.Name())
			if isSymlink(child) {
				w.log.Warn("skipping symlinked account dir", "path", child)
				continue
			}
			_ = addDirWatch(fsw, child) // best-effort; broken subdir is non-fatal
		}
	}

	// Debounce timer kept stopped until the first event arrives.
	debounce := time.NewTimer(time.Hour)
	if !debounce.Stop() {
		<-debounce.C
	}
	defer debounce.Stop()

	w.log.Info("watching accounts directory", "dir", w.dir, "debounce", w.debounce)

	for {
		select {
		case <-ctx.Done():
			return nil

		case ev, ok := <-fsw.Events:
			if !ok {
				return errors.New("fsnotify events channel closed")
			}
			// On creation of a new subdir at the root, start watching it
			// so the .credentials.json write inside also fires. We Lstat
			// to reject symlinks pointing outside the tree.
			if ev.Op&fsnotify.Create != 0 {
				if filepath.Dir(ev.Name) == w.dir {
					if fi, err := os.Lstat(ev.Name); err == nil && fi.IsDir() && fi.Mode()&os.ModeSymlink == 0 {
						_ = addDirWatch(fsw, ev.Name) // ignore add error; tx will retry
					}
				}
			}
			// Only schedule a scan for events that could mean a new
			// usable account. Cuts noise from token-file edits, vim
			// swapfiles, etc. while still catching the canonical
			// claude-login write of .credentials.json.
			if isInterestingEvent(ev) {
				debounce.Reset(w.debounce)
			}

		case <-debounce.C:
			if _, err := w.svc.ImportLocal(ctx); err != nil {
				w.log.Warn("import-local from watcher failed", "err", err)
			}

		case err, ok := <-fsw.Errors:
			if !ok {
				return errors.New("fsnotify errors channel closed")
			}
			// Some errors (ENOSPC on inotify limit) are recoverable
			// after a backoff; the outer loop handles that. Others
			// (broken watcher) ditto.
			return fmt.Errorf("fsnotify: %w", err)
		}
	}
}

// addDirWatch wraps fsw.Add with a pre-check for symlinks: we never want
// to add a watch on a symlinked directory, since the OS will then deliver
// events about whatever the symlink points at.
func addDirWatch(fsw *fsnotify.Watcher, path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink (refusing to watch)", path)
	}
	return fsw.Add(path)
}

// isSymlink is a thin wrapper around os.Lstat that swallows ENOENT etc.
// Used in pre-registration loops where a transient stat failure must not
// abort the whole scan.
func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

// isInterestingEvent decides whether an fsnotify event is worth a scan.
// We want:
//   - any Create/Write/Rename event touching a `.credentials.json`
//   - any Create event introducing a new subdir at the root
//
// We deliberately ignore Remove/Chmod-only events: deletions are handled
// by the operator via DELETE /claude-accounts/<id>, and chmod alone
// doesn't change account discoverability.
func isInterestingEvent(ev fsnotify.Event) bool {
	if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
		return false
	}
	name := filepath.Base(ev.Name)
	if name == ".credentials.json" {
		return true
	}
	// New subdir at the root: ev.Name will not have a "/.credentials.json" suffix
	// but ev.Op will include Create. Schedule a scan so a freshly mkdir'd
	// account dir surfaces even before its credentials file lands (the
	// debounce + subsequent .credentials.json write will refire as needed).
	if ev.Op&fsnotify.Create != 0 {
		// Heuristic: only react to Creates of plain entries at our watched
		// root or one level down; deeper Creates inside claude-internal
		// directories produce noise.
		return true
	}
	_ = fs.ErrNotExist // keep io/fs import for future use without lint warning
	return false
}
