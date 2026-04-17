package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// Engine manages a single PTY process.
type Engine struct {
	cmd  *exec.Cmd
	ptmx *os.File
	pid  int
}

// SpawnConfig holds parameters for spawning a PTY process.
type SpawnConfig struct {
	Command      string            // e.g. "claude" or "/bin/zsh"
	Args         []string          // CLI arguments
	CWD          string            // working directory
	Env          map[string]string // extra environment variables
	InitialRows  uint16
	InitialCols  uint16
}

// Spawn creates a new PTY process.
func Spawn(cfg SpawnConfig) (*Engine, error) {
	// Pre-flight: verify the working directory is usable. Without this, a
	// failure from cmd.Start lands as an opaque "fork/exec: no such file or
	// directory" which is ambiguous between the command and the cwd.
	if cfg.CWD == "" {
		return nil, fmt.Errorf("terminal: working directory is empty")
	}
	if info, err := os.Stat(cfg.CWD); err != nil {
		return nil, fmt.Errorf("terminal: cwd %q is not accessible: %w", cfg.CWD, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("terminal: cwd %q is not a directory", cfg.CWD)
	}

	// Pre-flight: verify the command is resolvable. A missing CLI binary is
	// the single most common cause of "I can't start a session" in practice.
	if _, err := exec.LookPath(cfg.Command); err != nil {
		return nil, fmt.Errorf(
			"terminal: command %q not found in PATH — install it or set the provider's command in Providers settings",
			cfg.Command,
		)
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = cfg.CWD

	// Build environment: inherit current + overlay
	env := os.Environ()
	env = append(env, "TERM=xterm-256color", "COLORTERM=truecolor")
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	rows := cfg.InitialRows
	if rows == 0 {
		rows = 40
	}
	cols := cfg.InitialCols
	if cols == 0 {
		cols = 120
	}

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
	if err != nil {
		return nil, fmt.Errorf("terminal: spawn pty: %w", err)
	}

	return &Engine{
		cmd:  cmd,
		ptmx: ptmx,
		pid:  cmd.Process.Pid,
	}, nil
}

// Read reads from the PTY. Blocks until data is available or PTY closes.
func (e *Engine) Read(buf []byte) (int, error) {
	return e.ptmx.Read(buf)
}

// Write sends data to the PTY stdin.
func (e *Engine) Write(data []byte) (int, error) {
	return e.ptmx.Write(data)
}

// Resize changes the PTY window size.
func (e *Engine) Resize(rows, cols uint16) error {
	return pty.Setsize(e.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

// PID returns the process ID.
func (e *Engine) PID() int {
	return e.pid
}

// Wait blocks until the process exits and returns the exit error (if any).
func (e *Engine) Wait() error {
	return e.cmd.Wait()
}

// Stop gracefully terminates the PTY process and all its children.
// Sends SIGHUP to the process group (mimics terminal close), then SIGTERM,
// then SIGKILL after a timeout to guarantee termination.
func (e *Engine) Stop() error {
	if e.cmd.Process == nil {
		return nil
	}
	pid := e.cmd.Process.Pid

	// Send signals to the entire process group (negative PID = process group)
	// This ensures shell + all child processes die together.
	signalGroup := func(sig syscall.Signal) error {
		if err := syscall.Kill(-pid, sig); err == nil {
			return nil
		}
		// Fallback: signal just the main process
		return e.cmd.Process.Signal(sig)
	}

	// 1. SIGHUP: interactive shells treat this as "terminal closed"
	_ = signalGroup(syscall.SIGHUP)

	// 2. Wait briefly for clean exit
	done := make(chan struct{})
	go func() {
		if e.cmd.ProcessState != nil {
			close(done)
			return
		}
		// Poll for exit
		for range 20 {
			time.Sleep(100 * time.Millisecond)
			if err := e.cmd.Process.Signal(syscall.Signal(0)); err != nil {
				close(done)
				return
			}
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
	}

	// 3. SIGTERM
	_ = signalGroup(syscall.SIGTERM)

	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
	}

	// 4. SIGKILL — guaranteed kill
	_ = signalGroup(syscall.SIGKILL)
	// Last resort: kill just the main process
	return e.cmd.Process.Kill()
}

// Close closes the PTY file descriptor.
func (e *Engine) Close() {
	e.ptmx.Close()
}

// IsAlive checks if the process is still running.
func (e *Engine) IsAlive() bool {
	if e.cmd.Process == nil {
		return false
	}
	err := e.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// BuildClaudeArgs constructs CLI arguments for a Claude session.
func BuildClaudeArgs(resumeID, model string, extraArgs []string) []string {
	var args []string
	if resumeID != "" {
		args = append(args, "--resume", resumeID)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extraArgs...)
	return args
}

// BuildShellCommand returns the user's login shell.
func BuildShellCommand() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	return shell
}

// ParseExtraArgs splits a string into arguments, respecting quotes.
func ParseExtraArgs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var args []string
	var current strings.Builder
	inQuote := rune(0)
	for _, r := range s {
		switch {
		case inQuote != 0 && r == inQuote:
			inQuote = 0
		case inQuote != 0:
			current.WriteRune(r)
		case r == '"' || r == '\'':
			inQuote = r
		case r == ' ':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
