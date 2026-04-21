// Package bridge — Exec namespace (M3 T11).
//
// # Scope
//
// Implements the `opendray.exec.*` bridge namespace: run, spawn, kill,
// wait, write. Every call passes through the capability gate before a
// process ever starts — a plugin granted `exec = ["git *"]` can run
// `git status` but not `rm -rf /`.
//
// # Process hardening
//
//   - Setpgid=true so kill tears down the full group including any
//     children the plugin launched from the subprocess.
//   - Optional network namespace unsharing on Linux when IsolateNetNS
//     is set AND the host has CAP_SYS_ADMIN.
//   - Default timeout 10 s; clamped by MaxTimeout (5 min default).
//   - Per-plugin hard cap of 4 concurrent spawn/run processes; excess
//     calls return ETIMEOUT with a retryAfterMs payload.
//
// # Streaming (spawn)
//
// stdout/stderr are combined into a single stream; each chunk is capped at
// 16 KiB. EOF is signalled with NewStreamEnd. Mid-stream errors surface
// via NewStreamChunkErr and do NOT terminate the stream — the SDK sees
// the error chunk followed by any remaining output and the final end
// envelope.
package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Hard caps.
const (
	// MaxSpawnsPerPlugin is the per-plugin concurrency ceiling for
	// run+spawn. Queue-past-that yields ETIMEOUT with retryAfterMs. This
	// is the first line of defence against fork-bomb abuse called out in
	// M3-PLAN §6.
	MaxSpawnsPerPlugin = 4
	// ExecChunkSize is the hard cap on a single stdout/stderr chunk in
	// spawn streaming mode. 16 KiB mirrors the wire-protocol guidance
	// and keeps the WS frame under the bridge's 1 MiB total limit.
	ExecChunkSize = 16 * 1024
	// DefaultExecTimeout applies when the caller did not pass
	// opts.timeoutMs. run and spawn both honour it.
	DefaultExecTimeout = 10 * time.Second
	// DefaultMaxExecTimeout is the ceiling an individual call may request
	// via opts.timeoutMs. 5 minutes.
	DefaultMaxExecTimeout = 5 * time.Minute
)

// ExecConfig wires ExecAPI's dependencies.
type ExecConfig struct {
	Gate     *Gate
	Resolver PathVarResolver
	Log      *slog.Logger
	// MaxTimeout clamps per-call opts.timeoutMs. Zero → DefaultMaxExecTimeout.
	MaxTimeout time.Duration
	// IsolateNetNS requests Linux CLONE_NEWNET isolation. Only applied
	// when the host process has CAP_SYS_ADMIN — otherwise a one-time
	// warning is logged and the flag is ignored.
	IsolateNetNS bool
	// AllowCwdOutsideFS lets spawn/run specify an opts.cwd outside the
	// plugin's fs.read/fs.write grants. Default false (safer).
	AllowCwdOutsideFS bool
}

// ExecAPI implements the exec.* bridge namespace. Safe for concurrent
// use. Construct via NewExecAPI.
type ExecAPI struct {
	gate     *Gate
	resolver PathVarResolver
	log      *slog.Logger

	maxTimeout   time.Duration
	isolateNetNS bool
	allowCwdOut  bool

	// Per-plugin concurrency gate and active-process book-keeping.
	mu        sync.Mutex
	inflight  map[string]int      // plugin → current active count
	processes map[string]*procEnt // subId → process entry
	warnedNetNS bool // one-time warning latch for missing CAP_SYS_ADMIN
}

// procEnt is an in-flight spawn tracked for kill/wait/write.
type procEnt struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	done   chan struct{} // closed when wait() returns
	exit   int
	err    error // wait error, if any
	plugin string
}

