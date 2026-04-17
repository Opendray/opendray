package tasks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// RunStatus is the lifecycle state of a single task execution.
type RunStatus string

const (
	StatusRunning RunStatus = "running"
	StatusExited  RunStatus = "exited"
	StatusKilled  RunStatus = "killed"
	StatusError   RunStatus = "error"
)

// Run is the public view of a single task execution.
type Run struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"taskId"`
	TaskName   string    `json:"taskName"`
	Workdir    string    `json:"workdir"`
	Display    string    `json:"display"`
	Status     RunStatus `json:"status"`
	ExitCode   int       `json:"exitCode"`
	StartedAt  int64     `json:"startedAt"`
	FinishedAt int64     `json:"finishedAt,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// Runner manages a pool of in-flight task executions for the task-runner panel.
//
// It is safe for concurrent use. Runs and their output buffers live in memory
// for ttl after completion, then are reaped by a background goroutine.
type Runner struct {
	mu     sync.RWMutex
	runs   map[string]*runState
	active int32
	ttl    time.Duration
}

// NewRunner builds a Runner with a background reaper for finished runs.
func NewRunner() *Runner {
	r := &Runner{
		runs: make(map[string]*runState),
		ttl:  10 * time.Minute,
	}
	go r.reapLoop()
	return r
}

// Start launches a task. It enforces cfg.MaxConcurrent and cfg.ShellTimeoutSec
// and returns the new run's id. The caller can subscribe to live output via
// Subscribe and inspect status via Get.
func (r *Runner) Start(cfg Config, t Task) (*Run, error) {
	if len(t.Argv) == 0 {
		return nil, errors.New("tasks: empty argv")
	}
	if cfg.MaxConcurrent > 0 && int(atomic.LoadInt32(&r.active)) >= cfg.MaxConcurrent {
		return nil, fmt.Errorf("tasks: max concurrent runs reached (%d)", cfg.MaxConcurrent)
	}

	bufSize := cfg.OutputBufferBytes
	if bufSize <= 0 {
		bufSize = 256 * 1024
	}

	ctx, cancel := context.WithCancel(context.Background())
	if cfg.ShellTimeoutSec > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(cfg.ShellTimeoutSec)*time.Second)
	}

	cmd := exec.CommandContext(ctx, t.Argv[0], t.Argv[1:]...)
	cmd.Dir = t.Workdir
	// New process group so we can kill children too.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("tasks: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("tasks: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("tasks: start: %w", err)
	}

	rs := &runState{
		Run: Run{
			ID:        newRunID(),
			TaskID:    t.ID,
			TaskName:  t.Name,
			Workdir:   t.Workdir,
			Display:   t.Display,
			Status:    StatusRunning,
			StartedAt: time.Now().Unix(),
		},
		buf:    newRingBuffer(bufSize),
		cmd:    cmd,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	atomic.AddInt32(&r.active, 1)
	r.mu.Lock()
	r.runs[rs.ID] = rs
	r.mu.Unlock()

	rs.mu.Lock()
	snapshot := rs.Run
	rs.mu.Unlock()

	go rs.pump(stdout)
	go rs.pump(stderr)
	go func() {
		err := cmd.Wait()
		rs.finish(err)
		atomic.AddInt32(&r.active, -1)
	}()

	return &snapshot, nil
}

// Stop signals the run to terminate. It first cancels the context (SIGKILL via
// exec.CommandContext) and then sends SIGTERM to the whole process group as a
// fallback for shells that detached children.
func (r *Runner) Stop(runID string) error {
	r.mu.RLock()
	rs, ok := r.runs[runID]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("tasks: run %q not found", runID)
	}
	rs.mu.Lock()
	rs.killed = true
	rs.mu.Unlock()
	rs.cancel()
	if rs.cmd != nil && rs.cmd.Process != nil {
		// Negative pid → process group. Errors are ignored; the process may
		// already be gone, which is fine.
		_ = syscall.Kill(-rs.cmd.Process.Pid, syscall.SIGTERM)
	}
	return nil
}

