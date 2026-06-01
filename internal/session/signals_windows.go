//go:build windows

package session

import "os"

// termProcess sends a termination signal to the process with the given PID.
// On Windows there is no SIGTERM; we call Kill as the best available equivalent.
// Returns nil if the process no longer exists.
func termProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil // process not found
	}
	if err := proc.Kill(); err != nil {
		return nil // already exited
	}
	return nil
}

// killProcess forcibly terminates the process with the given PID.
func killProcess(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Kill()
}
