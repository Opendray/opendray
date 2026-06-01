//go:build !windows

package session

import (
	"errors"
	"syscall"
)

// termProcess sends SIGTERM to the process with the given PID.
// Returns nil if the process no longer exists.
func termProcess(pid int) error {
	err := syscall.Kill(pid, syscall.SIGTERM)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

// killProcess sends SIGKILL to the process with the given PID.
func killProcess(pid int) {
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
