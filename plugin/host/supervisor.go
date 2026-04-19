package host

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/opendray/opendray/plugin"
)

// ─────────────────────────────────────────────
// Errors
// ─────────────────────────────────────────────

// ErrNoHost is returned when Ensure is called for a plugin whose
// manifest doesn't declare a Host block. Use errors.Is.
var ErrNoHost = errors.New("host: plugin has no host-form manifest")

// ErrRuntimeNotFound is returned when the configured host runtime
// binary (node, deno, python3, bun) is not on PATH.
var ErrRuntimeNotFound = errors.New("host: runtime binary not found on PATH")

// ErrSupervisorStopped is returned by Ensure after Stop was called.
var ErrSupervisorStopped = errors.New("host: supervisor stopped")

// ─────────────────────────────────────────────
// Provider lookup — decouples supervisor from plugin.Runtime's
// concrete type so tests can inject a fake.
// ─────────────────────────────────────────────

// ProviderLookup resolves a plugin name to its manifest Provider.
// *plugin.Runtime satisfies this via Get. Defined here as the
// "interface lives with its consumer" rule.
type ProviderLookup interface {
	Get(name string) (plugin.Provider, bool)
}

// ─────────────────────────────────────────────
// State writer — best-effort lifecycle persistence
// ─────────────────────────────────────────────

// StateWriter records per-plugin sidecar lifecycle events to the
// plugin_host_state table (migration 015). All methods are best-
// effort; a database outage must never block sidecar startup.
type StateWriter interface {
	RecordStarted(ctx context.Context, plugin string) error
	RecordExited(ctx context.Context, plugin string, exitCode int, lastErr string) error
}

// ─────────────────────────────────────────────
// Config + constructor
// ─────────────────────────────────────────────

// Config carries Supervisor dependencies. Fields with zero defaults
// fall back to documented sensible values (10 min idle, 200 ms initial
// backoff, 5 s max backoff).
type Config struct {
	DataDir           string // ${PluginsDataDir} for sidecar cwd resolution
	Providers         ProviderLookup
	State             StateWriter
	Log               *slog.Logger
	IdleShutdown      time.Duration
	InitialBackoff    time.Duration
	MaxRestartBackoff time.Duration

	// PluginVersion resolves the installed version for a plugin name.
	// Used to build the sidecar's cwd:
	// ${DataDir}/<plugin>/<version>/. Returning "" disables the
	// version suffix so the fallback is ${DataDir}/<plugin>/.
	PluginVersion func(plugin string) string
}

// Defaults fills unset fields with conservative values.
func (c *Config) applyDefaults() {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.IdleShutdown == 0 {
		c.IdleShutdown = 10 * time.Minute
	}
	if c.InitialBackoff == 0 {
		c.InitialBackoff = 200 * time.Millisecond
	}
	if c.MaxRestartBackoff == 0 {
		c.MaxRestartBackoff = 5 * time.Second
	}
}

// Supervisor owns every host-form plugin's sidecar lifecycle. One
// Supervisor per gateway process.
type Supervisor struct {
	cfg Config

	mu       sync.Mutex
	sidecars map[string]*sidecarEntry

	stopCh  chan struct{}
	stopped bool
	wg      sync.WaitGroup
}

type sidecarEntry struct {
	sidecar *Sidecar
}

// NewSupervisor constructs a Supervisor. Providers is required; State
// may be nil (best-effort — a nil writer silently drops lifecycle
// events).
func NewSupervisor(cfg Config) *Supervisor {
	cfg.applyDefaults()
	return &Supervisor{
		cfg:      cfg,
		sidecars: make(map[string]*sidecarEntry),
		stopCh:   make(chan struct{}),
	}
}

// ─────────────────────────────────────────────
// Ensure / Kill / Stop
// ─────────────────────────────────────────────

// Ensure starts the sidecar for plugin if not already running; if it
// is running, it touches lastUsedAt (resetting the idle-shutdown
// clock) and returns the existing handle.
//
// Returns ErrNoHost when the plugin is known but has no form:"host"
// manifest; ErrSupervisorStopped when Stop has been called; or a
// wrapped spawn error.
func (s *Supervisor) Ensure(ctx context.Context, pluginName string) (*Sidecar, error) {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil, ErrSupervisorStopped
	}
	if e, ok := s.sidecars[pluginName]; ok && e.sidecar.isAlive() {
		e.sidecar.touch()
		s.mu.Unlock()
		return e.sidecar, nil
	}
	s.mu.Unlock()

	// Resolve manifest outside the mutex.
	prov, ok := s.cfg.Providers.Get(pluginName)
	if !ok {
		return nil, fmt.Errorf("host: plugin %q not installed", pluginName)
	}
	if prov.EffectiveForm() != plugin.FormHost || prov.Host == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoHost, pluginName)
	}

	sc, err := s.spawn(ctx, pluginName, prov)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		_ = sc.shutdown(500 * time.Millisecond)
		return nil, ErrSupervisorStopped
	}
	// Double-check in case another goroutine raced Ensure.
	if e, ok := s.sidecars[pluginName]; ok && e.sidecar.isAlive() {
		s.mu.Unlock()
		_ = sc.shutdown(500 * time.Millisecond)
		e.sidecar.touch()
		return e.sidecar, nil
	}
	s.sidecars[pluginName] = &sidecarEntry{sidecar: sc}
	s.mu.Unlock()

	if s.cfg.State != nil {
		_ = s.cfg.State.RecordStarted(ctx, pluginName)
	}
	return sc, nil
}

