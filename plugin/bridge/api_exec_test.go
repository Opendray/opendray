//go:build !windows

package bridge

// api_exec_test.go — TDD suite for M3 T11 (opendray.exec.*).
//
// Skipped on windows via the build tag: exec/signal semantics differ too
// much (Setpgid, syscall.Kill(-pid, sig), fork-bomb assumptions) for
// cross-platform coverage in this PR.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ─────────────────────────────────────────────
// Harness
// ─────────────────────────────────────────────

type execTestHarness struct {
	api  *ExecAPI
	conn *Conn
	ws   *fakeWS
	plug string
}

// newExecHarness wires an ExecAPI + fakeWS-backed Conn with the given
// exec grants. readGrants optionally populates fs.read grants for cwd
// checks; pass nil to leave cwd grants empty.
func newExecHarness(t *testing.T, execGrants []string, fsReadGrants []string) *execTestHarness {
	t.Helper()
	perms := map[string]any{
		"exec": execGrants,
	}
	if len(fsReadGrants) > 0 {
		perms["fs"] = map[string]any{"read": fsReadGrants}
	}
	raw, err := json.Marshal(perms)
	if err != nil {
		t.Fatalf("marshal perms: %v", err)
	}
	cr := &fakeConsentReaderFS{perms: raw, found: true}
	gate := NewGate(cr, nil, slog.Default())
	resolver := &fakeResolver{vars: PathVarCtx{Workspace: t.TempDir(), Home: "/home/test", Tmp: "/tmp"}}
	api := NewExecAPI(ExecConfig{Gate: gate, Resolver: resolver, Log: slog.Default()})

	mgr := NewManager(slog.Default())
	ws := &fakeWS{}
	conn := mgr.Register("testplugin", ws)

	return &execTestHarness{api: api, conn: conn, ws: ws, plug: "testplugin"}
}

// call invokes Dispatch with positional args.
func (h *execTestHarness) call(t *testing.T, method, envID string, args ...any) (any, error) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return h.api.Dispatch(context.Background(), h.plug, method, raw, envID, h.conn)
}

// waitForWrites blocks up to d until ws has at least n writes OR d passes.
func (h *execTestHarness) waitForWrites(t *testing.T, n int, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if h.ws.writeCount() >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// envelopesFrom returns decoded envelopes from fakeWS.
func envelopesFrom(t *testing.T, ws *fakeWS) []Envelope {
	t.Helper()
	ws.mu.Lock()
	defer ws.mu.Unlock()
	out := make([]Envelope, 0, len(ws.writes))
	for _, raw := range ws.writes {
		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("unmarshal env: %v", err)
		}
		out = append(out, env)
	}
	return out
}

// ─────────────────────────────────────────────
// Cases
// ─────────────────────────────────────────────

