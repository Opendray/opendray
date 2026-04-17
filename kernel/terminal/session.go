package terminal

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// SessionState represents the current state of a terminal session.
type SessionState int

const (
	StateIdle    SessionState = iota // no output yet
	StateActive                     // output flowing
	StateWaiting                    // output stopped, waiting for input
)

// Session wraps a PTY engine with output buffering and idle detection.
type Session struct {
	engine *Engine
	buffer *RingBuffer

	state      atomic.Int32
	lastOutput atomic.Int64 // unix nano

	idleThreshold time.Duration
	idleCh        chan struct{} // closed when idle detected
	idleMu        sync.Mutex

	done     chan struct{} // closed when process exits
	exitErr  error
	onExit   func(exitErr error)

	logger *slog.Logger
}

// SessionConfig holds parameters for creating a session.
type SessionConfig struct {
	Engine        *Engine
	BufferSize    int
	IdleThreshold time.Duration
	OnExit        func(exitErr error)
	Logger        *slog.Logger
}

// NewSession creates a session that reads from the engine and manages idle detection.
func NewSession(cfg SessionConfig) *Session {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 4 * 1024 * 1024 // 4MB
	}
	if cfg.IdleThreshold <= 0 {
		cfg.IdleThreshold = 8 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	s := &Session{
		engine:        cfg.Engine,
		buffer:        NewRingBuffer(cfg.BufferSize),
		idleThreshold: cfg.IdleThreshold,
		idleCh:        make(chan struct{}),
		done:          make(chan struct{}),
		onExit:        cfg.OnExit,
		logger:        cfg.Logger,
	}

	go s.readLoop()
	go s.idleLoop()
	go s.waitLoop()
	return s
}

// Engine returns the underlying PTY engine.
func (s *Session) Engine() *Engine {
	return s.engine
}

// Buffer returns the output ring buffer.
func (s *Session) Buffer() *RingBuffer {
	return s.buffer
}

// State returns the current idle detection state.
func (s *Session) State() SessionState {
	return SessionState(s.state.Load())
}

// IdleCh returns a channel that is closed when idle is detected.
// After each idle detection, a new channel is created for the next cycle.
func (s *Session) IdleCh() <-chan struct{} {
	s.idleMu.Lock()
	defer s.idleMu.Unlock()
	return s.idleCh
}

// Done returns a channel that is closed when the process exits.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// ExitErr returns the process exit error (nil if clean exit).
func (s *Session) ExitErr() error {
	return s.exitErr
}

// WriteInput sends user input to the PTY.
func (s *Session) WriteInput(data []byte) error {
	_, err := s.engine.Write(data)
	return err
}

// Resize changes the terminal dimensions.
func (s *Session) Resize(rows, cols uint16) error {
	return s.engine.Resize(rows, cols)
}

// Stop terminates the session process.
func (s *Session) Stop() error {
	return s.engine.Stop()
}

// readLoop continuously reads PTY output into the ring buffer.
func (s *Session) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.engine.Read(buf)
		if n > 0 {
			s.buffer.Write(buf[:n])
			s.lastOutput.Store(time.Now().UnixNano())
			s.state.Store(int32(StateActive))

			// Re-arm idle detection
			s.idleMu.Lock()
			select {
			case <-s.idleCh:
				// already closed from previous idle, create new channel
				s.idleCh = make(chan struct{})
			default:
			}
			s.idleMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// idleLoop periodically checks for idle state.
func (s *Session) idleLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			lastNano := s.lastOutput.Load()
			if lastNano == 0 {
				continue // no output yet
			}
			elapsed := time.Since(time.Unix(0, lastNano))
			if elapsed >= s.idleThreshold && s.State() == StateActive {
				s.state.Store(int32(StateWaiting))
				s.idleMu.Lock()
				select {
				case <-s.idleCh:
					// already closed
				default:
					close(s.idleCh)
				}
				s.idleMu.Unlock()
			}
		}
	}
}

// waitLoop waits for the process to exit.
func (s *Session) waitLoop() {
	s.exitErr = s.engine.Wait()
	s.engine.Close()
	close(s.done)
	if s.onExit != nil {
		s.onExit(s.exitErr)
	}
}