// NewExecAPI constructs an ExecAPI. Panics on missing Gate/Resolver — a
// mis-wired namespace is a programming error, not a runtime one.
func NewExecAPI(cfg ExecConfig) *ExecAPI {
	if cfg.Gate == nil {
		panic("bridge: NewExecAPI: Gate is required")
	}
	if cfg.Resolver == nil {
		panic("bridge: NewExecAPI: Resolver is required")
	}
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	maxT := cfg.MaxTimeout
	if maxT <= 0 {
		maxT = DefaultMaxExecTimeout
	}
	return &ExecAPI{
		gate:         cfg.Gate,
		resolver:     cfg.Resolver,
		log:          log,
		maxTimeout:   maxT,
		isolateNetNS: cfg.IsolateNetNS,
		allowCwdOut:  cfg.AllowCwdOutsideFS,
		inflight:     make(map[string]int),
		processes:    make(map[string]*procEnt),
	}
}

// Dispatch implements gateway.Namespace.
func (e *ExecAPI) Dispatch(ctx context.Context, plugin, method string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	switch method {
	case "run":
		return e.handleRun(ctx, plugin, args)
	case "spawn":
		return e.handleSpawn(ctx, plugin, args, envID, conn)
	case "kill":
		return e.handleKill(args)
	case "wait":
		return e.handleWait(ctx, args)
	case "write":
		return e.handleWrite(args)
	default:
		we := &WireError{Code: "EUNAVAIL", Message: fmt.Sprintf("exec: method %q not available", method)}
		return nil, fmt.Errorf("exec %s: %w", method, we)
	}
}

// ─────────────────────────────────────────────
// Args parsing
// ─────────────────────────────────────────────

