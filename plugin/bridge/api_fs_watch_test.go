package bridge

// api_fs_watch_test.go — TDD suite for M5 D1-watch (fs.watch / fs.unwatch).
//
// Tests spin up a real *Conn on a fakeWS + a real FSAPI over a t.TempDir,
// then drive watch/unwatch through FSAPI.Dispatch so the whole path
// (handler → conn.Subscribe → fsnotify pump → stream chunk) is exercised.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// fsWatchHarness bundles an FSAPI + a real *Conn + the backing fakeWS so
// tests can read out-of-band stream chunks.
type fsWatchHarness struct {
	api  *FSAPI
	ws   *fakeWS
	conn *Conn
	mgr  *Manager
	ws2  string
	plug string
}

// newFSWatchHarness wires an FSAPI with the given read grants + a Conn.
func newFSWatchHarness(t *testing.T, readGrants []string) *fsWatchHarness {
	t.Helper()
	h := newFSHarnessRW(t, readGrants, nil)
	mgr := NewManager(slog.Default())
	ws := &fakeWS{}
	conn := mgr.Register(h.plug, ws)
	return &fsWatchHarness{
		api:  h.api,
		ws:   ws,
		conn: conn,
		mgr:  mgr,
		ws2:  h.ws,
		plug: h.plug,
	}
}

// watchCall dispatches fs.watch(glob) through the api with a given envID
// (envID is the subscription id). Args shape matches the bridge contract.
func (h *fsWatchHarness) watchCall(t *testing.T, envID, glob string) (any, error) {
	t.Helper()
	raw, _ := json.Marshal([]any{glob})
	return h.api.Dispatch(context.Background(), h.plug, "watch", raw, envID, h.conn)
}

func (h *fsWatchHarness) unwatchCall(t *testing.T, subID string) (any, error) {
	t.Helper()
	raw, _ := json.Marshal([]any{subID})
	return h.api.Dispatch(context.Background(), h.plug, "unwatch", raw, "", h.conn)
}

// waitChunk waits up to timeout for the next envelope with NS==""+Stream=="chunk"
// matching subID. Returns the parsed payload or fails the test.
func (h *fsWatchHarness) waitChunk(t *testing.T, subID string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	seen := 0
	for time.Now().Before(deadline) {
		h.ws.mu.Lock()
		n := len(h.ws.writes)
		writes := h.ws.writes
		h.ws.mu.Unlock()
		for i := seen; i < n; i++ {
			var env Envelope
			if err := json.Unmarshal(writes[i], &env); err != nil {
				t.Fatalf("decode envelope[%d]: %v", i, err)
			}
			if env.ID == subID && env.Stream == "chunk" {
				var payload map[string]any
				if env.Data != nil {
					_ = json.Unmarshal(env.Data, &payload)
				}
				return payload
			}
		}
		seen = n
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("no chunk received within %s for sub %q (writes=%d)", timeout, subID, seen)
	return nil
}

// writeCount returns the current number of envelopes written to the
// fake WS. Callers snapshot this before an action so assertNoChunkSince
// only inspects writes that happened afterward.
func (h *fsWatchHarness) writeCount() int {
	h.ws.mu.Lock()
	defer h.ws.mu.Unlock()
	return len(h.ws.writes)
}

// assertNoChunkSince waits `quiet` and asserts no chunk for subID
// arrived in write indices ≥ fromIdx. Use writeCount() as the baseline
// immediately before triggering the filesystem event you expect NOT to
// fire a chunk.
func (h *fsWatchHarness) assertNoChunkSince(t *testing.T, subID string, fromIdx int, quiet time.Duration) {
	t.Helper()
	time.Sleep(quiet)
	h.ws.mu.Lock()
	defer h.ws.mu.Unlock()
	for i := fromIdx; i < len(h.ws.writes); i++ {
		var env Envelope
		if err := json.Unmarshal(h.ws.writes[i], &env); err == nil && env.ID == subID && env.Stream == "chunk" {
			t.Errorf("unexpected chunk for %q: %s", subID, string(h.ws.writes[i]))
		}
	}
}

// ─── cases ─────────────────────────────────────────────

// 1. watch + create → chunk{kind:create}
func TestFS_Watch_CreateChunk(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	_, err := h.watchCall(t, "w1", filepath.Join(h.ws2, "**"))
	if err != nil {
		t.Fatalf("watch: %v", err)
	}

	target := filepath.Join(h.ws2, "newfile.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o644); err != nil {
		t.Fatalf("create: %v", err)
	}
	payload := h.waitChunk(t, "w1", 2*time.Second)
	if payload["kind"] != "create" {
		t.Errorf("kind=%v, want create", payload["kind"])
	}
	if payload["path"] != target {
		t.Errorf("path=%v, want %s", payload["path"], target)
	}
}

// 2. watch + modify on existing file → chunk{kind:modify}
func TestFS_Watch_ModifyChunk(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})

	target := filepath.Join(h.ws2, "existing.txt")
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := h.watchCall(t, "w2", filepath.Join(h.ws2, "*.txt"))
	if err != nil {
		t.Fatalf("watch: %v", err)
	}

	if err := os.WriteFile(target, []byte("v2"), 0o644); err != nil {
		t.Fatalf("modify: %v", err)
	}
	payload := h.waitChunk(t, "w2", 2*time.Second)
	if payload["kind"] != "modify" && payload["kind"] != "create" {
		// Some filesystems deliver create-on-write for existing files under
		// certain flags; accept both but flag anything else.
		t.Errorf("kind=%v, want modify|create", payload["kind"])
	}
}