// 1. run — happy path, zero exit.
func TestExec_Run_ZeroExit(t *testing.T) {
	h := newExecHarness(t, []string{"echo *"}, nil)
	res, err := h.call(t, "run", "r1", "echo", []string{"hello"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	rr, ok := res.(runResult)
	if !ok {
		t.Fatalf("unexpected type: %T", res)
	}
	if rr.ExitCode != 0 {
		t.Errorf("exitCode = %d", rr.ExitCode)
	}
	if !strings.Contains(rr.Stdout, "hello") {
		t.Errorf("stdout = %q, want to contain hello", rr.Stdout)
	}
	if rr.TimedOut {
		t.Errorf("unexpected timedOut=true")
	}
}

// 2. run — non-zero exit bubbles through.
func TestExec_Run_NonZeroExit(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	res, err := h.call(t, "run", "r1", "sh", []string{"-c", "exit 7"})
	if err != nil {
		t.Fatalf("run sh exit 7: %v", err)
	}
	rr, _ := res.(runResult)
	if rr.ExitCode != 7 {
		t.Errorf("exit = %d, want 7", rr.ExitCode)
	}
}

// 3. run — EPERM on ungranted command.
func TestExec_Run_UngrantedCommand_EPERM(t *testing.T) {
	h := newExecHarness(t, []string{"echo *"}, nil)
	_, err := h.call(t, "run", "r1", "rm", []string{"-rf", "/"})
	var pe *PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PermError, got %T: %v", err, err)
	}
}

// 4. run — timeout kills the child, returns timedOut=true.
func TestExec_Run_Timeout(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	res, err := h.call(t, "run", "r1",
		"sh", []string{"-c", "sleep 10"},
		map[string]any{"timeoutMs": 100})
	if err != nil {
		t.Fatalf("run with timeout: %v", err)
	}
	rr, _ := res.(runResult)
	if !rr.TimedOut {
		t.Errorf("timedOut = false, want true (exit=%d)", rr.ExitCode)
	}
}

// 5. run — opts.timeoutMs clamped to MaxTimeout. We can't easily test the
// full 5 min; instead verify resolveTimeout clamps via direct call.
func TestExec_ResolveTimeout_Clamp(t *testing.T) {
	h := newExecHarness(t, []string{"echo *"}, nil)
	// Override maxTimeout to 50 ms to make the clamp observable.
	h.api.maxTimeout = 50 * time.Millisecond
	got := h.api.resolveTimeout(execOpts{TimeoutMs: 10000})
	if got != 50*time.Millisecond {
		t.Errorf("timeout = %v, want clamp to 50ms", got)
	}
	// Default fires when TimeoutMs <= 0.
	got = h.api.resolveTimeout(execOpts{TimeoutMs: 0})
	if got != DefaultExecTimeout {
		t.Errorf("default timeout = %v, want %v", got, DefaultExecTimeout)
	}
}

// 6. run — EINVAL on malformed args.
func TestExec_Run_BadArgs_EINVAL(t *testing.T) {
	h := newExecHarness(t, []string{"echo *"}, nil)
	raw := json.RawMessage(`[]`)
	_, err := h.api.Dispatch(context.Background(), h.plug, "run", raw, "e", h.conn)
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINVAL" {
		t.Errorf("expected EINVAL, got %v", err)
	}
}

// 7. run — cwd outside fs grants returns EPERM.
func TestExec_Run_CwdOutsideGrants_EPERM(t *testing.T) {
	h := newExecHarness(t,
		[]string{"echo *"},
		[]string{"/tmp/sandbox/**"})
	_, err := h.call(t, "run", "r1",
		"echo", []string{"x"},
		map[string]any{"cwd": "/etc"})
	var pe *PermError
	if !errors.As(err, &pe) {
		t.Fatalf("expected EPERM for cwd escape, got %v", err)
	}
	if !strings.Contains(pe.Msg, "outside declared fs grants") {
		t.Errorf("unexpected msg: %v", pe.Msg)
	}
}

// 8. run — cwd allowed via AllowCwdOutsideFS.
func TestExec_Run_AllowCwdOutsideFS(t *testing.T) {
	cr := &fakeConsentReaderFS{perms: []byte(`{"exec":["echo *"]}`), found: true}
	gate := NewGate(cr, nil, slog.Default())
	resolver := &fakeResolver{vars: PathVarCtx{Workspace: "/tmp"}}
	api := NewExecAPI(ExecConfig{Gate: gate, Resolver: resolver, AllowCwdOutsideFS: true})
	raw, _ := json.Marshal([]any{
		"echo", []string{"hi"},
		map[string]any{"cwd": "/tmp"},
	})
	res, err := api.Dispatch(context.Background(), "p", "run", raw, "e", nil)
	if err != nil {
		t.Fatalf("allowed cwd run: %v", err)
	}
	if rr, _ := res.(runResult); rr.ExitCode != 0 {
		t.Errorf("exit=%d", rr.ExitCode)
	}
}

// 9. spawn — stream chunks + end envelope.
func TestExec_Spawn_StreamsThenEnds(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	res, err := h.call(t, "spawn", "sp-1",
		"sh", []string{"-c", "echo line1; echo line2"})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	sr, ok := res.(spawnResult)
	if !ok {
		t.Fatalf("unexpected type %T", res)
	}
	if sr.Pid <= 0 {
		t.Errorf("pid = %d", sr.Pid)
	}
	if sr.SubID != "sp-1" {
		t.Errorf("subId = %q", sr.SubID)
	}
	// Wait for the waiter goroutine to fire the end envelope.
	h.waitForWrites(t, 2, 3*time.Second)
	envs := envelopesFrom(t, h.ws)
	sawEnd := false
	sawChunk := false
	for _, e := range envs {
		if e.Stream == "chunk" {
			sawChunk = true
		}
		if e.Stream == "end" {
			sawEnd = true
		}
	}
	if !sawChunk {
		t.Errorf("no stream chunks written")
	}
	if !sawEnd {
		t.Errorf("no stream end envelope")
	}
}