// execOpts is the shared opts payload for run and spawn.
type execOpts struct {
	TimeoutMs int               `json:"timeoutMs,omitempty"`
	Cwd       string            `json:"cwd,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// parseExecArgs decodes [cmd, args[], opts?] into its components. Both cmd
// and the args list are required; opts is optional.
func parseExecArgs(method string, raw json.RawMessage) (cmd string, cmdArgs []string, opts execOpts, err error) {
	var argList []json.RawMessage
	if jErr := json.Unmarshal(raw, &argList); jErr != nil || len(argList) < 2 {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec.%s: args must be [cmd, args[], opts?]", method)}
		err = fmt.Errorf("exec.%s: %w", method, we)
		return
	}
	if uErr := json.Unmarshal(argList[0], &cmd); uErr != nil || cmd == "" {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec.%s: cmd must be a non-empty string", method)}
		err = fmt.Errorf("exec.%s: %w", method, we)
		return
	}
	if uErr := json.Unmarshal(argList[1], &cmdArgs); uErr != nil {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec.%s: args[1] must be []string", method)}
		err = fmt.Errorf("exec.%s: %w", method, we)
		return
	}
	if len(argList) >= 3 && len(argList[2]) > 0 && string(argList[2]) != "null" {
		if uErr := json.Unmarshal(argList[2], &opts); uErr != nil {
			we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec.%s: opts must be {timeoutMs?, cwd?, env?}", method)}
			err = fmt.Errorf("exec.%s: %w", method, we)
			return
		}
	}
	return
}

// parseSubIDArg returns the first positional subId argument.
func parseSubIDArg(method string, raw json.RawMessage) (string, []json.RawMessage, error) {
	var argList []json.RawMessage
	if err := json.Unmarshal(raw, &argList); err != nil || len(argList) < 1 {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec.%s: args must be [subId, ...]", method)}
		return "", nil, fmt.Errorf("exec.%s: %w", method, we)
	}
	var id string
	if err := json.Unmarshal(argList[0], &id); err != nil || id == "" {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec.%s: subId must be a non-empty string", method)}
		return "", nil, fmt.Errorf("exec.%s: %w", method, we)
	}
	return id, argList, nil
}

// ─────────────────────────────────────────────
// authorize — cap check + cwd grant check + concurrency slot
// ─────────────────────────────────────────────

// authorize runs the full pre-spawn check list: capability gate, cwd
// grant, concurrency limit. On success it reserves a slot (incFlight)
// and the caller MUST call decFlight once the process has exited.
func (e *ExecAPI) authorize(ctx context.Context, plugin, method, cmd string, cmdArgs []string, opts execOpts) error {
	// Build the cmdline the gate will match.
	cmdline := cmd
	if len(cmdArgs) > 0 {
		cmdline = cmd + " " + strings.Join(cmdArgs, " ")
	}
	if err := e.gate.Check(ctx, plugin, Need{Cap: "exec", Target: cmdline}); err != nil {
		return err
	}

	// cwd check: if set and AllowCwdOutsideFS is false, the cwd must lie
	// inside one of the plugin's fs.read OR fs.write grants.
	if opts.Cwd != "" && !e.allowCwdOut {
		if err := e.checkCwdInsideFSGrants(ctx, plugin, opts.Cwd); err != nil {
			return err
		}
	}

	// Concurrency slot — MUST be the last check so a denied call never
	// reserves a slot.
	if !e.tryIncFlight(plugin) {
		data, _ := json.Marshal(map[string]int64{"retryAfterMs": 250})
		we := &WireError{
			Code:    "ETIMEOUT",
			Message: fmt.Sprintf("exec.%s: per-plugin concurrent process cap (%d) exceeded", method, MaxSpawnsPerPlugin),
			Data:    data,
		}
		return fmt.Errorf("exec.%s: %w", method, we)
	}
	return nil
}

// checkCwdInsideFSGrants verifies cwd is inside the plugin's fs.read or
// fs.write grants after path-var expansion. We try both — either is
// sufficient for a working directory.
func (e *ExecAPI) checkCwdInsideFSGrants(ctx context.Context, plugin, cwd string) error {
	vars, vErr := e.resolver.Resolve(ctx, plugin)
	if vErr != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec: resolve path vars: %v", vErr)}
		return fmt.Errorf("exec: %w", we)
	}
	cleaned := filepath.Clean(ExpandPathVars(cwd, vars))
	if !filepath.IsAbs(cleaned) {
		we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec: cwd must be absolute, got %q", cwd)}
		return fmt.Errorf("exec: %w", we)
	}
	readErr := e.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.read", Target: cleaned}, vars)
	if readErr == nil {
		return nil
	}
	writeErr := e.gate.CheckExpanded(ctx, plugin, Need{Cap: "fs.write", Target: cleaned}, vars)
	if writeErr == nil {
		return nil
	}
	// Neither grant covers it — return EPERM with a specific message.
	return &PermError{Code: "EPERM", Msg: fmt.Sprintf("exec: cwd %q is outside declared fs grants", cleaned)}
}

// tryIncFlight atomically checks + reserves a slot. Returns false if the
// plugin is already at MaxSpawnsPerPlugin.
func (e *ExecAPI) tryIncFlight(plugin string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.inflight[plugin] >= MaxSpawnsPerPlugin {
		return false
	}
	e.inflight[plugin]++
	return true
}

// decFlight releases a slot reserved by tryIncFlight.
func (e *ExecAPI) decFlight(plugin string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.inflight[plugin] > 0 {
		e.inflight[plugin]--
	}
}

// ─────────────────────────────────────────────
// run
// ─────────────────────────────────────────────

type runResult struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timedOut"`
}

// handleRun executes synchronously and captures stdout+stderr fully.
func (e *ExecAPI) handleRun(ctx context.Context, plugin string, args json.RawMessage) (any, error) {
	cmd, cmdArgs, opts, err := parseExecArgs("run", args)
	if err != nil {
		return nil, err
	}
	if err := e.authorize(ctx, plugin, "run", cmd, cmdArgs, opts); err != nil {
		return nil, err
	}
	defer e.decFlight(plugin)

	timeout := e.resolveTimeout(opts)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204 — the command + args have just passed Gate.Check against
	// the plugin's declared `exec` grants. Capability gating IS the validation
	// layer for this call.
	c := exec.CommandContext(runCtx, cmd, cmdArgs...)
	if opts.Cwd != "" {
		c.Dir = opts.Cwd
	}
	if len(opts.Env) > 0 {
		c.Env = mergeEnv(opts.Env)
	}
	setProcAttrs(c, e.isolateNetNS, &e.mu, &e.warnedNetNS, e.log)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	runErr := c.Run()
	timedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)

	exitCode := 0
	if runErr != nil {
		var exErr *exec.ExitError
		switch {
		case errors.As(runErr, &exErr):
			exitCode = exErr.ExitCode()
		case timedOut:
			exitCode = -1
		default:
			we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec.run: %v", runErr)}
			return nil, fmt.Errorf("exec.run: %w", we)
		}
	}
	return runResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		TimedOut: timedOut,
	}, nil
}