// Kill terminates a running sidecar with reason logged. Idempotent —
// missing plugin is not an error.
func (s *Supervisor) Kill(pluginName, reason string) error {
	s.mu.Lock()
	e, ok := s.sidecars[pluginName]
	if ok {
		delete(s.sidecars, pluginName)
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	s.cfg.Log.Info("host: killing sidecar", "plugin", pluginName, "reason", reason)
	return e.sidecar.shutdown(5 * time.Second)
}

// Stop halts every sidecar gracefully and waits for them to exit,
// bounded by ctx. Subsequent Ensure calls return ErrSupervisorStopped.
// Idempotent.
func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	close(s.stopCh)
	entries := make([]*sidecarEntry, 0, len(s.sidecars))
	for _, e := range s.sidecars {
		entries = append(entries, e)
	}
	s.sidecars = make(map[string]*sidecarEntry)
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for _, e := range entries {
			wg.Add(1)
			go func(sc *Sidecar) {
				defer wg.Done()
				_ = sc.shutdown(3 * time.Second)
			}(e.sidecar)
		}
		wg.Wait()
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	s.wg.Wait()
	return nil
}

// ActiveCount returns the number of running sidecars. For tests.
func (s *Supervisor) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sidecars)
}

// ─────────────────────────────────────────────
// spawn — the actual process launch
// ─────────────────────────────────────────────