// 10. spawn + wait — exit code is surfaced.
func TestExec_SpawnWait_ExitCode(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	_, err := h.call(t, "spawn", "sp-exit",
		"sh", []string{"-c", "exit 3"})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	res, err := h.call(t, "wait", "", "sp-exit")
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	wr, _ := res.(waitResult)
	if wr.ExitCode != 3 {
		t.Errorf("exit = %d, want 3", wr.ExitCode)
	}
}

// 11. spawn + kill(SIGTERM) → child dies, wait returns.
func TestExec_KillSIGTERM(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	_, err := h.call(t, "spawn", "sp-k",
		"sh", []string{"-c", "sleep 30"})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // let the process install
	if _, err := h.call(t, "kill", "", "sp-k"); err != nil {
		t.Fatalf("kill: %v", err)
	}
	// wait must return within a few seconds.
	waitCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	raw, _ := json.Marshal([]any{"sp-k"})
	res, err := h.api.Dispatch(waitCtx, h.plug, "wait", raw, "", h.conn)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	wr, _ := res.(waitResult)
	if wr.ExitCode == 0 {
		t.Errorf("exitCode = 0 (expected non-zero after SIGTERM)")
	}
}

// 12. kill(SIGTERM) escalates to SIGKILL when the child ignores TERM.
// We use sh that traps TERM and sleeps; the 5s escalation will kill it.
func TestExec_KillSIGTERM_EscalatesToSIGKILL(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	_, err := h.call(t, "spawn", "sp-escalate",
		"sh", []string{"-c", "trap '' TERM; sleep 30"})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	if _, err := h.call(t, "kill", "", "sp-escalate"); err != nil {
		t.Fatalf("kill: %v", err)
	}
	// Wait up to 8 s for the escalation (5 s + headroom).
	waitCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	raw, _ := json.Marshal([]any{"sp-escalate"})
	_, err = h.api.Dispatch(waitCtx, h.plug, "wait", raw, "", h.conn)
	if err != nil {
		t.Fatalf("wait after escalate: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 4500*time.Millisecond {
		t.Errorf("escalation too fast (%v) — SIGKILL timer should wait ≥5s", elapsed)
	}
	if elapsed > 7500*time.Millisecond {
		t.Errorf("escalation too slow (%v) — SIGKILL should fire by ~5s", elapsed)
	}
}

// 13. kill on unknown subId → ENOENT.
func TestExec_Kill_UnknownSubId_ENOENT(t *testing.T) {
	h := newExecHarness(t, []string{"echo *"}, nil)
	_, err := h.call(t, "kill", "", "nope")
	var we *WireError
	if !errors.As(err, &we) || we.Code != "ENOENT" {
		t.Errorf("expected ENOENT, got %v", err)
	}
}

// 14. write — stdin pipe round-trip. Spawn `head -n 1` which exits after
// reading one line, so the drain goroutine has time to flush before the
// pipes close. This avoids the kill/drain race of a plain `cat`.
func TestExec_Write_ToStdin(t *testing.T) {
	h := newExecHarness(t, []string{"head *"}, nil)
	_, err := h.call(t, "spawn", "sp-head", "head", []string{"-n", "1"})
	if err != nil {
		t.Fatalf("spawn head: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := h.call(t, "write", "", "sp-head", "piped input\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	// head -n 1 exits after reading one line.
	waitCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	raw, _ := json.Marshal([]any{"sp-head"})
	res, err := h.api.Dispatch(waitCtx, h.plug, "wait", raw, "", h.conn)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if wr, _ := res.(waitResult); wr.ExitCode != 0 {
		t.Errorf("head exit = %d, want 0", wr.ExitCode)
	}
	// Give the drainer a final moment to flush.
	time.Sleep(50 * time.Millisecond)
	envs := envelopesFrom(t, h.ws)
	sawPiped := false
	for _, e := range envs {
		if e.Stream == "chunk" && strings.Contains(string(e.Data), "piped input") {
			sawPiped = true
		}
	}
	if !sawPiped {
		t.Errorf("expected head to echo piped input (envelopes=%d)", len(envs))
	}
}

// 15. Fork-bomb cap — spawning >4 yields ETIMEOUT with retryAfterMs.
func TestExec_ForkBombCapped(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	// Launch 4 long-running sleeps first (fills the slot pool).
	for i := 0; i < MaxSpawnsPerPlugin; i++ {
		_, err := h.call(t, "spawn", fmt.Sprintf("bomb-%d", i),
			"sh", []string{"-c", "sleep 5"})
		if err != nil {
			t.Fatalf("spawn %d: %v", i, err)
		}
	}
	// 5th spawn must be rejected.
	_, err := h.call(t, "spawn", "bomb-over",
		"sh", []string{"-c", "sleep 5"})
	if err == nil {
		t.Fatalf("expected ETIMEOUT on 5th spawn")
	}
	var we *WireError
	if !errors.As(err, &we) || we.Code != "ETIMEOUT" {
		t.Errorf("expected ETIMEOUT, got %v", err)
	}
	if !strings.Contains(we.Message, "concurrent") {
		t.Errorf("unexpected message: %s", we.Message)
	}
	// Clean up.
	for i := 0; i < MaxSpawnsPerPlugin; i++ {
		_, _ = h.call(t, "kill", "", fmt.Sprintf("bomb-%d", i))
	}
}

// 16. High-parallel fork bomb stress — 10 concurrent spawn attempts,
// never more than 4 running simultaneously.
func TestExec_ForkBomb_AtMost4Alive(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	var wg sync.WaitGroup
	var accepted atomic.Int32
	var rejected atomic.Int32
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw, _ := json.Marshal([]any{
				"sh",
				[]string{"-c", "sleep 2"},
			})
			_, err := h.api.Dispatch(context.Background(), h.plug, "spawn", raw, fmt.Sprintf("s-%d", i), h.conn)
			if err == nil {
				accepted.Add(1)
			} else {
				rejected.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if accepted.Load() > int32(MaxSpawnsPerPlugin) {
		t.Errorf("accepted %d spawns, cap is %d", accepted.Load(), MaxSpawnsPerPlugin)
	}
	if accepted.Load()+rejected.Load() != 10 {
		t.Errorf("accepted+rejected = %d+%d, want 10", accepted.Load(), rejected.Load())
	}
	// Clean up — wait for everyone.
	for i := 0; i < 10; i++ {
		raw, _ := json.Marshal([]any{fmt.Sprintf("s-%d", i)})
		_, _ = h.api.Dispatch(context.Background(), h.plug, "kill", raw, "", h.conn)
	}
}

// 17. Unknown method → EUNAVAIL.
func TestExec_UnknownMethod_EUNAVAIL(t *testing.T) {
	h := newExecHarness(t, []string{"echo *"}, nil)
	_, err := h.call(t, "bogus", "e1", "echo", []string{"x"})
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EUNAVAIL" {
		t.Errorf("expected EUNAVAIL, got %v", err)
	}
}

// 18. NewExecAPI panics on nil Gate.
func TestExec_NewExecAPI_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	_ = NewExecAPI(ExecConfig{Resolver: &fakeResolver{}})
}

// 19. spawn without envID rejects up front.
func TestExec_Spawn_RequiresEnvID(t *testing.T) {
	h := newExecHarness(t, []string{"echo *"}, nil)
	raw, _ := json.Marshal([]any{"echo", []string{"x"}})
	_, err := h.api.Dispatch(context.Background(), h.plug, "spawn", raw, "", h.conn)
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINVAL" {
		t.Errorf("expected EINVAL, got %v", err)
	}
}

// 20. kill with unknown signal → EINVAL.
func TestExec_Kill_UnknownSignal_EINVAL(t *testing.T) {
	h := newExecHarness(t, []string{"sh *"}, nil)
	_, err := h.call(t, "spawn", "sp-sig", "sh", []string{"-c", "sleep 1"})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	_, err = h.call(t, "kill", "", "sp-sig", "SIGBOGUS")
	var we *WireError
	if !errors.As(err, &we) || we.Code != "EINVAL" {
		t.Errorf("expected EINVAL, got %v", err)
	}
	_, _ = h.call(t, "kill", "", "sp-sig")
}