// resolveTimeout picks a timeout for a call: default 10 s, clamped to
// maxTimeout, honouring opts.timeoutMs when positive.
func (e *ExecAPI) resolveTimeout(opts execOpts) time.Duration {
	if opts.TimeoutMs <= 0 {
		return DefaultExecTimeout
	}
	d := time.Duration(opts.TimeoutMs) * time.Millisecond
	if d > e.maxTimeout {
		return e.maxTimeout
	}
	return d
}

// mergeEnv returns os.Environ-compatible KEY=VALUE slice.
func mergeEnv(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

// ─────────────────────────────────────────────
// spawn
// ─────────────────────────────────────────────

type spawnResult struct {
	Pid   int    `json:"pid"`
	SubID string `json:"subId"`
}

// handleSpawn launches the process, registers a subscription on conn, and
// returns {pid, subId}. stdout/stderr are combined and pushed back as
// stream chunks until EOF; a final NewStreamEnd envelope closes the sub.
func (e *ExecAPI) handleSpawn(ctx context.Context, plugin string, args json.RawMessage, envID string, conn *Conn) (any, error) {
	if envID == "" {
		we := &WireError{Code: "EINVAL", Message: "exec.spawn: envelope id required for stream correlation"}
		return nil, fmt.Errorf("exec.spawn: %w", we)
	}
	if conn == nil {
		we := &WireError{Code: "EINVAL", Message: "exec.spawn: conn required for streaming"}
		return nil, fmt.Errorf("exec.spawn: %w", we)
	}
	cmd, cmdArgs, opts, err := parseExecArgs("spawn", args)
	if err != nil {
		return nil, err
	}
	if err := e.authorize(ctx, plugin, "spawn", cmd, cmdArgs, opts); err != nil {
		return nil, err
	}

	timeout := e.resolveTimeout(opts)
	spawnCtx, cancel := context.WithTimeout(context.Background(), timeout)

	// #nosec G204 — cmd+args passed Gate.Check above.
	c := exec.CommandContext(spawnCtx, cmd, cmdArgs...)
	if opts.Cwd != "" {
		c.Dir = opts.Cwd
	}
	if len(opts.Env) > 0 {
		c.Env = mergeEnv(opts.Env)
	}
	setProcAttrs(c, e.isolateNetNS, &e.mu, &e.warnedNetNS, e.log)

	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		cancel()
		e.decFlight(plugin)
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec.spawn: stdout pipe: %v", err)}
		return nil, fmt.Errorf("exec.spawn: %w", we)
	}
	stderrPipe, err := c.StderrPipe()
	if err != nil {
		cancel()
		e.decFlight(plugin)
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec.spawn: stderr pipe: %v", err)}
		return nil, fmt.Errorf("exec.spawn: %w", we)
	}
	stdinPipe, err := c.StdinPipe()
	if err != nil {
		cancel()
		e.decFlight(plugin)
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec.spawn: stdin pipe: %v", err)}
		return nil, fmt.Errorf("exec.spawn: %w", we)
	}

	if err := c.Start(); err != nil {
		cancel()
		e.decFlight(plugin)
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec.spawn: start: %v", err)}
		return nil, fmt.Errorf("exec.spawn: %w", we)
	}

	subID := envID
	// Register a conn subscription so hot-revoke can terminate the
	// stream cleanly. "exec" is the cap used for matching in
	// InvalidateConsent.
	done, err := conn.Subscribe(subID, "exec")
	if err != nil {
		// Conflicting subId — kill the already-started process and bail.
		_ = c.Process.Kill()
		cancel()
		e.decFlight(plugin)
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec.spawn: subscribe: %v", err)}
		return nil, fmt.Errorf("exec.spawn: %w", we)
	}

	ent := &procEnt{
		cmd:    c,
		stdin:  stdinPipe,
		done:   make(chan struct{}),
		plugin: plugin,
	}
	e.mu.Lock()
	e.processes[subID] = ent
	e.mu.Unlock()

	// Stream goroutine — drains stdout+stderr into the conn, emits end on EOF.
	go e.streamOutput(subID, stdoutPipe, stderrPipe, conn, done)

	// Waiter goroutine — captures exit code, releases slot, fires done.
	go func() {
		waitErr := c.Wait()
		cancel() // always release spawnCtx resources

		e.mu.Lock()
		ent.err = waitErr
		if waitErr == nil {
			ent.exit = 0
		} else {
			var exErr *exec.ExitError
			if errors.As(waitErr, &exErr) {
				ent.exit = exErr.ExitCode()
			} else {
				ent.exit = -1
			}
		}
		close(ent.done)
		e.mu.Unlock()

		e.decFlight(plugin)

		// After output drain completes (streamOutput closes pipes), emit stream end.
		// We fire-and-forget the conn write — if the conn is already closed,
		// nothing to do.
		_ = conn.WriteEnvelope(NewStreamEnd(subID))
		conn.Unsubscribe(subID)

		// Keep the entry in e.processes so late wait/kill calls can still
		// observe the result — callers clean up with a future gc pass.
	}()

	return spawnResult{Pid: c.Process.Pid, SubID: subID}, nil
}