func (s *Supervisor) spawn(ctx context.Context, pluginName string, prov plugin.Provider) (*Sidecar, error) {
	h := prov.Host
	entry := h.Entry
	if plat, ok := h.Platforms[runtime.GOOS+"-"+runtime.GOARCH]; ok && plat != "" {
		entry = plat
	}

	version := ""
	if s.cfg.PluginVersion != nil {
		version = s.cfg.PluginVersion(pluginName)
	}
	installDir := filepath.Join(s.cfg.DataDir, pluginName)
	if version != "" {
		installDir = filepath.Join(installDir, version)
	}

	cwd := installDir
	if h.Cwd != "" {
		if filepath.IsAbs(h.Cwd) {
			cwd = h.Cwd
		} else {
			cwd = filepath.Join(installDir, h.Cwd)
		}
	}
	// Ensure the cwd exists with mode 0700 — the sidecar may expect to
	// write files there (logs, caches) and creating it lazily avoids
	// failing fork/exec with a misleading "no such file or directory"
	// error (which actually refers to the cwd, not the binary).
	if err := os.MkdirAll(cwd, 0o700); err != nil {
		return nil, fmt.Errorf("host: create cwd %s: %w", cwd, err)
	}

	cmdPath, args, err := resolveRuntime(h.Runtime, entry, installDir)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Dir = cwd
	cmd.Env = buildEnv(h.Env)
	configureProcAttrs(cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("host: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("host: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("host: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("host: start %s: %w", pluginName, err)
	}

	sc := &Sidecar{
		plugin: pluginName,
		log:    s.cfg.Log,
		cmd:    cmd,
		stdin:  stdin,
		writer: NewFramedWriter(stdin),
		reader: NewFramedReader(stdout),
		stderr: stderr,
		exited: make(chan struct{}),
	}
	sc.touch()
	sc.startWait(s)

	return sc, nil
}

// resolveRuntime picks the binary + argv for a given HostV1.Runtime.
// Entry paths are interpreted relative to installDir when not absolute.
func resolveRuntime(kind, entry, installDir string) (string, []string, error) {
	abs := entry
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(installDir, entry)
	}
	switch kind {
	case plugin.HostRuntimeBinary, plugin.HostRuntimeCustom, "":
		return abs, nil, nil
	case plugin.HostRuntimeNode:
		bin, err := exec.LookPath("node")
		if err != nil {
			return "", nil, fmt.Errorf("%w: node", ErrRuntimeNotFound)
		}
		return bin, []string{abs}, nil
	case plugin.HostRuntimeDeno:
		bin, err := exec.LookPath("deno")
		if err != nil {
			return "", nil, fmt.Errorf("%w: deno", ErrRuntimeNotFound)
		}
		return bin, []string{"run", "-A", abs}, nil
	case plugin.HostRuntimePython3:
		bin, err := exec.LookPath("python3")
		if err != nil {
			return "", nil, fmt.Errorf("%w: python3", ErrRuntimeNotFound)
		}
		return bin, []string{abs}, nil
	case plugin.HostRuntimeBun:
		bin, err := exec.LookPath("bun")
		if err != nil {
			return "", nil, fmt.Errorf("%w: bun", ErrRuntimeNotFound)
		}
		return bin, []string{"run", abs}, nil
	default:
		return "", nil, fmt.Errorf("host: unknown runtime %q", kind)
	}
}

func buildEnv(extra map[string]string) []string {
	// Minimal env for security: only PATH and the plugin's declared
	// vars. This avoids leaking the gateway's JWT_SECRET etc. into
	// arbitrary sidecars.
	base := []string{"PATH=/usr/local/bin:/usr/bin:/bin"}
	for k, v := range extra {
		base = append(base, k+"="+v)
	}
	return base
}

// ─────────────────────────────────────────────
// Sidecar — one running subprocess
// ─────────────────────────────────────────────

// Sidecar is the handle a caller holds after Ensure. It pipes
// JSON-RPC calls to the subprocess through a FramedWriter and reads
// responses via a FramedReader.
type Sidecar struct {
	plugin string
	log    *slog.Logger
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	writer *FramedWriter
	reader *FramedReader
	stderr io.ReadCloser

	mu         sync.Mutex
	lastUsedAt time.Time
	exited     chan struct{}
	exitErr    error
}

func (s *Sidecar) touch() {
	s.mu.Lock()
	s.lastUsedAt = time.Now()
	s.mu.Unlock()
}

func (s *Sidecar) isAlive() bool {
	select {
	case <-s.exited:
		return false
	default:
		return true
	}
}

// Writer exposes the framed writer for the Mux (M3 T16).
func (s *Sidecar) Writer() *FramedWriter { return s.writer }

// Reader exposes the framed reader for the Mux (M3 T16).
func (s *Sidecar) Reader() *FramedReader { return s.reader }

// Stderr exposes the stderr pipe for the Mux's drain-to-ring-buffer
// goroutine.
func (s *Sidecar) Stderr() io.Reader { return s.stderr }

// Plugin returns the plugin name this sidecar belongs to.
func (s *Sidecar) Plugin() string { return s.plugin }

// Exited returns a channel closed when the underlying process exits.
func (s *Sidecar) Exited() <-chan struct{} { return s.exited }

// ExitErr returns the process exit error after Exited fires.
func (s *Sidecar) ExitErr() error { return s.exitErr }

// startWait launches the waiter goroutine that detects process exit.
func (s *Sidecar) startWait(sup *Supervisor) {
	sup.wg.Add(1)
	// Drain stderr in a loop so a sidecar writing to it doesn't block
	// on pipe backpressure (it eats the messages — a future UI will
	// show them via a ring buffer; for MVP we log the last lines).
	go drainStderr(s.plugin, s.stderr, s.log)
	go func() {
		defer sup.wg.Done()
		err := s.cmd.Wait()
		s.mu.Lock()
		s.exitErr = err
		s.mu.Unlock()
		close(s.exited)
		code := -1
		if s.cmd.ProcessState != nil {
			code = s.cmd.ProcessState.ExitCode()
		}
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		s.log.Info("host: sidecar exited", "plugin", s.plugin, "code", code, "err", msg)
		if sup.cfg.State != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = sup.cfg.State.RecordExited(ctx, s.plugin, code, msg)
		}
		// Remove from map — if already removed by Kill, that's fine.
		sup.mu.Lock()
		if e, ok := sup.sidecars[s.plugin]; ok && e.sidecar == s {
			delete(sup.sidecars, s.plugin)
		}
		sup.mu.Unlock()
	}()
}

// drainStderr buffers stderr line by line and logs each line. Keeps
// the pipe flowing so the child doesn't stall on a full buffer.
func drainStderr(plugin string, r io.Reader, log *slog.Logger) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		log.Debug("host: sidecar stderr", "plugin", plugin, "line", sc.Text())
	}
}

// shutdown asks the sidecar to exit: first a graceful close of stdin
// (EOF), then wait up to timeout, then kill the process group. Safe
// for concurrent callers — only the first triggers the actual close.
func (s *Sidecar) shutdown(timeout time.Duration) error {
	// Attempt a graceful close of stdin so well-behaved sidecars exit
	// on EOF. Ignore errors — the writer may already be closed.
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	select {
	case <-s.exited:
		return s.exitErr
	case <-time.After(timeout):
	}
	// Force kill. killProcessGroup is platform-specific.
	killProcessGroup(s.cmd)
	select {
	case <-s.exited:
		return s.exitErr
	case <-time.After(2 * time.Second):
		return fmt.Errorf("host: sidecar %q did not exit after kill", s.plugin)
	}
}