// 3. watch + delete → chunk{kind:delete}
func TestFS_Watch_DeleteChunk(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	target := filepath.Join(h.ws2, "doomed.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := h.watchCall(t, "w3", filepath.Join(h.ws2, "*.txt")); err != nil {
		t.Fatalf("watch: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	payload := h.waitChunk(t, "w3", 2*time.Second)
	if payload["kind"] != "delete" {
		t.Errorf("kind=%v, want delete", payload["kind"])
	}
}

// 4. glob filter: only .md fires, .txt doesn't.
func TestFS_Watch_GlobFilterSuppressesUnrelated(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	if _, err := h.watchCall(t, "w4", filepath.Join(h.ws2, "*.md")); err != nil {
		t.Fatalf("watch: %v", err)
	}
	// Unrelated extension → must NOT fire a chunk.
	baseline := h.writeCount()
	if err := os.WriteFile(filepath.Join(h.ws2, "ignored.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write .txt: %v", err)
	}
	h.assertNoChunkSince(t, "w4", baseline, 200*time.Millisecond)

	// Matching file → must fire.
	md := filepath.Join(h.ws2, "hello.md")
	if err := os.WriteFile(md, []byte("x"), 0o644); err != nil {
		t.Fatalf("write .md: %v", err)
	}
	payload := h.waitChunk(t, "w4", 2*time.Second)
	if payload["path"] != md {
		t.Errorf("path=%v, want %s", payload["path"], md)
	}
}

// 5. recursive /** picks up nested writes.
func TestFS_Watch_RecursiveDoubleStar(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	nested := filepath.Join(h.ws2, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := h.watchCall(t, "w5", filepath.Join(h.ws2, "**")); err != nil {
		t.Fatalf("watch: %v", err)
	}
	target := filepath.Join(nested, "deep.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	payload := h.waitChunk(t, "w5", 2*time.Second)
	if payload["path"] != target {
		t.Errorf("recursive watch missed nested file: got path=%v", payload["path"])
	}
}

// 6. ungranted path → EPERM.
func TestFS_Watch_Ungranted_EPERM(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/sub/**"})
	_, err := h.watchCall(t, "w6", filepath.Join(h.ws2, "**"))
	if err == nil {
		t.Fatal("expected EPERM, got nil")
	}
	expectPermErr(t, err)
}

// 7. relative glob → EINVAL.
func TestFS_Watch_Relative_EINVAL(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	_, err := h.watchCall(t, "w7", "relative/path/**")
	expectWireErr(t, err, "EINVAL")
}

// 8. unwatch stops further chunks.
func TestFS_Watch_Unwatch_StopsChunks(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	if _, err := h.watchCall(t, "w8", filepath.Join(h.ws2, "*.txt")); err != nil {
		t.Fatalf("watch: %v", err)
	}
	// Prove it's live first.
	if err := os.WriteFile(filepath.Join(h.ws2, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = h.waitChunk(t, "w8", 2*time.Second)

	if _, err := h.unwatchCall(t, "w8"); err != nil {
		t.Fatalf("unwatch: %v", err)
	}

	// Any subsequent write must NOT produce a chunk for w8.
	baseline := h.writeCount()
	if err := os.WriteFile(filepath.Join(h.ws2, "b.txt"), []byte("y"), 0o644); err != nil {
		t.Fatalf("post-unwatch write: %v", err)
	}
	h.assertNoChunkSince(t, "w8", baseline, 250*time.Millisecond)
}

// 9. unwatch unknown subID → ENOENT.
func TestFS_Unwatch_Unknown_ENOENT(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	_, err := h.unwatchCall(t, "does-not-exist")
	expectWireErr(t, err, "ENOENT")
}

// 10. hot-revoke: InvalidateConsent(plugin, "fs") terminates the watch and
// emits a trailing EPERM stream:"end" envelope.
func TestFS_Watch_HotRevoke_EmitsStreamEnd(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	if _, err := h.watchCall(t, "w10", filepath.Join(h.ws2, "*.txt")); err != nil {
		t.Fatalf("watch: %v", err)
	}

	h.mgr.InvalidateConsent(h.plug, "fs")

	// The manager writes an EPERM stream:"end" for this sub synchronously-
	// enough that within 1s we should see it on the wire.
	deadline := time.Now().Add(time.Second)
	var found bool
	for time.Now().Before(deadline) {
		h.ws.mu.Lock()
		writes := append([][]byte{}, h.ws.writes...)
		h.ws.mu.Unlock()
		for _, raw := range writes {
			var env Envelope
			if err := json.Unmarshal(raw, &env); err != nil {
				continue
			}
			if env.ID == "w10" && env.Stream == "end" && env.Error != nil && env.Error.Code == "EPERM" {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !found {
		t.Error("hot-revoke did not emit EPERM stream:end for sub w10")
	}

	// After revoke, new filesystem events must be suppressed (pump has exited).
	baseline := h.writeCount()
	if err := os.WriteFile(filepath.Join(h.ws2, "later.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("post-revoke write: %v", err)
	}
	h.assertNoChunkSince(t, "w10", baseline, 200*time.Millisecond)
}

// 11. duplicate subID on the same conn → EINVAL.
func TestFS_Watch_DuplicateSubID(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	if _, err := h.watchCall(t, "dup", filepath.Join(h.ws2, "*.txt")); err != nil {
		t.Fatalf("first watch: %v", err)
	}
	_, err := h.watchCall(t, "dup", filepath.Join(h.ws2, "*.md"))
	expectWireErr(t, err, "EINVAL")
}

// 12. missing envID → EINVAL.
func TestFS_Watch_MissingEnvID_EINVAL(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	raw, _ := json.Marshal([]any{filepath.Join(h.ws2, "*.txt")})
	_, err := h.api.Dispatch(context.Background(), h.plug, "watch", raw, "", h.conn)
	expectWireErr(t, err, "EINVAL")
}

// 13. missing conn → EUNAVAIL (stream requires WS).
func TestFS_Watch_NilConn_EUNAVAIL(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	raw, _ := json.Marshal([]any{filepath.Join(h.ws2, "*.txt")})
	_, err := h.api.Dispatch(context.Background(), h.plug, "watch", raw, "w-nil", nil)
	expectWireErr(t, err, "EUNAVAIL")
}

// 14. watch walks capped at maxWatchWalkEnt for recursive globs.
// Creates a single-level fan-out deeper than the cap via a lowered limit.
func TestFS_Watch_RecursiveCap_TooManyDirs(t *testing.T) {
	orig := maxWatchDirs
	maxWatchDirs = 4
	t.Cleanup(func() { maxWatchDirs = orig })

	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	for i := 0; i < 6; i++ {
		if err := os.Mkdir(filepath.Join(h.ws2, fmt.Sprintf("d%d", i)), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	_, err := h.watchCall(t, "w14", filepath.Join(h.ws2, "**"))
	expectWireErr(t, err, "EINVAL")
}

// 15. watch on windows: symlink tests skipped elsewhere, but glob path
// expansion must still produce a directory for "x/*" style globs.
func TestFS_Watch_NonRecursiveGlob_OnlyWatchesRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping dir-watch semantics on windows")
	}
	h := newFSWatchHarness(t, []string{"${workspace}/**"})
	_ = os.Mkdir(filepath.Join(h.ws2, "siblingdir"), 0o755)

	if _, err := h.watchCall(t, "w15", filepath.Join(h.ws2, "*.txt")); err != nil {
		t.Fatalf("watch: %v", err)
	}
	// Writing inside a sibling subdirectory must NOT fire (glob is
	// non-recursive so the walk only added the root).
	nested := filepath.Join(h.ws2, "siblingdir", "hidden.txt")
	baseline := h.writeCount()
	if err := os.WriteFile(nested, []byte("x"), 0o644); err != nil {
		t.Fatalf("write nested: %v", err)
	}
	h.assertNoChunkSince(t, "w15", baseline, 200*time.Millisecond)
}

// ─── parallel safety: concurrent watch calls from one conn ──────────────

// TestFS_Watch_ConcurrentSubs verifies multiple concurrent watch subs
// from the same conn don't race the API's watch map.
func TestFS_Watch_ConcurrentSubs(t *testing.T) {
	h := newFSWatchHarness(t, []string{"${workspace}/**"})

	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, _ = h.watchCall(t, fmt.Sprintf("c%d", i), filepath.Join(h.ws2, "*.txt"))
		}()
	}
	wg.Wait()

	// Unwatch them all concurrently.
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, _ = h.unwatchCall(t, fmt.Sprintf("c%d", i))
		}()
	}
	wg.Wait()
}
