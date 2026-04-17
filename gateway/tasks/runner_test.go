package tasks

import (
	"strings"
	"testing"
	"time"
)

func mustRun(t *testing.T, r *Runner, cfg Config, task Task) *Run {
	t.Helper()
	run, err := r.Start(cfg, task)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return run
}

func waitFor(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for run to finish")
	}
}

func TestRunner_EchoSucceeds(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner()
	cfg := Config{AllowedRoots: []string{dir}, ShellTimeoutSec: 5, MaxConcurrent: 4}

	task := Task{
		ID: "t1", Name: "echo", Source: SourceShellScript,
		Workdir: dir, Display: "echo", Argv: []string{"sh", "-c", "echo hello"},
	}
	run := mustRun(t, r, cfg, task)

	snap, ch, done, unsub, err := r.Subscribe(run.ID)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	output := append([]byte{}, snap...)
	collect := func() {
		for {
			select {
			case b, ok := <-ch:
				if !ok {
					return
				}
				output = append(output, b...)
			case <-done:
				// Drain remaining buffered chunks.
				for {
					select {
					case b, ok := <-ch:
						if !ok {
							return
						}
						output = append(output, b...)
					default:
						return
					}
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("output read timeout")
			}
		}
	}
	collect()

	got, ok := r.Get(run.ID)
	if !ok {
		t.Fatalf("run lost")
	}
	if got.Status != StatusExited {
		t.Errorf("status=%s want %s", got.Status, StatusExited)
	}
	if got.ExitCode != 0 {
		t.Errorf("exit=%d want 0", got.ExitCode)
	}
	if !strings.Contains(string(output), "hello") {
		t.Errorf("output missing hello: %q", string(output))
	}
}

func TestRunner_NonZeroExit(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner()
	cfg := Config{AllowedRoots: []string{dir}, ShellTimeoutSec: 5}

	task := Task{
		ID: "t2", Name: "fail", Source: SourceShellScript,
		Workdir: dir, Argv: []string{"sh", "-c", "exit 7"},
	}
	run := mustRun(t, r, cfg, task)
	_, _, done, unsub, _ := r.Subscribe(run.ID)
	defer unsub()
	waitFor(t, done)

	got, _ := r.Get(run.ID)
	if got.Status != StatusExited {
		t.Errorf("status=%s want exited", got.Status)
	}
	if got.ExitCode != 7 {
		t.Errorf("exit=%d want 7", got.ExitCode)
	}
}

func TestRunner_Stop(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner()
	cfg := Config{AllowedRoots: []string{dir}, ShellTimeoutSec: 30}

	task := Task{
		ID: "t3", Name: "sleep", Source: SourceShellScript,
		Workdir: dir, Argv: []string{"sh", "-c", "sleep 30"},
	}
	run := mustRun(t, r, cfg, task)
	_, _, done, unsub, _ := r.Subscribe(run.ID)
	defer unsub()

	if err := r.Stop(run.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	waitFor(t, done)

	got, _ := r.Get(run.ID)
	if got.Status != StatusKilled {
		t.Errorf("status=%s want killed", got.Status)
	}
}

func TestRunner_MaxConcurrent(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner()
	cfg := Config{AllowedRoots: []string{dir}, ShellTimeoutSec: 30, MaxConcurrent: 1}

	long := Task{
		ID: "t4", Workdir: dir,
		Argv: []string{"sh", "-c", "sleep 5"},
	}
	run, err := r.Start(cfg, long)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	defer r.Stop(run.ID)

	if _, err := r.Start(cfg, long); err == nil {
		t.Fatalf("expected error when exceeding MaxConcurrent")
	}
}

func TestRunner_Timeout(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner()
	cfg := Config{AllowedRoots: []string{dir}, ShellTimeoutSec: 1}

	task := Task{
		ID: "t5", Workdir: dir,
		Argv: []string{"sh", "-c", "sleep 10"},
	}
	run := mustRun(t, r, cfg, task)
	_, _, done, unsub, _ := r.Subscribe(run.ID)
	defer unsub()
	waitFor(t, done)

	got, _ := r.Get(run.ID)
	// CommandContext timeout sends SIGKILL — process exits with signal.
	if got.Status == StatusRunning {
		t.Errorf("expected non-running, got %s", got.Status)
	}
}