// streamOutput chops stdout+stderr into ≤16 KiB chunks and pushes them
// through conn as NewStreamChunk envelopes. Exits on EOF on both streams
// OR on done-channel close (hot-revoke).
func (e *ExecAPI) streamOutput(subID string, stdout, stderr io.ReadCloser, conn *Conn, done <-chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(2)
	go drainPipe(subID, "stdout", stdout, conn, done, &wg, e.log)
	go drainPipe(subID, "stderr", stderr, conn, done, &wg, e.log)
	wg.Wait()
}

// drainPipe reads r in up-to-ExecChunkSize chunks and writes them as
// stream chunks on conn. stream="chunk" / data={stream:stream, bytes:base64}.
// Exits on EOF or done-channel close.
func drainPipe(subID, stream string, r io.ReadCloser, conn *Conn, done <-chan struct{}, wg *sync.WaitGroup, log *slog.Logger) {
	defer wg.Done()
	defer r.Close()
	buf := make([]byte, ExecChunkSize)
	for {
		// Non-blocking done check.
		select {
		case <-done:
			return
		default:
		}
		n, err := r.Read(buf)
		if n > 0 {
			payload := map[string]any{
				"stream": stream,
				"data":   string(buf[:n]),
			}
			env, buildErr := NewStreamChunk(subID, payload)
			if buildErr != nil {
				if log != nil {
					log.Warn("exec.spawn: chunk marshal failed", "subID", subID, "err", buildErr)
				}
				continue
			}
			if wErr := conn.WriteEnvelope(env); wErr != nil {
				// Conn closed — nothing more to do.
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && log != nil {
				log.Warn("exec.spawn: read pipe error", "subID", subID, "stream", stream, "err", err)
			}
			return
		}
	}
}

// ─────────────────────────────────────────────
// kill
// ─────────────────────────────────────────────

// handleKill sends a signal to the process group associated with subId.
// Default signal SIGTERM; after 5 s of no exit, SIGKILL escalates.
func (e *ExecAPI) handleKill(args json.RawMessage) (any, error) {
	id, argList, err := parseSubIDArg("kill", args)
	if err != nil {
		return nil, err
	}
	sig := syscall.SIGTERM
	if len(argList) >= 2 && len(argList[1]) > 0 && string(argList[1]) != "null" {
		var sigStr string
		if uErr := json.Unmarshal(argList[1], &sigStr); uErr != nil {
			we := &WireError{Code: "EINVAL", Message: "exec.kill: signal must be a string"}
			return nil, fmt.Errorf("exec.kill: %w", we)
		}
		switch strings.ToUpper(sigStr) {
		case "SIGTERM", "TERM":
			sig = syscall.SIGTERM
		case "SIGKILL", "KILL":
			sig = syscall.SIGKILL
		case "SIGINT", "INT":
			sig = syscall.SIGINT
		case "SIGHUP", "HUP":
			sig = syscall.SIGHUP
		default:
			we := &WireError{Code: "EINVAL", Message: fmt.Sprintf("exec.kill: unknown signal %q", sigStr)}
			return nil, fmt.Errorf("exec.kill: %w", we)
		}
	}

	e.mu.Lock()
	ent, ok := e.processes[id]
	e.mu.Unlock()
	if !ok {
		we := &WireError{Code: "ENOENT", Message: fmt.Sprintf("exec.kill: no process for subId %q", id)}
		return nil, fmt.Errorf("exec.kill: %w", we)
	}
	pid := ent.cmd.Process.Pid

	// Send to the full process group (Setpgid=true). Negative pid
	// targets the pgid on unix; on Windows we fall back to Process.Signal.
	if killErr := syscall.Kill(-pid, sig); killErr != nil {
		// Fall back to the direct process if group kill fails (e.g. race
		// where the process already reaped).
		_ = ent.cmd.Process.Signal(sig)
	}

	// Escalation: if SIGTERM doesn't reap within 5 s, send SIGKILL.
	if sig == syscall.SIGTERM {
		go func() {
			select {
			case <-ent.done:
				return
			case <-time.After(5 * time.Second):
				_ = syscall.Kill(-pid, syscall.SIGKILL)
			}
		}()
	}
	return nil, nil
}

// ─────────────────────────────────────────────
// wait
// ─────────────────────────────────────────────

type waitResult struct {
	ExitCode int `json:"exitCode"`
}

// handleWait blocks until the process identified by subId exits, returns
// its exit code. If the ctx is cancelled first, returns ETIMEOUT.
func (e *ExecAPI) handleWait(ctx context.Context, args json.RawMessage) (any, error) {
	id, _, err := parseSubIDArg("wait", args)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	ent, ok := e.processes[id]
	e.mu.Unlock()
	if !ok {
		we := &WireError{Code: "ENOENT", Message: fmt.Sprintf("exec.wait: no process for subId %q", id)}
		return nil, fmt.Errorf("exec.wait: %w", we)
	}
	select {
	case <-ent.done:
		return waitResult{ExitCode: ent.exit}, nil
	case <-ctx.Done():
		we := &WireError{Code: "ETIMEOUT", Message: "exec.wait: context deadline exceeded"}
		return nil, fmt.Errorf("exec.wait: %w", we)
	}
}

// ─────────────────────────────────────────────
// write
// ─────────────────────────────────────────────

// handleWrite pipes input to the process's stdin. Input is treated as a
// UTF-8 string; binary payloads should base64-encode before sending.
func (e *ExecAPI) handleWrite(args json.RawMessage) (any, error) {
	id, argList, err := parseSubIDArg("write", args)
	if err != nil {
		return nil, err
	}
	if len(argList) < 2 {
		we := &WireError{Code: "EINVAL", Message: "exec.write: args must be [subId, input]"}
		return nil, fmt.Errorf("exec.write: %w", we)
	}
	var input string
	if uErr := json.Unmarshal(argList[1], &input); uErr != nil {
		we := &WireError{Code: "EINVAL", Message: "exec.write: input must be a string"}
		return nil, fmt.Errorf("exec.write: %w", we)
	}
	e.mu.Lock()
	ent, ok := e.processes[id]
	e.mu.Unlock()
	if !ok {
		we := &WireError{Code: "ENOENT", Message: fmt.Sprintf("exec.write: no process for subId %q", id)}
		return nil, fmt.Errorf("exec.write: %w", we)
	}
	if _, wErr := ent.stdin.Write([]byte(input)); wErr != nil {
		we := &WireError{Code: "EINTERNAL", Message: fmt.Sprintf("exec.write: %v", wErr)}
		return nil, fmt.Errorf("exec.write: %w", we)
	}
	return nil, nil
}