// Get returns the public Run snapshot for runID.
func (r *Runner) Get(runID string) (Run, bool) {
	r.mu.RLock()
	rs, ok := r.runs[runID]
	r.mu.RUnlock()
	if !ok {
		return Run{}, false
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.Run, true
}

// List returns every known run, sorted newest-first.
func (r *Runner) List() []Run {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Run, 0, len(r.runs))
	for _, rs := range r.runs {
		rs.mu.Lock()
		out = append(out, rs.Run)
		rs.mu.Unlock()
	}
	return out
}

// Subscribe attaches a new output listener to a run. It returns the historical
// snapshot, a channel of future chunks, a done channel that closes when the
// run finishes, and an unsubscribe function.
func (r *Runner) Subscribe(runID string) (snapshot []byte, output <-chan []byte, done <-chan struct{}, unsubscribe func(), err error) {
	r.mu.RLock()
	rs, ok := r.runs[runID]
	r.mu.RUnlock()
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("tasks: run %q not found", runID)
	}
	id, ch := rs.buf.subscribe()
	return rs.buf.snapshot(), ch, rs.done, func() { rs.buf.unsubscribe(id) }, nil
}

// reapLoop drops finished runs after their ttl expires.
func (r *Runner) reapLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().Unix()
		r.mu.Lock()
		for id, rs := range r.runs {
			rs.mu.Lock()
			finished := rs.FinishedAt > 0
			expired := finished && now-rs.FinishedAt > int64(r.ttl/time.Second)
			rs.mu.Unlock()
			if expired {
				delete(r.runs, id)
			}
		}
		r.mu.Unlock()
	}
}

// ── Internal: per-run state ─────────────────────────────────────

type runState struct {
	Run

	mu     sync.Mutex
	buf    *ringBuffer
	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan struct{}
	killed bool
}

func (rs *runState) pump(rc io.ReadCloser) {
	defer rc.Close()
	chunk := make([]byte, 4096)
	for {
		n, err := rc.Read(chunk)
		if n > 0 {
			rs.buf.write(chunk[:n])
		}
		if err != nil {
			return
		}
	}
}

func (rs *runState) finish(err error) {
	rs.mu.Lock()
	rs.FinishedAt = time.Now().Unix()
	switch {
	case rs.killed:
		rs.Status = StatusKilled
	case err == nil:
		rs.Status = StatusExited
		rs.ExitCode = 0
	default:
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			rs.Status = StatusExited
			rs.ExitCode = ee.ExitCode()
		} else {
			rs.Status = StatusError
			rs.Error = err.Error()
		}
	}
	rs.mu.Unlock()
	rs.buf.close()
	close(rs.done)
}

func newRunID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// ── Internal: ring buffer with subscribers ──────────────────────

type ringBuffer struct {
	mu     sync.Mutex
	data   []byte
	max    int
	subs   map[uint64]chan []byte
	nextID uint64
	closed bool
}

func newRingBuffer(max int) *ringBuffer {
	return &ringBuffer{
		max:  max,
		subs: make(map[uint64]chan []byte),
	}
}

func (b *ringBuffer) write(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	cp := make([]byte, len(chunk))
	copy(cp, chunk)

	b.mu.Lock()
	b.data = append(b.data, cp...)
	if len(b.data) > b.max {
		b.data = b.data[len(b.data)-b.max:]
	}
	subs := make([]chan []byte, 0, len(b.subs))
	for _, ch := range b.subs {
		subs = append(subs, ch)
	}
	b.mu.Unlock()

	for _, ch := range subs {
		// Drop chunks for slow subscribers rather than block the producer.
		select {
		case ch <- cp:
		default:
		}
	}
}

func (b *ringBuffer) snapshot() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]byte, len(b.data))
	copy(out, b.data)
	return out
}

func (b *ringBuffer) subscribe() (uint64, chan []byte) {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	if b.closed {
		close(ch)
		return id, ch
	}
	b.subs[id] = ch
	return id, ch
}

func (b *ringBuffer) unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subs[id]; ok {
		delete(b.subs, id)
		close(ch)
	}
}

func (b *ringBuffer) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, ch := range b.subs {
		delete(b.subs, id)
		close(ch)
	}
}
