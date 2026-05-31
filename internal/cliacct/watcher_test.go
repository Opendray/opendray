package cliacct

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// fakeImporter stands in for the cliacct.Service.ImportLocal call so the
// watcher can be tested without a real Postgres. We only assert that
// ImportLocal was invoked at least once after a relevant fs event;
// ImportLocal's own behavior is covered by service_local_test.go.
type fakeImporter struct {
	mu        sync.Mutex
	callCount int
}

func (f *fakeImporter) ImportLocal(_ context.Context) ([]Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount++
	return nil, nil
}

func (f *fakeImporter) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

// newWatcherForTest builds a Watcher pointed at a fake importer. We
// rebind the watcher's svc-typed dependency via an interface shim that
// mirrors the single method (.ImportLocal) the watcher actually uses.
// In production the watcher takes *Service; here we use a thin wrapper
// to swap the implementation.
type minimalImporter interface {
	ImportLocal(context.Context) ([]Account, error)
}

// shimWatcher is the same as Watcher but accepts the interface so tests
// can inject a fake. The production constructor (NewWatcher) wires a
// concrete *Service.
type shimWatcher struct {
	imp      minimalImporter
	dir      string
	log      *slog.Logger
	debounce time.Duration
}

func (w *shimWatcher) run(ctx context.Context) {
	// Mirrors Watcher.Run() but uses the interface importer. Kept in
	// sync with the production Run() — if you change one, change both.
	if w.dir == "" {
		return
	}
	if _, err := w.imp.ImportLocal(ctx); err != nil && !isFSErrNotExist(err) {
		w.log.Warn("startup import failed", "err", err)
	}
	for ctx.Err() == nil {
		if err := w.watchOnce(ctx); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (w *shimWatcher) watchOnce(ctx context.Context) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer fsw.Close()
	if err := addDirWatch(fsw, w.dir); err != nil {
		return err
	}
	debounce := time.NewTimer(time.Hour)
	if !debounce.Stop() {
		<-debounce.C
	}
	defer debounce.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-fsw.Events:
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Lstat(ev.Name); err == nil && fi.IsDir() && fi.Mode()&os.ModeSymlink == 0 {
					_ = addDirWatch(fsw, ev.Name)
				}
			}
			if isInterestingEvent(ev) {
				debounce.Reset(w.debounce)
			}
		case <-debounce.C:
			_, _ = w.imp.ImportLocal(ctx)
		case err := <-fsw.Errors:
			return err
		}
	}
}

func isFSErrNotExist(err error) bool {
	return err != nil && (err == fs.ErrNotExist)
}

func TestWatcher_FiresOnCredentialsAppear(t *testing.T) {
	root := t.TempDir()
	imp := &fakeImporter{}
	w := &shimWatcher{
		imp:      imp,
		dir:      root,
		log:      slog.Default(),
		debounce: 50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		w.run(ctx)
		close(done)
	}()

	// Give the watcher a moment to register inotify watches.
	time.Sleep(150 * time.Millisecond)

	// The startup ImportLocal counts as call #1 — confirm.
	if got := imp.calls(); got < 1 {
		t.Fatalf("expected startup import-local call, got %d", got)
	}

	// Simulate the canonical user flow: mkdir, then claude-login writes
	// .credentials.json a moment later. Both should result in additional
	// ImportLocal invocations.
	startCalls := imp.calls()
	acctDir := filepath.Join(root, "test-acct")
	if err := os.Mkdir(acctDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Wait past the debounce window for the mkdir-triggered scan.
	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(acctDir, ".credentials.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// Now wait for the debounce-fired ImportLocal on the credentials write.
	waitFor(t, 2*time.Second, func() bool {
		return imp.calls() > startCalls
	})

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not exit after ctx cancel")
	}
}

func TestWatcher_RefusesSymlinkedRoot(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	link := filepath.Join(tmp, "link")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer fsw.Close()
	if err := addDirWatch(fsw, link); err == nil {
		t.Fatal("expected addDirWatch to reject symlinked dir")
	}
}

func TestIsInterestingEvent_CredentialsFile(t *testing.T) {
	tests := []struct {
		op    fsnotify.Op
		path  string
		want  bool
		label string
	}{
		{fsnotify.Create, "/x/foo/.credentials.json", true, "create credentials"},
		{fsnotify.Write, "/x/foo/.credentials.json", true, "write credentials"},
		{fsnotify.Remove, "/x/foo/.credentials.json", false, "remove credentials"},
		{fsnotify.Create, "/x/some-new-account", true, "create new subdir"},
		{fsnotify.Chmod, "/x/foo/.credentials.json", false, "chmod alone"},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			ev := fsnotify.Event{Op: tc.op, Name: tc.path}
			if got := isInterestingEvent(ev); got != tc.want {
				t.Errorf("isInterestingEvent(%v)=%v, want %v", ev, got, tc.want)
			}
		})
	}
}

// waitFor polls cond every 20ms until either it returns true or
// timeout elapses. Mirrors the helper in other internal/ test files.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("waitFor timed out after %s", timeout)
}
